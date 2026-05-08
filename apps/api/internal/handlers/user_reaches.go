package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/h2oflow/h2oflow/apps/api/internal/auth"
	"github.com/h2oflow/h2oflow/apps/api/internal/kmlimport"
	"github.com/jackc/pgx/v5/pgxpool"
)

// UserReachHandler handles /api/v1/me/reaches routes.
// All mutations are owner-scoped. devFallbackID is used in development when
// SUPABASE_JWKS_URL is not set so routes remain testable without a real JWT.
type UserReachHandler struct {
	db            *pgxpool.Pool
	devFallbackID string
}

func NewUserReachHandler(db *pgxpool.Pool, devFallbackID string) *UserReachHandler {
	return &UserReachHandler{db: db, devFallbackID: devFallbackID}
}

// ownerID returns the authenticated user's ID, or the dev fallback, or ("", false).
func (h *UserReachHandler) ownerID(r *http.Request) (string, bool) {
	if id, ok := auth.UserIDFromContext(r.Context()); ok {
		return id, true
	}
	if h.devFallbackID != "" {
		return h.devFallbackID, true
	}
	return "", false
}

// ── Types ────────────────────────────────────────────────────────────────────

type userReachFlowRange struct {
	Label    string   `json:"label"`
	MinValue *float64 `json:"min_value"`
	MaxValue *float64 `json:"max_value"`
}

type userReachSummary struct {
	ID          string     `json:"id"`
	Slug        string     `json:"slug"`
	Name        string     `json:"name"`
	RiverName   *string    `json:"river_name"`
	PutInLng    float64    `json:"put_in_lng"`
	PutInLat    float64    `json:"put_in_lat"`
	TakeOutLng  float64    `json:"take_out_lng"`
	TakeOutLat  float64    `json:"take_out_lat"`
	Note        *string    `json:"note"`
	CurrentCFS  *float64   `json:"current_cfs"`
	FlowBand    *string    `json:"flow_band"`
	FlowStatus  string     `json:"flow_status"`
	GaugeID          *string    `json:"gauge_id"`
	CustomGaugeID    *string    `json:"custom_gauge_id"`
	CustomGaugeSlug  *string    `json:"custom_gauge_slug"`
	CustomGaugeName  *string    `json:"custom_gauge_name"`
	LastReadAt       *time.Time `json:"last_reading_at"`
	CreatedAt        time.Time  `json:"created_at"`
}

type userReachDetail struct {
	userReachSummary
	UpComID          *string              `json:"up_comid"`
	DownComID        *string              `json:"down_comid"`
	Centerline       *json.RawMessage     `json:"centerline"`
	GaugeID               *string              `json:"gauge_id"`
	GaugeName             *string              `json:"gauge_name"`
	GaugeSource           *string              `json:"gauge_source"`
	GaugeExternalID       *string              `json:"gauge_external_id"`
	GaugePollHealth       *string              `json:"gauge_poll_health"`
	GaugeLastPollSuccess  *time.Time           `json:"gauge_last_poll_success_at"`
	CustomGaugeID         *string              `json:"custom_gauge_id"`
	CustomGaugeName       *string              `json:"custom_gauge_name"`
	FlowRanges            []userReachFlowRange `json:"flow_ranges"`
}

// ── MapAll ────────────────────────────────────────────────────────────────────

