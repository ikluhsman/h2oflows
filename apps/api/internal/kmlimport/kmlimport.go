// Package kmlimport imports reach features from a Google My Maps KML/KMZ export.
//
// Map conventions supported:
//   - Folder-per-reach maps: one folder per reach, folder name matched to DB
//   - Category-organized maps: folders named "Access Points", "Rivers", "Rapids"
//     with reach inferred by pin name + geographic proximity
//
// Pin name prefix → feature type:
//
//	"Rapid: <name>"    → rapids
//	"Put-in: <name>"   → reach_access type=put_in
//	"Take-out: <name>" → reach_access type=take_out
//	"Parking: <name>"  → reach_access.parking_location on nearest access
//	"Shuttle: <name>"  → reach_access type=shuttle_drop
package kmlimport

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"math"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

// ── Result types ─────────────────────────────────────────────────────────────

// Result summarises what was imported.
type Result struct {
	MapName string                   `json:"map_name"`
	Reaches map[string]*ReachResult  `json:"reaches"` // keyed by reach slug
	Log     []string                 `json:"log"`
}

// ReachResult holds per-reach counts.
type ReachResult struct {
	Name     string   `json:"name"`
	Rapids   int      `json:"rapids"`
	PutIns   int      `json:"put_ins"`
	TakeOuts int      `json:"take_outs"`
	Parking  int      `json:"parking"`
	Shuttle  int      `json:"shuttle"`
	Errors   []string `json:"errors,omitempty"`
}

// ── KML types ─────────────────────────────────────────────────────────────────

// KMLDoc is the parsed representation of a KML/KMZ file.
type KMLDoc struct {
	Name    string
	Folders []KMLFolder
}

// KMLFolder is a single layer/folder in the KML.
type KMLFolder struct {
	Name       string
	Placemarks []KMLPlacemark
}

// KMLPlacemark is a single pin or shape.
type KMLPlacemark struct {
	Name        string
	Description string
	Point       *KMLPoint // nil for LineStrings/Polygons
}

// KMLPoint holds the parsed coordinate string.
type KMLPoint struct {
	Coordinates string
}

// ── ParseKMLBytes ──────────────────────────────────────────────────────────────

