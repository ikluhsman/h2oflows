package handlers

import (
	"crypto/rand"
	"encoding/base32"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ContributionHandler struct {
	db *pgxpool.Pool
}

func NewContributionHandler(db *pgxpool.Pool) *ContributionHandler {
	return &ContributionHandler{db: db}
}

// ---- Contributions (flow updates, hazard alerts, general notes) -------------

// CreateContribution handles POST /api/v1/reaches/{slug}/contributions
func (h *ContributionHandler) CreateContribution(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")

	var body struct {
		DeviceID             string     `json:"device_id"`
		ContributionType     string     `json:"contribution_type"`
		FlowImpression       *string    `json:"flow_impression"`
		Body                 *string    `json:"body"`
		ObservedAt           *time.Time `json:"observed_at"`
		ShareConsentH2oflows bool       `json:"share_consent_h2oflows"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		errorResponse(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	if body.ContributionType == "" {
		errorResponse(w, http.StatusBadRequest, "contribution_type is required")
		return
	}
	observedAt := time.Now()
	if body.ObservedAt != nil {
		observedAt = *body.ObservedAt
	}

	ctx := r.Context()

	// Resolve reach ID.
	var reachID string
	if err := h.db.QueryRow(ctx,
		`SELECT id FROM reaches WHERE slug = $1`, slug,
	).Scan(&reachID); err != nil {
		errorResponse(w, http.StatusNotFound, "reach not found")
		return
	}

	// Auto-stamp current CFS from primary gauge.
	var cfsScan *float64
	_ = h.db.QueryRow(ctx, `
		SELECT gr.value_cfs
		FROM reach_gauges rg
		JOIN gauge_readings gr ON gr.gauge_id = rg.gauge_id
		WHERE rg.reach_id = $1 AND rg.is_primary = TRUE
		ORDER BY gr.observed_at DESC
		LIMIT 1
	`, reachID).Scan(&cfsScan)

	var id string
	err := h.db.QueryRow(ctx, `
		INSERT INTO contributions
			(device_id, reach_id, contribution_type, flow_impression,
			 body, observed_at, cfs_at_time, share_consent_h2oflows)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
		RETURNING id
	`,
		nullableStr(body.DeviceID),
		reachID,
		body.ContributionType,
		body.FlowImpression,
		body.Body,
		observedAt,
		cfsScan,
		body.ShareConsentH2oflows,
	).Scan(&id)
	if err != nil {
		errorResponse(w, http.StatusInternalServerError, fmt.Sprintf("create contribution: %v", err))
		return
	}

	jsonResponse(w, http.StatusCreated, map[string]string{"id": id})
}

// ---- Trip Reports -----------------------------------------------------------

// CreateTripReport handles POST /api/v1/reaches/{slug}/trip-reports
func (h *ContributionHandler) CreateTripReport(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")

	var body struct {
		DeviceID             string     `json:"device_id"`
		Title                *string    `json:"title"`
		Body                 *string    `json:"body"`
		FlowImpression       *string    `json:"flow_impression"`
		ObservedAt           *time.Time `json:"observed_at"`
		ShareConsentH2oflows bool       `json:"share_consent_h2oflows"`
		Published            bool       `json:"published"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		errorResponse(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	observedAt := time.Now()
	if body.ObservedAt != nil {
		observedAt = *body.ObservedAt
	}

	ctx := r.Context()

	var reachID string
	if err := h.db.QueryRow(ctx,
		`SELECT id FROM reaches WHERE slug = $1`, slug,
	).Scan(&reachID); err != nil {
		errorResponse(w, http.StatusNotFound, "reach not found")
		return
	}

	// Auto-stamp current CFS.
	var cfsScan *float64
	_ = h.db.QueryRow(ctx, `
		SELECT gr.value_cfs
		FROM reach_gauges rg
		JOIN gauge_readings gr ON gr.gauge_id = rg.gauge_id
		WHERE rg.reach_id = $1 AND rg.is_primary = TRUE
		ORDER BY gr.observed_at DESC
		LIMIT 1
	`, reachID).Scan(&cfsScan)

	publicSlug := generateSlug()

	var id string
	err := h.db.QueryRow(ctx, `
		INSERT INTO trip_reports
			(device_id, reach_id, title, body, flow_impression,
			 observed_at, cfs_at_time, public_slug,
			 share_consent_h2oflows, published)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
		RETURNING id
	`,
		nullableStr(body.DeviceID),
		reachID,
		body.Title,
		body.Body,
		body.FlowImpression,
		observedAt,
		cfsScan,
		publicSlug,
		body.ShareConsentH2oflows,
		body.Published,
	).Scan(&id)
	if err != nil {
		errorResponse(w, http.StatusInternalServerError, fmt.Sprintf("create trip report: %v", err))
		return
	}

	jsonResponse(w, http.StatusCreated, map[string]string{"id": id, "public_slug": publicSlug})
}

// ListTripReports handles GET /api/v1/reaches/{slug}/trip-reports
func (h *ContributionHandler) ListTripReports(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	ctx := r.Context()

	var reachID string
	if err := h.db.QueryRow(ctx,
		`SELECT id FROM reaches WHERE slug = $1`, slug,
	).Scan(&reachID); err != nil {
		errorResponse(w, http.StatusNotFound, "reach not found")
		return
	}

	rows, err := h.db.Query(ctx, `
		SELECT
			tr.id,
			tr.public_slug,
			tr.title,
			tr.body,
			tr.flow_impression,
			tr.observed_at,
			tr.cfs_at_time,
			tr.created_at
		FROM trip_reports tr
		WHERE tr.reach_id = $1 AND tr.published = TRUE
		ORDER BY tr.observed_at DESC
		LIMIT 50
	`, reachID)
	if err != nil {
		errorResponse(w, http.StatusInternalServerError, "query failed")
		return
	}
	defer rows.Close()

	type row struct {
		ID             string     `json:"id"`
		PublicSlug     string     `json:"public_slug"`
		Title          *string    `json:"title"`
		Body           *string    `json:"body"`
		FlowImpression *string    `json:"flow_impression"`
		ObservedAt     time.Time  `json:"observed_at"`
		CFSAtTime      *float64   `json:"cfs_at_time"`
		CreatedAt      time.Time  `json:"created_at"`
	}

	var reports []row
	for rows.Next() {
		var rep row
		if err := rows.Scan(
			&rep.ID, &rep.PublicSlug, &rep.Title, &rep.Body,
			&rep.FlowImpression, &rep.ObservedAt, &rep.CFSAtTime, &rep.CreatedAt,
		); err != nil {
			errorResponse(w, http.StatusInternalServerError, "scan failed")
			return
		}
		reports = append(reports, rep)
	}
	if reports == nil {
		reports = []row{}
	}
	jsonResponse(w, http.StatusOK, reports)
}

// GetTripReport handles GET /api/v1/trip-reports/{slug}
func (h *ContributionHandler) GetTripReport(w http.ResponseWriter, r *http.Request) {
	publicSlug := chi.URLParam(r, "slug")
	ctx := r.Context()

	type detail struct {
		ID             string    `json:"id"`
		PublicSlug     string    `json:"public_slug"`
		Title          *string   `json:"title"`
		Body           *string   `json:"body"`
		FlowImpression *string   `json:"flow_impression"`
		ObservedAt     time.Time `json:"observed_at"`
		CFSAtTime      *float64  `json:"cfs_at_time"`
		CreatedAt      time.Time `json:"created_at"`
		ReachName      string    `json:"reach_name"`
		ReachSlug      string    `json:"reach_slug"`
	}

	var d detail
	err := h.db.QueryRow(ctx, `
		SELECT
			tr.id,
			tr.public_slug,
			tr.title,
			tr.body,
			tr.flow_impression,
			tr.observed_at,
			tr.cfs_at_time,
			tr.created_at,
			COALESCE(re.name, '') AS reach_name,
			COALESCE(re.slug, '') AS reach_slug
		FROM trip_reports tr
		JOIN reaches re ON re.id = tr.reach_id
		WHERE tr.public_slug = $1 AND tr.published = TRUE
	`, publicSlug).Scan(
		&d.ID, &d.PublicSlug, &d.Title, &d.Body, &d.FlowImpression,
		&d.ObservedAt, &d.CFSAtTime, &d.CreatedAt, &d.ReachName, &d.ReachSlug,
	)
	if err != nil {
		errorResponse(w, http.StatusNotFound, "trip report not found")
		return
	}

	jsonResponse(w, http.StatusOK, d)
}

// PatchTripReport handles PATCH /api/v1/trip-reports/{slug}
//
// The caller must supply the device_id that created the report.
func (h *ContributionHandler) PatchTripReport(w http.ResponseWriter, r *http.Request) {
	publicSlug := chi.URLParam(r, "slug")

	var body struct {
		DeviceID       string  `json:"device_id"`
		Title          *string `json:"title"`
		Body           *string `json:"body"`
		FlowImpression *string `json:"flow_impression"`
		Published      *bool   `json:"published"`
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
		UPDATE trip_reports
		SET
			title           = COALESCE($1, title),
			body            = COALESCE($2, body),
			flow_impression = COALESCE($3, flow_impression),
			published       = COALESCE($4, published),
			updated_at      = NOW()
		WHERE public_slug = $5 AND device_id = $6
	`, body.Title, body.Body, body.FlowImpression, body.Published,
		publicSlug, body.DeviceID)
	if err != nil {
		errorResponse(w, http.StatusInternalServerError, "update failed")
		return
	}
	if tag.RowsAffected() == 0 {
		errorResponse(w, http.StatusNotFound, "trip report not found")
		return
	}

	jsonResponse(w, http.StatusOK, map[string]string{"status": "ok"})
}

// DeleteTripReport handles DELETE /api/v1/trip-reports/{slug}
func (h *ContributionHandler) DeleteTripReport(w http.ResponseWriter, r *http.Request) {
	publicSlug := chi.URLParam(r, "slug")

	var body struct {
		DeviceID string `json:"device_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		errorResponse(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	if body.DeviceID == "" {
		errorResponse(w, http.StatusBadRequest, "device_id is required")
		return
	}

	tag, err := h.db.Exec(r.Context(),
		`DELETE FROM trip_reports WHERE public_slug = $1 AND device_id = $2`,
		publicSlug, body.DeviceID,
	)
	if err != nil {
		errorResponse(w, http.StatusInternalServerError, "delete failed")
		return
	}
	if tag.RowsAffected() == 0 {
		errorResponse(w, http.StatusNotFound, "trip report not found")
		return
	}

	jsonResponse(w, http.StatusOK, map[string]string{"status": "ok"})
}

// ---- Proximity Events -------------------------------------------------------

// CreateProximityEvent handles POST /api/v1/proximity-events
func (h *ContributionHandler) CreateProximityEvent(w http.ResponseWriter, r *http.Request) {
	var body struct {
		DeviceID   string     `json:"device_id"`
		ReachSlug  string     `json:"reach_slug"`
		EventType  string     `json:"event_type"`
		Lat        *float64   `json:"lat"`
		Lng        *float64   `json:"lng"`
		DetectedAt *time.Time `json:"detected_at"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		errorResponse(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	if body.DeviceID == "" {
		errorResponse(w, http.StatusBadRequest, "device_id is required")
		return
	}
	if body.ReachSlug == "" {
		errorResponse(w, http.StatusBadRequest, "reach_slug is required")
		return
	}
	detectedAt := time.Now()
	if body.DetectedAt != nil {
		detectedAt = *body.DetectedAt
	}

	ctx := r.Context()

	var reachID string
	if err := h.db.QueryRow(ctx,
		`SELECT id FROM reaches WHERE slug = $1`, body.ReachSlug,
	).Scan(&reachID); err != nil {
		errorResponse(w, http.StatusNotFound, "reach not found")
		return
	}

	var locationExpr *string
	if body.Lat != nil && body.Lng != nil {
		s := fmt.Sprintf("ST_SetSRID(ST_MakePoint(%f, %f), 4326)", *body.Lng, *body.Lat)
		locationExpr = &s
	}

	var id string
	var err error
	if locationExpr != nil {
		err = h.db.QueryRow(ctx, fmt.Sprintf(`
			INSERT INTO proximity_events
				(device_id, reach_id, event_type, location, detected_at)
			VALUES ($1,$2,$3,%s,$4)
			RETURNING id
		`, *locationExpr),
			body.DeviceID, reachID, body.EventType, detectedAt,
		).Scan(&id)
	} else {
		err = h.db.QueryRow(ctx, `
			INSERT INTO proximity_events
				(device_id, reach_id, event_type, detected_at)
			VALUES ($1,$2,$3,$4)
			RETURNING id
		`, body.DeviceID, reachID, body.EventType, detectedAt,
		).Scan(&id)
	}
	if err != nil {
		errorResponse(w, http.StatusInternalServerError, fmt.Sprintf("create proximity event: %v", err))
		return
	}

	jsonResponse(w, http.StatusCreated, map[string]string{"id": id})
}

// ---- Helpers ----------------------------------------------------------------

// generateSlug returns an 8-character URL-safe random string for public trip report slugs.
func generateSlug() string {
	b := make([]byte, 5)
	if _, err := rand.Read(b); err != nil {
		panic(fmt.Sprintf("generateSlug: crypto/rand: %v", err))
	}
	return strings.ToLower(base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(b))
}
