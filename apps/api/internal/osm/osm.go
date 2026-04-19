package osm

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"net/url"
	"strings"
	"time"
)

var httpClient = &http.Client{Timeout: 15 * time.Second}

// Coord is a [longitude, latitude] pair (exported for callers that pass
// intermediate waypoints to FetchReachLine).
type Coord [2]float64

type coord = Coord

// ── Centerline algorithm overview ─────────────────────────────────────────────
//
// FetchReachLine fetches all river/stream waterways in the reach bbox from
// Overpass, selects the correct waterway, stitches its ways into one line, and
// clips it to the put-in → take-out span. The following cases are handled:
//
//   Main-stem reach
//     A single named river spans the bbox. filterByBestEndpointScore picks it
//     because both access points are close to it.
//
//   Tributary / small creek
//     A small creek (waterway=stream) enters a larger river near the take-out.
//     The creek's combined endpoint score is lower than the main river's because
//     both put-in AND take-out are on the creek, even though the take-out is
//     near the confluence. Old behaviour (filterByType→river only) would drop
//     the creek entirely; the new approach considers streams alongside rivers.
//
//   Preferred-name hint (river_name from DB)
//     If a non-empty preferredName is passed, any ways whose OSM name contains
//     that string (case-insensitive) are promoted: their combined score is halved
//     so they win ties against equally-close unnamed or differently-named ways.
//
//   Multi-channel braid
//     Multiple ways share the same name. filterByBestEndpointScore picks the
//     winning name; stitchWays and chainWays then join all its segments.
//
//   Canyon / wilderness gap
//     OSM coverage may not be complete inside a canyon. stitchWays joins
//     exactly-connected segments; chainWays then bridges any remaining gap
//     (up to ~5 km) by greedy nearest-endpoint chaining.
//
//   Confluence take-out
//     Take-out is near the mouth of a tributary. The creek still wins on
//     combined endpoint scoring because the put-in is far from the main river.
//
// Returns ("", nil) if no waterways are found.
func FetchReachLine(ctx context.Context, minLon, minLat, maxLon, maxLat, startLng, startLat, endLng, endLat float64, preferredName string, intermediatePoints []coord) (string, error) {
	const pad = 0.01 // small extra padding around the access-point bbox

	// Always expand the bbox to include the explicit put-in and take-out so
	// that downstream OSM ways aren't missed when access points don't reach
	// the full extent of the run (e.g. a take-out 10 km outside the access hull).
	bboxMinLon := math.Min(minLon, math.Min(startLng, endLng)) - pad
	bboxMinLat := math.Min(minLat, math.Min(startLat, endLat)) - pad
	bboxMaxLon := math.Max(maxLon, math.Max(startLng, endLng)) + pad
	bboxMaxLat := math.Max(maxLat, math.Max(startLat, endLat)) + pad

	tagged, err := fetchTaggedWays(ctx, bboxMinLon, bboxMinLat, bboxMaxLon, bboxMaxLat)
	if err != nil {
		return "", err
	}
	if len(tagged) == 0 {
		return "", nil
	}

	// Select the target waterway. Multiple named rivers/streams may exist in the
	// bbox (e.g. Chalk Creek alongside the Arkansas, or a small creek joining the
	// Colorado). Score each named waterway by combined closest-approach distance
	// from both put-in and take-out — this correctly handles tributaries where the
	// creek (stream-type) would be dropped by a river-only filter.
	// A preferredName hint (from the DB river_name) halves a matching waterway's
	// score so it wins ties against equally-close unnamed ways.
	ways := extractCoords(tagged)
	if named := filterByBestEndpointScore(tagged, startLng, startLat, endLng, endLat, preferredName, intermediatePoints); len(named) > 0 {
		ways = extractCoords(named)
	}

	// Stitch all same-name ways into one continuous line, then clip to the
	// reach. OSM often splits a single river into multiple long ways — greedy
	// chaining fails when each way is longer than the reach and neither
	// endpoint is near the put-in. Instead, join them at shared junction
	// nodes and let extractSubChain clip to [put-in, take-out].
	full := stitchWays(ways)

	// If stitchWays produces a chain whose end is far from the take-out,
	// fall back to chainWays which tolerates gaps in OSM coverage. This
	// handles canyons and wilderness segments where ways aren't perfectly
	// connected at junctions (e.g. Ruby Horsethief, GJ to Loma).
	// Threshold: ~2 km in degrees (rough, avoids sqrt).
	const shortChainThresh2 = 0.018 * 0.018 // ≈ 2 km squared
	if len(full) >= 2 {
		tail := full[len(full)-1]
		if dist2(tail, endLng, endLat) > shortChainThresh2 {
			if fallback := chainWays(ways, startLng, startLat, endLng, endLat); len(fallback) > len(full) {
				full = fallback
			}
		}
	} else {
		// stitchWays returned nothing — try chainWays directly.
		full = chainWays(ways, startLng, startLat, endLng, endLat)
	}

	if len(full) < 2 {
		return "", nil
	}
	chain := extractSubChain(full, startLng, startLat, endLng, endLat)
	if len(chain) < 2 {
		return "", nil
	}

	// If the chain's first point is more than ~200m from the put-in, the OSM
	// data doesn't reach the put-in. Prepend the put-in coordinate so the
	// line connects cleanly to the access point pin without a visible gap.
	const putInGapThresh2 = 0.002 * 0.002 // ≈ 200m squared
	if dist2(chain[0], startLng, startLat) > putInGapThresh2 {
		chain = append([]coord{{startLng, startLat}}, chain...)
	}
	// Same for take-out.
	if dist2(chain[len(chain)-1], endLng, endLat) > putInGapThresh2 {
		chain = append(chain, coord{endLng, endLat})
	}

	return buildLineString(chain), nil
}

