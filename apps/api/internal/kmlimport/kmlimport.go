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
//	"Wave: <name>"     → rapids (is_surf_wave=true)
//	"Surf: <name>"     → rapids (is_surf_wave=true)
//	"Put-in: <name>"   → reach_access type=put_in
//	"Take-out: <name>" → reach_access type=take_out
//	"Parking: <name>"  → reach_access type=parking
//	"Hazard: <name>"   → rapids (is_permanent_hazard=true)
//
// Hazard descriptions may include a hazard type keyword to classify them:
//
//	"low-head dam", "lowhead", "dam" → hazard_type="low_head_dam"
//	"rebar", "rebar/concrete"        → hazard_type="rebar"
//	"strainer"                        → hazard_type="strainer"
//	"bridge"                          → hazard_type="bridge_piling"
//	(default)                         → hazard_type="other"
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
	Name      string   `json:"name"`
	Rapids    int      `json:"rapids"`
	Hazards   int      `json:"hazards"`
	PutIns    int      `json:"put_ins"`
	TakeOuts  int      `json:"take_outs"`
	Parking   int      `json:"parking"`
	Campsites int      `json:"campsites"`
	Errors    []string `json:"errors,omitempty"`
}

// ── KML types ─────────────────────────────────────────────────────────────────

// KMLDoc is the parsed representation of a KML/KMZ file.
type KMLDoc struct {
	Name        string
	Description string // optional — may contain "Basin: South Platte" etc.
	Folders     []KMLFolder
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
	Point       *KMLPoint      // nil for LineStrings/Polygons
	LineString  *KMLLineString // non-nil when placemark is a LineString
}

// KMLPoint holds the parsed coordinate string.
type KMLPoint struct {
	Coordinates string
}

// KMLLineString holds raw KML coordinates for a LineString placemark.
type KMLLineString struct {
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
	type xmlLineString struct {
		Coordinates string `xml:"coordinates"`
	}
	type xmlPlacemark struct {
		Name        string         `xml:"name"`
		Description string         `xml:"description"`
		Point       *xmlPoint      `xml:"Point"`
		LineString  *xmlLineString `xml:"LineString"`
	}
	// xmlFolder is declared as a named type so it can reference itself for
	// nested sub-folders (Google My Maps sometimes wraps all reaches in one
	// outer folder; we need to recurse into those).
	type xmlFolder struct {
		Name       string         `xml:"name"`
		Placemarks []xmlPlacemark `xml:"Placemark"`
		SubFolders []xmlFolder    `xml:"Folder"`
	}
	type xmlDocument struct {
		Name        string      `xml:"name"`
		Description string      `xml:"description"`
		Folders     []xmlFolder `xml:"Folder"`
	}
	type xmlKML struct {
		Document xmlDocument `xml:"Document"`
	}

	var raw xmlKML
	if err := xml.NewDecoder(bytes.NewReader(data)).Decode(&raw); err != nil {
		return nil, err
	}

	// convertPM converts a raw xmlPlacemark to a KMLPlacemark.
	convertPM := func(xp xmlPlacemark) KMLPlacemark {
		pm := KMLPlacemark{
			Name:        strings.TrimSpace(xp.Name),
			Description: StripHTML(strings.TrimSpace(xp.Description)),
		}
		if xp.Point != nil {
			pm.Point = &KMLPoint{Coordinates: strings.TrimSpace(xp.Point.Coordinates)}
		}
		if xp.LineString != nil {
			pm.LineString = &KMLLineString{Coordinates: strings.TrimSpace(xp.LineString.Coordinates)}
		}
		return pm
	}

	// flattenFolders recursively collects reach folders.
	// A folder that has sub-folders but no placemarks is treated as a wrapper
	// and replaced by its children (one-level KML nesting is common in
	// Google My Maps exports where the user organises reaches inside a layer).
	// A folder that has placemarks (with or without sub-folders) is kept as-is;
	// any sub-folders it also has are then flattened separately.
	var flattenFolders func([]xmlFolder) []KMLFolder
	flattenFolders = func(folders []xmlFolder) []KMLFolder {
		var out []KMLFolder
		for _, xf := range folders {
			hasPins     := len(xf.Placemarks) > 0
			hasSubs     := len(xf.SubFolders) > 0

			if hasPins {
				// This folder has placemarks — treat it as a reach folder.
				kf := KMLFolder{Name: xf.Name}
				for _, xp := range xf.Placemarks {
					kf.Placemarks = append(kf.Placemarks, convertPM(xp))
				}
				out = append(out, kf)
			}
			if hasSubs {
				// Recurse into sub-folders (whether or not this folder also had pins).
				out = append(out, flattenFolders(xf.SubFolders)...)
			}
		}
		return out
	}

	doc := &KMLDoc{
		Name:        raw.Document.Name,
		Description: StripHTML(strings.TrimSpace(raw.Document.Description)),
	}
	doc.Folders = flattenFolders(raw.Document.Folders)
	return doc, nil
}

// isZIP checks the ZIP magic bytes.
func isZIP(data []byte) bool {
	return len(data) >= 4 && data[0] == 'P' && data[1] == 'K' && data[2] == 0x03 && data[3] == 0x04
}

