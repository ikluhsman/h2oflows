// seed-state-reaches discovers well-known whitewater runs for western US states
// using Claude AI, creates reach stubs in the database, and auto-links them to
// the nearest USGS gauge.
//
// All reach names are common geographic names — what the paddling community
// has always called these rivers. Descriptions, rapid inventories, and access
// data are generated originally by Claude. No data is copied verbatim from
// any third-party source.
//
// Usage:
//
//	go run ./cmd/seed-state-reaches/                      # all default states
//	go run ./cmd/seed-state-reaches/ CO UT AZ             # explicit states
//	FULL=true go run ./cmd/seed-state-reaches/ CO         # also seed rapids+access
//	DRY_RUN=true go run ./cmd/seed-state-reaches/         # discover without writing
//
// Env vars: DATABASE_URL, ANTHROPIC_API_KEY
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/h2oflow/h2oflow/apps/api/internal/ai"
	"github.com/h2oflow/h2oflow/apps/api/internal/db"
	"github.com/jackc/pgx/v5/pgxpool"
)

var defaultStates = []string{
	"CO", "UT", "WY", "NM", "AZ", "ID", "MT", "NV", "CA",
}

func main() {
	ctx := context.Background()

	dbURL  := mustEnv("DATABASE_URL")
	apiKey := mustEnv("ANTHROPIC_API_KEY")
	dryRun := os.Getenv("DRY_RUN") == "true"
	full   := os.Getenv("FULL") == "true"

	states := defaultStates
	if len(os.Args) > 1 {
		states = os.Args[1:]
		for i, s := range states {
			states[i] = strings.ToUpper(s)
		}
	}

	pool, err := db.Connect(ctx, dbURL)
	if err != nil {
		log.Fatalf("db: %v", err)
	}
	defer pool.Close()

	disc := ai.NewReachDiscoverer(apiKey)

	var seeder *ai.ReachSeeder
	if full {
		seeder = ai.NewReachSeeder(apiKey)
	}

	var totalInserted, totalSkipped, totalFailed int

	for _, state := range states {
		ins, skp, fail := seedState(ctx, pool, disc, seeder, state, dryRun)
		totalInserted += ins
		totalSkipped  += skp
		totalFailed   += fail
		// Brief pause between states to respect Claude rate limits
		if !dryRun {
			time.Sleep(2 * time.Second)
		}
	}

	fmt.Printf("\n══ Done ══\n")
	fmt.Printf("  inserted : %d\n", totalInserted)
	fmt.Printf("  skipped  : %d (already in DB)\n", totalSkipped)
	fmt.Printf("  failed   : %d\n", totalFailed)
}