func extractCoords(tagged []taggedWay) [][]coord {
	out := make([][]coord, len(tagged))
	for i, t := range tagged {
		out[i] = t.coords
	}
	return out
}

func extractCoordsByType(tagged []taggedWay, wtype string) [][]coord {
	var out [][]coord
	for _, t := range tagged {
		if t.wtype == wtype {
			out = append(out, t.coords)
		}
	}
	return out
}

// filterByType returns tagged ways matching the given waterway type.
func filterByType(tagged []taggedWay, wtype string) []taggedWay {
	var out []taggedWay
	for _, t := range tagged {
		if t.wtype == wtype {
			out = append(out, t)
		}
	}
	return out
}

// filterByBestEndpointScore returns all ways sharing the name whose waterways
// pass closest to BOTH put-in and take-out combined. This handles tributary
// reaches where a small creek (waterway=stream) joins a larger river near the
// take-out: the creek wins because both endpoints are close to it, even if the
// main river happens to be closer to one endpoint in isolation.
//
// preferredName (from DB river_name) halves the combined score of any waterway
// whose OSM name contains the hint (case-insensitive), so it wins ties.
// If no named ways exist, returns nil (caller falls back to all ways).
func filterByBestEndpointScore(tagged []taggedWay, putLng, putLat, takeLng, takeLat float64, preferredName string, intermediatePoints []coord) []taggedWay {
	type nameScore struct {
		putDist  float64
		takeDist float64
		midDist  float64 // sum of min distances to each intermediate point
	}
	scores := make(map[string]*nameScore)
	for _, t := range tagged {
		if t.name == "" {
			continue
		}
		s, ok := scores[t.name]
		if !ok {
			s = &nameScore{putDist: math.MaxFloat64, takeDist: math.MaxFloat64}
			scores[t.name] = s
		}
		for i := 0; i < len(t.coords)-1; i++ {
			if _, d := closestPointOnSegment(putLng, putLat, t.coords[i], t.coords[i+1]); d < s.putDist {
				s.putDist = d
			}
			if _, d := closestPointOnSegment(takeLng, takeLat, t.coords[i], t.coords[i+1]); d < s.takeDist {
				s.takeDist = d
			}
		}
	}
	if len(scores) == 0 {
		return nil
	}

	// Score each intermediate point (rapids, mid-reach access) against each
	// named waterway. For each point, find the min distance across all ways
	// sharing that name, then sum across all intermediate points.
	if len(intermediatePoints) > 0 {
		for name, s := range scores {
			var total float64
			for _, pt := range intermediatePoints {
				minD := math.MaxFloat64
				for _, t := range tagged {
					if t.name != name {
						continue
					}
					for i := 0; i < len(t.coords)-1; i++ {
						if _, d := closestPointOnSegment(pt[0], pt[1], t.coords[i], t.coords[i+1]); d < minD {
							minD = d
						}
					}
				}
				total += minD
			}
			s.midDist = total
		}
	}

	hint := strings.ToLower(preferredName)
	bestName := ""
	bestScore := math.MaxFloat64
	for name, s := range scores {
		combined := s.putDist + s.takeDist + s.midDist
		if hint != "" && strings.Contains(strings.ToLower(name), hint) {
			combined *= 0.5
		}
		if combined < bestScore {
			bestScore = combined
			bestName = name
		}
	}
	if bestName == "" {
		return nil
	}
	var out []taggedWay
	for _, t := range tagged {
		if t.name == bestName {
			out = append(out, t)
		}
	}
	return out
}

