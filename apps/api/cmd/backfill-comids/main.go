// backfill-comids resolves NHD ComIDs and GNIS names for reaches that pre-date
// the NLDI integration. For each reach with a centerline but no put_in_comid,
// it snaps the centerline's start/end points to NHD via the NLDI API, then
// looks up GNIS names for the resolved ComIDs from EPA's NHDPlus service.
//
// Usage:
//
//	go run ./cmd/backfill-comids                # write changes
//	go run ./cmd/backfill-comids -dry-run       # show changes only
//	go run ./cmd/backfill-comids -mismatches    # only show river_name conflicts
//	go run ./cmd/backfill-comids -report        # report (name, HUC8) for already-backfilled reaches
//
// Reads DATABASE_URL from the environment.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/h2oflow/h2oflow/apps/api/internal/db"
	"github.com/h2oflow/h2oflow/apps/api/internal/nldi"
	"github.com/jackc/pgx/v5/pgxpool"
)

const epaNHDPlusURL = "https://watersgeo.epa.gov/arcgis/rest/services/NHDPlus_NP21/NHDSnapshot_NP21/MapServer/0/query"

type reachRow struct {
	ID         string
	Slug       string
	RiverName  string
	PutInLat   float64
	PutInLng   float64
	TakeOutLat float64
	TakeOutLng float64
}

type snapResult struct {
	reachRow
	PutInComID, TakeOutComID string
	SnapErr                  string
}

type nhdInfo struct {
	GNIS      string
	GNISID    string
	Reachcode string
}

// HUC8 returns the 8-digit basin code from a REACHCODE, or "" if malformed.
func (n nhdInfo) HUC8() string {
	if len(n.Reachcode) < 8 {
		return ""
	}
	return n.Reachcode[:8]
}

