package elevation

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

var client = &http.Client{Timeout: 15 * time.Second}

type epqsResponse struct {
	Value float64 `json:"value"`
}

// QueryElevation returns the elevation in feet for the given WGS-84 coordinate
// using the USGS Elevation Point Query Service (1/3 arc-second resolution).
func QueryElevation(ctx context.Context, lng, lat float64) (float64, error) {
	url := fmt.Sprintf(
		"https://epqs.nationalmap.gov/v1/json?x=%.6f&y=%.6f&wkid=4326&units=Feet&includeDate=false",
		lng, lat,
	)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return 0, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return 0, fmt.Errorf("USGS elevation API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("USGS elevation API returned %d", resp.StatusCode)
	}

	var result epqsResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return 0, fmt.Errorf("parsing USGS elevation response: %w", err)
	}
	return result.Value, nil
}
