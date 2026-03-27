// seed-flow-ranges queries Claude for paddling flow range data for every
// featured (trusted) gauge that has an associated reach but no verified ranges yet.
//
// Run once after seeding reaches, or re-run safely — it never overwrites verified rows.
//
//	go run ./cmd/seed-flow-ranges
//
// Reads the same env vars as the API server: DATABASE_URL, ANTHROPIC_API_KEY.
// Set DRY_RUN=true to print results without writing to the database.
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/h2oflow/h2oflow/apps/api/internal/ai"
	"github.com/h2oflow/h2oflow/apps/api/internal/db"
)

func main() {
	ctx := context.Background()

	dbURL := mustEnv("DATABASE_URL")
	apiKey := mustEnv("ANTHROPIC_API_KEY")
	dryRun := os.Getenv("DRY_RUN") == "true"

	pool, err := db.Connect(ctx, dbURL)
	if err != nil {
		log.Fatalf("db: %v", err)
	}
	defer pool.Close()

	seeder := ai.NewFlowRangeSeeder(apiKey)

	// Load all featured gauges that have an associated reach but no verified flow ranges.
	// We skip gauges with verified=true ranges — human entries win, always.
	rows, err := pool.Query(ctx, `
		SELECT
			g.id,
			g.name,
			g.external_id,
			g.source,
			r.id        AS reach_id,
			r.name      AS reach_name,
			r.region,
			r.class_min,
			r.class_max,
			r.length_mi,
			r.aw_reach_id
		FROM gauges g
		JOIN reaches r ON r.id = g.reach_id
		WHERE g.featured = TRUE
		  AND NOT EXISTS (
			SELECT 1 FROM flow_ranges fr
			WHERE fr.gauge_id = g.id AND fr.verified = TRUE
		  )
		ORDER BY g.prominence_score DESC
	`)
	if err != nil {
		log.Fatalf("query: %v", err)
	}

	type target struct {
		GaugeID    string
		GaugeName  string
		ExternalID string
		Source     string
		ReachName  string
		Region     string
		ClassMin   float64
		ClassMax   float64
		LengthMi   float64
		AWReachID  *string // nullable — most reaches won't have this populated yet
	}

	var targets []target
	for rows.Next() {
		var t target
		var reachID string // unused beyond join
		if err := rows.Scan(
			&t.GaugeID, &t.GaugeName, &t.ExternalID, &t.Source,
			&reachID, &t.ReachName, &t.Region,
			&t.ClassMin, &t.ClassMax, &t.LengthMi, &t.AWReachID,
		); err != nil {
			log.Printf("scan: %v", err)
			continue
		}
		targets = append(targets, t)
	}
	rows.Close()

	if len(targets) == 0 {
		fmt.Println("No gauges to seed — all featured gauges either lack a reach association or already have verified flow ranges.")
		return
	}

	fmt.Printf("Seeding flow ranges for %d gauge(s)%s\n", len(targets), func() string {
		if dryRun {
			return " [DRY RUN — no writes]"
		}
		return ""
	}())

	var seeded, skipped, failed int

	for _, t := range targets {
		fmt.Printf("\n→ %s (%s %s) — %s\n", t.GaugeName, t.Source, t.ExternalID, t.ReachName)

		fc := ai.FlowRangeContext{
			GaugeName:   t.GaugeName,
			ExternalID:  t.ExternalID,
			Source:      t.Source,
			ReachName:   t.ReachName,
			ReachRegion: t.Region,
			ClassMin:    t.ClassMin,
			ClassMax:    t.ClassMax,
			LengthMi:    t.LengthMi,
			AWReachID:   derefStr(t.AWReachID),
		}

		seeds, err := seeder.SeedFlowRanges(ctx, fc)
		if err != nil {
			fmt.Printf("  ✗ seeder error: %v\n", err)
			failed++
			continue
		}
		if len(seeds) == 0 {
			fmt.Printf("  ○ no ranges above confidence floor (%.0f) — skipping\n", float64(50))
			skipped++
			continue
		}

		for _, s := range seeds {
			verified := s.AutoVerified()
			flag := "draft"
			if verified {
				flag = "auto-verified"
			}
			fmt.Printf("  %-10s  %6.0f–%-6s  craft=%-8s  conf=%d  %s  %s\n",
				s.Label,
				derefF(s.MinCFS),
				formatMax(s.MaxCFS),
				s.CraftType,
				s.Confidence,
				flag,
				s.SourceURL,
			)
		}

		if dryRun {
			seeded++
			continue
		}

		if err := writeFlowRanges(ctx, pool, t.GaugeID, seeds, seeder.DataSource()); err != nil {
			fmt.Printf("  ✗ write error: %v\n", err)
			failed++
			continue
		}
		seeded++
		// Polite rate limiting — Claude allows burst but this is batch work.
		time.Sleep(500 * time.Millisecond)
	}

	fmt.Printf("\nDone: %d seeded, %d skipped (low confidence), %d failed\n", seeded, skipped, failed)
}

// writeFlowRanges inserts AI-seeded flow ranges.
// ON CONFLICT DO NOTHING — verified manual entries are never overwritten.
func writeFlowRanges(ctx context.Context, pool *pgxpool.Pool, gaugeID string, seeds []ai.FlowRangeSeed, dataSource string) error {
	for _, s := range seeds {
		_, err := pool.Exec(ctx, `
			INSERT INTO flow_ranges
				(gauge_id, label, min_cfs, max_cfs, craft_type, class_modifier,
				 source_url, ai_confidence, data_source, verified)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
			ON CONFLICT (gauge_id, label, craft_type) DO NOTHING
		`,
			gaugeID,
			s.Label,
			s.MinCFS,
			s.MaxCFS,
			s.CraftType,
			s.ClassMod,
			nullStr(s.SourceURL),
			s.Confidence,
			dataSource,
			s.AutoVerified(),
		)
		if err != nil {
			return fmt.Errorf("insert %s: %w", s.Label, err)
		}
	}
	return nil
}

// --- Helpers ----------------------------------------------------------------

func mustEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		log.Fatalf("required env var %s is not set", key)
	}
	return v
}

func derefF(f *float64) float64 {
	if f == nil {
		return 0
	}
	return *f
}

func formatMax(f *float64) string {
	if f == nil {
		return "∞    "
	}
	return fmt.Sprintf("%-5.0f", *f)
}

func nullStr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func derefStr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
