package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/h2oflow/h2oflow/apps/api/internal/ai"
	"github.com/h2oflow/h2oflow/apps/api/internal/osm"
	"github.com/jackc/pgx/v5/pgxpool"
)

// reachMapCache holds a pre-warmed snapshot of all reach features for the
// /reaches/map/all endpoint. This lets the frontend load the full dataset
// in one request at startup and filter client-side on every viewport change,
// eliminating per-pan/zoom round-trips to the database.
type reachMapCache struct {
	mu       sync.RWMutex
	payload  []byte    // marshalled GeoJSON FeatureCollection
	warmedAt time.Time
}

func (c *reachMapCache) set(payload []byte) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.payload = payload
	c.warmedAt = time.Now()
}

func (c *reachMapCache) get() ([]byte, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if len(c.payload) == 0 {
		return nil, false
	}
	return c.payload, true
}

// gaugeFetcher is the narrow poller interface ReachHandler needs for
// on-demand fetching of stale primary gauges. Keeps this package free of a
// hard dependency on the poller implementation.
type gaugeFetcher interface {
	FetchNowIfStale(ctx context.Context, gaugeID string, maxAge time.Duration) bool
	TouchRequested(ctx context.Context, gaugeID string)
}

// ReachHandler handles reach-related HTTP routes.
type ReachHandler struct {
	db     *pgxpool.Pool
	asker  *ai.ReachAsker
	cache  *reachMapCache
	poller gaugeFetcher // nil = on-demand fetching disabled
}

func NewReachHandler(db *pgxpool.Pool, asker *ai.ReachAsker) *ReachHandler {
	return &ReachHandler{db: db, asker: asker, cache: &reachMapCache{}}
}

// WithPoller wires a poller for on-demand gauge fetching. Optional — without
// it, reach detail pages still work but show whatever the last poll tick
// captured.
func (h *ReachHandler) WithPoller(p gaugeFetcher) *ReachHandler {
	h.poller = p
	return h
}

// WarmCache fetches all reach features (no bbox filter) and stores the result
// in the in-memory cache. Call once at server startup, then every poll cycle.
func (h *ReachHandler) WarmCache(ctx context.Context) {
	features, err := h.queryAllFeatures(ctx)
	if err != nil {
		log.Printf("reach cache: warm failed: %v", err)
		return
	}
	payload, err := json.Marshal(newFeatureCollection(features))
	if err != nil {
		log.Printf("reach cache: marshal failed: %v", err)
		return
	}
	h.cache.set(payload)
	log.Printf("reach cache: warmed %d features", len(features))
}

// StartCacheRefresh launches a background goroutine that re-warms the cache
// on the given interval (typically the same as the gauge poll interval).
func (h *ReachHandler) StartCacheRefresh(ctx context.Context, interval time.Duration) {
	go func() {
		t := time.NewTicker(interval)
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				h.WarmCache(ctx)
			}
		}
	}()
}

