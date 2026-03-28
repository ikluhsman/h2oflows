// import-kml imports reach features (rapids, access points, centerlines) from
// a Google My Maps KML export into the H2OFlow database.
//
// Map convention:
//   - One Folder (layer) per reach — folder name matched to reaches.name or slug
//   - Pin name prefix determines feature type:
//       "Rapid: <name>"    → rapids.location (updates existing rapid or inserts)
//       "Put-in: <name>"   → reach_access type=put_in
//       "Take-out: <name>" → reach_access type=take_out
//       "Parking: <name>"  → reach_access type=parking (parking_location col)
//       "Shuttle: <name>"  → reach_access type=shuttle_drop
//   - Pin description field → notes (stored verbatim)
//
// Multiple put-ins are supported — each "Put-in:" pin becomes a separate
// reach_access row. The first put-in listed (by sequence in the KML) is
// treated as the primary/topmost put-in for the reach.
//
// Usage:
//
//	go run ./cmd/import-kml/ --file arkansas.kml
//	go run ./cmd/import-kml/ --file arkansas.kml --centerlines
//	go run ./cmd/import-kml/ --file arkansas.kml --dry-run
//
// --centerlines  after importing, fetch OSM river geometry and store
//                reach centerlines for any reach that gained a put-in
//                and take-out in this import run.
//
// Env vars: DATABASE_URL
package main

import (
	"context"
	"encoding/xml"
	"flag"
	"fmt"
	"log"
	"math"
	"os"
	"strconv"
	"strings"

	"github.com/h2oflow/h2oflow/apps/api/internal/db"
	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	file       := flag.String("file", "", "path to KML export from Google My Maps (required)")
	centerlines := flag.Bool("centerlines", false, "fetch OSM centerlines for imported reaches")
	dryRun     := flag.Bool("dry-run", false, "parse and match without writing to DB")
	flag.Parse()

	if *file == "" {
		log.Fatal("--file is required")
	}

	ctx := context.Background()
	dbURL := mustEnv("DATABASE_URL")

	pool, err := db.Connect(ctx, dbURL)
	if err != nil {
		log.Fatalf("db: %v", err)
	}
	defer pool.Close()

	doc, err := parseKML(*file)
	if err != nil {
		log.Fatalf("parse kml: %v", err)
	}
	fmt.Printf("Map: %s\n", doc.Name)

	imp := &importer{pool: pool, dryRun: *dryRun}

	var centerlineReaches []string

	for _, folder := range doc.Folders {
		reachID, reachSlug, reachName, err := imp.matchReach(ctx, folder.Name)
		if err != nil {
			fmt.Printf("\n⚠  folder %q — no matching reach in DB, skipping\n", folder.Name)
			continue
		}
		fmt.Printf("\n→ %s [%s]\n", reachName, reachSlug)

		var rapids, putIns, takeOuts, parking, shuttle int
		var hasPutIn, hasTakeOut bool

		for _, pm := range folder.Placemarks {
			if pm.Point == nil {
				continue // skip lines/polygons
			}
			lon, lat, ok := parseCoords(pm.Point.Coordinates)
			if !ok {
				fmt.Printf("  ⚠  %q — could not parse coordinates, skipping\n", pm.Name)
				continue
			}

			prefix, pinName := splitPrefix(pm.Name)
			desc := strings.TrimSpace(pm.Description)

			switch prefix {
			case "rapid":
				if err := imp.upsertRapidLocation(ctx, reachID, pinName, lon, lat); err != nil {
					fmt.Printf("  ✗ rapid %q: %v\n", pinName, err)
				} else {
					fmt.Printf("  ✓ rapid: %s\n", pinName)
					rapids++
				}

			case "put-in":
				if err := imp.upsertAccess(ctx, reachID, "put_in", pinName, desc, lon, lat); err != nil {
					fmt.Printf("  ✗ put-in %q: %v\n", pinName, err)
				} else {
					fmt.Printf("  ✓ put-in: %s\n", pinName)
					putIns++
					hasPutIn = true
				}

			case "take-out":
				if err := imp.upsertAccess(ctx, reachID, "take_out", pinName, desc, lon, lat); err != nil {
					fmt.Printf("  ✗ take-out %q: %v\n", pinName, err)
				} else {
					fmt.Printf("  ✓ take-out: %s\n", pinName)
					takeOuts++
					hasTakeOut = true
				}

			case "parking":
				if err := imp.upsertParking(ctx, reachID, pinName, desc, lon, lat); err != nil {
					fmt.Printf("  ✗ parking %q: %v\n", pinName, err)
				} else {
					fmt.Printf("  ✓ parking: %s\n", pinName)
					parking++
				}

			case "shuttle":
				if err := imp.upsertAccess(ctx, reachID, "shuttle_drop", pinName, desc, lon, lat); err != nil {
					fmt.Printf("  ✗ shuttle %q: %v\n", pinName, err)
				} else {
					fmt.Printf("  ✓ shuttle: %s\n", pinName)
					shuttle++
				}

			default:
				fmt.Printf("  ⚠  %q — unknown prefix %q, skipping\n", pm.Name, prefix)
			}
		}

		fmt.Printf("  rapids=%d put-ins=%d take-outs=%d parking=%d shuttle=%d\n",
			rapids, putIns, takeOuts, parking, shuttle)

		if hasPutIn && hasTakeOut {
			centerlineReaches = append(centerlineReaches, reachSlug)
		}
	}

	if *centerlines && len(centerlineReaches) > 0 {
		fmt.Printf("\n── Fetching OSM centerlines ──\n")
		for _, slug := range centerlineReaches {
			if err := syncCenterline(ctx, pool, slug, *dryRun); err != nil {
				fmt.Printf("  ✗ %s: %v\n", slug, err)
			}
		}
	}

	fmt.Println("\nDone.")
}

