package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/h2oflow/h2oflow/apps/api/internal/auth"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ReportHandler handles reports routes.
type ReportHandler struct {
	db            *pgxpool.Pool
	devFallbackID string
	rl            *reportRateLimiter
}

func NewReportHandler(db *pgxpool.Pool, devFallbackID string) *ReportHandler {
	return &ReportHandler{
		db:            db,
		devFallbackID: devFallbackID,
		rl:            newReportRateLimiter(),
	}
}

func (h *ReportHandler) ownerID(r *http.Request) (string, bool) {
	if id, ok := auth.UserIDFromContext(r.Context()); ok {
		return id, true
	}
	if h.devFallbackID != "" {
		return h.devFallbackID, true
	}
	return "", false
}

// ── Rate limiter ──────────────────────────────────────────────────────────────

type reportRateLimiter struct {
	mu      sync.Mutex
	windows map[string][]time.Time
}

func newReportRateLimiter() *reportRateLimiter {
	return &reportRateLimiter{windows: make(map[string][]time.Time)}
}

func (rl *reportRateLimiter) allow(ownerID string, limit int, window time.Duration) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	cutoff := time.Now().Add(-window)
	prev := rl.windows[ownerID]
	valid := prev[:0]
	for _, t := range prev {
		if t.After(cutoff) {
			valid = append(valid, t)
		}
	}
	if len(valid) >= limit {
		rl.windows[ownerID] = valid
		return false
	}
	rl.windows[ownerID] = append(valid, time.Now())
	return true
}

// ── Profile helpers ───────────────────────────────────────────────────────────

var nonAlnumRe = regexp.MustCompile(`[^a-z0-9]+`)

func handleFromEmail(email string) string {
	local := email
	if i := strings.Index(email, "@"); i > 0 {
		local = email[:i]
	}
	h := nonAlnumRe.ReplaceAllString(strings.ToLower(local), "-")
	h = strings.Trim(h, "-")
	if len(h) > 40 {
		h = h[:40]
	}
	if h == "" {
		h = "paddler"
	}
	return h
}

// ensureProfile returns the handle for ownerID, lazily creating a row derived
// from the user's email on first call.
func (h *ReportHandler) ensureProfile(ctx context.Context, ownerID, email string) (string, error) {
	var handle string
	err := h.db.QueryRow(ctx,
		`SELECT handle FROM user_profiles WHERE owner_id = $1`, ownerID,
	).Scan(&handle)
	if err == nil {
		return handle, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return "", fmt.Errorf("ensureProfile lookup: %w", err)
	}

	base := handleFromEmail(email)
	for i := 0; i < 20; i++ {
		candidate := base
		if i > 0 {
			candidate = fmt.Sprintf("%s-%d", base, i+1)
		}
		_, insErr := h.db.Exec(ctx,
			`INSERT INTO user_profiles (owner_id, handle) VALUES ($1, $2)`,
			ownerID, candidate,
		)
		if insErr == nil {
			return candidate, nil
		}
	}
	return "", fmt.Errorf("ensureProfile: could not assign unique handle for %s", ownerID)
}

// ── Flow band helpers ─────────────────────────────────────────────────────────

type flowRangeRow struct {
	label    string
	minValue *float64
	maxValue *float64
}

func computeFlowBand(cfs float64, ranges []flowRangeRow) *string {
	var runningMin, runningMax *float64
	for _, r := range ranges {
		if r.label == "running" {
			runningMin = r.minValue
			runningMax = r.maxValue
		}
	}
	if runningMin == nil && runningMax == nil {
		return nil
	}
	var band string
	switch {
	case runningMin != nil && cfs <= *runningMin:
		band = "low"
	case runningMax != nil && cfs >= *runningMax:
		band = "high"
	default:
		band = "running"
	}
	return &band
}

