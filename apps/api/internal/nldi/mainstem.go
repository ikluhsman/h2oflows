package nldi

import (
	"encoding/json"
	"fmt"
	"strings"
)

// Coord is [lng, lat] (GeoJSON order).
type Coord [2]float64

// MergeMainstem concatenates ordered downstream flowline features into one
// continuous line, stopping once the flowline whose nhdplus_comid equals
// targetComID has been consumed. If targetComID is empty, all features are
// used. Duplicate shared endpoints between consecutive reaches are skipped.
//
// NLDI returns DM flowlines in mainstem order, so a simple concat works as
// long as we drop the first node of each subsequent feature (which equals
// the last node of the previous feature for a connected NHD chain).
func MergeMainstem(features []Feature, targetComID string) ([]Coord, error) {
	out := make([]Coord, 0, 128)
	for _, f := range features {
		coords, err := flowlineCoords(f)
		if err != nil {
			return nil, err
		}
		if len(coords) == 0 {
			continue
		}
		if len(out) == 0 {
			out = append(out, coords...)
		} else {
			out = append(out, coords[1:]...)
		}
		if targetComID != "" && f.Props.NhdplusComID != nil &&
			fmt.Sprintf("%d", *f.Props.NhdplusComID) == targetComID {
			break
		}
	}
	return out, nil
}

// ToGeoJSONLineString renders a [lng,lat] slice as a GeoJSON LineString string,
// matching the format the existing kmlimport/osm package uses for PostGIS ingest.
func ToGeoJSONLineString(coords []Coord) string {
	parts := make([]string, len(coords))
	for i, c := range coords {
		parts[i] = fmt.Sprintf("[%.7f,%.7f]", c[0], c[1])
	}
	return fmt.Sprintf(`{"type":"LineString","coordinates":[%s]}`, strings.Join(parts, ","))
}

// flowlineCoords pulls the coordinates out of a flowline feature's geometry.
// NLDI returns LineString for individual flowlines; MultiLineString occasionally
// appears when the source NHD reach was multi-part.
func flowlineCoords(f Feature) ([]Coord, error) {
	raw, err := json.Marshal(f.Geometry.Coordinates)
	if err != nil {
		return nil, err
	}
	switch f.Geometry.Type {
	case "LineString":
		var pts [][]float64
		if err := json.Unmarshal(raw, &pts); err != nil {
			return nil, fmt.Errorf("nldi: parse LineString: %w", err)
		}
		return toCoords(pts), nil
	case "MultiLineString":
		var parts [][][]float64
		if err := json.Unmarshal(raw, &parts); err != nil {
			return nil, fmt.Errorf("nldi: parse MultiLineString: %w", err)
		}
		out := make([]Coord, 0, 64)
		for i, part := range parts {
			if i == 0 {
				out = append(out, toCoords(part)...)
			} else {
				out = append(out, toCoords(part)[1:]...)
			}
		}
		return out, nil
	default:
		return nil, fmt.Errorf("nldi: unexpected flowline geometry %q", f.Geometry.Type)
	}
}

func toCoords(pts [][]float64) []Coord {
	out := make([]Coord, 0, len(pts))
	for _, p := range pts {
		if len(p) < 2 {
			continue
		}
		out = append(out, Coord{p[0], p[1]})
	}
	return out
}