// ---- Importer ---------------------------------------------------------------

type importer struct {
	pool   *pgxpool.Pool
	dryRun bool
}

// matchReach finds a reach by fuzzy-matching the folder name against
// reaches.name and reaches.slug (case-insensitive).
func (imp *importer) matchReach(ctx context.Context, folderName string) (id, slug, name string, err error) {
	err = imp.pool.QueryRow(ctx, `
		SELECT id, slug, name FROM reaches
		WHERE LOWER(name) = LOWER($1) OR LOWER(slug) = LOWER($1)
		   OR LOWER(name) LIKE '%' || LOWER($1) || '%'
		   OR LOWER($1) LIKE '%' || LOWER(name) || '%'
		ORDER BY
			CASE WHEN LOWER(name) = LOWER($1) THEN 0
			     WHEN LOWER(slug) = LOWER($1) THEN 1
			     ELSE 2 END
		LIMIT 1
	`, folderName).Scan(&id, &slug, &name)
	return
}

// upsertRapidLocation sets the location on an existing rapid, or inserts a
// location-only placeholder if no rapid by that name exists on the reach.
func (imp *importer) upsertRapidLocation(ctx context.Context, reachID, name string, lon, lat float64) error {
	if imp.dryRun {
		return nil
	}
	tag, err := imp.pool.Exec(ctx, `
		UPDATE rapids
		SET location = ST_SetSRID(ST_MakePoint($3, $4), 4326)::geography
		WHERE reach_id = $1 AND LOWER(name) = LOWER($2)
	`, reachID, name, lon, lat)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		// No existing rapid — insert a stub so the location is recorded.
		_, err = imp.pool.Exec(ctx, `
			INSERT INTO rapids (reach_id, name, location, data_source, verified)
			VALUES ($1, $2, ST_SetSRID(ST_MakePoint($3, $4), 4326)::geography, 'import', true)
			ON CONFLICT (reach_id, name) DO UPDATE
			  SET location = EXCLUDED.location
		`, reachID, name, lon, lat)
	}
	return err
}

// upsertAccess inserts or updates a put_in / take_out / shuttle_drop access point.
func (imp *importer) upsertAccess(ctx context.Context, reachID, accessType, name, notes string, lon, lat float64) error {
	if imp.dryRun {
		return nil
	}
	_, err := imp.pool.Exec(ctx, `
		INSERT INTO reach_access
			(reach_id, access_type, name, notes,
			 location, data_source, verified)
		VALUES
			($1, $2, $3, NULLIF($4, ''),
			 ST_SetSRID(ST_MakePoint($5, $6), 4326)::geography, 'import', true)
		ON CONFLICT (reach_id, access_type, name) DO UPDATE
		  SET location = EXCLUDED.location,
		      notes    = COALESCE(EXCLUDED.notes, reach_access.notes),
		      verified = true
	`, reachID, accessType, name, notes, lon, lat)
	return err
}

