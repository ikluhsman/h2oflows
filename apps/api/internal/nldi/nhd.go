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

// GNISLookupResult holds the coordinate and HUC8 derived from an NHD GNIS ID query.
type GNISLookupResult struct {
	Lat   float64
	Lng   float64
	HUC8  string // first 8 chars of reachcode
}

// NHDCoordByGNISID queries NHD layer 6 for the first flowline feature with the
// given GNIS ID and returns its first path vertex (WGS84) and the HUC8 derived
// from the reachcode. Returns an error when no feature is found.
func NHDCoordByGNISID(ctx context.Context, gnisID string) (*GNISLookupResult, error) {
	params := url.Values{
		"where":           {fmt.Sprintf("gnis_id='%s'", gnisID)},
		"outFields":       {"gnis_id,reachcode"},
		"returnGeometry":  {"true"},
		"resultRecordCount": {"1"},
		"inSR":            {"4326"},
		"outSR":           {"4326"},
		"f":               {"json"},
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, nhdArcGISURL+"?"+params.Encode(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "h2oflows/1.0 (https://h2oflows.org)")

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("nhd gnis lookup: %w", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	var result struct {
		Features []struct {
			Attributes struct {
				ReachCode string `json:"reachcode"`
			} `json:"attributes"`
			Geometry struct {
				Paths [][][]float64 `json:"paths"`
			} `json:"geometry"`
		} `json:"features"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("nhd gnis lookup parse: %w", err)
	}
	if len(result.Features) == 0 {
		return nil, fmt.Errorf("nhd: no feature found for GNIS ID %q", gnisID)
	}
	f := result.Features[0]
	if len(f.Geometry.Paths) == 0 || len(f.Geometry.Paths[0]) == 0 {
		return nil, fmt.Errorf("nhd: feature has no geometry for GNIS ID %q", gnisID)
	}
	pt := f.Geometry.Paths[0][0] // [lng, lat]
	if len(pt) < 2 {
		return nil, fmt.Errorf("nhd: malformed coordinate for GNIS ID %q", gnisID)
	}
	huc8 := ""
	if len(f.Attributes.ReachCode) >= 8 {
		huc8 = f.Attributes.ReachCode[:8]
	}
	return &GNISLookupResult{Lat: pt[1], Lng: pt[0], HUC8: huc8}, nil
}
