package handlers

import (
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"
)

// GaugeHandler handles gauge-related HTTP routes.
type GaugeHandler struct {
	db *pgxpool.Pool
}

func NewGaugeHandler(db *pgxpool.Pool) *GaugeHandler {
	return &GaugeHandler{db: db}
}

// Search handles GET /api/v1/gauges/search
// Supports ?q=, ?bbox=, ?lat=&lon=&radius_mi=, ?source=, ?limit=
// TODO: implement
func (h *GaugeHandler) Search(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotImplemented)
}

// GetReadings handles GET /api/v1/gauges/{id}/readings
// TODO: implement
func (h *GaugeHandler) GetReadings(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotImplemented)
}

// GetFlowRanges handles GET /api/v1/gauges/{id}/flow-ranges
// TODO: implement
func (h *GaugeHandler) GetFlowRanges(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotImplemented)
}