// upsertParking sets the parking_location on the nearest access point for
// this reach, or inserts a dedicated parking row if none is close enough.
func (imp *importer) upsertParking(ctx context.Context, reachID, name, notes string, lon, lat float64) error {
	if imp.dryRun {
		return nil
	}
	// Try to attach to nearest access point within 500m.
	tag, err := imp.pool.Exec(ctx, `
		UPDATE reach_access
		SET parking_location = ST_SetSRID(ST_MakePoint($3, $4), 4326)::geography
		WHERE reach_id = $1
		  AND ST_DWithin(
		        location,
		        ST_SetSRID(ST_MakePoint($3, $4), 4326)::geography,
		        500
		      )
		  AND parking_location IS NULL
		ORDER BY ST_Distance(location, ST_SetSRID(ST_MakePoint($3, $4), 4326)::geography)
		LIMIT 1
	`, reachID, name, lon, lat)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		// No nearby access point — insert standalone parking row.
		_, err = imp.pool.Exec(ctx, `
			INSERT INTO reach_access
				(reach_id, access_type, name, notes,
				 parking_location, data_source, verified)
			VALUES
				($1, 'put_in', $2, NULLIF($3, ''),
				 ST_SetSRID(ST_MakePoint($4, $5), 4326)::geography, 'import', true)
			ON CONFLICT (reach_id, access_type, name) DO UPDATE
			  SET parking_location = EXCLUDED.parking_location
		`, reachID, name, notes, lon, lat)
	}
	return err
}

// ---- OSM centerline ---------------------------------------------------------

// syncCenterline fetches the river waterway from OSM between the reach's
// put-in and take-out, clips it, and stores it as reaches.centerline.
func syncCenterline(ctx context.Context, pool *pgxpool.Pool, slug string, dryRun bool) error {
	// Load put-in and take-out coordinates from DB.
	var putInLon, putInLat, takeOutLon, takeOutLat float64
	err := pool.QueryRow(ctx, `
		SELECT
		  ST_X(a_in.location::geometry),  ST_Y(a_in.location::geometry),
		  ST_X(a_out.location::geometry), ST_Y(a_out.location::geometry)
		FROM reaches r
		JOIN reach_access a_in  ON a_in.reach_id  = r.id AND a_in.access_type  = 'put_in'
		JOIN reach_access a_out ON a_out.reach_id = r.id AND a_out.access_type = 'take_out'
		WHERE r.slug = $1
		ORDER BY a_in.created_at ASC, a_out.created_at ASC
		LIMIT 1
	`, slug).Scan(&putInLon, &putInLat, &takeOutLon, &takeOutLat)
	if err != nil {
		return fmt.Errorf("no put-in/take-out found: %w", err)
	}

	// Expand a bounding box around both points with padding.
	minLon := math.Min(putInLon, takeOutLon) - 0.05
	maxLon := math.Max(putInLon, takeOutLon) + 0.05
	minLat := math.Min(putInLat, takeOutLat) - 0.05
	maxLat := math.Max(putInLat, takeOutLat) + 0.05

	fmt.Printf("  %s — querying OSM waterway (bbox %.4f,%.4f,%.4f,%.4f)\n",
		slug, minLat, minLon, maxLat, maxLon)

	geojson, err := fetchOSMRiverLine(ctx, minLon, minLat, maxLon, maxLat)
	if err != nil {
		return fmt.Errorf("osm fetch: %w", err)
	}
	if geojson == "" {
		return fmt.Errorf("no river waterway found in bbox")
	}

	if dryRun {
		fmt.Printf("  [dry-run] would store centerline (%d bytes)\n", len(geojson))
		return nil
	}

	// Clip the OSM linestring between the two nearest points to put-in and take-out,
	// then store as the reach centerline.
	_, err = pool.Exec(ctx, `
		UPDATE reaches SET centerline = (
			SELECT ST_LineSubstring(
				line,
				ST_LineLocatePoint(line, put_pt),
				ST_LineLocatePoint(line, take_pt)
			)::geography
			FROM (
				SELECT
					ST_GeomFromGeoJSON($2)                                     AS line,
					ST_ClosestPoint(ST_GeomFromGeoJSON($2),
					    ST_SetSRID(ST_MakePoint($3, $4), 4326))                AS put_pt,
					ST_ClosestPoint(ST_GeomFromGeoJSON($2),
					    ST_SetSRID(ST_MakePoint($5, $6), 4326))                AS take_pt
			) sub
		)
		WHERE slug = $1
	`, slug, geojson, putInLon, putInLat, takeOutLon, takeOutLat)
	if err != nil {
		return fmt.Errorf("store centerline: %w", err)
	}

	fmt.Printf("  ✓ %s — centerline stored\n", slug)
	return nil
}

