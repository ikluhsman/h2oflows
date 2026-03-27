package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type TripHandler struct {
	db *pgxpool.Pool
}

func NewTripHandler(db *pgxpool.Pool) *TripHandler {
	return &TripHandler{db: db}
}

// Create handles POST /api/v1/trips
//
// Accepts a completed trip with its raw GPS track from the device.
// The device may have been offline during the run — this endpoint is
// designed to accept uploads at any point after the trip ends.
//
// After storing the raw points the server builds a simplified PostGIS
// linestring for map rendering and computes distance in miles.
func (h *TripHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req tripCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		errorResponse(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	if err := req.validate(); err != nil {
		errorResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	ctx := r.Context()

	// Insert trip record.
	var tripID string
	err := h.db.QueryRow(ctx, `
		INSERT INTO trips
			(gauge_id, reach_id, start_cfs, end_cfs,
			 started_at, ended_at, notes, device_id, share_consent)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
		RETURNING id
	`,
		req.GaugeID,
		req.ReachID,
		req.StartCFS,
		req.EndCFS,
		req.StartedAt,
		req.EndedAt,
		nullableStr(req.Notes),
		nullableStr(req.DeviceID),
		req.ShareConsent,
	).Scan(&tripID)
	if err != nil {
		errorResponse(w, http.StatusInternalServerError, "failed to create trip")
		return
	}

	// Bulk-insert raw track points.
	if len(req.TrackPoints) > 0 {
		batch := &pgx.Batch{}
		for _, p := range req.TrackPoints {
			batch.Queue(`
				INSERT INTO trip_track_points
					(trip_id, timestamp, lat, lng, accuracy_m, altitude_m, speed_mps, heading)
				VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
				ON CONFLICT (trip_id, timestamp) DO NOTHING
			`,
				tripID, p.Timestamp, p.Lat, p.Lng,
				p.AccuracyM, p.AltitudeM, p.SpeedMPS, p.Heading,
			)
		}
		br := h.db.SendBatch(ctx, batch)
		for range req.TrackPoints {
			if _, err := br.Exec(); err != nil {
				fmt.Printf("trip %s: insert track point: %v\n", tripID, err)
			}
		}
		br.Close()

		// Build simplified linestring + distance from stored points.
		_, err := h.db.Exec(ctx, `
			WITH pts AS (
				SELECT ST_MakePoint(lng::float8, lat::float8) AS geom
				FROM trip_track_points
				WHERE trip_id = $1
				ORDER BY timestamp
			),
			line AS (
				SELECT ST_MakeLine(array_agg(geom ORDER BY geom)) AS geom FROM pts
			)
			UPDATE trips SET
				track       = ST_SimplifyPreserveTopology(line.geom, 0.0001)::geography,
				distance_mi = ST_Length(line.geom::geography) / 1609.34
			FROM line
			WHERE trips.id = $1
		`, tripID)
		if err != nil {
			// Non-fatal — geometry can be rebuilt later.
			fmt.Printf("trip %s: build geometry: %v\n", tripID, err)
		}
	}

	jsonResponse(w, http.StatusCreated, map[string]string{"id": tripID})
}

// ---- Request types ----------------------------------------------------------

type tripCreateRequest struct {
	GaugeID      *string      `json:"gauge_id"`
	ReachID      *string      `json:"reach_id"`
	StartCFS     *float64     `json:"start_cfs"`
	EndCFS       *float64     `json:"end_cfs"`
	StartedAt    time.Time    `json:"started_at"`
	EndedAt      *time.Time   `json:"ended_at"`
	Notes        string       `json:"notes"`
	DeviceID     string       `json:"device_id"`
	ShareConsent *bool        `json:"share_consent"`
	TrackPoints  []trackPoint `json:"track_points"`
}

type trackPoint struct {
	Timestamp time.Time `json:"timestamp"`
	Lat       float64   `json:"lat"`
	Lng       float64   `json:"lng"`
	AccuracyM *float64  `json:"accuracy_m"`
	AltitudeM *float64  `json:"altitude_m"`
	SpeedMPS  *float64  `json:"speed_mps"`
	Heading   *float64  `json:"heading"`
}

func (req *tripCreateRequest) validate() error {
	if req.StartedAt.IsZero() {
		return fmt.Errorf("started_at is required")
	}
	if req.GaugeID == nil && req.ReachID == nil {
		return fmt.Errorf("at least one of gauge_id or reach_id is required")
	}
	for i, p := range req.TrackPoints {
		if p.Lat < -90 || p.Lat > 90 {
			return fmt.Errorf("track_points[%d]: invalid lat %v", i, p.Lat)
		}
		if p.Lng < -180 || p.Lng > 180 {
			return fmt.Errorf("track_points[%d]: invalid lng %v", i, p.Lng)
		}
	}
	return nil
}

func nullableStr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
