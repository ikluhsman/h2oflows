// import-kml imports reach features (rapids, access points, centerlines) from
// a Google My Maps KML export into the H2OFlows database.
//
// Map conventions:
//   - One Folder per reach — folder name matched to reaches.name or slug
//   - Category-organized maps — folders named "Access Points", "Rivers", "Rapids"
//     with reach inferred from pin names + geographic proximity
//
// Pin name prefix → feature type:
//
//	"Rapid: <name>"    → rapids
//	"Put-in: <name>"   → reach_access type=put_in
//	"Take-out: <name>" → reach_access type=take_out
//	"Parking: <name>"  → reach_access.parking_location on nearest access
//	"Shuttle: <name>"  → reach_access type=shuttle_drop
//
// Usage:
//
//	go run ./cmd/import-kml/ --file arkansas.kmz
//	go run ./cmd/import-kml/ --file arkansas.kmz --centerlines=osm
//	go run ./cmd/import-kml/ --file arkansas.kmz --centerlines=nldi
//	go run ./cmd/import-kml/ --file arkansas.kmz --dry-run
//
// Env vars: DATABASE_URL
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/h2oflow/h2oflow/apps/api/internal/db"
	"github.com/h2oflow/h2oflow/apps/api/internal/kmlimport"
)

func main() {
	file        := flag.String("file", "", "path to KML or KMZ file (required)")
	centerlines := flag.String("centerlines", "", "fetch centerlines for imported reaches — 'osm' or 'nldi'")
	dryRun      := flag.Bool("dry-run", false, "parse and match without writing to DB")
	flag.Parse()

	if *file == "" {
		log.Fatal("--file is required")
	}

	var centerlineSource kmlimport.CenterlineSource
	switch *centerlines {
	case "":
		// centerline fetch disabled
	case "osm":
		centerlineSource = kmlimport.CenterlineOSM
	case "nldi":
		centerlineSource = kmlimport.CenterlineNLDI
	default:
		log.Fatalf("--centerlines must be 'osm' or 'nldi' (got %q)", *centerlines)
	}

	ctx := context.Background()
	dbURL := mustEnv("DATABASE_URL")

	pool, err := db.Connect(ctx, dbURL)
	if err != nil {
		log.Fatalf("db: %v", err)
	}
	defer pool.Close()

	data, err := os.ReadFile(*file)
	if err != nil {
		log.Fatalf("read file: %v", err)
	}

	doc, err := kmlimport.ParseKMLBytes(data)
	if err != nil {
		log.Fatalf("parse kml: %v", err)
	}
	fmt.Printf("Map: %s\n", doc.Name)

	imp := kmlimport.New(pool, *dryRun)
	res, err := imp.Import(ctx, doc)
	if err != nil {
		log.Fatalf("import: %v", err)
	}

	for _, line := range res.Log {
		fmt.Println(" ", line)
	}

	fmt.Println()
	var centerlineReaches []string
	for slug, st := range res.Reaches {
		fmt.Printf("  %s — rapids=%d put-ins=%d take-outs=%d parking=%d\n",
			st.Name, st.Rapids, st.PutIns, st.TakeOuts, st.Parking)
		if st.PutIns > 0 && st.TakeOuts > 0 {
			centerlineReaches = append(centerlineReaches, slug)
		}
	}

	if centerlineSource != "" && len(centerlineReaches) > 0 {
		fmt.Printf("\n── Fetching %s centerlines ──\n", centerlineSource)
		for _, slug := range centerlineReaches {
			if err := kmlimport.SyncCenterline(ctx, pool, slug, centerlineSource, *dryRun); err != nil {
				fmt.Printf("  ✗ %s: %v\n", slug, err)
			} else {
				fmt.Printf("  ✓ %s\n", slug)
			}
		}
	}

	fmt.Println("\nDone.")
}

func mustEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		log.Fatalf("required env var not set: %s", key)
	}
	return v
}
