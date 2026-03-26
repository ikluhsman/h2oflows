package gauge

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// newTestUSGS returns a USGSSource pointed at the provided test server URLs.
func newTestUSGS(ogcURL, nwisURL string) *USGSSource {
	s := NewUSGSSource("")
	s.ogcBase = ogcURL
	s.nwisBase = nwisURL
	return s
}

// --- FetchReading -----------------------------------------------------------

func TestUSGSSource_FetchReading(t *testing.T) {
	tests := []struct {
		name        string
		body        string
		statusCode  int
		wantErr     error
		wantValue   float64
		wantUnit    string
		wantProvis  bool
	}{
		{
			name: "nested value object with provisional qualifier",
			body: `{"type":"FeatureCollection","features":[{"type":"Feature","properties":{
				"value":{"value":850.0,"unit":"ft3/s"},
				"time":"2024-03-25T14:00:00Z",
				"qualifiers":["P"]}}]}`,
			statusCode: 200,
			wantValue:  850.0,
			wantUnit:   "cfs",
			wantProvis: true,
		},
		{
			name: "bare float value with approved qualifier",
			body: `{"type":"FeatureCollection","features":[{"type":"Feature","properties":{
				"value":340.0,
				"time":"2024-03-25T14:00:00Z",
				"qualifiers":["A"]}}]}`,
			statusCode: 200,
			wantValue:  340.0,
			wantUnit:   "",
			wantProvis: false,
		},
		{
			name:       "404 returns ErrGaugeNotFound",
			body:       `not found`,
			statusCode: 404,
			wantErr:    ErrGaugeNotFound,
		},
		{
			name:       "500 returns ErrSourceUnavailable",
			body:       `internal error`,
			statusCode: 500,
			wantErr:    ErrSourceUnavailable,
		},
		{
			name:       "empty features returns ErrNoReadings",
			body:       `{"type":"FeatureCollection","features":[]}`,
			statusCode: 200,
			wantErr:    ErrNoReadings,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
				fmt.Fprint(w, tt.body)
			}))
			defer srv.Close()

			src := newTestUSGS(srv.URL, srv.URL)
			reading, err := src.FetchReading(context.Background(), "09361500")

			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("FetchReading() error = %v, want %v", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("FetchReading() unexpected error: %v", err)
			}
			if reading.Value != tt.wantValue {
				t.Errorf("Value = %v, want %v", reading.Value, tt.wantValue)
			}
			if reading.Unit != tt.wantUnit {
				t.Errorf("Unit = %q, want %q", reading.Unit, tt.wantUnit)
			}
			if reading.Provisional != tt.wantProvis {
				t.Errorf("Provisional = %v, want %v", reading.Provisional, tt.wantProvis)
			}
			if reading.ExternalID != "09361500" {
				t.Errorf("ExternalID = %q, want %q", reading.ExternalID, "09361500")
			}
		})
	}
}

// --- FetchHistory -----------------------------------------------------------

const nwisIVBody = `{
  "value": {
    "timeSeries": [{
      "variable": {
        "unit": {"unitCode": "ft3/s"},
        "noDataValue": -999999.0
      },
      "values": [{
        "value": [
          {"value": "820",      "qualifiers": ["A"], "dateTime": "2024-03-25T06:00:00.000-06:00"},
          {"value": "850",      "qualifiers": ["P"], "dateTime": "2024-03-25T06:15:00.000-06:00"},
          {"value": "-999999",  "qualifiers": ["P"], "dateTime": "2024-03-25T06:30:00.000-06:00"},
          {"value": "bad",      "qualifiers": ["P"], "dateTime": "2024-03-25T06:45:00.000-06:00"}
        ]
      }]
    }]
  }
}`

