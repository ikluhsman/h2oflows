package nldi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// snapResponse is a minimal NLDI position response for a single flowline feature.
const snapResponse = `{
  "type": "FeatureCollection",
  "features": [{
    "type": "Feature",
    "geometry": {"type": "LineString", "coordinates": [[-106.38, 40.05], [-106.39, 40.06]]},
    "properties": {
      "identifier": "14837340",
      "name": "Colorado River",
      "nhdplus_comid": "14837340"
    }
  }]
}`

const emptyCollection = `{"type":"FeatureCollection","features":[]}`

func newTestClient(handler http.Handler) (*Client, *httptest.Server) {
	srv := httptest.NewServer(handler)
	return NewWithBase(srv.URL, srv.Client()), srv
}

func TestSnapToComID_ok(t *testing.T) {
	c, srv := newTestClient(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "comid/position") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if !strings.Contains(r.URL.RawQuery, "coords=") {
			t.Errorf("missing coords query param: %s", r.URL.RawQuery)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(snapResponse))
	}))
	defer srv.Close()

	res, err := c.SnapToComID(context.Background(), 40.05, -106.38)
	if err != nil {
		t.Fatal(err)
	}
	if res.ComID != "14837340" {
		t.Errorf("expected ComID 14837340, got %q", res.ComID)
	}
	if res.Name != "Colorado River" {
		t.Errorf("expected name Colorado River, got %q", res.Name)
	}
}

func TestSnapToComID_noFeatures(t *testing.T) {
	c, srv := newTestClient(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(emptyCollection))
	}))
	defer srv.Close()

	_, err := c.SnapToComID(context.Background(), 40.0, -106.0)
	if err == nil {
		t.Fatal("expected error for empty feature collection")
	}
}

func TestSnapToComID_serverError(t *testing.T) {
	c, srv := newTestClient(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "internal server error", http.StatusInternalServerError)
	}))
	defer srv.Close()

	_, err := c.SnapToComID(context.Background(), 40.0, -106.0)
	if err == nil {
		t.Fatal("expected error on 500")
	}
}

func TestNavigate_urlShape(t *testing.T) {
	var gotPath string
	c, srv := newTestClient(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path + "?" + r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(emptyCollection))
	}))
	defer srv.Close()

	_, err := c.UpstreamFlowlines(context.Background(), "14837340", 300)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(gotPath, "comid/14837340/navigation/UT/flowlines") {
		t.Errorf("unexpected URL path: %s", gotPath)
	}
	if !strings.Contains(gotPath, "distance=300") {
		t.Errorf("missing distance param: %s", gotPath)
	}
}

func TestDownstreamGauges_urlShape(t *testing.T) {
	var gotPath string
	c, srv := newTestClient(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(emptyCollection))
	}))
	defer srv.Close()

	c.DownstreamGauges(context.Background(), "14837340", 500)
	if !strings.Contains(gotPath, "navigation/DM/nwissite") {
		t.Errorf("unexpected path for DownstreamGauges: %s", gotPath)
	}
}
