package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/h2oflow/h2oflow/apps/api/internal/ai"
	"github.com/h2oflow/h2oflow/apps/api/internal/kmlimport"
	"github.com/h2oflow/h2oflow/apps/api/internal/nldi"
	"log"
)

type NLDIHandler struct {
	db           *pgxpool.Pool
	anthropicKey string
	cacheWarmer  func()
}

func NewNLDIHandler(db *pgxpool.Pool) *NLDIHandler { return &NLDIHandler{db: db} }

func (h *NLDIHandler) WithAnthropicKey(key string) *NLDIHandler {
	h.anthropicKey = key
	return h
}

func (h *NLDIHandler) WithCacheWarmer(fn func()) *NLDIHandler {
	h.cacheWarmer = fn
	return h
}

func (h *NLDIHandler) warmCache() {
	if h.cacheWarmer != nil {
		go h.cacheWarmer()
	}
}

// WatershedExplorer handles GET /api/v1/admin/nldi/watershed
//
// Query params:
//
//	lat, lng    float64  — coordinate to snap (required)
//	distance    int      — km radius for upstream navigation (default 150, max 500)
//
// Response: { snap, upstream_flowlines, downstream_flowlines, upstream_gauges }
func (h *NLDIHandler) WatershedExplorer(w http.ResponseWriter, r *http.Request) {
	lat, lng, err := parseLatLng(r)
	if err != nil {
		errorResponse(w, http.StatusBadRequest, err.Error())
		return
	}
	distanceKm := 150
	if d := r.URL.Query().Get("distance"); d != "" {
		if v, err := strconv.Atoi(d); err == nil && v > 0 && v <= 500 {
			distanceKm = v
		}
	}

	ctx := r.Context()
	c := nldi.New()

	snap, err := c.SnapToComID(ctx, lat, lng)
	if err != nil {
		errorResponse(w, http.StatusBadGateway, fmt.Sprintf("snap to NHD: %v", err))
		return
	}

	upFlowlines, err := c.UpstreamFlowlines(ctx, snap.ComID, distanceKm)
	if err != nil {
		errorResponse(w, http.StatusBadGateway, fmt.Sprintf("upstream flowlines: %v", err))
		return
	}

	downFlowlines, err := c.DownstreamFlowlines(ctx, snap.ComID, distanceKm)
	if err != nil {
		errorResponse(w, http.StatusBadGateway, fmt.Sprintf("downstream flowlines: %v", err))
		return
	}

	upGauges, err := c.UpstreamGauges(ctx, snap.ComID, distanceKm)
	if err != nil {
		upGauges = &nldi.Collection{Type: "FeatureCollection"}
	}

	type snapInfo struct {
		ComID string  `json:"comid"`
		Name  string  `json:"name"`
		Lat   float64 `json:"lat"`
		Lng   float64 `json:"lng"`
	}
	type response struct {
		Snap                snapInfo        `json:"snap"`
		UpstreamFlowlines   nldi.Collection `json:"upstream_flowlines"`
		DownstreamFlowlines nldi.Collection `json:"downstream_flowlines"`
		UpstreamGauges      nldi.Collection `json:"upstream_gauges"`
	}

	body := response{
		Snap:                snapInfo{ComID: snap.ComID, Name: snap.Name, Lat: lat, Lng: lng},
		UpstreamFlowlines:   *upFlowlines,
		DownstreamFlowlines: *downFlowlines,
		UpstreamGauges:      *upGauges,
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(body)
}

// ── Reach authoring ───────────────────────────────────────────────────────────

// createReachRequest uses ComID-first authoring: admin selects upstream and
// downstream ComID segments from the NHD network on the map. Access point
// coords are not required here — they come from KML import later, at which
// point the NLDI centerline is trimmed and stored.
type createReachRequest struct {
	Slug           string   `json:"slug"`        // optional — auto-derived from river+name if blank
	Name           string   `json:"name"`
	CommonName     string   `json:"common_name"`
	RiverName      string   `json:"river_name"`
	UpComID        string   `json:"up_comid"`    // reach-start ComID — required
	DownComID      string   `json:"down_comid"`  // reach-end ComID — required
	StartLat       *float64 `json:"start_lat"`   // clicked lat for reach start
	StartLng       *float64 `json:"start_lng"`
	EndLat         *float64 `json:"end_lat"`     // clicked lat for reach end
	EndLng         *float64 `json:"end_lng"`
	ClassMin       *float64 `json:"class_min"`
	ClassMax       *float64 `json:"class_max"`
	Description    string   `json:"description"`
	PermitRequired bool     `json:"permit_required"`
	MultiDayDays   int      `json:"multi_day_days"` // 0 or 1 = single day
}

// CreateReach handles POST /api/v1/admin/reaches.
//
// Creates a reach shell with the upstream/downstream ComIDs selected from the
// NHD network. No access points or centerline geometry are stored yet — those
// are finalised when a KML is imported for this reach.
func (h *NLDIHandler) CreateReach(w http.ResponseWriter, r *http.Request) {
	var req createReachRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		errorResponse(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	if strings.TrimSpace(req.Name) == "" {
		errorResponse(w, http.StatusBadRequest, "name is required")
		return
	}
	if strings.TrimSpace(req.UpComID) == "" {
		errorResponse(w, http.StatusBadRequest, "up_comid is required")
		return
	}
	if strings.TrimSpace(req.DownComID) == "" {
		errorResponse(w, http.StatusBadRequest, "down_comid is required")
		return
	}

	slug := strings.TrimSpace(req.Slug)
	if slug == "" {
		slug = buildSlug(req.RiverName, req.Name)
	}
	ctx := r.Context()

	days := req.MultiDayDays
	if days < 1 {
		days = 1
	}

	var startPoint, endPoint interface{}
	if req.StartLat != nil && req.StartLng != nil {
		startPoint = fmt.Sprintf("SRID=4326;POINT(%f %f)", *req.StartLng, *req.StartLat)
	}
	if req.EndLat != nil && req.EndLng != nil {
		endPoint = fmt.Sprintf("SRID=4326;POINT(%f %f)", *req.EndLng, *req.EndLat)
	}

	var reachID string
	err := h.db.QueryRow(ctx, `
		INSERT INTO reaches (
			slug, name, common_name, river_name,
			class_min, class_max,
			start_comid, end_comid, anchor_comid,
			start_point, end_point,
			centerline_source,
			description, permit_required, multi_day_days
		) VALUES (
			$1, $2, NULLIF($3,''), NULLIF($4,''),
			$5, $6,
			$7, $8, $7,
			$9::geography, $10::geography,
			'nldi',
			NULLIF($11,''), $12, $13
		)
		RETURNING id
	`, slug, req.Name, req.CommonName, req.RiverName,
		req.ClassMin, req.ClassMax,
		req.UpComID, req.DownComID,
		startPoint, endPoint,
		req.Description, req.PermitRequired, days,
	).Scan(&reachID)
	if err != nil {
		if strings.Contains(err.Error(), "duplicate key") || strings.Contains(err.Error(), "unique constraint") {
			errorResponse(w, http.StatusConflict, fmt.Sprintf("reach with slug %q already exists", slug))
			return
		}
		errorResponse(w, http.StatusInternalServerError, fmt.Sprintf("create reach: %v", err))
		return
	}

	// Auto-populate state from put-in coordinates.
	if req.StartLat != nil && req.StartLng != nil {
		fillReachState(ctx, h.db, slug, *req.StartLat, *req.StartLng)
	}

	// Auto-upsert a river record when river_name is provided, then link the reach.
	// Uses the name+basin unique index to avoid duplicates. Best-effort — a
	// failure here is silent so the reach is still usable.
	if req.RiverName != "" {
		riverSlug := kmlimport.Slugify(req.RiverName)
		var riverID string
		_ = h.db.QueryRow(ctx, `
			INSERT INTO rivers (slug, name)
			VALUES ($1, $2)
			ON CONFLICT (lower(name), COALESCE(lower(basin), '')) DO UPDATE SET name = EXCLUDED.name
			RETURNING id
		`, riverSlug, req.RiverName).Scan(&riverID)
		if riverID != "" {
			h.db.Exec(ctx, `UPDATE reaches SET river_id = $1 WHERE id = $2`, riverID, reachID)
		}
	}

	jsonResponse(w, http.StatusCreated, map[string]any{
		"slug": slug,
		"id":   reachID,
	})
}

// GetAdminReach handles GET /api/v1/admin/reaches/{slug}
//
// Returns admin-relevant reach detail: ComIDs, access point coords, metadata.
// Used to populate the re-pin and edit forms in the admin panel.
func (h *NLDIHandler) GetAdminReach(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	ctx := r.Context()

	var (
		id, name, riverName, commonName string
		riverID                         *string
		classMin, classMax              *float64
		description                     *string
		permitRequired                  bool
		multiDayDays                    int
		putInComID, takeOutComID        *string
		anchorComID                     *string
		putInLng, putInLat              *float64
		takeOutLng, takeOutLat          *float64
		primaryGaugeID                  *string
		primaryGaugeExtID               *string
		primaryGaugeName                *string
	)
	err := h.db.QueryRow(ctx, `
		SELECT
			r.id, r.name, COALESCE(r.river_name,''), COALESCE(r.common_name,''),
			r.river_id,
			r.class_min, r.class_max, r.description,
			COALESCE(r.permit_required, false), COALESCE(r.multi_day_days, 1),
			r.start_comid, r.end_comid, r.anchor_comid,
			ST_X(r.start_point::geometry),  ST_Y(r.start_point::geometry),
			ST_X(r.end_point::geometry),    ST_Y(r.end_point::geometry),
			r.primary_gauge_id::text,
			g.external_id,
			g.name
		FROM reaches r
		LEFT JOIN gauges g ON g.id = r.primary_gauge_id
		WHERE r.slug = $1
	`, slug).Scan(
		&id, &name, &riverName, &commonName,
		&riverID,
		&classMin, &classMax, &description,
		&permitRequired, &multiDayDays,
		&putInComID, &takeOutComID, &anchorComID,
		&putInLng, &putInLat, &takeOutLng, &takeOutLat,
		&primaryGaugeID, &primaryGaugeExtID, &primaryGaugeName,
	)
	if err != nil {
		errorResponse(w, http.StatusNotFound, fmt.Sprintf("reach %q not found", slug))
		return
	}

	type coordPair struct {
		Lat float64 `json:"lat"`
		Lng float64 `json:"lng"`
	}
	var putIn, takeOut *coordPair
	if putInLat != nil && putInLng != nil {
		putIn = &coordPair{Lat: *putInLat, Lng: *putInLng}
	}
	if takeOutLat != nil && takeOutLng != nil {
		takeOut = &coordPair{Lat: *takeOutLat, Lng: *takeOutLng}
	}

	jsonResponse(w, http.StatusOK, map[string]any{
		"id":                       id,
		"slug":                     slug,
		"name":                     name,
		"river_name":               riverName,
		"river_id":                 riverID,
		"common_name":              commonName,
		"class_min":                classMin,
		"class_max":                classMax,
		"description":              description,
		"permit_required":          permitRequired,
		"multi_day_days":           multiDayDays,
		"start_comid":              putInComID,
		"end_comid":                takeOutComID,
		"anchor_comid":             anchorComID,
		"put_in":                   putIn,
		"take_out":                 takeOut,
		"primary_gauge_id":         primaryGaugeID,
		"primary_gauge_external_id": primaryGaugeExtID,
		"primary_gauge_name":       primaryGaugeName,
	})
}

// NearbyGauges handles GET /api/v1/admin/nldi/nearby-gauges
// Returns active gauges from three sources in parallel:
//   - Local DB PostGIS radius (all seeded sources)
//   - Colorado DWR API live query (catches unseeded DWR stations)
//   - NLDI flow-network gauges (optional, requires comid param; catches
//     upstream USGS gauges far along the network)
//
// Query params: lat, lng (required), comid (optional), distance (km, default 100, max 500)
func (h *NLDIHandler) NearbyGauges(w http.ResponseWriter, r *http.Request) {
	lat, lng, err := parseLatLng(r)
	if err != nil {
		errorResponse(w, http.StatusBadRequest, err.Error())
		return
	}
	comid := r.URL.Query().Get("comid")
	distanceKm := 100
	if d := r.URL.Query().Get("distance"); d != "" {
		if v, err := strconv.Atoi(d); err == nil && v > 0 && v <= 500 {
			distanceKm = v
		}
	}

	ctx := r.Context()

	type gaugeFeature struct {
		Type       string `json:"type"`
		Geometry   any    `json:"geometry"`
		Properties any    `json:"properties"`
	}

	// Channel for collecting all gauge features from parallel sources.
	type result struct {
		features []gaugeFeature
		err      error
	}
	dbCh   := make(chan result, 1)
	dwrCh  := make(chan result, 1)
	nldiCh := make(chan result, 1)

	// 1. Local DB — all seeded gauges within radius.
	go func() {
		rows, err := h.db.Query(ctx, `
			SELECT external_id, source, name,
			       ST_Y(location::geometry),
			       ST_X(location::geometry)
			FROM gauges
			WHERE status = 'active'
			  AND location IS NOT NULL
			  AND ST_DWithin(
			        location,
			        ST_SetSRID(ST_MakePoint($1, $2), 4326)::geography,
			        $3
			      )
			ORDER BY location <-> ST_SetSRID(ST_MakePoint($1, $2), 4326)::geography
			LIMIT 60
		`, lng, lat, distanceKm*1000)
		if err != nil {
			dbCh <- result{err: err}
			return
		}
		defer rows.Close()
		var feats []gaugeFeature
		for rows.Next() {
			var extID, source string
			var name *string
			var gLat, gLng float64
			if rows.Scan(&extID, &source, &name, &gLat, &gLng) != nil {
				continue
			}
			n := ""
			if name != nil {
				n = *name
			}
			feats = append(feats, gaugeFeature{
				Type:       "Feature",
				Geometry:   map[string]any{"type": "Point", "coordinates": []float64{gLng, gLat}},
				Properties: map[string]any{"identifier": extID, "source": source, "name": n},
			})
		}
		dbCh <- result{features: feats}
	}()

	// 2. DWR live API — discharge stations within radius.
	go func() {
		stations, err := nldi.DWRNearby(ctx, lat, lng, distanceKm)
		if err != nil {
			dwrCh <- result{err: err}
			return
		}
		feats := make([]gaugeFeature, 0, len(stations))
		for _, st := range stations {
			feats = append(feats, gaugeFeature{
				Type:       "Feature",
				Geometry:   map[string]any{"type": "Point", "coordinates": []float64{st.Lng, st.Lat}},
				Properties: map[string]any{"identifier": st.ExternalID, "source": "dwr", "name": st.Name},
			})
		}
		dwrCh <- result{features: feats}
	}()

	// 3. NLDI flow-network gauges — upstream + short downstream if comid given.
	go func() {
		if comid == "" {
			nldiCh <- result{}
			return
		}
		c := nldi.New()
		upGauges, _ := c.UpstreamGauges(ctx, comid, distanceKm)
		downKm := distanceKm
		if downKm > 50 {
			downKm = 50
		}
		downGauges, _ := c.DownstreamGauges(ctx, comid, downKm)
		var feats []gaugeFeature
		for _, coll := range []*nldi.Collection{upGauges, downGauges} {
			if coll == nil {
				continue
			}
			for _, f := range coll.Features {
				raw, _ := json.Marshal(f.Geometry.Coordinates)
				var coords []float64
				if json.Unmarshal(raw, &coords) != nil || len(coords) < 2 {
					continue
				}
				feats = append(feats, gaugeFeature{
					Type:       "Feature",
					Geometry:   map[string]any{"type": "Point", "coordinates": coords},
					Properties: map[string]any{"identifier": f.Props.Identifier, "source": "usgs", "name": f.Props.Name},
				})
			}
		}
		nldiCh <- result{features: feats}
	}()

	// Merge — deduplicate by (source, identifier).
	seen := map[string]bool{}
	features := make([]gaugeFeature, 0)
	for _, res := range []result{<-dbCh, <-dwrCh, <-nldiCh} {
		for _, f := range res.features {
			p, _ := f.Properties.(map[string]any)
			key := fmt.Sprintf("%v|%v", p["source"], p["identifier"])
			if seen[key] {
				continue
			}
			seen[key] = true
			features = append(features, f)
		}
	}

	jsonResponse(w, http.StatusOK, map[string]any{
		"type":     "FeatureCollection",
		"features": features,
	})
}

// SetPrimaryGauge handles PUT /api/v1/admin/reaches/{slug}/primary-gauge
// Upserts the gauge by (external_id, source) and sets reaches.primary_gauge_id.
func (h *NLDIHandler) SetPrimaryGauge(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	var body struct {
		ExternalID string  `json:"external_id"`
		Source     string  `json:"source"`
		Name       string  `json:"name"`
		Lat        float64 `json:"lat"`
		Lng        float64 `json:"lng"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		errorResponse(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	if body.ExternalID == "" {
		errorResponse(w, http.StatusBadRequest, "external_id required")
		return
	}
	if body.Source == "" {
		body.Source = "usgs"
	}

	ctx := r.Context()

	var gaugeID string
	err := h.db.QueryRow(ctx, `
		INSERT INTO gauges (external_id, source, name, location)
		VALUES ($1, $2, $3, ST_SetSRID(ST_MakePoint($4, $5), 4326))
		ON CONFLICT (external_id, source) DO UPDATE
			SET name = EXCLUDED.name
		RETURNING id
	`, body.ExternalID, body.Source, body.Name, body.Lng, body.Lat).Scan(&gaugeID)
	if err != nil {
		errorResponse(w, http.StatusInternalServerError, fmt.Sprintf("upsert gauge: %v", err))
		return
	}

	var reachID string
	if err := h.db.QueryRow(ctx, `
		UPDATE reaches SET primary_gauge_id = $1 WHERE slug = $2 RETURNING id
	`, gaugeID, slug).Scan(&reachID); err != nil {
		errorResponse(w, http.StatusNotFound, "reach not found")
		return
	}

	// Link gauge to reach if it has no reach association yet.
	h.db.Exec(ctx, `UPDATE gauges SET reach_id = $1 WHERE id = $2 AND reach_id IS NULL`, reachID, gaugeID)

	h.warmCache()

	jsonResponse(w, http.StatusOK, map[string]any{
		"gauge_id":    gaugeID,
		"external_id": body.ExternalID,
		"name":        body.Name,
	})
}

// UpdateReachMeta handles PUT /api/v1/admin/reaches/{slug}/meta
//
// Full metadata update: name, common_name, river_name, class_min, class_max,
// permit_required, multi_day_days. Does not touch centerline or description.
func (h *NLDIHandler) UpdateReachMeta(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	var req struct {
		Name           string   `json:"name"`
		NewSlug        string   `json:"new_slug"`
		CommonName     string   `json:"common_name"`
		RiverName      string   `json:"river_name"`
		RiverID        *string  `json:"river_id"`
		ClassMin       *float64 `json:"class_min"`
		ClassMax       *float64 `json:"class_max"`
		PermitRequired bool     `json:"permit_required"`
		MultiDayDays   int      `json:"multi_day_days"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		errorResponse(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	if strings.TrimSpace(req.Name) == "" {
		errorResponse(w, http.StatusBadRequest, "name is required")
		return
	}
	days := req.MultiDayDays
	if days < 1 {
		days = 1
	}
	newSlug := slug
	if s := strings.TrimSpace(req.NewSlug); s != "" {
		newSlug = s
	}
	ctx := r.Context()
	tag, err := h.db.Exec(ctx, `
		UPDATE reaches SET
			slug            = $1,
			name            = $2,
			common_name     = NULLIF($3, ''),
			river_name      = CASE
			                    WHEN $10::uuid IS NOT NULL
			                    THEN (SELECT name FROM rivers WHERE id = $10::uuid)
			                    ELSE NULLIF($4, '')
			                  END,
			class_min       = $5,
			class_max       = $6,
			permit_required = $7,
			multi_day_days  = $8,
			river_id        = $10
		WHERE slug = $9
	`, newSlug, req.Name, req.CommonName, req.RiverName,
		req.ClassMin, req.ClassMax,
		req.PermitRequired, days,
		slug, req.RiverID,
	)
	if err != nil {
		if strings.Contains(err.Error(), "duplicate key") || strings.Contains(err.Error(), "unique constraint") {
			errorResponse(w, http.StatusConflict, fmt.Sprintf("slug %q already exists", newSlug))
			return
		}
		errorResponse(w, http.StatusInternalServerError, fmt.Sprintf("update: %v", err))
		return
	}
	if tag.RowsAffected() == 0 {
		errorResponse(w, http.StatusNotFound, fmt.Sprintf("reach %q not found", slug))
		return
	}
	h.warmCache()
	jsonResponse(w, http.StatusOK, map[string]any{"slug": newSlug})
}

// UpstreamTributaries handles GET /api/v1/admin/nldi/upstream-tributaries
//
// Snaps lat/lng to the nearest NHD ComID (anchor), then returns all upstream
// tributary flowlines (UT navigation). Used to discover ComIDs for small
// creeks that don't snap reliably via comid/position alone — once the larger
// river's anchor is found, all its tributaries appear as clickable segments.
//
// Query params:
//
//	lat, lng    float64  — coordinate to snap (required)
//	distance    int      — km radius (default 50, max 200)
func (h *NLDIHandler) UpstreamTributaries(w http.ResponseWriter, r *http.Request) {
	lat, lng, err := parseLatLng(r)
	if err != nil {
		errorResponse(w, http.StatusBadRequest, err.Error())
		return
	}
	distanceKm := 50
	if d := r.URL.Query().Get("distance"); d != "" {
		if v, err2 := strconv.Atoi(d); err2 == nil && v > 0 && v <= 200 {
			distanceKm = v
		}
	}

	ctx := r.Context()
	c := nldi.New()

	snap, err := c.SnapToComID(ctx, lat, lng)
	if err != nil {
		errorResponse(w, http.StatusBadGateway, fmt.Sprintf("snap to NHD: %v", err))
		return
	}

	tributaries, err := c.UpstreamFlowlines(ctx, snap.ComID, distanceKm)
	if err != nil {
		errorResponse(w, http.StatusBadGateway, fmt.Sprintf("upstream tributaries: %v", err))
		return
	}

	type snapInfo struct {
		ComID string  `json:"comid"`
		Name  string  `json:"name"`
		Lat   float64 `json:"lat"`
		Lng   float64 `json:"lng"`
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"snap":        snapInfo{ComID: snap.ComID, Name: snap.Name, Lat: lat, Lng: lng},
		"tributaries": tributaries,
	})
}

// DownstreamMainstem handles GET /api/v1/admin/nldi/downstream
//
// Returns the downstream mainstem flowlines from a known ComID. Used after the
// upstream ComID is selected in the author flow — displays the full downstream
// river so the user can click anywhere along it to set the take-out ComID,
// even for very long reaches (e.g. Grand Canyon ~300 mi).
//
// Query params:
//
//	comid       string — NHD ComID to trace downstream from (required)
//	distance    int    — km radius (default 500, max 1000)
func (h *NLDIHandler) DownstreamMainstem(w http.ResponseWriter, r *http.Request) {
	comid := strings.TrimSpace(r.URL.Query().Get("comid"))
	if comid == "" {
		errorResponse(w, http.StatusBadRequest, "comid is required")
		return
	}
	distanceKm := 500
	if d := r.URL.Query().Get("distance"); d != "" {
		if v, err := strconv.Atoi(d); err == nil && v > 0 && v <= 1000 {
			distanceKm = v
		}
	}

	ctx := r.Context()
	c := nldi.New()

	flowlines, err := c.DownstreamFlowlines(ctx, comid, distanceKm)
	if err != nil {
		errorResponse(w, http.StatusBadGateway, fmt.Sprintf("downstream flowlines: %v", err))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"downstream_flowlines": flowlines,
	})
}

// RiverName handles GET /api/v1/admin/nldi/river-name?comid=<comid>[&lat=<lat>&lng=<lng>]
//
// Returns the GNIS stream name at the given location. When lat/lng are provided
// (the exact point the user clicked on the flowline) we query the National Map
// NHD ArcGIS service directly at that coordinate — no NLDI navigation needed,
// and the result is the feature the user actually clicked, not a random tributary.
// When only comid is provided we fall back to fetching a short upstream slice to
// extract a coordinate (legacy path, less reliable for large rivers).
func (h *NLDIHandler) RiverName(w http.ResponseWriter, r *http.Request) {
	comid := strings.TrimSpace(r.URL.Query().Get("comid"))
	if comid == "" {
		errorResponse(w, http.StatusBadRequest, "comid is required")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 8*time.Second)
	defer cancel()

	var lat, lng float64

	// Prefer caller-supplied coordinates (the exact map click point).
	if latStr := r.URL.Query().Get("lat"); latStr != "" {
		lat, _ = strconv.ParseFloat(latStr, 64)
		lng, _ = strconv.ParseFloat(r.URL.Query().Get("lng"), 64)
	}

	// Fall back: navigate a short upstream slice to extract a coordinate.
	if lat == 0 && lng == 0 {
		c := nldi.New()
		up, err := c.UpstreamFlowlines(ctx, comid, 10)
		if err != nil || up == nil || len(up.Features) == 0 {
			up, _ = c.DownstreamFlowlines(ctx, comid, 5)
		}
		if up != nil {
			for _, f := range up.Features {
				if pt := nldi.FirstCoord(f.Geometry); pt != nil {
					lat, lng = pt[1], pt[0]
					break
				}
			}
		}
	}

	if lat == 0 && lng == 0 {
		jsonResponse(w, http.StatusOK, map[string]any{"river_name": ""})
		return
	}

	name, gnisID, err := nldi.NHDStreamNameAt(ctx, lat, lng)
	if err != nil {
		// Non-fatal: return empty rather than a 502.
		jsonResponse(w, http.StatusOK, map[string]any{"river_name": ""})
		return
	}
	jsonResponse(w, http.StatusOK, map[string]any{"river_name": name, "gnis_id": gnisID})
}

type updateReachCenterlineRequest struct {
	PutIn   latLng `json:"put_in"`
	TakeOut latLng `json:"take_out"`
	DryRun  bool   `json:"dry_run"`
}

type latLng struct {
	Lat float64 `json:"lat"`
	Lng float64 `json:"lng"`
}

// UpdateReachCenterline handles POST /api/v1/admin/reaches/{slug}/nldi-centerline
//
// Fetches an NLDI centerline between the supplied coordinates and replaces the
// reach's stored centerline geometry. The reach's reach_access rows are not
// modified — only centerline, length_mi, start_comid, end_comid update.
func (h *NLDIHandler) UpdateReachCenterline(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")

	var req updateReachCenterlineRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		errorResponse(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	if req.PutIn.Lat == 0 && req.PutIn.Lng == 0 {
		errorResponse(w, http.StatusBadRequest, "put_in coordinates required")
		return
	}
	if req.TakeOut.Lat == 0 && req.TakeOut.Lng == 0 {
		errorResponse(w, http.StatusBadRequest, "take_out coordinates required")
		return
	}

	ctx := r.Context()

	var reachID string
	if err := h.db.QueryRow(ctx, `SELECT id FROM reaches WHERE slug = $1`, slug).Scan(&reachID); err != nil {
		errorResponse(w, http.StatusNotFound, fmt.Sprintf("reach %q not found", slug))
		return
	}

	if err := kmlimport.SyncCenterlineAt(ctx, h.db, slug, kmlimport.CenterlineNLDI,
		req.PutIn.Lng, req.PutIn.Lat, req.TakeOut.Lng, req.TakeOut.Lat, req.DryRun); err != nil {
		errorResponse(w, http.StatusBadGateway, fmt.Sprintf("nldi centerline: %v", err))
		return
	}

	if req.DryRun {
		jsonResponse(w, http.StatusOK, map[string]any{"dry_run": true})
		return
	}

	fillReachState(ctx, h.db, slug, req.PutIn.Lat, req.PutIn.Lng)

	var lengthMi *float64
	var putInComID, takeOutComID *string
	_ = h.db.QueryRow(ctx, `
		SELECT length_mi, start_comid, end_comid FROM reaches WHERE id = $1
	`, reachID).Scan(&lengthMi, &putInComID, &takeOutComID)

	jsonResponse(w, http.StatusOK, map[string]any{
		"slug":        slug,
		"length_mi":   lengthMi,
		"start_comid": putInComID,
		"end_comid":   takeOutComID,
	})
}

// UpdateReachCenterlineByComID handles POST /api/v1/admin/reaches/{slug}/nldi-centerline-by-comid
//
// Re-traces the reach's centerline from the supplied upstream/downstream ComIDs.
// Trim coordinates come from (in priority order):
//  1. start_lat/start_lng/end_lat/end_lng in the request body (from map click)
//  2. start_point/end_point already stored on the reach row
//
// When body coordinates are provided they are also written to start_point/end_point.
func (h *NLDIHandler) UpdateReachCenterlineByComID(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")

	var req struct {
		UpComID   string   `json:"up_comid"`
		DownComID string   `json:"down_comid"`
		StartLat  *float64 `json:"start_lat"`
		StartLng  *float64 `json:"start_lng"`
		EndLat    *float64 `json:"end_lat"`
		EndLng    *float64 `json:"end_lng"`
		DryRun    bool     `json:"dry_run"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		errorResponse(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	req.UpComID = strings.TrimSpace(req.UpComID)
	req.DownComID = strings.TrimSpace(req.DownComID)
	if req.UpComID == "" {
		errorResponse(w, http.StatusBadRequest, "up_comid required")
		return
	}
	if req.DownComID == "" {
		errorResponse(w, http.StatusBadRequest, "down_comid required")
		return
	}

	ctx := r.Context()

	var (
		reachID                                    string
		dbStartLng, dbStartLat, dbEndLng, dbEndLat *float64
	)
	err := h.db.QueryRow(ctx, `
		SELECT id,
		       ST_X(start_point::geometry), ST_Y(start_point::geometry),
		       ST_X(end_point::geometry),   ST_Y(end_point::geometry)
		FROM reaches WHERE slug = $1
	`, slug).Scan(&reachID, &dbStartLng, &dbStartLat, &dbEndLng, &dbEndLat)
	if err != nil {
		errorResponse(w, http.StatusNotFound, fmt.Sprintf("reach %q not found", slug))
		return
	}

	// Prefer body-supplied coordinates; fall back to stored start/end point.
	startLng := first(req.StartLng, dbStartLng)
	startLat := first(req.StartLat, dbStartLat)
	endLng   := first(req.EndLng,   dbEndLng)
	endLat   := first(req.EndLat,   dbEndLat)

	if startLat == nil || startLng == nil || endLat == nil || endLng == nil {
		errorResponse(w, http.StatusBadRequest, "start and end coordinates are required (provide start_lat/start_lng/end_lat/end_lng or set start_point/end_point on the reach first)")
		return
	}

	// Persist body-supplied coordinates to start_point/end_point if provided.
	if req.StartLat != nil && req.StartLng != nil && req.EndLat != nil && req.EndLng != nil && !req.DryRun {
		_, _ = h.db.Exec(ctx, `
			UPDATE reaches
			SET start_point = ST_SetSRID(ST_MakePoint($2, $3), 4326)::geography,
			    end_point   = ST_SetSRID(ST_MakePoint($4, $5), 4326)::geography
			WHERE id = $1
		`, reachID, *req.StartLng, *req.StartLat, *req.EndLng, *req.EndLat)
		fillReachState(ctx, h.db, slug, *req.StartLat, *req.StartLng)
	}

	if err := kmlimport.SyncCenterlineNLDIByComID(ctx, h.db, slug,
		req.UpComID, req.DownComID,
		*startLng, *startLat, *endLng, *endLat,
		req.DryRun); err != nil {
		errorResponse(w, http.StatusBadGateway, fmt.Sprintf("nldi centerline: %v", err))
		return
	}

	if req.DryRun {
		jsonResponse(w, http.StatusOK, map[string]any{"dry_run": true})
		return
	}

	var lengthMi *float64
	var startComID, endComID *string
	_ = h.db.QueryRow(ctx, `
		SELECT length_mi, start_comid, end_comid FROM reaches WHERE id = $1
	`, reachID).Scan(&lengthMi, &startComID, &endComID)

	h.warmCache()
	jsonResponse(w, http.StatusOK, map[string]any{
		"slug":        slug,
		"length_mi":   lengthMi,
		"start_comid": startComID,
		"end_comid":   endComID,
	})
}

// first returns the first non-nil *float64 pointer.
func first(a, b *float64) *float64 {
	if a != nil {
		return a
	}
	return b
}

// GenerateDescription handles POST /api/v1/admin/reaches/{slug}/generate-description
//
// Asks Claude to write a 1-2 paragraph description for the reach using its
// training knowledge. Returns the generated text without storing it — the
// admin reviews and saves via a separate update. If the Anthropic key is not
// configured, returns 501.
func (h *NLDIHandler) GenerateDescription(w http.ResponseWriter, r *http.Request) {
	if h.anthropicKey == "" {
		errorResponse(w, http.StatusNotImplemented, "AI description generation not configured (ANTHROPIC_API_KEY missing)")
		return
	}
	slug := chi.URLParam(r, "slug")
	ctx := r.Context()

	var name, riverName, commonName string
	var classMin, classMax *float64
	if err := h.db.QueryRow(ctx, `
		SELECT name, COALESCE(river_name,''), COALESCE(common_name,''), class_min, class_max
		FROM reaches WHERE slug = $1
	`, slug).Scan(&name, &riverName, &commonName, &classMin, &classMax); err != nil {
		errorResponse(w, http.StatusNotFound, fmt.Sprintf("reach %q not found", slug))
		return
	}

	text, err := ai.GenerateReachDescription(ctx, h.anthropicKey, name, riverName, commonName, classMin, classMax)
	if err != nil {
		errorResponse(w, http.StatusBadGateway, fmt.Sprintf("generate description: %v", err))
		return
	}

	jsonResponse(w, http.StatusOK, map[string]any{"description": text})
}

// PatchReach handles PATCH /api/v1/admin/reaches/{slug}
//
// Accepts { description?: string } and updates only those fields.
// Currently only description is patchable via this endpoint.
func (h *NLDIHandler) PatchReach(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	var req struct {
		Description *string `json:"description"`
		RiverOrder  *int16  `json:"river_order"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		errorResponse(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	ctx := r.Context()
	var setClauses []string
	var args []any
	n := 1
	if req.Description != nil {
		setClauses = append(setClauses, fmt.Sprintf("description = $%d", n))
		args = append(args, req.Description)
		n++
	}
	if req.RiverOrder != nil {
		setClauses = append(setClauses, fmt.Sprintf("river_order = $%d", n))
		args = append(args, *req.RiverOrder)
		n++
	}
	if len(setClauses) == 0 {
		jsonResponse(w, http.StatusOK, map[string]any{"slug": slug})
		return
	}
	args = append(args, slug)
	tag, err := h.db.Exec(ctx,
		fmt.Sprintf("UPDATE reaches SET %s WHERE slug = $%d", strings.Join(setClauses, ", "), n),
		args...,
	)
	if err != nil {
		errorResponse(w, http.StatusInternalServerError, fmt.Sprintf("update: %v", err))
		return
	}
	if tag.RowsAffected() == 0 {
		errorResponse(w, http.StatusNotFound, fmt.Sprintf("reach %q not found", slug))
		return
	}
	jsonResponse(w, http.StatusOK, map[string]any{"slug": slug})
}

// PreviewCenterline handles GET /api/v1/admin/nldi/preview-centerline
// Returns the GeoJSON LineString between two NHD ComIDs without writing to the
// database. When start_lat/start_lng/end_lat/end_lng are provided the line is
// trimmed via PostGIS ST_LineSubstring to match the exact reach extent.
func (h *NLDIHandler) PreviewCenterline(w http.ResponseWriter, r *http.Request) {
	upComID   := r.URL.Query().Get("up_comid")
	downComID := r.URL.Query().Get("down_comid")
	if upComID == "" || downComID == "" {
		errorResponse(w, http.StatusBadRequest, "up_comid and down_comid are required")
		return
	}
	geojson, err := kmlimport.FetchCenterlinePreview(r.Context(), upComID, downComID)
	if err != nil {
		errorResponse(w, http.StatusUnprocessableEntity, err.Error())
		return
	}

	// Apply trim if coordinates supplied.
	q := r.URL.Query()
	if q.Get("start_lat") != "" && q.Get("end_lat") != "" {
		startLat, e1 := strconv.ParseFloat(q.Get("start_lat"), 64)
		startLng, e2 := strconv.ParseFloat(q.Get("start_lng"), 64)
		endLat,   e3 := strconv.ParseFloat(q.Get("end_lat"),   64)
		endLng,   e4 := strconv.ParseFloat(q.Get("end_lng"),   64)
		if e1 == nil && e2 == nil && e3 == nil && e4 == nil {
			if trimmed, err := trimLineGeoJSON(r.Context(), h.db, geojson, startLng, startLat, endLng, endLat); err == nil {
				geojson = trimmed
			}
		}
	}

	jsonResponse(w, http.StatusOK, map[string]any{"geojson": json.RawMessage(geojson)})
}

// trimLineGeoJSON applies PostGIS ST_LineSubstring to trim the raw GeoJSON line
// to the closest points on the line to putIn and takeOut coordinates.
func trimLineGeoJSON(ctx context.Context, db *pgxpool.Pool, geojson string, putInLon, putInLat, takeOutLon, takeOutLat float64) (string, error) {
	var result string
	err := db.QueryRow(ctx, `
		SELECT ST_AsGeoJSON(
			ST_LineSubstring(
				line,
				LEAST(ST_LineLocatePoint(line, put_pt), ST_LineLocatePoint(line, take_pt)),
				GREATEST(ST_LineLocatePoint(line, put_pt), ST_LineLocatePoint(line, take_pt))
			)
		)
		FROM (
			SELECT
				ST_GeomFromGeoJSON($1)                                      AS line,
				ST_ClosestPoint(ST_GeomFromGeoJSON($1),
				    ST_SetSRID(ST_MakePoint($2, $3), 4326))                 AS put_pt,
				ST_ClosestPoint(ST_GeomFromGeoJSON($1),
				    ST_SetSRID(ST_MakePoint($4, $5), 4326))                 AS take_pt
		) sub
	`, geojson, putInLon, putInLat, takeOutLon, takeOutLat).Scan(&result)
	return result, err
}

// buildSlug produces a URL-safe slug from river name + reach name,
// matching the KML importer convention.
func buildSlug(riverName, reachName string) string {
	r := kmlimport.Slugify(riverName)
	n := kmlimport.Slugify(reachName)
	if r == "" {
		return n
	}
	if n == "" {
		return r
	}
	return r + "-" + n
}

// fillReachState queries TIGERweb for the US state at the given coordinate and
// updates reaches.state_abbr. Best-effort — failures are logged and not fatal.
func fillReachState(ctx context.Context, db *pgxpool.Pool, reachSlug string, lat, lng float64) {
	stateAbbr, err := nldi.StateAt(ctx, lat, lng)
	if err != nil {
		log.Printf("fillReachState %s: %v", reachSlug, err)
		return
	}
	if stateAbbr == "" {
		return
	}
	if _, err := db.Exec(ctx, `UPDATE reaches SET state_abbr = $1 WHERE slug = $2`, stateAbbr, reachSlug); err != nil {
		log.Printf("fillReachState %s: update: %v", reachSlug, err)
	}
}

// ── Shared helpers ────────────────────────────────────────────────────────────

func parseLatLng(r *http.Request) (lat, lng float64, err error) {
	latStr := r.URL.Query().Get("lat")
	lngStr := r.URL.Query().Get("lng")
	if latStr == "" || lngStr == "" {
		return 0, 0, fmt.Errorf("lat and lng are required")
	}
	lat, err = strconv.ParseFloat(latStr, 64)
	if err != nil || lat < -90 || lat > 90 {
		return 0, 0, fmt.Errorf("invalid lat: %s", latStr)
	}
	lng, err = strconv.ParseFloat(lngStr, 64)
	if err != nil || lng < -180 || lng > 180 {
		return 0, 0, fmt.Errorf("invalid lng: %s", lngStr)
	}
	return lat, lng, nil
}
