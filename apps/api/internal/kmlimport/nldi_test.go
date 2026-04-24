package kmlimport

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/h2oflow/h2oflow/apps/api/internal/nldi"
)

// Mock NLDI responses. Put-in snaps to ComID "100", take-out to "300".
// Downstream mainstem from "100" returns 3 chained flowlines: 100 → 200 → 300.

const snapPutIn = `{
  "type": "FeatureCollection",
  "features": [{
    "type": "Feature",
    "geometry": {"type": "LineString", "coordinates": [[-106.38,40.05],[-106.39,40.06]]},
    "properties": {
      "identifier": "100", "source": "comid", "sourceName": "NHDPlus comid", "comid": "100"
    }
  }]
}`

const snapTakeOut = `{
  "type": "FeatureCollection",
  "features": [{
    "type": "Feature",
    "geometry": {"type": "LineString", "coordinates": [[-106.50,40.10],[-106.51,40.11]]},
    "properties": {"identifier": "300", "name": "Colorado River", "nhdplus_comid": "300"}
  }]
}`

const snapTakeOutStranded = `{
  "type": "FeatureCollection",
  "features": [{
    "type": "Feature",
    "geometry": {"type": "LineString", "coordinates": [[-100.0,40.0],[-100.01,40.01]]},
    "properties": {"identifier": "999", "name": "Other Basin", "nhdplus_comid": "999"}
  }]
}`

const dmFlowlines = `{
  "type": "FeatureCollection",
  "features": [
    {"type":"Feature","geometry":{"type":"LineString","coordinates":[[-106.38,40.05],[-106.40,40.06]]},
     "properties":{"nhdplus_comid":"100"}},
    {"type":"Feature","geometry":{"type":"LineString","coordinates":[[-106.40,40.06],[-106.45,40.08]]},
     "properties":{"nhdplus_comid":"200"}},
    {"type":"Feature","geometry":{"type":"LineString","coordinates":[[-106.45,40.08],[-106.50,40.10]]},
     "properties":{"nhdplus_comid":"300"}}
  ]
}`

// newMockNLDI routes test NLDI requests based on URL shape. coordsMatcher lets
// a single server distinguish put-in vs take-out snap calls by the POINT(...)
// query param.
func newMockNLDI(t *testing.T, putInPoint, takeOutPoint string, snapTakeOutBody string) (*nldi.Client, *httptest.Server) {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case strings.Contains(r.URL.Path, "comid/position"):
			q := r.URL.RawQuery
			switch {
			case strings.Contains(q, putInPoint):
				w.Write([]byte(snapPutIn))
			case strings.Contains(q, takeOutPoint):
				w.Write([]byte(snapTakeOutBody))
			default:
				t.Errorf("unexpected snap query: %s", q)
				http.Error(w, "no match", 500)
			}
		case strings.Contains(r.URL.Path, "/navigation/DM/flowlines"):
			w.Write([]byte(dmFlowlines))
		default:
			t.Errorf("unexpected request: %s", r.URL.Path)
			http.Error(w, "no route", 404)
		}
	}))
	return nldi.NewWithBase(srv.URL, srv.Client()), srv
}

func TestFetchNLDIRiverLine_ok(t *testing.T) {
	// URL-encoded POINT queries as emitted by SnapToComID.
	putInPoint := "POINT%28-106.380000+40.050000%29"
	takeOutPoint := "POINT%28-106.500000+40.100000%29"

	c, srv := newMockNLDI(t, putInPoint, takeOutPoint, snapTakeOut)
	defer srv.Close()

	line, err := fetchNLDIRiverLineWithClient(context.Background(), c,
		-106.38, 40.05, -106.50, 40.10)
	if err != nil {
		t.Fatalf("fetchNLDIRiverLine: %v", err)
	}
	if line.PutInComID != "100" {
		t.Errorf("PutInComID = %q, want 100", line.PutInComID)
	}
	if line.TakeOutComID != "300" {
		t.Errorf("TakeOutComID = %q, want 300", line.TakeOutComID)
	}
	// 3 flowlines chained end-to-start → 4 unique coords after MergeMainstem.
	if !strings.HasPrefix(line.GeoJSON, `{"type":"LineString","coordinates":[`) {
		t.Errorf("GeoJSON shape unexpected: %s", line.GeoJSON)
	}
	// Should contain all 4 waypoints.
	for _, want := range []string{"-106.3800000", "-106.4000000", "-106.4500000", "-106.5000000"} {
		if !strings.Contains(line.GeoJSON, want) {
			t.Errorf("GeoJSON missing %s: %s", want, line.GeoJSON)
		}
	}
}

func TestFetchNLDIRiverLine_takeOutNotDownstream(t *testing.T) {
	putInPoint := "POINT%28-106.380000+40.050000%29"
	takeOutPoint := "POINT%28-100.000000+40.000000%29"

	c, srv := newMockNLDI(t, putInPoint, takeOutPoint, snapTakeOutStranded)
	defer srv.Close()

	_, err := fetchNLDIRiverLineWithClient(context.Background(), c,
		-106.38, 40.05, -100.00, 40.00)
	if err == nil {
		t.Fatal("expected error when take-out is not downstream of put-in")
	}
	if !strings.Contains(err.Error(), "not downstream") {
		t.Errorf("error should explain not-downstream: %v", err)
	}
}
