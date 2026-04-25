package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/h2oflow/h2oflow/apps/api/internal/ai"
	"github.com/h2oflow/h2oflow/apps/api/internal/kmlimport"
	"github.com/h2oflow/h2oflow/apps/api/internal/nldi"
)

type NLDIHandler struct {
	db           *pgxpool.Pool
	anthropicKey string
}

func NewNLDIHandler(db *pgxpool.Pool) *NLDIHandler { return &NLDIHandler{db: db} }

func (h *NLDIHandler) WithAnthropicKey(key string) *NLDIHandler {
	h.anthropicKey = key
	return h
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
	Name           string   `json:"name"`
	CommonName     string   `json:"common_name"`
	RiverName      string   `json:"river_name"`
	UpComID        string   `json:"up_comid"`   // upstream (put-in) ComID — required
	DownComID      string   `json:"down_comid"` // downstream (take-out) ComID — required
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

	slug := buildSlug(req.RiverName, req.Name)
	ctx := r.Context()

	days := req.MultiDayDays
	if days < 1 {
		days = 1
	}

	var reachID string
	err := h.db.QueryRow(ctx, `
		INSERT INTO reaches (
			slug, name, common_name, river_name,
			class_min, class_max,
			put_in_comid, take_out_comid, anchor_comid,
			centerline_source,
			description, permit_required, multi_day_days
		) VALUES (
			$1, $2, NULLIF($3,''), NULLIF($4,''),
			$5, $6,
			$7, $8, $7,
			'nldi',
			NULLIF($9,''), $10, $11
		)
		RETURNING id
	`, slug, req.Name, req.CommonName, req.RiverName,
		req.ClassMin, req.ClassMax,
		req.UpComID, req.DownComID,
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
		classMin, classMax              *float64
		description                     *string
		permitRequired                  bool
		multiDayDays                    int
		putInComID, takeOutComID        *string
		putInLng, putInLat              *float64
		takeOutLng, takeOutLat          *float64
	)
	err := h.db.QueryRow(ctx, `
		SELECT
			id, name, COALESCE(river_name,''), COALESCE(common_name,''),
			class_min, class_max, description,
			COALESCE(permit_required, false), COALESCE(multi_day_days, 1),
			put_in_comid, take_out_comid,
			ST_X(put_in::geometry),  ST_Y(put_in::geometry),
			ST_X(take_out::geometry), ST_Y(take_out::geometry)
		FROM reaches WHERE slug = $1
	`, slug).Scan(
		&id, &name, &riverName, &commonName,
		&classMin, &classMax, &description,
		&permitRequired, &multiDayDays,
		&putInComID, &takeOutComID,
		&putInLng, &putInLat, &takeOutLng, &takeOutLat,
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
		"id":              id,
		"slug":            slug,
		"name":            name,
		"river_name":      riverName,
		"common_name":     commonName,
		"class_min":       classMin,
		"class_max":       classMax,
		"description":     description,
		"permit_required": permitRequired,
		"multi_day_days":  multiDayDays,
		"put_in_comid":    putInComID,
		"take_out_comid":  takeOutComID,
		"put_in":          putIn,
		"take_out":        takeOut,
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
		CommonName     string   `json:"common_name"`
		RiverName      string   `json:"river_name"`
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
	ctx := r.Context()
	tag, err := h.db.Exec(ctx, `
		UPDATE reaches SET
			name            = $1,
			common_name     = NULLIF($2, ''),
			river_name      = NULLIF($3, ''),
			class_min       = $4,
			class_max       = $5,
			permit_required = $6,
			multi_day_days  = $7
		WHERE slug = $8
	`, req.Name, req.CommonName, req.RiverName,
		req.ClassMin, req.ClassMax,
		req.PermitRequired, days,
		slug,
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
// modified — only centerline, length_mi, put_in_comid, take_out_comid update.
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

	var lengthMi *float64
	var putInComID, takeOutComID *string
	_ = h.db.QueryRow(ctx, `
		SELECT length_mi, put_in_comid, take_out_comid FROM reaches WHERE id = $1
	`, reachID).Scan(&lengthMi, &putInComID, &takeOutComID)

	jsonResponse(w, http.StatusOK, map[string]any{
		"slug":           slug,
		"length_mi":      lengthMi,
		"put_in_comid":   putInComID,
		"take_out_comid": takeOutComID,
	})
}

// UpdateReachCenterlineByComID handles POST /api/v1/admin/reaches/{slug}/nldi-centerline-by-comid
//
// Re-traces the reach's centerline from the supplied upstream/downstream ComIDs
// while keeping the existing put_in/take_out access point coordinates intact.
// The access points are looked up from the reaches row and used solely for
// trimming the merged mainstem to the exact reach extent.
func (h *NLDIHandler) UpdateReachCenterlineByComID(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")

	var req struct {
		UpComID   string `json:"up_comid"`
		DownComID string `json:"down_comid"`
		DryRun    bool   `json:"dry_run"`
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
		reachID                                   string
		putInLng, putInLat, takeOutLng, takeOutLat *float64
	)
	err := h.db.QueryRow(ctx, `
		SELECT id,
		       ST_X(put_in::geometry),  ST_Y(put_in::geometry),
		       ST_X(take_out::geometry), ST_Y(take_out::geometry)
		FROM reaches WHERE slug = $1
	`, slug).Scan(&reachID, &putInLng, &putInLat, &takeOutLng, &takeOutLat)
	if err != nil {
		errorResponse(w, http.StatusNotFound, fmt.Sprintf("reach %q not found", slug))
		return
	}
	if putInLat == nil || putInLng == nil || takeOutLat == nil || takeOutLng == nil {
		errorResponse(w, http.StatusBadRequest, "reach is missing put-in or take-out access points")
		return
	}

	if err := kmlimport.SyncCenterlineNLDIByComID(ctx, h.db, slug,
		req.UpComID, req.DownComID,
		*putInLng, *putInLat, *takeOutLng, *takeOutLat,
		req.DryRun); err != nil {
		errorResponse(w, http.StatusBadGateway, fmt.Sprintf("nldi centerline: %v", err))
		return
	}

	if req.DryRun {
		jsonResponse(w, http.StatusOK, map[string]any{"dry_run": true})
		return
	}

	var lengthMi *float64
	var putInComID, takeOutComID *string
	_ = h.db.QueryRow(ctx, `
		SELECT length_mi, put_in_comid, take_out_comid FROM reaches WHERE id = $1
	`, reachID).Scan(&lengthMi, &putInComID, &takeOutComID)

	jsonResponse(w, http.StatusOK, map[string]any{
		"slug":           slug,
		"length_mi":      lengthMi,
		"put_in_comid":   putInComID,
		"take_out_comid": takeOutComID,
	})
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
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		errorResponse(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	ctx := r.Context()
	tag, err := h.db.Exec(ctx,
		`UPDATE reaches SET description = $1 WHERE slug = $2`,
		req.Description, slug,
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