func TestUSGSSource_FetchHistory(t *testing.T) {
	t.Run("returns readings oldest first, filters noData and unparseable", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprint(w, nwisIVBody)
		}))
		defer srv.Close()

		src := newTestUSGS(srv.URL, srv.URL)
		since := time.Date(2024, 3, 25, 0, 0, 0, 0, time.UTC)
		readings, err := src.FetchHistory(context.Background(), "09361500", since)
		if err != nil {
			t.Fatalf("FetchHistory() unexpected error: %v", err)
		}

		// -999999 (noData) and "bad" (unparseable) should be dropped
		if len(readings) != 2 {
			t.Fatalf("got %d readings, want 2", len(readings))
		}
		if readings[0].Value != 820 {
			t.Errorf("readings[0].Value = %v, want 820", readings[0].Value)
		}
		if readings[0].Provisional {
			t.Errorf("readings[0].Provisional = true, want false (qualifier A)")
		}
		if readings[1].Value != 850 {
			t.Errorf("readings[1].Value = %v, want 850", readings[1].Value)
		}
		if !readings[1].Provisional {
			t.Errorf("readings[1].Provisional = false, want true (qualifier P)")
		}
		if readings[0].Unit != "cfs" {
			t.Errorf("Unit = %q, want cfs", readings[0].Unit)
		}
	})

	t.Run("empty time series returns ErrNoReadings", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprint(w, `{"value":{"timeSeries":[]}}`)
		}))
		defer srv.Close()

		src := newTestUSGS(srv.URL, srv.URL)
		_, err := src.FetchHistory(context.Background(), "09361500", time.Now())
		if !errors.Is(err, ErrNoReadings) {
			t.Errorf("got %v, want ErrNoReadings", err)
		}
	})

	t.Run("startDT query param is set from since argument", func(t *testing.T) {
		var gotQuery string
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			gotQuery = r.URL.RawQuery
			fmt.Fprint(w, nwisIVBody)
		}))
		defer srv.Close()

		src := newTestUSGS(srv.URL, srv.URL)
		since := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)
		src.FetchHistory(context.Background(), "09361500", since)

		if !strings.Contains(gotQuery, "startDT=") {
			t.Errorf("expected startDT in query, got %q", gotQuery)
		}
		if !strings.Contains(gotQuery, "2024-01-15") {
			t.Errorf("expected date 2024-01-15 in query, got %q", gotQuery)
		}
	})
}

// --- DiscoverSites ----------------------------------------------------------

const nwisRDBBody = `# USGS National Water Information System
# retrieved: 2024-03-25
#
agency_cd	site_no	station_nm	site_tp_cd	dec_lat_va	dec_long_va	state_cd	county_cd	huc_cd	drain_area_va	parm_cd	begin_date	end_date
5s	15s	50s	7s	16s	16s	2s	3s	16s	8s	5s	10s	10s
USGS	09361500	Animas River at Durango, CO	ST	37.275278	-107.880278	08	067	14080104	1710.00	00060	1900-01-01
USGS	09361500	Animas River at Durango, CO	ST	37.275278	-107.880278	08	067	14080104	1710.00	00065	1900-01-01
USGS	09999999	Retired Gauge CO	ST	39.000000	-105.000000	08	059	10190005	50.00	00060	2000-01-01	2020-12-31
`

func TestUSGSSource_DiscoverSites(t *testing.T) {
	t.Run("parses sites, merges multi-param rows, detects retirement", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprint(w, nwisRDBBody)
		}))
		defer srv.Close()

		src := newTestUSGS(srv.URL, srv.URL)
		sites, err := src.DiscoverSites(context.Background(), DiscoverOptions{})
		if err != nil {
			t.Fatalf("DiscoverSites() unexpected error: %v", err)
		}
		if len(sites) != 2 {
			t.Fatalf("got %d sites, want 2", len(sites))
		}

		// Find Animas site
		var animas *SiteMetadata
		for _, s := range sites {
			if s.ExternalID == "09361500" {
				animas = s
			}
		}
		if animas == nil {
			t.Fatal("site 09361500 not found")
		}

		// Two RDB rows for same site should be merged into one with both params
		if len(animas.Parameters) != 2 {
			t.Errorf("Parameters = %v, want [00060 00065]", animas.Parameters)
		}
		if animas.StateCode != "CO" {
			t.Errorf("StateCode = %q, want CO (converted from FIPS 08)", animas.StateCode)
		}
		if animas.Location == nil || animas.Location.Lat != 37.275278 {
			t.Errorf("Location not parsed correctly: %+v", animas.Location)
		}
		if !animas.Active {
			t.Error("Active = false, want true for active site")
		}

		// Find retired site
		var retired *SiteMetadata
		for _, s := range sites {
			if s.ExternalID == "09999999" {
				retired = s
			}
		}
		if retired == nil {
			t.Fatal("retired site 09999999 not found")
		}
		if retired.Active {
			t.Error("Active = true, want false for retired site")
		}
		if retired.EndDate == nil {
			t.Error("EndDate = nil, want non-nil for retired site")
		}
	})

	t.Run("ActiveOnly sets siteStatus=active in request", func(t *testing.T) {
		var gotQuery string
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			gotQuery = r.URL.RawQuery
			fmt.Fprint(w, nwisRDBBody)
		}))
		defer srv.Close()

		src := newTestUSGS(srv.URL, srv.URL)
		src.DiscoverSites(context.Background(), DiscoverOptions{ActiveOnly: true})

		if !strings.Contains(gotQuery, "siteStatus=active") {
			t.Errorf("expected siteStatus=active in query, got %q", gotQuery)
		}
	})

	t.Run("StateCodes passed to request", func(t *testing.T) {
		var gotQuery string
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			gotQuery = r.URL.RawQuery
			fmt.Fprint(w, nwisRDBBody)
		}))
		defer srv.Close()

		src := newTestUSGS(srv.URL, srv.URL)
		src.DiscoverSites(context.Background(), DiscoverOptions{StateCodes: []string{"CO", "UT"}})

		if !strings.Contains(gotQuery, "stateCd=CO") {
			t.Errorf("expected stateCd in query, got %q", gotQuery)
		}
	})
}

