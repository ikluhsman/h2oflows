// seed-reaches creates reach stubs for classic whitewater runs, links them to
// their featured gauges, and calls ReachSeeder to populate rapids, access
// points, and descriptions via Claude.
//
// Run order: seed-reaches → seed-flow-ranges
// (flow ranges need reach associations to exist before they can be seeded)
//
//	go run ./cmd/seed-reaches
//	DRY_RUN=true go run ./cmd/seed-reaches   # print without writing
//
// Env vars: DATABASE_URL, ANTHROPIC_API_KEY
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/h2oflow/h2oflow/apps/api/internal/ai"
	"github.com/h2oflow/h2oflow/apps/api/internal/db"
)

func main() {
	ctx := context.Background()

	dbURL  := mustEnv("DATABASE_URL")
	apiKey := mustEnv("ANTHROPIC_API_KEY")
	dryRun := os.Getenv("DRY_RUN") == "true"

	pool, err := db.Connect(ctx, dbURL)
	if err != nil {
		log.Fatalf("db: %v", err)
	}
	defer pool.Close()

	seeder := ai.NewReachSeeder(apiKey)

	var seeded, skipped, failed int

	for _, rd := range reaches {
		fmt.Printf("\n→ %s (%s)\n", rd.Name, rd.Region)

		if dryRun {
			fmt.Printf("  [dry run] would seed reach and link gauge %s/%s\n", rd.GaugeSource, rd.GaugeExtID)
			seeded++
			continue
		}

		// 1. Upsert the reach stub.
		reachID, err := upsertReach(ctx, pool, rd)
		if err != nil {
			fmt.Printf("  ✗ upsert reach: %v\n", err)
			failed++
			continue
		}

		// 2. Link the primary gauge ↔ reach (both sides of the FK pair).
		if err := linkGauge(ctx, pool, reachID, rd.GaugeExtID, rd.GaugeSource); err != nil {
			fmt.Printf("  ✗ link gauge: %v\n", err)
			// Non-fatal — the reach was created, we can fix the link manually.
		}

		// 2b. Link any additional related gauges (upstream/downstream indicators, tributaries).
		for _, ga := range rd.RelatedGauges {
			if err := linkRelatedGauge(ctx, pool, reachID, ga); err != nil {
				fmt.Printf("  ✗ link %s gauge %s: %v\n", ga.Relationship, ga.ExtID, err)
			} else {
				fmt.Printf("  ✓ %s gauge %s/%s\n", ga.Relationship, ga.Source, ga.ExtID)
			}
		}

		// 3. Write any domain-expert flow ranges we already know (before AI seeding).
		for _, fr := range rd.KnownFlowRanges {
			if err := writeKnownFlowRange(ctx, pool, rd.GaugeExtID, rd.GaugeSource, fr); err != nil {
				fmt.Printf("  ✗ flow range %s: %v\n", fr.Label, err)
			} else {
				fmt.Printf("  ✓ flow range %s (manual, verified)\n", fr.Label)
			}
		}

		// 4. Run ReachSeeder — skip description/rapids/access if already seeded.
		// Flow ranges are seeded independently (step 8) so they always run.
		// Override with RESEED=true to force full re-seeding.
		var existingDesc *string
		pool.QueryRow(ctx, `SELECT description FROM reaches WHERE id = $1`, reachID).Scan(&existingDesc)
		alreadySeeded := existingDesc != nil && os.Getenv("RESEED") != "true"

		if alreadySeeded {
			fmt.Printf("  ○ already seeded — skipping description/rapids/access\n")
		} else {
			rc := ai.ReachContext{
				Name:     rd.Name,
				Region:   rd.Region,
				ClassMin: rd.ClassMin,
				ClassMax: rd.ClassMax,
				LengthMi: rd.LengthMi,
				Notes:    rd.Notes,
			}
			fmt.Printf("  ◌ calling Claude ReachSeeder…\n")
			seed, err := seeder.SeedReach(ctx, rc)
			if err != nil {
				fmt.Printf("  ✗ seeder: %v\n", err)
				failed++
				continue
			}

			// 5. Write description.
			if seed.Description != "" {
				if err := writeDescription(ctx, pool, reachID, seed); err != nil {
					fmt.Printf("  ✗ write description: %v\n", err)
				} else {
					flag := "draft"
					if seed.DescriptionAutoVerified() {
						flag = "auto-verified"
					}
					fmt.Printf("  ✓ description (conf=%d, %s)\n", seed.DescriptionConfidence, flag)
				}
			}

			// 6. Write rapids.
			if len(seed.Rapids) > 0 {
				written, autoVerified := writeRapids(ctx, pool, reachID, seed.Rapids)
				fmt.Printf("  ✓ %d rapids (%d auto-verified)\n", written, autoVerified)
			} else {
				fmt.Printf("  ○ no rapids above confidence floor\n")
			}

			// 7. Write access points + waypoints.
			if len(seed.Access) > 0 {
				written, autoVerified := writeAccess(ctx, pool, reachID, seed.Access)
				fmt.Printf("  ✓ %d access points (%d auto-verified)\n", written, autoVerified)
			} else {
				fmt.Printf("  ○ no access points above confidence floor\n")
			}

			// Seed flow ranges in the same pass when doing a full seed.
			if len(seed.FlowRanges) > 0 {
				written, autoVerified := writeFlowRanges(ctx, pool, rd.GaugeExtID, rd.GaugeSource, seed.FlowRanges)
				fmt.Printf("  ✓ %d flow ranges (%d auto-verified)\n", written, autoVerified)
			}

			time.Sleep(1 * time.Second)
		}

		// 8. Seed flow ranges independently — runs even on already-seeded reaches
		// so we can add ranges to existing reaches without a full reseed.
		// Skip if this gauge already has ai_seed ranges (ON CONFLICT handles the rest).
		if alreadySeeded {
			var existingRanges int
			pool.QueryRow(ctx, `
				SELECT COUNT(*) FROM flow_ranges fr
				JOIN gauges g ON g.id = fr.gauge_id
				WHERE g.external_id = $1 AND g.source = $2
				  AND fr.data_source = 'ai_seed'
			`, rd.GaugeExtID, rd.GaugeSource).Scan(&existingRanges)

			if existingRanges == 0 {
				rc := ai.ReachContext{
					Name: rd.Name, Region: rd.Region,
					ClassMin: rd.ClassMin, ClassMax: rd.ClassMax,
					LengthMi: rd.LengthMi, Notes: rd.Notes,
				}
				fmt.Printf("  ◌ calling Claude for flow ranges…\n")
				seed, err := seeder.SeedReach(ctx, rc)
				if err != nil {
					fmt.Printf("  ✗ flow range seeder: %v\n", err)
				} else if len(seed.FlowRanges) > 0 {
					written, autoVerified := writeFlowRanges(ctx, pool, rd.GaugeExtID, rd.GaugeSource, seed.FlowRanges)
					fmt.Printf("  ✓ %d flow ranges (%d auto-verified)\n", written, autoVerified)
				} else {
					fmt.Printf("  ○ no flow ranges above confidence floor\n")
				}
				time.Sleep(1 * time.Second)
			} else {
				fmt.Printf("  ○ flow ranges already present (%d)\n", existingRanges)
			}
		}

		seeded++
	}

	fmt.Printf("\nDone: %d seeded, %d skipped, %d failed\n", seeded, skipped, failed)
}