// ParseKMLBytes parses a KML or KMZ file from raw bytes.
func ParseKMLBytes(data []byte) (*KMLDoc, error) {
	// KMZ is a ZIP archive — extract the first .kml file inside.
	if isZIP(data) {
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

	doc := &KMLDoc{Name: raw.Document.Name}
	for _, xf := range raw.Document.Folders {
		folder := KMLFolder{Name: xf.Name}
		for _, xp := range xf.Placemarks {
			pm := KMLPlacemark{
				Name:        strings.TrimSpace(xp.Name),
				Description: StripHTML(strings.TrimSpace(xp.Description)),
			}
			if xp.Point != nil {
				pm.Point = &KMLPoint{Coordinates: strings.TrimSpace(xp.Point.Coordinates)}
			}
			folder.Placemarks = append(folder.Placemarks, pm)
		}
		doc.Folders = append(doc.Folders, folder)
	}
	return doc, nil
}

// isZIP checks the ZIP magic bytes.
func isZIP(data []byte) bool {
	return len(data) >= 4 && data[0] == 'P' && data[1] == 'K' && data[2] == 0x03 && data[3] == 0x04
}

// ── Importer ──────────────────────────────────────────────────────────────────

type reachInfo struct {
	id       string
	slug     string
	name     string
	keywords []string
}

// Importer runs KML imports against a live database pool.
type Importer struct {
	pool   *pgxpool.Pool
	DryRun bool
	reaches []reachInfo // cached for category-map mode
}

// New creates a new Importer.
func New(pool *pgxpool.Pool, dryRun bool) *Importer {
	return &Importer{pool: pool, DryRun: dryRun}
}

// Import processes all placemarks in doc and writes reach features to the DB.
func (imp *Importer) Import(ctx context.Context, doc *KMLDoc) (*Result, error) {
	res := &Result{
		MapName: doc.Name,
		Reaches: map[string]*ReachResult{},
	}

	type assignment struct {
		reachID    string
		reachSlug  string
		reachName  string
		pm         KMLPlacemark
		folderName string
	}
	var pins []assignment

	if IsCategoryMap(doc.Folders) {
		res.Log = append(res.Log, "category-organized map — inferring reach from pin names + geography")

		type pendingPin struct {
			pm         KMLPlacemark
			folderName string
			lon, lat   float64
			matched    bool
		}
		var pending []pendingPin
		for _, folder := range doc.Folders {
			for _, pm := range folder.Placemarks {
				if pm.Point == nil {
					continue
				}
				lon, lat, ok := ParseCoords(pm.Point.Coordinates)
				if !ok {
					continue
				}
				pending = append(pending, pendingPin{pm: pm, folderName: folder.Name, lon: lon, lat: lat})
			}
		}

		type geoAnchor struct {
			id, slug, name string
			lon, lat       float64
		}
		var anchors []geoAnchor

		// Pass 1: name-based matching.
		for i := range pending {
			pp := &pending[i]
			rid, rslug, rname, err := imp.inferReachFromText(ctx, pp.pm.Name+" "+pp.pm.Description)
			if err != nil {
				continue
			}
			pp.matched = true
			pins = append(pins, assignment{rid, rslug, rname, pp.pm, pp.folderName})
			anchors = append(anchors, geoAnchor{rid, rslug, rname, pp.lon, pp.lat})
		}

		// Pass 2: proximity-based fallback.
		for i := range pending {
			pp := &pending[i]
			if pp.matched || len(anchors) == 0 {
				if !pp.matched {
					res.Log = append(res.Log, fmt.Sprintf("⚠  %q — no anchors, skipping", pp.pm.Name))
				}
				continue
			}
			best := anchors[0]
			bestDist := sq(anchors[0].lon-pp.lon) + sq(anchors[0].lat-pp.lat)
			for _, a := range anchors[1:] {
				if d := sq(a.lon-pp.lon) + sq(a.lat-pp.lat); d < bestDist {
					bestDist = d
					best = a
				}
			}
			res.Log = append(res.Log, fmt.Sprintf("~ %q → %s (by proximity)", pp.pm.Name, best.name))
			pins = append(pins, assignment{best.id, best.slug, best.name, pp.pm, pp.folderName})
		}
	} else {
		for _, folder := range doc.Folders {
			rid, rslug, rname, err := imp.matchReach(ctx, folder.Name)
			if err != nil {
				res.Log = append(res.Log, fmt.Sprintf("⚠  folder %q — no matching reach, skipping", folder.Name))
				continue
			}
			for _, pm := range folder.Placemarks {
				pins = append(pins, assignment{rid, rslug, rname, pm, ""})
			}
		}
	}

	for _, a := range pins {
		pm := a.pm
		if pm.Point == nil {
			continue
		}
		lon, lat, ok := ParseCoords(pm.Point.Coordinates)
		if !ok {
			res.Log = append(res.Log, fmt.Sprintf("⚠  %q — bad coordinates", pm.Name))
			continue
		}

		st := res.reachStats(a.reachSlug, a.reachName)
		prefix, pinName := SplitPrefixWithHint(pm.Name, pm.Description, a.folderName)
		desc := strings.TrimSpace(pm.Description)

		switch prefix {
		case "rapid", "wave":
			isSurf := prefix == "wave"
			if err := imp.upsertRapidLocation(ctx, a.reachID, pinName, desc, isSurf, lon, lat); err != nil {
				st.Errors = append(st.Errors, fmt.Sprintf("rapid %q: %v", pinName, err))
				res.Log = append(res.Log, fmt.Sprintf("✗ [%s] rapid %q: %v", a.reachName, pinName, err))
			} else {
				st.Rapids++
				if isSurf {
					res.Log = append(res.Log, fmt.Sprintf("✓ [%s] wave: %s", a.reachName, pinName))
				} else {
					res.Log = append(res.Log, fmt.Sprintf("✓ [%s] rapid: %s", a.reachName, pinName))
				}
			}
		case "put-in":
			if err := imp.upsertAccess(ctx, a.reachID, "put_in", pinName, desc, lon, lat); err != nil {
				st.Errors = append(st.Errors, fmt.Sprintf("put-in %q: %v", pinName, err))
				res.Log = append(res.Log, fmt.Sprintf("✗ [%s] put-in %q: %v", a.reachName, pinName, err))
			} else {
				st.PutIns++
				res.Log = append(res.Log, fmt.Sprintf("✓ [%s] put-in: %s", a.reachName, pinName))
			}
		case "take-out":
			if err := imp.upsertAccess(ctx, a.reachID, "take_out", pinName, desc, lon, lat); err != nil {
				st.Errors = append(st.Errors, fmt.Sprintf("take-out %q: %v", pinName, err))
				res.Log = append(res.Log, fmt.Sprintf("✗ [%s] take-out %q: %v", a.reachName, pinName, err))
			} else {
				st.TakeOuts++
				res.Log = append(res.Log, fmt.Sprintf("✓ [%s] take-out: %s", a.reachName, pinName))
			}
		case "parking":
			if err := imp.upsertParking(ctx, a.reachID, pinName, desc, lon, lat); err != nil {
				st.Errors = append(st.Errors, fmt.Sprintf("parking %q: %v", pinName, err))
				res.Log = append(res.Log, fmt.Sprintf("✗ [%s] parking %q: %v", a.reachName, pinName, err))
			} else {
				st.Parking++
				res.Log = append(res.Log, fmt.Sprintf("✓ [%s] parking: %s", a.reachName, pinName))
			}
		case "shuttle":
			if err := imp.upsertAccess(ctx, a.reachID, "shuttle_drop", pinName, desc, lon, lat); err != nil {
				st.Errors = append(st.Errors, fmt.Sprintf("shuttle %q: %v", pinName, err))
				res.Log = append(res.Log, fmt.Sprintf("✗ [%s] shuttle %q: %v", a.reachName, pinName, err))
			} else {
				st.Shuttle++
				res.Log = append(res.Log, fmt.Sprintf("✓ [%s] shuttle: %s", a.reachName, pinName))
			}
		default:
			res.Log = append(res.Log, fmt.Sprintf("⚠  [%s] %q — unknown type, skipping", a.reachName, pm.Name))
		}
	}

	return res, nil
}

// reachStats returns-or-creates a ReachResult for the given reach.
func (res *Result) reachStats(slug, name string) *ReachResult {
	if _, ok := res.Reaches[slug]; !ok {
		res.Reaches[slug] = &ReachResult{Name: name}
	}
	return res.Reaches[slug]
}

// ── Reach matching ────────────────────────────────────────────────────────────

func (imp *Importer) matchReach(ctx context.Context, folderName string) (id, slug, name string, err error) {
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

// genericGeoWords are words that appear in many reach names but shouldn't
// be used alone to identify a specific reach (prevents false-positive matches).
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

func (imp *Importer) loadReaches(ctx context.Context) ([]reachInfo, error) {
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

func (imp *Importer) inferReachFromText(ctx context.Context, text string) (id, slug, name string, err error) {
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

// ── DB upserts ────────────────────────────────────────────────────────────────

func (imp *Importer) upsertRapidLocation(ctx context.Context, reachID, name, desc string, isSurfWave bool, lon, lat float64) error {
	if imp.DryRun {
		return nil
	}
	classRating := ParseClassRating(desc)
	tag, err := imp.pool.Exec(ctx, `
		UPDATE rapids
		SET location     = ST_SetSRID(ST_MakePoint($3, $4), 4326)::geography,
		    description  = CASE WHEN $5 <> '' THEN $5 ELSE description END,
		    class_rating = CASE WHEN $6::numeric IS NOT NULL THEN $6::numeric ELSE class_rating END,
		    is_surf_wave = is_surf_wave OR $7
		WHERE reach_id = $1 AND LOWER(name) = LOWER($2)
	`, reachID, name, lon, lat, desc, classRating, isSurfWave)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		_, err = imp.pool.Exec(ctx, `
			INSERT INTO rapids (reach_id, name, location, description, class_rating, is_surf_wave, data_source, verified)
			VALUES ($1, $2, ST_SetSRID(ST_MakePoint($3, $4), 4326)::geography, NULLIF($5,''), $6::numeric, $7, 'import', true)
			ON CONFLICT (reach_id, name) DO UPDATE
			  SET location     = EXCLUDED.location,
			      description  = COALESCE(EXCLUDED.description, rapids.description),
			      class_rating = COALESCE(EXCLUDED.class_rating, rapids.class_rating),
			      is_surf_wave = rapids.is_surf_wave OR EXCLUDED.is_surf_wave
		`, reachID, name, lon, lat, desc, classRating, isSurfWave)
	}
	return err
}

func (imp *Importer) upsertAccess(ctx context.Context, reachID, accessType, name, notes string, lon, lat float64) error {
	if imp.DryRun {
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

func (imp *Importer) upsertParking(ctx context.Context, reachID, name, notes string, lon, lat float64) error {
	if imp.DryRun {
		return nil
	}
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

// ── Parsing helpers ───────────────────────────────────────────────────────────

// IsCategoryMap returns true when all folders have generic type names
// ("Access Points", "Rivers", "Rapids") rather than reach-specific names.
func IsCategoryMap(folders []KMLFolder) bool {
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

// SplitPrefixWithHint wraps SplitPrefix with folder-name and description hints.
func SplitPrefixWithHint(name, description, folderHint string) (prefix, rest string) {
	prefix, rest = SplitPrefix(name)
	if prefix != "" {
		return
	}
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
	case strings.Contains(descLower, "surf wave") || strings.Contains(descLower, "surf spot") ||
		strings.Contains(descLower, "surfable") || strings.Contains(descLower, "play wave"):
		return "wave", name
	case strings.Contains(descLower, "class") || strings.Contains(descLower, "line is") ||
		strings.Contains(descLower, "boof") || strings.Contains(descLower, "ledge"):
		return "rapid", name
	}
	switch strings.ToLower(folderHint) {
	case "rapids", "waves", "surf waves":
		return "rapid", name
	case "access points", "access":
		return "put-in", name
	}
	return "", name
}

// SplitPrefix splits "Rapid: Zoom Flume" → ("rapid", "Zoom Flume").
func SplitPrefix(name string) (prefix, rest string) {
	lower := strings.ToLower(name)
	for _, p := range []string{"Rapid", "Wave", "Surf", "Put-in", "Take-out", "Parking", "Shuttle"} {
		if strings.HasPrefix(lower, strings.ToLower(p)+":") {
			prefix := strings.ToLower(p)
			if prefix == "surf" {
				prefix = "wave"
			}
			return prefix, strings.TrimSpace(name[len(p)+1:])
		}
	}
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

// ParseCoords parses "lon,lat[,alt]" from a KML coordinates string.
func ParseCoords(raw string) (lon, lat float64, ok bool) {
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

// ParseClassRating extracts a numeric class from text like "Class III+" → 3.5.
func ParseClassRating(desc string) *float64 {
	lower := strings.ToLower(desc)
	idx := strings.Index(lower, "class ")
	if idx < 0 {
		return nil
	}
	rest := lower[idx+6:]
	var base float64
	switch {
	case strings.HasPrefix(rest, "v"):
		base, rest = 5, rest[1:]
	case strings.HasPrefix(rest, "iv"):
		base, rest = 4, rest[2:]
	case strings.HasPrefix(rest, "iii"):
		base, rest = 3, rest[3:]
	case strings.HasPrefix(rest, "ii"):
		base, rest = 2, rest[2:]
	case strings.HasPrefix(rest, "i"):
		base, rest = 1, rest[1:]
	default:
		return nil
	}
	if strings.HasPrefix(rest, "+") {
		base += 0.5
	}
	return &base
}

// StripHTML removes basic HTML tags from Google Maps description fields.
func StripHTML(s string) string {
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

// SyncCenterline fetches the OSM river geometry between a reach's put-in and
// take-out and stores it as reaches.centerline.
func SyncCenterline(ctx context.Context, pool *pgxpool.Pool, slug string, dryRun bool) error {
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

	minLon := math.Min(putInLon, takeOutLon) - 0.05
	maxLon := math.Max(putInLon, takeOutLon) + 0.05
	minLat := math.Min(putInLat, takeOutLat) - 0.05
	maxLat := math.Max(putInLat, takeOutLat) + 0.05

	geojson, err := fetchOSMRiverLine(ctx, minLon, minLat, maxLon, maxLat)
	if err != nil {
		return fmt.Errorf("osm fetch: %w", err)
	}
	if geojson == "" {
		return fmt.Errorf("no river waterway found in bbox")
	}
	if dryRun {
		return nil
	}

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
	return err
}

func sq(x float64) float64 { return x * x }
