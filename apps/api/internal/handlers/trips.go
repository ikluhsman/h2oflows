package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
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

// List handles GET /api/v1/trips?device_id=xxx
//
// Returns the caller's trips ordered newest-first.
// Joins gauges and reaches so the response includes human-readable names.
func (h *TripHandler) List(w http.ResponseWriter, r *http.Request) {
	deviceID := r.URL.Query().Get("device_id")
	if deviceID == "" {
		errorResponse(w, http.StatusBadRequest, "device_id is required")
		return
	}

	rows, err := h.db.Query(r.Context(), `
		SELECT
			t.id,
			t.started_at,
			t.ended_at,
			t.duration_min,
			t.start_cfs,
			t.end_cfs,
			t.distance_mi,
			t.notes,
			t.share_consent,
			COALESCE(re.name, '') AS reach_name,
			COALESCE(re.slug, '') AS reach_slug,
			COALESCE(g.name,  '') AS gauge_name
		FROM trips t
		LEFT JOIN gauges  g  ON g.id  = t.gauge_id
		LEFT JOIN reaches re ON re.id = t.reach_id
		WHERE t.device_id = $1
		ORDER BY t.started_at DESC
		LIMIT 100
	`, deviceID)
	if err != nil {
		errorResponse(w, http.StatusInternalServerError, "query failed")
		return
	}
	defer rows.Close()

	type tripRow struct {
		ID           string     `json:"id"`
		StartedAt    time.Time  `json:"started_at"`
		EndedAt      *time.Time `json:"ended_at"`
		DurationMin  *int32     `json:"duration_min"`
		StartCFS     *float64   `json:"start_cfs"`
		EndCFS       *float64   `json:"end_cfs"`
		DistanceMi   *float64   `json:"distance_mi"`
		Notes        *string    `json:"notes"`
		ShareConsent *bool      `json:"share_consent"`
		ReachName    string     `json:"reach_name"`
		ReachSlug    string     `json:"reach_slug"`
		GaugeName    string     `json:"gauge_name"`
	}

	var trips []tripRow
	for rows.Next() {
		var t tripRow
		if err := rows.Scan(
			&t.ID, &t.StartedAt, &t.EndedAt, &t.DurationMin,
			&t.StartCFS, &t.EndCFS, &t.DistanceMi, &t.Notes,
			&t.ShareConsent, &t.ReachName, &t.ReachSlug, &t.GaugeName,
		); err != nil {
			errorResponse(w, http.StatusInternalServerError, "scan failed")
			return
		}
		trips = append(trips, t)
	}
	if trips == nil {
		trips = []tripRow{}
	}
	jsonResponse(w, http.StatusOK, trips)
}

// Get handles GET /api/v1/trips/{id}?device_id=xxx
//
// Returns the full trip detail including the GPS track as a GeoJSON LineString.
func (h *TripHandler) Get(w http.ResponseWriter, r *http.Request) {
	id       := chi.URLParam(r, "id")
	deviceID := r.URL.Query().Get("device_id")
	if deviceID == "" {
		errorResponse(w, http.StatusBadRequest, "device_id is required")
		return
	}

	type tripDetail struct {
		ID           string          `json:"id"`
		StartedAt    time.Time       `json:"started_at"`
		EndedAt      *time.Time      `json:"ended_at"`
		DurationMin  *int32          `json:"duration_min"`
		StartCFS     *float64        `json:"start_cfs"`
		EndCFS       *float64        `json:"end_cfs"`
		DistanceMi   *float64        `json:"distance_mi"`
		Notes        *string         `json:"notes"`
		ShareConsent *bool           `json:"share_consent"`
		ReachName    string          `json:"reach_name"`
		ReachSlug    string          `json:"reach_slug"`
		GaugeName    string          `json:"gauge_name"`
		Track        json.RawMessage `json:"track"` // GeoJSON LineString or null
		PointCount   int             `json:"point_count"`
	}

	var t tripDetail
	var trackJSON []byte
	err := h.db.QueryRow(r.Context(), `
		SELECT
			t.id,
			t.started_at,
			t.ended_at,
			t.duration_min,
			t.start_cfs,
			t.end_cfs,
			t.distance_mi,
			t.notes,
			t.share_consent,
			COALESCE(re.name, '') AS reach_name,
			COALESCE(re.slug,  '') AS reach_slug,
			COALESCE(g.name,   '') AS gauge_name,
			CASE WHEN t.track IS NOT NULL THEN ST_AsGeoJSON(t.track)::text ELSE NULL END AS track_geojson,
			(SELECT COUNT(*) FROM trip_track_points WHERE trip_id = t.id) AS point_count
		FROM trips t
		LEFT JOIN gauges  g  ON g.id  = t.gauge_id
		LEFT JOIN reaches re ON re.id = t.reach_id
		WHERE t.id = $1 AND t.device_id = $2
	`, id, deviceID).Scan(
		&t.ID, &t.StartedAt, &t.EndedAt, &t.DurationMin,
		&t.StartCFS, &t.EndCFS, &t.DistanceMi, &t.Notes,
		&t.ShareConsent, &t.ReachName, &t.ReachSlug, &t.GaugeName,
		&trackJSON, &t.PointCount,
	)
	if err != nil {
		errorResponse(w, http.StatusNotFound, "trip not found")
		return
	}

	if trackJSON != nil {
		t.Track = json.RawMessage(trackJSON)
	} else {
		t.Track = json.RawMessage("null")
	}

	jsonResponse(w, http.StatusOK, t)
}

// Patch handles PATCH /api/v1/trips/{id}
//
// Allows the device that recorded the trip to update notes and share_consent.
func (h *TripHandler) Patch(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var body struct {
		DeviceID     string  `json:"device_id"`
		Notes        *string `json:"notes"`
		ShareConsent *bool   `json:"share_consent"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		errorResponse(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	if body.DeviceID == "" {
		errorResponse(w, http.StatusBadRequest, "device_id is required")
		return
	}

	tag, err := h.db.Exec(r.Context(), `
		UPDATE trips
		SET
			notes         = COALESCE($1, notes),
			share_consent = COALESCE($2, share_consent)
		WHERE id = $3 AND device_id = $4
	`, body.Notes, body.ShareConsent, id, body.DeviceID)
	if err != nil {
		errorResponse(w, http.StatusInternalServerError, "update failed")
		return
	}
	if tag.RowsAffected() == 0 {
		errorResponse(w, http.StatusNotFound, "trip not found")
		return
	}

	jsonResponse(w, http.StatusOK, map[string]string{"status": "ok"})
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
