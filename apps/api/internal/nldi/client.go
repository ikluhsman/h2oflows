package nldi

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

// BaseURL points at the current authoritative NLDI endpoint. The older
// labs.waterdata.usgs.gov host was retired in favour of api.water.usgs.gov.
const BaseURL = "https://api.water.usgs.gov/nldi/linked-data"

var httpClient = &http.Client{Timeout: 20 * time.Second}

// Client is a minimal NLDI HTTP client. Construct with New().
type Client struct {
	baseURL string
	hc      *http.Client
}

func New() *Client {
	return &Client{baseURL: BaseURL, hc: httpClient}
}

// NewWithBase is for tests / alternate deployments.
func NewWithBase(base string, hc *http.Client) *Client {
	if hc == nil {
		hc = httpClient
	}
	return &Client{baseURL: base, hc: hc}
}

// SnapToComID resolves a lat/lng to the nearest NHD reach.
// NLDI expects coords as POINT(lng lat) — longitude first.
func (c *Client) SnapToComID(ctx context.Context, lat, lng float64) (*SnapResult, error) {
	point := url.QueryEscape(fmt.Sprintf("POINT(%f %f)", lng, lat))
	u := fmt.Sprintf("%s/comid/position?coords=%s", c.baseURL, point)
	var coll Collection
	if err := c.get(ctx, u, &coll); err != nil {
		return nil, err
	}
	if len(coll.Features) == 0 {
		return nil, fmt.Errorf("nldi: no reach found at (%f, %f)", lat, lng)
	}
	f := coll.Features[0]
	return &SnapResult{ComID: f.Props.Identifier, Name: f.Props.Name}, nil
}

// UpstreamFlowlines returns all NHD flowlines upstream of comid within distance (km).
// NLDI caps distance at ~9999; callers should pick a bound appropriate to the basin.
func (c *Client) UpstreamFlowlines(ctx context.Context, comid string, distanceKm int) (*Collection, error) {
	return c.navigate(ctx, comid, "UT", "flowlines", distanceKm)
}

// UpstreamGauges returns USGS gauge sites upstream of comid within distance (km).
func (c *Client) UpstreamGauges(ctx context.Context, comid string, distanceKm int) (*Collection, error) {
	return c.navigate(ctx, comid, "UT", "nwissite", distanceKm)
}

// DownstreamFlowlines returns mainstem flowlines downstream of comid (DM = downstream mainstem).
func (c *Client) DownstreamFlowlines(ctx context.Context, comid string, distanceKm int) (*Collection, error) {
	return c.navigate(ctx, comid, "DM", "flowlines", distanceKm)
}

// DownstreamGauges returns USGS gauges along the downstream mainstem.
func (c *Client) DownstreamGauges(ctx context.Context, comid string, distanceKm int) (*Collection, error) {
	return c.navigate(ctx, comid, "DM", "nwissite", distanceKm)
}

func (c *Client) navigate(ctx context.Context, comid, mode, dataSource string, distanceKm int) (*Collection, error) {
	u := fmt.Sprintf("%s/comid/%s/navigation/%s/%s?distance=%d", c.baseURL, comid, mode, dataSource, distanceKm)
	var coll Collection
	if err := c.get(ctx, u, &coll); err != nil {
		return nil, err
	}
	return &coll, nil
}

func (c *Client) get(ctx context.Context, u string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "h2oflows/1.0 (https://h2oflows.org)")
	resp, err := c.hc.Do(req)
	if err != nil {
		return fmt.Errorf("nldi request: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("nldi read: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		snippet := string(body)
		if len(snippet) > 200 {
			snippet = snippet[:200]
		}
		return fmt.Errorf("nldi %s: status %d — %s", u, resp.StatusCode, snippet)
	}
	if err := json.Unmarshal(body, out); err != nil {
		return fmt.Errorf("nldi parse: %w", err)
	}
	return nil
}
