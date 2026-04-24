package kmlimport

import (
	"context"
	"fmt"

	"github.com/h2oflow/h2oflow/apps/api/internal/nldi"
	"github.com/jackc/pgx/v5/pgxpool"
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

// SnapReachComIDs snaps the reach's existing put-in/take-out access points to
// NHD ComIDs and stores them. It only writes when put_in_comid is NULL, so it
// won't overwrite data set by the NLDI centerline path. Designed to be called
// in a background goroutine after a successful OSM centerline fetch.
func SnapReachComIDs(ctx context.Context, pool *pgxpool.Pool, slug string) error {
	var putInLon, putInLat, takeOutLon, takeOutLat float64
	err := pool.QueryRow(ctx, `
		SELECT
		  ST_X(a_in.location::geometry),  ST_Y(a_in.location::geometry),
		  ST_X(a_out.location::geometry), ST_Y(a_out.location::geometry)
		FROM reaches r
		JOIN reach_access a_in  ON a_in.reach_id  = r.id AND a_in.access_type  = 'put_in'
		JOIN reach_access a_out ON a_out.reach_id = r.id AND a_out.access_type = 'take_out'
		WHERE r.slug = $1
		ORDER BY a_in.created_at ASC, a_out.created_at ASC
		LIMIT 1
	`, slug).Scan(&putInLon, &putInLat, &takeOutLon, &takeOutLat)
	if err != nil {
		return fmt.Errorf("no access points for %q: %w", slug, err)
	}

	c := nldi.New()
	putInSnap, err := c.SnapToComID(ctx, putInLat, putInLon)
	if err != nil {
		return nil // best-effort
	}
	takeOutSnap, err := c.SnapToComID(ctx, takeOutLat, takeOutLon)
	if err != nil {
		return nil
	}

	_, err = pool.Exec(ctx, `
		UPDATE reaches
		SET    put_in_comid   = $2,
		       take_out_comid = $3,
		       anchor_comid   = COALESCE(anchor_comid, $2)
		WHERE  slug = $1
		  AND  put_in_comid IS NULL
	`, slug, putInSnap.ComID, takeOutSnap.ComID)
	return err
}
