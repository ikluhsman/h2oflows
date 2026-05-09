package handlers

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/h2oflow/h2oflow/apps/api/internal/auth"
	"github.com/jackc/pgx/v5/pgxpool"
)

// CustomGaugeHandler handles /api/v1/me/custom-gauges routes.
type CustomGaugeHandler struct {
	db            *pgxpool.Pool
	devFallbackID string
}

func NewCustomGaugeHandler(db *pgxpool.Pool, devFallbackID string) *CustomGaugeHandler {
	return &CustomGaugeHandler{db: db, devFallbackID: devFallbackID}
}

func (h *CustomGaugeHandler) ownerID(r *http.Request) (string, bool) {
	if id, ok := auth.UserIDFromContext(r.Context()); ok {
		return id, true
	}
	if h.devFallbackID != "" {
		return h.devFallbackID, true
	}
	return "", false
}

// ── Types ─────────────────────────────────────────────────────────────────────

type customGaugeInputRow struct {
	Position   int    `json:"position"`
	GaugeID    string `json:"gauge_id"`
	ExternalID string `json:"external_id"`
	Source     string `json:"source"`
	GaugeName  string `json:"gauge_name"`
	Sign       int    `json:"sign"`
}

type customGaugeSummary struct {
	ID                 string     `json:"id"`
	Slug               string     `json:"slug"`
	Name               string     `json:"name"`
	Description        *string    `json:"description"`
	Note               *string    `json:"note"`
	Unit               string     `json:"unit"`
	LastValue          *float64   `json:"last_value_cfs"`
	LastValueAt        *time.Time `json:"last_value_at"`
	AnyInputUnhealthy  bool       `json:"any_input_unhealthy"`
	CreatedAt          time.Time  `json:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at"`
}

type customGaugeDetail struct {
	customGaugeSummary
	Inputs []customGaugeInputRow `json:"inputs"`
}

// sharePayload is encoded in the share token — portable across users.
type sharePayload struct {
	Name        string             `json:"name"`
	Description *string            `json:"description"`
	Note        *string            `json:"note"`
	Unit        string             `json:"unit"`
	Inputs      []shareInputRecord `json:"inputs"`
}

type shareInputRecord struct {
	Position   int    `json:"position"`
	ExternalID string `json:"external_id"`
	Source     string `json:"source"`
	Sign       int    `json:"sign"`
}

// ── Helpers ───────────────────────────────────────────────────────────────────

var nonAlnum = regexp.MustCompile(`[^a-z0-9]+`)

func slugify(s string) string {
	s = strings.ToLower(s)
	s = nonAlnum.ReplaceAllString(s, "-")
	s = strings.Trim(s, "-")
	if len(s) > 64 {
		s = s[:64]
	}
	if s == "" {
		s = "gauge"
	}
	return s
}

func isUniqueViolation(err error) bool {
	msg := err.Error()
	return strings.Contains(msg, "unique") || strings.Contains(msg, "duplicate") || strings.Contains(msg, "23505")
}

