package handlers

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/h2oflow/h2oflow/apps/api/internal/ai"
	"github.com/jackc/pgx/v5/pgxpool"
)

// toucher is the narrow poller interface GaugeHandler needs — keeps the handler
// package free of a direct dependency on the full poller implementation.
type toucher interface {
	TouchRequested(ctx context.Context, gaugeID string)
}

// GaugeHandler handles gauge-related HTTP routes.
type GaugeHandler struct {
	db       *pgxpool.Pool
	enricher *ai.SearchEnricher // nil = AI enrichment disabled
	poller   toucher            // nil = demand-polling disabled
}

func NewGaugeHandler(db *pgxpool.Pool, enricher *ai.SearchEnricher, poller toucher) *GaugeHandler {
	return &GaugeHandler{db: db, enricher: enricher, poller: poller}
}

// Search handles GET /api/v1/gauges/search
//
// Query params (all optional, combinable):
//
//	q=arkansas           text search on name and external_id
//	bbox=-106,38,-104,40 bounding box: west,south,east,north
//	lat=38.5&lon=-105.8&radius_mi=25  proximity search
//	source=usgs,dwr      filter by source (comma-separated)
//	limit=100            max results (1–500, default 100)
//
// Returns GeoJSON FeatureCollection sorted by prominence_score DESC.
// MapLibre can consume this directly as a clustered GeoJSON source.
func (h *GaugeHandler) Search(w http.ResponseWriter, r *http.Request) {
	p, err := parseSearchParams(r)
	if err != nil {
		errorResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	// AI search enrichment — expands nicknames, resolves hint IDs.
	// Failures are non-fatal: we log and fall back to plain text search.
	if h.enricher != nil && p.Q != "" {
		if enrich, err := h.enricher.Enrich(r.Context(), p.Q); err != nil {
			log.Printf("search enrichment: %v", err)
		} else if enrich != nil && !enrich.Irrelevant {
			p.ExtraTerms = enrich.Terms
			p.HintIDs = enrich.HintIDs
		}
	}

	rows, err := h.querySearch(r, p)
	if err != nil {
		errorResponse(w, http.StatusInternalServerError, "query failed")
		return
	}
	defer rows.Close()

	features := make([]Feature, 0)
	for rows.Next() {
		var (
			id                  string
			externalID          string
			source              string
			name                *string
			status              string
			featured            bool
			prominenceScore     float64
			reachID             *string
			reachNamesRaw       []string
			reachSlugsRaw       []string
			reachRelationship   *string
			lastReadingAt       *time.Time
			lng                 float64
			lat                 float64
			stateAbbr           *string
			basinName           *string
			watershedName       *string
			riverName           *string
			currentCFS          *float64
			flowStatus          string
			flowBandLabel       *string
			pollTier            string
		)
		if err := rows.Scan(
			&id, &externalID, &source, &name, &status,
			&featured, &prominenceScore, &reachID, &reachNamesRaw, &reachSlugsRaw, &reachRelationship, &lastReadingAt,
			&lng, &lat, &stateAbbr, &basinName, &watershedName, &riverName,
			&currentCFS, &flowStatus, &flowBandLabel, &pollTier,
		); err != nil {
			continue
		}

		features = append(features, Feature{
			Type:     "Feature",
			Geometry: PointGeometry{Type: "Point", Coordinates: [2]float64{lng, lat}},
			Properties: map[string]any{
				"id":                 id,
				"external_id":        externalID,
				"source":             source,
				"name":               name,
				"status":             status,
				"featured":           featured,
				"prominence_score":   prominenceScore,
				"reach_id":           reachID,
				"reach_name":         combineReachNames(reachNamesRaw),
				"reach_names":        reachNamesRaw,
				"reach_slug":         firstOrNil(reachSlugsRaw),
				"reach_slugs":        reachSlugsRaw,
				"reach_relationship": reachRelationship,
				"last_reading_at":    lastReadingAt,
				"state_abbr":         stateAbbr,
				"basin_name":         basinName,
				"watershed_name":     watershedName,
				"river_name":         riverName,
				"current_cfs":        currentCFS,
				"flow_status":        flowStatus,
				"flow_band_label":    flowBandLabel,
				"poll_tier":          pollTier,
			},
		})
	}
	if err := rows.Err(); err != nil {
		errorResponse(w, http.StatusInternalServerError, "scan failed")
		return
	}

	jsonResponse(w, http.StatusOK, newFeatureCollection(features))

	// Touch every returned gauge so cold gauges enter the demand-polling window.
	// Fire-and-forget — the user already has their search results.
	if h.poller != nil {
		go func() {
			ctx := context.Background()
			for _, f := range features {
				if id, ok := f.Properties["id"].(string); ok {
					h.poller.TouchRequested(ctx, id)
				}
			}
		}()
	}
}

// GetReadings handles GET /api/v1/gauges/{id}/readings
//
// Returns up to `limit` readings from the 48-hour rolling cache, newest first.
// The frontend uses this to populate the gauge graph (uPlot).
//
// Query params:
//
//	limit=96   max rows (1–500, default 96 — 48h at 30min intervals)
//	since=     ISO 8601 timestamp — only return readings after this time
func (h *GaugeHandler) GetReadings(w http.ResponseWriter, r *http.Request) {
	gaugeID := chi.URLParam(r, "id")
	if gaugeID == "" {
		errorResponse(w, http.StatusBadRequest, "gauge id is required")
		return
	}

	q := r.URL.Query()
	limit := clampInt(parseIntOr(q.Get("limit"), 96), 1, 500)

	var since *time.Time
	if raw := q.Get("since"); raw != "" {
		t, err := time.Parse(time.RFC3339, raw)
		if err != nil {
			errorResponse(w, http.StatusBadRequest, "since must be RFC3339 (e.g. 2024-01-01T00:00:00Z)")
			return
		}
		since = &t
	}

	// Touch for demand polling — user is actively viewing this gauge.
	if h.poller != nil {
		go h.poller.TouchRequested(context.Background(), gaugeID)
	}

	var rows interface {
		Next() bool
		Scan(dest ...any) error
		Close()
		Err() error
	}
	var err error

	if since != nil {
		rows, err = h.db.Query(r.Context(), `
			SELECT value, timestamp
			FROM gauge_readings
			WHERE gauge_id = $1
			  AND timestamp > $2
			ORDER BY timestamp DESC
			LIMIT $3
		`, gaugeID, since, limit)
	} else {
		rows, err = h.db.Query(r.Context(), `
			SELECT value, timestamp
			FROM gauge_readings
			WHERE gauge_id = $1
			ORDER BY timestamp DESC
			LIMIT $2
		`, gaugeID, limit)
	}
	if err != nil {
		errorResponse(w, http.StatusInternalServerError, "query failed")
		return
	}
	defer rows.Close()

	type reading struct {
		CFS       float64   `json:"cfs"`
		Timestamp time.Time `json:"timestamp"`
	}
	results := make([]reading, 0)
	for rows.Next() {
		var rd reading
		if err := rows.Scan(&rd.CFS, &rd.Timestamp); err != nil {
			continue
		}
		results = append(results, rd)
	}
	if err := rows.Err(); err != nil {
		errorResponse(w, http.StatusInternalServerError, "scan failed")
		return
	}

	jsonResponse(w, http.StatusOK, results)
}

// GetFlowRanges handles GET /api/v1/gauges/{id}/flow-ranges
//
// Returns the flow range bands for a gauge, sorted min_cfs ASC.
// The frontend overlays these as colored bands on the gauge graph.
//
// Query params:
//
//	craft=general   craft type filter (default "general")
func (h *GaugeHandler) GetFlowRanges(w http.ResponseWriter, r *http.Request) {
	gaugeID := chi.URLParam(r, "id")
	if gaugeID == "" {
		errorResponse(w, http.StatusBadRequest, "gauge id is required")
		return
	}

	craft := r.URL.Query().Get("craft")
	if craft == "" {
		craft = "general"
	}

	// After migration 039, flow_ranges are per-reach. For the gauge endpoint
	// (used by sparklines on the dashboard), return ranges for the alphabetically-
	// first reach that uses this gauge as its primary gauge.
	rows, err := h.db.Query(r.Context(), `
		SELECT
			fr.label,
			fr.min_cfs,
			fr.max_cfs,
			fr.craft_type,
			fr.class_modifier,
			fr.source_url,
			fr.data_source,
			fr.ai_confidence,
			fr.verified
		FROM flow_ranges fr
		JOIN reaches rch ON rch.id = fr.reach_id
		WHERE rch.primary_gauge_id = $1
		  AND fr.craft_type         = $2
		  AND rch.id = (
			  SELECT id FROM reaches
			  WHERE primary_gauge_id = $1
			  ORDER BY slug LIMIT 1
		  )
		ORDER BY fr.min_cfs ASC NULLS FIRST
	`, gaugeID, craft)
	if err != nil {
		errorResponse(w, http.StatusInternalServerError, "query failed")
		return
	}
	defer rows.Close()

	type flowRange struct {
		Label        string   `json:"label"`
		MinCFS       *float64 `json:"min_cfs"`
		MaxCFS       *float64 `json:"max_cfs"`
		CraftType    string   `json:"craft_type"`
		ClassMod     *float64 `json:"class_modifier"`
		SourceURL    *string  `json:"source_url,omitempty"`
		DataSource   string   `json:"data_source"`
		AIConfidence *int     `json:"ai_confidence,omitempty"`
		Verified     bool     `json:"verified"`
	}
	results := make([]flowRange, 0)
	for rows.Next() {
		var fr flowRange
		if err := rows.Scan(
			&fr.Label, &fr.MinCFS, &fr.MaxCFS,
			&fr.CraftType, &fr.ClassMod, &fr.SourceURL,
			&fr.DataSource, &fr.AIConfidence, &fr.Verified,
		); err != nil {
			continue
		}
		results = append(results, fr)
	}
	if err := rows.Err(); err != nil {
		errorResponse(w, http.StatusInternalServerError, "scan failed")
		return
	}

	jsonResponse(w, http.StatusOK, results)
}

// --- Search params ----------------------------------------------------------

type searchParams struct {
	Q          string
	BBox       *searchBBox
	Radius     *searchRadius
	Sources    []string
	Limit      int
	ExtraTerms []string // AI-derived additional search terms
	HintIDs    []string // AI-derived external_ids to boost to top
}

type searchBBox struct {
	West, South, East, North float64
}

type searchRadius struct {
	Lat, Lon float64
	Meters   float64 // converted from radius_mi
}

func parseSearchParams(r *http.Request) (searchParams, error) {
	q := r.URL.Query()
	p := searchParams{
		Q:     strings.TrimSpace(q.Get("q")),
		Limit: clampInt(parseIntOr(q.Get("limit"), 100), 1, 500),
	}

	// bbox=west,south,east,north
	if raw := q.Get("bbox"); raw != "" {
		parts := strings.Split(raw, ",")
		if len(parts) != 4 {
			return p, fmt.Errorf("bbox must be west,south,east,north")
		}
		floats, err := parseFloats(parts)
		if err != nil {
			return p, fmt.Errorf("bbox: %w", err)
		}
		p.BBox = &searchBBox{
			West: floats[0], South: floats[1],
			East: floats[2], North: floats[3],
		}
	}

	// lat=X&lon=Y&radius_mi=Z
	latStr, lonStr, radStr := q.Get("lat"), q.Get("lon"), q.Get("radius_mi")
	if latStr != "" || lonStr != "" || radStr != "" {
		if latStr == "" || lonStr == "" || radStr == "" {
			return p, fmt.Errorf("lat, lon, and radius_mi are all required for proximity search")
		}
		lat, err1 := strconv.ParseFloat(latStr, 64)
		lon, err2 := strconv.ParseFloat(lonStr, 64)
		mi, err3 := strconv.ParseFloat(radStr, 64)
		if err1 != nil || err2 != nil || err3 != nil {
			return p, fmt.Errorf("lat, lon, radius_mi must be numbers")
		}
		p.Radius = &searchRadius{Lat: lat, Lon: lon, Meters: mi * 1609.34}
	}

	// source=usgs,dwr
	if raw := q.Get("source"); raw != "" {
		for _, s := range strings.Split(raw, ",") {
			if s = strings.TrimSpace(s); s != "" {
				p.Sources = append(p.Sources, s)
			}
		}
	}

	return p, nil
}

// --- SQL query builder ------------------------------------------------------

func (h *GaugeHandler) querySearch(r *http.Request, p searchParams) (interface {
	Next() bool
	Scan(dest ...any) error
	Close()
	Err() error
}, error) {
	args := []any{}
	where := []string{
		"g.status != 'retired'",
	}

	addArg := func(v any) int {
		args = append(args, v)
		return len(args)
	}

	// Text search — substring (ILIKE) + trigram similarity fallback.
	//
	// Pass 1: exact substring match on name, external_id, and linked reach names.
	//         Fast via GIN trigram index (migration 038).
	// Pass 2: trigram similarity on name when pg_trgm is available.
	//         Handles compound-word normalization ("Elevenmile" ≈ "Eleven Mile")
	//         and minor typos ("Grore Canyon" ≈ "Gore Canyon").
	//         Similarity threshold 0.25 — loose enough to catch spacing/hypen
	//         differences without surfacing irrelevant results.
	//
	// AI-derived ExtraTerms are OR-ed in alongside the original query.
	// qArgN tracks the arg index for p.Q so the SELECT can reuse it to
	// look up the relationship for the matched reach.
	var qArgN int
	if p.Q != "" || len(p.ExtraTerms) > 0 {
		var textClauses []string
		for _, term := range append([]string{p.Q}, p.ExtraTerms...) {
			if term == "" {
				continue
			}
			likeN := addArg("%" + term + "%")
			termN := addArg(term) // raw term for similarity()
			if qArgN == 0 && term == p.Q {
				qArgN = likeN // capture for SELECT subquery below
			}
			// Association table subquery: returns all gauges linked to ANY reach
			// whose name matches — including upstream/downstream indicators.
			// e.g. searching "Foxton" surfaces PLAGRACO even though its station
			// name says "North Fork S Platte at Grant", because it has an
			// upstream_indicator association with the Foxton reach.
			textClauses = append(textClauses, fmt.Sprintf(
				`(g.name ILIKE $%d OR g.external_id ILIKE $%d
				  OR similarity(g.name, $%d) > 0.25
				  OR EXISTS (
					SELECT 1 FROM gauge_reach_associations gra
					JOIN reaches ra ON ra.id = gra.reach_id
					WHERE gra.gauge_id = g.id
					  AND (ra.name ILIKE $%d
					    OR ra.common_name ILIKE $%d
					    OR ra.river_name  ILIKE $%d
					    OR similarity(ra.name, $%d) > 0.25
					    OR similarity(COALESCE(ra.common_name, ''), $%d) > 0.25)
				  ))`, likeN, likeN, termN, likeN, likeN, likeN, termN, termN))
		}
		if len(textClauses) > 0 {
			where = append(where, "("+strings.Join(textClauses, " OR ")+")")
		}
	}

	// Bounding box
	if p.BBox != nil {
		w := addArg(p.BBox.West)
		s := addArg(p.BBox.South)
		e := addArg(p.BBox.East)
		n := addArg(p.BBox.North)
		where = append(where, fmt.Sprintf(
			"ST_Within(g.location::geometry, ST_MakeEnvelope($%d,$%d,$%d,$%d,4326))",
			w, s, e, n))
	}

	// Proximity radius
	if p.Radius != nil {
		lon := addArg(p.Radius.Lon)
		lat := addArg(p.Radius.Lat)
		m := addArg(p.Radius.Meters)
		where = append(where, fmt.Sprintf(
			"ST_DWithin(g.location::geography, ST_MakePoint($%d,$%d)::geography, $%d)",
			lon, lat, m))
	}

	// Source filter
	if len(p.Sources) > 0 {
		n := addArg(p.Sources)
		where = append(where, fmt.Sprintf("g.source = ANY($%d)", n))
	}

	limitN := addArg(p.Limit)

	// AI hint IDs sort before all other results; within each group sort by prominence.
	orderBy := "g.prominence_score DESC"
	if len(p.HintIDs) > 0 {
		hintN := addArg(p.HintIDs)
		orderBy = fmt.Sprintf("(CASE WHEN g.external_id = ANY($%d) THEN 0 ELSE 1 END), g.prominence_score DESC", hintN)
	}

	// When a reach-name text search is active, show the relationship for the
	// matched reach (e.g. "downstream_indicator" when searching "Foxton" and
	// PLASPLCO is linked as downstream). Falls back to gauges.reach_relationship.
	reachRelCol := "g.reach_relationship"
	if qArgN > 0 {
		// qArgN is the ILIKE arg ($N = '%term%'); the raw-term arg for similarity is qArgN+1.
		reachRelCol = fmt.Sprintf(`COALESCE(
				(SELECT gra.relationship FROM gauge_reach_associations gra
				 JOIN reaches ra ON ra.id = gra.reach_id
				 WHERE gra.gauge_id = g.id
				   AND (ra.name ILIKE $%d OR similarity(ra.name, $%d) > 0.25)
				 LIMIT 1),
				g.reach_relationship
			)`, qArgN, qArgN+1)
	}

	sql := fmt.Sprintf(`
		SELECT
			g.id,
			g.external_id,
			g.source,
			g.name,
			g.status,
			g.featured,
			g.prominence_score,
			g.reach_id,
			ARRAY(
				SELECT ra.name FROM reaches ra
				WHERE ra.primary_gauge_id = g.id
				ORDER BY ra.name LIMIT 4
			)                  AS reach_names,
			ARRAY(
				SELECT ra.slug FROM reaches ra
				WHERE ra.primary_gauge_id = g.id
				ORDER BY ra.name LIMIT 4
			)                  AS reach_slugs,
			%s                 AS reach_relationship,
			g.last_reading_at,
			COALESCE(ST_X(g.location::geometry), 0) AS lng,
			COALESCE(ST_Y(g.location::geometry), 0) AS lat,
			g.state_abbr,
			g.basin_name,
			g.watershed_name,
			(SELECT ra.river_name FROM reaches ra
			 WHERE ra.primary_gauge_id = g.id AND ra.river_name IS NOT NULL
			 ORDER BY ra.name LIMIT 1
			) AS river_name,
			g.current_cfs,
			COALESCE(fr_band.flow_status, 'unknown') AS flow_status,
			fr_band.label                            AS flow_band_label,
			CASE
				WHEN g.reach_id IS NOT NULL                                     THEN 'trusted'
				WHEN g.last_requested_at > NOW() - INTERVAL '7 days'           THEN 'demand'
				ELSE                                                                 'cold'
			END AS poll_tier
		FROM gauges g
		LEFT JOIN LATERAL (
			SELECT fr.label,
			       CASE
			           WHEN fr.label = 'runnable'          THEN 'runnable'
			           WHEN fr.label = 'below_recommended' THEN 'low'
			           WHEN fr.label = 'above_recommended' THEN 'flood'
			           WHEN fr.label IN ('fun', 'optimal')   THEN 'runnable'
			           WHEN fr.label IN ('minimum', 'pushy') THEN 'caution'
			           WHEN fr.label = 'too_low'             THEN 'low'
			           WHEN fr.label IN ('high', 'flood')    THEN 'flood'
			           ELSE 'unknown'
			       END AS flow_status
			FROM flow_ranges fr
			JOIN reaches rch ON rch.id = fr.reach_id
			WHERE rch.primary_gauge_id = g.id
			  AND rch.id = (
			      SELECT id FROM reaches
			      WHERE primary_gauge_id = g.id
			      ORDER BY slug LIMIT 1
			  )
			  AND fr.craft_type = 'general'
			  AND (fr.min_cfs IS NULL OR g.current_cfs >= fr.min_cfs)
			  AND (fr.max_cfs IS NULL OR g.current_cfs < fr.max_cfs)
			ORDER BY fr.min_cfs ASC NULLS FIRST
			LIMIT 1
		) fr_band ON TRUE
		WHERE %s
		ORDER BY %s
		LIMIT $%d
	`, reachRelCol, strings.Join(where, " AND "), orderBy, limitN)

	return h.db.Query(r.Context(), sql, args...)
}

// --- Helpers ----------------------------------------------------------------

// combineReachNames joins primary-associated reach names into a single display
// string (e.g. "Bailey / Foxton"). Capped at 3 names; a 4th triggers " / …".
func firstOrNil(ss []string) *string {
	if len(ss) == 0 {
		return nil
	}
	return &ss[0]
}

func combineReachNames(names []string) *string {
	if len(names) == 0 {
		return nil
	}
	suffix := ""
	if len(names) > 3 {
		names = names[:3]
		suffix = " / \u2026"
	}
	s := strings.Join(names, " / ") + suffix
	return &s
}

func parseFloats(ss []string) ([]float64, error) {
	out := make([]float64, len(ss))
	for i, s := range ss {
		f, err := strconv.ParseFloat(strings.TrimSpace(s), 64)
		if err != nil {
			return nil, fmt.Errorf("invalid float %q", s)
		}
		out[i] = f
	}
	return out, nil
}

// BatchGet handles GET /api/v1/gauges/batch?ids=uuid1,uuid2,...
//
// Returns a GeoJSON FeatureCollection of fresh gauge data for the given IDs.
// The frontend calls this on dashboard mount to refresh cached watchlist data
// (current_cfs, watershed_name, etc.) without asking the user to re-add gauges.
func (h *GaugeHandler) BatchGet(w http.ResponseWriter, r *http.Request) {
	raw := strings.TrimSpace(r.URL.Query().Get("ids"))
	if raw == "" {
		jsonResponse(w, http.StatusOK, newFeatureCollection(nil))
		return
	}
	ids := strings.Split(raw, ",")
	if len(ids) > 200 {
		ids = ids[:200]
	}

	rows, err := h.db.Query(r.Context(), `
		SELECT
			g.id,
			g.external_id,
			g.source,
			g.name,
			g.status,
			g.featured,
			g.prominence_score,
			g.reach_id,
			ARRAY(
				SELECT ra.name FROM reaches ra
				WHERE ra.primary_gauge_id = g.id
				ORDER BY ra.name LIMIT 4
			)                  AS reach_names,
			ARRAY(
				SELECT ra.slug FROM reaches ra
				WHERE ra.primary_gauge_id = g.id
				ORDER BY ra.name LIMIT 4
			)                  AS reach_slugs,
			g.reach_relationship,
			g.last_reading_at,
			ST_X(g.location::geometry) AS lng,
			ST_Y(g.location::geometry) AS lat,
			g.state_abbr,
			g.basin_name,
			g.watershed_name,
			(SELECT ra.river_name FROM reaches ra
			 WHERE ra.primary_gauge_id = g.id AND ra.river_name IS NOT NULL
			 ORDER BY ra.name LIMIT 1
			) AS river_name,
			g.current_cfs,
			COALESCE(fr_band.flow_status, 'unknown') AS flow_status,
			fr_band.label                            AS flow_band_label,
			CASE
				WHEN g.reach_id IS NOT NULL                              THEN 'trusted'
				WHEN g.last_requested_at > NOW() - INTERVAL '7 days'     THEN 'demand'
				ELSE                                                           'cold'
			END AS poll_tier
		FROM gauges g
		LEFT JOIN LATERAL (
			SELECT fr.label,
			       CASE
			           WHEN fr.label = 'runnable'          THEN 'runnable'
			           WHEN fr.label = 'below_recommended' THEN 'low'
			           WHEN fr.label = 'above_recommended' THEN 'flood'
			           WHEN fr.label IN ('fun', 'optimal')   THEN 'runnable'
			           WHEN fr.label IN ('minimum', 'pushy') THEN 'caution'
			           WHEN fr.label = 'too_low'             THEN 'low'
			           WHEN fr.label IN ('high', 'flood')    THEN 'flood'
			           ELSE 'unknown'
			       END AS flow_status
			FROM flow_ranges fr
			JOIN reaches rch ON rch.id = fr.reach_id
			WHERE rch.primary_gauge_id = g.id
			  AND rch.id = (
			      SELECT id FROM reaches
			      WHERE primary_gauge_id = g.id
			      ORDER BY slug LIMIT 1
			  )
			  AND fr.craft_type = 'general'
			  AND (fr.min_cfs IS NULL OR g.current_cfs >= fr.min_cfs)
			  AND (fr.max_cfs IS NULL OR g.current_cfs < fr.max_cfs)
			ORDER BY fr.min_cfs ASC NULLS FIRST
			LIMIT 1
		) fr_band ON TRUE
		WHERE g.id = ANY($1)
	`, ids)
	if err != nil {
		errorResponse(w, http.StatusInternalServerError, "query failed")
		return
	}
	defer rows.Close()

	features := make([]Feature, 0)
	for rows.Next() {
		var (
			id                string
			externalID        string
			source            string
			name              *string
			status            string
			featured          bool
			prominenceScore   float64
			reachID           *string
			reachNamesRaw     []string
			reachSlugsRaw     []string
			reachRelationship *string
			lastReadingAt     *time.Time
			lng               *float64
			lat               *float64
			stateAbbr         *string
			basinName         *string
			watershedName     *string
			riverName         *string
			currentCFS        *float64
			flowStatus        string
			flowBandLabel     *string
			pollTier          string
		)
		if err := rows.Scan(
			&id, &externalID, &source, &name, &status,
			&featured, &prominenceScore, &reachID, &reachNamesRaw, &reachSlugsRaw, &reachRelationship, &lastReadingAt,
			&lng, &lat, &stateAbbr, &basinName, &watershedName, &riverName,
			&currentCFS, &flowStatus, &flowBandLabel, &pollTier,
		); err != nil {
			continue
		}
		var geom any
		if lng != nil && lat != nil {
			geom = PointGeometry{Type: "Point", Coordinates: [2]float64{*lng, *lat}}
		}
		features = append(features, Feature{
			Type:     "Feature",
			Geometry: geom,
			Properties: map[string]any{
				"id":                 id,
				"external_id":        externalID,
				"source":             source,
				"name":               name,
				"status":             status,
				"featured":           featured,
				"prominence_score":   prominenceScore,
				"reach_id":           reachID,
				"reach_name":         combineReachNames(reachNamesRaw),
				"reach_names":        reachNamesRaw,
				"reach_slug":         firstOrNil(reachSlugsRaw),
				"reach_slugs":        reachSlugsRaw,
				"reach_relationship": reachRelationship,
				"last_reading_at":    lastReadingAt,
				"state_abbr":         stateAbbr,
				"basin_name":         basinName,
				"watershed_name":     watershedName,
				"river_name":         riverName,
				"current_cfs":        currentCFS,
				"flow_status":        flowStatus,
				"flow_band_label":    flowBandLabel,
				"poll_tier":          pollTier,
			},
		})
	}
	if err := rows.Err(); err != nil {
		errorResponse(w, http.StatusInternalServerError, "scan failed")
		return
	}

	jsonResponse(w, http.StatusOK, newFeatureCollection(features))
}

// GetSeasonalStats handles GET /api/v1/gauges/{id}/seasonal
//
// Returns 12 monthly statistics derived from the USGS period-of-record data.
// The USGS monthly stats endpoint returns one row per year per month; we compute
// mean and percentiles ourselves from the full distribution.
// Data is proxied on demand — not stored. Non-USGS gauges return an empty array.
func (h *GaugeHandler) GetSeasonalStats(w http.ResponseWriter, r *http.Request) {
	gaugeID := chi.URLParam(r, "id")

	var externalID, source string
	if err := h.db.QueryRow(r.Context(),
		`SELECT external_id, source FROM gauges WHERE id = $1`, gaugeID,
	).Scan(&externalID, &source); err != nil {
		jsonResponse(w, http.StatusOK, []any{})
		return
	}
	if source != "usgs" {
		jsonResponse(w, http.StatusOK, []any{})
		return
	}

	url := fmt.Sprintf(
		"https://waterservices.usgs.gov/nwis/stat/?format=rdb&sites=%s"+
			"&statReportType=monthly&statTypeCd=mean&parameterCd=00060",
		externalID,
	)
	resp, err := http.Get(url) //nolint:gosec
	if err != nil {
		log.Printf("seasonal stats fetch for %s: %v", externalID, err)
		jsonResponse(w, http.StatusOK, []any{})
		return
	}
	defer resp.Body.Close()

	stats, err := parseUSGSMonthlyStats(resp.Body)
	if err != nil {
		log.Printf("seasonal stats parse for %s: %v", externalID, err)
		jsonResponse(w, http.StatusOK, []any{})
		return
	}

	jsonResponse(w, http.StatusOK, stats)
}

type monthlyStats struct {
	Month    int      `json:"month"`    // 1–12
	Mean     *float64 `json:"mean"`
	P10      *float64 `json:"p10"`
	P25      *float64 `json:"p25"`
	P50      *float64 `json:"p50"`     // median
	P75      *float64 `json:"p75"`
	P90      *float64 `json:"p90"`
	Count    int      `json:"count"`   // years of record for this month
	Coverage float64  `json:"coverage"` // 0–1 relative to the most-active month
}

// parseUSGSMonthlyStats reads an RDB response where each row is one year+month
// record (agency_cd, site_no, parameter_cd, ts_id, loc_web_ds, year_nu, month_nu, mean_va).
// It groups all values by calendar month and computes mean + percentiles.
func parseUSGSMonthlyStats(r io.Reader) ([]monthlyStats, error) {
	scanner := bufio.NewScanner(r)
	var headers []string
	monthValues := make(map[int][]float64, 12)
	headersDone := false

	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "#") {
			continue
		}
		fields := strings.Split(line, "\t")

		if headers == nil {
			headers = fields
			continue
		}
		// Skip the RDB type line (5s, 15s, 12n …)
		if !headersDone {
			headersDone = true
			continue
		}

		if len(fields) < len(headers) {
			continue
		}
		row := make(map[string]string, len(headers))
		for i, h := range headers {
			if i < len(fields) {
				row[h] = strings.TrimSpace(fields[i])
			}
		}

		month, err := strconv.Atoi(row["month_nu"])
		if err != nil || month < 1 || month > 12 {
			continue
		}
		meanStr := row["mean_va"]
		if meanStr == "" {
			continue
		}
		val, err := strconv.ParseFloat(meanStr, 64)
		if err != nil {
			continue
		}
		monthValues[month] = append(monthValues[month], val)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}

	// Find the maximum count across all months so we can express each month's
	// data density as a coverage ratio (1.0 = as many years as the most active month).
	maxCount := 0
	for _, vals := range monthValues {
		if len(vals) > maxCount {
			maxCount = len(vals)
		}
	}

	// Emit all 12 months — including offline ones (coverage = 0) so the UI
	// can show them as gaps in the seasonal bar chart.
	out := make([]monthlyStats, 0, 12)
	for m := 1; m <= 12; m++ {
		vals := monthValues[m]
		ms := monthlyStats{
			Month: m,
			Count: len(vals),
		}
		if maxCount > 0 {
			ms.Coverage = float64(len(vals)) / float64(maxCount)
		}
		if len(vals) > 0 {
			sortFloats(vals)
			mean := 0.0
			for _, v := range vals {
				mean += v
			}
			mean /= float64(len(vals))
			ms.Mean = &mean
			ms.P10 = pct(vals, 0.10)
			ms.P25 = pct(vals, 0.25)
			ms.P50 = pct(vals, 0.50)
			ms.P75 = pct(vals, 0.75)
			ms.P90 = pct(vals, 0.90)
		}
		out = append(out, ms)
	}
	return out, nil
}

func pct(sorted []float64, p float64) *float64 {
	if len(sorted) == 0 {
		return nil
	}
	idx := int(float64(len(sorted)-1) * p)
	v := sorted[idx]
	return &v
}

func sortFloats(s []float64) {
	// insertion sort — month slices are small (≤100 years)
	for i := 1; i < len(s); i++ {
		key := s[i]
		j := i - 1
		for j >= 0 && s[j] > key {
			s[j+1] = s[j]
			j--
		}
		s[j+1] = key
	}
}

func parseIntOr(s string, def int) int {
	if n, err := strconv.Atoi(s); err == nil {
		return n
	}
	return def
}

func clampInt(n, min, max int) int {
	if n < min {
		return min
	}
	if n > max {
		return max
	}
	return n
}