// filterByNearestName finds the named river that passes closest to (lng, lat)
// by checking every segment of every way, then returns all ways with that name.
// This handles long OSM ways whose endpoints are far from the access point but
// whose path passes directly through it (e.g. a single Arkansas River way
// spanning 50+ miles).
func filterByNearestName(tagged []taggedWay, lng, lat float64) []taggedWay {
	bestName := ""
	bestDist := math.MaxFloat64
	for _, t := range tagged {
		if t.name == "" {
			continue
		}
		for i := 0; i < len(t.coords)-1; i++ {
			_, d := closestPointOnSegment(lng, lat, t.coords[i], t.coords[i+1])
			if d < bestDist {
				bestDist = d
				bestName = t.name
			}
		}
	}
	if bestName == "" {
		return nil
	}
	var out []taggedWay
	for _, t := range tagged {
		if t.name == bestName {
			out = append(out, t)
		}
	}
	return out
}

// FetchRiverLine queries Overpass for the longest river/stream waterway within
// the given bounding box and returns it as a GeoJSON LineString JSON string.
// Returns ("", nil) if no waterways are found.
func FetchRiverLine(ctx context.Context, minLon, minLat, maxLon, maxLat float64) (string, error) {
	ways, err := fetchWays(ctx, minLon, minLat, maxLon, maxLat)
	if err != nil {
		return "", err
	}
	if len(ways) == 0 {
		return "", nil
	}

	// Pick the longest single way as a simple fallback.
	best := ways[0]
	for _, w := range ways[1:] {
		if len(w) > len(best) {
			best = w
		}
	}
	return buildLineString(best), nil
}

// fetchWays queries Overpass and returns each waterway way as a slice of coords,
// tagged with its waterway type ("river" or "stream").
type taggedWay struct {
	coords []coord
	wtype  string // "river" | "stream"
	name   string // OSM name tag (empty if unnamed)
}

