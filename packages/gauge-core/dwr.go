package gauge

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

const (
	dwrAPIBase = "https://dwr.state.co.us/Rest/GET/api/v2"

	// dwrDateFormat is the date format accepted by DWR query parameters.
	dwrDateFormat = "01/02/2006"
)

// DWRSource implements GaugeSource and SiteDiscoverer for the Colorado
// Division of Water Resources telemetry API. No API key is required.
//
// The primary identifier for DWR gauges is the station abbreviation
// (e.g. "PLAWATCO"), not a numeric ID. This is what gets stored as
// external_id in the gauges table for DWR-sourced gauges.
//
// Many DWR stations are seasonal — active during irrigation season
// (roughly April–October) and offline in winter. The poller's
// consecutive_failures tracking handles this automatically; gauges that
// go quiet in November and return in April will transition through
// StatusMaintenance rather than StatusRetired.
type DWRSource struct {
	httpClient *http.Client
	apiBase    string // override in tests via httptest server URL
}

// NewDWRSource creates a DWRSource.
func NewDWRSource() *DWRSource {
	return &DWRSource{
		httpClient: &http.Client{
			Timeout: 15 * time.Second,
		},
		apiBase: dwrAPIBase,
	}
}

func (s *DWRSource) Name() string           { return "Colorado DWR" }
func (s *DWRSource) SourceType() SourceType { return SourceDWR }

// FetchReading returns the most recent discharge reading for the given DWR
// station abbreviation (e.g. "PLAWATCO"). Fetches the last 2 days of raw
// telemetry and returns the most recent non-null value.
func (s *DWRSource) FetchReading(ctx context.Context, externalID string) (*Reading, error) {
	params := url.Values{
		"abbrev":       {externalID},
		"parameter":    {"DISCHRG"},
		"min-modified": {"-2days"},
		"format":       {"json"},
	}
	endpoint := fmt.Sprintf("%s/telemetrystations/telemetrytimeseriesraw/?%s", s.apiBase, params.Encode())

	resp, err := s.get(ctx, endpoint)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrSourceUnavailable, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, ErrGaugeNotFound
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%w: HTTP %d", ErrSourceUnavailable, resp.StatusCode)
	}

	var result dwrResponse[dwrReading]
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decoding DWR response: %w", err)
	}
	if len(result.ResultList) == 0 {
		return nil, ErrNoReadings
	}

	// ResultList is oldest-first; most recent reading is last
	return result.ResultList[len(result.ResultList)-1].toReading(externalID)
}

// FetchHistory returns discharge readings since the given time, oldest first.
func (s *DWRSource) FetchHistory(ctx context.Context, externalID string, since time.Time) ([]*Reading, error) {
	params := url.Values{
		"abbrev":    {externalID},
		"parameter": {"DISCHRG"},
		"startDate": {since.Format(dwrDateFormat)},
		"endDate":   {time.Now().Format(dwrDateFormat)},
		"format":    {"json"},
	}
	endpoint := fmt.Sprintf("%s/telemetrystations/telemetrytimeseriesraw/?%s", s.apiBase, params.Encode())

	resp, err := s.get(ctx, endpoint)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrSourceUnavailable, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, ErrGaugeNotFound
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%w: HTTP %d", ErrSourceUnavailable, resp.StatusCode)
	}

	var result dwrResponse[dwrReading]
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decoding DWR history response: %w", err)
	}
	if len(result.ResultList) == 0 {
		return nil, ErrNoReadings
	}

	readings := make([]*Reading, 0, len(result.ResultList))
	for _, r := range result.ResultList {
		reading, err := r.toReading(externalID)
		if err != nil {
			continue // skip malformed rows, don't fail the batch
		}
		readings = append(readings, reading)
	}
	return readings, nil
}

// DiscoverSites returns all DWR telemetry stations that measure discharge.
// DWR is Colorado-only and the full station list is small enough (~600 stations)
// that we fetch everything in one request. Most DiscoverOptions fields are
// ignored; BoundingBox filtering is applied client-side after fetching.
func (s *DWRSource) DiscoverSites(ctx context.Context, opts DiscoverOptions) ([]*SiteMetadata, error) {
	params := url.Values{
		"measType": {"DISCHRG"}, // discharge stations only
		"format":   {"json"},
	}
	endpoint := fmt.Sprintf("%s/telemetrystations/telemetrystations/?%s", s.apiBase, params.Encode())

	resp, err := s.get(ctx, endpoint)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrSourceUnavailable, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%w: HTTP %d", ErrSourceUnavailable, resp.StatusCode)
	}

	var result dwrResponse[dwrStation]
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decoding DWR station list: %w", err)
	}

	sites := make([]*SiteMetadata, 0, len(result.ResultList))
	for _, st := range result.ResultList {
		site := st.toSiteMetadata()

		// Client-side bbox filter — DWR API doesn't support server-side bbox
		if opts.BoundingBox != nil && site.Location != nil {
			bb := opts.BoundingBox
			if site.Location.Lat < bb.South || site.Location.Lat > bb.North ||
				site.Location.Lng < bb.West || site.Location.Lng > bb.East {
				continue
			}
		}

		sites = append(sites, site)
	}
	return sites, nil
}

