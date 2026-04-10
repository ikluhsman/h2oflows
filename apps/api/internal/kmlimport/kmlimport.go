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
//	"Shuttle: <name>"  → reach_access type=shuttle_drop
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
	Shuttle   int      `json:"shuttle"`
	Campsites int      `json:"campsites"`
	Errors    []string `json:"errors,omitempty"`
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
	pool    *pgxpool.Pool
	DryRun  bool
	reaches []reachInfo   // cached for category-map mode
	cleared map[string]bool // reaches whose import data has been cleared this run
}

// New creates a new Importer.
func New(pool *pgxpool.Pool, dryRun bool) *Importer {
	return &Importer{pool: pool, DryRun: dryRun, cleared: map[string]bool{}}
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
		type reachFlowRange struct {
			reachID   string
			reachName string
			label     string
			minCFS    *float64
			maxCFS    *float64
		}
		var flowRanges []reachFlowRange

		for _, folder := range doc.Folders {
			rid, rslug, rname, created, err := imp.matchOrCreateReach(ctx, folder.Name, doc.Name)
			if err != nil {
				res.Log = append(res.Log, fmt.Sprintf("⚠  folder %q — %v", folder.Name, err))
				continue
			}
			if created {
				res.Log = append(res.Log, fmt.Sprintf("+ created reach %q (slug: %s)", folder.Name, rslug))
			}
			for _, pm := range folder.Placemarks {
				// Flow range metadata — no coordinates, special keyword name.
				if pm.Point == nil {
					if label, minCFS, maxCFS, ok := parseFlowRangePM(pm.Name, pm.Description); ok {
						flowRanges = append(flowRanges, reachFlowRange{rid, rname, label, minCFS, maxCFS})
						res.Log = append(res.Log, fmt.Sprintf("~ [%s] flow range %s", rname, label))
					}
					continue
				}
				pins = append(pins, assignment{rid, rslug, rname, pm, ""})
			}
		}

		// Upsert flow ranges after reach matching so reach IDs are known.
		for _, fr := range flowRanges {
			if err := imp.upsertFlowRange(ctx, fr.reachID, fr.label, fr.minCFS, fr.maxCFS); err != nil {
				res.Log = append(res.Log, fmt.Sprintf("⚠  [%s] flow range %s: %v", fr.reachName, fr.label, err))
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
		case "shuttle":
			if err := imp.upsertAccess(ctx, a.reachID, "shuttle_drop", pinName, desc, lon, lat); err != nil {
				st.Errors = append(st.Errors, fmt.Sprintf("shuttle %q: %v", pinName, err))
				res.Log = append(res.Log, fmt.Sprintf("✗ [%s] shuttle %q: %v", a.reachName, pinName, err))
			} else {
				st.Shuttle++
				res.Log = append(res.Log, fmt.Sprintf("✓ [%s] shuttle: %s", a.reachName, pinName))
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

// ── Reach matching / creation ─────────────────────────────────────────────────

// folderMeta holds the parsed components of a KML folder name.
//
// Format: "Display Name (CommonName,classMin,classMax)"
// Example: "Buffalo Creek to South Platte Hotel (Foxton,3,4)"
type folderMeta struct {
	baseName   string  // "Buffalo Creek to South Platte Hotel"
	commonName string  // "Foxton" (empty if not present)
	classMin   float64 // 3.0 (0 if not present)
	classMax   float64 // 4.0 (0 if not present)
}

// parseFolderMeta extracts reach metadata from a KML folder name.
// The trailing parenthetical is optional — if absent, baseName == folderName.
func parseFolderMeta(folderName string) folderMeta {
	m := folderMeta{}
	// Find last '(' … ')' pair.
	open := strings.LastIndex(folderName, "(")
	close := strings.LastIndex(folderName, ")")
	if open < 0 || close <= open {
		m.baseName = strings.TrimSpace(folderName)
		return m
	}
	m.baseName = strings.TrimSpace(folderName[:open])
	inner := strings.TrimSpace(folderName[open+1 : close])
	parts := strings.SplitN(inner, ",", 3)
	if len(parts) >= 1 {
		m.commonName = strings.TrimSpace(parts[0])
	}
	if len(parts) >= 2 {
		if v, err := strconv.ParseFloat(strings.TrimSpace(parts[1]), 64); err == nil {
			m.classMin = v
		}
	}
	if len(parts) >= 3 {
		if v, err := strconv.ParseFloat(strings.TrimSpace(parts[2]), 64); err == nil {
			m.classMax = v
		}
	}
	return m
}

// matchOrCreateReach finds an existing reach by folder name or creates a new
// stub reach so the KML can be imported without pre-seeding in Go code.
//
// folderName format: "Display Name (CommonName,classMin,classMax)"
// riverName: the KML document name, used as river_name on new reaches.
//
// Slug on creation: slugify(riverName) + "-" + slugify(commonName or baseName)
//
// The returned created flag is true when a new row was inserted.
func (imp *Importer) matchOrCreateReach(ctx context.Context, folderName, riverName string) (id, slug, name string, created bool, err error) {
	meta := parseFolderMeta(folderName)

	// Build candidate search terms: common name, base name, full folder name.
	candidates := []string{folderName}
	if meta.commonName != "" {
		candidates = append([]string{meta.commonName, meta.baseName}, candidates...)
	} else {
		candidates = append([]string{meta.baseName}, candidates...)
	}

	// Try to match existing reach by any candidate.
	for _, term := range candidates {
		matchErr := imp.pool.QueryRow(ctx, `
			SELECT id, slug, name FROM reaches
			WHERE  LOWER(name)        = LOWER($1)
			    OR LOWER(slug)        = LOWER($1)
			    OR LOWER(common_name) = LOWER($1)
			ORDER BY
				CASE WHEN LOWER(name)        = LOWER($1) THEN 0
				     WHEN LOWER(slug)        = LOWER($1) THEN 1
				     WHEN LOWER(common_name) = LOWER($1) THEN 2
				     ELSE 3 END
			LIMIT 1
		`, term).Scan(&id, &slug, &name)
		if matchErr == nil {
			return id, slug, name, false, nil
		}
	}

	// No match — derive slug from riverName + commonName (or baseName).
	identifier := meta.baseName
	if meta.commonName != "" {
		identifier = meta.commonName
	}
	newSlug := slugify(riverName) + "-" + slugify(identifier)

	// Build INSERT with all available metadata.
	displayName := meta.baseName
	if displayName == "" {
		displayName = folderName
	}
	commonNameVal := meta.commonName
	if commonNameVal == "" {
		commonNameVal = displayName
	}

	var classMinArg, classMaxArg interface{}
	if meta.classMin > 0 {
		classMinArg = meta.classMin
	}
	if meta.classMax > 0 {
		classMaxArg = meta.classMax
	}

	err = imp.pool.QueryRow(ctx, `
		INSERT INTO reaches (slug, name, common_name, river_name, class_min, class_max)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (slug) DO UPDATE
			SET common_name = EXCLUDED.common_name,
			    river_name  = EXCLUDED.river_name,
			    class_min   = COALESCE(reaches.class_min, EXCLUDED.class_min),
			    class_max   = COALESCE(reaches.class_max, EXCLUDED.class_max)
		RETURNING id, slug, name
	`, newSlug, displayName, commonNameVal, riverName, classMinArg, classMaxArg).Scan(&id, &slug, &name)
	if err != nil {
		return "", "", "", false, fmt.Errorf("create reach %q: %w", folderName, err)
	}
	return id, slug, name, true, nil
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
	derivedName := *putInName + " to " + *takeOutName
	_, err = imp.pool.Exec(ctx, `
		UPDATE reaches
		SET name        = $2,
		    put_in_name  = $3,
		    take_out_name = $4
		WHERE id = $1
	`, reachID, derivedName, *putInName, *takeOutName)
	return err
}

// slugify converts a display name to a URL-safe slug.
// "Browns Canyon" → "browns-canyon", "Cache La Poudre" → "cache-la-poudre"
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

// ── Flow range parsing ────────────────────────────────────────────────────────

// flowRangeKeywords maps KML placemark names to flow_ranges label values.
var flowRangeKeywords = map[string]string{
	"below": "below_recommended",
	"low":   "low_runnable",
	"med":   "runnable",
	"high":  "high_runnable",
	"above": "above_recommended",
}

// parseFlowRangePM detects a flow-range metadata placemark and returns the
// DB label, min/max CFS, and true when the placemark name is a known keyword.
//
// Description format:
//   - "below" / "above": single CFS value — max_cfs or min_cfs respectively
//   - "low" / "med" / "high": "min,max" pair (or single value treated as min)
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
	case "below_recommended":
		// single value = upper bound (< this)
		maxCFS = parseVal(parts[0])
	case "above_recommended":
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

// ── DB upserts ────────────────────────────────────────────────────────────────

func (imp *Importer) upsertRapidLocation(ctx context.Context, reachID, name, desc string, isSurfWave, isPermanentHazard bool, hazardType string, lon, lat float64) error {
	if imp.DryRun {
		return nil
	}
	classRating := ParseClassRating(name, desc)
	tag, err := imp.pool.Exec(ctx, `
		UPDATE rapids
		SET location             = ST_SetSRID(ST_MakePoint($3, $4), 4326)::geography,
		    description          = CASE WHEN $5 <> '' THEN $5 ELSE description END,
		    class_rating         = CASE WHEN $6::numeric IS NOT NULL THEN $6::numeric ELSE class_rating END,
		    is_surf_wave         = is_surf_wave OR $7,
		    is_permanent_hazard  = is_permanent_hazard OR $8,
		    hazard_type          = CASE WHEN $9 <> '' THEN $9 ELSE hazard_type END
		WHERE reach_id = $1 AND LOWER(name) = LOWER($2)
	`, reachID, name, lon, lat, desc, classRating, isSurfWave, isPermanentHazard, hazardType)
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
		`, reachID, name, lon, lat, desc, classRating, isSurfWave, isPermanentHazard, hazardType)
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
	for _, p := range []string{"Rapid", "Wave", "Surf", "Put-in", "Take-out", "Parking", "Shuttle", "Hazard", "Campsite"} {
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

func sq(x float64) float64 { return x * x }