// stampCFSAndBand queries the gauge reading nearest to `at` for the reach's
// primary gauge, then derives the flow band from the reach's flow_ranges.
func (h *ReportHandler) stampCFSAndBand(ctx context.Context, reachID string, at time.Time) (*float64, *string) {
	var cfs float64
	err := h.db.QueryRow(ctx, `
		SELECT gr.value_cfs
		FROM reach_gauges rg
		JOIN gauge_readings gr ON gr.gauge_id = rg.gauge_id
		WHERE rg.reach_id = $1 AND rg.is_primary = TRUE
		ORDER BY ABS(EXTRACT(EPOCH FROM (gr.observed_at - $2::TIMESTAMPTZ))) ASC
		LIMIT 1
	`, reachID, at).Scan(&cfs)
	if err != nil {
		return nil, nil
	}

	frRows, err := h.db.Query(ctx,
		`SELECT label, min_value, max_value FROM flow_ranges WHERE reach_id = $1`, reachID,
	)
	if err != nil {
		return &cfs, nil
	}
	defer frRows.Close()

	var ranges []flowRangeRow
	for frRows.Next() {
		var fr flowRangeRow
		if frRows.Scan(&fr.label, &fr.minValue, &fr.maxValue) == nil {
			ranges = append(ranges, fr)
		}
	}

	band := computeFlowBand(cfs, ranges)
	return &cfs, band
}

// reportObservedAt converts report_date + optional report_time to UTC.
// Defaults to noon UTC when time is absent.
func reportObservedAt(date, timeStr string) time.Time {
	if timeStr != "" {
		t, err := time.Parse("2006-01-02 15:04", date+" "+timeStr)
		if err == nil {
			return t.UTC()
		}
	}
	d, err := time.Parse("2006-01-02", date)
	if err != nil {
		return time.Now().UTC()
	}
	return time.Date(d.Year(), d.Month(), d.Day(), 12, 0, 0, 0, time.UTC)
}

// uniqueSlug appends -2, -3, … until the (ownerID, slug) pair is free.
func (h *ReportHandler) uniqueSlug(ctx context.Context, ownerID, base string) string {
	candidate := base
	for i := 2; i < 100; i++ {
		var exists bool
		h.db.QueryRow(ctx,
			`SELECT EXISTS(SELECT 1 FROM reports WHERE owner_id=$1 AND slug=$2)`,
			ownerID, candidate,
		).Scan(&exists)
		if !exists {
			return candidate
		}
		candidate = fmt.Sprintf("%s-%d", base, i)
	}
	return fmt.Sprintf("%s-%d", base, time.Now().UnixMilli())
}

// ── POST /reaches/{slug}/reports ─────────────────────────────────────────────