// ---- Reach definitions ------------------------------------------------------
//
// Each entry corresponds to one or more featured gauges already in the DB.
// ClassMin/ClassMax use the international scale with .5 increments.
// LengthMi is approximate — the AI will refine it.
// KnownFlowRanges: domain-expert data we enter directly as verified/manual rows
// so it's available immediately without waiting for FlowRangeSeeder to run.

type reachDef struct {
	Slug             string
	Name             string
	Region           string
	ClassMin         float64
	ClassMax         float64
	Character        string // pool-drop / continuous / creeking / big-water / canyon
	LengthMi         float64
	AWReachID        string
	GaugeExtID       string
	GaugeSource      string
	// Additional gauges with explicit relationship types.
	// The primary gauge (GaugeExtID/GaugeSource) always gets reach_relationship='primary'.
	// These get the stated relationship type.
	RelatedGauges    []gaugeAssoc
	// Notes passed to the AI seeder as extra context (gauge math, local knowledge, etc.)
	Notes            string
	KnownFlowRanges  []knownFlowRange
}

// gaugeAssoc links a gauge to a reach with an explicit relationship type.
type gaugeAssoc struct {
	ExtID        string
	Source       string
	Relationship string // upstream_indicator | downstream_indicator | tributary
}

// knownFlowRange is a flow range band entered from direct domain knowledge
// (not AI-generated). Written with data_source='manual' and verified=true.
type knownFlowRange struct {
	Label   string
	MinCFS  *float64
	MaxCFS  *float64
	Notes   string
}