// GET /api/v1/me/reaches/map/all
//
// Returns GeoJSON FeatureCollection of the owner's user reaches. Uses stored
// centerline when present; falls back to a 2-point LineString from put_in →
// take_out so every reach renders on the map.
func (h *UserReachHandler) MapAll(w http.ResponseWriter, r *http.Request) {
	ownerID, ok := h.ownerID(r)
	if !ok {
		errorResponse(w, http.StatusUnauthorized, "authentication required")
		return
	}

	rows, err := h.db.Query(r.Context(), `
		SELECT
			ur.id, ur.slug, ur.name, ur.river_name,
			ST_AsGeoJSON(ur.centerline::geometry)  AS centerline_json,
			ST_X(ur.put_in::geometry)              AS put_in_lng,
			ST_Y(ur.put_in::geometry)              AS put_in_lat,
			ST_X(ur.take_out::geometry)            AS take_out_lng,
			ST_Y(ur.take_out::geometry)            AS take_out_lat,
			COALESCE(lr.value, cg.last_value_cfs)  AS current_cfs,
			CASE
				WHEN COALESCE(lr.value, cg.last_value_cfs) IS NULL OR fr.label IS NULL THEN 'unknown'
				WHEN fr.label = 'running' THEN 'runnable'
				WHEN fr.label = 'low'     THEN 'caution'
				WHEN fr.label = 'high'    THEN 'flood'
				ELSE 'unknown'
			END AS flow_status,
			ur.primary_gauge_id::text AS gauge_id
		FROM user_reaches ur
		LEFT JOIN custom_gauges cg ON cg.id = ur.custom_gauge_id
		LEFT JOIN LATERAL (
			SELECT value FROM gauge_readings
			WHERE gauge_id = ur.primary_gauge_id
			  AND timestamp > NOW() - INTERVAL '48 hours'
			ORDER BY timestamp DESC LIMIT 1
		) lr ON TRUE
		LEFT JOIN LATERAL (
			SELECT label FROM user_reach_flow_ranges
			WHERE user_reach_id = ur.id
			  AND (min_value IS NULL OR COALESCE(lr.value, cg.last_value_cfs) >= min_value)
			  AND (max_value IS NULL OR COALESCE(lr.value, cg.last_value_cfs) <  max_value)
			ORDER BY min_value ASC NULLS FIRST
			LIMIT 1
		) fr ON TRUE
		WHERE ur.owner_id = $1
	`, ownerID)
	if err != nil {
		errorResponse(w, http.StatusInternalServerError, "query failed")
		return
	}
	defer rows.Close()

	type featureProps struct {
		ID          string   `json:"id"`
		Slug        string   `json:"slug"`
		Name        string   `json:"name"`
		RiverName   *string  `json:"river_name"`
		CommonName  *string  `json:"common_name"`
		ClassMax    *float64 `json:"class_max"`
		FlowStatus  string   `json:"flow_status"`
		CurrentCFS  *float64 `json:"current_cfs"`
		GaugeID     *string  `json:"gauge_id"`
		IsUserReach bool     `json:"is_user_reach"`
	}
	type feature struct {
		Type       string          `json:"type"`
		Geometry   json.RawMessage `json:"geometry"`
		Properties featureProps    `json:"properties"`
	}

	features := make([]feature, 0)
	for rows.Next() {
		var (
			id, slug, name  string
			riverName       *string
			centerlineJSON  *string
			putInLng, putInLat   float64
			takeOutLng, takeOutLat float64
			currentCFS      *float64
			flowStatus      string
			gaugeID         *string
		)
		if err := rows.Scan(
			&id, &slug, &name, &riverName,
			&centerlineJSON,
			&putInLng, &putInLat, &takeOutLng, &takeOutLat,
			&currentCFS, &flowStatus, &gaugeID,
		); err != nil {
			continue
		}

		var geom json.RawMessage
		if centerlineJSON != nil && *centerlineJSON != "" {
			geom = json.RawMessage(*centerlineJSON)
		} else {
			// Synthesize a 2-point LineString from put_in → take_out.
			type lineString struct {
				Type        string      `json:"type"`
				Coordinates [][2]float64 `json:"coordinates"`
			}
			raw, _ := json.Marshal(lineString{
				Type:        "LineString",
				Coordinates: [][2]float64{{putInLng, putInLat}, {takeOutLng, takeOutLat}},
			})
			geom = json.RawMessage(raw)
		}

		features = append(features, feature{
			Type:     "Feature",
			Geometry: geom,
			Properties: featureProps{
				ID:          id,
				Slug:        slug,
				Name:        name,
				RiverName:   riverName,
				CommonName:  nil,
				ClassMax:    nil,
				FlowStatus:  flowStatus,
				CurrentCFS:  currentCFS,
				GaugeID:     gaugeID,
				IsUserReach: true,
			},
		})
	}

	type featureCollection struct {
		Type     string    `json:"type"`
		Features []feature `json:"features"`
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "private, no-cache")
	_ = json.NewEncoder(w).Encode(featureCollection{Type: "FeatureCollection", Features: features})
}

// ── List ─────────────────────────────────────────────────────────────────────

