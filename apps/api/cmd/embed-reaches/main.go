// embed-reaches generates vector embeddings for all reach content and stores
// them in the reach_embeddings table for RAG-powered river assistant queries.
//
// Chunks per reach:
//   - reach_description — name, class, region, overview text
//   - rapid             — one per named rapid (name, class, beta)
//   - access_point      — one per access point (type, name, directions, notes)
//   - flow_ranges       — one per reach covering all flow bands from the primary gauge
//
// Already-embedded chunks are skipped (idempotent via ON CONFLICT DO NOTHING).
// Use REEMBED=true to wipe and re-embed everything.
//
//	go run ./cmd/embed-reaches
//	REEMBED=true go run ./cmd/embed-reaches
//
// Env vars: DATABASE_URL, VOYAGE_API_KEY
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/h2oflow/h2oflow/apps/api/internal/ai"
	"github.com/h2oflow/h2oflow/apps/api/internal/db"
	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	ctx := context.Background()

	dbURL    := mustEnv("DATABASE_URL")
	apiKey   := mustEnv("VOYAGE_API_KEY")
	reembed := os.Getenv("REEMBED") == "true"

	pool, err := db.Connect(ctx, dbURL)
	if err != nil {
		log.Fatalf("db: %v", err)
	}
	defer pool.Close()

	embedder := ai.NewEmbedder(apiKey)

	if reembed {
		fmt.Println("REEMBED=true — deleting existing embeddings")
		if _, err := pool.Exec(ctx, `DELETE FROM reach_embeddings`); err != nil {
			log.Fatalf("delete embeddings: %v", err)
		}
	}

	reaches, err := loadReaches(ctx, pool)
	if err != nil {
		log.Fatalf("load reaches: %v", err)
	}
	fmt.Printf("loaded %d reaches\n", len(reaches))

	var totalChunks, embedded, skipped int

	for _, r := range reaches {
		fmt.Printf("\n→ %s\n", r.name)

		chunks := buildChunks(r)
		totalChunks += len(chunks)

		if len(chunks) == 0 {
			fmt.Printf("  ○ no content to embed\n")
			continue
		}

		// Filter to chunks not yet in the DB (skip existing, don't count as error).
		toEmbed := filterUnembedded(ctx, pool, r.id, chunks)
		if len(toEmbed) == 0 {
			fmt.Printf("  ○ all %d chunks already embedded\n", len(chunks))
			skipped += len(chunks)
			continue
		}
		skipped += len(chunks) - len(toEmbed)
		fmt.Printf("  embedding %d/%d chunks…\n", len(toEmbed), len(chunks))

		// Embed in batches of 100.
		for i := 0; i < len(toEmbed); i += 100 {
			end := i + 100
			if end > len(toEmbed) {
				end = len(toEmbed)
			}
			batch := toEmbed[i:end]

			texts := make([]string, len(batch))
			for j, c := range batch {
				texts[j] = c.text
			}

			vecs, err := embedder.Embed(ctx, texts)
			if err != nil {
				log.Fatalf("  ✗ embed: %v", err)
			}

			for j, c := range batch {
				if vecs[j] == nil {
					fmt.Printf("  ✗ nil embedding for chunk %q\n", c.chunkType)
					continue
				}
				if err := insertEmbedding(ctx, pool, r.id, c, vecs[j]); err != nil {
					fmt.Printf("  ✗ insert %s: %v\n", c.chunkType, err)
				} else {
					embedded++
				}
			}
		}
		fmt.Printf("  ✓ done (%d chunks embedded)\n", len(toEmbed))
		time.Sleep(22 * time.Second) // stay under Voyage free-tier 3 RPM limit
	}

	fmt.Printf("\ndone — %d total chunks, %d embedded, %d skipped\n", totalChunks, embedded, skipped)
}

// ── data model ──────────────────────────────────────────────────────────────

type reachRow struct {
	id          string
	name        string
	region      string
	classMin    *float64
	classMax    *float64
	lengthMi    *float64
	description *string

	rapids  []rapidRow
	access  []accessRow
	flowRanges []flowRangeRow
}

type rapidRow struct {
	id          string
	name        string
	classRating *float64
	riverMile   *float64
	description *string
	portageDesc *string
}

type accessRow struct {
	id         string
	accessType string
	name       *string
	directions *string
	notes      *string
}

type flowRangeRow struct {
	label     string
	minCFS    *float64
	maxCFS    *float64
}

type chunk struct {
	chunkType string
	rapidID   *string // set for 'rapid' chunks
	accessID  *string // set for 'access_point' chunks
	text      string
}

// ── queries ──────────────────────────────────────────────────────────────────