// fetchOSMRiverLine queries the Overpass API for the river waterway within
// the given bounding box and returns a GeoJSON LineString of the merged ways.
func fetchOSMRiverLine(ctx context.Context, minLon, minLat, maxLon, maxLat float64) (string, error) {
	query := fmt.Sprintf(
		`[out:json];way["waterway"~"^(river|stream)$"](%.6f,%.6f,%.6f,%.6f);out geom;`,
		minLat, minLon, maxLat, maxLon,
	)

	resp, err := overpassQuery(ctx, query)
	if err != nil {
		return "", err
	}

	// Collect all coordinate sequences from returned ways.
	type osmNode struct {
		Lat float64 `json:"lat"`
		Lon float64 `json:"lon"`
	}
	type osmWay struct {
		Geometry []osmNode `json:"geometry"`
		Tags     map[string]string `json:"tags"`
	}
	type osmResp struct {
		Elements []struct {
			Type     string            `json:"type"`
			Geometry []osmNode         `json:"geometry"`
			Tags     map[string]string `json:"tags"`
		} `json:"elements"`
	}

	var parsed osmResp
	if err := jsonUnmarshal(resp, &parsed); err != nil {
		return "", fmt.Errorf("parse osm response: %w", err)
	}
	_ = osmWay{}

	if len(parsed.Elements) == 0 {
		return "", nil
	}

	// Pick the longest named waterway (the main river, not a side channel).
	best := parsed.Elements[0]
	for _, el := range parsed.Elements[1:] {
		if len(el.Geometry) > len(best.Geometry) {
			best = el
		}
	}

	// Build a GeoJSON LineString.
	var coords []string
	for _, n := range best.Geometry {
		coords = append(coords, fmt.Sprintf("[%.7f,%.7f]", n.Lon, n.Lat))
	}
	return fmt.Sprintf(`{"type":"LineString","coordinates":[%s]}`, strings.Join(coords, ",")), nil
}

// ---- KML parsing ------------------------------------------------------------

type kmlDoc struct {
	Name    string
	Folders []kmlFolder
}

type kmlFolder struct {
	Name       string
	Placemarks []kmlPlacemark
}

type kmlPlacemark struct {
	Name        string
	Description string
	Point       *kmlPoint
}

type kmlPoint struct {
	Coordinates string
}

func parseKML(path string) (*kmlDoc, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	type xmlPoint struct {
		Coordinates string `xml:"coordinates"`
	}
	type xmlPlacemark struct {
		Name        string   `xml:"name"`
		Description string   `xml:"description"`
		Point       *xmlPoint `xml:"Point"`
	}
	type xmlFolder struct {
		Name       string         `xml:"name"`
		Placemarks []xmlPlacemark `xml:"Placemark"`
	}
	type xmlDocument struct {
		Name    string      `xml:"name"`
		Folders []xmlFolder `xml:"Folder"`
	}
	type xmlKML struct {
		Document xmlDocument `xml:"Document"`
	}

	var raw xmlKML
	if err := xml.NewDecoder(f).Decode(&raw); err != nil {
		return nil, err
	}

	doc := &kmlDoc{Name: raw.Document.Name}
	for _, xf := range raw.Document.Folders {
		folder := kmlFolder{Name: xf.Name}
		for _, xp := range xf.Placemarks {
			pm := kmlPlacemark{
				Name:        strings.TrimSpace(xp.Name),
				Description: stripHTML(strings.TrimSpace(xp.Description)),
			}
			if xp.Point != nil {
				pm.Point = &kmlPoint{Coordinates: strings.TrimSpace(xp.Point.Coordinates)}
			}
			folder.Placemarks = append(folder.Placemarks, pm)
		}
		doc.Folders = append(doc.Folders, folder)
	}
	return doc, nil
}

// splitPrefix splits "Rapid: Zoom Flume" into ("rapid", "Zoom Flume").
// Returns ("", original) if no recognized prefix is found.
func splitPrefix(name string) (prefix, rest string) {
	for _, p := range []string{"Rapid", "Put-in", "Take-out", "Parking", "Shuttle"} {
		if strings.HasPrefix(strings.ToLower(name), strings.ToLower(p)+":") {
			rest = strings.TrimSpace(name[len(p)+1:])
			return strings.ToLower(p), rest
		}
	}
	return "", name
}

// parseCoords parses "lon,lat,alt" from a KML coordinates string.
func parseCoords(raw string) (lon, lat float64, ok bool) {
	// KML coordinates can be multiline for LineStrings; take first point.
	parts := strings.Fields(raw)
	if len(parts) == 0 {
		return 0, 0, false
	}
	fields := strings.Split(parts[0], ",")
	if len(fields) < 2 {
		return 0, 0, false
	}
	lon, err1 := strconv.ParseFloat(fields[0], 64)
	lat, err2 := strconv.ParseFloat(fields[1], 64)
	return lon, lat, err1 == nil && err2 == nil
}

// stripHTML removes basic HTML tags from Google Maps description fields.
func stripHTML(s string) string {
	var b strings.Builder
	inTag := false
	for _, r := range s {
		switch {
		case r == '<':
			inTag = true
		case r == '>':
			inTag = false
		case !inTag:
			b.WriteRune(r)
		}
	}
	return strings.TrimSpace(b.String())
}

// ---- Helpers ----------------------------------------------------------------

func mustEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		log.Fatalf("required env var not set: %s", key)
	}
	return v
}