// GET /api/v1/me/reaches
func (h *UserReachHandler) List(w http.ResponseWriter, r *http.Request) {
	ownerID, ok := h.ownerID(r)
	if !ok {
		errorResponse(w, http.StatusUnauthorized, "authentication required")
		return
	}

	rows, err := h.db.Query(r.Context(), `
		SELECT
			ur.id, ur.slug, ur.name, ur.river_name,
			ST_X(ur.put_in::geometry)    AS put_in_lng,
			ST_Y(ur.put_in::geometry)    AS put_in_lat,
			ST_X(ur.take_out::geometry)  AS take_out_lng,
			ST_Y(ur.take_out::geometry)  AS take_out_lat,
			ur.note, ur.created_at,
			COALESCE(lr.value, cg.last_value_cfs) AS current_cfs,
			COALESCE(lr.timestamp, cg.last_value_at) AS last_reading_at,
			CASE
				WHEN COALESCE(lr.value, cg.last_value_cfs) IS NULL OR fr.label IS NULL THEN 'unknown'
				WHEN fr.label = 'running' THEN 'runnable'
				WHEN fr.label = 'low'     THEN 'caution'
				WHEN fr.label = 'high'    THEN 'flood'
				ELSE 'unknown'
			END AS flow_status,
			fr.label AS flow_band,
			ur.primary_gauge_id::text AS gauge_id,
			ur.custom_gauge_id::text AS custom_gauge_id,
			cg.slug AS custom_gauge_slug,
			cg.name AS custom_gauge_name
		FROM user_reaches ur
		LEFT JOIN custom_gauges cg ON cg.id = ur.custom_gauge_id
		LEFT JOIN LATERAL (
			SELECT value, timestamp FROM gauge_readings
			WHERE gauge_id = ur.primary_gauge_id
			  AND timestamp > NOW() - INTERVAL '48 hours'
			ORDER BY timestamp DESC LIMIT 1
		) lr ON TRUE
		LEFT JOIN LATERAL (
			SELECT label FROM user_reach_flow_ranges
			WHERE user_reach_id = ur.id
			  AND (min_value IS NULL OR COALESCE(lr.value, cg.last_value_cfs) >= min_value)
			  AND (max_value IS NULL OR COALESCE(lr.value, cg.last_value_cfs) <  max_value)
			ORDER BY min_value ASC NULLS FIRST
			LIMIT 1
		) fr ON TRUE
		WHERE ur.owner_id = $1
		ORDER BY ur.name
	`, ownerID)
	if err != nil {
		errorResponse(w, http.StatusInternalServerError, "query failed")
		return
	}
	defer rows.Close()

	items := make([]userReachSummary, 0)
	for rows.Next() {
		var s userReachSummary
		if err := rows.Scan(
			&s.ID, &s.Slug, &s.Name, &s.RiverName,
			&s.PutInLng, &s.PutInLat, &s.TakeOutLng, &s.TakeOutLat,
			&s.Note, &s.CreatedAt,
			&s.CurrentCFS, &s.LastReadAt,
			&s.FlowStatus, &s.FlowBand, &s.GaugeID,
			&s.CustomGaugeID, &s.CustomGaugeSlug, &s.CustomGaugeName,
		); err == nil {
			items = append(items, s)
		}
	}
	jsonResponse(w, http.StatusOK, items)
}

// ── Get ──────────────────────────────────────────────────────────────────────

// GET /api/v1/me/reaches/{slug}
func (h *UserReachHandler) Get(w http.ResponseWriter, r *http.Request) {
	ownerID, ok := h.ownerID(r)
	if !ok {
		errorResponse(w, http.StatusUnauthorized, "authentication required")
		return
	}
	slug := chi.URLParam(r, "slug")

	var d userReachDetail
	var geojsonBytes []byte

	err := h.db.QueryRow(r.Context(), `
		SELECT
			ur.id, ur.slug, ur.name, ur.river_name,
			ST_X(ur.put_in::geometry)    AS put_in_lng,
			ST_Y(ur.put_in::geometry)    AS put_in_lat,
			ST_X(ur.take_out::geometry)  AS take_out_lng,
			ST_Y(ur.take_out::geometry)  AS take_out_lat,
			ur.note, ur.created_at,
			ur.up_comid, ur.down_comid,
			CASE WHEN ur.centerline IS NOT NULL
				 THEN ST_AsGeoJSON(ur.centerline::geometry)
				 ELSE NULL END,
			ur.primary_gauge_id::text,
			g.name AS gauge_name,
			g.source AS gauge_source,
			g.external_id AS gauge_external_id,
			g.poll_health AS gauge_poll_health,
			g.last_poll_success_at AS gauge_last_poll_success_at,
			ur.custom_gauge_id::text,
			cg.name AS custom_gauge_name,
			COALESCE(lr.value, cg.last_value_cfs) AS current_cfs,
			COALESCE(lr.timestamp, cg.last_value_at) AS last_reading_at,
			CASE
				WHEN COALESCE(lr.value, cg.last_value_cfs) IS NULL OR fr.label IS NULL THEN 'unknown'
				WHEN fr.label = 'running' THEN 'runnable'
				WHEN fr.label = 'low'     THEN 'caution'
				WHEN fr.label = 'high'    THEN 'flood'
				ELSE 'unknown'
			END AS flow_status,
			fr.label AS flow_band
		FROM user_reaches ur
		LEFT JOIN gauges g ON g.id = ur.primary_gauge_id
		LEFT JOIN custom_gauges cg ON cg.id = ur.custom_gauge_id
		LEFT JOIN LATERAL (
			SELECT value, timestamp FROM gauge_readings
			WHERE gauge_id = ur.primary_gauge_id
			  AND timestamp > NOW() - INTERVAL '48 hours'
			ORDER BY timestamp DESC LIMIT 1
		) lr ON TRUE
		LEFT JOIN LATERAL (
			SELECT label FROM user_reach_flow_ranges
			WHERE user_reach_id = ur.id
			  AND (min_value IS NULL OR COALESCE(lr.value, cg.last_value_cfs) >= min_value)
			  AND (max_value IS NULL OR COALESCE(lr.value, cg.last_value_cfs) <  max_value)
			ORDER BY min_value ASC NULLS FIRST
			LIMIT 1
		) fr ON TRUE
		WHERE ur.owner_id = $1 AND ur.slug = $2
	`, ownerID, slug).Scan(
		&d.ID, &d.Slug, &d.Name, &d.RiverName,
		&d.PutInLng, &d.PutInLat, &d.TakeOutLng, &d.TakeOutLat,
		&d.Note, &d.CreatedAt,
		&d.UpComID, &d.DownComID,
		&geojsonBytes,
		&d.GaugeID, &d.GaugeName, &d.GaugeSource, &d.GaugeExternalID,
		&d.GaugePollHealth, &d.GaugeLastPollSuccess,
		&d.CustomGaugeID, &d.CustomGaugeName,
		&d.CurrentCFS, &d.LastReadAt,
		&d.FlowStatus, &d.FlowBand,
	)
	if err != nil {
		errorResponse(w, http.StatusNotFound, "user reach not found")
		return
	}

	if geojsonBytes != nil {
		raw := json.RawMessage(geojsonBytes)
		d.Centerline = &raw
	}

	// Flow ranges
	frRows, _ := h.db.Query(r.Context(), `
		SELECT label, min_value, max_value
		FROM user_reach_flow_ranges
		WHERE user_reach_id = $1
		ORDER BY min_value ASC NULLS FIRST
	`, d.ID)
	if frRows != nil {
		defer frRows.Close()
		d.FlowRanges = make([]userReachFlowRange, 0)
		for frRows.Next() {
			var fr userReachFlowRange
			if frRows.Scan(&fr.Label, &fr.MinValue, &fr.MaxValue) == nil {
				d.FlowRanges = append(d.FlowRanges, fr)
			}
		}
	}

	jsonResponse(w, http.StatusOK, d)
}