func loadCustomGaugeInputs(db *pgxpool.Pool, r *http.Request, gaugeID string) ([]customGaugeInputRow, error) {
	rows, err := db.Query(r.Context(), `
		SELECT cgi.position, cgi.gauge_id::text, g.external_id, g.source, g.name, cgi.sign
		FROM   custom_gauge_inputs cgi
		JOIN   gauges g ON g.id = cgi.gauge_id
		WHERE  cgi.custom_gauge_id = $1::uuid
		ORDER  BY cgi.position
	`, gaugeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var inputs []customGaugeInputRow
	for rows.Next() {
		var inp customGaugeInputRow
		if err := rows.Scan(&inp.Position, &inp.GaugeID, &inp.ExternalID, &inp.Source, &inp.GaugeName, &inp.Sign); err != nil {
			return nil, err
		}
		inputs = append(inputs, inp)
	}
	if inputs == nil {
		inputs = []customGaugeInputRow{}
	}
	return inputs, nil
}

type inputSpec struct {
	GaugeID string `json:"gauge_id"`
	Sign    int    `json:"sign"`
}

func replaceInputs(db *pgxpool.Pool, r *http.Request, customGaugeID string, specs []inputSpec) error {
	if _, err := db.Exec(r.Context(),
		`DELETE FROM custom_gauge_inputs WHERE custom_gauge_id = $1::uuid`, customGaugeID); err != nil {
		return err
	}
	for i, inp := range specs {
		sign := inp.Sign
		if sign != 1 && sign != -1 {
			sign = 1
		}
		if _, err := db.Exec(r.Context(),
			`INSERT INTO custom_gauge_inputs (custom_gauge_id, position, gauge_id, sign)
			 VALUES ($1::uuid, $2, $3::uuid, $4)`,
			customGaugeID, i, inp.GaugeID, sign,
		); err != nil {
			return err
		}
	}
	return nil
}

// ── List ──────────────────────────────────────────────────────────────────────

// GET /api/v1/me/custom-gauges
func (h *CustomGaugeHandler) List(w http.ResponseWriter, r *http.Request) {
	ownerID, ok := h.ownerID(r)
	if !ok {
		errorResponse(w, http.StatusUnauthorized, "authentication required")
		return
	}

	rows, err := h.db.Query(r.Context(), `
		SELECT cg.id::text, cg.slug, cg.name, cg.description, cg.note, cg.unit,
		       cg.last_value_cfs, cg.last_value_at,
		       EXISTS (
		           SELECT 1 FROM custom_gauge_inputs cgi
		           JOIN gauges g ON g.id = cgi.gauge_id
		           WHERE cgi.custom_gauge_id = cg.id
		             AND g.poll_health != 'healthy'
		       ) AS any_input_unhealthy,
		       cg.created_at, cg.updated_at
		FROM   custom_gauges cg
		WHERE  cg.owner_id = $1
		ORDER  BY cg.name
	`, ownerID)
	if err != nil {
		errorResponse(w, http.StatusInternalServerError, "query failed")
		return
	}
	defer rows.Close()

	items := []customGaugeSummary{}
	for rows.Next() {
		var g customGaugeSummary
		if err := rows.Scan(&g.ID, &g.Slug, &g.Name, &g.Description, &g.Note, &g.Unit,
			&g.LastValue, &g.LastValueAt, &g.AnyInputUnhealthy, &g.CreatedAt, &g.UpdatedAt); err == nil {
			items = append(items, g)
		}
	}

	jsonResponse(w, http.StatusOK, map[string]any{"items": items})
}

// ── Get ───────────────────────────────────────────────────────────────────────

// GET /api/v1/me/custom-gauges/{slug}
func (h *CustomGaugeHandler) Get(w http.ResponseWriter, r *http.Request) {
	ownerID, ok := h.ownerID(r)
	if !ok {
		errorResponse(w, http.StatusUnauthorized, "authentication required")
		return
	}

	slug := chi.URLParam(r, "slug")
	var g customGaugeSummary
	err := h.db.QueryRow(r.Context(), `
		SELECT id::text, slug, name, description, note, unit,
		       last_value_cfs, last_value_at, created_at, updated_at
		FROM   custom_gauges
		WHERE  owner_id = $1 AND slug = $2
	`, ownerID, slug).Scan(
		&g.ID, &g.Slug, &g.Name, &g.Description, &g.Note, &g.Unit,
		&g.LastValue, &g.LastValueAt, &g.CreatedAt, &g.UpdatedAt,
	)
	if err != nil {
		errorResponse(w, http.StatusNotFound, "not found")
		return
	}

	inputs, err := loadCustomGaugeInputs(h.db, r, g.ID)
	if err != nil {
		errorResponse(w, http.StatusInternalServerError, "query failed")
		return
	}

	jsonResponse(w, http.StatusOK, customGaugeDetail{customGaugeSummary: g, Inputs: inputs})
}

// ── Create ────────────────────────────────────────────────────────────────────

// POST /api/v1/me/custom-gauges
func (h *CustomGaugeHandler) Create(w http.ResponseWriter, r *http.Request) {
	ownerID, ok := h.ownerID(r)
	if !ok {
		errorResponse(w, http.StatusUnauthorized, "authentication required")
		return
	}

	var body struct {
		Name        string      `json:"name"`
		Slug        string      `json:"slug"`
		Description *string     `json:"description"`
		Note        *string     `json:"note"`
		Unit        string      `json:"unit"`
		Inputs      []inputSpec `json:"inputs"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Name == "" {
		errorResponse(w, http.StatusBadRequest, "name required")
		return
	}
	if len(body.Inputs) == 0 {
		errorResponse(w, http.StatusBadRequest, "at least one input required")
		return
	}
	if body.Unit == "" {
		body.Unit = "cfs"
	}

	var count int
	_ = h.db.QueryRow(r.Context(),
		`SELECT COUNT(*) FROM custom_gauges WHERE owner_id = $1`, ownerID).Scan(&count)
	if count >= 25 {
		errorResponse(w, http.StatusForbidden, "custom gauge limit reached (25 max)")
		return
	}

	slug := body.Slug
	if slug == "" {
		slug = slugify(body.Name)
	}

	var id string
	err := h.db.QueryRow(r.Context(), `
		INSERT INTO custom_gauges (owner_id, slug, name, description, note, unit)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id::text
	`, ownerID, slug, body.Name, body.Description, body.Note, body.Unit).Scan(&id)
	if err != nil {
		if isUniqueViolation(err) {
			errorResponse(w, http.StatusConflict, "slug already in use")
		} else {
			errorResponse(w, http.StatusInternalServerError, "insert failed")
		}
		return
	}

	if err := replaceInputs(h.db, r, id, body.Inputs); err != nil {
		errorResponse(w, http.StatusInternalServerError, "insert inputs failed")
		return
	}

	jsonResponse(w, http.StatusCreated, map[string]string{"id": id, "slug": slug})
}

// ── Update ────────────────────────────────────────────────────────────────────

// PATCH /api/v1/me/custom-gauges/{slug}
// Nil fields are left unchanged. Inputs array, if present, fully replaces existing inputs.
func (h *CustomGaugeHandler) Update(w http.ResponseWriter, r *http.Request) {
	ownerID, ok := h.ownerID(r)
	if !ok {
		errorResponse(w, http.StatusUnauthorized, "authentication required")
		return
	}

	currentSlug := chi.URLParam(r, "slug")

	var body struct {
		Name        *string     `json:"name"`
		Slug        *string     `json:"slug"`
		Description *string     `json:"description"`
		Note        *string     `json:"note"`
		Unit        *string     `json:"unit"`
		Inputs      []inputSpec `json:"inputs"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		errorResponse(w, http.StatusBadRequest, "invalid body")
		return
	}

	var id string
	err := h.db.QueryRow(r.Context(), `
		UPDATE custom_gauges
		SET    name        = COALESCE($3, name),
		       slug        = COALESCE($4, slug),
		       description = COALESCE($5, description),
		       note        = COALESCE($6, note),
		       unit        = COALESCE($7, unit),
		       updated_at  = NOW()
		WHERE  owner_id = $1 AND slug = $2
		RETURNING id::text
	`, ownerID, currentSlug, body.Name, body.Slug, body.Description, body.Note, body.Unit).Scan(&id)
	if err != nil {
		if isUniqueViolation(err) {
			errorResponse(w, http.StatusConflict, "slug already in use")
		} else {
			errorResponse(w, http.StatusNotFound, "not found")
		}
		return
	}

	if body.Inputs != nil {
		if err := replaceInputs(h.db, r, id, body.Inputs); err != nil {
			errorResponse(w, http.StatusInternalServerError, "update inputs failed")
			return
		}
	}

	newSlug := currentSlug
	if body.Slug != nil {
		newSlug = *body.Slug
	}
	jsonResponse(w, http.StatusOK, map[string]string{"id": id, "slug": newSlug})
}

// ── Delete ────────────────────────────────────────────────────────────────────

// DELETE /api/v1/me/custom-gauges/{slug}
func (h *CustomGaugeHandler) Delete(w http.ResponseWriter, r *http.Request) {
	ownerID, ok := h.ownerID(r)
	if !ok {
		errorResponse(w, http.StatusUnauthorized, "authentication required")
		return
	}

	slug := chi.URLParam(r, "slug")
	ct, err := h.db.Exec(r.Context(),
		`DELETE FROM custom_gauges WHERE owner_id = $1 AND slug = $2`,
		ownerID, slug,
	)
	if err != nil {
		errorResponse(w, http.StatusInternalServerError, "delete failed")
		return
	}
	if ct.RowsAffected() == 0 {
		errorResponse(w, http.StatusNotFound, "not found")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ── Share ─────────────────────────────────────────────────────────────────────

// GET /api/v1/me/custom-gauges/{slug}/share
// Returns a base64-encoded JSON payload the recipient can import.
func (h *CustomGaugeHandler) Share(w http.ResponseWriter, r *http.Request) {
	ownerID, ok := h.ownerID(r)
	if !ok {
		errorResponse(w, http.StatusUnauthorized, "authentication required")
		return
	}

	slug := chi.URLParam(r, "slug")
	var g customGaugeSummary
	err := h.db.QueryRow(r.Context(), `
		SELECT id::text, slug, name, description, note, unit,
		       last_value_cfs, last_value_at, created_at, updated_at
		FROM   custom_gauges
		WHERE  owner_id = $1 AND slug = $2
	`, ownerID, slug).Scan(
		&g.ID, &g.Slug, &g.Name, &g.Description, &g.Note, &g.Unit,
		&g.LastValue, &g.LastValueAt, &g.CreatedAt, &g.UpdatedAt,
	)
	if err != nil {
		errorResponse(w, http.StatusNotFound, "not found")
		return
	}

	rows, err := h.db.Query(r.Context(), `
		SELECT cgi.position, g.external_id, g.source, cgi.sign
		FROM   custom_gauge_inputs cgi
		JOIN   gauges g ON g.id = cgi.gauge_id
		WHERE  cgi.custom_gauge_id = $1::uuid
		ORDER  BY cgi.position
	`, g.ID)
	if err != nil {
		errorResponse(w, http.StatusInternalServerError, "query failed")
		return
	}
	defer rows.Close()

	payload := sharePayload{
		Name:        g.Name,
		Description: g.Description,
		Note:        g.Note,
		Unit:        g.Unit,
		Inputs:      []shareInputRecord{},
	}
	for rows.Next() {
		var inp shareInputRecord
		if err := rows.Scan(&inp.Position, &inp.ExternalID, &inp.Source, &inp.Sign); err == nil {
			payload.Inputs = append(payload.Inputs, inp)
		}
	}

	raw, err := json.Marshal(payload)
	if err != nil {
		errorResponse(w, http.StatusInternalServerError, "encode failed")
		return
	}

	token := base64.URLEncoding.EncodeToString(raw)
	jsonResponse(w, http.StatusOK, map[string]string{"token": token})
}

// ── Import ────────────────────────────────────────────────────────────────────

// POST /api/v1/me/custom-gauges/import
// Body: { "token": "<base64 share token>" }
// Decodes token, resolves gauge external_ids to local UUIDs, creates custom gauge.
func (h *CustomGaugeHandler) Import(w http.ResponseWriter, r *http.Request) {
	ownerID, ok := h.ownerID(r)
	if !ok {
		errorResponse(w, http.StatusUnauthorized, "authentication required")
		return
	}

	var body struct {
		Token string `json:"token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Token == "" {
		errorResponse(w, http.StatusBadRequest, "token required")
		return
	}

	raw, err := base64.URLEncoding.DecodeString(body.Token)
	if err != nil {
		errorResponse(w, http.StatusBadRequest, "invalid token")
		return
	}

	var payload sharePayload
	if err := json.Unmarshal(raw, &payload); err != nil || payload.Name == "" {
		errorResponse(w, http.StatusBadRequest, "invalid token")
		return
	}

	// Resolve external_id+source → gauge UUID for each input.
	resolved := make([]inputSpec, 0, len(payload.Inputs))
	for _, inp := range payload.Inputs {
		var gaugeID string
		err := h.db.QueryRow(r.Context(),
			`SELECT id::text FROM gauges WHERE external_id = $1 AND source = $2`,
			inp.ExternalID, inp.Source,
		).Scan(&gaugeID)
		if err != nil {
			errorResponse(w, http.StatusUnprocessableEntity,
				"gauge not found: "+inp.ExternalID+" ("+inp.Source+")")
			return
		}
		sign := inp.Sign
		if sign != 1 && sign != -1 {
			sign = 1
		}
		resolved = append(resolved, inputSpec{GaugeID: gaugeID, Sign: sign})
	}

	var count int
	_ = h.db.QueryRow(r.Context(),
		`SELECT COUNT(*) FROM custom_gauges WHERE owner_id = $1`, ownerID).Scan(&count)
	if count >= 25 {
		errorResponse(w, http.StatusForbidden, "custom gauge limit reached (25 max)")
		return
	}

	unit := payload.Unit
	if unit == "" {
		unit = "cfs"
	}

	// Try base slug, fall back to slug+"-import" on conflict.
	slug := slugify(payload.Name)
	var id string
	err = h.db.QueryRow(r.Context(), `
		INSERT INTO custom_gauges (owner_id, slug, name, description, note, unit)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id::text
	`, ownerID, slug, payload.Name, payload.Description, payload.Note, unit).Scan(&id)
	if err != nil && isUniqueViolation(err) {
		slug = slug + "-import"
		err = h.db.QueryRow(r.Context(), `
			INSERT INTO custom_gauges (owner_id, slug, name, description, note, unit)
			VALUES ($1, $2, $3, $4, $5, $6)
			RETURNING id::text
		`, ownerID, slug, payload.Name, payload.Description, payload.Note, unit).Scan(&id)
	}
	if err != nil {
		if isUniqueViolation(err) {
			errorResponse(w, http.StatusConflict, "custom gauge already imported")
		} else {
			errorResponse(w, http.StatusInternalServerError, "insert failed")
		}
		return
	}

	if err := replaceInputs(h.db, r, id, resolved); err != nil {
		errorResponse(w, http.StatusInternalServerError, "insert inputs failed")
		return
	}

	jsonResponse(w, http.StatusCreated, map[string]string{"id": id, "slug": slug})
}

// ── Readings ──────────────────────────────────────────────────────────────────

// GET /api/v1/me/custom-gauges/{slug}/readings?since=<rfc3339>
// Returns a combined time-series by bucketing component gauge readings into
// 15-minute intervals and summing them with their signs. Only buckets where all
// input gauges have a reading are included (HAVING clause). The `since` query
// param defines the window (defaults to 48h, clamped to [1h, 30d]).
func (h *CustomGaugeHandler) Readings(w http.ResponseWriter, r *http.Request) {
	ownerID, ok := h.ownerID(r)
	if !ok {
		errorResponse(w, http.StatusUnauthorized, "authentication required")
		return
	}
	slug := chi.URLParam(r, "slug")

	window := 48 * time.Hour
	if s := r.URL.Query().Get("since"); s != "" {
		if t, err := time.Parse(time.RFC3339, s); err == nil {
			d := time.Since(t)
			if d < time.Hour {
				d = time.Hour
			} else if d > 30*24*time.Hour {
				d = 30 * 24 * time.Hour
			}
			window = d
		}
	}

	type reading struct {
		Timestamp time.Time `json:"timestamp"`
		CFS       float64   `json:"cfs"`
	}

	rows, err := h.db.Query(r.Context(), `
		WITH cg AS (
			SELECT id FROM custom_gauges WHERE slug = $1 AND owner_id = $2 LIMIT 1
		),
		inputs AS (
			SELECT cgi.gauge_id, cgi.sign
			FROM custom_gauge_inputs cgi
			WHERE cgi.custom_gauge_id = (SELECT id FROM cg)
		),
		bucketed AS (
			SELECT
				date_trunc('minute', gr.timestamp)
					- ((EXTRACT(MINUTE FROM gr.timestamp)::int % 30) * INTERVAL '1 minute') AS bucket,
				inp.gauge_id,
				AVG(gr.value) AS avg_val,
				inp.sign
			FROM inputs inp
			JOIN gauge_readings gr ON gr.gauge_id = inp.gauge_id
			WHERE gr.timestamp > NOW() - $3::interval
			GROUP BY bucket, inp.gauge_id, inp.sign
		)
		SELECT bucket, SUM(avg_val * sign) AS cfs
		FROM bucketed
		GROUP BY bucket
		ORDER BY bucket DESC
		LIMIT 500
	`, slug, ownerID, fmt.Sprintf("%d seconds", int(window.Seconds())))
	if err != nil {
		errorResponse(w, http.StatusInternalServerError, "query failed")
		return
	}
	defer rows.Close()

	results := make([]reading, 0, 200)
	for rows.Next() {
		var rd reading
		if err := rows.Scan(&rd.Timestamp, &rd.CFS); err == nil {
			results = append(results, rd)
		}
	}
	jsonResponse(w, http.StatusOK, results)
}
