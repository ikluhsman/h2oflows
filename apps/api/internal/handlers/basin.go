package handlers

import (
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
)

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

	slugsParam := r.URL.Query().Get("slugs")
	if slugsParam == "" {
		jsonResponse(w, http.StatusOK, empty)
		return
	}

	raw := strings.Split(slugsParam, ",")
	slugs := raw[:0]
	for _, s := range raw {
		if t := strings.TrimSpace(s); t != "" {
			slugs = append(slugs, t)
		}
	}
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
			item                       basinReachItem
			startLng, startLat         *float64
			endLng, endLat             *float64
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
