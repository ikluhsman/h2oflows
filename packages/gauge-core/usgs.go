package gauge

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const (
	usgsOGCBase  = "https://api.waterdata.usgs.gov/ogcapi/v0"
	usgsNWISBase = "https://waterservices.usgs.gov/nwis"
)

// USGSSource implements GaugeSource and SiteDiscoverer for USGS Water Services.
//
// An API key is optional — unauthenticated requests work but have lower rate
// limits. A key matters most during bulk DiscoverSites calls. Get a free key at
// https://api.waterdata.usgs.gov
type USGSSource struct {
	apiKey     string
	httpClient *http.Client
	ogcBase    string // override in tests via httptest server URL
	nwisBase   string // override in tests via httptest server URL
}

// NewUSGSSource creates a USGSSource. Pass an empty string for apiKey to use
// unauthenticated access.
func NewUSGSSource(apiKey string) *USGSSource {
	return &USGSSource{
		apiKey:   apiKey,
		httpClient: &http.Client{
			Timeout: 15 * time.Second,
		},
		ogcBase:  usgsOGCBase,
		nwisBase: usgsNWISBase,
	}
}

func (s *USGSSource) Name() string           { return "USGS Water Services" }
func (s *USGSSource) SourceType() SourceType { return SourceUSGS }

// FetchReading returns the most recent instantaneous discharge reading for the
// given USGS site number (e.g. "09361500"). Uses the USGS OGC API.
func (s *USGSSource) FetchReading(ctx context.Context, externalID string) (*Reading, error) {
	params := url.Values{
		"monitoring_location_id": {fmt.Sprintf("USGS-%s", externalID)},
		"parameter_code":         {"00060"},
		"statistic_id":           {"00011"}, // instantaneous value
	}
	endpoint := fmt.Sprintf("%s/collections/latest-continuous/items?%s", s.ogcBase, params.Encode())

	resp, err := s.get(ctx, endpoint)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrSourceUnavailable, err)
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusNotFound:
		return nil, ErrGaugeNotFound
	case http.StatusOK:
		// continue
	default:
		return nil, fmt.Errorf("%w: HTTP %d", ErrSourceUnavailable, resp.StatusCode)
	}

	var fc ogcFeatureCollection
	if err := json.NewDecoder(resp.Body).Decode(&fc); err != nil {
		return nil, fmt.Errorf("decoding OGC response: %w", err)
	}
	if len(fc.Features) == 0 {
		return nil, ErrNoReadings
	}

	return fc.Features[0].toReading(externalID)
}

// FetchHistory returns all instantaneous discharge readings since the given
// time, oldest first. Uses the NWIS Instantaneous Values service.
func (s *USGSSource) FetchHistory(ctx context.Context, externalID string, since time.Time) ([]*Reading, error) {
	params := url.Values{
		"sites":       {externalID},
		"parameterCd": {"00060"},
		"startDT":     {since.UTC().Format(time.RFC3339)},
		"format":      {"json"},
	}
	endpoint := fmt.Sprintf("%s/iv/?%s", s.nwisBase, params.Encode())

	resp, err := s.get(ctx, endpoint)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrSourceUnavailable, err)
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusNotFound:
		return nil, ErrGaugeNotFound
	case http.StatusOK:
		// continue
	default:
		return nil, fmt.Errorf("%w: HTTP %d", ErrSourceUnavailable, resp.StatusCode)
	}

	var iv nwisIVResponse
	if err := json.NewDecoder(resp.Body).Decode(&iv); err != nil {
		return nil, fmt.Errorf("decoding NWIS IV response: %w", err)
	}

	return iv.toReadings(externalID)
}