// ── Create ───────────────────────────────────────────────────────────────────

// POST /api/v1/me/reaches
func (h *UserReachHandler) Create(w http.ResponseWriter, r *http.Request) {
	ownerID, ok := h.ownerID(r)
	if !ok {
		errorResponse(w, http.StatusUnauthorized, "authentication required")
		return
	}

	type latLng struct {
		Lat float64 `json:"lat"`
		Lng float64 `json:"lng"`
	}
	type bandRange struct {
		MinValue *float64 `json:"min_value"`
		MaxValue *float64 `json:"max_value"`
	}
	var body struct {
		Name      string  `json:"name"`
		RiverName string  `json:"river_name"`
		GnisID    string  `json:"gnis_id"`
		PutIn     latLng  `json:"put_in"`
		TakeOut   latLng  `json:"take_out"`
		UpComID   string  `json:"up_comid"`
		DownComID string  `json:"down_comid"`
		Note      *string `json:"note"`
		FlowRanges *struct {
			Low     *bandRange `json:"low"`
			Running *bandRange `json:"running"`
			High    *bandRange `json:"high"`
		} `json:"flow_ranges"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		errorResponse(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	body.Name = strings.TrimSpace(body.Name)
	if body.Name == "" {
		errorResponse(w, http.StatusBadRequest, "name is required")
		return
	}
	if body.PutIn.Lat == 0 && body.PutIn.Lng == 0 {
		errorResponse(w, http.StatusBadRequest, "put_in coordinates required")
		return
	}
	if body.TakeOut.Lat == 0 && body.TakeOut.Lng == 0 {
		errorResponse(w, http.StatusBadRequest, "take_out coordinates required")
		return
	}

	ctx := r.Context()

	// Auto-assign or create river if river_name provided.
	var riverID *string
	var finalRiverName *string
	if rn := strings.TrimSpace(body.RiverName); rn != "" {
		finalRiverName = &rn
		riverSlug := kmlimport.Slugify(rn)
		var rid string
		if body.GnisID != "" {
			_ = h.db.QueryRow(ctx, `SELECT id FROM rivers WHERE gnis_id = $1`, body.GnisID).Scan(&rid)
		}
		if rid == "" {
			_ = h.db.QueryRow(ctx, `SELECT id FROM rivers WHERE lower(name) = lower($1) LIMIT 1`, rn).Scan(&rid)
		}
		if rid == "" {
			verified := body.GnisID != ""
			var gnisParam interface{}
			if body.GnisID != "" {
				gnisParam = body.GnisID
			}
			_ = h.db.QueryRow(ctx, `
				INSERT INTO rivers (slug, name, gnis_id, verified)
				VALUES ($1, $2, $3, $4)
				ON CONFLICT (slug) DO UPDATE SET name = EXCLUDED.name
				RETURNING id
			`, riverSlug, rn, gnisParam, verified).Scan(&rid)
		}
		if rid != "" {
			riverID = &rid
		}
	}

	// Generate slug from name.
	baseSlug := kmlimport.Slugify(body.Name)
	slug := baseSlug
	// Ensure uniqueness for this owner.
	for i := 2; i <= 20; i++ {
		var existing string
		err := h.db.QueryRow(ctx,
			`SELECT id FROM user_reaches WHERE owner_id = $1 AND slug = $2`, ownerID, slug).Scan(&existing)
		if err != nil {
			break // no conflict
		}
		slug = fmt.Sprintf("%s-%d", baseSlug, i)
	}

	var reachID string
	err := h.db.QueryRow(ctx, `
		INSERT INTO user_reaches
			(owner_id, slug, name, river_id, river_name, put_in, take_out, up_comid, down_comid, note)
		VALUES
			($1, $2, $3, $4, $5,
			 ST_SetSRID(ST_MakePoint($6, $7), 4326)::geography,
			 ST_SetSRID(ST_MakePoint($8, $9), 4326)::geography,
			 NULLIF($10,''), NULLIF($11,''), $12)
		RETURNING id
	`, ownerID, slug, body.Name, riverID, finalRiverName,
		body.PutIn.Lng, body.PutIn.Lat,
		body.TakeOut.Lng, body.TakeOut.Lat,
		body.UpComID, body.DownComID, body.Note,
	).Scan(&reachID)
	if err != nil {
		errorResponse(w, http.StatusInternalServerError, fmt.Sprintf("create failed: %v", err))
		return
	}

	// Insert flow ranges.
	if fr := body.FlowRanges; fr != nil {
		type entry struct {
			label string
			r     *bandRange
		}
		for _, e := range []entry{{"low", fr.Low}, {"running", fr.Running}, {"high", fr.High}} {
			if e.r == nil {
				continue
			}
			_, _ = h.db.Exec(ctx, `
				INSERT INTO user_reach_flow_ranges (user_reach_id, label, min_value, max_value)
				VALUES ($1, $2, $3, $4)
				ON CONFLICT (user_reach_id, label) DO UPDATE
					SET min_value = EXCLUDED.min_value, max_value = EXCLUDED.max_value
			`, reachID, e.label, e.r.MinValue, e.r.MaxValue)
		}
	}

	jsonResponse(w, http.StatusCreated, map[string]string{"id": reachID, "slug": slug})
}

// ── Import ────────────────────────────────────────────────────────────────────

// POST /api/v1/me/reaches/import
// Accepts a share payload (same shape as Create body) and creates a new reach
// owned by the authenticated user. Slug is re-generated to avoid collisions.
func (h *UserReachHandler) Import(w http.ResponseWriter, r *http.Request) {
	ownerID, ok := h.ownerID(r)
	if !ok {
		errorResponse(w, http.StatusUnauthorized, "authentication required")
		return
	}

	type latLng struct {
		Lat float64 `json:"lat"`
		Lng float64 `json:"lng"`
	}
	type bandRange struct {
		MinValue *float64 `json:"min_value"`
		MaxValue *float64 `json:"max_value"`
	}
	var body struct {
		Name      string  `json:"name"`
		RiverName string  `json:"river_name"`
		GnisID    string  `json:"gnis_id"`
		PutIn     latLng  `json:"put_in"`
		TakeOut   latLng  `json:"take_out"`
		UpComID   string  `json:"up_comid"`
		DownComID string  `json:"down_comid"`
		Note      *string `json:"note"`
		FlowRanges *struct {
			Low     *bandRange `json:"low"`
			Running *bandRange `json:"running"`
			High    *bandRange `json:"high"`
		} `json:"flow_ranges"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		errorResponse(w, http.StatusBadRequest, "invalid share payload")
		return
	}
	body.Name = strings.TrimSpace(body.Name)
	if body.Name == "" {
		errorResponse(w, http.StatusBadRequest, "name is required")
		return
	}
	if body.PutIn.Lat == 0 && body.PutIn.Lng == 0 {
		errorResponse(w, http.StatusBadRequest, "put_in coordinates required")
		return
	}
	if body.TakeOut.Lat == 0 && body.TakeOut.Lng == 0 {
		errorResponse(w, http.StatusBadRequest, "take_out coordinates required")
		return
	}

	ctx := r.Context()

	var riverID *string
	var finalRiverName *string
	if rn := strings.TrimSpace(body.RiverName); rn != "" {
		finalRiverName = &rn
		riverSlug := kmlimport.Slugify(rn)
		var rid string
		if body.GnisID != "" {
			_ = h.db.QueryRow(ctx, `SELECT id FROM rivers WHERE gnis_id = $1`, body.GnisID).Scan(&rid)
		}
		if rid == "" {
			_ = h.db.QueryRow(ctx, `SELECT id FROM rivers WHERE lower(name) = lower($1) LIMIT 1`, rn).Scan(&rid)
		}
		if rid == "" {
			verified := body.GnisID != ""
			var gnisParam interface{}
			if body.GnisID != "" {
				gnisParam = body.GnisID
			}
			_ = h.db.QueryRow(ctx, `
				INSERT INTO rivers (slug, name, gnis_id, verified)
				VALUES ($1, $2, $3, $4)
				ON CONFLICT (slug) DO UPDATE SET name = EXCLUDED.name
				RETURNING id
			`, riverSlug, rn, gnisParam, verified).Scan(&rid)
		}
		if rid != "" {
			riverID = &rid
		}
	}

	baseSlug := kmlimport.Slugify(body.Name)
	slug := baseSlug
	for i := 2; i <= 20; i++ {
		var existing string
		err := h.db.QueryRow(ctx,
			`SELECT id FROM user_reaches WHERE owner_id = $1 AND slug = $2`, ownerID, slug).Scan(&existing)
		if err != nil {
			break
		}
		slug = fmt.Sprintf("%s-%d", baseSlug, i)
	}

	var reachID string
	err := h.db.QueryRow(ctx, `
		INSERT INTO user_reaches
			(owner_id, slug, name, river_id, river_name, put_in, take_out, up_comid, down_comid, note)
		VALUES
			($1, $2, $3, $4, $5,
			 ST_SetSRID(ST_MakePoint($6, $7), 4326)::geography,
			 ST_SetSRID(ST_MakePoint($8, $9), 4326)::geography,
			 NULLIF($10,''), NULLIF($11,''), $12)
		RETURNING id
	`, ownerID, slug, body.Name, riverID, finalRiverName,
		body.PutIn.Lng, body.PutIn.Lat,
		body.TakeOut.Lng, body.TakeOut.Lat,
		body.UpComID, body.DownComID, body.Note,
	).Scan(&reachID)
	if err != nil {
		errorResponse(w, http.StatusInternalServerError, fmt.Sprintf("import failed: %v", err))
		return
	}

	if fr := body.FlowRanges; fr != nil {
		type entry struct {
			label string
			r     *bandRange
		}
		for _, e := range []entry{{"low", fr.Low}, {"running", fr.Running}, {"high", fr.High}} {
			if e.r == nil {
				continue
			}
			_, _ = h.db.Exec(ctx, `
				INSERT INTO user_reach_flow_ranges (user_reach_id, label, min_value, max_value)
				VALUES ($1, $2, $3, $4)
				ON CONFLICT (user_reach_id, label) DO UPDATE
					SET min_value = EXCLUDED.min_value, max_value = EXCLUDED.max_value
			`, reachID, e.label, e.r.MinValue, e.r.MaxValue)
		}
	}

	jsonResponse(w, http.StatusCreated, map[string]string{"id": reachID, "slug": slug})
}

