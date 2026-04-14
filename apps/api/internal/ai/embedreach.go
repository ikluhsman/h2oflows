// embedreach.go — reusable logic for generating and storing reach embeddings.
//
// Used by both cmd/embed-reaches (full batch re-embed) and the KMZ import
// handler (auto-embed newly imported reaches in the background).
//
// Key design choices:
//   - Reaches without descriptions ARE included — rapids and access points
//     with only a name and coordinates still produce useful chunks.
//   - Rapids without description text ARE included — the name, class, and
//     river mile alone are enough for the AI to answer "where is Rapid 10?"
//   - Access points without directions ARE included — name + type let the
//     AI answer "what's the put-in for X?"
//   - Already-embedded chunks are skipped (ON CONFLICT DO NOTHING).
//   - Rate-limited at ~22s per reach to respect Voyage free tier (3 RPM).
package ai

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// EmbedReachesAll embeds every reach in the database. Idempotent — already-
// embedded chunks are skipped unless reembed is true (which first wipes all
// existing embeddings).
func EmbedReachesAll(ctx context.Context, pool *pgxpool.Pool, embedder *Embedder, reembed bool) (embedded, skipped int, err error) {
	if reembed {
		if _, err := pool.Exec(ctx, `DELETE FROM reach_embeddings`); err != nil {
			return 0, 0, fmt.Errorf("delete embeddings: %w", err)
		}
	}
	rows, err := pool.Query(ctx, `SELECT id FROM reaches ORDER BY name`)
	if err != nil {
		return 0, 0, fmt.Errorf("list reaches: %w", err)
	}
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return 0, 0, err
		}
		ids = append(ids, id)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return 0, 0, err
	}
	return EmbedReaches(ctx, pool, embedder, ids, true)
}

// EmbedReaches embeds a specific set of reaches (by ID). Used after KMZ import
// to embed only the affected reaches without rate-limiting the full corpus.
// When rateLimit is false the per-reach sleep is skipped (useful in tests or
// when the caller already manages concurrency).
func EmbedReaches(ctx context.Context, pool *pgxpool.Pool, embedder *Embedder, ids []string, rateLimit bool) (embedded, skipped int, err error) {
	for i, id := range ids {
		r, loadErr := loadEmbedReach(ctx, pool, id)
		if loadErr != nil {
			return embedded, skipped, fmt.Errorf("load reach %s: %w", id, loadErr)
		}

		chunks := buildEmbedChunks(r)
		if len(chunks) == 0 {
			skipped++
			continue
		}

		toEmbed := filterUnembeddedChunks(ctx, pool, id, chunks)
		skipped += len(chunks) - len(toEmbed)
		if len(toEmbed) == 0 {
			continue
		}

		// Embed in batches of 100.
		for j := 0; j < len(toEmbed); j += 100 {
			end := j + 100
			if end > len(toEmbed) {
				end = len(toEmbed)
			}
			batch := toEmbed[j:end]
			texts := make([]string, len(batch))
			for k, c := range batch {
				texts[k] = c.text
			}
			vecs, embedErr := embedder.Embed(ctx, texts)
			if embedErr != nil {
				return embedded, skipped, fmt.Errorf("embed reach %s: %w", id, embedErr)
			}
			for k, c := range batch {
				if vecs[k] == nil {
					continue
				}
				if insErr := insertEmbeddingRow(ctx, pool, id, c, vecs[k]); insErr != nil {
					return embedded, skipped, fmt.Errorf("insert chunk for reach %s: %w", id, insErr)
				}
				embedded++
			}
		}

		// Stay under Voyage free-tier 3 RPM between reaches (not after the last one).
		if rateLimit && i < len(ids)-1 {
			select {
			case <-ctx.Done():
				return embedded, skipped, ctx.Err()
			case <-time.After(22 * time.Second):
			}
		}
	}
	return embedded, skipped, nil
}

// ── internal data model ───────────────────────────────────────────────────────

type embedReachRow struct {
	id          string
	name        string
	region      string
	classMin    *float64
	classMax    *float64
	lengthMi    *float64
	description *string
	rapids      []embedRapidRow
	access      []embedAccessRow
	flowRanges  []embedFlowRangeRow
}

type embedRapidRow struct {
	id          string
	name        string
	classRating *float64
	riverMile   *float64
	description *string
	portageDesc *string
}

type embedAccessRow struct {
	id         string
	accessType string
	name       *string
	directions *string
	notes      *string
}

type embedFlowRangeRow struct {
	label  string
	minCFS *float64
	maxCFS *float64
}

type embedChunk struct {
	chunkType string
	rapidID   *string
	accessID  *string
	text      string
}

// ── loaders ───────────────────────────────────────────────────────────────────

