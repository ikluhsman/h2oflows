// Package gauge provides a common interface for fetching streamflow data
// from multiple source adapters (USGS, DWR, CDEC, etc.) and discovering
// available gauge sites.
package gauge

import (
	"context"
	"errors"
	"time"
)

// SourceType identifies which upstream data provider a gauge belongs to.
type SourceType string

const (
	SourceUSGS      SourceType = "usgs"
	SourceDWR       SourceType = "dwr"
	SourceCDEC      SourceType = "cdec"
	SourceManual    SourceType = "manual"
	SourceCommunity SourceType = "community"
)

// GaugeStatus represents the lifecycle state of a gauge in the database.
// Adapters return this via SiteMetadata; the poller and sync job use it
// to decide whether to poll and whether to fire alerts.
type GaugeStatus string

const (
	// StatusActive — reporting normally, poll on schedule, full alert coverage.
	StatusActive GaugeStatus = "active"

	// StatusSeasonal — expected to go offline outside its season window.
	// Poll normally; suppress "offline" alerts when outside seasonal_start/end.
	StatusSeasonal GaugeStatus = "seasonal"

	// StatusInactive — not currently reporting; do not poll.
	// Exists in the database for reach binding and historical queries.
	StatusInactive GaugeStatus = "inactive"

	// StatusRetired — decommissioned upstream. Do not poll.
	// May have a successor gauge; UI follows successor_id if set.
	StatusRetired GaugeStatus = "retired"

	// StatusMaintenance — temporarily offline, expected to return.
	// Poll but suppress alerts for the maintenance window.
	StatusMaintenance GaugeStatus = "maintenance"
)

// Sentinel errors returned by adapters. Callers can distinguish between
// "gauge doesn't exist", "gauge exists but has no data", and "source is down".
var (
	ErrGaugeNotFound    = errors.New("gauge not found")
	ErrNoReadings       = errors.New("no readings available")
	ErrSourceUnavailable = errors.New("gauge source unavailable")
)

// LatLng is a WGS84 coordinate pair.
type LatLng struct {
	Lat float64
	Lng float64
}

// BoundingBox is a WGS84 bounding box used for spatial filtering.
// Follows the GeoJSON convention: west, south, east, north.
type BoundingBox struct {
	West  float64
	South float64
	East  float64
	North float64
}

// Reading is a single instantaneous measurement from a gauge.
type Reading struct {
	ExternalID  string
	Value       float64
	Unit        string     // "cfs", "ft", "m3/s"
	Timestamp   time.Time
	QualCode    string     // source-specific quality code, e.g. "A" (approved), "P" (provisional)
	Provisional bool       // true if data has not yet been reviewed by the source agency
}

// SiteMetadata describes a gauge site as returned by a source's discovery
// endpoint. Used to seed and sync the gauges table.
type SiteMetadata struct {
	ExternalID       string
	Name             string
	Location         *LatLng
	StateCode        string   // two-letter, e.g. "CO"
	CountyCode       string
	HUCCode          string   // 8-digit hydrologic unit code (watershed)
	Parameters       []string // available parameter codes, e.g. ["00060", "00065"]
	Active           bool
	BeginDate        time.Time
	EndDate          *time.Time // nil = still active; non-nil = retired upstream
	DrainageAreaSqMi float64
	SourceType       SourceType
}

// GaugeSource is the core interface every adapter must implement.
// FetchReading returns the most recent available reading.
// FetchHistory returns all readings since the given time, oldest first.
type GaugeSource interface {
	FetchReading(ctx context.Context, externalID string) (*Reading, error)
	FetchHistory(ctx context.Context, externalID string, since time.Time) ([]*Reading, error)
	Name() string
	SourceType() SourceType
}

// SiteDiscoverer is an optional interface for sources that support bulk
// site enumeration. Implement it alongside GaugeSource when the upstream
// API can return site lists (USGS OGC API, DWR station list, etc.).
//
// The poller and sync job check for this interface at runtime:
//
//	if discoverer, ok := source.(SiteDiscoverer); ok {
//	    sites, err := discoverer.DiscoverSites(ctx, opts)
//	}
type SiteDiscoverer interface {
	DiscoverSites(ctx context.Context, opts DiscoverOptions) ([]*SiteMetadata, error)
}

// DiscoverOptions controls which sites a SiteDiscoverer returns.
// All fields are optional — zero value returns everything the source has.
type DiscoverOptions struct {
	// StateCodes filters by US state. Empty slice means all states.
	// Not all sources support this (DWR is Colorado-only regardless).
	StateCodes []string

	// Parameters filters to sites that measure at least one of these
	// parameter codes. Defaults to ["00060"] (CFS discharge) if nil.
	Parameters []string

	// ActiveOnly excludes retired and discontinued sites when true.
	ActiveOnly bool

	// BoundingBox spatially filters results. Nil means no spatial filter.
	// For sources that don't support server-side bbox filtering, adapters
	// should filter the results themselves before returning.
	BoundingBox *BoundingBox
}
