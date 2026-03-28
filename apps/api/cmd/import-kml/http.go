package main

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

func jsonUnmarshal(data []byte, v any) error {
	return json.Unmarshal(data, v)
}
