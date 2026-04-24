package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/h2oflow/h2oflow/apps/api/internal/kmlimport"
	"github.com/h2oflow/h2oflow/apps/api/internal/nldi"
)

type NLDIHandler struct {
	db *pgxpool.Pool
}

func NewNLDIHandler(db *pgxpool.Pool) *NLDIHandler { return &NLDIHandler{db: db} }

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

type createReachRequest struct {
	Name            string       `json:"name"`
	CommonName      string       `json:"common_name"`
	RiverName       string       `json:"river_name"`
	PutIn           accessPoint  `json:"put_in"`
	TakeOut         accessPoint  `json:"take_out"`
	ClassMin        *float64     `json:"class_min"`
	ClassMax        *float64     `json:"class_max"`
	FetchCenterline bool         `json:"fetch_centerline"`
}

type accessPoint struct {
	Lat   float64 `json:"lat"`
	Lng   float64 `json:"lng"`
	Name  string  `json:"name"`
	ComID string  `json:"comid"`
}

// CreateReach handles POST /api/v1/admin/reaches.
//
// Creates a reach with its put-in and take-out access points. If
// fetch_centerline is true it calls the NLDI centerline path synchronously
// (same 500 km budget as the CLI tool) and returns the computed length_mi.
//
// The slug is derived from river_name + name using the same slugify logic as
// the KML importer. On conflict the request fails — the admin should dedupe.
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
	if req.PutIn.Lat == 0 && req.PutIn.Lng == 0 {
		errorResponse(w, http.StatusBadRequest, "put_in coordinates are required")
		return
	}
	if req.TakeOut.Lat == 0 && req.TakeOut.Lng == 0 {
		errorResponse(w, http.StatusBadRequest, "take_out coordinates are required")
		return
	}

	slug := buildSlug(req.RiverName, req.Name)
	ctx := r.Context()

	var reachID string
	err := pgx.BeginTxFunc(ctx, h.db, pgx.TxOptions{}, func(tx pgx.Tx) error {
		// Create the reach. put_in / take_out geography columns are convenience
		// denorms — SyncCenterline reads from reach_access.location, but the map
		// handler reads from these columns for display.
		err := tx.QueryRow(ctx, `
			INSERT INTO reaches (
				slug, name, common_name, river_name,
				class_min, class_max,
				put_in, take_out,
				put_in_name, take_out_name,
				put_in_comid, take_out_comid,
				anchor_comid,
				centerline_source
			) VALUES (
				$1, $2, NULLIF($3,''), NULLIF($4,''),
				$5, $6,
				ST_SetSRID(ST_MakePoint($7, $8),  4326)::geography,
				ST_SetSRID(ST_MakePoint($9, $10), 4326)::geography,
				NULLIF($11,''), NULLIF($12,''),
				NULLIF($13,''), NULLIF($14,''),
				NULLIF($13,''),
				'nldi'
			)
			RETURNING id
		`, slug, req.Name, req.CommonName, req.RiverName,
			req.ClassMin, req.ClassMax,
			req.PutIn.Lng, req.PutIn.Lat,
			req.TakeOut.Lng, req.TakeOut.Lat,
			req.PutIn.Name, req.TakeOut.Name,
			req.PutIn.ComID, req.TakeOut.ComID,
		).Scan(&reachID)
		if err != nil {
			return fmt.Errorf("insert reach: %w", err)
		}

		// Put-in access point.
		_, err = tx.Exec(ctx, `
			INSERT INTO reach_access (reach_id, access_type, name, location, data_source)
			VALUES ($1, 'put_in', NULLIF($2,''),
			        ST_SetSRID(ST_MakePoint($3, $4), 4326)::geography,
			        'admin')
		`, reachID, req.PutIn.Name, req.PutIn.Lng, req.PutIn.Lat)
		if err != nil {
			return fmt.Errorf("insert put-in: %w", err)
		}

		// Take-out access point (may be same coords as put-in for playspots).
		_, err = tx.Exec(ctx, `
			INSERT INTO reach_access (reach_id, access_type, name, location, data_source)
			VALUES ($1, 'take_out', NULLIF($2,''),
			        ST_SetSRID(ST_MakePoint($3, $4), 4326)::geography,
			        'admin')
		`, reachID, req.TakeOut.Name, req.TakeOut.Lng, req.TakeOut.Lat)
		return err
	})
	if err != nil {
		if strings.Contains(err.Error(), "duplicate key") || strings.Contains(err.Error(), "unique constraint") {
			errorResponse(w, http.StatusConflict, fmt.Sprintf("reach with slug %q already exists", slug))
			return
		}
		errorResponse(w, http.StatusInternalServerError, fmt.Sprintf("create reach: %v", err))
		return
	}

	// NLDI centerline fetch — skip for playspots (identical put-in + take-out).
	var lengthMi *float64
	isPlayspot := req.PutIn.Lat == req.TakeOut.Lat && req.PutIn.Lng == req.TakeOut.Lng
	if req.FetchCenterline && !isPlayspot {
		if err := kmlimport.SyncCenterline(context.Background(), h.db, slug, kmlimport.CenterlineNLDI, false); err != nil {
			// Non-fatal — reach is already created; admin can re-fetch later.
			_ = err
		} else {
			var mi float64
			_ = h.db.QueryRow(context.Background(),
				`SELECT length_mi FROM reaches WHERE id = $1`, reachID).Scan(&mi)
			lengthMi = &mi
		}
	}

	jsonResponse(w, http.StatusCreated, map[string]any{
		"slug":      slug,
		"id":        reachID,
		"length_mi": lengthMi,
	})
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
