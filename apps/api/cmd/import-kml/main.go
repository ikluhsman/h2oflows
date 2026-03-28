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
	"archive/zip"
	"bytes"
	"context"
	"encoding/xml"
	"flag"
	"fmt"
	"io"
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

	// Collect all (reach, placemark) pairs regardless of map style.
	type assignment struct {
		reachID    string
		reachSlug  string
		reachName  string
		pm         kmlPlacemark
		folderName string // category-map folder hint ("Rapids", "Access Points", …)
	}
	var pins []assignment

	if isCategoryMap(doc.Folders) {
		fmt.Println("(category-organized map — inferring reach from pin names + geography)")

		// Flatten all placemarks first, carrying the folder name for type hints.
		type pendingPin struct {
			pm         kmlPlacemark
			folderName string // e.g. "Rapids", "Access Points"
			lon        float64
			lat        float64
			matched    bool
		}
		var pending []pendingPin
		for _, folder := range doc.Folders {
			for _, pm := range folder.Placemarks {
				if pm.Point == nil {
					continue
				}
				lon, lat, ok := parseCoords(pm.Point.Coordinates)
				if !ok {
					continue
				}
				pending = append(pending, pendingPin{pm: pm, folderName: folder.Name, lon: lon, lat: lat})
			}
		}

		// Pass 1: name-based matching — builds geographic anchors.
		type geoAnchor struct {
			id, slug, name string
			lon, lat       float64
		}
		var anchors []geoAnchor
		for i := range pending {
			pp := &pending[i]
			rid, rslug, rname, err := imp.inferReachFromText(ctx, pp.pm.Name+" "+pp.pm.Description)
			if err != nil {
				continue // will try pass 2
			}
			pp.matched = true
			pins = append(pins, assignment{rid, rslug, rname, pp.pm, pp.folderName})
			anchors = append(anchors, geoAnchor{rid, rslug, rname, pp.lon, pp.lat})
		}

		// Pass 2: assign unmatched pins to nearest geographic anchor.
		for i := range pending {
			pp := &pending[i]
			if pp.matched {
				continue
			}
			if len(anchors) == 0 {
				fmt.Printf("  ⚠  %q — no anchors yet, skipping\n", pp.pm.Name)
				continue
			}
			best := anchors[0]
			bestDist := (anchors[0].lon-pp.lon)*(anchors[0].lon-pp.lon) +
				(anchors[0].lat-pp.lat)*(anchors[0].lat-pp.lat)
			for _, a := range anchors[1:] {
				d := (a.lon-pp.lon)*(a.lon-pp.lon) + (a.lat-pp.lat)*(a.lat-pp.lat)
				if d < bestDist {
					bestDist = d
					best = a
				}
			}
			fmt.Printf("  ~ %q → %s (by proximity)\n", pp.pm.Name, best.name)
			pins = append(pins, assignment{best.id, best.slug, best.name, pp.pm, pp.folderName})
		}
	} else {
		for _, folder := range doc.Folders {
			rid, rslug, rname, err := imp.matchReach(ctx, folder.Name)
			if err != nil {
				fmt.Printf("\n⚠  folder %q — no matching reach in DB, skipping\n", folder.Name)
				continue
			}
			for _, pm := range folder.Placemarks {
				pins = append(pins, assignment{rid, rslug, rname, pm, ""})
			}
		}
	}

	// Group pins by reach for reporting.
	type reachStats struct {
		name                                    string
		rapids, putIns, takeOuts, parking, shuttle int
		hasPutIn, hasTakeOut                    bool
	}
	stats := map[string]*reachStats{}

	for _, a := range pins {
		pm := a.pm
		if pm.Point == nil {
			continue // skip lines/polygons
		}
		lon, lat, ok := parseCoords(pm.Point.Coordinates)
		if !ok {
			fmt.Printf("  ⚠  %q — could not parse coordinates, skipping\n", pm.Name)
			continue
		}

		if _, ok := stats[a.reachSlug]; !ok {
			stats[a.reachSlug] = &reachStats{name: a.reachName}
		}
		st := stats[a.reachSlug]

		prefix, pinName := splitPrefixWithHint(pm.Name, pm.Description, a.folderName)
		desc := strings.TrimSpace(pm.Description)

		switch prefix {
		case "rapid":
			if err := imp.upsertRapidLocation(ctx, a.reachID, pinName, lon, lat); err != nil {
				fmt.Printf("  ✗ rapid %q: %v\n", pinName, err)
			} else {
				fmt.Printf("  ✓ [%s] rapid: %s\n", a.reachName, pinName)
				st.rapids++
			}

		case "put-in":
			if err := imp.upsertAccess(ctx, a.reachID, "put_in", pinName, desc, lon, lat); err != nil {
				fmt.Printf("  ✗ put-in %q: %v\n", pinName, err)
			} else {
				fmt.Printf("  ✓ [%s] put-in: %s\n", a.reachName, pinName)
				st.putIns++
				st.hasPutIn = true
			}

		case "take-out":
			if err := imp.upsertAccess(ctx, a.reachID, "take_out", pinName, desc, lon, lat); err != nil {
				fmt.Printf("  ✗ take-out %q: %v\n", pinName, err)
			} else {
				fmt.Printf("  ✓ [%s] take-out: %s\n", a.reachName, pinName)
				st.takeOuts++
				st.hasTakeOut = true
			}

		case "parking":
			if err := imp.upsertParking(ctx, a.reachID, pinName, desc, lon, lat); err != nil {
				fmt.Printf("  ✗ parking %q: %v\n", pinName, err)
			} else {
				fmt.Printf("  ✓ [%s] parking: %s\n", a.reachName, pinName)
				st.parking++
			}

		case "shuttle":
			if err := imp.upsertAccess(ctx, a.reachID, "shuttle_drop", pinName, desc, lon, lat); err != nil {
				fmt.Printf("  ✗ shuttle %q: %v\n", pinName, err)
			} else {
				fmt.Printf("  ✓ [%s] shuttle: %s\n", a.reachName, pinName)
				st.shuttle++
			}

		default:
			fmt.Printf("  ⚠  [%s] %q — unknown type, skipping\n", a.reachName, pm.Name)
		}
	}

	fmt.Println()
	for slug, st := range stats {
		fmt.Printf("  %s — rapids=%d put-ins=%d take-outs=%d parking=%d shuttle=%d\n",
			st.name, st.rapids, st.putIns, st.takeOuts, st.parking, st.shuttle)
		if st.hasPutIn && st.hasTakeOut {
			centerlineReaches = append(centerlineReaches, slug)
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

type reachInfo struct {
	id       string
	slug     string
	name     string
	keywords []string // lowercased words to match in pin names
}

type importer struct {
	pool    *pgxpool.Pool
	dryRun  bool
	reaches []reachInfo // cached for category-map mode
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

// genericGeoWords are common geographic/descriptor words that appear in many
// reach names but should NOT be used alone to identify a specific reach.
var genericGeoWords = map[string]bool{
	"river": true, "creek": true, "canyon": true, "falls": true,
	"lake": true, "park": true, "south": true, "north": true,
	"upper": true, "lower": true, "east": true, "west": true,
	"fork": true, "run": true, "gorge": true, "section": true,
	"whitewater": true, "town": true, "platte": true, "arkansas": true,
	"rapids": true, "reach": true, "class": true, "buena": true,
	"vista": true, "brown": true, "royal": true, "chutes": true,
	"slide": true, "wave": true, "hole": true, "drop": true,
}

// loadReaches caches all reach names/slugs for category-map matching.
func (imp *importer) loadReaches(ctx context.Context) ([]reachInfo, error) {
	rows, err := imp.pool.Query(ctx, `SELECT id, slug, name FROM reaches ORDER BY LENGTH(name) DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []reachInfo
	for rows.Next() {
		var r reachInfo
		if err := rows.Scan(&r.id, &r.slug, &r.name); err != nil {
			return nil, err
		}
		// Build keyword list: full name + distinctive words only.
		// Skip generic geographic words that would cause false positives
		// (e.g. "park" matching "Whitewater Park", "creek" matching "Clear Creek Canyon").
		lower := strings.ToLower(r.name)
		r.keywords = []string{lower}
		for _, w := range strings.Fields(lower) {
			if len(w) >= 4 && !genericGeoWords[w] {
				r.keywords = append(r.keywords, w)
			}
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// inferReachFromText scans text for any known reach name keyword and returns
// the best match. Loads and caches the reach list on first call.
func (imp *importer) inferReachFromText(ctx context.Context, text string) (id, slug, name string, err error) {
	if imp.reaches == nil {
		imp.reaches, err = imp.loadReaches(ctx)
		if err != nil {
			return "", "", "", err
		}
	}
	lower := strings.ToLower(text)
	for _, r := range imp.reaches {
		for _, kw := range r.keywords {
			if strings.Contains(lower, kw) {
				return r.id, r.slug, r.name, nil
			}
		}
	}
	return "", "", "", fmt.Errorf("no reach match found in %q", text)
}

// isCategoryMap returns true when all folders are generic type names rather
// than reach names (e.g. "Access Points", "Rivers", "Rapids").
func isCategoryMap(folders []kmlFolder) bool {
	typeNames := map[string]bool{
		"access points": true, "access": true,
		"rivers": true, "waterways": true, "river lines": true,
		"rapids": true, "features": true,
	}
	if len(folders) == 0 {
		return false
	}
	for _, f := range folders {
		if !typeNames[strings.ToLower(f.Name)] {
			return false
		}
	}
	return true
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
	// PostgreSQL UPDATE doesn't support ORDER BY/LIMIT directly — use a CTE
	// to identify the single row to update, then join on its primary key.
	tag, err := imp.pool.Exec(ctx, `
		WITH nearest AS (
			SELECT id
			FROM reach_access
			WHERE reach_id = $1
			  AND parking_location IS NULL
			  AND ST_DWithin(
			        location,
			        ST_SetSRID(ST_MakePoint($2, $3), 4326)::geography,
			        500
			      )
			ORDER BY ST_Distance(location, ST_SetSRID(ST_MakePoint($2, $3), 4326)::geography)
			LIMIT 1
		)
		UPDATE reach_access
		SET parking_location = ST_SetSRID(ST_MakePoint($2, $3), 4326)::geography
		WHERE id IN (SELECT id FROM nearest)
	`, reachID, lon, lat)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		// No nearby access point — insert standalone parking row.
		// No access point nearby — insert as 'intermediate' access with
		// parking_location set (schema has no standalone parking type).
		_, err = imp.pool.Exec(ctx, `
			INSERT INTO reach_access
				(reach_id, access_type, name, notes,
				 location, parking_location, data_source, verified)
			VALUES
				($1, 'intermediate', $2, NULLIF($3, ''),
				 ST_SetSRID(ST_MakePoint($4, $5), 4326)::geography,
				 ST_SetSRID(ST_MakePoint($4, $5), 4326)::geography,
				 'import', true)
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
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	// KMZ is a ZIP archive — extract doc.kml from inside it.
	if strings.HasSuffix(strings.ToLower(path), ".kmz") {
		zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
		if err != nil {
			return nil, fmt.Errorf("open kmz: %w", err)
		}
		for _, f := range zr.File {
			if strings.HasSuffix(strings.ToLower(f.Name), ".kml") {
				rc, err := f.Open()
				if err != nil {
					return nil, fmt.Errorf("open %s inside kmz: %w", f.Name, err)
				}
				data, err = io.ReadAll(rc)
				rc.Close()
				if err != nil {
					return nil, fmt.Errorf("read %s inside kmz: %w", f.Name, err)
				}
				break
			}
		}
	}

	type xmlPoint struct {
		Coordinates string `xml:"coordinates"`
	}
	type xmlPlacemark struct {
		Name        string    `xml:"name"`
		Description string    `xml:"description"`
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
	if err := xml.NewDecoder(bytes.NewReader(data)).Decode(&raw); err != nil {
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

// splitPrefixWithHint wraps splitPrefix, using the KML folder name and pin
// description as fallback hints when the pin name alone is ambiguous.
// folderHint is the category folder name (e.g. "Rapids", "Access Points").
func splitPrefixWithHint(name, description, folderHint string) (prefix, rest string) {
	prefix, rest = splitPrefix(name)
	if prefix != "" {
		return
	}

	// Check description for type keywords.
	// Parking is checked before put-in since many access points mention both
	// ("can park as well") but the primary use determines type.
	descLower := strings.ToLower(description)
	switch {
	case strings.Contains(descLower, "parking") || strings.Contains(descLower, "can park") ||
		strings.Contains(descLower, "park as well") || strings.Contains(descLower, "park here"):
		return "parking", name
	case strings.Contains(descLower, "take-out") || strings.Contains(descLower, "takeout") ||
		strings.Contains(descLower, "take out"):
		return "take-out", name
	case strings.Contains(descLower, "put-in") || strings.Contains(descLower, "put in") ||
		strings.Contains(descLower, "put_in"):
		return "put-in", name
	case strings.Contains(descLower, "class") || strings.Contains(descLower, "line is") ||
		strings.Contains(descLower, "boof") || strings.Contains(descLower, "ledge"):
		return "rapid", name
	}

	// Fall back to folder name hint.
	switch strings.ToLower(folderHint) {
	case "rapids":
		return "rapid", name
	case "access points", "access":
		return "put-in", name // treat ambiguous access as put-in
	}

	return "", name
}

// splitPrefix splits "Rapid: Zoom Flume" into ("rapid", "Zoom Flume").
// Falls back to keyword detection for maps without the colon convention
// (e.g. "Put-In", "Take-out at Gravel Ponds", "Waterton Canyon Parking").
// Returns ("", original) if no type can be determined.
func splitPrefix(name string) (prefix, rest string) {
	lower := strings.ToLower(name)

	// Explicit colon prefix — highest priority.
	for _, p := range []string{"Rapid", "Put-in", "Take-out", "Parking", "Shuttle"} {
		if strings.HasPrefix(lower, strings.ToLower(p)+":") {
			rest = strings.TrimSpace(name[len(p)+1:])
			return strings.ToLower(p), rest
		}
	}

	// Keyword fallback — order matters (check more specific terms first).
	// Use Contains so "Deckers Put-in", "Standard Foxton Takeout", etc. work.
	switch {
	case strings.Contains(lower, "put-in") || strings.Contains(lower, "put in") ||
		strings.Contains(lower, "putin") || strings.Contains(lower, "put_in"):
		return "put-in", name

	case strings.Contains(lower, "take-out") || strings.Contains(lower, "takeout") ||
		strings.Contains(lower, "take out") || strings.Contains(lower, "takout"):
		return "take-out", name

	case strings.Contains(lower, "parking") || strings.Contains(lower, "trailhead"):
		return "parking", name

	case strings.Contains(lower, "shuttle"):
		return "shuttle", name

	case strings.Contains(lower, "rapid") || strings.Contains(lower, "falls") ||
		strings.Contains(lower, "drop") || strings.Contains(lower, "hole"):
		return "rapid", name
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
