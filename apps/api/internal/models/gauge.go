package models

import "time"

// Gauge mirrors the gauges table. Geography columns are stored as lat/lng
// pairs since pgx scans PostGIS points via ST_X/ST_Y in queries.
type Gauge struct {
	ID                  string
	ReachID             *string
	ExternalID          string
	Source              string  // usgs / dwr / cdec / manual / community
	Name                *string
	LocationLat         *float64
	LocationLng         *float64
	ParamCode           string
	// Lifecycle
	Status              string  // active / seasonal / inactive / retired / maintenance
	SeasonalStartMMDD   *string // "MM-DD"
	SeasonalEndMMDD     *string // "MM-DD"
	SuccessorID         *string
	LastReadingAt       *time.Time
	ConsecutiveFailures int
	AutoManaged         bool
	Notes               *string
	// Prominence
	Featured        bool
	ProminenceScore float64
	// NHD/NLDI reference (nullable — populated for USGS gauges once NLDI lookup runs).
	ComID     *string
	CreatedAt time.Time
}

// GaugeReading mirrors the gauge_readings table.
type GaugeReading struct {
	ID          string
	GaugeID     string
	Value       float64
	Unit        string
	Timestamp   time.Time
	QualCode    *string
	Provisional bool
	CreatedAt   time.Time
}

// FlowRange mirrors the flow_ranges table.
type FlowRange struct {
	ID            string
	GaugeID       string
	Label         string  // too_low / minimum / fun / optimal / pushy / high / flood
	MinCFS        *float64
	MaxCFS        *float64
	ClassModifier *float64
	CreatedAt     time.Time
}