// kmlCoordsToLineStringGeoJSON converts a KML coordinate string (space/newline-
// separated "lng,lat,ele" triples) to a GeoJSON LineString JSON string.
func kmlCoordsToLineStringGeoJSON(raw string) (string, error) {
	var coords [][2]float64
	for _, token := range strings.Fields(raw) {
		parts := strings.Split(token, ",")
		if len(parts) < 2 {
			continue
		}
		lng, err1 := strconv.ParseFloat(strings.TrimSpace(parts[0]), 64)
		lat, err2 := strconv.ParseFloat(strings.TrimSpace(parts[1]), 64)
		if err1 != nil || err2 != nil {
			continue
		}
		coords = append(coords, [2]float64{lng, lat})
	}
	if len(coords) < 2 {
		return "", fmt.Errorf("LineString has fewer than 2 valid coordinates")
	}
	// Build GeoJSON manually to avoid importing encoding/json at the top level.
	var sb strings.Builder
	sb.WriteString(`{"type":"LineString","coordinates":[`)
	for i, c := range coords {
		if i > 0 {
			sb.WriteByte(',')
		}
		sb.WriteString(fmt.Sprintf("[%g,%g]", c[0], c[1]))
	}
	sb.WriteString(`]}`)
	return sb.String(), nil
}

// ── Importer ──────────────────────────────────────────────────────────────────

// Importer runs KML imports against a live database pool.
type Importer struct {
	pool    *pgxpool.Pool
	DryRun  bool
	cleared map[string]bool // reaches whose import data has been cleared this run
}

// New creates a new Importer.
func New(pool *pgxpool.Pool, dryRun bool) *Importer {
	return &Importer{pool: pool, DryRun: dryRun, cleared: map[string]bool{}}
}