// Map handles GET /api/v1/reaches/map
//
// Returns a GeoJSON FeatureCollection of reach centerlines with their current
// flow status. MapLibre uses this to color river lines on the dashboard map.
//
// Query params:
//
//	bbox=west,south,east,north  (required — viewport bounds)
//
// Flow status values and their map colors:
//
//	runnable  → green  (flow is within fun/optimal range)
//	caution   → yellow (minimum or pushy — paddle at your level)
//	low       → red    (below minimum, not recommended)
//	flood     → red    (high water or flood stage, dangerous)
//	unknown   → grey   (no current reading or no flow ranges defined)
//
// Reaches with no centerline geometry are omitted — they appear as gauge
// markers via the gauge search endpoint instead.
func (h *ReachHandler) Map(w http.ResponseWriter, r *http.Request) {
	bbox, err := parseBBoxParam(r)
	if err != nil {
		errorResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	rows, err := h.db.Query(r.Context(), `
		WITH latest_reading AS (
			-- Most recent reading for each gauge within the last 48 hours.
			-- Outside this window we treat the status as unknown.
			SELECT DISTINCT ON (gauge_id)
				gauge_id, value
			FROM gauge_readings
			WHERE timestamp > NOW() - INTERVAL '48 hours'
			ORDER BY gauge_id, timestamp DESC
		)
		SELECT
			r.id,
			r.name,
			r.slug,
			r.river_name,
			r.common_name,
			r.put_in_name,
			r.take_out_name,
			r.class_min,
			COALESCE(
				(SELECT MAX(class_rating) FROM rapids WHERE reach_id = r.id AND class_rating IS NOT NULL),
				r.class_max
			) AS class_max,
			r.character,
			r.length_mi,
			ST_AsGeoJSON(r.centerline::geometry)::json        AS centerline,
			ST_X(r.put_in::geometry)                          AS put_in_lng,
			ST_Y(r.put_in::geometry)                          AS put_in_lat,
			ST_X(r.take_out::geometry)                        AS take_out_lng,
			ST_Y(r.take_out::geometry)                        AS take_out_lat,
			lr.value                                          AS current_cfs,
			g.last_reading_at,
			fr.label                                          AS flow_label,
			g.id                                              AS gauge_id,
			g.reach_relationship,
			g.featured                                        AS gauge_trusted,
			g.gauge_notes,
			g.info_links,
			CASE
				WHEN lr.value IS NULL OR fr.label IS NULL  THEN 'unknown'
				WHEN fr.label = 'runnable'                 THEN 'runnable'
				WHEN fr.label = 'below_recommended'        THEN 'caution'
				WHEN fr.label = 'above_recommended'        THEN 'flood'
				-- legacy labels (pre-migration 034) kept for any un-migrated rows
				WHEN fr.label IN ('fun', 'optimal')        THEN 'runnable'
				WHEN fr.label IN ('minimum', 'pushy')      THEN 'caution'
				WHEN fr.label = 'too_low'                  THEN 'low'
				WHEN fr.label IN ('high', 'flood')         THEN 'flood'
				ELSE                                            'unknown'
			END AS flow_status
		FROM reaches r
		LEFT JOIN gauges g
			ON g.id = r.primary_gauge_id
		LEFT JOIN latest_reading lr
			ON lr.gauge_id = g.id
		LEFT JOIN LATERAL (
			-- Match the reading to the first flow range it falls within.
			SELECT label FROM flow_ranges
			WHERE gauge_id = g.id
			  AND (min_cfs IS NULL OR lr.value >= min_cfs)
			  AND (max_cfs IS NULL OR lr.value <  max_cfs)
			LIMIT 1
		) fr ON TRUE
		WHERE r.centerline IS NOT NULL
		  AND ST_Intersects(
			r.centerline::geometry,
			ST_MakeEnvelope($1, $2, $3, $4, 4326)
		  )
	`, bbox.West, bbox.South, bbox.East, bbox.North)
	if err != nil {
		errorResponse(w, http.StatusInternalServerError, "query failed")
		return
	}
	defer rows.Close()

	features := make([]Feature, 0)
	for rows.Next() {
		var (
			id                string
			name              string
			slug              string
			riverName         *string
			commonName        *string
			putInName         *string
			takeOutName       *string
			classMin          *float64
			classMax          *float64
			character         *string
			lengthMi          *float64
			centerlineJSON    []byte
			putInLng          *float64
			putInLat          *float64
			takeOutLng        *float64
			takeOutLat        *float64
			currentCFS        *float64
			lastReadingAt     *time.Time
			flowLabel         *string
			gaugeID           *string
			reachRelationship *string
			gaugeTrusted      *bool
			gaugeNotes        *string
			infoLinks         []byte
			flowStatus        string
		)
		if err := rows.Scan(
			&id, &name, &slug, &riverName, &commonName, &putInName, &takeOutName,
			&classMin, &classMax, &character, &lengthMi,
			&centerlineJSON,
			&putInLng, &putInLat, &takeOutLng, &takeOutLat,
			&currentCFS, &lastReadingAt, &flowLabel, &gaugeID,
			&reachRelationship, &gaugeTrusted, &gaugeNotes, &infoLinks,
			&flowStatus,
		); err != nil {
			continue
		}

		// put_in and take_out as [lng, lat] pairs for frontend marker rendering on hover.
		var putIn, takeOut *[2]float64
		if putInLng != nil && putInLat != nil {
			putIn = &[2]float64{*putInLng, *putInLat}
		}
		if takeOutLng != nil && takeOutLat != nil {
			takeOut = &[2]float64{*takeOutLng, *takeOutLat}
		}

		features = append(features, Feature{
			Type:     "Feature",
			Geometry: rawGeometry(centerlineJSON),
			Properties: map[string]any{
				"id":                 id,
				"name":               name,
				"slug":               slug,
				"river_name":         riverName,
				"common_name":        commonName,
				"put_in_name":        putInName,
				"take_out_name":      takeOutName,
				"class_min":          classMin,
				"class_max":          classMax,
				"character":          character,
				"length_mi":          lengthMi,
				"put_in":             putIn,
				"take_out":           takeOut,
				"current_cfs":        currentCFS,
				"last_reading_at":    lastReadingAt,
				"flow_label":         flowLabel,
				"gauge_id":           gaugeID,
				"flow_status":        flowStatus,
				"flow_color":         flowColor(flowStatus),
				"reach_relationship": reachRelationship,
				"gauge_trusted":      gaugeTrusted,
				"gauge_notes":        gaugeNotes,
				"info_links":         rawJSON(infoLinks),
			},
		})
	}
	if err := rows.Err(); err != nil {
		errorResponse(w, http.StatusInternalServerError, "scan failed")
		return
	}

	jsonResponse(w, http.StatusOK, newFeatureCollection(features))
}

// MapAll handles GET /api/v1/reaches/map/all
//
// Returns the full reach GeoJSON dataset (no bbox filter). Served from an
// in-memory cache warmed at startup and refreshed every poll cycle. The
// frontend loads this once and filters client-side on every viewport change,
// eliminating per-pan/zoom round-trips.
func (h *ReachHandler) MapAll(w http.ResponseWriter, r *http.Request) {
	if payload, ok := h.cache.get(); ok {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "public, max-age=60")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(payload)
		return
	}
	// Cache cold (first request before WarmCache finishes) — query directly.
	features, err := h.queryAllFeatures(r.Context())
	if err != nil {
		errorResponse(w, http.StatusInternalServerError, "query failed")
		return
	}
	jsonResponse(w, http.StatusOK, newFeatureCollection(features))
}