// DiscoverSites returns USGS streamflow gauge sites matching opts.
// Uses the NWIS Site Service (RDB format) which includes begin/end dates for
// retirement detection. Sites with a non-empty end_date are marked Active=false
// and EndDate set — the sync job uses this to transition gauges to StatusRetired.
//
// When opts.StateCodes is empty and opts.BoundingBox is nil, all lower-48 sites
// are returned. This is a large response; prefer state or bbox filtering for
// routine syncs.
func (s *USGSSource) DiscoverSites(ctx context.Context, opts DiscoverOptions) ([]*SiteMetadata, error) {
	params := url.Values{
		"format":           {"rdb"},
		"siteType":         {"ST"},  // streams only
		"hasDataTypeCd":    {"iv"},  // must have instantaneous values
		"outputDataTypeCd": {"iv"},  // include begin/end date columns in output
	}

	if len(opts.Parameters) > 0 {
		params.Set("parameterCd", strings.Join(opts.Parameters, ","))
	} else {
		params.Set("parameterCd", "00060")
	}

	if opts.ActiveOnly {
		params.Set("siteStatus", "active")
	} else {
		params.Set("siteStatus", "all")
	}

	// BoundingBox takes precedence over StateCodes
	if opts.BoundingBox != nil {
		params.Set("bBox", fmt.Sprintf("%.6f,%.6f,%.6f,%.6f",
			opts.BoundingBox.West, opts.BoundingBox.South,
			opts.BoundingBox.East, opts.BoundingBox.North))
	} else if len(opts.StateCodes) > 0 {
		params.Set("stateCd", strings.Join(opts.StateCodes, ","))
	}

	endpoint := fmt.Sprintf("%s/site/?%s", s.nwisBase, params.Encode())

	resp, err := s.get(ctx, endpoint)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrSourceUnavailable, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%w: HTTP %d", ErrSourceUnavailable, resp.StatusCode)
	}

	return parseNWISSiteRDB(resp.Body)
}

// get performs a GET request, attaching the API key header if one is set.
func (s *USGSSource) get(ctx context.Context, rawURL string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, err
	}
	if s.apiKey != "" {
		req.Header.Set("X-Api-Key", s.apiKey)
	}
	return s.httpClient.Do(req)
}

// --- OGC API response types -------------------------------------------------

type ogcFeatureCollection struct {
	Features []ogcFeature `json:"features"`
}

type ogcFeature struct {
	Properties struct {
		Value      json.RawMessage `json:"value"`
		Time       string          `json:"time"`
		Qualifiers []string        `json:"qualifiers"`
	} `json:"properties"`
}

func (f *ogcFeature) toReading(externalID string) (*Reading, error) {
	val, unit, err := parseOGCValue(f.Properties.Value)
	if err != nil {
		return nil, err
	}

	ts, err := parseTimestamp(f.Properties.Time)
	if err != nil {
		return nil, fmt.Errorf("parsing OGC timestamp %q: %w", f.Properties.Time, err)
	}

	return &Reading{
		ExternalID:  externalID,
		Value:       val,
		Unit:        normalizeUnit(unit),
		Timestamp:   ts,
		QualCode:    strings.Join(f.Properties.Qualifiers, ","),
		Provisional: containsQualifier(f.Properties.Qualifiers, "P"),
	}, nil
}

// parseOGCValue handles the OGC API's value field, which can be either a bare
// number (123.4) or a nested object ({"value": 123.4, "unit": "ft3/s"}).
func parseOGCValue(raw json.RawMessage) (float64, string, error) {
	// Try nested object first
	var nested struct {
		Value float64 `json:"value"`
		Unit  string  `json:"unit"`
	}
	if err := json.Unmarshal(raw, &nested); err == nil {
		return nested.Value, nested.Unit, nil
	}
	// Fall back to bare number
	var f float64
	if err := json.Unmarshal(raw, &f); err == nil {
		return f, "", nil
	}
	return 0, "", fmt.Errorf("cannot parse OGC value field: %s", raw)
}

// --- NWIS IV response types -------------------------------------------------