func (h *ReportHandler) Create(w http.ResponseWriter, r *http.Request) {
	reachSlug := chi.URLParam(r, "slug")

	ownerID, ok := h.ownerID(r)
	if !ok {
		errorResponse(w, http.StatusUnauthorized, "authentication required")
		return
	}
	if !h.rl.allow(ownerID, 5, time.Hour) {
		errorResponse(w, http.StatusTooManyRequests, "rate limit: 5 reports per hour")
		return
	}

	var body struct {
		Name          string  `json:"name"`
		ReportDate    string  `json:"report_date"`
		ReportTime    string  `json:"report_time"`
		Content       string  `json:"content"`
		HazardWarning *string `json:"hazard_warning"`
		Paddled       bool    `json:"paddled"`
		Slug          string  `json:"slug"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		errorResponse(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	if body.Name == "" {
		errorResponse(w, http.StatusBadRequest, "name is required")
		return
	}
	if body.ReportDate == "" {
		errorResponse(w, http.StatusBadRequest, "report_date is required")
		return
	}
	if body.Content == "" {
		errorResponse(w, http.StatusBadRequest, "content is required")
		return
	}

	ctx := r.Context()

	var reachID string
	if err := h.db.QueryRow(ctx,
		`SELECT id FROM reaches WHERE slug = $1`, reachSlug,
	).Scan(&reachID); err != nil {
		errorResponse(w, http.StatusNotFound, "reach not found")
		return
	}

	email, _ := auth.EmailFromContext(ctx)
	handle, err := h.ensureProfile(ctx, ownerID, email)
	if err != nil {
		errorResponse(w, http.StatusInternalServerError, "could not assign user profile")
		return
	}

	reportSlug := body.Slug
	if reportSlug == "" {
		reportSlug = reachSlug + "-" + body.ReportDate
	}
	reportSlug = h.uniqueSlug(ctx, ownerID, reportSlug)

	at := reportObservedAt(body.ReportDate, body.ReportTime)
	cfs, band := h.stampCFSAndBand(ctx, reachID, at)

	var reportTimeVal *string
	if body.ReportTime != "" {
		reportTimeVal = &body.ReportTime
	}

	var id string
	err = h.db.QueryRow(ctx, `
		INSERT INTO reports
			(owner_id, slug, reach_id, name, report_date, report_time,
			 content, hazard_warning, paddled, flow_cfs, flow_band)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
		RETURNING id
	`,
		ownerID, reportSlug, reachID, body.Name, body.ReportDate, reportTimeVal,
		body.Content, body.HazardWarning, body.Paddled, cfs, band,
	).Scan(&id)
	if err != nil {
		errorResponse(w, http.StatusInternalServerError, fmt.Sprintf("create report: %v", err))
		return
	}

	jsonResponse(w, http.StatusCreated, map[string]any{
		"id":     id,
		"slug":   reportSlug,
		"handle": handle,
		"url":    fmt.Sprintf("/reports/%s", id),
		"notice": "All reach reports are public on this site. Please be courteous.",
	})
}

// ── GET /reaches/{slug}/reports ───────────────────────────────────────────────

func (h *ReportHandler) ListByReach(w http.ResponseWriter, r *http.Request) {
	reachSlug := chi.URLParam(r, "slug")
	ctx := r.Context()

	var reachID string
	if err := h.db.QueryRow(ctx,
		`SELECT id FROM reaches WHERE slug = $1`, reachSlug,
	).Scan(&reachID); err != nil {
		errorResponse(w, http.StatusNotFound, "reach not found")
		return
	}

	const limit = 25
	cursor := r.URL.Query().Get("cursor")

	var (
		query string
		args  []any
	)
	if cursor != "" {
		query = `
			SELECT rp.id, rp.slug, rp.name, rp.report_date::TEXT, rp.report_time::TEXT,
			       rp.content, rp.hazard_warning, rp.paddled,
			       rp.flow_cfs, rp.flow_band, rp.created_at, up.handle
			FROM reports rp
			LEFT JOIN user_profiles up ON up.owner_id = rp.owner_id
			WHERE rp.reach_id = $1 AND rp.report_date < $2::DATE
			ORDER BY rp.report_date DESC, rp.created_at DESC
			LIMIT $3`
		args = []any{reachID, cursor, limit + 1}
	} else {
		query = `
			SELECT rp.id, rp.slug, rp.name, rp.report_date::TEXT, rp.report_time::TEXT,
			       rp.content, rp.hazard_warning, rp.paddled,
			       rp.flow_cfs, rp.flow_band, rp.created_at, up.handle
			FROM reports rp
			LEFT JOIN user_profiles up ON up.owner_id = rp.owner_id
			WHERE rp.reach_id = $1
			ORDER BY rp.report_date DESC, rp.created_at DESC
			LIMIT $2`
		args = []any{reachID, limit + 1}
	}

	rows, err := h.db.Query(ctx, query, args...)
	if err != nil {
		errorResponse(w, http.StatusInternalServerError, "query failed")
		return
	}
	defer rows.Close()

	type reportRow struct {
		ID            string   `json:"id"`
		Slug          string   `json:"slug"`
		Name          string   `json:"name"`
		ReportDate    string   `json:"report_date"`
		ReportTime    *string  `json:"report_time,omitempty"`
		Content       string   `json:"content"`
		HazardWarning *string  `json:"hazard_warning,omitempty"`
		Paddled       bool     `json:"paddled"`
		FlowCFS       *float64 `json:"flow_cfs,omitempty"`
		FlowBand      *string  `json:"flow_band,omitempty"`
		CreatedAt     string   `json:"created_at"`
		Handle        *string  `json:"handle,omitempty"`
		URL           string   `json:"url,omitempty"`
	}

	var reports []reportRow
	for rows.Next() {
		var rep reportRow
		var createdAt time.Time
		if err := rows.Scan(
			&rep.ID, &rep.Slug, &rep.Name,
			&rep.ReportDate, &rep.ReportTime,
			&rep.Content, &rep.HazardWarning, &rep.Paddled,
			&rep.FlowCFS, &rep.FlowBand, &createdAt,
			&rep.Handle,
		); err != nil {
			errorResponse(w, http.StatusInternalServerError, "scan failed")
			return
		}
		rep.CreatedAt = createdAt.Format(time.RFC3339)
		rep.URL = fmt.Sprintf("/reports/%s", rep.ID)
		reports = append(reports, rep)
	}
	if reports == nil {
		reports = []reportRow{}
	}

	var nextCursor *string
	if len(reports) > limit {
		reports = reports[:limit]
		last := reports[len(reports)-1].ReportDate
		nextCursor = &last
	}

	// Hazard reports float to top within the page.
	var hazards, rest []reportRow
	for _, rep := range reports {
		if rep.HazardWarning != nil && *rep.HazardWarning != "" {
			hazards = append(hazards, rep)
		} else {
			rest = append(rest, rep)
		}
	}
	sorted := append(hazards, rest...)

	jsonResponse(w, http.StatusOK, map[string]any{
		"reports":     sorted,
		"next_cursor": nextCursor,
	})
}

// ── GET /reports/{handle}/{slug} ──────────────────────────────────────────────

func (h *ReportHandler) Get(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	ctx := r.Context()

	type detail struct {
		ID            string   `json:"id"`
		Slug          string   `json:"slug"`
		Handle        string   `json:"handle"`
		Name          string   `json:"name"`
		ReportDate    string   `json:"report_date"`
		ReportTime    *string  `json:"report_time,omitempty"`
		Content       string   `json:"content"`
		HazardWarning *string  `json:"hazard_warning,omitempty"`
		Paddled       bool     `json:"paddled"`
		FlowCFS       *float64 `json:"flow_cfs,omitempty"`
		FlowBand      *string  `json:"flow_band,omitempty"`
		AWsyncedAt    *string  `json:"aw_synced_at,omitempty"`
		CreatedAt     string   `json:"created_at"`
		ReachName     string   `json:"reach_name"`
		ReachSlug     string   `json:"reach_slug"`
	}

	var d detail
	var createdAt time.Time
	var awSyncedAt *time.Time
	err := h.db.QueryRow(ctx, `
		SELECT
			rp.id, rp.slug, COALESCE(up.handle, '') AS handle,
			rp.name, rp.report_date::TEXT, rp.report_time::TEXT,
			rp.content, rp.hazard_warning, rp.paddled,
			rp.flow_cfs, rp.flow_band, rp.aw_synced_at, rp.created_at,
			COALESCE(re.name, '') AS reach_name,
			COALESCE(re.slug, '') AS reach_slug
		FROM reports rp
		LEFT JOIN user_profiles up ON up.owner_id = rp.owner_id
		JOIN reaches re ON re.id = rp.reach_id
		WHERE rp.id = $1
	`, id).Scan(
		&d.ID, &d.Slug, &d.Handle,
		&d.Name, &d.ReportDate, &d.ReportTime,
		&d.Content, &d.HazardWarning, &d.Paddled,
		&d.FlowCFS, &d.FlowBand, &awSyncedAt, &createdAt,
		&d.ReachName, &d.ReachSlug,
	)
	if err != nil {
		errorResponse(w, http.StatusNotFound, "report not found")
		return
	}
	d.CreatedAt = createdAt.Format(time.RFC3339)
	if awSyncedAt != nil {
		s := awSyncedAt.Format(time.RFC3339)
		d.AWsyncedAt = &s
	}
	jsonResponse(w, http.StatusOK, d)
}

// ── PATCH /me/reports/{slug} ──────────────────────────────────────────────────

func (h *ReportHandler) Update(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")

	ownerID, ok := h.ownerID(r)
	if !ok {
		errorResponse(w, http.StatusUnauthorized, "authentication required")
		return
	}

	var body struct {
		Name          *string `json:"name"`
		Content       *string `json:"content"`
		HazardWarning *string `json:"hazard_warning"`
		Paddled       *bool   `json:"paddled"`
		ReportTime    *string `json:"report_time"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		errorResponse(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	ctx := r.Context()

	var createdAt time.Time
	err := h.db.QueryRow(ctx,
		`SELECT created_at FROM reports WHERE owner_id = $1 AND slug = $2`,
		ownerID, slug,
	).Scan(&createdAt)
	if errors.Is(err, pgx.ErrNoRows) {
		errorResponse(w, http.StatusNotFound, "report not found")
		return
	}
	if err != nil {
		errorResponse(w, http.StatusInternalServerError, "lookup failed")
		return
	}
	if time.Since(createdAt) > 24*time.Hour {
		errorResponse(w, http.StatusForbidden, "reports are locked for editing after 24 hours")
		return
	}

	tag, err := h.db.Exec(ctx, `
		UPDATE reports
		SET
			name           = COALESCE($1, name),
			content        = COALESCE($2, content),
			hazard_warning = COALESCE($3, hazard_warning),
			paddled        = COALESCE($4, paddled),
			report_time    = COALESCE($5, report_time),
			updated_at     = NOW()
		WHERE owner_id = $6 AND slug = $7
	`, body.Name, body.Content, body.HazardWarning, body.Paddled, body.ReportTime,
		ownerID, slug)
	if err != nil {
		errorResponse(w, http.StatusInternalServerError, "update failed")
		return
	}
	if tag.RowsAffected() == 0 {
		errorResponse(w, http.StatusNotFound, "report not found")
		return
	}

	jsonResponse(w, http.StatusOK, map[string]string{"status": "ok"})
}

// ── POST /me/reports/{slug}/aw-sync ──────────────────────────────────────────

func (h *ReportHandler) AWSync(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")

	ownerID, ok := h.ownerID(r)
	if !ok {
		errorResponse(w, http.StatusUnauthorized, "authentication required")
		return
	}

	tag, err := h.db.Exec(r.Context(),
		`UPDATE reports SET aw_synced_at = NOW(), updated_at = NOW()
		 WHERE owner_id = $1 AND slug = $2`,
		ownerID, slug,
	)
	if err != nil {
		errorResponse(w, http.StatusInternalServerError, "sync stamp failed")
		return
	}
	if tag.RowsAffected() == 0 {
		errorResponse(w, http.StatusNotFound, "report not found")
		return
	}

	jsonResponse(w, http.StatusOK, map[string]string{"status": "ok"})
}

// ── DELETE /me/reports/{slug} ─────────────────────────────────────────────────

func (h *ReportHandler) Delete(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")

	ownerID, ok := h.ownerID(r)
	if !ok {
		errorResponse(w, http.StatusUnauthorized, "authentication required")
		return
	}

	tag, err := h.db.Exec(r.Context(),
		`DELETE FROM reports WHERE owner_id = $1 AND slug = $2`,
		ownerID, slug,
	)
	if err != nil {
		errorResponse(w, http.StatusInternalServerError, "delete failed")
		return
	}
	if tag.RowsAffected() == 0 {
		errorResponse(w, http.StatusNotFound, "report not found")
		return
	}
	jsonResponse(w, http.StatusOK, map[string]string{"status": "ok"})
}

// ── GET /me/reports ───────────────────────────────────────────────────────────

func (h *ReportHandler) ListMine(w http.ResponseWriter, r *http.Request) {
	ownerID, ok := h.ownerID(r)
	if !ok {
		errorResponse(w, http.StatusUnauthorized, "authentication required")
		return
	}
	ctx := r.Context()

	rows, err := h.db.Query(ctx, `
		SELECT
			rp.id, rp.slug,
			rp.name, rp.report_date::TEXT, rp.report_time::TEXT,
			rp.content, rp.hazard_warning, rp.paddled,
			rp.flow_cfs, rp.flow_band, rp.created_at,
			COALESCE(re.name, '') AS reach_name,
			COALESCE(re.slug, '') AS reach_slug,
			up.handle
		FROM reports rp
		JOIN reaches re ON re.id = rp.reach_id
		LEFT JOIN user_profiles up ON up.owner_id = rp.owner_id
		WHERE rp.owner_id = $1
		ORDER BY rp.report_date DESC, rp.created_at DESC
		LIMIT 200
	`, ownerID)
	if err != nil {
		errorResponse(w, http.StatusInternalServerError, "query failed")
		return
	}
	defer rows.Close()

	type myReport struct {
		ID            string   `json:"id"`
		Slug          string   `json:"slug"`
		Name          string   `json:"name"`
		ReportDate    string   `json:"report_date"`
		ReportTime    *string  `json:"report_time,omitempty"`
		Content       string   `json:"content"`
		HazardWarning *string  `json:"hazard_warning,omitempty"`
		Paddled       bool     `json:"paddled"`
		FlowCFS       *float64 `json:"flow_cfs,omitempty"`
		FlowBand      *string  `json:"flow_band,omitempty"`
		CreatedAt     string   `json:"created_at"`
		ReachName     string   `json:"reach_name"`
		ReachSlug     string   `json:"reach_slug"`
		URL           string   `json:"url,omitempty"`
	}

	var reports []myReport
	for rows.Next() {
		var rep myReport
		var createdAt time.Time
		var handle *string
		if err := rows.Scan(
			&rep.ID, &rep.Slug,
			&rep.Name, &rep.ReportDate, &rep.ReportTime,
			&rep.Content, &rep.HazardWarning, &rep.Paddled,
			&rep.FlowCFS, &rep.FlowBand, &createdAt,
			&rep.ReachName, &rep.ReachSlug,
			&handle,
		); err != nil {
			errorResponse(w, http.StatusInternalServerError, "scan failed")
			return
		}
		rep.CreatedAt = createdAt.Format(time.RFC3339)
		rep.URL = fmt.Sprintf("/reports/%s", rep.ID)
		reports = append(reports, rep)
	}
	if reports == nil {
		reports = []myReport{}
	}
	jsonResponse(w, http.StatusOK, reports)
}

// ── GET /reaches/active-hazards ───────────────────────────────────────────────

// ActiveHazards returns all reaches with hazard reports filed within the last 2
// days — one call lets the dashboard badge every card in the watchlist at once.
func (h *ReportHandler) ActiveHazards(w http.ResponseWriter, r *http.Request) {
	type activeHazard struct {
		Slug          string `json:"slug"`
		HazardWarning string `json:"hazard_warning"`
		ReportDate    string `json:"report_date"`
		ReporterName  string `json:"reporter_name"`
	}

	rows, err := h.db.Query(r.Context(), `
		SELECT re.slug, rp.hazard_warning, rp.report_date::TEXT, rp.name
		FROM reports rp
		JOIN reaches re ON re.id = rp.reach_id
		WHERE rp.hazard_warning IS NOT NULL
		  AND rp.report_date >= CURRENT_DATE - INTERVAL '2 days'
		ORDER BY rp.report_date DESC, rp.created_at DESC
	`)
	if err != nil {
		errorResponse(w, http.StatusInternalServerError, "query failed")
		return
	}
	defer rows.Close()

	var hazards []activeHazard
	for rows.Next() {
		var item activeHazard
		if err := rows.Scan(&item.Slug, &item.HazardWarning, &item.ReportDate, &item.ReporterName); err != nil {
			continue
		}
		hazards = append(hazards, item)
	}
	if hazards == nil {
		hazards = []activeHazard{}
	}
	w.Header().Set("Cache-Control", "public, max-age=60")
	jsonResponse(w, http.StatusOK, hazards)
}
