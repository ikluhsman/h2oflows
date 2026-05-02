package handlers

import (
	"fmt"
	"log"
	"math"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/h2oflow/h2oflow/apps/api/internal/nldi"
)

// ── BasinMap ──────────────────────────────────────────────────────────────────

type basinReachItem struct {
	Slug        string      `json:"slug"`
	Name        string      `json:"name"`
	RiverName   *string     `json:"river_name"`
	CommonName  *string     `json:"common_name"`
	RiverOrder  *int16      `json:"river_order"`
	ClassMin    *float64    `json:"class_min"`
	ClassMax    *float64    `json:"class_max"`
	AnchorComID *string     `json:"anchor_comid"`
	StartComID  *string     `json:"start_comid"`
	EndComID    *string     `json:"end_comid"`
	Centerline  rawGeometry `json:"centerline"`
	StartPoint  *[2]float64 `json:"start_point"`
	EndPoint    *[2]float64 `json:"end_point"`
	FlowStatus  string      `json:"flow_status"`
}

type basinMapResponse struct {
	BasinSlug string           `json:"basin_slug"`
	Reaches   []basinReachItem `json:"reaches"`
}

// BasinMap handles GET /api/v1/reaches/basin/{slug}/map
//
// Returns centerlines, class, comIDs, and flow status for a set of reaches
// identified by the ?slugs= comma-separated list. The {slug} path param is
// the normalized basin name (used for caching/telemetry; not a DB filter).
func (h *ReachHandler) BasinMap(w http.ResponseWriter, r *http.Request) {
	basinSlug := chi.URLParam(r, "slug")
	empty := basinMapResponse{BasinSlug: basinSlug, Reaches: []basinReachItem{}}

	slugs := parseSlugsParam(r.URL.Query().Get("slugs"))
	if len(slugs) == 0 {
		jsonResponse(w, http.StatusOK, empty)
		return
	}

	rows, err := h.db.Query(r.Context(), `
		WITH latest_reading AS (
			SELECT DISTINCT ON (gauge_id)
				gauge_id, value
			FROM gauge_readings
			WHERE timestamp > NOW() - INTERVAL '48 hours'
			ORDER BY gauge_id, timestamp DESC
		)
		SELECT
			r.slug,
			r.name,
			COALESCE(r.river_name, rv.name) AS river_name,
			r.common_name,
			r.river_order,
			r.class_min,
			COALESCE(
				(SELECT MAX(class_rating) FROM rapids WHERE reach_id = r.id AND class_rating IS NOT NULL),
				r.class_max
			) AS class_max,
			r.anchor_comid,
			r.start_comid,
			r.end_comid,
			ST_AsGeoJSON(r.centerline::geometry) AS centerline,
			ST_X(r.start_point::geometry) AS start_lng,
			ST_Y(r.start_point::geometry) AS start_lat,
			ST_X(r.end_point::geometry)   AS end_lng,
			ST_Y(r.end_point::geometry)   AS end_lat,
			CASE
				WHEN lr.value IS NULL OR fr.label IS NULL  THEN 'unknown'
				WHEN fr.label IN ('running', 'high')       THEN 'runnable'
				WHEN fr.label = 'too_low'                  THEN 'caution'
				WHEN fr.label = 'very_high'                THEN 'flood'
				ELSE                                            'unknown'
			END AS flow_status
		FROM reaches r
		LEFT JOIN rivers rv ON rv.id = r.river_id
		LEFT JOIN gauges g ON g.id = r.primary_gauge_id
		LEFT JOIN latest_reading lr ON lr.gauge_id = g.id
		LEFT JOIN LATERAL (
			SELECT label FROM flow_ranges
			WHERE reach_id = r.id
			  AND craft_type = 'general'
			  AND (min_cfs IS NULL OR lr.value >= min_cfs)
			  AND (max_cfs IS NULL OR lr.value <  max_cfs)
			ORDER BY min_cfs ASC NULLS FIRST
			LIMIT 1
		) fr ON TRUE
		WHERE r.slug = ANY($1)
		  AND r.centerline IS NOT NULL
		ORDER BY r.river_order ASC NULLS LAST,
		         ST_X(r.start_point::geometry) ASC NULLS LAST
	`, slugs)
	if err != nil {
		errorResponse(w, http.StatusInternalServerError, "query failed")
		return
	}
	defer rows.Close()

	items := make([]basinReachItem, 0, len(slugs))
	for rows.Next() {
		var (
			item               basinReachItem
			startLng, startLat *float64
			endLng, endLat     *float64
		)
		if err := rows.Scan(
			&item.Slug, &item.Name, &item.RiverName, &item.CommonName, &item.RiverOrder,
			&item.ClassMin, &item.ClassMax,
			&item.AnchorComID, &item.StartComID, &item.EndComID,
			&item.Centerline,
			&startLng, &startLat, &endLng, &endLat,
			&item.FlowStatus,
		); err != nil {
			continue
		}
		if startLng != nil && startLat != nil {
			item.StartPoint = &[2]float64{*startLng, *startLat}
		}
		if endLng != nil && endLat != nil {
			item.EndPoint = &[2]float64{*endLng, *endLat}
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		errorResponse(w, http.StatusInternalServerError, "scan failed")
		return
	}

	jsonResponse(w, http.StatusOK, basinMapResponse{BasinSlug: basinSlug, Reaches: items})
}

// ── BasinNetwork ──────────────────────────────────────────────────────────────

type networkCacheEntry struct {
	upstream nldi.Collection // UT result for one comid
	cachedAt time.Time
}

var (
	networkCache    sync.Map
	networkCacheTTL = 24 * time.Hour
)

type basinNetworkResponse struct {
	Tributaries   nldi.Collection `json:"tributaries"`
	Mainstem      nldi.Collection `json:"mainstem"`
	NLDIAvailable bool            `json:"nldi_available"`
	NLDIError     string          `json:"nldi_error,omitempty"`
}

// BasinNetwork handles GET /api/v1/reaches/basin/{slug}/network
//
// For each reach slug in ?slugs=, fetches UT (upstream tributaries) for each
// start_comid → shown as blue tributaries, and UT for each end_comid → shown as
// teal mainstem. This shows all flowlines contributing to each reach section.
// Results are cached per (comid, distance) for 24 h.
//
// Flowlines whose nhdplus_comid matches a dashboard reach start/end comid are
// stripped so the overlay doesn't repaint over the colored reach lines.
//
// On NLDI downtime, returns 200 with empty collections and nldi_available=false
// so the map still renders without the network halo.
func (h *ReachHandler) BasinNetwork(w http.ResponseWriter, r *http.Request) {
	emptyOK := basinNetworkResponse{
		Tributaries:   nldi.Collection{Type: "FeatureCollection", Features: []nldi.Feature{}},
		Mainstem:      nldi.Collection{Type: "FeatureCollection", Features: []nldi.Feature{}},
		NLDIAvailable: true,
	}

	slugs := parseSlugsParam(r.URL.Query().Get("slugs"))
	if len(slugs) == 0 {
		jsonResponse(w, http.StatusOK, emptyOK)
		return
	}

	// Resolve start_comid, end_comid, and reach length from DB.
	// Length drives the NLDI search distance: 40km base × 1.75 per 20km of reach.
	rows, err := h.db.Query(r.Context(),
		`SELECT start_comid, end_comid,
		        COALESCE(ST_Length(centerline::geography)/1000, 0) AS length_km
		 FROM reaches
		 WHERE slug = ANY($1)
		   AND (start_comid IS NOT NULL OR end_comid IS NOT NULL)`,
		slugs,
	)
	if err != nil {
		jsonResponse(w, http.StatusOK, emptyOK)
		return
	}
	defer rows.Close()

	seenStart := make(map[string]struct{})
	seenEnd := make(map[string]struct{})
	var startComIDs, endComIDs []string
	var maxLengthKm float64

	for rows.Next() {
		var startComID, endComID *string
		var lengthKm float64
		if err := rows.Scan(&startComID, &endComID, &lengthKm); err != nil {
			continue
		}
		if lengthKm > maxLengthKm {
			maxLengthKm = lengthKm
		}
		if startComID != nil {
			if _, ok := seenStart[*startComID]; !ok {
				seenStart[*startComID] = struct{}{}
				startComIDs = append(startComIDs, *startComID)
			}
		}
		if endComID != nil {
			if _, ok := seenEnd[*endComID]; !ok {
				seenEnd[*endComID] = struct{}{}
				endComIDs = append(endComIDs, *endComID)
			}
		}
	}
	_ = rows.Err()

	if len(startComIDs) == 0 && len(endComIDs) == 0 {
		jsonResponse(w, http.StatusOK, emptyOK)
		return
	}

	// Auto-compute NLDI search distance from longest reach.
	// Steps of 20 km each multiply the base by 1.75:
	//   0–19 km → 40 km,  20–39 km → 70 km,  40–59 km → 123 km,  60+ km → 200 km (cap)
	steps := int(maxLengthKm / 20.0)
	distanceKm := int(math.Min(200, math.Max(20, math.Round(40*math.Pow(1.75, float64(steps))))))
	// Allow explicit override for testing (?distance=N).
	if d := r.URL.Query().Get("distance"); d != "" {
		if v, err := strconv.Atoi(d); err == nil && v >= 1 && v <= 200 {
			distanceKm = v
		}
	}

	c := nldi.New()
	cacheKey := func(comid string) string { return fmt.Sprintf("%s|%d", comid, distanceKm) }

	var (
		allTributaries []nldi.Feature
		allMainstem    []nldi.Feature
		nldiErr        error
	)

	fetchUT := func(comid string, dest *[]nldi.Feature) {
		key := cacheKey(comid)
		if v, ok := networkCache.Load(key); ok {
			entry := v.(networkCacheEntry)
			if time.Since(entry.cachedAt) < networkCacheTTL {
				*dest = append(*dest, entry.upstream.Features...)
				return
			}
			networkCache.Delete(key)
		}
		ut, errUT := c.UpstreamFlowlines(r.Context(), comid, distanceKm)
		if errUT != nil {
			log.Printf("basin network: UT comid %s: %v", comid, errUT)
			if nldiErr == nil {
				nldiErr = errUT
			}
			return
		}
		networkCache.Store(key, networkCacheEntry{upstream: *ut, cachedAt: time.Now()})
		*dest = append(*dest, ut.Features...)
	}

	for _, comid := range startComIDs {
		fetchUT(comid, &allTributaries)
	}
	for _, comid := range endComIDs {
		fetchUT(comid, &allMainstem)
	}

	// Deduplicate and strip flowlines that overlap dashboard reaches.
	// Deduplicate but keep all flowlines — reach centerlines paint on top in the map.
	tributaries := dedupeFlowlines(allTributaries)
	mainstem := dedupeFlowlines(allMainstem)

	resp := basinNetworkResponse{
		Tributaries:   nldi.Collection{Type: "FeatureCollection", Features: tributaries},
		Mainstem:      nldi.Collection{Type: "FeatureCollection", Features: mainstem},
		NLDIAvailable: nldiErr == nil,
	}
	if nldiErr != nil {
		resp.NLDIError = "River network data is temporarily unavailable (NLDI). The map will display your dashboard reaches without the upstream/downstream overlay."
	}

	jsonResponse(w, http.StatusOK, resp)
}

// dedupeFlowlines removes duplicate NLDI features by nhdplus_comid.
func dedupeFlowlines(features []nldi.Feature) []nldi.Feature {
	seen := make(map[string]struct{}, len(features))
	out := make([]nldi.Feature, 0, len(features))
	for _, f := range features {
		id := ""
		if f.Props.NhdplusComID != nil {
			id = *f.Props.NhdplusComID
		}
		if id == "" {
			out = append(out, f)
			continue
		}
		if _, dup := seen[id]; dup {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, f)
	}
	return out
}

// parseSlugsParam splits a comma-separated slugs query param, trims whitespace,
// and drops empty strings.
func parseSlugsParam(raw string) []string {
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := parts[:0]
	for _, s := range parts {
		if t := strings.TrimSpace(s); t != "" {
			out = append(out, t)
		}
	}
	return out
}
