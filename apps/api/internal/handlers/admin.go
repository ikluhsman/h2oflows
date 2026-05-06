package handlers

import (
	"context"
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
		       rv.verified, COUNT(re.id) AS reach_count,
		       COUNT(g.id) FILTER (WHERE g.poll_health = 'degraded')    AS gauges_degraded,
		       COUNT(g.id) FILTER (WHERE g.poll_health = 'stale')       AS gauges_stale,
		       COUNT(g.id) FILTER (WHERE g.poll_health = 'unreachable') AS gauges_unreachable
		FROM rivers rv
		LEFT JOIN reaches re ON re.river_id = rv.id
		LEFT JOIN gauges g ON g.id = re.primary_gauge_id
		GROUP BY rv.id
		ORDER BY rv.name
	`)
	if err != nil {
		errorResponse(w, http.StatusInternalServerError, "query failed")
		return
	}
	defer rows.Close()

	type River struct {
		ID                string  `json:"id"`
		Slug              string  `json:"slug"`
		Name              string  `json:"name"`
		GNISID            *string `json:"gnis_id"`
		Basin             *string `json:"basin"`
		BasinLocked       bool    `json:"basin_locked"`
		StateAbbr         *string `json:"state_abbr"`
		Verified          bool    `json:"verified"`
		ReachCount        int     `json:"reach_count"`
		GaugesDegraded    int     `json:"gauges_degraded"`
		GaugesStale       int     `json:"gauges_stale"`
		GaugesUnreachable int     `json:"gauges_unreachable"`
	}

	rivers := make([]River, 0)
	for rows.Next() {
		var rv River
		if err := rows.Scan(
			&rv.ID, &rv.Slug, &rv.Name, &rv.GNISID, &rv.Basin, &rv.BasinLocked, &rv.StateAbbr,
			&rv.Verified, &rv.ReachCount,
			&rv.GaugesDegraded, &rv.GaugesStale, &rv.GaugesUnreachable,
		); err != nil {
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
		ID            string   `json:"id"`
		Slug          string   `json:"slug"`
		Name          string   `json:"name"`
		CommonName    *string  `json:"common_name"`
		ClassMin      *float64 `json:"class_min"`
		ClassMax      *float64 `json:"class_max"`
		HasCenterline bool     `json:"has_centerline"`
		StateAbbr     *string  `json:"state_abbr"`
		RiverOrder    *int16   `json:"river_order"`
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
		       (centerline IS NOT NULL) AS has_centerline, state_abbr, river_order
		FROM reaches
		WHERE river_id = $1
		ORDER BY river_order NULLS LAST, name
	`, rv.ID)
	if err != nil {
		errorResponse(w, http.StatusInternalServerError, "query failed")
		return
	}
	defer rows.Close()

	rv.Reaches = make([]Reach, 0)
	for rows.Next() {
		var re Reach
		if err := rows.Scan(&re.ID, &re.Slug, &re.Name, &re.CommonName, &re.ClassMin, &re.ClassMax, &re.HasCenterline, &re.StateAbbr, &re.RiverOrder); err != nil {
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
		HUC8      *string `json:"huc8"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		errorResponse(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	if body.Slug == "" || body.Name == "" {
		errorResponse(w, http.StatusBadRequest, "slug and name are required")
		return
	}

	// Auto-fill geo meta from GNIS ID when basin/state are not explicitly provided.
	if body.GnisID != nil && *body.GnisID != "" && body.Basin == nil && body.StateAbbr == nil {
		stateAbbr, basin, huc8 := riverMetaFromGNIS(r.Context(), *body.GnisID)
		if stateAbbr != "" {
			body.StateAbbr = &stateAbbr
		}
		if basin != "" {
			body.Basin = &basin
		}
		if huc8 != "" {
			body.HUC8 = &huc8
		}
	}

	var id string
	err := h.db.QueryRow(r.Context(), `
		INSERT INTO rivers (slug, name, basin, state_abbr, gnis_id, huc8)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id
	`, body.Slug, body.Name, body.Basin, body.StateAbbr, body.GnisID, body.HUC8).Scan(&id)
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

// riverMetaFromGNIS resolves a GNIS stream ID to state abbreviation, canonical
// basin label, and HUC8 by querying NHD → TIGERweb + WBD. Returns zero values
// without error when the lookup fails — callers treat this as best-effort.
func riverMetaFromGNIS(ctx context.Context, gnisID string) (stateAbbr, basin, huc8 string) {
	coord, err := nldi.NHDCoordByGNISID(ctx, gnisID)
	if err != nil {
		log.Printf("riverMetaFromGNIS %s: nhd: %v", gnisID, err)
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

	stateAbbr = stateRes.val
	if stateAbbr == "" && basinRes.info.States != "" {
		stateAbbr = strings.SplitN(basinRes.info.States, ",", 2)[0]
	}
	basin = gauge.CanonicalBasin(basinRes.info.HUC8)
	huc8 = basinRes.info.HUC8
	return
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
		// Insert a new river, auto-filling geo meta from GNIS ID when present.
		var gnisParam interface{}
		var stateAbbr, basin, huc8 string
		if body.GnisID != "" {
			gnisParam = body.GnisID
			stateAbbr, basin, huc8 = riverMetaFromGNIS(ctx, body.GnisID)
		}
		// verified = true when GNIS match confirms the river identity, false otherwise.
		verified := body.GnisID != ""
		if err := h.db.QueryRow(ctx, `
			INSERT INTO rivers (slug, name, gnis_id, state_abbr, basin, huc8, verified)
			VALUES ($1, $2, $3, NULLIF($4,''), NULLIF($5,''), NULLIF($6,''), $7)
			RETURNING id, name, slug, gnis_id
		`, riverSlug, body.RiverName, gnisParam, stateAbbr, basin, huc8, verified).Scan(&riverID, &riverName, &riverSlugOut, &riverGnisID); err != nil {
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

// GroupedReaches returns all assigned reaches grouped by reach state → river,
// sorted by river_order NULLS LAST then name. Designed for the admin list view.
// GET /api/v1/admin/reaches/grouped
func (h *AdminHandler) GroupedReaches(w http.ResponseWriter, r *http.Request) {
	rows, err := h.db.Query(r.Context(), `
		SELECT
			re.id, re.slug, re.name, re.common_name, re.river_order,
			COALESCE(re.state_abbr, '') AS state_abbr,
			(re.centerline IS NOT NULL) AS has_centerline,
			rv.id AS river_id, rv.slug AS river_slug, rv.name AS river_name, COALESCE(rv.basin, '') AS river_basin
		FROM reaches re
		JOIN rivers rv ON rv.id = re.river_id
		ORDER BY
			COALESCE(re.state_abbr, 'ZZZ') ASC,
			rv.name ASC,
			re.river_order NULLS LAST,
			re.name ASC
	`)
	if err != nil {
		errorResponse(w, http.StatusInternalServerError, "query failed")
		return
	}
	defer rows.Close()

	type ReachRow struct {
		ID            string  `json:"id"`
		Slug          string  `json:"slug"`
		Name          string  `json:"name"`
		CommonName    *string `json:"common_name"`
		RiverOrder    *int16  `json:"river_order"`
		StateAbbr     string  `json:"state_abbr"`
		HasCenterline bool    `json:"has_centerline"`
	}
	type RiverGroup struct {
		RiverID   string     `json:"river_id"`
		RiverSlug string     `json:"river_slug"`
		RiverName string     `json:"river_name"`
		RiverBasin string    `json:"river_basin"`
		Reaches   []ReachRow `json:"reaches"`
	}
	type StateGroup struct {
		State  string       `json:"state"`
		Rivers []RiverGroup `json:"rivers"`
	}

	// Preserve insertion order while deduping state and river keys.
	var stateOrder []string
	stateIndex := map[string]int{}
	riverIndex := map[string]map[string]int{} // state → riverID → index
	var groups []StateGroup

	for rows.Next() {
		var re ReachRow
		var riverID, riverSlug, riverName, riverBasin string
		if err := rows.Scan(&re.ID, &re.Slug, &re.Name, &re.CommonName, &re.RiverOrder,
			&re.StateAbbr, &re.HasCenterline,
			&riverID, &riverSlug, &riverName, &riverBasin); err != nil {
			continue
		}

		state := re.StateAbbr
		if state == "" {
			state = "—"
		}

		si, ok := stateIndex[state]
		if !ok {
			si = len(groups)
			stateIndex[state] = si
			stateOrder = append(stateOrder, state)
			groups = append(groups, StateGroup{State: state, Rivers: []RiverGroup{}})
			riverIndex[state] = map[string]int{}
		}

		ri, ok := riverIndex[state][riverID]
		if !ok {
			ri = len(groups[si].Rivers)
			riverIndex[state][riverID] = ri
			groups[si].Rivers = append(groups[si].Rivers, RiverGroup{
				RiverID:   riverID,
				RiverSlug: riverSlug,
				RiverName: riverName,
				RiverBasin: riverBasin,
				Reaches:   []ReachRow{},
			})
		}

		groups[si].Rivers[ri].Reaches = append(groups[si].Rivers[ri].Reaches, re)
	}

	_ = stateOrder
	jsonResponse(w, http.StatusOK, groups)
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

// ReorderReachesForRiver handles POST /api/v1/admin/rivers/{riverSlug}/reorder-reaches
//
// Assigns river_order 1..N to all reaches on the river, sorted by start_point
// longitude. Direction is auto-detected: if the sum of (end_lng - start_lng) across
// all reaches with both points is positive, the river flows W→E (ASC); otherwise
// E→W (DESC, e.g. Colorado River flowing toward Utah). Falls back to ASC when
// no reaches have both endpoints set.
func (h *AdminHandler) ReorderReachesForRiver(w http.ResponseWriter, r *http.Request) {
	riverSlug := chi.URLParam(r, "riverSlug")
	if riverSlug == "" {
		errorResponse(w, http.StatusBadRequest, "riverSlug required")
		return
	}

	// Detect flow direction: sum (end_lng - start_lng) across reaches with both points.
	var lngDelta float64
	_ = h.db.QueryRow(r.Context(), `
		SELECT COALESCE(SUM(
			ST_X(re.end_point::geometry) - ST_X(re.start_point::geometry)
		), 0)
		FROM reaches re
		JOIN rivers rv ON rv.id = re.river_id
		WHERE rv.slug = $1
		  AND re.start_point IS NOT NULL
		  AND re.end_point IS NOT NULL
	`, riverSlug).Scan(&lngDelta)

	orderDir := "ASC"
	if lngDelta < 0 {
		orderDir = "DESC"
	}

	rows, err := h.db.Query(r.Context(), fmt.Sprintf(`
		SELECT re.id
		FROM reaches re
		JOIN rivers rv ON rv.id = re.river_id
		WHERE rv.slug = $1
		ORDER BY
			ST_X(re.start_point::geometry) %s NULLS LAST,
			re.name ASC
	`, orderDir), riverSlug)
	if err != nil {
		errorResponse(w, http.StatusInternalServerError, "query failed")
		return
	}
	defer rows.Close()

	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err == nil {
			ids = append(ids, id)
		}
	}
	if err := rows.Err(); err != nil {
		errorResponse(w, http.StatusInternalServerError, "scan failed")
		return
	}
	if len(ids) == 0 {
		jsonResponse(w, http.StatusOK, map[string]any{"updated": 0})
		return
	}

	tx, err := h.db.Begin(r.Context())
	if err != nil {
		errorResponse(w, http.StatusInternalServerError, "tx failed")
		return
	}
	defer tx.Rollback(r.Context())

	for i, id := range ids {
		if _, err := tx.Exec(r.Context(),
			`UPDATE reaches SET river_order = $1 WHERE id = $2`,
			i+1, id,
		); err != nil {
			errorResponse(w, http.StatusInternalServerError, "update failed")
			return
		}
	}
	if err := tx.Commit(r.Context()); err != nil {
		errorResponse(w, http.StatusInternalServerError, "commit failed")
		return
	}

	jsonResponse(w, http.StatusOK, map[string]any{"updated": len(ids)})
}
