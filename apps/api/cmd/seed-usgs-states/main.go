// seed-usgs-states bulk-inserts USGS streamflow gauge sites for a set of
// western US states into the gauges table.
//
// Only gauges that don't already exist are inserted (ON CONFLICT DO NOTHING),
// so existing trusted/featured gauges are never overwritten.
// All new gauges start as cold-tier (featured=false, prominence_score=0).
//
// Usage:
//
//	go run ./cmd/seed-usgs-states/                    # default state list
//	go run ./cmd/seed-usgs-states/ CO UT AZ           # explicit states
//	DRY_RUN=true go run ./cmd/seed-usgs-states/       # count without writing
//
// Env vars: DATABASE_URL
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	gauge "github.com/h2oflow/h2oflow/packages/gauge-core"
	"github.com/h2oflow/h2oflow/apps/api/internal/db"
	"github.com/jackc/pgx/v5/pgxpool"
)

// defaultStates is the western paddling corridor — where Colorado-based
// paddlers commonly travel. Plains states (KS, NE, TX, OK) excluded.
var defaultStates = []string{
	"CO", // home state
	"UT", // Colorado Plateau, Green River, San Juan
	"WY", // upper Green, Snake headwaters, Shoshone
	"NM", // Rio Grande, Taos Box, Chama
	"AZ", // Grand Canyon / Lee's Ferry, Salt River, Verde
	"ID", // Salmon, Payette, Lochsa, Snake
	"MT", // Gallatin, Yellowstone, Clark Fork
	"NV", // Black Canyon / Colorado border
	"CA", // Kern, Kings, American, Tuolumne
}

func main() {
	ctx := context.Background()

	dbURL  := mustEnv("DATABASE_URL")
	dryRun := os.Getenv("DRY_RUN") == "true"

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

	src := gauge.NewUSGSSource(os.Getenv("USGS_API_KEY")) // key is optional

	var totalInserted, totalSkipped, totalFailed int

	for _, state := range states {
		inserted, skipped, failed := seedState(ctx, pool, src, state, dryRun)
		totalInserted += inserted
		totalSkipped  += skipped
		totalFailed   += failed
	}

	fmt.Printf("\n══ Done ══\n")
	fmt.Printf("  inserted : %d\n", totalInserted)
	fmt.Printf("  skipped  : %d (already in DB)\n", totalSkipped)
	fmt.Printf("  failed   : %d\n", totalFailed)
}

func seedState(ctx context.Context, pool *pgxpool.Pool, src *gauge.USGSSource, state string, dryRun bool) (inserted, skipped, failed int) {
	fmt.Printf("\n→ %s — discovering sites...\n", state)

	// Give USGS a generous timeout per state — some states have 400+ gauges.
	fetchCtx, cancel := context.WithTimeout(ctx, 90*time.Second)
	defer cancel()

	sites, err := src.DiscoverSites(fetchCtx, gauge.DiscoverOptions{
		StateCodes: []string{state},
		ActiveOnly: true,
	})
	if err != nil {
		fmt.Printf("  ERROR fetching %s: %v\n", state, err)
		failed++
		return
	}

	fmt.Printf("  found %d active discharge sites\n", len(sites))

	if dryRun {
		inserted = len(sites)
		return
	}

	for _, site := range sites {
		ok, err := upsertGauge(ctx, pool, site)
		if err != nil {
			fmt.Printf("  WARN: %s — %v\n", site.ExternalID, err)
			failed++
			continue
		}
		if ok {
			inserted++
		} else {
			skipped++
		}
	}

	fmt.Printf("  inserted %d, skipped %d (existing), failed %d\n", inserted, skipped, failed)
	return
}

// upsertGauge inserts a gauge if it doesn't already exist.
// Returns true if a new row was inserted, false if it already existed.
func upsertGauge(ctx context.Context, pool *pgxpool.Pool, site *gauge.SiteMetadata) (bool, error) {
	var lng, lat *float64
	if site.Location != nil {
		lng = &site.Location.Lng
		lat = &site.Location.Lat
	}

	var huc8 *string
	if site.HUCCode != "" {
		huc8 = &site.HUCCode
	}

	var stateAbbr *string
	if site.StateCode != "" {
		stateAbbr = &site.StateCode
	}

	tag, err := pool.Exec(ctx, `
		INSERT INTO gauges (
			external_id, source, name, location,
			state_abbr, huc8,
			status, featured, prominence_score, auto_managed
		) VALUES (
			$1, 'usgs', $2,
			CASE WHEN $3::double precision IS NOT NULL AND $4::double precision IS NOT NULL
			     THEN ST_SetSRID(ST_MakePoint($3, $4), 4326)::geography
			     ELSE NULL END,
			$5, $6,
			'active', false, 0, true
		)
		ON CONFLICT (external_id, source) DO NOTHING
	`, site.ExternalID, site.Name, lng, lat, stateAbbr, huc8)

	if err != nil {
		return false, err
	}
	return tag.RowsAffected() == 1, nil
}

func mustEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		log.Fatalf("required env var not set: %s", key)
	}
	return v
}
