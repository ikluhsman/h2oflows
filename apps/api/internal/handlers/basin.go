package handlers

import (
	"fmt"
	"log"
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
	tributaries nldi.Collection
	mainstem    nldi.Collection
	cachedAt    time.Time
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
// For each reach slug in ?slugs=, looks up anchor_comid and fetches upstream
// tributaries (UT) and downstream mainstem (DM) from NLDI within ?distance= km
// (default 50, clamped 1–200). Results are cached per (comid, distance) for 24 h.
//
// Flowlines whose nhdplus_comid matches a dashboard reach's anchor_comid are
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

	distanceKm := 50
	if d := r.URL.Query().Get("distance"); d != "" {
		if v, err := strconv.Atoi(d); err == nil {
			if v < 1 {
				v = 1
			} else if v > 200 {
				v = 200
			}
			distanceKm = v
		}
	}

	// Resolve anchor_comids from DB.
	rows, err := h.db.Query(r.Context(),
		`SELECT slug, anchor_comid FROM reaches WHERE slug = ANY($1) AND anchor_comid IS NOT NULL`,
		slugs,
	)
	if err != nil {
		jsonResponse(w, http.StatusOK, emptyOK)
		return
	}
	defer rows.Close()

	type slugComID struct{ slug, comid string }
	var pairs []slugComID
	dashboardComIDs := make(map[string]struct{}) // for stripping overlap
	for rows.Next() {
		var sc slugComID
		if err := rows.Scan(&sc.slug, &sc.comid); err != nil {
			continue
		}
		pairs = append(pairs, sc)
		dashboardComIDs[sc.comid] = struct{}{}
	}
	_ = rows.Err()

	if len(pairs) == 0 {
		jsonResponse(w, http.StatusOK, emptyOK)
		return
	}

	// Deduplicate comIDs (multiple slugs may share one anchor).
	seen := make(map[string]struct{})
	var comIDs []string
	for _, p := range pairs {
		if _, ok := seen[p.comid]; !ok {
			seen[p.comid] = struct{}{}
			comIDs = append(comIDs, p.comid)
		}
	}

	c := nldi.New()
	cacheKey := func(comid string) string { return fmt.Sprintf("%s|%d", comid, distanceKm) }

	var (
		allTributaries []nldi.Feature
		allMainstem    []nldi.Feature
		nldiErr        error
	)

	for _, comid := range comIDs {
		key := cacheKey(comid)

		// Check cache.
		if v, ok := networkCache.Load(key); ok {
			entry := v.(networkCacheEntry)
			if time.Since(entry.cachedAt) < networkCacheTTL {
				allTributaries = append(allTributaries, entry.tributaries.Features...)
				allMainstem = append(allMainstem, entry.mainstem.Features...)
				continue
			}
			networkCache.Delete(key)
		}

		// Fetch from NLDI. Errors are soft — log and continue.
		ut, errUT := c.UpstreamFlowlines(r.Context(), comid, distanceKm)
		dm, errDM := c.DownstreamFlowlines(r.Context(), comid, distanceKm)

		if errUT != nil || errDM != nil {
			log.Printf("basin network: comid %s: ut=%v dm=%v", comid, errUT, errDM)
			if nldiErr == nil {
				if errUT != nil {
					nldiErr = errUT
				} else {
					nldiErr = errDM
				}
			}
			continue
		}

		entry := networkCacheEntry{
			tributaries: *ut,
			mainstem:    *dm,
			cachedAt:    time.Now(),
		}
		networkCache.Store(key, entry)

		allTributaries = append(allTributaries, ut.Features...)
		allMainstem = append(allMainstem, dm.Features...)
	}

	// Deduplicate and strip flowlines that overlap dashboard reaches.
	tributaries := dedupeFlowlines(allTributaries, dashboardComIDs)
	mainstem := dedupeFlowlines(allMainstem, dashboardComIDs)

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

// dedupeFlowlines removes duplicate features by nhdplus_comid and strips any
// whose comid is in the exclude set (dashboard reach centerlines).
func dedupeFlowlines(features []nldi.Feature, exclude map[string]struct{}) []nldi.Feature {
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
		if _, excluded := exclude[id]; excluded {
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
