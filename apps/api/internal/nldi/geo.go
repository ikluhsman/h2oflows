package nldi

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
)

const (
	tigerweb   = "https://tigerweb.geo.census.gov/arcgis/rest/services/TIGERweb/State_County/MapServer/0/query"
	wbdHUC4    = "https://hydro.nationalmap.gov/arcgis/rest/services/wbd/MapServer/4/query"
	dwrAPIBase = "https://dwr.state.co.us/Rest/GET/api/v2"
)

// DWRStation is a nearby DWR discharge station.
type DWRStation struct {
	ExternalID string
	Name       string
	Lat        float64
	Lng        float64
}

func haversineKm(lat1, lng1, lat2, lng2 float64) float64 {
	const r = 6371.0
	dLat := (lat2 - lat1) * math.Pi / 180
	dLng := (lng2 - lng1) * math.Pi / 180
	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(lat1*math.Pi/180)*math.Cos(lat2*math.Pi/180)*
			math.Sin(dLng/2)*math.Sin(dLng/2)
	return r * 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
}

// DWRNearby fetches all Colorado DWR stream-gage telemetry stations and
// filters client-side to those within distanceKm. The DWR API does not support
// server-side radius filtering, so we fetch the full list (~550 stations).
func DWRNearby(ctx context.Context, lat, lng float64, distanceKm int) ([]DWRStation, error) {
	params := url.Values{
		"stationType": {"Stream Gage"},
		"format":      {"json"},
	}
	endpoint := fmt.Sprintf("%s/telemetrystations/telemetrystation/?%s", dwrAPIBase, params.Encode())
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "h2oflows/1.0 (https://h2oflows.org)")

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("dwr nearby: %w", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	var result struct {
		ResultList []struct {
			Abbrev      string  `json:"abbrev"`
			StationName string  `json:"stationName"`
			Latitude    float64 `json:"latitude"`
			Longitude   float64 `json:"longitude"`
		} `json:"ResultList"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("dwr nearby parse: %w", err)
	}

	out := make([]DWRStation, 0)
	for _, st := range result.ResultList {
		if st.Abbrev == "" || (st.Latitude == 0 && st.Longitude == 0) {
			continue
		}
		if haversineKm(lat, lng, st.Latitude, st.Longitude) > float64(distanceKm) {
			continue
		}
		out = append(out, DWRStation{
			ExternalID: st.Abbrev,
			Name:       st.StationName,
			Lat:        st.Latitude,
			Lng:        st.Longitude,
		})
	}
	return out, nil
}

// StateAt queries the TIGERweb Census service for the US state abbreviation at
// the given coordinate. Returns ("", nil) when the point falls outside all
// state polygons (offshore or non-US).
func StateAt(ctx context.Context, lat, lng float64) (abbr string, err error) {
	params := url.Values{
		"geometry":     {fmt.Sprintf("%f,%f", lng, lat)},
		"geometryType": {"esriGeometryPoint"},
		"spatialRel":   {"esriSpatialRelIntersects"},
		"outFields":    {"STUSAB"},
		"inSR":         {"4326"},
		"f":            {"json"},
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, tigerweb+"?"+params.Encode(), nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "h2oflows/1.0 (https://h2oflows.org)")

	resp, err := httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("tigerweb state: %w", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	var result struct {
		Features []struct {
			Attributes struct {
				STUSAB string `json:"STUSAB"`
			} `json:"attributes"`
		} `json:"features"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("tigerweb state parse: %w", err)
	}
	if len(result.Features) > 0 {
		return result.Features[0].Attributes.STUSAB, nil
	}
	return "", nil
}

// BasinAt queries the USGS WBD HUC4 sub-region layer for the basin name at the
// given coordinate. Returns ("", nil) when no HUC4 polygon covers the point.
func BasinAt(ctx context.Context, lat, lng float64) (name string, err error) {
	params := url.Values{
		"geometry":     {fmt.Sprintf("%f,%f", lng, lat)},
		"geometryType": {"esriGeometryPoint"},
		"spatialRel":   {"esriSpatialRelIntersects"},
		"outFields":    {"name"},
		"inSR":         {"4326"},
		"f":            {"json"},
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, wbdHUC4+"?"+params.Encode(), nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "h2oflows/1.0 (https://h2oflows.org)")

	resp, err := httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("wbd basin: %w", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	var result struct {
		Features []struct {
			Attributes struct {
				Name string `json:"name"`
			} `json:"attributes"`
		} `json:"features"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("wbd basin parse: %w", err)
	}
	if len(result.Features) > 0 {
		return result.Features[0].Attributes.Name, nil
	}
	return "", nil
}
