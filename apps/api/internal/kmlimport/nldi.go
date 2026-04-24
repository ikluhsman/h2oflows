package kmlimport

import (
	"context"
	"fmt"

	"github.com/h2oflow/h2oflow/apps/api/internal/nldi"
)

// defaultNLDIDistanceKm bounds how far downstream we'll fetch flowlines from a
// put-in. 500 km covers any whitewater reach in the continental US with margin;
// MergeMainstem stops as soon as it consumes the take-out ComID, so the bound
// mostly protects us from runaway requests when the take-out ComID never matches.
const defaultNLDIDistanceKm = 500

// nldiCenterline is the metadata captured alongside the fetched GeoJSON
// centerline. We persist the ComIDs on the reach so future runs can refresh
// geometry from NLDI without re-snapping.
type nldiCenterline struct {
	GeoJSON      string // LineString from put-in to take-out (untrimmed; PostGIS trims it)
	PutInComID   string
	TakeOutComID string
}

// fetchNLDIRiverLine snaps the put-in + take-out to NHD ComIDs, walks the
// downstream mainstem from the put-in, and returns a GeoJSON LineString that
// terminates at (or includes) the take-out flowline. The caller is expected to
// pass the GeoJSON through PostGIS ST_LineSubstring for exact trimming to the
// put-in/take-out pins — matching how the OSM path works.
func fetchNLDIRiverLine(ctx context.Context, putInLon, putInLat, takeOutLon, takeOutLat float64) (*nldiCenterline, error) {
	return fetchNLDIRiverLineWithClient(ctx, nldi.New(), putInLon, putInLat, takeOutLon, takeOutLat)
}

func fetchNLDIRiverLineWithClient(ctx context.Context, c *nldi.Client, putInLon, putInLat, takeOutLon, takeOutLat float64) (*nldiCenterline, error) {
	putIn, err := c.SnapToComID(ctx, putInLat, putInLon)
	if err != nil {
		return nil, fmt.Errorf("snap put-in: %w", err)
	}
	takeOut, err := c.SnapToComID(ctx, takeOutLat, takeOutLon)
	if err != nil {
		return nil, fmt.Errorf("snap take-out: %w", err)
	}

	coll, err := c.DownstreamFlowlines(ctx, putIn.ComID, defaultNLDIDistanceKm)
	if err != nil {
		return nil, fmt.Errorf("downstream flowlines: %w", err)
	}
	if coll == nil || len(coll.Features) == 0 {
		return nil, fmt.Errorf("no downstream flowlines from ComID %s", putIn.ComID)
	}

	coords, err := nldi.MergeMainstem(coll.Features, takeOut.ComID)
	if err != nil {
		return nil, fmt.Errorf("merge mainstem: %w", err)
	}
	if len(coords) < 2 {
		return nil, fmt.Errorf("merged mainstem too short (%d coords)", len(coords))
	}

	// Sanity check: MergeMainstem breaks once it consumes the take-out flowline.
	// If we never saw that ComID, the take-out wasn't downstream of the put-in
	// (wrong order, different watershed, etc). Return an error rather than
	// silently trimming against an unrelated mainstem tail.
	foundTakeOut := false
	for _, f := range coll.Features {
		if f.Props.NhdplusComID != nil && *f.Props.NhdplusComID == takeOut.ComID {
			foundTakeOut = true
			break
		}
	}
	if !foundTakeOut {
		return nil, fmt.Errorf("take-out ComID %s not downstream of put-in ComID %s within %dkm",
			takeOut.ComID, putIn.ComID, defaultNLDIDistanceKm)
	}

	return &nldiCenterline{
		GeoJSON:      nldi.ToGeoJSONLineString(coords),
		PutInComID:   putIn.ComID,
		TakeOutComID: takeOut.ComID,
	}, nil
}