func fetchTaggedWays(ctx context.Context, minLon, minLat, maxLon, maxLat float64) ([]taggedWay, error) {
	query := fmt.Sprintf(
		`[out:json];way["waterway"~"^(river|stream)$"](%.6f,%.6f,%.6f,%.6f);out geom tags;`,
		minLat, minLon, maxLat, maxLon,
	)
	data, err := OverpassQuery(ctx, query)
	if err != nil {
		return nil, err
	}

	type osmNode struct {
		Lat float64 `json:"lat"`
		Lon float64 `json:"lon"`
	}
	var parsed struct {
		Elements []struct {
			Geometry []osmNode         `json:"geometry"`
			Tags     map[string]string `json:"tags"`
		} `json:"elements"`
	}
	if err := json.Unmarshal(data, &parsed); err != nil {
		return nil, fmt.Errorf("parse osm response: %w", err)
	}

	ways := make([]taggedWay, 0, len(parsed.Elements))
	for _, el := range parsed.Elements {
		if len(el.Geometry) < 2 {
			continue
		}
		w := make([]coord, len(el.Geometry))
		for i, n := range el.Geometry {
			w[i] = coord{n.Lon, n.Lat}
		}
		ways = append(ways, taggedWay{coords: w, wtype: el.Tags["waterway"], name: el.Tags["name"]})
	}
	return ways, nil
}

// fetchWays returns just the coord slices, for callers that don't need tags.
func fetchWays(ctx context.Context, minLon, minLat, maxLon, maxLat float64) ([][]coord, error) {
	tagged, err := fetchTaggedWays(ctx, minLon, minLat, maxLon, maxLat)
	if err != nil {
		return nil, err
	}
	ways := make([][]coord, len(tagged))
	for i, t := range tagged {
		ways[i] = t.coords
	}
	return ways, nil
}

// chainWays assembles disconnected OSM ways into a single connected path
// starting from the end nearest to (startLng, startLat) and heading toward
// (endLng, endLat). At each step, candidate ways are only accepted if they
// move the chain tip closer to the end anchor — this prevents the greedy
// algorithm from appending a large reversed way that shoots the chain back
// upstream when OSM has a single long way spanning the whole river.
func chainWays(ways [][]coord, startLng, startLat, endLng, endLat float64) []coord {
	if len(ways) == 0 {
		return nil
	}

	remaining := make([][]coord, len(ways))
	copy(remaining, ways)

	// Find the way that passes closest to the start point, oriented so
	// its tail end heads toward the end anchor. This prevents picking a
	// way by endpoint proximity then reversing it to head upstream.
	bestIdx, bestDist, bestReverse := 0, math.MaxFloat64, false
	for i, w := range remaining {
		head, tail := w[0], w[len(w)-1]
		headToEnd := dist2(head, endLng, endLat)
		tailToEnd := dist2(tail, endLng, endLat)

		// Non-reversed: attach at head, chain continues from tail toward end.
		if d := dist2(head, startLng, startLat); d < bestDist && tailToEnd < headToEnd {
			bestDist, bestIdx, bestReverse = d, i, false
		}
		// Reversed: attach at tail, chain continues from head toward end.
		if d := dist2(tail, startLng, startLat); d < bestDist && headToEnd < tailToEnd {
			bestDist, bestIdx, bestReverse = d, i, true
		}
	}

	w := remaining[bestIdx]
	if bestReverse {
		w = reverseWay(w)
	}
	result := append([]coord{}, w...)
	remaining = append(remaining[:bestIdx], remaining[bestIdx+1:]...)

	// Greedily chain remaining ways by nearest endpoint.
	// maxGap² ≈ (0.05°)² — roughly 5 km; handles gaps in OSM coverage through
	// canyons and wilderness areas where ways aren't perfectly connected.
	const maxGap2 = 0.05 * 0.05
	distToEnd := func(c coord) float64 { return dist2(c, endLng, endLat) }
	for len(remaining) > 0 {
		tip := result[len(result)-1]
		tipDistToEnd := distToEnd(tip)
		bestIdx, bestDist, bestReverse = -1, maxGap2, false

		for i, w := range remaining {
			// Only accept this way if it passes closer to the destination than
			// the current tip — prevents appending a reversed upstream way.
			// Check the nearest point on the way to the end anchor, not just endpoints,
			// since a way may share a junction node with the tip but still head toward the goal.
			wayMinDistToEnd := math.MaxFloat64
			for j := 0; j < len(w)-1; j++ {
				_, d := closestPointOnSegment(endLng, endLat, w[j], w[j+1])
				if d < wayMinDistToEnd {
					wayMinDistToEnd = d
				}
			}
			if wayMinDistToEnd >= tipDistToEnd {
				continue
			}
			head, tail := w[0], w[len(w)-1]
			if d := dist2(head, tip[0], tip[1]); d < bestDist {
				bestDist, bestIdx, bestReverse = d, i, false
			}
			if d := dist2(tail, tip[0], tip[1]); d < bestDist {
				bestDist, bestIdx, bestReverse = d, i, true
			}
		}

		if bestIdx == -1 {
			break // no more connected ways moving toward the end anchor
		}

		w := remaining[bestIdx]
		if bestReverse {
			w = reverseWay(w)
		}
		result = append(result, w[1:]...) // skip first node — duplicate of tip
		remaining = append(remaining[:bestIdx], remaining[bestIdx+1:]...)
	}

	return result
}