type nwisIVResponse struct {
	Value struct {
		TimeSeries []struct {
			Variable struct {
				Unit struct {
					UnitCode string `json:"unitCode"`
				} `json:"unit"`
				NoDataValue float64 `json:"noDataValue"`
			} `json:"variable"`
			Values []struct {
				Value []struct {
					Value      string   `json:"value"`
					Qualifiers []string `json:"qualifiers"`
					DateTime   string   `json:"dateTime"`
				} `json:"value"`
			} `json:"values"`
		} `json:"timeSeries"`
	} `json:"value"`
}

func (r *nwisIVResponse) toReadings(externalID string) ([]*Reading, error) {
	if len(r.Value.TimeSeries) == 0 {
		return nil, ErrNoReadings
	}

	ts := r.Value.TimeSeries[0]
	unit := normalizeUnit(ts.Variable.Unit.UnitCode)
	noData := ts.Variable.NoDataValue

	if len(ts.Values) == 0 || len(ts.Values[0].Value) == 0 {
		return nil, ErrNoReadings
	}

	raw := ts.Values[0].Value
	readings := make([]*Reading, 0, len(raw))

	for _, v := range raw {
		f, err := strconv.ParseFloat(v.Value, 64)
		if err != nil || f == noData {
			continue
		}

		t, err := parseTimestamp(v.DateTime)
		if err != nil {
			continue // skip unreadable timestamps rather than failing the whole batch
		}

		readings = append(readings, &Reading{
			ExternalID:  externalID,
			Value:       f,
			Unit:        unit,
			Timestamp:   t,
			QualCode:    strings.Join(v.Qualifiers, ","),
			Provisional: containsQualifier(v.Qualifiers, "P"),
		})
	}

	return readings, nil
}

// --- NWIS Site RDB parser ----------------------------------------------------

// parseNWISSiteRDB parses the NWIS site service tab-delimited (RDB) response.
//
// RDB format:
//   - Lines starting with # are comments
//   - First non-comment line is tab-separated column names
//   - Second non-comment line is format widths (e.g. "5s\t15s\t...") — skipped
//   - Remaining lines are data rows
//
// Sites that appear multiple times (one row per parameter) are merged into a
// single SiteMetadata entry with all parameter codes collected.
func parseNWISSiteRDB(r io.Reader) ([]*SiteMetadata, error) {
	scanner := bufio.NewScanner(r)

	// Find column header row (first non-comment line)
	var headers []string
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "#") {
			continue
		}
		headers = strings.Split(line, "\t")
		break
	}
	if len(headers) == 0 {
		return nil, fmt.Errorf("no header row in NWIS RDB response")
	}

	// Skip the format-width row
	scanner.Scan()

	col := make(map[string]int, len(headers))
	for i, h := range headers {
		col[h] = i
	}

	get := func(fields []string, name string) string {
		i, ok := col[name]
		if !ok || i >= len(fields) {
			return ""
		}
		return strings.TrimSpace(fields[i])
	}

	// Use a map to merge rows for the same site (one row per parameter in RDB)
	siteMap := make(map[string]*SiteMetadata)

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		fields := strings.Split(line, "\t")
		// Trailing empty columns (e.g. end_date on active sites) are often
		// omitted in RDB output. The get() helper handles out-of-bounds safely,
		// so we only need enough fields to hold at least the site number.
		if len(fields) < 2 {
			continue
		}

		externalID := get(fields, "site_no")
		if externalID == "" {
			continue
		}

		// If we've seen this site, just append the parameter code
		if existing, ok := siteMap[externalID]; ok {
			if parm := get(fields, "parm_cd"); parm != "" {
				existing.Parameters = append(existing.Parameters, parm)
			}
			continue
		}

		lat, _ := strconv.ParseFloat(get(fields, "dec_lat_va"), 64)
		lng, _ := strconv.ParseFloat(get(fields, "dec_long_va"), 64)
		drainArea, _ := strconv.ParseFloat(get(fields, "drain_area_va"), 64)

		beginDate := parseNWISDate(get(fields, "begin_date"))
		endDate := parseNWISDatePtr(get(fields, "end_date"))

		site := &SiteMetadata{
			ExternalID:       externalID,
			Name:             get(fields, "station_nm"),
			StateCode:        fipsToAlpha(get(fields, "state_cd")),
			CountyCode:       get(fields, "county_cd"),
			HUCCode:          get(fields, "huc_cd"),
			Parameters:       []string{},
			Active:           endDate == nil,
			BeginDate:        beginDate,
			EndDate:          endDate,
			DrainageAreaSqMi: drainArea,
			SourceType:       SourceUSGS,
		}

		if lat != 0 && lng != 0 {
			site.Location = &LatLng{Lat: lat, Lng: lng}
		}

		if parm := get(fields, "parm_cd"); parm != "" {
			site.Parameters = append(site.Parameters, parm)
		}

		siteMap[externalID] = site
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	sites := make([]*SiteMetadata, 0, len(siteMap))
	for _, s := range siteMap {
		sites = append(sites, s)
	}
	return sites, nil
}

