package nldi

import (
	"encoding/json"
	"testing"
)

func makeLineStringFeature(comid string, coords [][]float64) Feature {
	raw, _ := json.Marshal(coords)
	return Feature{
		Geometry: Geometry{
			Type:        "LineString",
			Coordinates: json.RawMessage(raw),
		},
		Props: FeatureProps{NhdplusComID: &comid},
	}
}

func makeMultiLineStringFeature(comid string, parts [][][]float64) Feature {
	raw, _ := json.Marshal(parts)
	return Feature{
		Geometry: Geometry{
			Type:        "MultiLineString",
			Coordinates: json.RawMessage(raw),
		},
		Props: FeatureProps{NhdplusComID: &comid},
	}
}

func TestMergeMainstem_empty(t *testing.T) {
	out, err := MergeMainstem(nil, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 0 {
		t.Errorf("expected empty, got %d coords", len(out))
	}
}

func TestMergeMainstem_singleFeature(t *testing.T) {
	coords := [][]float64{{-106.0, 40.0}, {-106.1, 40.1}, {-106.2, 40.2}}
	f := makeLineStringFeature("100", coords)
	out, err := MergeMainstem([]Feature{f}, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 3 {
		t.Errorf("expected 3 coords, got %d", len(out))
	}
	if out[0] != (Coord{-106.0, 40.0}) {
		t.Errorf("unexpected first coord %v", out[0])
	}
}

func TestMergeMainstem_consecutiveDuplicateDropped(t *testing.T) {
	// NHD flowlines chain: end of f1 == start of f2.
	// MergeMainstem must skip that shared endpoint to avoid a duplicate node.
	f1 := makeLineStringFeature("100", [][]float64{{-106.0, 40.0}, {-106.1, 40.1}})
	f2 := makeLineStringFeature("200", [][]float64{{-106.1, 40.1}, {-106.2, 40.2}, {-106.3, 40.3}})
	out, err := MergeMainstem([]Feature{f1, f2}, "")
	if err != nil {
		t.Fatal(err)
	}
	// f1 contributes 2; f2 contributes 2 (3 - 1 skipped shared endpoint)
	if len(out) != 4 {
		t.Errorf("expected 4 coords, got %d: %v", len(out), out)
	}
	if out[1] != (Coord{-106.1, 40.1}) {
		t.Errorf("expected shared endpoint once at index 1, got %v", out[1])
	}
}

func TestMergeMainstem_stopsAtTargetComID(t *testing.T) {
	f1 := makeLineStringFeature("100", [][]float64{{-106.0, 40.0}, {-106.1, 40.1}})
	f2 := makeLineStringFeature("200", [][]float64{{-106.1, 40.1}, {-106.2, 40.2}})
	f3 := makeLineStringFeature("300", [][]float64{{-106.2, 40.2}, {-106.3, 40.3}})

	out, err := MergeMainstem([]Feature{f1, f2, f3}, "200")
	if err != nil {
		t.Fatal(err)
	}
	// Should stop after consuming f2 (ComID "200"). f3 excluded.
	// f1: 2 coords; f2: 1 (shared dropped) => 3 total
	if len(out) != 3 {
		t.Errorf("expected 3 coords (stopped at 200), got %d: %v", len(out), out)
	}
}

func TestMergeMainstem_multiLineString(t *testing.T) {
	parts := [][][]float64{
		{{-106.0, 40.0}, {-106.05, 40.05}},
		{{-106.05, 40.05}, {-106.1, 40.1}},
	}
	f := makeMultiLineStringFeature("100", parts)
	out, err := MergeMainstem([]Feature{f}, "")
	if err != nil {
		t.Fatal(err)
	}
	// part[0]: 2 coords; part[1]: 1 (shared dropped internally)
	if len(out) != 3 {
		t.Errorf("expected 3 coords from MultiLineString, got %d: %v", len(out), out)
	}
}

func TestToGeoJSONLineString(t *testing.T) {
	coords := []Coord{{-106.1234567, 40.1234567}, {-106.7654321, 40.7654321}}
	got := ToGeoJSONLineString(coords)
	want := `{"type":"LineString","coordinates":[[-106.1234567,40.1234567],[-106.7654321,40.7654321]]}`
	if got != want {
		t.Errorf("ToGeoJSONLineString:\ngot:  %s\nwant: %s", got, want)
	}
}

func TestToGeoJSONLineString_empty(t *testing.T) {
	got := ToGeoJSONLineString(nil)
	want := `{"type":"LineString","coordinates":[]}`
	if got != want {
		t.Errorf("empty coords: got %s", got)
	}
}