// stitchWays joins ways that share junction nodes into a single continuous line.
// Unlike chainWays (greedy nearest-endpoint), this looks for exact shared
// endpoints to concatenate ways, then returns the longest resulting chain.
// Works well when OSM splits a single named river into a few long segments.
func stitchWays(ways [][]coord) []coord {
	if len(ways) == 0 {
		return nil
	}
	if len(ways) == 1 {
		return ways[0]
	}

	// Index way endpoints for fast junction lookup.
	// key = "lng,lat" of an endpoint, value = list of (wayIndex, isEnd)
	type endpoint struct {
		idx   int
		isEnd bool // true = last node, false = first node
	}
	key := func(c coord) [2]float64 { return [2]float64{c[0], c[1]} }

	// Try building a chain from each way and keep the longest.
	var best []coord
	for start := range ways {
		chain := append([]coord{}, ways[start]...)
		tried := make([]bool, len(ways))
		tried[start] = true

		// Extend forward from tail.
		changed := true
		for changed {
			changed = false
			tip := key(chain[len(chain)-1])
			for i, w := range ways {
				if tried[i] {
					continue
				}
				if key(w[0]) == tip {
					chain = append(chain, w[1:]...)
					tried[i] = true
					changed = true
					break
				}
				if key(w[len(w)-1]) == tip {
					chain = append(chain, reverseWay(w)[1:]...)
					tried[i] = true
					changed = true
					break
				}
			}
		}

		// Extend backward from head.
		changed = true
		for changed {
			changed = false
			head := key(chain[0])
			for i, w := range ways {
				if tried[i] {
					continue
				}
				if key(w[len(w)-1]) == head {
					chain = append(w[:len(w)-1], chain...)
					tried[i] = true
					changed = true
					break
				}
				if key(w[0]) == head {
					rev := reverseWay(w)
					chain = append(rev[:len(rev)-1], chain...)
					tried[i] = true
					changed = true
					break
				}
			}
		}

		if len(chain) > len(best) {
			best = chain
		}
	}
	return best
}

// extractSubChain returns the portion of chain between the two access points,
// with endpoints snapped perpendicularly to the nearest river segment.
// Handles reversed chains: if the end anchor is upstream of the start anchor
// in the chain ordering, the chain is reversed before extraction.
func extractSubChain(chain []coord, startLng, startLat, endLng, endLat float64) []coord {
	if len(chain) < 2 {
		return chain
	}
	startSeg, startPt := nearestSegment(chain, startLng, startLat)
	endSeg, endPt := nearestSegment(chain, endLng, endLat)

	// If the end point is upstream in the chain, flip it.
	if endSeg < startSeg {
		chain = reverseWay(chain)
		startSeg, startPt = nearestSegment(chain, startLng, startLat)
		endSeg, endPt = nearestSegment(chain, endLng, endLat)
	}

	// Build: snapped start → interior nodes → snapped end.
	result := []coord{startPt}
	for i := startSeg + 1; i <= endSeg; i++ {
		result = append(result, chain[i])
	}
	result = append(result, endPt)
	return result
}