func loadEmbedReach(ctx context.Context, pool *pgxpool.Pool, id string) (embedReachRow, error) {
	var r embedReachRow
	r.id = id
	err := pool.QueryRow(ctx, `
		SELECT name, COALESCE(region,''), class_min, class_max, length_mi, description
		FROM reaches WHERE id = $1
	`, id).Scan(&r.name, &r.region, &r.classMin, &r.classMax, &r.lengthMi, &r.description)
	if err != nil {
		return r, err
	}

	// Rapids — include all, even those without description text.
	rapRows, err := pool.Query(ctx, `
		SELECT id, name, class_rating, river_mile, description, portage_description
		FROM rapids WHERE reach_id = $1 ORDER BY river_mile NULLS LAST, name
	`, id)
	if err != nil {
		return r, err
	}
	defer rapRows.Close()
	for rapRows.Next() {
		var rr embedRapidRow
		if err := rapRows.Scan(&rr.id, &rr.name, &rr.classRating, &rr.riverMile, &rr.description, &rr.portageDesc); err != nil {
			return r, err
		}
		r.rapids = append(r.rapids, rr)
	}
	if err := rapRows.Err(); err != nil {
		return r, err
	}

	// Access points — include all, even those without directions.
	accRows, err := pool.Query(ctx, `
		SELECT id, access_type, name, directions, notes
		FROM reach_access WHERE reach_id = $1 ORDER BY access_type, name
	`, id)
	if err != nil {
		return r, err
	}
	defer accRows.Close()
	for accRows.Next() {
		var a embedAccessRow
		if err := accRows.Scan(&a.id, &a.accessType, &a.name, &a.directions, &a.notes); err != nil {
			return r, err
		}
		r.access = append(r.access, a)
	}
	if err := accRows.Err(); err != nil {
		return r, err
	}

	// Flow ranges via primary gauge.
	frRows, err := pool.Query(ctx, `
		SELECT fr.label, fr.min_cfs, fr.max_cfs
		FROM flow_ranges fr
		JOIN reaches rc ON rc.primary_gauge_id = fr.gauge_id
		WHERE rc.id = $1 AND fr.craft_type = 'general'
		ORDER BY CASE fr.label
			WHEN 'below_recommended' THEN 1
			WHEN 'low_runnable'      THEN 2
			WHEN 'runnable'          THEN 3
			WHEN 'med_runnable'      THEN 4
			WHEN 'high_runnable'     THEN 5
			WHEN 'above_recommended' THEN 6
			ELSE 7
		END
	`, id)
	if err != nil {
		return r, err
	}
	defer frRows.Close()
	for frRows.Next() {
		var fr embedFlowRangeRow
		if err := frRows.Scan(&fr.label, &fr.minCFS, &fr.maxCFS); err != nil {
			return r, err
		}
		r.flowRanges = append(r.flowRanges, fr)
	}
	return r, frRows.Err()
}

// ── chunk builders ────────────────────────────────────────────────────────────

func buildEmbedChunks(r embedReachRow) []embedChunk {
	var chunks []embedChunk

	// Reach description chunk — only when description is set.
	if r.description != nil && *r.description != "" {
		chunks = append(chunks, embedChunk{
			chunkType: "reach_description",
			text:      buildDescChunk(r),
		})
	}

	// Rapid chunks — always emit, even with no description text.
	// A chunk with just "Cataract Canyon – Rapid 10 (Class IV, mile 5.2)" is
	// enough for the AI to answer location questions.
	for _, rr := range r.rapids {
		if txt := buildRapidChunk(r.name, rr); txt != "" {
			id := rr.id
			chunks = append(chunks, embedChunk{
				chunkType: "rapid",
				rapidID:   &id,
				text:      txt,
			})
		}
	}

	// Access chunks — always emit when there is at least a name or type.
	for _, a := range r.access {
		if txt := buildAccessChunk(r.name, a); txt != "" {
			id := a.id
			chunks = append(chunks, embedChunk{
				chunkType: "access_point",
				accessID:  &id,
				text:      txt,
			})
		}
	}

	// Flow ranges — one summary chunk.
	if len(r.flowRanges) > 0 {
		chunks = append(chunks, embedChunk{
			chunkType: "flow_ranges",
			text:      buildFlowRangesChunk(r),
		})
	}

	return chunks
}

func buildDescChunk(r embedReachRow) string {
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

func buildRapidChunk(reachName string, rr embedRapidRow) string {
	// Always emit at minimum: "Reach – Rapid Name"
	// That's enough for the AI to answer "where is Rapid 10?"
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

func buildAccessChunk(reachName string, a embedAccessRow) string {
	// Always emit at minimum: "Reach – access type: Name"
	name := derefStr(a.name, "")
	dirs := derefStr(a.directions, "")
	notes := derefStr(a.notes, "")

	// Skip if there is genuinely nothing to say (no name, no directions, no notes).
	if name == "" && dirs == "" && notes == "" {
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

func buildFlowRangesChunk(r embedReachRow) string {
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

func derefStr(s *string, fallback string) string {
	if s == nil {
		return fallback
	}
	return *s
}

// ── DB helpers ────────────────────────────────────────────────────────────────

func filterUnembeddedChunks(ctx context.Context, pool *pgxpool.Pool, reachID string, chunks []embedChunk) []embedChunk {
	var count int
	pool.QueryRow(ctx, `SELECT COUNT(*) FROM reach_embeddings WHERE reach_id = $1`, reachID).Scan(&count)
	if count == 0 {
		return chunks
	}
	var out []embedChunk
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

func insertEmbeddingRow(ctx context.Context, pool *pgxpool.Pool, reachID string, c embedChunk, vec []float32) error {
	_, err := pool.Exec(ctx, `
		INSERT INTO reach_embeddings
			(reach_id, rapid_id, access_id, chunk_type, content, embedding)
		VALUES
			($1, $2, $3, $4, $5, $6::vector)
		ON CONFLICT DO NOTHING
	`, reachID, c.rapidID, c.accessID, c.chunkType, c.text, FormatVector(vec))
	return err
}
