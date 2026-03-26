package gauge

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// newTestDWR returns a DWRSource pointed at the provided test server URL.
func newTestDWR(apiURL string) *DWRSource {
	s := NewDWRSource()
	s.apiBase = apiURL
	return s
}

// --- FetchReading -----------------------------------------------------------

const dwrReadingBody = `{
  "ResultList": [
    {"measValue": 120.5, "measDateTime": "2024-03-25 08:00", "measUnit": "cfs", "qualityType": "Approved"},
    {"measValue": 125.0, "measDateTime": "2024-03-25 08:15", "measUnit": "cfs", "qualityType": "Provisional"}
  ]
}`

func TestDWRSource_FetchReading(t *testing.T) {
	t.Run("returns most recent reading (last in ResultList)", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprint(w, dwrReadingBody)
		}))
		defer srv.Close()

		src := newTestDWR(srv.URL)
		reading, err := src.FetchReading(context.Background(), "PLAWATCO")
		if err != nil {
			t.Fatalf("FetchReading() unexpected error: %v", err)
		}
		if reading.Value != 125.0 {
			t.Errorf("Value = %v, want 125.0 (last in list)", reading.Value)
		}
		if !reading.Provisional {
			t.Error("Provisional = false, want true")
		}
		if reading.Unit != "cfs" {
			t.Errorf("Unit = %q, want cfs", reading.Unit)
		}
		if reading.ExternalID != "PLAWATCO" {
			t.Errorf("ExternalID = %q, want PLAWATCO", reading.ExternalID)
		}
	})

	t.Run("approved reading is not provisional", func(t *testing.T) {
		body := `{"ResultList":[{"measValue":100.0,"measDateTime":"2024-03-25 08:00","measUnit":"cfs","qualityType":"Approved"}]}`
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprint(w, body)
		}))
		defer srv.Close()

		src := newTestDWR(srv.URL)
		reading, err := src.FetchReading(context.Background(), "PLAWATCO")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if reading.Provisional {
			t.Error("Provisional = true, want false for Approved reading")
		}
	})

	t.Run("empty ResultList returns ErrNoReadings", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprint(w, `{"ResultList":[]}`)
		}))
		defer srv.Close()

		src := newTestDWR(srv.URL)
		_, err := src.FetchReading(context.Background(), "PLAWATCO")
		if !errors.Is(err, ErrNoReadings) {
			t.Errorf("got %v, want ErrNoReadings", err)
		}
	})

	t.Run("404 returns ErrGaugeNotFound", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(404)
		}))
		defer srv.Close()

		src := newTestDWR(srv.URL)
		_, err := src.FetchReading(context.Background(), "PLAWATCO")
		if !errors.Is(err, ErrGaugeNotFound) {
			t.Errorf("got %v, want ErrGaugeNotFound", err)
		}
	})

	t.Run("abbrev passed as query param", func(t *testing.T) {
		var gotQuery string
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			gotQuery = r.URL.RawQuery
			fmt.Fprint(w, dwrReadingBody)
		}))
		defer srv.Close()

		src := newTestDWR(srv.URL)
		src.FetchReading(context.Background(), "PLAWATCO")

		if gotQuery == "" {
			t.Fatal("no query params sent")
		}
		q := r_parseQuery(gotQuery)
		if q["abbrev"] != "PLAWATCO" {
			t.Errorf("abbrev = %q, want PLAWATCO", q["abbrev"])
		}
		if q["parameter"] != "DISCHRG" {
			t.Errorf("parameter = %q, want DISCHRG", q["parameter"])
		}
	})
}

// --- FetchHistory -----------------------------------------------------------

func TestDWRSource_FetchHistory(t *testing.T) {
	t.Run("returns all readings oldest first", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprint(w, dwrReadingBody)
		}))
		defer srv.Close()

		src := newTestDWR(srv.URL)
		since := time.Date(2024, 3, 20, 0, 0, 0, 0, time.UTC)
		readings, err := src.FetchHistory(context.Background(), "PLAWATCO", since)
		if err != nil {
			t.Fatalf("FetchHistory() unexpected error: %v", err)
		}
		if len(readings) != 2 {
			t.Fatalf("got %d readings, want 2", len(readings))
		}
		if readings[0].Value != 120.5 {
			t.Errorf("readings[0].Value = %v, want 120.5", readings[0].Value)
		}
		if readings[1].Value != 125.0 {
			t.Errorf("readings[1].Value = %v, want 125.0", readings[1].Value)
		}
	})

	t.Run("startDate and endDate in request", func(t *testing.T) {
		var gotQuery string
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			gotQuery = r.URL.RawQuery
			fmt.Fprint(w, dwrReadingBody)
		}))
		defer srv.Close()

		src := newTestDWR(srv.URL)
		since := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)
		src.FetchHistory(context.Background(), "PLAWATCO", since)

		q := r_parseQuery(gotQuery)
		if q["startDate"] != "01/15/2024" {
			t.Errorf("startDate = %q, want 01/15/2024", q["startDate"])
		}
		if q["endDate"] == "" {
			t.Error("endDate not set")
		}
	})

	t.Run("skips malformed rows, returns remaining", func(t *testing.T) {
		body := `{"ResultList":[
			{"measValue": 100.0, "measDateTime": "2024-03-25 08:00", "measUnit": "cfs", "qualityType": "Approved"},
			{"measValue": 105.0, "measDateTime": "not-a-date",       "measUnit": "cfs", "qualityType": "Approved"},
			{"measValue": 110.0, "measDateTime": "2024-03-25 08:30", "measUnit": "cfs", "qualityType": "Approved"}
		]}`
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprint(w, body)
		}))
		defer srv.Close()

		src := newTestDWR(srv.URL)
		readings, err := src.FetchHistory(context.Background(), "PLAWATCO", time.Now())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(readings) != 2 {
			t.Errorf("got %d readings, want 2 (bad timestamp skipped)", len(readings))
		}
	})
}

