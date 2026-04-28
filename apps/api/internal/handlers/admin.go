package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/h2oflow/h2oflow/apps/api/internal/auth"
	"github.com/h2oflow/h2oflow/apps/api/internal/kmlimport"
	"github.com/h2oflow/h2oflow/apps/api/internal/nldi"
	gauge "github.com/h2oflow/h2oflow/packages/gauge-core"
)

type AdminHandler struct {
	db *pgxpool.Pool
}

func NewAdminHandler(db *pgxpool.Pool) *AdminHandler {
	return &AdminHandler{db: db}
}

// SlugCheck reports whether a reach slug is available (not already taken).
// GET /api/v1/admin/slug-check?slug=…&exclude=…
// exclude is the caller's current slug so an in-place rename doesn't self-conflict.
func (h *AdminHandler) SlugCheck(w http.ResponseWriter, r *http.Request) {
	slug := r.URL.Query().Get("slug")
	if slug == "" {
		errorResponse(w, http.StatusBadRequest, "slug is required")
		return
	}
	exclude := r.URL.Query().Get("exclude")
	var exists bool
	_ = h.db.QueryRow(r.Context(),
		`SELECT EXISTS(SELECT 1 FROM reaches WHERE slug = $1 AND slug != $2)`,
		slug, exclude,
	).Scan(&exists)
	jsonResponse(w, http.StatusOK, map[string]bool{"available": !exists})
}

// ── Rivers ────────────────────────────────────────────────────────────────────

// ListRivers returns all rivers with their reach count.
// GET /api/v1/admin/rivers
func (h *AdminHandler) ListRivers(w http.ResponseWriter, r *http.Request) {
	rows, err := h.db.Query(r.Context(), `
		SELECT rv.id, rv.slug, rv.name, rv.gnis_id, rv.basin, rv.basin_locked, rv.state_abbr,
		       COUNT(re.id) AS reach_count
		FROM rivers rv
		LEFT JOIN reaches re ON re.river_id = rv.id
		GROUP BY rv.id
		ORDER BY rv.name
	`)
	if err != nil {
		errorResponse(w, http.StatusInternalServerError, "query failed")
		return
	}
	defer rows.Close()

	type River struct {
		ID          string  `json:"id"`
		Slug        string  `json:"slug"`
		Name        string  `json:"name"`
		GNISID      *string `json:"gnis_id"`
		Basin       *string `json:"basin"`
		BasinLocked bool    `json:"basin_locked"`
		StateAbbr   *string `json:"state_abbr"`
		ReachCount  int     `json:"reach_count"`
	}

	rivers := make([]River, 0)
	for rows.Next() {
		var rv River
		if err := rows.Scan(&rv.ID, &rv.Slug, &rv.Name, &rv.GNISID, &rv.Basin, &rv.BasinLocked, &rv.StateAbbr, &rv.ReachCount); err != nil {
			continue
		}
		rivers = append(rivers, rv)
	}
	jsonResponse(w, http.StatusOK, rivers)
}