func (s *DWRSource) get(ctx context.Context, rawURL string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, err
	}
	return s.httpClient.Do(req)
}

// --- DWR response types -----------------------------------------------------

// dwrResponse is the generic envelope the DWR API wraps all results in.
type dwrResponse[T any] struct {
	ResultList []T `json:"ResultList"`
}

// dwrReading is a single telemetry measurement from the timeseriesraw endpoint.
//
// Note: field names are inferred from DWR API conventions and the existing
// Python exporter (measValue confirmed). Verify measDateTime format and
// qualityType name against the live API when first testing.
type dwrReading struct {
	MeasValue    float64  `json:"measValue"`
	MeasDateTime string   `json:"measDateTime"` // "2006-01-02T15:04:05" Mountain Time (no zone offset)
	MeasUnit     string   `json:"measUnit"`     // typically "cfs"
	FlagA        *string  `json:"flagA"`        // "A"=approved, "O"=observed, etc.
	FlagB        *string  `json:"flagB"`
}

func (r *dwrReading) toReading(externalID string) (*Reading, error) {
	ts, err := parseDWRDateTime(r.MeasDateTime)
	if err != nil {
		return nil, fmt.Errorf("parsing DWR timestamp %q: %w", r.MeasDateTime, err)
	}

	qualCode := ""
	if r.FlagA != nil {
		qualCode = *r.FlagA
	}
	provisional := qualCode != "A" && qualCode != "a" // "O"=observed (provisional), "A"=approved

	return &Reading{
		ExternalID:  externalID,
		Value:       r.MeasValue,
		Unit:        normalizeUnit(r.MeasUnit),
		Timestamp:   ts,
		QualCode:    qualCode,
		Provisional: provisional,
	}, nil
}

// dwrStation is a station record from the telemetrystations list endpoint.
//
// Note: field names follow DWR API conventions. Verify latitude/longitude
// casing and dataSource field name against the live API when first testing.
type dwrStation struct {
	Abbrev      string  `json:"abbrev"`
	StationName string  `json:"stationName"`
	Latitude    float64 `json:"latitude"`
	Longitude   float64 `json:"longitude"`
	County      string  `json:"county"`
	Division    int     `json:"division"`    // Colorado water division (1–7)
	WaterDistrict int   `json:"waterDistrict"`
	DataSource  string  `json:"dataSource"`
}

func (st *dwrStation) toSiteMetadata() *SiteMetadata {
	site := &SiteMetadata{
		ExternalID:  st.Abbrev,
		Name:        st.StationName,
		StateCode:   "CO", // DWR is Colorado-only
		CountyCode:  st.County,
		Parameters:  []string{"DISCHRG"},
		Active:      true, // station list only returns active stations
		BeginDate:   time.Time{},
		EndDate:     nil,
		SourceType:  SourceDWR,
	}

	if st.Latitude != 0 && st.Longitude != 0 {
		site.Location = &LatLng{Lat: st.Latitude, Lng: st.Longitude}
	}

	return site
}

// --- Helpers ----------------------------------------------------------------

// parseDWRDateTime handles the DWR API's datetime format. DWR uses
// "YYYY-MM-DD HH:MM" in most responses (no seconds, no timezone — Mountain
// Time implied). Falls back to RFC3339 in case the API is updated.
func parseDWRDateTime(s string) (time.Time, error) {
	mt := mountainTime()
	// ISO 8601 without timezone — actual live format as of 2026
	if t, err := time.ParseInLocation("2006-01-02T15:04:05", s, mt); err == nil {
		return t, nil
	}
	// Space-separated variant
	if t, err := time.ParseInLocation("2006-01-02 15:04:05", s, mt); err == nil {
		return t, nil
	}
	// Space-separated without seconds
	if t, err := time.ParseInLocation("2006-01-02 15:04", s, mt); err == nil {
		return t, nil
	}
	// RFC3339 with explicit offset (future-proofing)
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t, nil
	}
	return time.Time{}, fmt.Errorf("unrecognized DWR datetime format: %q", s)
}

// mountainTime returns the America/Denver timezone. DWR timestamps are in
// Mountain Time without an explicit offset.
func mountainTime() *time.Location {
	loc, err := time.LoadLocation("America/Denver")
	if err != nil {
		// Fall back to a fixed -7h offset if the timezone database isn't available
		return time.FixedZone("MST", -7*60*60)
	}
	return loc
}
