// Package nldi is a thin client for USGS NLDI (Network-Linked Data Index).
// NLDI exposes NHDPlus flowlines, upstream/downstream navigation, and gauge
// discovery anchored on a ComID. h2oflows uses it to:
//   - snap a (lat, lng) to the nearest NHD reach (ComID)
//   - fetch mainstem geometry between two ComIDs for reach centerlines
//   - enumerate upstream gauges for a reach
//   - traverse the dendritic tree for the admin reach-authoring tool
//
// Docs: https://api.water.usgs.gov/nldi/
package nldi

// Feature is a GeoJSON feature as returned by NLDI endpoints.
// Geometry coordinates are always [lng, lat] pairs (GeoJSON order).
// The union on Coordinates mirrors the API: Point geometries return []float64,
// LineString returns [][]float64, MultiLineString returns [][][]float64.
type Feature struct {
	Type     string          `json:"type"`
	Geometry Geometry        `json:"geometry"`
	Props    FeatureProps    `json:"properties"`
}

type Geometry struct {
	Type        string `json:"type"`
	Coordinates any    `json:"coordinates"`
}

// FeatureProps captures the subset of NLDI feature properties h2oflows reads.
// NLDI returns additional fields we ignore (sourcename, mainstem, etc).
type FeatureProps struct {
	Identifier   string   `json:"identifier"`    // e.g. "USGS-09058000" for gauges, "14837340" for flowlines
	Name         string   `json:"name"`          // gauge name or flowline GNIS name
	NhdplusComID *string  `json:"nhdplus_comid"` // set on flowline features; NLDI returns as a JSON string
	ComID        *string  `json:"comid"`         // set on gauge features — the reach the gauge sits on
	ReachCode    *string  `json:"reachcode"`
	GnisName     *string  `json:"gnis_name"`
	TotDASqKm    *float64 `json:"totdasqkm"`
}

type Collection struct {
	Type     string    `json:"type"`
	Features []Feature `json:"features"`
}

// SnapResult is the resolved ComID for a (lat, lng). The comid/position
// endpoint returns only identifier/comid/navigation-link — name, reachcode,
// and totdasqkm are NOT populated by this endpoint (they live on the NHDPlus
// characteristics service, which we don't query yet).
type SnapResult struct {
	ComID string
	Name  string // usually empty from comid/position
}
