//go:build integration

package nldi

import (
	"context"
	"fmt"
	"testing"
	"time"
)

// Run with: go test -tags=integration -v ./internal/nldi/
// Requires internet access to api.water.usgs.gov.

func TestIntegration_SnapKremmling(t *testing.T) {
	c := New()
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// Colorado River at Kremmling — well-known anchor used in nldi-explorer.
	res, err := c.SnapToComID(ctx, 40.0594, -106.3875)
	if err != nil {
		t.Fatalf("SnapToComID: %v", err)
	}
	if res.ComID == "" {
		t.Fatal("got empty ComID")
	}
	t.Logf("Kremmling snap: ComID=%s Name=%q", res.ComID, res.Name)
}

func TestIntegration_SnapBigSouth(t *testing.T) {
	c := New()
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// Big South Fork of the Poudre, near Long Draw Reservoir.
	res, err := c.SnapToComID(ctx, 40.7012, -105.8124)
	if err != nil {
		t.Fatalf("SnapToComID: %v", err)
	}
	if res.ComID == "" {
		t.Fatal("got empty ComID")
	}
	t.Logf("Big South snap: ComID=%s Name=%q", res.ComID, res.Name)
}

func TestIntegration_UpstreamFlowlinesKremmling(t *testing.T) {
	c := New()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	snap, err := c.SnapToComID(ctx, 40.0594, -106.3875)
	if err != nil {
		t.Fatalf("snap: %v", err)
	}
	coll, err := c.UpstreamFlowlines(ctx, snap.ComID, 100)
	if err != nil {
		t.Fatalf("UpstreamFlowlines: %v", err)
	}
	if len(coll.Features) == 0 {
		t.Fatal("expected upstream flowlines, got none")
	}
	t.Logf("Kremmling upstream: %d flowlines", len(coll.Features))
}

func TestIntegration_MergeMainstemKremmling(t *testing.T) {
	c := New()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	putIn, err := c.SnapToComID(ctx, 40.0594, -106.3875) // Kremmling put-in
	if err != nil {
		t.Fatalf("snap put-in: %v", err)
	}
	takeOut, err := c.SnapToComID(ctx, 39.9500, -106.4200) // downstream point
	if err != nil {
		t.Fatalf("snap take-out: %v", err)
	}

	dm, err := c.DownstreamFlowlines(ctx, putIn.ComID, 50)
	if err != nil {
		t.Fatalf("DownstreamFlowlines: %v", err)
	}

	merged, err := MergeMainstem(dm.Features, takeOut.ComID)
	if err != nil {
		t.Fatalf("MergeMainstem: %v", err)
	}
	if len(merged) < 2 {
		t.Fatalf("merged line too short: %d coords", len(merged))
	}

	geoJSON := ToGeoJSONLineString(merged)
	if len(geoJSON) < 50 {
		t.Fatalf("suspiciously short GeoJSON: %s", geoJSON)
	}
	t.Logf("Merged mainstem: %d coords, GeoJSON length=%d", len(merged), len(geoJSON))
	t.Logf("Preview: %s…", fmt.Sprintf("%.120s", geoJSON))
}
