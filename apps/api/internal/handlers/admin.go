package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/h2oflow/h2oflow/apps/api/internal/auth"
	gauge "github.com/h2oflow/h2oflow/packages/gauge-core"
)

type AdminHandler struct {
	db *pgxpool.Pool
}

func NewAdminHandler(db *pgxpool.Pool) *AdminHandler {
	return &AdminHandler{db: db}
}

// ── Rivers ────────────────────────────────────────────────────────────────────

// ListRivers returns all rivers with their reach count.
// GET /api/v1/admin/rivers
func (h *AdminHandler) ListRivers(w http.ResponseWriter, r *http.Request) {
	rows, err := h.db.Query(r.Context(), `
		SELECT rv.id, rv.slug, rv.name, rv.basin, rv.basin_locked, rv.state_abbr,
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
		Basin       *string `json:"basin"`
		BasinLocked bool    `json:"basin_locked"`
		StateAbbr   *string `json:"state_abbr"`
		ReachCount  int     `json:"reach_count"`
	}

	rivers := make([]River, 0)
	for rows.Next() {
		var rv River
		if err := rows.Scan(&rv.ID, &rv.Slug, &rv.Name, &rv.Basin, &rv.BasinLocked, &rv.StateAbbr, &rv.ReachCount); err != nil {
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
		ID              string  `json:"id"`
		Slug            string  `json:"slug"`
		Name            string  `json:"name"`
		Basin           *string `json:"basin"`
		BasinLocked     bool    `json:"basin_locked"`
		StateAbbr       *string `json:"state_abbr"`
		// HUC-derived basin from the primary gauge of the first linked reach.
		// Null when no linked reach has a gauge with metadata yet.
		GaugeBasin      *string `json:"gauge_basin"`       // e.g. "South Platte"
		GaugeWatershed  *string `json:"gauge_watershed"`   // e.g. "Cache La Poudre River"
		GaugeHUC8       *string `json:"gauge_huc8"`        // e.g. "10190007"
		Reaches         []Reach `json:"reaches"`
	}

	var rv RiverDetail
	err := h.db.QueryRow(r.Context(), `
		SELECT id, slug, name, basin, basin_locked, state_abbr FROM rivers WHERE slug = $1
	`, slug).Scan(&rv.ID, &rv.Slug, &rv.Name, &rv.Basin, &rv.BasinLocked, &rv.StateAbbr)
	if err != nil {
		errorResponse(w, http.StatusNotFound, "river not found")
		return
	}

	// Pull the HUC-derived basin from the primary gauge of any linked reach.
	// This lets the admin compare the system-derived value against the stored one.
	var gaugeHUC8 string
	_ = h.db.QueryRow(r.Context(), `
		SELECT g.watershed_name, g.huc8
		FROM   reaches re
		JOIN   gauges  g ON g.id = re.primary_gauge_id
		WHERE  re.river_id = $1
		  AND  g.watershed_name IS NOT NULL
		LIMIT 1
	`, rv.ID).Scan(&rv.GaugeBasin, &gaugeHUC8)
	if gaugeHUC8 != "" {
		rv.GaugeHUC8 = &gaugeHUC8
		_, watershedName := gauge.HUCNames(gaugeHUC8)
		if watershedName != "" {
			rv.GaugeWatershed = &watershedName
		}
	}

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
		INSERT INTO rivers (slug, name, basin, state_abbr)
		VALUES ($1, $2, $3, $4)
		RETURNING id
	`, body.Slug, body.Name, body.Basin, body.StateAbbr).Scan(&id)
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
		    state_abbr   = COALESCE($5, state_abbr)
		WHERE slug = $1
	`, slug, body.Name, body.Basin, body.BasinLocked, body.StateAbbr)
	if err != nil {
		errorResponse(w, http.StatusInternalServerError, "update failed")
		return
	}
	w.WriteHeader(http.StatusNoContent)
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