// Import processes all placemarks in doc and writes reach features to the DB.
//
// Each KML folder must contain a coordinate-less "slug" placemark whose
// description matches an existing reach slug. Folders without a slug placemark
// are skipped. Centerlines and river associations set via the admin NLDI flow
// are never overwritten — this function is metadata-only.
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

	type reachFlowRange struct {
		reachID   string
		reachName string
		label     string
		minCFS    *float64
		maxCFS    *float64
	}
	type gaugeAssoc struct {
		reachID    string
		reachName  string
		externalID string
	}
	var flowRanges []reachFlowRange
	var gaugeAssocs []gaugeAssoc

	// Extract document-level basin from description (e.g. "Basin: South Platte").
	// Per-folder metadata placemarks take precedence; this is just the default.
	var docBasin string
	for _, line := range strings.Split(doc.Description, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(strings.ToLower(line), "basin:") {
			docBasin = strings.TrimSpace(line[len("basin:"):])
			break
		}
	}
	if docBasin != "" {
		res.Log = append(res.Log, fmt.Sprintf("~ document basin: %q", docBasin))
	}

	for _, folder := range doc.Folders {
		var (
			folderSlug     string
			commonName     string
			reachDesc      string
			classMin       *float64
			classMax       *float64
			gaugeExtID     string
			basinGroup     = docBasin
			permitRequired *bool
			multiDayDays   *int
		)
		var folderFlowRanges []reachFlowRange
		var folderPins []KMLPlacemark

		for _, pm := range folder.Placemarks {
			// Ignore LineString geometry — centerlines are sourced from NLDI, not KML.
			if pm.LineString != nil {
				continue
			}
			if pm.Point == nil {
				key := strings.ToLower(strings.TrimSpace(pm.Name))
				val := strings.TrimSpace(pm.Description)
				switch key {
				case "slug":
					folderSlug = val
				case "common_name":
					commonName = val
				case "description":
					reachDesc = val
				case "min_class":
					if v, err2 := strconv.ParseFloat(val, 64); err2 == nil {
						classMin = &v
					}
				case "max_class":
					if v, err2 := strconv.ParseFloat(val, 64); err2 == nil {
						classMax = &v
					}
				case "gauge":
					gaugeExtID = val
				case "basin":
					basinGroup = val
				case "permit_required":
					b := strings.ToLower(val) == "true"
					permitRequired = &b
				case "multi_day":
					if v, err2 := strconv.Atoi(val); err2 == nil {
						multiDayDays = &v
					}
				default:
					if label, minCFS, maxCFS, ok := parseFlowRangePM(pm.Name, pm.Description); ok {
						folderFlowRanges = append(folderFlowRanges, reachFlowRange{"", "", label, minCFS, maxCFS})
					}
				}
				continue
			}
			folderPins = append(folderPins, pm)
		}

		// Slug is required — fail this folder (not the whole import) if missing.
		if folderSlug == "" {
			res.Log = append(res.Log, fmt.Sprintf("⚠  folder %q — missing slug placemark; create the reach in admin first (skipping)", folder.Name))
			continue
		}
		var rid, rslug, rname string
		if err := imp.pool.QueryRow(ctx,
			`SELECT id, slug, name FROM reaches WHERE slug = $1`, folderSlug,
		).Scan(&rid, &rslug, &rname); err != nil {
			res.Log = append(res.Log, fmt.Sprintf("⚠  folder %q — reach %q not found in database; create it in admin first (skipping)", folder.Name, folderSlug))
			continue
		}
		res.Log = append(res.Log, fmt.Sprintf("~ folder %q → matched reach %q (%s)", folder.Name, rname, rslug))

		// Apply metadata overrides the KML provides.
		if !imp.DryRun {
			_, _ = imp.pool.Exec(ctx, `
				UPDATE reaches SET
					common_name = COALESCE(NULLIF($2, ''), common_name),
					class_min   = COALESCE($3::numeric, class_min),
					class_max   = COALESCE($4::numeric, class_max)
				WHERE id = $1
			`, rid, commonName, classMin, classMax)
		}

		// Overwrite description when KML provides one.
		if reachDesc != "" {
			if _, err := imp.pool.Exec(ctx, `UPDATE reaches SET description = $1 WHERE id = $2`, reachDesc, rid); err != nil {
				res.Log = append(res.Log, fmt.Sprintf("⚠  [%s] description update failed: %v", rname, err))
			} else {
				res.Log = append(res.Log, fmt.Sprintf("~ [%s] description updated", rname))
			}
		}

		// Upsert the river from the KML document name and link only if the reach
		// has no existing river association — preserves admin-set river links.
		if doc.Name != "" && !imp.DryRun {
			riverSlug := slugify(doc.Name)
			if basinGroup != "" {
				riverSlug = riverSlug + "-" + slugify(basinGroup)
			}
			var riverID string
			err := imp.pool.QueryRow(ctx, `
				INSERT INTO rivers (slug, name, basin)
				VALUES ($1, $2, NULLIF($3, ''))
				ON CONFLICT (slug) DO UPDATE
					SET basin = COALESCE(NULLIF(EXCLUDED.basin, ''), rivers.basin)
				RETURNING id
			`, riverSlug, doc.Name, basinGroup).Scan(&riverID)
			if err != nil {
				res.Log = append(res.Log, fmt.Sprintf("⚠  [%s] river upsert failed: %v", rname, err))
			} else {
				if _, err := imp.pool.Exec(ctx,
					`UPDATE reaches SET river_id = $1 WHERE id = $2 AND river_id IS NULL`,
					riverID, rid,
				); err != nil {
					res.Log = append(res.Log, fmt.Sprintf("⚠  [%s] river_id link failed: %v", rname, err))
				}
				if basinGroup != "" {
					if _, err := imp.pool.Exec(ctx,
						`UPDATE reaches SET watershed_name = $1 WHERE id = $2 AND watershed_name IS NULL`,
						basinGroup, rid,
					); err != nil {
						res.Log = append(res.Log, fmt.Sprintf("⚠  [%s] watershed_name update failed: %v", rname, err))
					}
				}
			}
		}

		if permitRequired != nil {
			if _, err := imp.pool.Exec(ctx, `UPDATE reaches SET permit_required = $1 WHERE id = $2`, *permitRequired, rid); err != nil {
				res.Log = append(res.Log, fmt.Sprintf("⚠  [%s] permit_required update failed: %v", rname, err))
			}
		}
		if multiDayDays != nil {
			if _, err := imp.pool.Exec(ctx, `UPDATE reaches SET multi_day_days = $1 WHERE id = $2`, *multiDayDays, rid); err != nil {
				res.Log = append(res.Log, fmt.Sprintf("⚠  [%s] multi_day_days update failed: %v", rname, err))
			}
		}

		for _, fr := range folderFlowRanges {
			fr.reachID = rid
			fr.reachName = rname
			flowRanges = append(flowRanges, fr)
			res.Log = append(res.Log, fmt.Sprintf("~ [%s] flow range %s", rname, fr.label))
		}
		if gaugeExtID != "" {
			gaugeAssocs = append(gaugeAssocs, gaugeAssoc{rid, rname, gaugeExtID})
			res.Log = append(res.Log, fmt.Sprintf("~ [%s] gauge %s", rname, gaugeExtID))
		}
		for _, pm := range folderPins {
			pins = append(pins, assignment{rid, rslug, rname, pm, ""})
		}
	}

	// Upsert flow ranges after reach matching so reach IDs are known.
	for _, fr := range flowRanges {
		if err := imp.upsertFlowRange(ctx, fr.reachID, fr.label, fr.minCFS, fr.maxCFS); err != nil {
			res.Log = append(res.Log, fmt.Sprintf("⚠  [%s] flow range %s: %v", fr.reachName, fr.label, err))
		}
	}

	// Associate gauges by USGS/DWR external ID.
	for _, ga := range gaugeAssocs {
		if err := imp.setReachGauge(ctx, ga.reachID, ga.externalID); err != nil {
			res.Log = append(res.Log, fmt.Sprintf("⚠  [%s] gauge %s: %v", ga.reachName, ga.externalID, err))
		} else {
			res.Log = append(res.Log, fmt.Sprintf("✓ [%s] linked gauge %s", ga.reachName, ga.externalID))
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

		// Clear existing import-sourced data for this reach on first encounter,
		// so re-importing replaces rather than accumulates.
		if !imp.cleared[a.reachID] {
			if err := imp.clearImportData(ctx, a.reachID); err != nil {
				res.Log = append(res.Log, fmt.Sprintf("⚠  [%s] clear failed: %v", a.reachName, err))
			} else {
				imp.cleared[a.reachID] = true
				res.Log = append(res.Log, fmt.Sprintf("↺  [%s] cleared previous import data", a.reachName))
			}
		}

		st := res.reachStats(a.reachSlug, a.reachName)
		prefix, pinName := SplitPrefixWithHint(pm.Name, pm.Description, a.folderName)
		desc := strings.TrimSpace(pm.Description)

		switch prefix {
		case "rapid", "wave":
			isSurf := prefix == "wave"
			if err := imp.upsertRapidLocation(ctx, a.reachID, pinName, desc, isSurf, false, "", lon, lat); err != nil {
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
		case "hazard":
			htype := inferHazardType(desc + " " + pinName)
			if err := imp.upsertRapidLocation(ctx, a.reachID, pinName, desc, false, true, htype, lon, lat); err != nil {
				st.Errors = append(st.Errors, fmt.Sprintf("hazard %q: %v", pinName, err))
				res.Log = append(res.Log, fmt.Sprintf("✗ [%s] hazard %q: %v", a.reachName, pinName, err))
			} else {
				st.Hazards++
				res.Log = append(res.Log, fmt.Sprintf("✓ [%s] hazard (%s): %s", a.reachName, htype, pinName))
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
		case "campsite":
			if err := imp.upsertAccess(ctx, a.reachID, "camp", pinName, desc, lon, lat); err != nil {
				st.Errors = append(st.Errors, fmt.Sprintf("campsite %q: %v", pinName, err))
				res.Log = append(res.Log, fmt.Sprintf("✗ [%s] campsite %q: %v", a.reachName, pinName, err))
			} else {
				st.Campsites++
				res.Log = append(res.Log, fmt.Sprintf("✓ [%s] campsite: %s", a.reachName, pinName))
			}
		default:
			res.Log = append(res.Log, fmt.Sprintf("⚠  [%s] %q — unknown type, skipping", a.reachName, pm.Name))
		}
	}

	// After all pins are inserted, derive put_in_name / take_out_name for each
	// reach that was touched, and update name = "<put_in> to <take_out>".
	seen := map[string]struct{}{}
	for _, a := range pins {
		if _, ok := seen[a.reachID]; ok {
			continue
		}
		seen[a.reachID] = struct{}{}
		if err := imp.updateReachNaming(ctx, a.reachID); err != nil {
			res.Log = append(res.Log, fmt.Sprintf("⚠  [%s] name update: %v", a.reachName, err))
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


// updateReachNaming derives put_in_name / take_out_name from the imported access
// points and updates name = "<put_in_name> to <take_out_name>" on the reach.
// Uses extreme longitudes as a proxy for upstream/downstream ordering when no
// centerline river_order is available (works for west→east rivers; good enough
// for initial import since centerline can be fetched afterward).
func (imp *Importer) updateReachNaming(ctx context.Context, reachID string) error {
	if imp.DryRun {
		return nil
	}
	var putInName, takeOutName *string
	err := imp.pool.QueryRow(ctx, `
		WITH
		  put_ins AS (
		    SELECT name FROM reach_access
		    WHERE reach_id = $1 AND access_type = 'put_in'
		    ORDER BY ST_X(location::geometry) ASC   -- westernmost = most upstream
		    LIMIT 1
		  ),
		  take_outs AS (
		    SELECT name FROM reach_access
		    WHERE reach_id = $1 AND access_type = 'take_out'
		    ORDER BY ST_X(location::geometry) DESC  -- easternmost = most downstream
		    LIMIT 1
		  )
		SELECT p.name, t.name FROM put_ins p, take_outs t
	`, reachID).Scan(&putInName, &takeOutName)
	if err != nil {
		// No put-in/take-out yet — skip; name stays as common_name for now.
		return nil
	}
	if putInName == nil || takeOutName == nil {
		return nil
	}
	_, err = imp.pool.Exec(ctx, `
		UPDATE reaches
		SET put_in_name   = $2,
		    take_out_name = $3
		WHERE id = $1
	`, reachID, *putInName, *takeOutName)
	return err
}

// slugify converts a display name to a URL-safe slug.
// "Browns Canyon" → "browns-canyon", "Cache La Poudre" → "cache-la-poudre"
// Slugify converts a string to a URL-safe slug. Exported so admin handlers
// can generate consistent slugs without re-implementing the logic.
func Slugify(s string) string { return slugify(s) }

func slugify(s string) string {
	var b strings.Builder
	prevDash := false
	for _, r := range strings.ToLower(s) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
			prevDash = false
		} else if !prevDash && b.Len() > 0 {
			b.WriteByte('-')
			prevDash = true
		}
	}
	return strings.TrimRight(b.String(), "-")
}


// ── Flow range parsing ────────────────────────────────────────────────────────

// flowRangeKeywords maps KML placemark names to flow_ranges label values.
// Accepts both the legacy 5-tier keywords and the new 4-band names as aliases.
var flowRangeKeywords = map[string]string{
	"below":     "too_low",
	"too_low":   "too_low",
	"low":       "running", // legacy alias
	"running":   "running",
	"med":       "running", // legacy alias (was the middle tier)
	"high":      "high",
	"above":     "very_high",
	"very_high": "very_high",
}

// parseFlowRangePM detects a flow-range metadata placemark and returns the
// DB label, min/max CFS, and true when the placemark name is a known keyword.
//
// Description format:
//   - "below"/"too_low" / "above"/"very_high": single CFS value — max_cfs or min_cfs respectively
//   - "running" / "high": "min,max" pair (or single value treated as min)
func parseFlowRangePM(name, desc string) (label string, minCFS, maxCFS *float64, ok bool) {
	label, ok = flowRangeKeywords[strings.ToLower(strings.TrimSpace(name))]
	if !ok {
		return "", nil, nil, false
	}
	parts := strings.SplitN(strings.TrimSpace(desc), ",", 2)
	parseVal := func(s string) *float64 {
		s = strings.TrimSpace(s)
		if s == "" {
			return nil
		}
		if v, err := strconv.ParseFloat(s, 64); err == nil {
			return &v
		}
		return nil
	}
	switch label {
	case "too_low":
		// single value = upper bound (< this)
		maxCFS = parseVal(parts[0])
	case "very_high":
		// single value = lower bound (> this)
		minCFS = parseVal(parts[0])
	default:
		// "min,max" or just "min"
		minCFS = parseVal(parts[0])
		if len(parts) == 2 {
			maxCFS = parseVal(parts[1])
		}
	}
	return label, minCFS, maxCFS, true
}

// upsertFlowRange writes a single flow range band for a reach.
// gauge_id is intentionally left NULL — KML ranges are reach-level descriptions
// not tied to a specific gauge reading source.
func (imp *Importer) upsertFlowRange(ctx context.Context, reachID, label string, minCFS, maxCFS *float64) error {
	if imp.DryRun {
		return nil
	}
	_, err := imp.pool.Exec(ctx, `
		INSERT INTO flow_ranges (reach_id, label, min_cfs, max_cfs, craft_type, data_source)
		VALUES ($1, $2, $3, $4, 'general', 'manual')
		ON CONFLICT (reach_id, label, craft_type)
		DO UPDATE SET
			min_cfs     = EXCLUDED.min_cfs,
			max_cfs     = EXCLUDED.max_cfs,
			data_source = EXCLUDED.data_source
	`, reachID, label, minCFS, maxCFS)
	return err
}

// setReachGauge links a reach to its primary gauge by the gauge's external ID.
// Accepts bare IDs ("07094500", "PLAGEOCO") or prefixed IDs ("USGS-07094500",
// "DWR-PLAGEOCO") — the prefix is stripped and used to narrow the source lookup.
func (imp *Importer) setReachGauge(ctx context.Context, reachID, externalID string) error {
	if imp.DryRun {
		return nil
	}

	// Strip optional source prefix and record it for a more precise query.
	source := ""
	bareID := externalID
	upper := strings.ToUpper(externalID)
	switch {
	case strings.HasPrefix(upper, "USGS-"):
		source = "usgs"
		bareID = externalID[5:]
	case strings.HasPrefix(upper, "DWR-"):
		source = "dwr"
		bareID = externalID[4:]
	}

	var gaugeID string
	var err error
	if source != "" {
		err = imp.pool.QueryRow(ctx, `
			SELECT id FROM gauges
			WHERE external_id = $1 AND source = $2
			LIMIT 1
		`, bareID, source).Scan(&gaugeID)
	} else {
		err = imp.pool.QueryRow(ctx, `
			SELECT id FROM gauges
			WHERE external_id = $1
			ORDER BY CASE WHEN source = 'usgs' THEN 0 ELSE 1 END
			LIMIT 1
		`, bareID).Scan(&gaugeID)
	}
	if err != nil {
		// If the caller specified an explicit source prefix (DWR- or USGS-),
		// auto-create a stub gauge row so the link works immediately.
		// The poller will populate name/lat/lng on its next cycle.
		if source != "" {
			createErr := imp.pool.QueryRow(ctx, `
				INSERT INTO gauges (external_id, source, name)
				VALUES ($1, $2, $1)
				ON CONFLICT (external_id, source) DO UPDATE SET external_id = EXCLUDED.external_id
				RETURNING id
			`, bareID, source).Scan(&gaugeID)
			if createErr != nil {
				return fmt.Errorf("gauge %q not found and auto-create failed: %w", externalID, createErr)
			}
		} else {
			return fmt.Errorf("gauge %q not found — use 'DWR-%s' or 'USGS-%s' prefix to auto-create", externalID, externalID, externalID)
		}
	}
	_, err = imp.pool.Exec(ctx, `
		UPDATE reaches SET primary_gauge_id = $1 WHERE id = $2
	`, gaugeID, reachID)
	if err != nil {
		return fmt.Errorf("set primary_gauge_id: %w", err)
	}
	_, err = imp.pool.Exec(ctx, `
		UPDATE gauges SET reach_id = $1 WHERE id = $2
	`, reachID, gaugeID)
	if err != nil {
		return err
	}
	// Best-effort: backfill reach watershed_name from the gauge's HUC-derived
	// watershed when the reach doesn't already have one. USGS gauges carry a
	// precise huc8 → CanonicalBasin value; this catches the common case where
	// a KML is imported without a `basin` metadata field.
	_, _ = imp.pool.Exec(ctx, `
		UPDATE reaches
		SET watershed_name = (SELECT watershed_name FROM gauges WHERE id = $1)
		WHERE id = $2 AND watershed_name IS NULL
	`, gaugeID, reachID)
	return nil
}

// ── DB upserts ────────────────────────────────────────────────────────────────

// stripClassSuffix removes a trailing "(IV+)" / "(III)" style class annotation
// from a rapid name so "Phone Boof (IV)" is stored as "Phone Boof".
func stripClassSuffix(name string) string {
	open := strings.LastIndex(name, "(")
	close := strings.LastIndex(name, ")")
	if open < 0 || close != len(name)-1 {
		return name
	}
	inner := strings.TrimSpace(name[open+1 : close])
	// Only strip if the parenthetical looks like a class rating.
	if ParseClassRating(inner) != nil {
		return strings.TrimSpace(name[:open])
	}
	return name
}

func (imp *Importer) upsertRapidLocation(ctx context.Context, reachID, name, desc string, isSurfWave, isPermanentHazard bool, hazardType string, lon, lat float64) error {
	if imp.DryRun {
		return nil
	}
	// Strip class suffix from name before storing ("Phone Boof (IV)" → "Phone Boof").
	// ParseClassRating still sees the full original name+desc for the rating.
	classRating := ParseClassRating(name, desc)
	cleanName := stripClassSuffix(name)
	tag, err := imp.pool.Exec(ctx, `
		UPDATE rapids
		SET location             = ST_SetSRID(ST_MakePoint($3, $4), 4326)::geography,
		    description          = CASE WHEN $5 <> '' THEN $5 ELSE description END,
		    class_rating         = CASE WHEN $6::numeric IS NOT NULL THEN $6::numeric ELSE class_rating END,
		    is_surf_wave         = is_surf_wave OR $7,
		    is_permanent_hazard  = is_permanent_hazard OR $8,
		    hazard_type          = CASE WHEN $9 <> '' THEN $9 ELSE hazard_type END
		WHERE reach_id = $1 AND LOWER(name) = LOWER($2)
	`, reachID, cleanName, lon, lat, desc, classRating, isSurfWave, isPermanentHazard, hazardType)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		_, err = imp.pool.Exec(ctx, `
			INSERT INTO rapids (reach_id, name, location, description, class_rating,
			                    is_surf_wave, is_permanent_hazard, hazard_type,
			                    data_source, verified)
			VALUES ($1, $2, ST_SetSRID(ST_MakePoint($3, $4), 4326)::geography,
			        NULLIF($5,''), $6::numeric, $7, $8, NULLIF($9,''), 'import', true)
			ON CONFLICT (reach_id, name) DO UPDATE
			  SET location            = EXCLUDED.location,
			      description         = COALESCE(EXCLUDED.description, rapids.description),
			      class_rating        = COALESCE(EXCLUDED.class_rating, rapids.class_rating),
			      is_surf_wave        = rapids.is_surf_wave OR EXCLUDED.is_surf_wave,
			      is_permanent_hazard = rapids.is_permanent_hazard OR EXCLUDED.is_permanent_hazard,
			      hazard_type         = COALESCE(EXCLUDED.hazard_type, rapids.hazard_type)
		`, reachID, cleanName, lon, lat, desc, classRating, isSurfWave, isPermanentHazard, hazardType)
	}
	return err
}

// inferHazardType classifies a permanent hazard from its name/description text.
func inferHazardType(text string) string {
	lower := strings.ToLower(text)
	switch {
	case strings.Contains(lower, "low-head") || strings.Contains(lower, "lowhead") ||
		strings.Contains(lower, "low head") || strings.Contains(lower, "weir"):
		return "low_head_dam"
	case strings.Contains(lower, "dam"):
		return "dam"
	case strings.Contains(lower, "rebar") || strings.Contains(lower, "rebar/concrete") ||
		strings.Contains(lower, "rebar / concrete"):
		return "rebar"
	case strings.Contains(lower, "strainer"):
		return "strainer"
	case strings.Contains(lower, "bridge") || strings.Contains(lower, "piling"):
		return "bridge_piling"
	default:
		return "other"
	}
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
	// Store parking pins as their own 'parking' access type rows.
	// Each pin is a distinct record, so all parking pins in the KML are preserved.
	_, err := imp.pool.Exec(ctx, `
		INSERT INTO reach_access
			(reach_id, access_type, name, notes,
			 location, parking_location, data_source, verified)
		VALUES
			($1, 'parking', $2, NULLIF($3, ''),
			 ST_SetSRID(ST_MakePoint($4, $5), 4326)::geography,
			 ST_SetSRID(ST_MakePoint($4, $5), 4326)::geography,
			 'import', true)
		ON CONFLICT (reach_id, access_type, name) DO UPDATE
		  SET location         = EXCLUDED.location,
		      parking_location = EXCLUDED.parking_location,
		      notes            = COALESCE(EXCLUDED.notes, reach_access.notes),
		      verified         = true
	`, reachID, name, notes, lon, lat)
	return err
}

// clearImportData removes all rapids and access points seeded by AI or a prior KML import
// for the given reach. Human KML imports are authoritative and supersede AI seeds.
// Records with data_source = 'maintainer' are preserved.
func (imp *Importer) clearImportData(ctx context.Context, reachID string) error {
	if imp.DryRun {
		return nil
	}
	if _, err := imp.pool.Exec(ctx,
		`DELETE FROM rapids WHERE reach_id = $1 AND data_source IN ('import', 'ai_seed')`, reachID,
	); err != nil {
		return err
	}
	_, err := imp.pool.Exec(ctx,
		`DELETE FROM reach_access WHERE reach_id = $1 AND data_source IN ('import', 'ai_seed')`, reachID,
	)
	return err
}

// ── Parsing helpers ───────────────────────────────────────────────────────────

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
	case "hazards", "permanent hazards":
		return "hazard", name
	case "campsites", "camps", "camping":
		return "campsite", name
	}
	return "", name
}

