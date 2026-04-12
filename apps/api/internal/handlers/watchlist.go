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

type watchlistItem struct {
	GaugeID   string  `json:"gauge_id"`
	ReachSlug *string `json:"reach_slug"`
}

// List handles GET /api/v1/watchlist
// Returns [{gauge_id, reach_slug}] for the current user.
func (h *WatchlistHandler) List(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		errorResponse(w, http.StatusUnauthorized, "authentication required")
		return
	}

	rows, err := h.db.Query(r.Context(),
		`SELECT gauge_id::text, reach_slug FROM user_watchlists WHERE user_id = $1 ORDER BY created_at`,
		userID,
	)
	if err != nil {
		errorResponse(w, http.StatusInternalServerError, "query failed")
		return
	}
	defer rows.Close()

	items := []watchlistItem{}
	for rows.Next() {
		var item watchlistItem
		if err := rows.Scan(&item.GaugeID, &item.ReachSlug); err == nil {
			items = append(items, item)
		}
	}

	jsonResponse(w, http.StatusOK, map[string]any{"items": items})
}

// Add handles POST /api/v1/watchlist
// Body: { "gauge_id": "<uuid>", "reach_slug": "<slug>" (optional) }
// Idempotent — re-adding the same gauge+reach pair is a no-op.
func (h *WatchlistHandler) Add(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		errorResponse(w, http.StatusUnauthorized, "authentication required")
		return
	}

	var body struct {
		GaugeID   string  `json:"gauge_id"`
		ReachSlug *string `json:"reach_slug"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.GaugeID == "" {
		errorResponse(w, http.StatusBadRequest, "gauge_id required")
		return
	}

	_, err := h.db.Exec(r.Context(),
		`INSERT INTO user_watchlists (user_id, gauge_id, reach_slug)
		 VALUES ($1, $2::uuid, $3)
		 ON CONFLICT (user_id, gauge_id, reach_slug) DO NOTHING`,
		userID, body.GaugeID, body.ReachSlug,
	)
	if err != nil {
		errorResponse(w, http.StatusInternalServerError, "insert failed")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// Remove handles DELETE /api/v1/watchlist/{gaugeId}?reach_slug=<slug>
// reach_slug is optional; omit to remove a standalone (no-reach) entry.
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

	reachSlug := r.URL.Query().Get("reach_slug")
	var reachSlugPtr *string
	if reachSlug != "" {
		reachSlugPtr = &reachSlug
	}

	_, err := h.db.Exec(r.Context(),
		`DELETE FROM user_watchlists
		 WHERE user_id = $1 AND gauge_id = $2::uuid AND reach_slug IS NOT DISTINCT FROM $3`,
		userID, gaugeID, reachSlugPtr,
	)
	if err != nil {
		errorResponse(w, http.StatusInternalServerError, "delete failed")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
