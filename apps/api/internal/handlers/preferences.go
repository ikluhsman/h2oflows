package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/h2oflow/h2oflow/apps/api/internal/auth"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PreferencesHandler handles /me/preferences routes.
type PreferencesHandler struct {
	db            *pgxpool.Pool
	devFallbackID string
}

func NewPreferencesHandler(db *pgxpool.Pool, devFallbackID string) *PreferencesHandler {
	return &PreferencesHandler{db: db, devFallbackID: devFallbackID}
}

func (h *PreferencesHandler) ownerID(r *http.Request) (string, bool) {
	if id, ok := auth.UserIDFromContext(r.Context()); ok {
		return id, true
	}
	if h.devFallbackID != "" {
		return h.devFallbackID, true
	}
	return "", false
}

// ── GET /me/preferences ───────────────────────────────────────────────────────

func (h *PreferencesHandler) Get(w http.ResponseWriter, r *http.Request) {
	ownerID, ok := h.ownerID(r)
	if !ok {
		errorResponse(w, http.StatusUnauthorized, "authentication required")
		return
	}

	var awBandMapping *json.RawMessage
	err := h.db.QueryRow(r.Context(),
		`SELECT aw_band_mapping FROM user_preferences WHERE owner_id = $1`,
		ownerID,
	).Scan(&awBandMapping)

	if err != nil {
		// No row = return empty prefs
		jsonResponse(w, http.StatusOK, map[string]any{"aw_band_mapping": nil})
		return
	}

	jsonResponse(w, http.StatusOK, map[string]any{"aw_band_mapping": awBandMapping})
}

// ── PATCH /me/preferences ─────────────────────────────────────────────────────

func (h *PreferencesHandler) Update(w http.ResponseWriter, r *http.Request) {
	ownerID, ok := h.ownerID(r)
	if !ok {
		errorResponse(w, http.StatusUnauthorized, "authentication required")
		return
	}

	var body struct {
		AWBandMapping *json.RawMessage `json:"aw_band_mapping"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		errorResponse(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	_, err := h.db.Exec(r.Context(), `
		INSERT INTO user_preferences (owner_id, aw_band_mapping, updated_at)
		VALUES ($1, $2, NOW())
		ON CONFLICT (owner_id) DO UPDATE
		SET aw_band_mapping = COALESCE($2, user_preferences.aw_band_mapping),
		    updated_at = NOW()
	`, ownerID, body.AWBandMapping)
	if err != nil {
		errorResponse(w, http.StatusInternalServerError, "save failed")
		return
	}

	jsonResponse(w, http.StatusOK, map[string]string{"status": "ok"})
}
