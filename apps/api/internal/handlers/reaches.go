package handlers

import (
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"
)

// ReachHandler handles reach-related HTTP routes.
type ReachHandler struct {
	db *pgxpool.Pool
}

func NewReachHandler(db *pgxpool.Pool) *ReachHandler {
	return &ReachHandler{db: db}
}

// List handles GET /api/v1/reaches
// TODO: implement
func (h *ReachHandler) List(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotImplemented)
}

// Get handles GET /api/v1/reaches/{slug}
// TODO: implement
func (h *ReachHandler) Get(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotImplemented)
}

// GetConditions handles GET /api/v1/reaches/{slug}/conditions
// TODO: implement
func (h *ReachHandler) GetConditions(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotImplemented)
}

// GetHazards handles GET /api/v1/reaches/{slug}/hazards
// TODO: implement
func (h *ReachHandler) GetHazards(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotImplemented)
}