func main() {
	var (
		dryRun         = flag.Bool("dry-run", false, "show changes without writing")
		rateMs         = flag.Int("rate-ms", 600, "min ms between NLDI requests")
		onlyMismatches = flag.Bool("mismatches", false, "only print river_name conflicts")
		report         = flag.Bool("report", false, "report (name, HUC8) pairs for already-backfilled reaches; no snap, no writes")
	)
	flag.Parse()

	ctx := context.Background()
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatalf("DATABASE_URL is required")
	}
	pool, err := db.Connect(ctx, dbURL)
	if err != nil {
		log.Fatalf("db: %v", err)
	}
	defer pool.Close()

	if *report {
		runReport(ctx, pool)
		return
	}

	rows, err := pool.Query(ctx, `
		SELECT
			id, slug, COALESCE(river_name,''),
			ST_Y(ST_StartPoint(centerline::geometry)::geometry),
			ST_X(ST_StartPoint(centerline::geometry)::geometry),
			ST_Y(ST_EndPoint(centerline::geometry)::geometry),
			ST_X(ST_EndPoint(centerline::geometry)::geometry)
		FROM reaches
		WHERE put_in_comid IS NULL AND centerline IS NOT NULL
		ORDER BY river_name, slug
	`)
	if err != nil {
		log.Fatalf("query: %v", err)
	}
	var todo []reachRow
	for rows.Next() {
		var r reachRow
		if err := rows.Scan(&r.ID, &r.Slug, &r.RiverName, &r.PutInLat, &r.PutInLng, &r.TakeOutLat, &r.TakeOutLng); err != nil {
			rows.Close()
			log.Fatalf("scan: %v", err)
		}
		todo = append(todo, r)
	}
	rows.Close()
	if len(todo) == 0 {
		log.Printf("no reaches need backfill")
		return
	}
	log.Printf("backfill candidates: %d", len(todo))

	nldiC := nldi.New()
	results := make([]snapResult, 0, len(todo))
	comidSet := map[string]bool{}
	for i, r := range todo {
		time.Sleep(time.Duration(*rateMs) * time.Millisecond)
		in, err1 := nldiC.SnapToComID(ctx, r.PutInLat, r.PutInLng)
		time.Sleep(time.Duration(*rateMs) * time.Millisecond)
		out, err2 := nldiC.SnapToComID(ctx, r.TakeOutLat, r.TakeOutLng)
		res := snapResult{reachRow: r}
		switch {
		case err1 != nil:
			res.SnapErr = fmt.Sprintf("put_in: %v", err1)
		case err2 != nil:
			res.SnapErr = fmt.Sprintf("take_out: %v", err2)
		default:
			res.PutInComID = in.ComID
			res.TakeOutComID = out.ComID
			comidSet[in.ComID] = true
			comidSet[out.ComID] = true
		}
		results = append(results, res)
		status := res.SnapErr
		if status == "" {
			status = fmt.Sprintf("%s -> %s", res.PutInComID, res.TakeOutComID)
		}
		log.Printf("[%d/%d] %s: %s", i+1, len(todo), r.Slug, status)
	}

	nhdByComID, err := lookupNHD(ctx, comidSet)
	if err != nil {
		log.Printf("nhd lookup partial: %v", err)
	}

	fmt.Println()
	fmt.Println("── results ──────────────────────────────────────────────")
	mismatches := 0
	updated := 0
	failed := 0
	type riverKey struct{ GNISID, Name string }
	newRivers := map[riverKey]bool{}
	for _, res := range results {
		if res.SnapErr != "" {
			failed++
			fmt.Printf("✗  %-45s  %s\n", res.Slug, res.SnapErr)
			continue
		}
		info := nhdByComID[res.PutInComID]
		if info.GNIS == "" {
			info = nhdByComID[res.TakeOutComID]
		}
		gnis := info.GNIS
		mismatch := gnis != "" && !strings.EqualFold(strings.TrimSpace(res.RiverName), strings.TrimSpace(gnis))
		if mismatch {
			mismatches++
		}
		if *onlyMismatches && !mismatch {
			continue
		}
		marker := "✓"
		if mismatch {
			marker = "≠"
		}
		fmt.Printf("%s  %-45s  db=%-22q  nhd=%-22q  gnis_id=%-8s  comids=[%s, %s]\n",
			marker, res.Slug, res.RiverName, gnis, info.GNISID, res.PutInComID, res.TakeOutComID)
		if gnis != "" {
			newRivers[riverKey{GNISID: info.GNISID, Name: gnis}] = true
		}
		if *dryRun {
			continue
		}
		_, err := pool.Exec(ctx, `
			UPDATE reaches SET
				put_in_comid   = $1,
				take_out_comid = $2,
				anchor_comid   = COALESCE(anchor_comid, $1),
				put_in         = ST_SetSRID(ST_MakePoint($3,$4),4326)::geography,
				take_out       = ST_SetSRID(ST_MakePoint($5,$6),4326)::geography
			WHERE id = $7
		`, res.PutInComID, res.TakeOutComID,
			res.PutInLng, res.PutInLat, res.TakeOutLng, res.TakeOutLat,
			res.ID)
		if err != nil {
			failed++
			fmt.Printf("✗  update failed for %s: %v\n", res.Slug, err)
			continue
		}
		updated++
	}

	fmt.Println()
	fmt.Println("── summary ──────────────────────────────────────────────")
	fmt.Printf("snap success:      %d / %d\n", len(results)-failed, len(results))
	fmt.Printf("river_name match:  %d\n", len(results)-failed-mismatches)
	fmt.Printf("river_name diff:   %d (review manually — not auto-overwritten)\n", mismatches)
	fmt.Printf("rows updated:      %d\n", updated)
	if *dryRun {
		fmt.Println("(dry-run — no DB changes written)")
	}

	// Distinct (GNIS name, HUC8) pairs — basin disambiguates collisions
	// like Clear Creek (Arkansas) vs Clear Creek (South Platte). These
	// are the candidate rows for seeding the rivers table.
	if len(newRivers) > 0 {
		fmt.Println()
		fmt.Println("── candidate river rows (gnis_id is unique key) ─────────")
		fmt.Printf("  %-10s  %-30s\n", "GNIS_ID", "Name")
		fmt.Println("  " + strings.Repeat("-", 44))
		keys := make([]riverKey, 0, len(newRivers))
		for k := range newRivers {
			keys = append(keys, k)
		}
		sort.Slice(keys, func(i, j int) bool {
			if keys[i].Name != keys[j].Name {
				return keys[i].Name < keys[j].Name
			}
			return keys[i].GNISID < keys[j].GNISID
		})
		for _, k := range keys {
			fmt.Printf("  %-10s  %s\n", k.GNISID, k.Name)
		}
	}
}

