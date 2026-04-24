package models

import "time"

// Reach mirrors the reaches table.
type Reach struct {
	ID             string
	Slug           string
	Name           string
	PutInLat       *float64
	PutInLng       *float64
	TakeOutLat     *float64
	TakeOutLng     *float64
	ClassMin       *float64
	ClassMax       *float64
	ClassAtLow     *float64
	ClassAtHigh    *float64
	Character      *string // creeking/pool-drop/continuous/big-water/flatwater
	LengthMi       *float64
	Region         *string
	PrimaryGaugeID *string
	// NHD/NLDI reference layer (nullable — not required for boatable-reach definition).
	AnchorComID   *string
	PutInComID    *string
	TakeOutComID  *string
	ReachCode     *string
	TotDASqKm     *float64
	CreatedAt     time.Time
}

// ReachCondition mirrors the reach_conditions table.
type ReachCondition struct {
	ID           string
	ReachID      string
	SourceType   string  // gauge / personal / word-of-mouth / discord / outfitter
	Summary      string
	Runnable     *bool
	ReportedBy   *string
	CFSAtReport  *float64
	ExpiresAt    time.Time
	CreatedAt    time.Time
}

// Hazard mirrors the hazards table.
type Hazard struct {
	ID           string
	ReachID      string
	LocationLat  *float64
	LocationLng  *float64
	HazardType   string  // strainer / sieve / undercut / low-head-dam / other
	Description  string
	CFSAtReport  *float64
	ReportedBy   *string
	Active       bool
	CreatedAt    time.Time
}