// --- DiscoverSites ----------------------------------------------------------

const dwrStationBody = `{
  "ResultList": [
    {"abbrev":"PLAWATCO","stationName":"South Platte at Waterton","latitude":39.45,"longitude":-105.10,"county":"Jefferson","division":1,"waterDistrict":8,"dataSource":"DWR"},
    {"abbrev":"PLAGRACO","stationName":"N Fork S Platte at Grant", "latitude":39.41,"longitude":-105.65,"county":"Park",      "division":1,"waterDistrict":23,"dataSource":"DWR"}
  ]
}`

func TestDWRSource_DiscoverSites(t *testing.T) {
	t.Run("returns all stations with correct metadata", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprint(w, dwrStationBody)
		}))
		defer srv.Close()

		src := newTestDWR(srv.URL)
		sites, err := src.DiscoverSites(context.Background(), DiscoverOptions{})
		if err != nil {
			t.Fatalf("DiscoverSites() unexpected error: %v", err)
		}
		if len(sites) != 2 {
			t.Fatalf("got %d sites, want 2", len(sites))
		}

		s := sites[0]
		if s.StateCode != "CO" {
			t.Errorf("StateCode = %q, want CO", s.StateCode)
		}
		if s.SourceType != SourceDWR {
			t.Errorf("SourceType = %q, want dwr", s.SourceType)
		}
		if s.Location == nil {
			t.Fatal("Location is nil")
		}
		if s.Active != true {
			t.Error("Active = false, want true")
		}
		if s.EndDate != nil {
			t.Error("EndDate non-nil for active station")
		}
	})

	t.Run("bbox filters client-side", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprint(w, dwrStationBody)
		}))
		defer srv.Close()

		src := newTestDWR(srv.URL)
		// Bbox that contains PLAWATCO (39.45, -105.10) but not PLAGRACO (39.41, -105.65)
		opts := DiscoverOptions{
			BoundingBox: &BoundingBox{West: -105.30, South: 39.40, East: -105.00, North: 39.50},
		}
		sites, err := src.DiscoverSites(context.Background(), opts)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(sites) != 1 {
			t.Fatalf("got %d sites, want 1 after bbox filter", len(sites))
		}
		if sites[0].ExternalID != "PLAWATCO" {
			t.Errorf("ExternalID = %q, want PLAWATCO", sites[0].ExternalID)
		}
	})
}

// --- Helper functions -------------------------------------------------------

func TestParseDWRDateTime(t *testing.T) {
	mtn, _ := time.LoadLocation("America/Denver")

	tests := []struct {
		input   string
		wantErr bool
		wantH   int // expected hour in Mountain time
	}{
		{"2024-03-25 08:00", false, 8},
		{"2024-03-25 14:30:00", false, 14},
		{"2024-03-25T14:00:00-07:00", false, 15}, // RFC3339 fallback: -07:00 = MST, Denver is MDT (UTC-6) in March → 15:00 local
		{"not-a-date", true, 0},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := parseDWRDateTime(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got.In(mtn).Hour() != tt.wantH {
				t.Errorf("hour = %d, want %d", got.In(mtn).Hour(), tt.wantH)
			}
		})
	}
}

func TestDWRStation_ToSiteMetadata(t *testing.T) {
	t.Run("zero coordinates produce nil Location", func(t *testing.T) {
		st := dwrStation{Abbrev: "TESTCO", StationName: "Test", Latitude: 0, Longitude: 0}
		site := st.toSiteMetadata()
		if site.Location != nil {
			t.Errorf("Location = %+v, want nil for zero coordinates", site.Location)
		}
	})

	t.Run("valid coordinates produce Location", func(t *testing.T) {
		st := dwrStation{Abbrev: "TESTCO", StationName: "Test", Latitude: 39.5, Longitude: -105.1}
		site := st.toSiteMetadata()
		if site.Location == nil {
			t.Fatal("Location is nil")
		}
		if site.Location.Lat != 39.5 {
			t.Errorf("Lat = %v, want 39.5", site.Location.Lat)
		}
	})

	t.Run("always Colorado", func(t *testing.T) {
		st := dwrStation{Abbrev: "TESTCO"}
		site := st.toSiteMetadata()
		if site.StateCode != "CO" {
			t.Errorf("StateCode = %q, want CO", site.StateCode)
		}
	})
}

// r_parseQuery is a minimal query string parser for test assertions.
// Avoids importing net/url just for tests.
func r_parseQuery(raw string) map[string]string {
	m := make(map[string]string)
	for _, pair := range splitAmp(raw) {
		parts := splitEq(pair)
		if len(parts) == 2 {
			m[parts[0]] = unescapeSimple(parts[1])
		}
	}
	return m
}

func splitAmp(s string) []string {
	var parts []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '&' {
			parts = append(parts, s[start:i])
			start = i + 1
		}
	}
	return append(parts, s[start:])
}

func splitEq(s string) []string {
	for i := 0; i < len(s); i++ {
		if s[i] == '=' {
			return []string{s[:i], s[i+1:]}
		}
	}
	return []string{s}
}

func unescapeSimple(s string) string {
	// Handle %2F → / for dates like 01%2F15%2F2024
	result := make([]byte, 0, len(s))
	for i := 0; i < len(s); i++ {
		if s[i] == '%' && i+2 < len(s) {
			var b byte
			fmt.Sscanf(s[i+1:i+3], "%02X", &b)
			result = append(result, b)
			i += 2
		} else {
			result = append(result, s[i])
		}
	}
	return string(result)
}