// GetRiver returns a single river and its reaches.
// GET /api/v1/admin/rivers/{riverSlug}
func (h *AdminHandler) GetRiver(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "riverSlug")

	type Reach struct {
		ID         string  `json:"id"`
		Slug       string  `json:"slug"`
		Name       string  `json:"name"`
		CommonName *string `json:"common_name"`
		ClassMin   *float64 `json:"class_min"`
		ClassMax   *float64 `json:"class_max"`
		HasCenterline bool  `json:"has_centerline"`
	}
	type RiverDetail struct {
		ID          string  `json:"id"`
		Slug        string  `json:"slug"`
		Name        string  `json:"name"`
		GNISID      *string `json:"gnis_id"`
		Basin       *string `json:"basin"`
		BasinLocked bool    `json:"basin_locked"`
		StateAbbr   *string `json:"state_abbr"`
		Reaches     []Reach `json:"reaches"`
	}

	var rv RiverDetail
	err := h.db.QueryRow(r.Context(), `
		SELECT id, slug, name, gnis_id, basin, basin_locked, state_abbr FROM rivers WHERE slug = $1
	`, slug).Scan(&rv.ID, &rv.Slug, &rv.Name, &rv.GNISID, &rv.Basin, &rv.BasinLocked, &rv.StateAbbr)
	if err != nil {
		errorResponse(w, http.StatusNotFound, "river not found")
		return
	}

	// Pull the HUC-derived basin from the primary gauge of any linked reach.
	// This lets the admin compare the system-derived value against the stored one.
	rows, err := h.db.Query(r.Context(), `
		SELECT id, slug, name, common_name, class_min, class_max,
		       (centerline IS NOT NULL) AS has_centerline
		FROM reaches
		WHERE river_id = $1
		ORDER BY name
	`, rv.ID)
	if err != nil {
		errorResponse(w, http.StatusInternalServerError, "query failed")
		return
	}
	defer rows.Close()

	rv.Reaches = make([]Reach, 0)
	for rows.Next() {
		var re Reach
		if err := rows.Scan(&re.ID, &re.Slug, &re.Name, &re.CommonName, &re.ClassMin, &re.ClassMax, &re.HasCenterline); err != nil {
			continue
		}
		rv.Reaches = append(rv.Reaches, re)
	}
	jsonResponse(w, http.StatusOK, rv)
}

