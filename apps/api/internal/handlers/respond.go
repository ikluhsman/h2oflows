package handlers

import (
	"encoding/json"
	"net/http"
)

// jsonResponse writes v as JSON with the given status code.
func jsonResponse(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// errorResponse writes a standard error JSON body.
func errorResponse(w http.ResponseWriter, status int, msg string) {
	jsonResponse(w, status, map[string]string{"error": msg})
}

// FeatureCollection is a GeoJSON FeatureCollection.
// MapLibre consumes this directly as a source — no frontend transformation needed.
type FeatureCollection struct {
	Type     string    `json:"type"`
	Features []Feature `json:"features"`
}

// Feature is a GeoJSON Feature. Geometry is any JSON-serialisable geometry
// (PointGeometry for gauge markers, rawGeometry for reach centerlines).
type Feature struct {
	Type       string         `json:"type"`
	Geometry   any            `json:"geometry"`
	Properties map[string]any `json:"properties"`
}

// PointGeometry is a GeoJSON Point. Coordinates are [longitude, latitude]
// per the GeoJSON spec (not lat/lng).
type PointGeometry struct {
	Type        string     `json:"type"`
	Coordinates [2]float64 `json:"coordinates"`
}

func newFeatureCollection(features []Feature) FeatureCollection {
	if features == nil {
		features = []Feature{} // never return null — MapLibre expects an array
	}
	return FeatureCollection{Type: "FeatureCollection", Features: features}
}
