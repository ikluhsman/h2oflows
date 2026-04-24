//go:build smoke

package kmlimport

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/h2oflow/h2oflow/apps/api/internal/db"
)

// Run with:
//
//	DATABASE_URL=postgres://... SMOKE_SLUG=colorado-gore-canyon \
//	  go test -tags smoke -run TestSmokeSyncCenterlineNLDI -v ./internal/kmlimport/
//
// This test HITS THE LIVE USGS NLDI API and WRITES to the configured database.
// It overwrites reaches.centerline and reaches.centerline_source for the slug
// passed via SMOKE_SLUG. A revert SQL snippet is printed to the test log so
// you can restore the prior OSM-sourced centerline if desired.

func TestSmokeSyncCenterlineNLDI(t *testing.T) {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		t.Skip("DATABASE_URL not set")
	}
	slug := os.Getenv("SMOKE_SLUG")
	if slug == "" {
		t.Skip("SMOKE_SLUG not set")
	}

	ctx := context.Background()
	pool, err := db.Connect(ctx, dbURL)
	if err != nil {
		t.Fatalf("db connect: %v", err)
	}
	defer pool.Close()

	var (
		beforeSource                     *string
		beforePts                        int
		beforeMi                         *float64
		beforePutInCom, beforeTakeOutCom *string
	)
	err = pool.QueryRow(ctx, `
		SELECT centerline_source,
		       COALESCE(ST_NPoints(centerline::geometry), 0),
		       length_mi,
		       put_in_comid,
		       take_out_comid
		FROM reaches WHERE slug=$1`, slug).Scan(
		&beforeSource, &beforePts, &beforeMi, &beforePutInCom, &beforeTakeOutCom)
	if err != nil {
		t.Fatalf("read before: %v", err)
	}
	t.Logf("BEFORE: source=%v pts=%d length_mi=%v put_in_comid=%v take_out_comid=%v",
		deref(beforeSource), beforePts, derefF(beforeMi),
		deref(beforePutInCom), deref(beforeTakeOutCom))

	if err := SyncCenterline(ctx, pool, slug, CenterlineNLDI, false); err != nil {
		t.Fatalf("SyncCenterline(nldi): %v", err)
	}

	var (
		afterSource                    *string
		afterPts                       int
		afterMi                        *float64
		afterPutInCom, afterTakeOutCom *string
		afterReachCode                 *string
		afterTotDA                     *float64
	)
	err = pool.QueryRow(ctx, `
		SELECT centerline_source,
		       COALESCE(ST_NPoints(centerline::geometry), 0),
		       length_mi,
		       put_in_comid,
		       take_out_comid,
		       reachcode,
		       totdasqkm
		FROM reaches WHERE slug=$1`, slug).Scan(
		&afterSource, &afterPts, &afterMi, &afterPutInCom, &afterTakeOutCom,
		&afterReachCode, &afterTotDA)
	if err != nil {
		t.Fatalf("read after: %v", err)
	}
	t.Logf("AFTER:  source=%v pts=%d length_mi=%v put_in_comid=%v take_out_comid=%v reachcode=%v totdasqkm=%v",
		deref(afterSource), afterPts, derefF(afterMi),
		deref(afterPutInCom), deref(afterTakeOutCom),
		deref(afterReachCode), derefF(afterTotDA))

	if afterSource == nil || *afterSource != "nldi" {
		t.Errorf("centerline_source should be 'nldi', got %v", deref(afterSource))
	}
	if afterPutInCom == nil || *afterPutInCom == "" {
		t.Errorf("put_in_comid should be populated")
	}
	if afterTakeOutCom == nil || *afterTakeOutCom == "" {
		t.Errorf("take_out_comid should be populated")
	}
	if afterPts < 10 {
		t.Errorf("centerline suspiciously short (%d points)", afterPts)
	}

	// Revert hint — user can paste this if they want to restore the OSM version.
	t.Logf("If you want to revert, re-run with --centerlines=osm, e.g.:")
	t.Logf("  DATABASE_URL=... SMOKE_SLUG=%s go test -tags smoke -run TestSmokeSyncCenterlineOSM -v ./internal/kmlimport/", slug)
}

// Optional companion — revert via OSM. Run only if needed.
func TestSmokeSyncCenterlineOSM(t *testing.T) {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		t.Skip("DATABASE_URL not set")
	}
	slug := os.Getenv("SMOKE_SLUG")
	if slug == "" {
		t.Skip("SMOKE_SLUG not set")
	}
	ctx := context.Background()
	pool, err := db.Connect(ctx, dbURL)
	if err != nil {
		t.Fatalf("db connect: %v", err)
	}
	defer pool.Close()
	if err := SyncCenterline(ctx, pool, slug, CenterlineOSM, false); err != nil {
		t.Fatalf("SyncCenterline(osm): %v", err)
	}
	t.Logf("Reverted to OSM centerline for %s", slug)
}

func deref(s *string) string {
	if s == nil {
		return "<nil>"
	}
	return *s
}
func derefF(f *float64) string {
	if f == nil {
		return "<nil>"
	}
	return fmt.Sprintf("%.3f", *f)
}
