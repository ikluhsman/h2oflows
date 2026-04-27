package nldi

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

// FirstCoord returns the first [lng, lat] pair from a GeoJSON geometry, or nil
// if the geometry is empty or an unrecognised type.
func FirstCoord(g Geometry) []float64 {
	switch g.Type {
	case "LineString":
		// coordinates: [[lng, lat], ...]
		if outer, ok := g.Coordinates.([]interface{}); ok && len(outer) > 0 {
			if pt, ok := outer[0].([]interface{}); ok && len(pt) >= 2 {
				lng, _ := pt[0].(float64)
				lat, _ := pt[1].(float64)
				return []float64{lng, lat}
			}
		}
	case "MultiLineString":
		// coordinates: [[[lng, lat], ...], ...]
		if lines, ok := g.Coordinates.([]interface{}); ok && len(lines) > 0 {
			if line, ok := lines[0].([]interface{}); ok && len(line) > 0 {
				if pt, ok := line[0].([]interface{}); ok && len(pt) >= 2 {
					lng, _ := pt[0].(float64)
					lat, _ := pt[1].(float64)
					return []float64{lng, lat}
				}
			}
		}
	case "Point":
		if pt, ok := g.Coordinates.([]interface{}); ok && len(pt) >= 2 {
			lng, _ := pt[0].(float64)
			lat, _ := pt[1].(float64)
			return []float64{lng, lat}
		}
	}
	return nil
}

const nhdArcGISURL = "https://hydro.nationalmap.gov/arcgis/rest/services/nhd/MapServer/6/query"

// NHDStreamNameAt queries the National Map NHD ArcGIS service for the GNIS
// stream name and ID at the given coordinate. Returns empty strings without an
// error when no named feature is found.
func NHDStreamNameAt(ctx context.Context, lat, lng float64) (name, gnisID string, err error) {
	params := url.Values{
		"geometry":     {fmt.Sprintf("%f,%f", lng, lat)},
		"geometryType": {"esriGeometryPoint"},
		"spatialRel":   {"esriSpatialRelIntersects"},
		"outFields":    {"gnis_name,gnis_id"},
		"distance":     {"100"},
		"units":        {"esriSRUnit_Meter"},
		"inSR":         {"4326"},
		"f":            {"json"},
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, nhdArcGISURL+"?"+params.Encode(), nil)
	if err != nil {
		return "", "", err
	}
	req.Header.Set("User-Agent", "h2oflows/1.0 (https://h2oflows.org)")

	resp, err := httpClient.Do(req)
	if err != nil {
		return "", "", fmt.Errorf("nhd arcgis: %w", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	var result struct {
		Features []struct {
			Attributes struct {
				GnisName string `json:"gnis_name"`
				GnisID   string `json:"gnis_id"`
			} `json:"attributes"`
		} `json:"features"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", "", fmt.Errorf("nhd arcgis parse: %w", err)
	}
	for _, f := range result.Features {
		if f.Attributes.GnisName != "" {
			return f.Attributes.GnisName, f.Attributes.GnisID, nil
		}
	}
	return "", "", nil
}
