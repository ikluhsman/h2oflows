package kmlimport

import (
	"context"

	"github.com/h2oflow/h2oflow/apps/api/internal/osm"
)

func fetchOSMRiverLine(ctx context.Context, minLon, minLat, maxLon, maxLat float64) (string, error) {
	return osm.FetchRiverLine(ctx, minLon, minLat, maxLon, maxLat)
}