func ptr(f float64) *float64 { return &f }

var reaches = []reachDef{

	// ---- Arkansas River -------------------------------------------------------
	// Dam-regulated below Pueblo Reservoir; above Pueblo the river is snowmelt/runoff.
	// The Numbers, Browns Canyon, and Royal Gorge are all classic commercial and
	// private runs — nationally recognized, extensively documented.
	{
		Slug: "arkansas-the-numbers", Name: "The Numbers",
		Region: "Arkansas River, Colorado — above Buena Vista",
		ClassMin: 4.0, ClassMax: 5.0, Character: "continuous", LengthMi: 10.0,
		GaugeExtID: "07087050", GaugeSource: "usgs",
	},
	{
		Slug: "arkansas-browns-canyon", Name: "Browns Canyon",
		Region: "Arkansas River, Colorado — Buena Vista to Salida",
		ClassMin: 3.0, ClassMax: 4.0, Character: "pool-drop", LengthMi: 9.0,
		GaugeExtID: "07091200", GaugeSource: "usgs",
		// Put-in: Ruby Mountain / Fisherman's Bridge; Take-out: Hecla Junction.
		// (Hecla is the take-out, not the put-in — common point of confusion.)
	},
	{
		Slug: "arkansas-royal-gorge", Name: "Royal Gorge",
		Region: "Arkansas River, Colorado — below Cañon City",
		ClassMin: 4.0, ClassMax: 5.0, Character: "continuous", LengthMi: 10.0,
		GaugeExtID: "07094500", GaugeSource: "usgs",
	},

	// ---- Colorado River -------------------------------------------------------
	{
		Slug: "colorado-gore-canyon", Name: "Gore Canyon",
		Region: "Colorado River, Colorado — near Kremmling",
		ClassMin: 5.0, ClassMax: 5.0, Character: "continuous", LengthMi: 10.0,
		GaugeExtID: "09058000", GaugeSource: "usgs",
	},

	// ---- Yampa River ----------------------------------------------------------
	// Cross Mountain Gorge is a remote, technical canyon run.
	// The user noted 1040 cfs rising on a diurnal cycle as a typical morning reading.
	{
		Slug: "yampa-cross-mountain-gorge", Name: "Cross Mountain Gorge",
		Region: "Yampa River, Colorado — near Maybell",
		ClassMin: 4.0, ClassMax: 4.0, Character: "canyon", LengthMi: 7.0,
		GaugeExtID: "09251000", GaugeSource: "usgs",
	},
	{
		Slug: "yampa-canyon", Name: "Yampa Canyon",
		Region: "Yampa River, Colorado — Deerlodge Park through Dinosaur National Monument",
		ClassMin: 3.0, ClassMax: 4.0, Character: "canyon", LengthMi: 46.0,
		GaugeExtID: "09260050", GaugeSource: "usgs",
	},

	// ---- Gunnison River -------------------------------------------------------
	{
		Slug: "gunnison-black-canyon", Name: "Black Canyon of the Gunnison",
		Region: "Gunnison River, Colorado — Black Canyon National Park",
		ClassMin: 5.0, ClassMax: 5.0, Character: "creeking", LengthMi: 14.0,
		GaugeExtID: "09128000", GaugeSource: "usgs",
	},
	{
		Slug: "gunnison-gorge", Name: "Gunnison Gorge",
		Region: "Gunnison River, Colorado — below Black Canyon",
		ClassMin: 4.0, ClassMax: 4.0, Character: "canyon", LengthMi: 16.0,
		GaugeExtID: "09152500", GaugeSource: "usgs",
	},

	// ---- Escalante Creek ------------------------------------------------------
	// Remote snowmelt-fed run near Delta, CO. No significant upstream dams so it
	// follows a classic diurnal pattern — flows peak in the early-to-mid afternoon.
	// Drive time from the Front Range is significant; timing matters.
	// Domain-expert data: 300–500 cfs is good medium flow (community-verified).
	{
		Slug: "escalante-creek", Name: "Escalante Creek",
		Region: "Escalante Creek, Colorado — near Delta",
		ClassMin: 3.0, ClassMax: 4.0, Character: "creeking", LengthMi: 12.0,
		GaugeExtID: "09151500", GaugeSource: "usgs",
		KnownFlowRanges: []knownFlowRange{
			{Label: "too_low", MinCFS: nil,       MaxCFS: ptr(200),  Notes: "Boat-dragging conditions below 200 cfs."},
			{Label: "minimum", MinCFS: ptr(200),  MaxCFS: ptr(300),  Notes: "Runnable but bony."},
			{Label: "fun",     MinCFS: ptr(300),  MaxCFS: ptr(500),  Notes: "Good medium flow. Classic Escalante conditions."},
			{Label: "pushy",   MinCFS: ptr(500),  MaxCFS: ptr(800),  Notes: "Higher, faster. Diurnal peak range on big snowmelt days."},
			{Label: "flood",   MinCFS: ptr(800),  MaxCFS: nil,       Notes: "Do not run."},
		},
	},

	// ---- Clear Creek ----------------------------------------------------------
	// Front Range creek — no upstream reservoir, purely snowmelt/runoff driven.
	// Strong diurnal pattern in spring; peaks mid-afternoon on hot days.
	// Two main paddle sections are currently seeded; Upper Clear Creek, Idaho Springs,
	// and Golden Whitewater Park are stubs for future local-knowledge verification.

	// Lawson to Idaho Springs — the upper canyon, classic IV spring run.
	// Shuttle on Hwy 6, continuous granite canyon, strong diurnal swing in April–June.
	{
		Slug: "clear-creek-lawson", Name: "Lawson",
		Region: "Clear Creek, Colorado — Lawson to Idaho Springs",
		ClassMin: 4.0, ClassMax: 4.0, Character: "canyon", LengthMi: 7.0,
		GaugeExtID: "06716500", GaugeSource: "usgs",
	},

	// Idaho Springs to Golden — the lower canyon, the main destination run.
	// Includes Blackrock (Class V), the Screaming Quarter Mile, and Elbow Falls (Class IV-V).
	// Very committing in places; limited egress through the gorge sections.
	// USGS gauge 06719505 at Golden is the reference for this run.
	// 06716500 (Lawson, upstream) is an upstream indicator.
	{
		Slug: "clear-creek-canyon", Name: "Clear Creek Canyon",
		Region: "Clear Creek, Colorado — Idaho Springs to Golden",
		ClassMin: 4.0, ClassMax: 5.0, Character: "canyon", LengthMi: 14.0,
		GaugeExtID: "06719505", GaugeSource: "usgs",
		RelatedGauges: []gaugeAssoc{
			{ExtID: "06716500", Source: "usgs", Relationship: "upstream_indicator"},
		},
	},

	// ---- Cache La Poudre -------------------------------------------------------
	// The Fort Collins USGS gauge (06752260) is the reference point, but
	// correlation to canyon conditions is imperfect — reservoir outflows from
	// upstream and inflow from the Laramie River tunnel (right at the take-out)
	// can inflate lower readings. Flows can rise very quickly from snowmelt.
	//
	// Locals rely on a painted rock gauge on river right, just downstream of the
	// highway bridge at the top of Boneyard rapid: 2.5 ft = medium, 3+ ft = high.
	// At high water, sticky powerful holes form below Boneyard.
	// poudrerockreport.com (Fort Collins locals) checks the rock gauge daily
	// and posts conditions + hazard notes — a candidate for future RSS/AI integration.
	{
		Slug: "cache-la-poudre-canyon", Name: "Cache La Poudre Canyon",
		Region: "Cache La Poudre River, Colorado — Fort Collins Canyon",
		ClassMin: 3.0, ClassMax: 4.0, Character: "canyon", LengthMi: 14.0,
		GaugeExtID: "06752260", GaugeSource: "usgs",
		Notes: `The USGS gauge at Fort Collins (06752260) is at the canyon mouth. Correlation to canyon flow is imperfect: reservoir outflows from Halligan/Seaman and inflow from the Laramie River tunnel near the take-out can inflate readings, especially at lower flows.

Locals use a painted rock gauge on river right, just downstream of the highway bridge at the top of Boneyard Rapid. 2.5 ft on the rock gauge = medium conditions; 3+ ft = high (scout carefully — powerful sticky holes form in the gorge below Boneyard at high water). Flows can rise quickly from snowmelt on hot spring days.

poudrerockreport.com is maintained by Fort Collins locals who check the rock gauge daily and post current conditions plus hazard notes. This is the most trusted local beta source for this run.`,
	},

	// ---- Animas River ---------------------------------------------------------
	{
		Slug: "animas-durango", Name: "Animas River — Durango Town Run",
		Region: "Animas River, Colorado — through Durango",
		ClassMin: 2.0, ClassMax: 3.0, Character: "continuous", LengthMi: 5.0,
		GaugeExtID: "09361500", GaugeSource: "usgs",
	},

	// ---- Rio Grande -----------------------------------------------------------
	{
		Slug: "rio-grande-taos-box", Name: "Taos Box",
		Region: "Rio Grande, New Mexico — below Taos Junction Bridge",
		ClassMin: 4.0, ClassMax: 4.0, Character: "canyon", LengthMi: 17.0,
		GaugeExtID: "08276500", GaugeSource: "usgs",
	},

	// ---- North Platte ---------------------------------------------------------
	{
		Slug: "north-platte-six-mile-gap", Name: "Six Mile Gap",
		Region: "North Platte River, Colorado — near Northgate",
		ClassMin: 3.0, ClassMax: 4.0, Character: "canyon", LengthMi: 6.0,
		GaugeExtID: "06620000", GaugeSource: "usgs",
	},

	// ---- South Platte ---------------------------------------------------------
	// Cheesman Canyon is one of the most demanding runs in Colorado — Class V,
	// extremely committing, limited egress. Primarily a kayak run.
	{
		Slug: "south-platte-cheesman-canyon", Name: "Cheesman Canyon",
		Region: "South Platte River, Colorado — above Deckers",
		ClassMin: 5.0, ClassMax: 5.0, Character: "creeking", LengthMi: 8.0,
		GaugeExtID: "06700000", GaugeSource: "usgs",
	},

	// ---- North Fork South Platte ---------------------------------------------
	// Classic Front Range creeking corridor for Denver-area paddlers.
	// The Grant gauge (PLAGRACO) is the controlling gauge for the whole corridor —
	// locals check it before driving. Below ~150 cfs it's too low; above ~600
	// it gets serious. Water travels from Grant to Bailey in ~2 hours.
	// Note: PLABAICO (Bailey gauge) is not in the telemetry DB; add it manually
	// when it becomes available. PLAGRACO is the best upstream proxy for now.
	{
		Slug: "n-fork-south-platte-bailey", Name: "Bailey",
		Region: "North Fork South Platte, Colorado — Bailey to Cliffdale",
		ClassMin: 4.0, ClassMax: 5.0, Character: "creeking", LengthMi: 6.0,
		GaugeExtID: "PLAGRACO", GaugeSource: "dwr",
		// PLAGRACO is upstream at Grant — acts as upstream indicator for the run.
		// Listed as primary here until PLABAICO (Bailey gauge) is in the DB.
	},
	{
		Slug: "n-fork-south-platte-foxton", Name: "Foxton",
		Region: "North Fork South Platte, Colorado — Ferndale (Boulder Garden) to confluence",
		ClassMin: 4.0, ClassMax: 4.5, Character: "creeking", LengthMi: 5.0,
		GaugeExtID: "PLAGRACO", GaugeSource: "dwr",
		RelatedGauges: []gaugeAssoc{
			{ExtID: "PLASPLCO", Source: "dwr", Relationship: "downstream_indicator"},
		},
		Notes: `Put-in is Boulder Garden at Ferndale. The run takes out at the confluence of the North and South Forks of the South Platte.

Gauge math: The PLAGRACO gauge at Grant is the best upstream indicator but does not account for unmonitored tributary creeks that enter between Grant and the confluence. Local paddlers use a derived calculation to estimate true N Fork flow through Foxton: PLASPLCO (South Platte at South Platte, at the confluence) minus USGS 06701900 (S Platte below Brush Creek near Trumbull, i.e. the Deckers gauge) gives the North Fork contribution. This difference is more accurate than PLAGRACO alone when tributary runoff is significant.`,
	},

	// ---- South Platte (S Fork) — Deckers corridor ----------------------------
	// Popular intermediate run for Denver boaters. The S Platte below Brush Creek
	// near Trumbull (USGS 06701900, also known as the PLABRUCO equivalent) is the
	// controlling gauge. PLASPLCO sits at the bottom of the run near the confluence.
	{
		Slug: "south-platte-deckers", Name: "Deckers",
		Region: "South Platte River, Colorado — Trumbull/Deckers to South Platte confluence",
		ClassMin: 2.0, ClassMax: 3.0, Character: "canyon", LengthMi: 10.0,
		GaugeExtID: "06701900", GaugeSource: "usgs",
		RelatedGauges: []gaugeAssoc{
			{ExtID: "PLASPLCO", Source: "dwr", Relationship: "downstream_indicator"},
		},
	},

	// ---- Waterton Canyon -------------------------------------------------------
	// Below the N/S fork confluence, the South Platte cuts through Waterton Canyon —
	// a popular Front Range run with easy trail access and no shuttle needed from the
	// top. Take-out is ~1/4 mile below PLASPLCO before Strontia Springs Reservoir;
	// going further means portaging the dam or paddling the flatwater reservoir.
	// PLASPLCO (South Platte at South Platte) is essentially at the put-in.
	// PLAWATCO (South Platte at Waterton) is downstream near the canyon exit.
	{
		Slug: "south-platte-waterton-canyon", Name: "Waterton Canyon",
		Region: "South Platte River, Colorado — South Platte to Strontia Springs",
		ClassMin: 2.0, ClassMax: 3.0, Character: "canyon", LengthMi: 10.0,
		GaugeExtID: "PLASPLCO", GaugeSource: "dwr",
		RelatedGauges: []gaugeAssoc{
			{ExtID: "PLAWATCO", Source: "dwr", Relationship: "downstream_indicator"},
		},
	},
}

// ---- Helpers ----------------------------------------------------------------

func mustEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		log.Fatalf("required env var %s is not set", key)
	}
	return v
}
