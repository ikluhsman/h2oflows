package kmlimport

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

var httpClient = &http.Client{Timeout: 30 * time.Second}

func fetchOSMRiverLine(ctx context.Context, minLon, minLat, maxLon, maxLat float64) (string, error) {
	query := fmt.Sprintf(
		`[out:json];way["waterway"~"^(river|stream)$"](%.6f,%.6f,%.6f,%.6f);out geom;`,
		minLat, minLon, maxLat, maxLon,
	)
	data, err := overpassQuery(ctx, query)
	if err != nil {
		return "", err
	}

	type osmNode struct {
		Lat float64 `json:"lat"`
		Lon float64 `json:"lon"`
	}
	type osmResp struct {
		Elements []struct {
			Geometry []osmNode         `json:"geometry"`
			Tags     map[string]string `json:"tags"`
		} `json:"elements"`
	}

	var parsed osmResp
	if err := json.Unmarshal(data, &parsed); err != nil {
		return "", fmt.Errorf("parse osm response: %w", err)
	}
	if len(parsed.Elements) == 0 {
		return "", nil
	}

	best := parsed.Elements[0]
	for _, el := range parsed.Elements[1:] {
		if len(el.Geometry) > len(best.Geometry) {
			best = el
		}
	}

	var coords []string
	for _, n := range best.Geometry {
		coords = append(coords, fmt.Sprintf("[%.7f,%.7f]", n.Lon, n.Lat))
	}
	return fmt.Sprintf(`{"type":"LineString","coordinates":[%s]}`, strings.Join(coords, ",")), nil
}

func overpassQuery(ctx context.Context, query string) ([]byte, error) {
	form := url.Values{"data": {query}}
	req, err := http.NewRequestWithContext(ctx, "POST",
		"https://overpass-api.de/api/interpreter",
		strings.NewReader(form.Encode()),
	)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("overpass returned %d", resp.StatusCode)
	}
	return io.ReadAll(resp.Body)
}