func loadReaches(ctx context.Context, pool *pgxpool.Pool) ([]reachRow, error) {
	rows, err := pool.Query(ctx, `
		SELECT id, name, region, class_min, class_max, length_mi, description
		FROM reaches
		WHERE description IS NOT NULL
		ORDER BY name
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var reaches []reachRow
	for rows.Next() {
		var r reachRow
		if err := rows.Scan(&r.id, &r.name, &r.region, &r.classMin, &r.classMax, &r.lengthMi, &r.description); err != nil {
			return nil, err
		}
		reaches = append(reaches, r)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Hydrate rapids, access, and flow ranges for each reach.
	for i := range reaches {
		r := &reaches[i]
		if err := loadRapids(ctx, pool, r); err != nil {
			return nil, fmt.Errorf("rapids for %s: %w", r.name, err)
		}
		if err := loadAccess(ctx, pool, r); err != nil {
			return nil, fmt.Errorf("access for %s: %w", r.name, err)
		}
		if err := loadFlowRanges(ctx, pool, r); err != nil {
			return nil, fmt.Errorf("flow ranges for %s: %w", r.name, err)
		}
	}
	return reaches, nil
}

func loadRapids(ctx context.Context, pool *pgxpool.Pool, r *reachRow) error {
	rows, err := pool.Query(ctx, `
		SELECT id, name, class_rating, river_mile, description, portage_description
		FROM rapids WHERE reach_id = $1 ORDER BY river_mile NULLS LAST, name
	`, r.id)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var rr rapidRow
		if err := rows.Scan(&rr.id, &rr.name, &rr.classRating, &rr.riverMile, &rr.description, &rr.portageDesc); err != nil {
			return err
		}
		r.rapids = append(r.rapids, rr)
	}
	return rows.Err()
}

func loadAccess(ctx context.Context, pool *pgxpool.Pool, r *reachRow) error {
	rows, err := pool.Query(ctx, `
		SELECT id, access_type, name, directions, notes
		FROM reach_access WHERE reach_id = $1 ORDER BY access_type, name
	`, r.id)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var a accessRow
		if err := rows.Scan(&a.id, &a.accessType, &a.name, &a.directions, &a.notes); err != nil {
			return err
		}
		r.access = append(r.access, a)
	}
	return rows.Err()
}

func loadFlowRanges(ctx context.Context, pool *pgxpool.Pool, r *reachRow) error {
	// Join through primary_gauge_id → flow_ranges. craft_type='general' preferred;
	// fall back to any if no general rows exist.
	rows, err := pool.Query(ctx, `
		SELECT fr.label, fr.min_cfs, fr.max_cfs
		FROM flow_ranges fr
		JOIN reaches rc ON rc.primary_gauge_id = fr.gauge_id
		WHERE rc.id = $1
		  AND fr.craft_type = 'general'
		ORDER BY CASE fr.label
			WHEN 'too_low'  THEN 1
			WHEN 'minimum'  THEN 2
			WHEN 'fun'      THEN 3
			WHEN 'optimal'  THEN 4
			WHEN 'pushy'    THEN 5
			WHEN 'high'     THEN 6
			WHEN 'flood'    THEN 7
			ELSE 8
		END
	`, r.id)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var fr flowRangeRow
		if err := rows.Scan(&fr.label, &fr.minCFS, &fr.maxCFS); err != nil {
			return err
		}
		r.flowRanges = append(r.flowRanges, fr)
	}
	return rows.Err()
}

// ── chunk builders ───────────────────────────────────────────────────────────

func buildChunks(r reachRow) []chunk {
	var chunks []chunk

	// reach_description
	if r.description != nil && *r.description != "" {
		chunks = append(chunks, chunk{
			chunkType: "reach_description",
			text:      reachDescChunk(r),
		})
	}

	// rapids
	for _, rr := range r.rapids {
		if txt := rapidChunk(r.name, rr); txt != "" {
			id := rr.id
			chunks = append(chunks, chunk{
				chunkType: "rapid",
				rapidID:   &id,
				text:      txt,
			})
		}
	}

	// access points
	for _, a := range r.access {
		if txt := accessChunk(r.name, a); txt != "" {
			id := a.id
			chunks = append(chunks, chunk{
				chunkType: "access_point",
				accessID:  &id,
				text:      txt,
			})
		}
	}

	// flow ranges — one chunk summarising all bands
	if len(r.flowRanges) > 0 {
		chunks = append(chunks, chunk{
			chunkType: "flow_ranges",
			text:      flowRangesChunk(r),
		})
	}

	return chunks
}

func reachDescChunk(r reachRow) string {
	var sb strings.Builder
	sb.WriteString(r.name)
	if r.region != "" {
		fmt.Fprintf(&sb, " (%s)", r.region)
	}
	if r.classMin != nil || r.classMax != nil {
		sb.WriteString(" — Class ")
		if r.classMin != nil && r.classMax != nil && *r.classMin != *r.classMax {
			fmt.Fprintf(&sb, "%.0f–%.0f", *r.classMin, *r.classMax)
		} else if r.classMin != nil {
			fmt.Fprintf(&sb, "%.0f", *r.classMin)
		} else {
			fmt.Fprintf(&sb, "%.0f", *r.classMax)
		}
	}
	if r.lengthMi != nil {
		fmt.Fprintf(&sb, ", %.1f miles", *r.lengthMi)
	}
	if r.description != nil {
		fmt.Fprintf(&sb, "\n%s", *r.description)
	}
	return sb.String()
}

func rapidChunk(reachName string, rr rapidRow) string {
	if rr.description == nil && rr.portageDesc == nil {
		return ""
	}
	var sb strings.Builder
	fmt.Fprintf(&sb, "%s – %s", reachName, rr.name)
	if rr.classRating != nil {
		fmt.Fprintf(&sb, " (Class %.0f", *rr.classRating)
		if rr.riverMile != nil {
			fmt.Fprintf(&sb, ", mile %.1f", *rr.riverMile)
		}
		sb.WriteByte(')')
	} else if rr.riverMile != nil {
		fmt.Fprintf(&sb, " (mile %.1f)", *rr.riverMile)
	}
	if rr.description != nil && *rr.description != "" {
		fmt.Fprintf(&sb, "\n%s", *rr.description)
	}
	if rr.portageDesc != nil && *rr.portageDesc != "" {
		fmt.Fprintf(&sb, "\nPortage: %s", *rr.portageDesc)
	}
	return sb.String()
}

func accessChunk(reachName string, a accessRow) string {
	name := deref(a.name, "")
	dirs := deref(a.directions, "")
	notes := deref(a.notes, "")
	if dirs == "" && notes == "" {
		return ""
	}
	var sb strings.Builder
	label := strings.ReplaceAll(a.accessType, "_", " ")
	if name != "" {
		fmt.Fprintf(&sb, "%s – %s: %s", reachName, label, name)
	} else {
		fmt.Fprintf(&sb, "%s – %s", reachName, label)
	}
	if dirs != "" {
		fmt.Fprintf(&sb, "\n%s", dirs)
	}
	if notes != "" {
		fmt.Fprintf(&sb, "\n%s", notes)
	}
	return sb.String()
}

func flowRangesChunk(r reachRow) string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "%s flow conditions:\n", r.name)
	for _, fr := range r.flowRanges {
		label := strings.ReplaceAll(fr.label, "_", " ")
		switch {
		case fr.minCFS != nil && fr.maxCFS != nil:
			fmt.Fprintf(&sb, "- %s: %.0f–%.0f cfs\n", label, *fr.minCFS, *fr.maxCFS)
		case fr.minCFS != nil:
			fmt.Fprintf(&sb, "- %s: above %.0f cfs\n", label, *fr.minCFS)
		case fr.maxCFS != nil:
			fmt.Fprintf(&sb, "- %s: below %.0f cfs\n", label, *fr.maxCFS)
		}
	}
	return strings.TrimRight(sb.String(), "\n")
}

func deref(s *string, fallback string) string {
	if s == nil {
		return fallback
	}
	return *s
}

// ── insert ───────────────────────────────────────────────────────────────────

// filterUnembedded returns only chunks whose (reach_id, chunk_type, rapid_id, access_id)
// don't already have a row in reach_embeddings.
func filterUnembedded(ctx context.Context, pool *pgxpool.Pool, reachID string, chunks []chunk) []chunk {
	// Check if this reach has any embeddings at all — fast path.
	var count int
	pool.QueryRow(ctx, `SELECT COUNT(*) FROM reach_embeddings WHERE reach_id = $1`, reachID).Scan(&count)
	if count == 0 {
		return chunks
	}

	var out []chunk
	for _, c := range chunks {
		var exists bool
		pool.QueryRow(ctx, `
			SELECT EXISTS(
				SELECT 1 FROM reach_embeddings
				WHERE reach_id   = $1
				  AND chunk_type = $2
				  AND ($3::uuid IS NULL OR rapid_id  = $3)
				  AND ($4::uuid IS NULL OR access_id = $4)
			)
		`, reachID, c.chunkType, c.rapidID, c.accessID).Scan(&exists)
		if !exists {
			out = append(out, c)
		}
	}
	return out
}

func insertEmbedding(ctx context.Context, pool *pgxpool.Pool, reachID string, c chunk, vec []float32) error {
	_, err := pool.Exec(ctx, `
		INSERT INTO reach_embeddings
			(reach_id, rapid_id, access_id, chunk_type, content, embedding)
		VALUES
			($1, $2, $3, $4, $5, $6::vector)
		ON CONFLICT DO NOTHING
	`, reachID, c.rapidID, c.accessID, c.chunkType, c.text, ai.FormatVector(vec))
	return err
}

// ── helpers ───────────────────────────────────────────────────────────────────

func mustEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		log.Fatalf("env var %s is required", key)
	}
	return v
}