// nearestSegment returns the index of the segment in chain closest to (lng, lat)
// and the projected foot point on that segment.
func nearestSegment(chain []coord, lng, lat float64) (int, coord) {
	bestSeg, bestDist := 0, math.MaxFloat64
	var bestPt coord
	for i := 0; i < len(chain)-1; i++ {
		pt, d := closestPointOnSegment(lng, lat, chain[i], chain[i+1])
		if d < bestDist {
			bestDist, bestSeg, bestPt = d, i, pt
		}
	}
	return bestSeg, bestPt
}

// closestPointOnSegment returns the point on segment [a,b] closest to (lng,lat)
// and the squared distance to it.
func closestPointOnSegment(lng, lat float64, a, b coord) (coord, float64) {
	dx, dy := b[0]-a[0], b[1]-a[1]
	if dx == 0 && dy == 0 {
		return a, dist2(a, lng, lat)
	}
	t := ((lng-a[0])*dx + (lat-a[1])*dy) / (dx*dx + dy*dy)
	if t < 0 {
		t = 0
	} else if t > 1 {
		t = 1
	}
	p := coord{a[0] + t*dx, a[1] + t*dy}
	return p, dist2(p, lng, lat)
}

func dist2(c coord, lng, lat float64) float64 {
	dl := c[0] - lng
	dp := c[1] - lat
	return dl*dl + dp*dp
}

func reverseWay(w []coord) []coord {
	r := make([]coord, len(w))
	for i, c := range w {
		r[len(w)-1-i] = c
	}
	return r
}

func buildLineString(coords []coord) string {
	parts := make([]string, len(coords))
	for i, c := range coords {
		parts[i] = fmt.Sprintf("[%.7f,%.7f]", c[0], c[1])
	}
	return fmt.Sprintf(`{"type":"LineString","coordinates":[%s]}`, strings.Join(parts, ","))
}

// overpassEndpoints is tried in order until one succeeds.
// lz4/z are load-balanced slaves of overpass-api.de; osm.ch is an independent
// Swiss instance — both tend to be reachable from cloud-hosted servers.
var overpassEndpoints = []string{
	"https://lz4.overpass-api.de/api/interpreter",
	"https://z.overpass-api.de/api/interpreter",
	"https://overpass.osm.ch/api/interpreter",
}

// OverpassQuery POSTs a query to the Overpass API and returns the raw response body.
// Falls back through all overpassEndpoints before giving up — a 406 or 503 from
// one endpoint is endpoint-specific and doesn't mean others will fail too.
func OverpassQuery(ctx context.Context, query string) ([]byte, error) {
	form := url.Values{"data": {query}}
	var lastErr error
	for _, endpoint := range overpassEndpoints {
		req, err := http.NewRequestWithContext(ctx, "POST", endpoint,
			strings.NewReader(form.Encode()),
		)
		if err != nil {
			return nil, err
		}
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Set("User-Agent", "h2oflows/1.0 (https://h2oflows.org)")
		resp, err := httpClient.Do(req)
		if err != nil {
			log.Printf("overpass %s: %v", endpoint, err)
			lastErr = err
			continue
		}
		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			lastErr = err
			continue
		}
		if resp.StatusCode == http.StatusOK {
			return body, nil
		}
		snippet := string(body)
		if len(snippet) > 120 {
			snippet = snippet[:120]
		}
		log.Printf("overpass %s: status %d — %s", endpoint, resp.StatusCode, snippet)
		lastErr = fmt.Errorf("overpass returned %d", resp.StatusCode)
		// Always try the next endpoint — never bail early on a specific status code.
	}
	return nil, lastErr
}