// queryAllFeatures runs the reach-map query without a bbox filter and returns
// the raw Feature slice. Shared by MapAll and WarmCache.
func (h *ReachHandler) queryAllFeatures(ctx context.Context) ([]Feature, error) {
	rows, err := h.db.Query(ctx, `
		WITH latest_reading AS (
			SELECT DISTINCT ON (gauge_id)
				gauge_id, value
			FROM gauge_readings
			WHERE timestamp > NOW() - INTERVAL '48 hours'
			ORDER BY gauge_id, timestamp DESC
		)
		SELECT
			r.id, r.name, r.slug,
			r.river_name, r.common_name, r.put_in_name, r.take_out_name,
			r.class_min,
			COALESCE(
				(SELECT MAX(class_rating) FROM rapids WHERE reach_id = r.id AND class_rating IS NOT NULL),
				r.class_max
			) AS class_max,
			r.character, r.length_mi,
			ST_AsGeoJSON(r.centerline::geometry)::json AS centerline,
			ST_X(r.put_in::geometry)   AS put_in_lng,
			ST_Y(r.put_in::geometry)   AS put_in_lat,
			ST_X(r.take_out::geometry) AS take_out_lng,
			ST_Y(r.take_out::geometry) AS take_out_lat,
			lr.value                   AS current_cfs,
			g.last_reading_at,
			fr.label                   AS flow_label,
			g.id                       AS gauge_id,
			g.reach_relationship,
			g.featured                 AS gauge_trusted,
			g.gauge_notes,
			g.info_links,
			CASE
				WHEN lr.value IS NULL OR fr.label IS NULL  THEN 'unknown'
				WHEN fr.label = 'runnable'                 THEN 'runnable'
				WHEN fr.label = 'below_recommended'        THEN 'caution'
				WHEN fr.label = 'above_recommended'        THEN 'flood'
				WHEN fr.label IN ('fun', 'optimal')        THEN 'runnable'
				WHEN fr.label IN ('minimum', 'pushy')      THEN 'caution'
				WHEN fr.label = 'too_low'                  THEN 'low'
				WHEN fr.label IN ('high', 'flood')         THEN 'flood'
				ELSE                                            'unknown'
			END AS flow_status
		FROM reaches r
		LEFT JOIN gauges g ON g.id = r.primary_gauge_id
		LEFT JOIN latest_reading lr ON lr.gauge_id = g.id
		LEFT JOIN LATERAL (
			SELECT label FROM flow_ranges
			WHERE gauge_id = g.id
			  AND (min_cfs IS NULL OR lr.value >= min_cfs)
			  AND (max_cfs IS NULL OR lr.value <  max_cfs)
			LIMIT 1
		) fr ON TRUE
		WHERE r.centerline IS NOT NULL
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	features := make([]Feature, 0)
	for rows.Next() {
		var (
			id, name, slug                    string
			riverName, commonName             *string
			putInName, takeOutName            *string
			classMin, classMax                *float64
			character                         *string
			lengthMi                          *float64
			centerlineJSON                    []byte
			putInLng, putInLat                *float64
			takeOutLng, takeOutLat            *float64
			currentCFS                        *float64
			lastReadingAt                     *time.Time
			flowLabel, gaugeID                *string
			reachRelationship                 *string
			gaugeTrusted                      *bool
			gaugeNotes                        *string
			infoLinks                         []byte
			flowStatus                        string
		)
		if err := rows.Scan(
			&id, &name, &slug,
			&riverName, &commonName, &putInName, &takeOutName,
			&classMin, &classMax, &character, &lengthMi,
			&centerlineJSON,
			&putInLng, &putInLat, &takeOutLng, &takeOutLat,
			&currentCFS, &lastReadingAt, &flowLabel, &gaugeID,
			&reachRelationship, &gaugeTrusted, &gaugeNotes, &infoLinks,
			&flowStatus,
		); err != nil {
			continue
		}
		var putIn, takeOut *[2]float64
		if putInLng != nil && putInLat != nil {
			putIn = &[2]float64{*putInLng, *putInLat}
		}
		if takeOutLng != nil && takeOutLat != nil {
			takeOut = &[2]float64{*takeOutLng, *takeOutLat}
		}
		features = append(features, Feature{
			Type:     "Feature",
			Geometry: rawGeometry(centerlineJSON),
			Properties: map[string]any{
				"id": id, "name": name, "slug": slug,
				"river_name": riverName, "common_name": commonName,
				"put_in_name": putInName, "take_out_name": takeOutName,
				"class_min": classMin, "class_max": classMax,
				"character": character, "length_mi": lengthMi,
				"put_in": putIn, "take_out": takeOut,
				"current_cfs": currentCFS, "last_reading_at": lastReadingAt,
				"flow_label": flowLabel, "gauge_id": gaugeID,
				"flow_status": flowStatus, "flow_color": flowColor(flowStatus),
				"reach_relationship": reachRelationship,
				"gauge_trusted": gaugeTrusted, "gauge_notes": gaugeNotes,
				"info_links": rawJSON(infoLinks),
			},
		})
	}
	return features, rows.Err()
}

// List handles GET /api/v1/reaches
// TODO: implement
func (h *ReachHandler) List(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotImplemented)
}

// Get handles GET /api/v1/reaches/{slug}
//
// Returns full reach detail: description, rapids inventory, access points,
// and current gauge conditions. Used by the SSR reach detail page.
func (h *ReachHandler) Get(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	if slug == "" {
		errorResponse(w, http.StatusBadRequest, "slug is required")
		return
	}

	// ---- On-demand gauge refresh --------------------------------------------
	// If the primary gauge's most recent reading is older than 1 hour,
	// fetch synchronously before the main query so the user sees current data
	// on first view rather than waiting for the next poll tick.
	if h.poller != nil {
		var primaryGaugeID *string
		_ = h.db.QueryRow(r.Context(),
			`SELECT primary_gauge_id::text FROM reaches WHERE slug = $1`, slug,
		).Scan(&primaryGaugeID)
		if primaryGaugeID != nil && *primaryGaugeID != "" {
			h.poller.FetchNowIfStale(r.Context(), *primaryGaugeID, time.Hour)
			go h.poller.TouchRequested(context.Background(), *primaryGaugeID)
		}
	}

	// ---- Reach + gauge info -------------------------------------------------
	var reach reachDetail
	err := h.db.QueryRow(r.Context(), `
		SELECT
			r.id,
			r.slug,
			r.name,
			r.region,
			r.class_min,
			COALESCE(
				(SELECT MAX(class_rating) FROM rapids WHERE reach_id = r.id AND class_rating IS NOT NULL AND is_permanent_hazard = FALSE),
				r.class_max
			) AS class_max,
			r.class_hardest,
			r.character,
			r.length_mi,
			r.description,
			r.description_source,
			r.description_ai_confidence,
			r.description_verified,
			r.aw_reach_id,
			r.watershed_name,
			r.river_name,
			r.common_name,
			r.put_in_name,
			r.take_out_name,
			ST_AsGeoJSON(r.centerline::geometry) AS centerline,
			-- Primary gauge fields (all nullable — reach may not have a gauge yet)
			g.id                AS gauge_id,
			g.external_id       AS gauge_external_id,
			g.source            AS gauge_source,
			g.name              AS gauge_name,
			g.featured          AS gauge_featured,
			lr.value            AS current_cfs,
			lr.timestamp        AS last_reading_at,
			COALESCE(ST_X(g.location::geometry), NULL) AS gauge_lng,
			COALESCE(ST_Y(g.location::geometry), NULL) AS gauge_lat,
			CASE
				WHEN lr.value IS NULL OR fr.label IS NULL  THEN 'unknown'
				WHEN fr.label = 'runnable'                 THEN 'runnable'
				WHEN fr.label = 'below_recommended'        THEN 'caution'
				WHEN fr.label = 'above_recommended'        THEN 'flood'
				-- legacy fallbacks (pre-migration 034)
				WHEN fr.label IN ('fun','optimal')         THEN 'runnable'
				WHEN fr.label IN ('minimum','pushy')       THEN 'caution'
				WHEN fr.label = 'too_low'                  THEN 'low'
				WHEN fr.label IN ('high','flood')          THEN 'flood'
				ELSE 'unknown'
			END AS flow_status
		FROM reaches r
		LEFT JOIN gauges g ON g.id = r.primary_gauge_id
		LEFT JOIN LATERAL (
			SELECT value, timestamp FROM gauge_readings
			WHERE gauge_id = g.id
			  AND timestamp > NOW() - INTERVAL '48 hours'
			ORDER BY timestamp DESC LIMIT 1
		) lr ON TRUE
		LEFT JOIN LATERAL (
			SELECT label FROM flow_ranges
			WHERE gauge_id = g.id
			  AND craft_type = 'general'
			  AND (min_cfs IS NULL OR lr.value >= min_cfs)
			  AND (max_cfs IS NULL OR lr.value <  max_cfs)
			ORDER BY min_cfs ASC NULLS FIRST
			LIMIT 1
		) fr ON TRUE
		WHERE r.slug = $1
	`, slug).Scan(
		&reach.ID, &reach.Slug, &reach.Name, &reach.Region,
		&reach.ClassMin, &reach.ClassMax, &reach.ClassHardest, &reach.Character, &reach.LengthMi,
		&reach.Description, &reach.DescriptionSource,
		&reach.DescriptionConfidence, &reach.DescriptionVerified,
		&reach.AWReachID, &reach.WatershedName,
		&reach.RiverName, &reach.CommonName, &reach.PutInName, &reach.TakeOutName,
		&reach.Centerline,
		&reach.Gauge.ID, &reach.Gauge.ExternalID, &reach.Gauge.Source,
		&reach.Gauge.Name, &reach.Gauge.Featured,
		&reach.Gauge.CurrentCFS, &reach.Gauge.LastReadingAt,
		&reach.Gauge.Lng, &reach.Gauge.Lat,
		&reach.Gauge.FlowStatus,
	)
	if err != nil {
		errorResponse(w, http.StatusNotFound, "reach not found")
		return
	}

	// Ensure arrays serialize as [] not null when empty
	reach.Rapids   = make([]rapidRow, 0)
	reach.Access   = make([]accessRow, 0)
	reach.Gauges   = make([]gaugeSnippet, 0)

	// Primary gauge goes in Gauges[0] if it exists
	if reach.Gauge.ID != nil {
		reach.Gauges = append(reach.Gauges, reach.Gauge)
	}

	// Secondary gauges — all gauges linked to this reach (excluding primary)
	secRows, err := h.db.Query(r.Context(), `
		SELECT
			g.id, g.external_id, g.source, g.name, g.featured,
			g.reach_relationship,
			g.current_cfs, g.flow_status, g.last_reading_at,
			ST_X(g.location::geometry) AS lng,
			ST_Y(g.location::geometry) AS lat
		FROM gauges g
		WHERE g.reach_id = $1
		  AND ($2::uuid IS NULL OR g.id != $2::uuid)
		  AND g.status = 'active'
		ORDER BY CASE g.reach_relationship
			WHEN 'primary'              THEN 1
			WHEN 'upstream_indicator'   THEN 2
			WHEN 'downstream_indicator' THEN 3
			ELSE 4
		END, g.name
	`, reach.ID, reach.Gauge.ID)
	if err == nil {
		defer secRows.Close()
		for secRows.Next() {
			var sg gaugeSnippet
			if err := secRows.Scan(
				&sg.ID, &sg.ExternalID, &sg.Source, &sg.Name, &sg.Featured,
				&sg.Relationship,
				&sg.CurrentCFS, &sg.FlowStatus, &sg.LastReadingAt,
				&sg.Lng, &sg.Lat,
			); err != nil {
				continue
			}
			reach.Gauges = append(reach.Gauges, sg)
		}
	}

	// ---- Rapids -------------------------------------------------------------
	// river_order: 0→1 position along the stored centerline (put-in=0, take-out=1).
	// Falls back to NULL when no centerline — frontend then falls back to lng sort.
	rapidRows, err := h.db.Query(r.Context(), `
		WITH rap AS (
			SELECT
				id, name, river_mile, class_rating, class_at_low, class_at_high,
				description, portage_description, is_portage_recommended, is_surf_wave,
				is_permanent_hazard, hazard_type,
				data_source, ai_confidence, verified,
				ST_X(location::geometry) AS lng,
				ST_Y(location::geometry) AS lat,
				CASE WHEN $2::text IS NOT NULL AND location IS NOT NULL
				     THEN ST_LineLocatePoint(ST_GeomFromGeoJSON($2), location::geometry)
				     ELSE NULL
				END AS river_order
			FROM rapids
			WHERE reach_id = $1
		)
		SELECT * FROM rap
		ORDER BY river_order ASC NULLS LAST, river_mile ASC NULLS LAST, name ASC
	`, reach.ID, reach.Centerline)
	if err != nil {
		errorResponse(w, http.StatusInternalServerError, "rapids query failed")
		return
	}
	defer rapidRows.Close()
	for rapidRows.Next() {
		var rr rapidRow
		if err := rapidRows.Scan(
			&rr.ID, &rr.Name, &rr.RiverMile,
			&rr.ClassRating, &rr.ClassAtLow, &rr.ClassAtHigh,
			&rr.Description, &rr.PortageDescription, &rr.IsPortageRecommended, &rr.IsSurfWave,
			&rr.IsPermanentHazard, &rr.HazardType,
			&rr.DataSource, &rr.AIConfidence, &rr.Verified,
			&rr.Lng, &rr.Lat, &rr.RiverOrder,
		); err != nil {
			continue
		}
		reach.Rapids = append(reach.Rapids, rr)
	}

	// ---- Access points + waypoints ------------------------------------------
	accessRows, err := h.db.Query(r.Context(), `
		SELECT
			id, access_type, name, directions, road_type,
			entry_style, approach_dist_mi, approach_notes,
			parking_fee, permit_required, permit_info, permit_url,
			seasonal_close_start, seasonal_close_end, notes,
			ST_X(location::geometry)         AS water_lng,
			ST_Y(location::geometry)         AS water_lat,
			ST_X(parking_location::geometry) AS parking_lng,
			ST_Y(parking_location::geometry) AS parking_lat,
			hike_to_water_min,
			data_source, ai_confidence, verified,
			CASE WHEN $2::text IS NOT NULL AND location IS NOT NULL
			     THEN ST_LineLocatePoint(ST_GeomFromGeoJSON($2), location::geometry)
			     ELSE NULL
			END AS river_order
		FROM reach_access
		WHERE reach_id = $1
		ORDER BY
			CASE access_type
				WHEN 'put_in'       THEN 1
				WHEN 'take_out'     THEN 2
				WHEN 'intermediate' THEN 3
				WHEN 'shuttle_drop' THEN 4
				WHEN 'parking'      THEN 5
				WHEN 'camp'         THEN 6
			END
	`, reach.ID, reach.Centerline)
	if err != nil {
		errorResponse(w, http.StatusInternalServerError, "access query failed")
		return
	}
	defer accessRows.Close()

	accessByID := map[string]*accessRow{}
	for accessRows.Next() {
		var a accessRow
		if err := accessRows.Scan(
			&a.ID, &a.AccessType, &a.Name, &a.Directions, &a.RoadType,
			&a.EntryStyle, &a.ApproachDistMi, &a.ApproachNotes,
			&a.ParkingFee, &a.PermitRequired, &a.PermitInfo, &a.PermitURL,
			&a.SeasonalCloseStart, &a.SeasonalCloseEnd, &a.Notes,
			&a.WaterLng, &a.WaterLat, &a.ParkingLng, &a.ParkingLat,
			&a.HikeToWaterMin,
			&a.DataSource, &a.AIConfidence, &a.Verified, &a.RiverOrder,
		); err != nil {
			continue
		}
		a.Waypoints = make([]waypointRow, 0)
		reach.Access = append(reach.Access, a)
		accessByID[a.ID] = &reach.Access[len(reach.Access)-1]
	}

	// Waypoints for each access point
	if len(accessByID) > 0 {
		ids := make([]string, 0, len(accessByID))
		for id := range accessByID {
			ids = append(ids, id)
		}
		wpRows, err := h.db.Query(r.Context(), `
			SELECT access_id, sequence, label, description,
			       ST_X(location::geometry), ST_Y(location::geometry),
			       verified
			FROM access_waypoints
			WHERE access_id = ANY($1)
			ORDER BY access_id, sequence
		`, ids)
		if err == nil {
			defer wpRows.Close()
			for wpRows.Next() {
				var wp waypointRow
				var accessID string
				if err := wpRows.Scan(
					&accessID, &wp.Sequence, &wp.Label, &wp.Description,
					&wp.Lng, &wp.Lat, &wp.Verified,
				); err != nil {
					continue
				}
				if a, ok := accessByID[accessID]; ok {
					a.Waypoints = append(a.Waypoints, wp)
				}
			}
		}
	}

	// ---- Related reaches --------------------------------------------------------
	reach.Related = make([]relatedReach, 0)
	relRows, err := h.db.Query(r.Context(), `
		SELECT t.slug, t.name, rr.relationship
		FROM reach_relationships rr
		JOIN reaches t ON t.id = rr.to_reach_id
		WHERE rr.from_reach_id = $1
		ORDER BY
			CASE rr.relationship
				WHEN 'upstream'     THEN 1
				WHEN 'downstream'   THEN 2
				WHEN 'tributary'    THEN 3
				WHEN 'continuation' THEN 4
			END, t.name
	`, reach.ID)
	if err == nil {
		defer relRows.Close()
		for relRows.Next() {
			var rel relatedReach
			if err := relRows.Scan(&rel.Slug, &rel.Name, &rel.Relationship); err == nil {
				reach.Related = append(reach.Related, rel)
			}
		}
	}

	jsonResponse(w, http.StatusOK, reach)
}

// ---- Response types ---------------------------------------------------------

type reachDetail struct {
	ID                      string          `json:"id"`
	Slug                    string          `json:"slug"`
	Name                    string          `json:"name"`
	RiverName               *string         `json:"river_name"`
	CommonName              *string         `json:"common_name"`
	PutInName               *string         `json:"put_in_name"`
	TakeOutName             *string         `json:"take_out_name"`
	Region                  *string         `json:"region"`
	ClassMin                *float64        `json:"class_min"`
	ClassMax                *float64        `json:"class_max"`
	ClassHardest            *float64        `json:"class_hardest"`
	Character               *string         `json:"character"`
	LengthMi                *float64        `json:"length_mi"`
	Description             *string         `json:"description"`
	DescriptionSource       *string         `json:"description_source"`
	DescriptionConfidence   *int            `json:"description_ai_confidence"`
	DescriptionVerified     bool            `json:"description_verified"`
	AWReachID               *string         `json:"aw_reach_id"`
	WatershedName           *string         `json:"watershed_name"`
	Centerline              rawGeometry     `json:"centerline"`
	Gauge                   gaugeSnippet    `json:"gauge"`
	Gauges                  []gaugeSnippet  `json:"gauges"`
	Rapids                  []rapidRow      `json:"rapids"`
	Access                  []accessRow     `json:"access"`
	Related                 []relatedReach  `json:"related"`
}

type gaugeSnippet struct {
	ID            *string    `json:"id"`
	ExternalID    *string    `json:"external_id"`
	Source        *string    `json:"source"`
	Name          *string    `json:"name"`
	Featured      *bool      `json:"featured"`
	Relationship  *string    `json:"reach_relationship"`
	CurrentCFS    *float64   `json:"current_cfs"`
	LastReadingAt *time.Time `json:"last_reading_at"`
	FlowStatus    string     `json:"flow_status"`
	Lng           *float64   `json:"lng"`
	Lat           *float64   `json:"lat"`
}

type rapidRow struct {
	ID                   string   `json:"id"`
	Name                 string   `json:"name"`
	RiverMile            *float64 `json:"river_mile"`
	ClassRating          *float64 `json:"class_rating"`
	ClassAtLow           *float64 `json:"class_at_low"`
	ClassAtHigh          *float64 `json:"class_at_high"`
	Description          *string  `json:"description"`
	PortageDescription   *string  `json:"portage_description"`
	IsPortageRecommended bool     `json:"is_portage_recommended"`
	IsSurfWave           bool     `json:"is_surf_wave"`
	IsPermanentHazard    bool     `json:"is_permanent_hazard"`
	HazardType           *string  `json:"hazard_type"`
	DataSource           string   `json:"data_source"`
	AIConfidence         *int     `json:"ai_confidence"`
	Verified             bool     `json:"verified"`
	Lng                  *float64 `json:"lng"`
	Lat                  *float64 `json:"lat"`
	RiverOrder           *float64 `json:"river_order"`
}

type accessRow struct {
	ID                 string        `json:"id"`
	AccessType         string        `json:"access_type"`
	Name               *string       `json:"name"`
	Directions         *string       `json:"directions"`
	RoadType           *string       `json:"road_type"`
	EntryStyle         *string       `json:"entry_style"`
	ApproachDistMi     *float64      `json:"approach_dist_mi"`
	ApproachNotes      *string       `json:"approach_notes"`
	ParkingFee         *float64      `json:"parking_fee"`
	PermitRequired     bool          `json:"permit_required"`
	PermitInfo         *string       `json:"permit_info"`
	PermitURL          *string       `json:"permit_url"`
	SeasonalCloseStart *string       `json:"seasonal_close_start"`
	SeasonalCloseEnd   *string       `json:"seasonal_close_end"`
	Notes              *string       `json:"notes"`
	WaterLng           *float64      `json:"water_lng"`
	WaterLat           *float64      `json:"water_lat"`
	ParkingLng         *float64      `json:"parking_lng"`
	ParkingLat         *float64      `json:"parking_lat"`
	HikeToWaterMin     *int          `json:"hike_to_water_min"`
	DataSource         string        `json:"data_source"`
	AIConfidence       *int          `json:"ai_confidence"`
	Verified           bool          `json:"verified"`
	Waypoints          []waypointRow `json:"waypoints"`
	RiverOrder         *float64      `json:"river_order"`
}

type relatedReach struct {
	Slug         string `json:"slug"`
	Name         string `json:"name"`
	Relationship string `json:"relationship"` // upstream | downstream | tributary | continuation
}

type waypointRow struct {
	Sequence    int      `json:"sequence"`
	Label       string   `json:"label"`
	Description *string  `json:"description"`
	Lng         *float64 `json:"lng"`
	Lat         *float64 `json:"lat"`
	Verified    bool     `json:"verified"`
}

// GetConditions handles GET /api/v1/reaches/{slug}/conditions
// TODO: implement
func (h *ReachHandler) GetConditions(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotImplemented)
}

// GetHazards handles GET /api/v1/reaches/{slug}/hazards
// TODO: implement
func (h *ReachHandler) GetHazards(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotImplemented)
}

// FetchCenterline handles POST /api/v1/reaches/{slug}/fetch-centerline
//
// Queries the Overpass API for the longest river/stream waterway within a
// ±0.05° bounding box around a centre point, stores it as the reach's
// centerline, and returns the GeoJSON geometry.
//
// Centre point resolution order:
//  1. lat/lng query params (explicit override)
//  2. reach put_in / take_out midpoint
//  3. access point coordinates (put_in / take_out types from reach_access)
//  4. primary gauge location
func (h *ReachHandler) FetchCenterline(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")

	// Parse optional explicit lat/lng override from query string.
	var explicitLat, explicitLng *float64
	if latStr := r.URL.Query().Get("lat"); latStr != "" {
		if v, err := strconv.ParseFloat(latStr, 64); err == nil {
			explicitLat = &v
		}
	}
	if lngStr := r.URL.Query().Get("lng"); lngStr != "" {
		if v, err := strconv.ParseFloat(lngStr, 64); err == nil {
			explicitLng = &v
		}
	}

	// Look up reach ID plus any stored geometry we can use as centre.
	var (
		reachID    string
		putInLng   *float64
		putInLat   *float64
		takeOutLng *float64
		takeOutLat *float64
		gaugeLng   *float64
		gaugeLat   *float64
	)
	err := h.db.QueryRow(r.Context(), `
		SELECT r.id,
		       ST_X(r.put_in::geometry)      AS put_in_lng,
		       ST_Y(r.put_in::geometry)      AS put_in_lat,
		       ST_X(r.take_out::geometry)    AS take_out_lng,
		       ST_Y(r.take_out::geometry)    AS take_out_lat,
		       ST_X(g.location::geometry)    AS gauge_lng,
		       ST_Y(g.location::geometry)    AS gauge_lat
		FROM reaches r
		LEFT JOIN gauges g ON g.id = r.primary_gauge_id
		WHERE r.slug = $1
	`, slug).Scan(
		&reachID,
		&putInLng, &putInLat,
		&takeOutLng, &takeOutLat,
		&gaugeLng, &gaugeLat,
	)
	if err != nil {
		errorResponse(w, http.StatusNotFound, "reach not found")
		return
	}

	// Query access points: full bbox for Overpass, plus the put-in/take-out
	// pair separated by the largest geographic distance to use as chain anchors.
	// Picking by maximum distance correctly handles rivers of any orientation
	// (E-W, N-S, or anywhere in between) — earlier versions used MIN/MAX(lng)
	// which broke on N-S flowing rivers like the South Platte at Deckers,
	// where a take-out further north can still sit slightly west of another.
	var (
		accessMinLng, accessMinLat *float64
		accessMaxLng, accessMaxLat *float64
		putInCentreLng, putInCentreLat *float64
		takeOutCentreLng, takeOutCentreLat *float64
	)
	_ = h.db.QueryRow(r.Context(), `
		WITH pts AS (
			-- Use water entry point; fall back to parking location when water point is unknown.
			-- Parking is close enough to snap the river segment correctly.
			SELECT access_type,
			       COALESCE(
			           ST_X(location::geometry),
			           ST_X(parking_location::geometry)
			       ) AS lng,
			       COALESCE(
			           ST_Y(location::geometry),
			           ST_Y(parking_location::geometry)
			       ) AS lat
			FROM reach_access
			WHERE reach_id = $1
			  AND (location IS NOT NULL OR parking_location IS NOT NULL)
		),
		extremes AS (
			-- Pair of (put_in, take_out) with the maximum geographic distance.
			-- This identifies the longest reach span regardless of river direction.
			SELECT p.lng AS put_in_lng,    p.lat AS put_in_lat,
			       t.lng AS take_out_lng,  t.lat AS take_out_lat
			FROM pts p, pts t
			WHERE p.access_type = 'put_in' AND t.access_type = 'take_out'
			ORDER BY ST_Distance(
			    ST_SetSRID(ST_MakePoint(p.lng, p.lat), 4326)::geography,
			    ST_SetSRID(ST_MakePoint(t.lng, t.lat), 4326)::geography
			) DESC
			LIMIT 1
		)
		SELECT
			MIN(p.lng), MIN(p.lat), MAX(p.lng), MAX(p.lat),
			e.put_in_lng,   e.put_in_lat,
			e.take_out_lng, e.take_out_lat
		FROM pts p, extremes e
		GROUP BY e.put_in_lng, e.put_in_lat, e.take_out_lng, e.take_out_lat
	`, reachID).Scan(
		&accessMinLng, &accessMinLat,
		&accessMaxLng, &accessMaxLat,
		&putInCentreLng, &putInCentreLat,
		&takeOutCentreLng, &takeOutCentreLat,
	)

	// Prefer access point geometry over the reaches.put_in / reaches.take_out
	// columns (which are often NULL for seeded reaches).
	if putInLng == nil && putInCentreLng != nil {
		putInLng, putInLat = putInCentreLng, putInCentreLat
	}
	if takeOutLng == nil && takeOutCentreLng != nil {
		takeOutLng, takeOutLat = takeOutCentreLng, takeOutCentreLat
	}

	// Resolve a single centre point for single-point fallback mode.
	var centreLng, centreLat float64
	switch {
	case explicitLng != nil && explicitLat != nil:
		centreLng, centreLat = *explicitLng, *explicitLat
	case accessMinLng != nil:
		centreLng = (*accessMinLng + *accessMaxLng) / 2
		centreLat = (*accessMinLat + *accessMaxLat) / 2
	case putInLng != nil:
		centreLng, centreLat = *putInLng, *putInLat
	case gaugeLng != nil:
		centreLng, centreLat = *gaugeLng, *gaugeLat
	default:
		errorResponse(w, http.StatusBadRequest,
			"no location available — pass ?lat=&lng= with the reach's approximate centre")
		return
	}

	// When we have a full spatial extent from access points, fetch the river
	// line using the tight bbox around all access points + put-in centroid as
	// the upstream chain start.  Fall back to a ±0.05° single-centre bbox.
	var lineJSON string
	if accessMinLng != nil && putInLng != nil && takeOutLng != nil && explicitLng == nil {
		lineJSON, err = osm.FetchReachLine(
			r.Context(),
			*accessMinLng, *accessMinLat,
			*accessMaxLng, *accessMaxLat,
			*putInLng, *putInLat,
			*takeOutLng, *takeOutLat,
		)
	} else {
		const pad = 0.05
		lineJSON, err = osm.FetchRiverLine(
			r.Context(),
			centreLng-pad, centreLat-pad,
			centreLng+pad, centreLat+pad,
		)
	}
	if err != nil {
		log.Printf("osm fetch for %s: %v", slug, err)
		errorResponse(w, http.StatusBadGateway, "OSM fetch failed: "+err.Error())
		return
	}
	if lineJSON == "" {
		errorResponse(w, http.StatusNotFound, "no waterway found near gauge location")
		return
	}

	// Store as PostGIS geography and derive length_mi from the line if not set.
	// ST_Length on a geography returns metres; divide by 1609.344 for miles.
	_, err = h.db.Exec(r.Context(), `
		UPDATE reaches
		SET    centerline = ST_GeomFromGeoJSON($1)::geography,
		       length_mi  = COALESCE(
		           length_mi,
		           ROUND((ST_Length(ST_GeomFromGeoJSON($1)::geography) / 1609.344)::numeric, 2)
		       )
		WHERE  id = $2
	`, lineJSON, reachID)
	if err != nil {
		log.Printf("centerline update for %s: %v", slug, err)
		errorResponse(w, http.StatusInternalServerError, "failed to save centerline")
		return
	}

	// Return the computed length alongside the geometry so the frontend can
	// display it immediately without a separate reach reload.
	var lengthMi *float64
	_ = h.db.QueryRow(r.Context(), `SELECT length_mi FROM reaches WHERE id = $1`, reachID).Scan(&lengthMi)

	jsonResponse(w, http.StatusOK, map[string]any{
		"centerline": rawGeometry([]byte(lineJSON)),
		"length_mi":  lengthMi,
	})
}

// ClearCenterline handles DELETE /api/v1/reaches/{slug}/centerline
// Nulls out the stored centerline so it can be re-fetched from OSM.
func (h *ReachHandler) ClearCenterline(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	_, err := h.db.Exec(r.Context(), `UPDATE reaches SET centerline = NULL WHERE slug = $1`, slug)
	if err != nil {
		errorResponse(w, http.StatusInternalServerError, "failed to clear centerline")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// --- Helpers ----------------------------------------------------------------

// flowColor maps a flow status to its hex color for MapLibre line-color expressions.
// These values should stay in sync with the frontend's style constants.
func flowColor(status string) string {
	switch status {
	case "runnable":
		return "#22c55e" // emerald-500 — go paddle
	case "caution":
		return "#eab308" // yellow-500 — minimum/pushy
	case "low":
		return "#ef4444" // red-500 — too low, not runnable
	case "flood":
		return "#3b82f6" // blue-500 — too much water
	default:
		return "#6b7280" // gray-500 — unknown/no data
	}
}

// rawGeometry wraps a pre-serialized GeoJSON geometry blob so it passes through
// the JSON encoder without double-encoding. The centerline comes out of PostGIS
// as a JSON string via ST_AsGeoJSON; we want it embedded as an object, not a string.
type rawGeometry []byte

func (g rawGeometry) MarshalJSON() ([]byte, error) {
	if len(g) == 0 {
		return []byte("null"), nil
	}
	return g, nil
}

// rawJSON passes a JSONB []byte from Postgres straight through the JSON encoder.
// Used for info_links so the array of {label,url} objects isn't double-encoded.
type rawJSON []byte

func (j rawJSON) MarshalJSON() ([]byte, error) {
	if len(j) == 0 {
		return []byte("[]"), nil
	}
	return j, nil
}

// parseBBoxParam parses bbox=west,south,east,north from the query string.
// GlobalAsk handles POST /api/v1/ask
//
// Accepts {"question": "..."}, identifies the reach from the question text,
// then answers using that reach's embedded content.
// Returns the answer plus the matched reach slug and name.
func (h *ReachHandler) GlobalAsk(w http.ResponseWriter, r *http.Request) {
	if h.asker == nil {
		errorResponse(w, http.StatusServiceUnavailable, "river assistant not configured")
		return
	}

	var body struct {
		Question string `json:"question"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || strings.TrimSpace(body.Question) == "" {
		errorResponse(w, http.StatusBadRequest, "question is required")
		return
	}

	// Load all reach slugs for identification.
	rows, err := h.db.Query(r.Context(), `SELECT slug, name FROM reaches ORDER BY name`)
	if err != nil {
		errorResponse(w, http.StatusInternalServerError, "could not load reaches")
		return
	}
	defer rows.Close()
	type reachStub struct{ slug, name string }
	var all []reachStub
	slugs := []string{}
	for rows.Next() {
		var s reachStub
		rows.Scan(&s.slug, &s.name)
		all = append(all, s)
		slugs = append(slugs, s.slug)
	}

	askCtx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	// Step 1 — identify reach(es).
	identified, err := h.asker.IdentifyReach(askCtx, body.Question, slugs)
	if err != nil {
		log.Printf("global ask identify: %v", err)
		errorResponse(w, http.StatusInternalServerError, fmt.Sprintf("could not identify reach: %v", err))
		return
	}

	if len(identified.Slugs) == 0 {
		jsonResponse(w, http.StatusOK, map[string]any{
			"results": []any{},
			"answer":  "I couldn't identify a specific reach from your question. Try asking about a named run — for example, \"What flows are good for Browns Canyon?\"",
		})
		return
	}

	// Step 2 — answer each matched reach in parallel (up to 3).
	type reachResult struct {
		Answer    string `json:"answer"`
		ReachSlug string `json:"reach_slug"`
		ReachName string `json:"reach_name"`
	}
	results := make([]reachResult, len(identified.Slugs))
	var wg sync.WaitGroup
	for i, slug := range identified.Slugs {
		wg.Add(1)
		go func(i int, slug string) {
			defer wg.Done()
			var reachID, reachName string
			if err := h.db.QueryRow(askCtx, `SELECT id, name FROM reaches WHERE slug = $1`, slug).Scan(&reachID, &reachName); err != nil {
				log.Printf("global ask: reach not found for slug %q: %v", slug, err)
				return
			}
			answer, err := h.asker.Answer(askCtx, reachID, reachName, identified.Question)
			if err != nil {
				log.Printf("global ask answer [%s]: %v", slug, err)
				return
			}
			results[i] = reachResult{Answer: answer, ReachSlug: slug, ReachName: reachName}
		}(i, slug)
	}
	wg.Wait()

	// Filter slots that failed (empty Answer).
	var finalResults []reachResult
	for _, rr := range results {
		if rr.Answer != "" {
			finalResults = append(finalResults, rr)
		}
	}
	if len(finalResults) == 0 {
		errorResponse(w, http.StatusInternalServerError, "could not generate answer")
		return
	}

	jsonResponse(w, http.StatusOK, map[string]any{"results": finalResults})
}

// Ask handles POST /api/v1/reaches/{slug}/ask
//
// Accepts {"question": "..."} and returns Claude's answer grounded in the
// reach's embedded content (rapids, access points, descriptions, flow ranges).
// Returns 503 if the AI keys are not configured.
func (h *ReachHandler) Ask(w http.ResponseWriter, r *http.Request) {
	if h.asker == nil {
		errorResponse(w, http.StatusServiceUnavailable, "river assistant not configured")
		return
	}

	slug := chi.URLParam(r, "slug")

	var body struct {
		Question string `json:"question"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || strings.TrimSpace(body.Question) == "" {
		errorResponse(w, http.StatusBadRequest, "question is required")
		return
	}

	// Look up the reach ID and name by slug.
	var reachID, reachName string
	err := h.db.QueryRow(r.Context(), `
		SELECT id, name FROM reaches WHERE slug = $1
	`, slug).Scan(&reachID, &reachName)
	if err != nil {
		errorResponse(w, http.StatusNotFound, "reach not found")
		return
	}

	// Use a detached context with a generous timeout — Voyage's free tier (3 RPM)
	// can retry up to 22s, which outlasts the default HTTP request context.
	askCtx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	answer, err := h.asker.Answer(askCtx, reachID, reachName, body.Question)
	if err != nil {
		log.Printf("reach ask [%s]: %v", slug, err)
		errorResponse(w, http.StatusInternalServerError, "could not generate answer")
		return
	}

	jsonResponse(w, http.StatusOK, map[string]string{"answer": answer})
}

// Required for the reach map endpoint — without a viewport bound the result
// set could be enormous.
func parseBBoxParam(r *http.Request) (*searchBBox, error) {
	raw := r.URL.Query().Get("bbox")
	if raw == "" {
		return nil, fmt.Errorf("bbox is required")
	}
	parts := strings.Split(raw, ",")
	if len(parts) != 4 {
		return nil, fmt.Errorf("bbox must be west,south,east,north")
	}
	floats, err := parseFloats(parts)
	if err != nil {
		return nil, fmt.Errorf("bbox: %w", err)
	}
	return &searchBBox{
		West: floats[0], South: floats[1],
		East: floats[2], North: floats[3],
	}, nil
}