// --- Helpers ----------------------------------------------------------------

// parseTimestamp tries RFC3339Nano then RFC3339. NWIS IV timestamps carry
// milliseconds and variable timezone offsets; both formats cover the range.
func parseTimestamp(s string) (time.Time, error) {
	if t, err := time.Parse(time.RFC3339Nano, s); err == nil {
		return t, nil
	}
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t, nil
	}
	return time.Time{}, fmt.Errorf("unrecognized timestamp format: %q", s)
}

func parseNWISDate(s string) time.Time {
	t, _ := time.Parse("2006-01-02", s)
	return t
}

func parseNWISDatePtr(s string) *time.Time {
	if s == "" {
		return nil
	}
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		return nil
	}
	return &t
}

// normalizeUnit maps USGS unit strings to the short codes used in the app.
func normalizeUnit(code string) string {
	switch strings.ToLower(strings.TrimSpace(code)) {
	case "ft3/s", "cfs":
		return "cfs"
	case "ft", "feet":
		return "ft"
	case "m3/s":
		return "m3/s"
	default:
		return code
	}
}

// containsQualifier returns true if qualifiers contains the given code
// (case-insensitive).
func containsQualifier(qualifiers []string, code string) bool {
	for _, q := range qualifiers {
		if strings.EqualFold(q, code) {
			return true
		}
	}
	return false
}

// fipsToAlpha converts a 2-digit numeric FIPS state code to a 2-letter alpha
// code. NWIS RDB returns numeric FIPS in the state_cd column. If the value is
// already alphabetic (or unrecognized), it is returned as-is.
func fipsToAlpha(fips string) string {
	codes := map[string]string{
		"01": "AL", "02": "AK", "04": "AZ", "05": "AR", "06": "CA",
		"08": "CO", "09": "CT", "10": "DE", "11": "DC", "12": "FL",
		"13": "GA", "15": "HI", "16": "ID", "17": "IL", "18": "IN",
		"19": "IA", "20": "KS", "21": "KY", "22": "LA", "23": "ME",
		"24": "MD", "25": "MA", "26": "MI", "27": "MN", "28": "MS",
		"29": "MO", "30": "MT", "31": "NE", "32": "NV", "33": "NH",
		"34": "NJ", "35": "NM", "36": "NY", "37": "NC", "38": "ND",
		"39": "OH", "40": "OK", "41": "OR", "42": "PA", "44": "RI",
		"45": "SC", "46": "SD", "47": "TN", "48": "TX", "49": "UT",
		"50": "VT", "51": "VA", "53": "WA", "54": "WV", "55": "WI",
		"56": "WY",
	}
	if alpha, ok := codes[fips]; ok {
		return alpha
	}
	return fips
}
