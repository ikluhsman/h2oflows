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
	Kind          string  `json:"kind"` // "gauge" | "custom_gauge"
	GaugeID       *string `json:"gauge_id"`
	CustomGaugeID *string `json:"custom_gauge_id"`
	ReachSlug     *string `json:"reach_slug"`
}

// List handles GET /api/v1/watchlist?dashboard_id=<uuid>
// Returns items for a specific dashboard (or all items if no dashboard_id given).
func (h *WatchlistHandler) List(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		errorResponse(w, http.StatusUnauthorized, "authentication required")
		return
	}

	dashboardID := r.URL.Query().Get("dashboard_id")

	var (
		pgrows interface {
			Next() bool
			Scan(...any) error
			Close()
		}
		err error
	)

	if dashboardID != "" {
		pgrows, err = h.db.Query(r.Context(), `
			SELECT gauge_id::text, custom_gauge_id::text, reach_slug
			FROM   user_watchlists
			WHERE  user_id = $1 AND dashboard_id = $2::uuid
			ORDER  BY created_at
		`, userID, dashboardID)
	} else {
		pgrows, err = h.db.Query(r.Context(), `
			SELECT gauge_id::text, custom_gauge_id::text, reach_slug
			FROM   user_watchlists
			WHERE  user_id = $1
			ORDER  BY created_at
		`, userID)
	}
	if err != nil {
		errorResponse(w, http.StatusInternalServerError, "query failed")
		return
	}
	defer pgrows.Close()

	items := []watchlistItem{}
	for pgrows.Next() {
		var item watchlistItem
		if err := pgrows.Scan(&item.GaugeID, &item.CustomGaugeID, &item.ReachSlug); err == nil {
			if item.CustomGaugeID != nil {
				item.Kind = "custom_gauge"
			} else {
				item.Kind = "gauge"
			}
			items = append(items, item)
		}
	}

	jsonResponse(w, http.StatusOK, map[string]any{"items": items})
}

// Add handles POST /api/v1/watchlist
// Body: { "gauge_id": "<uuid>", "reach_slug": "<slug>", "dashboard_id": "<uuid>" }
//    or { "custom_gauge_id": "<uuid>", "reach_slug": "<slug>", "dashboard_id": "<uuid>" }
// If dashboard_id is omitted, defaults to the user's first dashboard (position=0).
// Idempotent — re-adding the same pair is a no-op.
func (h *WatchlistHandler) Add(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		errorResponse(w, http.StatusUnauthorized, "authentication required")
		return
	}

	var body struct {
		GaugeID       *string `json:"gauge_id"`
		CustomGaugeID *string `json:"custom_gauge_id"`
		ReachSlug     *string `json:"reach_slug"`
		DashboardID   *string `json:"dashboard_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		errorResponse(w, http.StatusBadRequest, "invalid body")
		return
	}
	if body.GaugeID == nil && body.CustomGaugeID == nil {
		errorResponse(w, http.StatusBadRequest, "gauge_id or custom_gauge_id required")
		return
	}
	if body.GaugeID != nil && body.CustomGaugeID != nil {
		errorResponse(w, http.StatusBadRequest, "only one of gauge_id or custom_gauge_id allowed")
		return
	}

	// Resolve dashboard_id: use provided or default to first dashboard.
	dashboardID := body.DashboardID
	if dashboardID == nil {
		var dbID string
		err := h.db.QueryRow(r.Context(), `
			SELECT id::text FROM user_dashboards
			WHERE owner_id = $1 ORDER BY position, created_at LIMIT 1
		`, userID).Scan(&dbID)
		if err != nil {
			// No dashboard yet — auto-create one.
			err = h.db.QueryRow(r.Context(), `
				INSERT INTO user_dashboards (owner_id, slug, name, position)
				VALUES ($1, 'default', 'My Dashboard', 0)
				ON CONFLICT (owner_id, slug) DO UPDATE SET name = EXCLUDED.name
				RETURNING id::text
			`, userID).Scan(&dbID)
			if err != nil {
				errorResponse(w, http.StatusInternalServerError, "dashboard resolution failed")
				return
			}
		}
		dashboardID = &dbID
	}

	var err error
	if body.GaugeID != nil {
		_, err = h.db.Exec(r.Context(), `
			INSERT INTO user_watchlists (user_id, gauge_id, reach_slug, dashboard_id)
			VALUES ($1, $2::uuid, $3, $4::uuid)
			ON CONFLICT DO NOTHING
		`, userID, body.GaugeID, body.ReachSlug, dashboardID)
	} else {
		_, err = h.db.Exec(r.Context(), `
			INSERT INTO user_watchlists (user_id, custom_gauge_id, reach_slug, dashboard_id)
			VALUES ($1, $2::uuid, $3, $4::uuid)
			ON CONFLICT DO NOTHING
		`, userID, body.CustomGaugeID, body.ReachSlug, dashboardID)
	}
	if err != nil {
		errorResponse(w, http.StatusInternalServerError, "insert failed")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// Remove handles DELETE /api/v1/watchlist/{id}?kind=gauge|custom_gauge&reach_slug=<slug>
// kind defaults to "gauge". reach_slug is optional.
func (h *WatchlistHandler) Remove(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		errorResponse(w, http.StatusUnauthorized, "authentication required")
		return
	}

	id := chi.URLParam(r, "gaugeId")
	if id == "" {
		errorResponse(w, http.StatusBadRequest, "id required")
		return
	}

	kind := r.URL.Query().Get("kind")
	if kind == "" {
		kind = "gauge"
	}

	reachSlug := r.URL.Query().Get("reach_slug")
	var reachSlugPtr *string
	if reachSlug != "" {
		reachSlugPtr = &reachSlug
	}

	var err error
	if kind == "custom_gauge" {
		_, err = h.db.Exec(r.Context(), `
			DELETE FROM user_watchlists
			WHERE user_id = $1 AND custom_gauge_id = $2::uuid
			  AND reach_slug IS NOT DISTINCT FROM $3
		`, userID, id, reachSlugPtr)
	} else {
		_, err = h.db.Exec(r.Context(), `
			DELETE FROM user_watchlists
			WHERE user_id = $1 AND gauge_id = $2::uuid
			  AND reach_slug IS NOT DISTINCT FROM $3
		`, userID, id, reachSlugPtr)
	}
	if err != nil {
		errorResponse(w, http.StatusInternalServerError, "delete failed")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