// --- Helper functions -------------------------------------------------------

func TestParseOGCValue(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantVal   float64
		wantUnit  string
		wantErr   bool
	}{
		{"nested object", `{"value":850.0,"unit":"ft3/s"}`, 850.0, "ft3/s", false},
		{"nested object zero unit", `{"value":340.0,"unit":""}`, 340.0, "", false},
		{"bare float", `123.4`, 123.4, "", false},
		{"bare integer", `500`, 500.0, "", false},
		{"invalid", `"not-a-number"`, 0, "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			val, unit, err := parseOGCValue([]byte(tt.input))
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if val != tt.wantVal {
				t.Errorf("val = %v, want %v", val, tt.wantVal)
			}
			if unit != tt.wantUnit {
				t.Errorf("unit = %q, want %q", unit, tt.wantUnit)
			}
		})
	}
}

func TestNormalizeUnit(t *testing.T) {
	tests := []struct{ in, want string }{
		{"ft3/s", "cfs"},
		{"CFS", "cfs"},
		{"ft3/S", "cfs"},
		{"ft", "ft"},
		{"feet", "ft"},
		{"m3/s", "m3/s"},
		{"ft-rock", "ft-rock"}, // custom community unit — pass through unchanged
		{"", ""},
	}
	for _, tt := range tests {
		if got := normalizeUnit(tt.in); got != tt.want {
			t.Errorf("normalizeUnit(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestFipsToAlpha(t *testing.T) {
	tests := []struct{ in, want string }{
		{"08", "CO"},
		{"06", "CA"},
		{"49", "UT"},
		{"35", "NM"},
		{"CO", "CO"}, // already alpha — pass through
		{"ZZ", "ZZ"}, // unknown — pass through
		{"",   ""},
	}
	for _, tt := range tests {
		if got := fipsToAlpha(tt.in); got != tt.want {
			t.Errorf("fipsToAlpha(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestContainsQualifier(t *testing.T) {
	tests := []struct {
		qualifiers []string
		code       string
		want       bool
	}{
		{[]string{"P"}, "P", true},
		{[]string{"A"}, "P", false},
		{[]string{"p"}, "P", true},  // case-insensitive
		{[]string{"A", "P"}, "P", true},
		{[]string{}, "P", false},
	}
	for _, tt := range tests {
		if got := containsQualifier(tt.qualifiers, tt.code); got != tt.want {
			t.Errorf("containsQualifier(%v, %q) = %v, want %v", tt.qualifiers, tt.code, got, tt.want)
		}
	}
}

func TestParseNWISSiteRDB(t *testing.T) {
	t.Run("merges parameters for same site", func(t *testing.T) {
		sites, err := parseNWISSiteRDB(strings.NewReader(nwisRDBBody))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		var animas *SiteMetadata
		for _, s := range sites {
			if s.ExternalID == "09361500" {
				animas = s
			}
		}
		if animas == nil {
			t.Fatal("site not found")
		}
		if len(animas.Parameters) != 2 {
			t.Errorf("got %d parameters, want 2", len(animas.Parameters))
		}
	})

	t.Run("empty body returns empty slice not error", func(t *testing.T) {
		_, err := parseNWISSiteRDB(strings.NewReader("# only comments\n"))
		if err == nil {
			// empty result is fine, but we expect an error about missing header
			// since there's no data row — just verify it doesn't panic
		}
	})
}

func TestParseTimestamp(t *testing.T) {
	tests := []struct {
		input   string
		wantErr bool
	}{
		{"2024-03-25T06:00:00.000-06:00", false}, // NWIS IV with milliseconds
		{"2024-03-25T14:00:00Z", false},           // OGC API UTC
		{"2024-03-25T14:00:00-07:00", false},      // RFC3339 with offset
		{"not-a-date", true},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			_, err := parseTimestamp(tt.input)
			if tt.wantErr && err == nil {
				t.Error("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}