// SplitPrefix splits "Rapid: Zoom Flume" → ("rapid", "Zoom Flume").
func SplitPrefix(name string) (prefix, rest string) {
	lower := strings.ToLower(name)
	for _, p := range []string{"Rapid", "Wave", "Surf", "Put-in", "Take-out", "Parking", "Hazard", "Campsite"} {
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

// ParseClassRating extracts a numeric class rating from one or more text fields.
// Priority: parenthesized notation "(IV+)", "(III-)" > "class III+" prefix.
// Accepts variadic strings so callers can pass both name and description.
func ParseClassRating(texts ...string) *float64 {
	lower := strings.ToLower(strings.Join(texts, " "))
	if v := parseParenClass(lower); v != nil {
		return v
	}
	return parseClassPrefix(lower)
}

// parseParenClass finds the first "(IV+)" / "(III-)" / "(III)" style annotation.
func parseParenClass(lower string) *float64 {
	for i := 0; i < len(lower); i++ {
		if lower[i] != '(' {
			continue
		}
		rest := lower[i+1:]
		if strings.HasPrefix(rest, "class ") {
			rest = rest[6:]
		}
		var base float64
		var eaten int
		switch {
		case strings.HasPrefix(rest, "v"):
			base, eaten = 5, 1
		case strings.HasPrefix(rest, "iv"):
			base, eaten = 4, 2
		case strings.HasPrefix(rest, "iii"):
			base, eaten = 3, 3
		case strings.HasPrefix(rest, "ii"):
			base, eaten = 2, 2
		case strings.HasPrefix(rest, "i"):
			base, eaten = 1, 1
		default:
			continue
		}
		rest = rest[eaten:]
		if strings.HasPrefix(rest, "+") {
			base += 0.5
			rest = rest[1:]
		} else if strings.HasPrefix(rest, "-") {
			base -= 0.5
			rest = rest[1:]
		}
		if strings.HasPrefix(rest, ")") {
			return &base
		}
	}
	return nil
}

// parseClassPrefix finds "class III+" / "class IV-" style annotations.
func parseClassPrefix(lower string) *float64 {
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
	} else if strings.HasPrefix(rest, "-") {
		base -= 0.5
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

// CenterlineSource selects which upstream geometry source SyncCenterline uses.
type CenterlineSource string

const (
	CenterlineOSM  CenterlineSource = "osm"
	CenterlineNLDI CenterlineSource = "nldi"
)

// SyncCenterline fetches a river centerline from the chosen source, trims it to
// the reach's start/end via PostGIS ST_LineSubstring, and stores it on
// reaches.centerline. When source is NLDI the reach's NHD reference fields
// (start_comid, end_comid, reachcode, totdasqkm) are populated too.
func SyncCenterline(ctx context.Context, pool *pgxpool.Pool, slug string, source CenterlineSource, dryRun bool) error {
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

	switch source {
	case "", CenterlineOSM:
		return syncCenterlineOSM(ctx, pool, slug, putInLon, putInLat, takeOutLon, takeOutLat, dryRun)
	case CenterlineNLDI:
		return syncCenterlineNLDI(ctx, pool, slug, putInLon, putInLat, takeOutLon, takeOutLat, dryRun)
	default:
		return fmt.Errorf("unknown centerline source %q", source)
	}
}

// SyncCenterlineAt is like SyncCenterline but uses the supplied put-in/take-out
// coordinates instead of reading them from reach_access. The reach's access
// points are left unchanged — only centerline geometry and ComID fields update.
func SyncCenterlineAt(ctx context.Context, pool *pgxpool.Pool, slug string, source CenterlineSource,
	putInLon, putInLat, takeOutLon, takeOutLat float64, dryRun bool) error {
	switch source {
	case "", CenterlineOSM:
		return syncCenterlineOSM(ctx, pool, slug, putInLon, putInLat, takeOutLon, takeOutLat, dryRun)
	case CenterlineNLDI:
		return syncCenterlineNLDI(ctx, pool, slug, putInLon, putInLat, takeOutLon, takeOutLat, dryRun)
	default:
		return fmt.Errorf("unknown centerline source %q", source)
	}
}

func syncCenterlineOSM(ctx context.Context, pool *pgxpool.Pool, slug string, putInLon, putInLat, takeOutLon, takeOutLat float64, dryRun bool) error {
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
		UPDATE reaches
		SET    centerline = (
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
		),
		       centerline_source = 'osm',
		       length_mi = ROUND((
		           ST_Length((
		               SELECT ST_LineSubstring(
		                   ST_GeomFromGeoJSON($2),
		                   ST_LineLocatePoint(ST_GeomFromGeoJSON($2), ST_ClosestPoint(ST_GeomFromGeoJSON($2), ST_SetSRID(ST_MakePoint($3,$4),4326))),
		                   ST_LineLocatePoint(ST_GeomFromGeoJSON($2), ST_ClosestPoint(ST_GeomFromGeoJSON($2), ST_SetSRID(ST_MakePoint($5,$6),4326)))
		               )::geography
		           )) / 1609.344
		       )::numeric, 2)
		WHERE slug = $1
	`, slug, geojson, putInLon, putInLat, takeOutLon, takeOutLat)
	return err
}

func syncCenterlineNLDI(ctx context.Context, pool *pgxpool.Pool, slug string, putInLon, putInLat, takeOutLon, takeOutLat float64, dryRun bool) error {
	line, err := fetchNLDIRiverLine(ctx, putInLon, putInLat, takeOutLon, takeOutLat)
	if err != nil {
		return fmt.Errorf("nldi fetch: %w", err)
	}
	if dryRun {
		return nil
	}

	_, err = pool.Exec(ctx, `
		UPDATE reaches
		SET    centerline = (
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
		),
		       centerline_source = 'nldi',
		       start_comid       = $7,
		       end_comid         = $8,
		       length_mi = ROUND((
		           ST_Length((
		               SELECT ST_LineSubstring(
		                   ST_GeomFromGeoJSON($2),
		                   ST_LineLocatePoint(ST_GeomFromGeoJSON($2), ST_ClosestPoint(ST_GeomFromGeoJSON($2), ST_SetSRID(ST_MakePoint($3,$4),4326))),
		                   ST_LineLocatePoint(ST_GeomFromGeoJSON($2), ST_ClosestPoint(ST_GeomFromGeoJSON($2), ST_SetSRID(ST_MakePoint($5,$6),4326)))
		               )::geography
		           )) / 1609.344
		       )::numeric, 2)
		WHERE slug = $1
	`, slug, line.GeoJSON, putInLon, putInLat, takeOutLon, takeOutLat,
		line.PutInComID, line.TakeOutComID)
	return err
}

// SyncCenterlineNLDIByComID is like syncCenterlineNLDI but the upstream and
// downstream ComIDs are supplied directly instead of being snapped from
// coordinates. The start/end coordinates are still used for trimming the
// merged mainstem to the exact reach extent. Used by the admin UI when the
// user picks ComIDs by clicking flowline segments.
func SyncCenterlineNLDIByComID(ctx context.Context, pool *pgxpool.Pool, slug string,
	upComID, downComID string,
	putInLon, putInLat, takeOutLon, takeOutLat float64,
	dryRun bool,
) error {
	line, err := fetchNLDIRiverLineByComID(ctx, upComID, downComID)
	if err != nil {
		return fmt.Errorf("nldi fetch: %w", err)
	}
	if dryRun {
		return nil
	}
	_, err = pool.Exec(ctx, `
		UPDATE reaches
		SET    centerline = (
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
		),
		       centerline_source = 'nldi',
		       start_comid       = $7,
		       end_comid         = $8,
		       length_mi = ROUND((
		           ST_Length((
		               SELECT ST_LineSubstring(
		                   ST_GeomFromGeoJSON($2),
		                   ST_LineLocatePoint(ST_GeomFromGeoJSON($2), ST_ClosestPoint(ST_GeomFromGeoJSON($2), ST_SetSRID(ST_MakePoint($3,$4),4326))),
		                   ST_LineLocatePoint(ST_GeomFromGeoJSON($2), ST_ClosestPoint(ST_GeomFromGeoJSON($2), ST_SetSRID(ST_MakePoint($5,$6),4326)))
		               )::geography
		           )) / 1609.344
		       )::numeric, 2)
		WHERE slug = $1
	`, slug, line.GeoJSON, putInLon, putInLat, takeOutLon, takeOutLat,
		line.PutInComID, line.TakeOutComID)
	return err
}

func sq(x float64) float64 { return x * x }
