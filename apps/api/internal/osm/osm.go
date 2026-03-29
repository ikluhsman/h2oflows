package osm

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"strings"
	"time"
)

var httpClient = &http.Client{Timeout: 30 * time.Second}

// coord is a [longitude, latitude] pair.
type coord [2]float64

// FetchReachLine fetches all river/stream ways within the given bounding box,
// then chains them into a single connected LineString starting from the end
// nearest (startLng, startLat) — typically the put-in centroid.
//
// The caller should pass a bbox that covers all access points for the reach
// so the full river section is captured. Returns ("", nil) if no waterways found.
func FetchReachLine(ctx context.Context, minLon, minLat, maxLon, maxLat, startLng, startLat, endLng, endLat float64) (string, error) {
	const pad = 0.01 // small extra padding around the access-point bbox
	tagged, err := fetchTaggedWays(ctx, minLon-pad, minLat-pad, maxLon+pad, maxLat+pad)
	if err != nil {
		return "", err
	}
	if len(tagged) == 0 {
		return "", nil
	}

	// Prefer named-river ways over streams to avoid chaining up tributaries.
	// If the bbox contains at least one "river" way, drop all "stream" ways.
	ways := extractCoords(tagged)
	if riverCoords := extractCoordsByType(tagged, "river"); len(riverCoords) > 0 {
		ways = riverCoords
	}

	chain := chainWays(ways, startLng, startLat)
	if len(chain) == 0 {
		return "", nil
	}
	// Snap the start to the put-in and the end to the take-out so the
	// rendered line is bounded by the actual access points.
	chain = trimChainStart(chain, startLng, startLat)
	chain = trimChainEnd(chain, endLng, endLat)
	if len(chain) < 2 {
		return "", nil
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
		ways = append(ways, taggedWay{coords: w, wtype: el.Tags["waterway"]})
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
// starting from the end nearest to (startLng, startLat).
//
// OSM ways for a river arrive in arbitrary order and may be digitized in
// either direction. This greedy nearest-endpoint algorithm reconnects them:
// at each step it picks the unvisited way whose nearest endpoint is closest
// to the current path tip, reversing the way if needed, and appends it.
// A gap threshold of ~1 km stops chaining if there's no plausible connection.
func chainWays(ways [][]coord, startLng, startLat float64) []coord {
	if len(ways) == 0 {
		return nil
	}

	remaining := make([][]coord, len(ways))
	copy(remaining, ways)

	// Find the way whose nearest endpoint is closest to the start point.
	bestIdx, bestDist, bestReverse := 0, math.MaxFloat64, false
	for i, w := range remaining {
		if d := dist2(w[0], startLng, startLat); d < bestDist {
			bestDist, bestIdx, bestReverse = d, i, false
		}
		if d := dist2(w[len(w)-1], startLng, startLat); d < bestDist {
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
	// maxGap² ≈ (0.01°)² — roughly 700 m; stops us bridging tributaries.
	const maxGap2 = 0.01 * 0.01
	for len(remaining) > 0 {
		tip := result[len(result)-1]
		bestIdx, bestDist, bestReverse = -1, maxGap2, false

		for i, w := range remaining {
			if d := dist2(w[0], tip[0], tip[1]); d < bestDist {
				bestDist, bestIdx, bestReverse = d, i, false
			}
			if d := dist2(w[len(w)-1], tip[0], tip[1]); d < bestDist {
				bestDist, bestIdx, bestReverse = d, i, true
			}
		}

		if bestIdx == -1 {
			break // no more connected ways within gap threshold
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

// trimChainStart drops leading nodes from chain until the node closest to
// (startLng, startLat) becomes the new head. This snaps the line start to
// the put-in rather than the upstream OSM way endpoint.
func trimChainStart(chain []coord, startLng, startLat float64) []coord {
	if len(chain) == 0 {
		return chain
	}
	bestIdx, bestDist := 0, dist2(chain[0], startLng, startLat)
	for i := 1; i < len(chain); i++ {
		if d := dist2(chain[i], startLng, startLat); d < bestDist {
			bestDist, bestIdx = d, i
		}
	}
	return chain[bestIdx:]
}

// trimChainEnd drops trailing nodes from chain after the node closest to
// (endLng, endLat). This snaps the line end to the take-out.
func trimChainEnd(chain []coord, endLng, endLat float64) []coord {
	if len(chain) == 0 {
		return chain
	}
	bestIdx, bestDist := len(chain)-1, dist2(chain[len(chain)-1], endLng, endLat)
	for i := len(chain) - 2; i >= 0; i-- {
		if d := dist2(chain[i], endLng, endLat); d < bestDist {
			bestDist, bestIdx = d, i
		}
	}
	return chain[:bestIdx+1]
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

// overpassEndpoints is tried in order; on 429/5xx the next is tried.
var overpassEndpoints = []string{
	"https://overpass-api.de/api/interpreter",
	"https://overpass.kumi.systems/api/interpreter",
	"https://maps.mail.ru/osm/tools/overpass/api/interpreter",
}

// OverpassQuery POSTs a query to the Overpass API and returns the raw response body.
// Falls back through overpassEndpoints on 429/5xx errors.
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
		resp, err := httpClient.Do(req)
		if err != nil {
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
		// Retry on server errors and rate limiting; bail on client errors.
		lastErr = fmt.Errorf("overpass returned %d", resp.StatusCode)
		if resp.StatusCode < 429 {
			return nil, lastErr
		}
	}
	return nil, lastErr
}