func seedState(
	ctx context.Context,
	pool *pgxpool.Pool,
	disc *ai.ReachDiscoverer,
	seeder *ai.ReachSeeder,
	state string,
	dryRun bool,
) (inserted, skipped, failed int) {
	fmt.Printf("\n→ %s — discovering reaches via Claude...\n", state)

	reaches, err := disc.DiscoverReaches(ctx, state)
	if err != nil {
		fmt.Printf("  ERROR: %v\n", err)
		failed++
		return
	}
	fmt.Printf("  discovered %d reaches\n", len(reaches))

	for _, r := range reaches {
		slug := slugify(r.CommonName)

		if dryRun {
			fmt.Printf("  [dry] %-35s  Class %.1f–%.1f  %s  %.0f mi  gauge:%s\n",
				r.CommonName, r.ClassMin, r.ClassMax, r.Character, r.LengthMi, r.USGSGaugeID)
			inserted++
			continue
		}

		// Skip if this reach already exists
		var exists bool
		pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM reaches WHERE slug = $1)`, slug).Scan(&exists)
		if exists {
			fmt.Printf("  ○ %s — already in DB\n", r.CommonName)
			skipped++
			continue
		}

		// Upsert the reach stub
		reachID, err := upsertReach(ctx, pool, slug, r)
		if err != nil {
			fmt.Printf("  ✗ %s — upsert: %v\n", r.CommonName, err)
			failed++
			continue
		}

		// Auto-link gauge: prefer the ID Claude suggested; fall back to proximity
		if r.USGSGaugeID != "" {
			if extID, err := linkGaugeByExtID(ctx, pool, reachID, r.USGSGaugeID); err == nil {
				fmt.Printf("  ✓ %-35s  linked gauge %s\n", r.CommonName, extID)
			} else {
				fmt.Printf("  ✓ %-35s  (gauge %s not found: %v)\n", r.CommonName, r.USGSGaugeID, err)
			}
		} else if r.PutInLat != 0 && r.PutInLon != 0 {
			if extID, err := linkNearestGauge(ctx, pool, reachID, r.PutInLat, r.PutInLon); err == nil {
				fmt.Printf("  ✓ %-35s  linked nearest gauge %s\n", r.CommonName, extID)
			} else {
				fmt.Printf("  ✓ %-35s  (no nearby gauge)\n", r.CommonName)
			}
		} else {
			fmt.Printf("  ✓ %-35s  Class %.1f–%.1f %s\n", r.CommonName, r.ClassMin, r.ClassMax, r.Character)
		}

		inserted++

		// Optional full seed: rapids + access + description via Claude ReachSeeder
		if seeder != nil {
			time.Sleep(1 * time.Second) // respect Claude batch rate limit
			rc := ai.ReachContext{
				Name:       r.CommonName,
				Region:     r.SectionDesc,
				ClassMin:   r.ClassMin,
				ClassMax:   r.ClassMax,
				LengthMi:   r.LengthMi,
				PutInLat:   r.PutInLat,
				PutInLon:   r.PutInLon,
				TakeOutLat: r.TakeOutLat,
				TakeOutLon: r.TakeOutLon,
			}
			seed, err := seeder.SeedReach(ctx, rc)
			if err != nil {
				fmt.Printf("    ✗ seeder: %v\n", err)
				continue
			}
			if seed.Description != "" {
				if err := writeDescription(ctx, pool, reachID, seed); err != nil {
					fmt.Printf("    ✗ description: %v\n", err)
				}
			}
			if nRapids := writeRapids(ctx, pool, reachID, seed.Rapids); nRapids > 0 {
				fmt.Printf("    ✓ %d rapids\n", nRapids)
			}
			if nAccess := writeAccess(ctx, pool, reachID, seed.Access); nAccess > 0 {
				fmt.Printf("    ✓ %d access points\n", nAccess)
			}
		}
	}

	fmt.Printf("  inserted %d, skipped %d, failed %d\n", inserted, skipped, failed)
	return
}

// --- DB helpers --------------------------------------------------------------

func upsertReach(ctx context.Context, pool *pgxpool.Pool, slug string, r ai.DiscoveredReach) (string, error) {
	var reachID string
	err := pool.QueryRow(ctx, `
		INSERT INTO reaches (
			slug, name, region,
			class_min, class_max, character, length_mi,
			put_in, take_out
		) VALUES (
			$1, $2, $3,
			$4, $5, $6, $7,
			CASE WHEN $8::double precision IS NOT NULL AND $9::double precision IS NOT NULL
			     THEN ST_SetSRID(ST_MakePoint($9, $8), 4326)::geography ELSE NULL END,
			CASE WHEN $10::double precision IS NOT NULL AND $11::double precision IS NOT NULL
			     THEN ST_SetSRID(ST_MakePoint($11, $10), 4326)::geography ELSE NULL END
		)
		ON CONFLICT (slug) DO NOTHING
		RETURNING id
	`,
		slug, r.CommonName, r.SectionDesc,
		nullFloat(r.ClassMin), nullFloat(r.ClassMax),
		nullStr(r.Character), nullFloat(r.LengthMi),
		nullFloat(r.PutInLat), nullFloat(r.PutInLon),
		nullFloat(r.TakeOutLat), nullFloat(r.TakeOutLon),
	).Scan(&reachID)
	if err != nil {
		return "", err
	}
	if reachID == "" {
		return "", fmt.Errorf("conflict — slug %q already exists", slug)
	}
	return reachID, nil
}

// linkGaugeByExtID links a reach to a USGS gauge by its external_id.
// Sets reach_id on the gauge and primary_gauge_id on the reach (both sides of the FK).
func linkGaugeByExtID(ctx context.Context, pool *pgxpool.Pool, reachID, externalID string) (string, error) {
	tag, err := pool.Exec(ctx, `
		UPDATE gauges SET reach_id = $1
		WHERE external_id = $2 AND source = 'usgs' AND reach_id IS NULL
	`, reachID, externalID)
	if err != nil {
		return "", err
	}
	if tag.RowsAffected() == 0 {
		return "", fmt.Errorf("gauge %s not found or already linked", externalID)
	}
	pool.Exec(ctx, `
		UPDATE reaches SET primary_gauge_id = (
			SELECT id FROM gauges WHERE external_id = $2 AND source = 'usgs' LIMIT 1
		) WHERE id = $1 AND primary_gauge_id IS NULL
	`, reachID, externalID)
	return externalID, nil
}

// linkNearestGauge finds the closest unlinked USGS gauge within 30 km of the
// put-in and links it to the reach. Returns the gauge's external_id.
func linkNearestGauge(ctx context.Context, pool *pgxpool.Pool, reachID string, lat, lon float64) (string, error) {
	var gaugeID, extID string
	err := pool.QueryRow(ctx, `
		SELECT id, external_id
		FROM gauges
		WHERE source = 'usgs'
		  AND reach_id IS NULL
		  AND location IS NOT NULL
		  AND ST_DWithin(location::geography, ST_MakePoint($2, $1)::geography, 30000)
		ORDER BY ST_Distance(location::geography, ST_MakePoint($2, $1)::geography)
		LIMIT 1
	`, lat, lon).Scan(&gaugeID, &extID)
	if err != nil {
		return "", err
	}

	pool.Exec(ctx, `UPDATE gauges SET reach_id = $1 WHERE id = $2 AND reach_id IS NULL`, reachID, gaugeID)
	pool.Exec(ctx, `UPDATE reaches SET primary_gauge_id = $2 WHERE id = $1 AND primary_gauge_id IS NULL`, reachID, gaugeID)
	return extID, nil
}

func writeDescription(ctx context.Context, pool *pgxpool.Pool, reachID string, seed *ai.ReachSeed) error {
	_, err := pool.Exec(ctx, `
		UPDATE reaches SET
			description               = $2,
			description_source        = 'ai_seed',
			description_ai_confidence = $3,
			description_verified      = $4,
			description_updated_at    = NOW()
		WHERE id = $1
	`, reachID, seed.Description, seed.DescriptionConfidence, seed.DescriptionAutoVerified())
	return err
}

func writeRapids(ctx context.Context, pool *pgxpool.Pool, reachID string, rapids []ai.RapidSeed) (written int) {
	for _, r := range rapids {
		_, err := pool.Exec(ctx, `
			INSERT INTO rapids
				(reach_id, name, river_mile, class_rating, class_at_low, class_at_high,
				 description, portage_description, is_portage_recommended,
				 data_source, ai_confidence, verified)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,'ai_seed',$10,$11)
			ON CONFLICT DO NOTHING
		`,
			reachID, r.Name, r.RiverMile, r.ClassRating, r.ClassAtLow, r.ClassAtHigh,
			nullStr(r.Description), nullStr(r.PortageDescription), r.IsPortageRecommended,
			r.Confidence, r.AutoVerified(),
		)
		if err == nil {
			written++
		}
	}
	return
}

func writeAccess(ctx context.Context, pool *pgxpool.Pool, reachID string, access []ai.AccessSeed) (written int) {
	for _, a := range access {
		var accessID string
		err := pool.QueryRow(ctx, `
			INSERT INTO reach_access
				(reach_id, access_type, name, directions, road_type,
				 parking_fee, permit_required, permit_info, permit_url,
				 seasonal_close_start, seasonal_close_end, notes,
				 entry_style, approach_dist_mi, approach_notes,
				 data_source, ai_confidence, verified)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,'ai_seed',$16,$17)
			RETURNING id
		`,
			reachID, a.AccessType, nullStr(a.Name), nullStr(a.Directions), nullStr(a.RoadType),
			a.ParkingFee, a.PermitRequired, nullStr(a.PermitInfo), nullStr(a.PermitURL),
			nullStr(a.SeasonalCloseStart), nullStr(a.SeasonalCloseEnd), nullStr(a.Notes),
			nullStr(a.EntryStyle), a.ApproachDistMi, nullStr(a.ApproachNotes),
			a.Confidence, a.AutoVerified(),
		).Scan(&accessID)
		if err != nil {
			continue
		}
		written++
		if a.WaterLat != nil && a.WaterLon != nil {
			pool.Exec(ctx, `UPDATE reach_access SET location = ST_MakePoint($2,$3)::geography WHERE id = $1`,
				accessID, *a.WaterLon, *a.WaterLat)
		}
		if a.ParkingLat != nil && a.ParkingLon != nil {
			pool.Exec(ctx, `UPDATE reach_access SET parking_location = ST_MakePoint($2,$3)::geography WHERE id = $1`,
				accessID, *a.ParkingLon, *a.ParkingLat)
		}
	}
	return
}

// --- Utilities ---------------------------------------------------------------

var slugRe = regexp.MustCompile(`[^a-z0-9]+`)

// slugify converts a reach name to a URL-safe slug.
// "Browns Canyon" → "browns-canyon"
// "Rio Grande — Taos Box" → "rio-grande-taos-box"
func slugify(name string) string {
	s := strings.ToLower(name)
	s = slugRe.ReplaceAllString(s, "-")
	s = strings.Trim(s, "-")
	return s
}

func nullStr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func nullFloat(f float64) *float64 {
	if f == 0 {
		return nil
	}
	return &f
}

func mustEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		log.Fatalf("required env var not set: %s", key)
	}
	return v
}