// ── Update ───────────────────────────────────────────────────────────────────

// PATCH /api/v1/me/reaches/{slug}
func (h *UserReachHandler) Update(w http.ResponseWriter, r *http.Request) {
	ownerID, ok := h.ownerID(r)
	if !ok {
		errorResponse(w, http.StatusUnauthorized, "authentication required")
		return
	}
	slug := chi.URLParam(r, "slug")

	type latLng struct {
		Lat float64 `json:"lat"`
		Lng float64 `json:"lng"`
	}
	var body struct {
		Name      *string `json:"name"`
		Note      *string `json:"note"`
		RiverName *string `json:"river_name"`
		PutIn     *latLng `json:"put_in"`
		TakeOut   *latLng `json:"take_out"`
		UpComID   *string `json:"up_comid"`
		DownComID *string `json:"down_comid"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		errorResponse(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	// Treat empty river_name as NULL (clear it).
	var riverName *string
	if body.RiverName != nil && strings.TrimSpace(*body.RiverName) != "" {
		rn := strings.TrimSpace(*body.RiverName)
		riverName = &rn
	}

	ctx := r.Context()

	// Core fields update.
	tag, err := h.db.Exec(ctx, `
		UPDATE user_reaches
		SET
			name       = COALESCE($3, name),
			note       = $4,
			river_name = CASE WHEN $5 THEN $6 ELSE river_name END,
			updated_at = NOW()
		WHERE owner_id = $1 AND slug = $2
	`, ownerID, slug, body.Name, body.Note, body.RiverName != nil, riverName)
	if err != nil || tag.RowsAffected() == 0 {
		errorResponse(w, http.StatusNotFound, "user reach not found")
		return
	}

	// Geometry update — only when all four geometry fields provided.
	if body.PutIn != nil && body.TakeOut != nil && body.UpComID != nil && body.DownComID != nil {
		_, _ = h.db.Exec(ctx, `
			UPDATE user_reaches
			SET
				put_in     = ST_SetSRID(ST_MakePoint($3, $4), 4326)::geography,
				take_out   = ST_SetSRID(ST_MakePoint($5, $6), 4326)::geography,
				up_comid   = NULLIF($7, ''),
				down_comid = NULLIF($8, ''),
				updated_at = NOW()
			WHERE owner_id = $1 AND slug = $2
		`, ownerID, slug,
			body.PutIn.Lng, body.PutIn.Lat,
			body.TakeOut.Lng, body.TakeOut.Lat,
			*body.UpComID, *body.DownComID)
	}

}

// ── Delete ───────────────────────────────────────────────────────────────────

// DELETE /api/v1/me/reaches/{slug}
func (h *UserReachHandler) Delete(w http.ResponseWriter, r *http.Request) {
	ownerID, ok := h.ownerID(r)
	if !ok {
		errorResponse(w, http.StatusUnauthorized, "authentication required")
		return
	}
	slug := chi.URLParam(r, "slug")

	tag, err := h.db.Exec(r.Context(),
		`DELETE FROM user_reaches WHERE owner_id = $1 AND slug = $2`, ownerID, slug)
	if err != nil || tag.RowsAffected() == 0 {
		errorResponse(w, http.StatusNotFound, "user reach not found")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ── Flow ranges ──────────────────────────────────────────────────────────────

// GET /api/v1/me/reaches/{slug}/flow-ranges
func (h *UserReachHandler) GetFlowRanges(w http.ResponseWriter, r *http.Request) {
	ownerID, ok := h.ownerID(r)
	if !ok {
		errorResponse(w, http.StatusUnauthorized, "authentication required")
		return
	}
	slug := chi.URLParam(r, "slug")

	var reachID string
	if err := h.db.QueryRow(r.Context(),
		`SELECT id FROM user_reaches WHERE owner_id = $1 AND slug = $2`, ownerID, slug).Scan(&reachID); err != nil {
		errorResponse(w, http.StatusNotFound, "user reach not found")
		return
	}

	rows, _ := h.db.Query(r.Context(), `
		SELECT label, min_value, max_value
		FROM user_reach_flow_ranges
		WHERE user_reach_id = $1
		ORDER BY min_value ASC NULLS FIRST
	`, reachID)
	items := make([]userReachFlowRange, 0)
	if rows != nil {
		defer rows.Close()
		for rows.Next() {
			var fr userReachFlowRange
			if rows.Scan(&fr.Label, &fr.MinValue, &fr.MaxValue) == nil {
				items = append(items, fr)
			}
		}
	}
	jsonResponse(w, http.StatusOK, items)
}

// PUT /api/v1/me/reaches/{slug}/flow-ranges
func (h *UserReachHandler) SetFlowRanges(w http.ResponseWriter, r *http.Request) {
	ownerID, ok := h.ownerID(r)
	if !ok {
		errorResponse(w, http.StatusUnauthorized, "authentication required")
		return
	}
	slug := chi.URLParam(r, "slug")

	var reachID string
	if err := h.db.QueryRow(r.Context(),
		`SELECT id FROM user_reaches WHERE owner_id = $1 AND slug = $2`, ownerID, slug).Scan(&reachID); err != nil {
		errorResponse(w, http.StatusNotFound, "user reach not found")
		return
	}

	type bandRange struct {
		MinValue *float64 `json:"min_value"`
		MaxValue *float64 `json:"max_value"`
	}
	var body struct {
		Low     *bandRange `json:"low"`
		Running *bandRange `json:"running"`
		High    *bandRange `json:"high"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		errorResponse(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	ctx := r.Context()
	_, _ = h.db.Exec(ctx, `DELETE FROM user_reach_flow_ranges WHERE user_reach_id = $1`, reachID)
	type entry struct {
		label string
		r     *bandRange
	}
	for _, e := range []entry{{"low", body.Low}, {"running", body.Running}, {"high", body.High}} {
		if e.r == nil {
			continue
		}
		_, _ = h.db.Exec(ctx, `
			INSERT INTO user_reach_flow_ranges (user_reach_id, label, min_value, max_value)
			VALUES ($1, $2, $3, $4)
		`, reachID, e.label, e.r.MinValue, e.r.MaxValue)
	}

	w.WriteHeader(http.StatusNoContent)
}

// DELETE /api/v1/me/reaches/{slug}/gauge
func (h *UserReachHandler) ClearGauge(w http.ResponseWriter, r *http.Request) {
	ownerID, ok := h.ownerID(r)
	if !ok {
		errorResponse(w, http.StatusUnauthorized, "authentication required")
		return
	}
	slug := chi.URLParam(r, "slug")
	tag, err := h.db.Exec(r.Context(), `
		UPDATE user_reaches
		SET primary_gauge_id = NULL, custom_gauge_id = NULL, updated_at = NOW()
		WHERE owner_id = $1 AND slug = $2
	`, ownerID, slug)
	if err != nil || tag.RowsAffected() == 0 {
		errorResponse(w, http.StatusNotFound, "user reach not found")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ── Centerline ───────────────────────────────────────────────────────────────

// POST /api/v1/me/reaches/{slug}/centerline
// Body: { "up_comid": "...", "down_comid": "...", "start_lat": ..., ... }
// Delegates to the NLDI handler's centerline fetch logic via internal fetch.
func (h *UserReachHandler) SetCenterline(w http.ResponseWriter, r *http.Request) {
	ownerID, ok := h.ownerID(r)
	if !ok {
		errorResponse(w, http.StatusUnauthorized, "authentication required")
		return
	}
	slug := chi.URLParam(r, "slug")

	var reachID string
	if err := h.db.QueryRow(r.Context(),
		`SELECT id FROM user_reaches WHERE owner_id = $1 AND slug = $2`, ownerID, slug).Scan(&reachID); err != nil {
		errorResponse(w, http.StatusNotFound, "user reach not found")
		return
	}

	var body struct {
		GeoJSON json.RawMessage `json:"geojson"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.GeoJSON == nil {
		errorResponse(w, http.StatusBadRequest, "geojson required")
		return
	}

	_, err := h.db.Exec(r.Context(), `
		UPDATE user_reaches
		SET centerline = ST_GeomFromGeoJSON($2)::geography, updated_at = NOW()
		WHERE id = $1
	`, reachID, string(body.GeoJSON))
	if err != nil {
		errorResponse(w, http.StatusInternalServerError, fmt.Sprintf("centerline update failed: %v", err))
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// DELETE /api/v1/me/reaches/{slug}/centerline
func (h *UserReachHandler) ClearCenterline(w http.ResponseWriter, r *http.Request) {
	ownerID, ok := h.ownerID(r)
	if !ok {
		errorResponse(w, http.StatusUnauthorized, "authentication required")
		return
	}
	slug := chi.URLParam(r, "slug")
	tag, err := h.db.Exec(r.Context(), `
		UPDATE user_reaches
		SET centerline = NULL, up_comid = NULL, down_comid = NULL, updated_at = NOW()
		WHERE owner_id = $1 AND slug = $2
	`, ownerID, slug)
	if err != nil || tag.RowsAffected() == 0 {
		errorResponse(w, http.StatusNotFound, "user reach not found")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ── Gauge ─────────────────────────────────────────────────────────────────────

// PUT /api/v1/me/reaches/{slug}/gauge
// Body (pick one):
//   { "gauge_id": "<uuid>" }
//   { "custom_gauge_id": "<uuid>" }
//   { "external_id": "...", "source": "...", "name": "...", "lat": 0.0, "lng": 0.0 }
func (h *UserReachHandler) SetGauge(w http.ResponseWriter, r *http.Request) {
	ownerID, ok := h.ownerID(r)
	if !ok {
		errorResponse(w, http.StatusUnauthorized, "authentication required")
		return
	}
	slug := chi.URLParam(r, "slug")

	var body struct {
		GaugeID       *string  `json:"gauge_id"`
		CustomGaugeID *string  `json:"custom_gauge_id"`
		ExternalID    *string  `json:"external_id"`
		Source        *string  `json:"source"`
		Name          *string  `json:"name"`
		Lat           *float64 `json:"lat"`
		Lng           *float64 `json:"lng"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		errorResponse(w, http.StatusBadRequest, "invalid body")
		return
	}

	// Upsert path: caller provides external_id+source+name+lat+lng
	if body.ExternalID != nil && body.Source != nil {
		if body.Name == nil || body.Lat == nil || body.Lng == nil {
			errorResponse(w, http.StatusBadRequest, "name, lat, lng required with external_id")
			return
		}
		var gaugeID string
		err := h.db.QueryRow(r.Context(), `
			INSERT INTO gauges (external_id, source, name, location)
			VALUES ($1, $2, $3, ST_SetSRID(ST_MakePoint($5, $4), 4326)::geography)
			ON CONFLICT (external_id, source) DO UPDATE
			  SET name = EXCLUDED.name, location = EXCLUDED.location
			RETURNING id
		`, body.ExternalID, body.Source, body.Name, body.Lat, body.Lng).Scan(&gaugeID)
		if err != nil {
			errorResponse(w, http.StatusInternalServerError, "failed to upsert gauge")
			return
		}
		body.GaugeID = &gaugeID
	}

	if body.GaugeID == nil && body.CustomGaugeID == nil {
		errorResponse(w, http.StatusBadRequest, "gauge_id, custom_gauge_id, or external_id+source required")
		return
	}
	if body.GaugeID != nil && body.CustomGaugeID != nil {
		errorResponse(w, http.StatusBadRequest, "only one of gauge_id or custom_gauge_id allowed")
		return
	}

	var tag interface{ RowsAffected() int64 }
	var err error
	if body.GaugeID != nil {
		tag, err = h.db.Exec(r.Context(), `
			UPDATE user_reaches
			SET primary_gauge_id = $3::uuid, custom_gauge_id = NULL, updated_at = NOW()
			WHERE owner_id = $1 AND slug = $2
		`, ownerID, slug, body.GaugeID)
	} else {
		tag, err = h.db.Exec(r.Context(), `
			UPDATE user_reaches
			SET custom_gauge_id = $3::uuid, primary_gauge_id = NULL, updated_at = NOW()
			WHERE owner_id = $1 AND slug = $2
		`, ownerID, slug, body.CustomGaugeID)
	}
	if err != nil || tag.RowsAffected() == 0 {
		errorResponse(w, http.StatusNotFound, "user reach not found or gauge invalid")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
