package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/h2oflow/h2oflow/apps/api/internal/auth"
	"github.com/jackc/pgx/v5/pgxpool"
)

// WatchlistHandler handles /api/v1/watchlist routes.
// All routes require an authenticated user (auth.Required middleware).
type WatchlistHandler struct {
	db *pgxpool.Pool
}

func NewWatchlistHandler(db *pgxpool.Pool) *WatchlistHandler {
	return &WatchlistHandler{db: db}
}

// List handles GET /api/v1/watchlist
// Returns the gauge IDs saved by the current user.
func (h *WatchlistHandler) List(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		errorResponse(w, http.StatusUnauthorized, "authentication required")
		return
	}

	rows, err := h.db.Query(r.Context(),
		`SELECT gauge_id::text FROM user_watchlists WHERE user_id = $1 ORDER BY created_at`,
		userID,
	)
	if err != nil {
		errorResponse(w, http.StatusInternalServerError, "query failed")
		return
	}
	defer rows.Close()

	ids := []string{}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err == nil {
			ids = append(ids, id)
		}
	}

	jsonResponse(w, http.StatusOK, map[string]any{"gauge_ids": ids})
}

// Add handles POST /api/v1/watchlist
// Body: { "gauge_id": "<uuid>" }
// Idempotent — re-adding a gauge already on the list is a no-op.
func (h *WatchlistHandler) Add(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		errorResponse(w, http.StatusUnauthorized, "authentication required")
		return
	}

	var body struct {
		GaugeID string `json:"gauge_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.GaugeID == "" {
		errorResponse(w, http.StatusBadRequest, "gauge_id required")
		return
	}

	_, err := h.db.Exec(r.Context(),
		`INSERT INTO user_watchlists (user_id, gauge_id)
		 VALUES ($1, $2::uuid)
		 ON CONFLICT (user_id, gauge_id) DO NOTHING`,
		userID, body.GaugeID,
	)
	if err != nil {
		errorResponse(w, http.StatusInternalServerError, "insert failed")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// Remove handles DELETE /api/v1/watchlist/{gaugeId}
func (h *WatchlistHandler) Remove(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		errorResponse(w, http.StatusUnauthorized, "authentication required")
		return
	}

	gaugeID := chi.URLParam(r, "gaugeId")
	if gaugeID == "" {
		errorResponse(w, http.StatusBadRequest, "gaugeId required")
		return
	}

	_, err := h.db.Exec(r.Context(),
		`DELETE FROM user_watchlists WHERE user_id = $1 AND gauge_id = $2::uuid`,
		userID, gaugeID,
	)
	if err != nil {
		errorResponse(w, http.StatusInternalServerError, "delete failed")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
