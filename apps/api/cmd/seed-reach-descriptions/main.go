// seed-reach-descriptions backfills prose descriptions on reaches that are
// missing one. Useful after importing reaches via KMZ (which only seeds
// geometry, rapids, and access points), or for any reach that has rapids
// and access but no description yet.
//
// It calls Claude (via ai.ReachSeeder.SeedDescription) for each reach, then
// writes the result to reaches.description plus the description_* metadata
// columns. Existing descriptions are left alone unless RESEED=true.
//
//	go run ./cmd/seed-reach-descriptions
//	go run ./cmd/seed-reach-descriptions -slug arkansas-numbers
//	DRY_RUN=true go run ./cmd/seed-reach-descriptions
//	RESEED=true  go run ./cmd/seed-reach-descriptions -slug arkansas-numbers
//
// Env vars: DATABASE_URL, ANTHROPIC_API_KEY
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/h2oflow/h2oflow/apps/api/internal/ai"
	"github.com/h2oflow/h2oflow/apps/api/internal/db"
)

func main() {
	slug := flag.String("slug", "", "Only seed the reach with this slug (default: all reaches missing a description)")
	flag.Parse()

	ctx := context.Background()

	dbURL := mustEnv("DATABASE_URL")
	apiKey := mustEnv("ANTHROPIC_API_KEY")
	dryRun := os.Getenv("DRY_RUN") == "true"
	reseed := os.Getenv("RESEED") == "true"

	pool, err := db.Connect(ctx, dbURL)
	if err != nil {
		log.Fatalf("db: %v", err)
	}
	defer pool.Close()

	targets, err := loadTargets(ctx, pool, *slug, reseed)
	if err != nil {
		log.Fatalf("load targets: %v", err)
	}

	if len(targets) == 0 {
		if *slug != "" {
			fmt.Printf("No reach found for slug %q (or it already has a description; use RESEED=true to overwrite).\n", *slug)
		} else {
			fmt.Println("No reaches missing descriptions.")
		}
		return
	}

	fmt.Printf("Seeding descriptions for %d reach(es)%s\n",
		len(targets),
		ifThen(dryRun, " [DRY RUN — no writes]", ""),
	)

	seeder := ai.NewReachSeeder(apiKey)

	var seeded, skipped, failed int
	for _, t := range targets {
		fmt.Printf("\n→ %s (%s)\n", t.Name, t.Slug)

		rc := ai.ReachContext{
			Name:       t.Name,
			Region:     t.Region,
			ClassMin:   t.ClassMin,
			ClassMax:   t.ClassMax,
			LengthMi:   t.LengthMi,
			PutInLat:   t.PutInLat,
			PutInLon:   t.PutInLon,
			TakeOutLat: t.TakeOutLat,
			TakeOutLon: t.TakeOutLon,
		}

		fmt.Printf("  ◌ calling Claude…\n")
		seed, err := seeder.SeedDescription(ctx, rc)
		if err != nil {
			fmt.Printf("  ✗ seeder error: %v\n", err)
			failed++
			continue
		}
		if seed.Description == "" || seed.Confidence < 50 {
			fmt.Printf("  ○ low confidence (%d) — skipping\n", seed.Confidence)
			skipped++
			continue
		}

		flag := "draft"
		if seed.AutoVerified() {
			flag = "auto-verified"
		}
		preview := strings.SplitN(seed.Description, "\n", 2)[0]
		if len(preview) > 100 {
			preview = preview[:97] + "..."
		}
		fmt.Printf("  ✓ description (conf=%d, %s): %s\n", seed.Confidence, flag, preview)

		if dryRun {
			seeded++
			continue
		}

		if err := writeDescription(ctx, pool, t.ID, seed); err != nil {
			fmt.Printf("  ✗ write: %v\n", err)
			failed++
			continue
		}
		seeded++
		// Polite rate limiting — Claude allows burst but this is batch work.
		time.Sleep(500 * time.Millisecond)
	}

	fmt.Printf("\nDone: %d seeded, %d skipped (low confidence), %d failed\n", seeded, skipped, failed)
}

// target is one reach to seed.
type target struct {
	ID         string
	Slug       string
	Name       string
	Region     string
	ClassMin   float64
	ClassMax   float64
	LengthMi   float64
	PutInLat   float64
	PutInLon   float64
	TakeOutLat float64
	TakeOutLon float64
}

// loadTargets pulls reaches that are missing a description (or all reaches
// matching the given slug, optionally re-seeding existing descriptions).
//
// We pull put-in/take-out coordinates from reach_access when present so the
// AI seeder has geographic anchors to work from.
func loadTargets(ctx context.Context, pool *pgxpool.Pool, slug string, reseed bool) ([]target, error) {
	var (
		rows pgx.Rows
		err  error
	)

	base := `
		SELECT
			r.id,
			r.slug,
			r.name,
			COALESCE(r.region, '')                AS region,
			COALESCE(r.class_min, 0)              AS class_min,
			COALESCE(r.class_max, 0)              AS class_max,
			COALESCE(r.length_mi, 0)              AS length_mi,
			COALESCE(ST_Y(pi.location::geometry), 0) AS put_in_lat,
			COALESCE(ST_X(pi.location::geometry), 0) AS put_in_lon,
			COALESCE(ST_Y(to_.location::geometry), 0) AS take_out_lat,
			COALESCE(ST_X(to_.location::geometry), 0) AS take_out_lon
		FROM reaches r
		LEFT JOIN LATERAL (
			SELECT location FROM reach_access
			WHERE reach_id = r.id AND access_type = 'put_in' AND location IS NOT NULL
			ORDER BY verified DESC, id LIMIT 1
		) pi ON TRUE
		LEFT JOIN LATERAL (
			SELECT location FROM reach_access
			WHERE reach_id = r.id AND access_type = 'take_out' AND location IS NOT NULL
			ORDER BY verified DESC, id LIMIT 1
		) to_ ON TRUE
	`

	switch {
	case slug != "":
		rows, err = pool.Query(ctx, base+` WHERE r.slug = $1`, slug)
	case reseed:
		rows, err = pool.Query(ctx, base+` ORDER BY r.name`)
	default:
		rows, err = pool.Query(ctx, base+`
			WHERE r.description IS NULL OR TRIM(r.description) = ''
			ORDER BY r.name
		`)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []target
	for rows.Next() {
		var t target
		if err := rows.Scan(
			&t.ID, &t.Slug, &t.Name, &t.Region,
			&t.ClassMin, &t.ClassMax, &t.LengthMi,
			&t.PutInLat, &t.PutInLon,
			&t.TakeOutLat, &t.TakeOutLon,
		); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

// writeDescription stores the AI-generated description on the reach row.
// Mirrors writeDescription in cmd/seed-reaches/write.go so the metadata columns
// stay consistent across both commands.
func writeDescription(ctx context.Context, pool *pgxpool.Pool, reachID string, seed *ai.DescriptionSeed) error {
	_, err := pool.Exec(ctx, `
		UPDATE reaches SET
			description               = $2,
			description_source        = 'ai_seed',
			description_ai_confidence = $3,
			description_verified      = $4,
			description_updated_at    = NOW()
		WHERE id = $1
	`,
		reachID,
		seed.Description,
		seed.Confidence,
		seed.AutoVerified(),
	)
	return err
}

// --- Helpers ----------------------------------------------------------------

func mustEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		log.Fatalf("required env var %s is not set", key)
	}
	return v
}

func ifThen(cond bool, yes, no string) string {
	if cond {
		return yes
	}
	return no
}
