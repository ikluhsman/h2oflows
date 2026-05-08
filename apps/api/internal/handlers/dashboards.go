package handlers

import (
	"encoding/json"
	"net/http"
	"regexp"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/h2oflow/h2oflow/apps/api/internal/auth"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// DashboardHandler handles /me/dashboards routes.
type DashboardHandler struct {
	db            *pgxpool.Pool
	devFallbackID string
}

func NewDashboardHandler(db *pgxpool.Pool, devFallbackID string) *DashboardHandler {
	return &DashboardHandler{db: db, devFallbackID: devFallbackID}
}

func (h *DashboardHandler) ownerID(r *http.Request) (string, bool) {
	if id, ok := auth.UserIDFromContext(r.Context()); ok {
		return id, true
	}
	if h.devFallbackID != "" {
		return h.devFallbackID, true
	}
	return "", false
}

type dashboardRow struct {
	ID        string `json:"id"`
	Slug      string `json:"slug"`
	Name      string `json:"name"`
	Position  int    `json:"position"`
	CreatedAt string `json:"created_at"`
}

// ── GET /me/dashboards ────────────────────────────────────────────────────────

func (h *DashboardHandler) List(w http.ResponseWriter, r *http.Request) {
	ownerID, ok := h.ownerID(r)
	if !ok {
		errorResponse(w, http.StatusUnauthorized, "authentication required")
		return
	}

	rows, err := h.db.Query(r.Context(), `
		SELECT id::text, slug, name, position, created_at::text
		FROM   user_dashboards
		WHERE  owner_id = $1
		ORDER  BY position, created_at
	`, ownerID)
	if err != nil {
		errorResponse(w, http.StatusInternalServerError, "query failed")
		return
	}
	defer rows.Close()

	items := []dashboardRow{}
	for rows.Next() {
		var d dashboardRow
		if err := rows.Scan(&d.ID, &d.Slug, &d.Name, &d.Position, &d.CreatedAt); err == nil {
			items = append(items, d)
		}
	}

	// First-time user: auto-create default dashboard so they always have one.
	if len(items) == 0 {
		var d dashboardRow
		err := h.db.QueryRow(r.Context(), `
			INSERT INTO user_dashboards (owner_id, slug, name, position)
			VALUES ($1, 'default', 'My Dashboard', 0)
			ON CONFLICT (owner_id, slug) DO UPDATE SET name = EXCLUDED.name
			RETURNING id::text, slug, name, position, created_at::text
		`, ownerID).Scan(&d.ID, &d.Slug, &d.Name, &d.Position, &d.CreatedAt)
		if err == nil {
			items = []dashboardRow{d}
		}
	}

	jsonResponse(w, http.StatusOK, map[string]any{"dashboards": items})
}

// ── POST /me/dashboards ───────────────────────────────────────────────────────

func (h *DashboardHandler) Create(w http.ResponseWriter, r *http.Request) {
	ownerID, ok := h.ownerID(r)
	if !ok {
		errorResponse(w, http.StatusUnauthorized, "authentication required")
		return
	}

	var body struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || strings.TrimSpace(body.Name) == "" {
		errorResponse(w, http.StatusBadRequest, "name required")
		return
	}

	slug := slugifyDashboard(body.Name)

	var d dashboardRow
	err := h.db.QueryRow(r.Context(), `
		INSERT INTO user_dashboards (owner_id, slug, name, position)
		VALUES (
			$1, $2, $3,
			COALESCE((SELECT MAX(position)+1 FROM user_dashboards WHERE owner_id = $1), 0)
		)
		RETURNING id::text, slug, name, position, created_at::text
	`, ownerID, slug, strings.TrimSpace(body.Name)).Scan(
		&d.ID, &d.Slug, &d.Name, &d.Position, &d.CreatedAt,
	)
	if err != nil {
		// Slug collision — try appending a counter
		slug = slug + "-2"
		err = h.db.QueryRow(r.Context(), `
			INSERT INTO user_dashboards (owner_id, slug, name, position)
			VALUES (
				$1, $2, $3,
				COALESCE((SELECT MAX(position)+1 FROM user_dashboards WHERE owner_id = $1), 0)
			)
			RETURNING id::text, slug, name, position, created_at::text
		`, ownerID, slug, strings.TrimSpace(body.Name)).Scan(
			&d.ID, &d.Slug, &d.Name, &d.Position, &d.CreatedAt,
		)
	}
	if err != nil {
		errorResponse(w, http.StatusInternalServerError, "create failed")
		return
	}

	jsonResponse(w, http.StatusCreated, d)
}

// ── PATCH /me/dashboards/{slug} ───────────────────────────────────────────────

func (h *DashboardHandler) Update(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	ownerID, ok := h.ownerID(r)
	if !ok {
		errorResponse(w, http.StatusUnauthorized, "authentication required")
		return
	}

	var body struct {
		Name *string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		errorResponse(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	tag, err := h.db.Exec(r.Context(), `
		UPDATE user_dashboards
		SET    name = COALESCE($1, name)
		WHERE  owner_id = $2 AND slug = $3
	`, body.Name, ownerID, slug)
	if err != nil || tag.RowsAffected() == 0 {
		errorResponse(w, http.StatusNotFound, "dashboard not found")
		return
	}

	jsonResponse(w, http.StatusOK, map[string]string{"status": "ok"})
}

// ── DELETE /me/dashboards/{slug} ──────────────────────────────────────────────

func (h *DashboardHandler) Delete(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	ownerID, ok := h.ownerID(r)
	if !ok {
		errorResponse(w, http.StatusUnauthorized, "authentication required")
		return
	}

	// Block deleting the last dashboard.
	var count int
	if err := h.db.QueryRow(r.Context(),
		`SELECT COUNT(*) FROM user_dashboards WHERE owner_id = $1`, ownerID,
	).Scan(&count); err != nil || count <= 1 {
		errorResponse(w, http.StatusConflict, "cannot delete the last dashboard")
		return
	}

	tag, err := h.db.Exec(r.Context(),
		`DELETE FROM user_dashboards WHERE owner_id = $1 AND slug = $2`,
		ownerID, slug,
	)
	if err != nil || tag.RowsAffected() == 0 {
		errorResponse(w, http.StatusNotFound, "dashboard not found")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// ── PATCH /me/dashboards/reorder ─────────────────────────────────────────────
// Body: { "ids": ["<uuid>", "<uuid>", ...] } — ordered list of dashboard IDs.

func (h *DashboardHandler) Reorder(w http.ResponseWriter, r *http.Request) {
	ownerID, ok := h.ownerID(r)
	if !ok {
		errorResponse(w, http.StatusUnauthorized, "authentication required")
		return
	}

	var body struct {
		IDs []string `json:"ids"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || len(body.IDs) == 0 {
		errorResponse(w, http.StatusBadRequest, "ids required")
		return
	}

	tx, err := h.db.Begin(r.Context())
	if err != nil {
		errorResponse(w, http.StatusInternalServerError, "tx failed")
		return
	}
	defer tx.Rollback(r.Context()) //nolint:errcheck

	for i, id := range body.IDs {
		tag, err := tx.Exec(r.Context(), `
			UPDATE user_dashboards SET position = $1
			WHERE id = $2::uuid AND owner_id = $3
		`, i, id, ownerID)
		if err != nil || tag.RowsAffected() == 0 {
			errorResponse(w, http.StatusBadRequest, "invalid dashboard id")
			return
		}
	}

	if err := tx.Commit(r.Context()); err != nil {
		errorResponse(w, http.StatusInternalServerError, "commit failed")
		return
	}

	jsonResponse(w, http.StatusOK, map[string]string{"status": "ok"})
}

// ── GET /me/dashboards/{slug} ─────────────────────────────────────────────────

func (h *DashboardHandler) Get(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	ownerID, ok := h.ownerID(r)
	if !ok {
		errorResponse(w, http.StatusUnauthorized, "authentication required")
		return
	}

	var d dashboardRow
	err := h.db.QueryRow(r.Context(), `
		SELECT id::text, slug, name, position, created_at::text
		FROM   user_dashboards
		WHERE  owner_id = $1 AND slug = $2
	`, ownerID, slug).Scan(&d.ID, &d.Slug, &d.Name, &d.Position, &d.CreatedAt)
	if err == pgx.ErrNoRows {
		errorResponse(w, http.StatusNotFound, "dashboard not found")
		return
	}
	if err != nil {
		errorResponse(w, http.StatusInternalServerError, "query failed")
		return
	}

	jsonResponse(w, http.StatusOK, d)
}

var nonAlphanumDash = regexp.MustCompile(`[^a-z0-9]+`)

func slugifyDashboard(name string) string {
	s := strings.ToLower(strings.TrimSpace(name))
	s = nonAlphanumDash.ReplaceAllString(s, "-")
	s = strings.Trim(s, "-")
	if s == "" {
		s = "dashboard"
	}
	if len(s) > 40 {
		s = s[:40]
	}
	return s
}