// runReport scans reaches that already have ComIDs, fetches their GNIS/HUC8
// from NHD, and prints the distinct (name, basin) pairs plus river_name
// mismatches. Read-only — useful after the initial backfill to review what
// would seed the rivers table.
func runReport(ctx context.Context, pool *pgxpool.Pool) {
	rows, err := pool.Query(ctx, `
		SELECT slug, COALESCE(river_name,''), put_in_comid, take_out_comid
		FROM reaches
		WHERE put_in_comid IS NOT NULL
		ORDER BY river_name, slug
	`)
	if err != nil {
		log.Fatalf("query: %v", err)
	}
	type row struct{ Slug, River, PutIn, TakeOut string }
	var all []row
	comidSet := map[string]bool{}
	for rows.Next() {
		var r row
		if err := rows.Scan(&r.Slug, &r.River, &r.PutIn, &r.TakeOut); err != nil {
			rows.Close()
			log.Fatalf("scan: %v", err)
		}
		all = append(all, r)
		comidSet[r.PutIn] = true
		comidSet[r.TakeOut] = true
	}
	rows.Close()
	log.Printf("reaches with ComIDs: %d", len(all))

	nhdByComID, err := lookupNHD(ctx, comidSet)
	if err != nil {
		log.Printf("nhd lookup partial: %v", err)
	}

	type riverKey struct{ GNISID, Name string }
	pairs := map[riverKey]bool{}
	mismatches := 0
	fmt.Println()
	fmt.Println("── per-reach NHD info ───────────────────────────────────")
	for _, r := range all {
		info := nhdByComID[r.PutIn]
		if info.GNIS == "" {
			info = nhdByComID[r.TakeOut]
		}
		mismatch := info.GNIS != "" && !strings.EqualFold(strings.TrimSpace(r.River), strings.TrimSpace(info.GNIS))
		marker := "✓"
		if mismatch {
			marker = "≠"
			mismatches++
		}
		fmt.Printf("%s  %-45s  db=%-22q  nhd=%-22q  gnis_id=%-8s  huc8=%s\n",
			marker, r.Slug, r.River, info.GNIS, info.GNISID, info.HUC8())
		if info.GNIS != "" {
			pairs[riverKey{GNISID: info.GNISID, Name: info.GNIS}] = true
		}
	}

	fmt.Println()
	fmt.Println("── candidate river rows (gnis_id is unique key) ─────────")
	fmt.Printf("  %-10s  %-30s\n", "GNIS_ID", "Name")
	fmt.Println("  " + strings.Repeat("-", 44))
	keys := make([]riverKey, 0, len(pairs))
	for k := range pairs {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		if keys[i].Name != keys[j].Name {
			return keys[i].Name < keys[j].Name
		}
		return keys[i].GNISID < keys[j].GNISID
	})
	for _, k := range keys {
		fmt.Printf("  %-10s  %s\n", k.GNISID, k.Name)
	}

	fmt.Println()
	fmt.Printf("river_name mismatches: %d (manual review)\n", mismatches)
}

// lookupNHD queries EPA's NHDPlus snapshot for GNIS names and REACHCODE by ComID.
// Returns a map from ComID (string) to nhdInfo. The first 8 digits of REACHCODE
// are the HUC8 basin code, which disambiguates same-named flowlines across basins.
func lookupNHD(ctx context.Context, comidSet map[string]bool) (map[string]nhdInfo, error) {
	out := map[string]nhdInfo{}
	if len(comidSet) == 0 {
		return out, nil
	}
	ids := make([]string, 0, len(comidSet))
	for c := range comidSet {
		ids = append(ids, c)
	}

	hc := &http.Client{Timeout: 30 * time.Second}
	const batchSize = 50
	for i := 0; i < len(ids); i += batchSize {
		end := i + batchSize
		if end > len(ids) {
			end = len(ids)
		}
		batch := ids[i:end]

		q := url.Values{}
		q.Set("where", "COMID IN ("+strings.Join(batch, ",")+")")
		q.Set("outFields", "GNIS_NAME,GNIS_ID,COMID,REACHCODE")
		q.Set("returnGeometry", "false")
		q.Set("f", "json")
		u := epaNHDPlusURL + "?" + q.Encode()

		req, _ := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
		req.Header.Set("Accept", "application/json")
		resp, err := hc.Do(req)
		if err != nil {
			return out, fmt.Errorf("epa request: %w", err)
		}
		var body struct {
			Features []struct {
				Attributes struct {
					GNIS      *string `json:"GNIS_NAME"`
					GNISID    *string `json:"GNIS_ID"`
					ComID     int     `json:"COMID"`
					Reachcode *string `json:"REACHCODE"`
				} `json:"attributes"`
			} `json:"features"`
		}
		err = json.NewDecoder(resp.Body).Decode(&body)
		resp.Body.Close()
		if err != nil {
			return out, fmt.Errorf("epa decode: %w", err)
		}
		for _, f := range body.Features {
			key := fmt.Sprintf("%d", f.Attributes.ComID)
			info := nhdInfo{}
			if f.Attributes.GNIS != nil {
				info.GNIS = *f.Attributes.GNIS
			}
			if f.Attributes.GNISID != nil {
				info.GNISID = *f.Attributes.GNISID
			}
			if f.Attributes.Reachcode != nil {
				info.Reachcode = *f.Attributes.Reachcode
			}
			out[key] = info
		}
	}
	return out, nil
}