// CreateRiver creates a new river.
// POST /api/v1/admin/rivers
func (h *AdminHandler) CreateRiver(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Slug      string  `json:"slug"`
		Name      string  `json:"name"`
		Basin     *string `json:"basin"`
		StateAbbr *string `json:"state_abbr"`
		GnisID    *string `json:"gnis_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		errorResponse(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	if body.Slug == "" || body.Name == "" {
		errorResponse(w, http.StatusBadRequest, "slug and name are required")
		return
	}

	var id string
	err := h.db.QueryRow(r.Context(), `
		INSERT INTO rivers (slug, name, basin, state_abbr, gnis_id)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id
	`, body.Slug, body.Name, body.Basin, body.StateAbbr, body.GnisID).Scan(&id)
	if err != nil {
		errorResponse(w, http.StatusConflict, "river already exists or invalid data")
		return
	}
	jsonResponse(w, http.StatusCreated, map[string]string{"id": id})
}

// DeleteRiver permanently deletes a river and unlinks its reaches.
// Reaches are NOT deleted — they remain but lose their river association.
// DELETE /api/v1/admin/rivers/{riverSlug}
func (h *AdminHandler) DeleteRiver(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "riverSlug")

	// Unlink reaches first so the FK constraint doesn't block deletion.
	if _, err := h.db.Exec(r.Context(),
		`UPDATE reaches SET river_id = NULL WHERE river_id = (SELECT id FROM rivers WHERE slug = $1)`,
		slug,
	); err != nil {
		errorResponse(w, http.StatusInternalServerError, "unlink reaches failed")
		return
	}

	tag, err := h.db.Exec(r.Context(), `DELETE FROM rivers WHERE slug = $1`, slug)
	if err != nil {
		errorResponse(w, http.StatusInternalServerError, "delete failed")
		return
	}
	if tag.RowsAffected() == 0 {
		errorResponse(w, http.StatusNotFound, "river not found")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// UpdateRiver updates a river's metadata.
// PUT /api/v1/admin/rivers/{riverSlug}
// When basin is provided it is always written (even if identical) so the
// caller can explicitly set it. basin_locked controls whether the metadata
// sync is allowed to overwrite it in the future.
func (h *AdminHandler) UpdateRiver(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "riverSlug")
	var body struct {
		Name        *string `json:"name"`
		Basin       *string `json:"basin"`
		BasinLocked *bool   `json:"basin_locked"`
		StateAbbr   *string `json:"state_abbr"`
		GnisID      *string `json:"gnis_id"`
		HUC8        *string `json:"huc8"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		errorResponse(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	_, err := h.db.Exec(r.Context(), `
		UPDATE rivers
		SET name         = COALESCE($2, name),
		    basin        = COALESCE($3, basin),
		    basin_locked = COALESCE($4, basin_locked),
		    state_abbr   = COALESCE($5, state_abbr),
		    gnis_id      = COALESCE($6, gnis_id),
		    huc8         = COALESCE($7, huc8)
		WHERE slug = $1
	`, slug, body.Name, body.Basin, body.BasinLocked, body.StateAbbr, body.GnisID, body.HUC8)
	if err != nil {
		errorResponse(w, http.StatusInternalServerError, "update failed")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// AutoAssignRiver upserts a river by GNIS ID (preferred) or name, then links
// the reach to it. Called after the admin fetches the river name from NLDI.
// POST /api/v1/admin/reaches/{slug}/auto-river
func (h *AdminHandler) AutoAssignRiver(w http.ResponseWriter, r *http.Request) {
	reachSlug := chi.URLParam(r, "slug")
	var body struct {
		RiverName string `json:"river_name"`
		GnisID    string `json:"gnis_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		errorResponse(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	body.RiverName = strings.TrimSpace(body.RiverName)
	body.GnisID = strings.TrimSpace(body.GnisID)
	if body.RiverName == "" {
		errorResponse(w, http.StatusBadRequest, "river_name is required")
		return
	}

	ctx := r.Context()
	riverSlug := kmlimport.Slugify(body.RiverName)

	var riverID, riverName, riverSlugOut string
	var riverGnisID *string

	// Look up existing river: prefer gnis_id match, fall back to name match.
	// This avoids INSERT conflicts when the river already exists under either key.
	if body.GnisID != "" {
		_ = h.db.QueryRow(ctx,
			`SELECT id, name, slug, gnis_id FROM rivers WHERE gnis_id = $1`,
			body.GnisID).Scan(&riverID, &riverName, &riverSlugOut, &riverGnisID)
	}
	if riverID == "" {
		_ = h.db.QueryRow(ctx,
			`SELECT id, name, slug, gnis_id FROM rivers WHERE lower(name) = lower($1) LIMIT 1`,
			body.RiverName).Scan(&riverID, &riverName, &riverSlugOut, &riverGnisID)
	}
	if riverID != "" {
		// River exists — backfill gnis_id if we now have one and it was missing.
		if body.GnisID != "" && riverGnisID == nil {
			_, _ = h.db.Exec(ctx, `UPDATE rivers SET gnis_id = $1 WHERE id = $2`, body.GnisID, riverID)
			g := body.GnisID
			riverGnisID = &g
		}
	} else {
		// Insert a new river.
		var gnisParam interface{}
		if body.GnisID != "" {
			gnisParam = body.GnisID
		}
		if err := h.db.QueryRow(ctx, `
			INSERT INTO rivers (slug, name, gnis_id)
			VALUES ($1, $2, $3)
			RETURNING id, name, slug, gnis_id
		`, riverSlug, body.RiverName, gnisParam).Scan(&riverID, &riverName, &riverSlugOut, &riverGnisID); err != nil {
			errorResponse(w, http.StatusInternalServerError, fmt.Sprintf("insert river: %v", err))
			return
		}
	}

	// Link the reach and keep river_name text in sync.
	_, err := h.db.Exec(ctx, `
		UPDATE reaches SET river_id = $1, river_name = $2 WHERE slug = $3
	`, riverID, riverName, reachSlug)
	if err != nil {
		errorResponse(w, http.StatusInternalServerError, "assign river failed")
		return
	}

	jsonResponse(w, http.StatusOK, map[string]any{
		"river_id":   riverID,
		"river_name": riverName,
		"river_slug": riverSlugOut,
		"gnis_id":    riverGnisID,
	})
}

// ListUnassignedReaches returns reaches that have no river association.
// GET /api/v1/admin/reaches/unassigned
func (h *AdminHandler) ListUnassignedReaches(w http.ResponseWriter, r *http.Request) {
	type Reach struct {
		ID            string   `json:"id"`
		Slug          string   `json:"slug"`
		Name          string   `json:"name"`
		CommonName    *string  `json:"common_name"`
		RiverName     *string  `json:"river_name"`
		ClassMin      *float64 `json:"class_min"`
		ClassMax      *float64 `json:"class_max"`
		HasCenterline bool     `json:"has_centerline"`
	}

	rows, err := h.db.Query(r.Context(), `
		SELECT id, slug, name, common_name, river_name, class_min, class_max,
		       (centerline IS NOT NULL) AS has_centerline
		FROM reaches
		WHERE river_id IS NULL
		ORDER BY name
	`)
	if err != nil {
		errorResponse(w, http.StatusInternalServerError, "query failed")
		return
	}
	defer rows.Close()

	reaches := make([]Reach, 0)
	for rows.Next() {
		var re Reach
		if err := rows.Scan(&re.ID, &re.Slug, &re.Name, &re.CommonName, &re.RiverName,
			&re.ClassMin, &re.ClassMax, &re.HasCenterline); err != nil {
			continue
		}
		reaches = append(reaches, re)
	}
	jsonResponse(w, http.StatusOK, reaches)
}

// AssignReachToRiver sets reaches.river_id.
// PUT /api/v1/admin/reaches/{slug}/river
func (h *AdminHandler) AssignReachToRiver(w http.ResponseWriter, r *http.Request) {
	reachSlug := chi.URLParam(r, "slug")
	var body struct {
		RiverID *string `json:"river_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		errorResponse(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	_, err := h.db.Exec(r.Context(), `
		UPDATE reaches SET river_id = $2 WHERE slug = $1
	`, reachSlug, body.RiverID)
	if err != nil {
		errorResponse(w, http.StatusInternalServerError, "update failed")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ── User Roles ────────────────────────────────────────────────────────────────

type userRoleRow struct {
	ID        string  `json:"id"`
	UserID    string  `json:"user_id"`
	Email     *string `json:"email"`
	Role      string  `json:"role"`
	RiverID   *string `json:"river_id"`
	RiverName *string `json:"river_name"`
	CreatedAt time.Time `json:"created_at"`
}

// ListUserRoles returns all role assignments. Site admin only.
// GET /api/v1/admin/users/roles
func (h *AdminHandler) ListUserRoles(w http.ResponseWriter, r *http.Request) {
	rows, err := h.db.Query(r.Context(), `
		SELECT ur.id, ur.user_id, NULL::text AS email,
		       ur.role, ur.river_id, rv.name AS river_name, ur.created_at
		FROM user_roles ur
		LEFT JOIN rivers rv ON rv.id = ur.river_id
		ORDER BY ur.created_at DESC
	`)
	if err != nil {
		errorResponse(w, http.StatusInternalServerError, "query failed")
		return
	}
	defer rows.Close()

	result := make([]userRoleRow, 0)
	for rows.Next() {
		var ur userRoleRow
		if err := rows.Scan(&ur.ID, &ur.UserID, &ur.Email, &ur.Role, &ur.RiverID, &ur.RiverName, &ur.CreatedAt); err != nil {
			continue
		}
		result = append(result, ur)
	}
	jsonResponse(w, http.StatusOK, result)
}

// AssignRole grants a role to a user. Site admin only.
// POST /api/v1/admin/users/roles
func (h *AdminHandler) AssignRole(w http.ResponseWriter, r *http.Request) {
	var body struct {
		UserID  string  `json:"user_id"`
		Role    string  `json:"role"`
		RiverID *string `json:"river_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		errorResponse(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	if body.UserID == "" || body.Role == "" {
		errorResponse(w, http.StatusBadRequest, "user_id and role are required")
		return
	}

	var id string
	err := h.db.QueryRow(r.Context(), `
		INSERT INTO user_roles (user_id, role, river_id)
		VALUES ($1, $2, $3)
		ON CONFLICT DO NOTHING
		RETURNING id
	`, body.UserID, body.Role, body.RiverID).Scan(&id)
	if err != nil {
		// ON CONFLICT DO NOTHING means no rows returned if duplicate — that's fine
		jsonResponse(w, http.StatusCreated, map[string]string{"id": id})
		return
	}
	jsonResponse(w, http.StatusCreated, map[string]string{"id": id})
}

// RevokeRole removes a role assignment. Site admin only.
// DELETE /api/v1/admin/users/roles/{roleId}
func (h *AdminHandler) RevokeRole(w http.ResponseWriter, r *http.Request) {
	roleID := chi.URLParam(r, "roleId")
	_, err := h.db.Exec(r.Context(), `DELETE FROM user_roles WHERE id = $1`, roleID)
	if err != nil {
		errorResponse(w, http.StatusInternalServerError, "delete failed")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// GetMyRoles returns the caller's own role assignments.
// GET /api/v1/admin/me/roles
func (h *AdminHandler) GetMyRoles(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		errorResponse(w, http.StatusUnauthorized, "not authenticated")
		return
	}

	rows, err := h.db.Query(r.Context(), `
		SELECT ur.id, ur.user_id, NULL::text, ur.role, ur.river_id, rv.name, ur.created_at
		FROM user_roles ur
		LEFT JOIN rivers rv ON rv.id = ur.river_id
		WHERE ur.user_id = $1
	`, userID)
	if err != nil {
		errorResponse(w, http.StatusInternalServerError, "query failed")
		return
	}
	defer rows.Close()

	result := make([]userRoleRow, 0)
	for rows.Next() {
		var ur userRoleRow
		if err := rows.Scan(&ur.ID, &ur.UserID, &ur.Email, &ur.Role, &ur.RiverID, &ur.RiverName, &ur.CreatedAt); err != nil {
			continue
		}
		result = append(result, ur)
	}

	// Include site_admin status from Supabase JWT
	isSiteAdmin := auth.IsSiteAdminFromContext(r.Context())
	jsonResponse(w, http.StatusOK, map[string]any{
		"is_site_admin": isSiteAdmin,
		"is_data_admin": auth.IsDataAdminFromContext(r.Context()),
		"roles":         result,
	})
}

// AutoFillRiverMeta returns suggested state and basin for a river by querying
// external geo services at the upstream-most reach coordinate.
// GET /api/v1/admin/rivers/{riverSlug}/auto-fill
func (h *AdminHandler) AutoFillRiverMeta(w http.ResponseWriter, r *http.Request) {
	riverSlug := chi.URLParam(r, "riverSlug")
	ctx := r.Context()

	var lat, lng float64
	err := h.db.QueryRow(ctx, `
		SELECT COALESCE(ST_Y(start_point::geometry), ST_Y(ST_StartPoint(centerline::geometry))),
		       COALESCE(ST_X(start_point::geometry), ST_X(ST_StartPoint(centerline::geometry)))
		FROM reaches
		WHERE river_id = (SELECT id FROM rivers WHERE slug = $1)
		  AND (start_point IS NOT NULL OR centerline IS NOT NULL)
		LIMIT 1
	`, riverSlug).Scan(&lat, &lng)
	if err != nil {
		// No reach coordinates — fall back to GNIS ID if the river has one.
		var gnisID *string
		_ = h.db.QueryRow(ctx, `SELECT gnis_id FROM rivers WHERE slug = $1`, riverSlug).Scan(&gnisID)
		if gnisID == nil || *gnisID == "" {
			errorResponse(w, http.StatusNotFound, "no reach coordinates or GNIS ID found for river — add a GNIS ID first")
			return
		}
		coord, gErr := nldi.NHDCoordByGNISID(ctx, *gnisID)
		if gErr != nil {
			errorResponse(w, http.StatusNotFound, fmt.Sprintf("no reach coordinates found and GNIS lookup failed: %v", gErr))
			return
		}
		lat, lng = coord.Lat, coord.Lng
	}

	type stateResult struct {
		val string
		err error
	}
	type basinResult struct {
		info nldi.BasinInfo
		err  error
	}
	stateCh := make(chan stateResult, 1)
	basinCh := make(chan basinResult, 1)
	go func() { v, e := nldi.StateAt(ctx, lat, lng); stateCh <- stateResult{v, e} }()
	go func() { v, e := nldi.BasinAt(ctx, lat, lng); basinCh <- basinResult{v, e} }()

	stateRes := <-stateCh
	basinRes := <-basinCh

	if stateRes.err != nil {
		log.Printf("auto-fill river %s: state lookup: %v", riverSlug, stateRes.err)
	}
	if basinRes.err != nil {
		log.Printf("auto-fill river %s: basin lookup: %v", riverSlug, basinRes.err)
	}

	// Prefer state from TIGERweb; fall back to first state in WBD states field.
	stateAbbr := stateRes.val
	if stateAbbr == "" && basinRes.info.States != "" {
		stateAbbr = strings.SplitN(basinRes.info.States, ",", 2)[0]
	}

	jsonResponse(w, http.StatusOK, map[string]any{
		"state_abbr": stateAbbr,
		"basin":      gauge.CanonicalBasin(basinRes.info.HUC8),
		"huc8":       basinRes.info.HUC8,
		"states":     basinRes.info.States,
		"lat":        lat,
		"lng":        lng,
	})
}

// GNISLookup handles GET /api/v1/admin/rivers/gnis-lookup?gnis_id=X
//
// Resolves a GNIS stream ID to state, basin name, and HUC8. Uses the NHD
// ArcGIS flowline layer to get a representative coordinate, then calls
// TIGERweb for state and WBD for basin info. Returns 404 when the GNIS ID
// has no NHD features.
func (h *AdminHandler) GNISLookup(w http.ResponseWriter, r *http.Request) {
	gnisID := strings.TrimSpace(r.URL.Query().Get("gnis_id"))
	if gnisID == "" {
		errorResponse(w, http.StatusBadRequest, "gnis_id is required")
		return
	}
	ctx := r.Context()

	coord, err := nldi.NHDCoordByGNISID(ctx, gnisID)
	if err != nil {
		errorResponse(w, http.StatusNotFound, fmt.Sprintf("GNIS lookup: %v", err))
		return
	}

	type stateResult struct {
		val string
		err error
	}
	type basinResult struct {
		info nldi.BasinInfo
		err  error
	}
	stateCh := make(chan stateResult, 1)
	basinCh := make(chan basinResult, 1)
	go func() { v, e := nldi.StateAt(ctx, coord.Lat, coord.Lng); stateCh <- stateResult{v, e} }()
	go func() { v, e := nldi.BasinAt(ctx, coord.Lat, coord.Lng); basinCh <- basinResult{v, e} }()

	stateRes := <-stateCh
	basinRes := <-basinCh

	stateAbbr := stateRes.val
	if stateAbbr == "" && basinRes.info.States != "" {
		stateAbbr = strings.SplitN(basinRes.info.States, ",", 2)[0]
	}

	jsonResponse(w, http.StatusOK, map[string]any{
		"state_abbr": stateAbbr,
		"basin":      gauge.CanonicalBasin(basinRes.info.HUC8),
		"huc8":       basinRes.info.HUC8,
		"states":     basinRes.info.States,
		"lat":        coord.Lat,
		"lng":        coord.Lng,
	})
}
