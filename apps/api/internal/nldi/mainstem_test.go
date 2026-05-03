package nldi

import (
	"encoding/json"
	"fmt"
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

func TestSampleDownstreamComIDs_empty(t *testing.T) {
	if got := SampleDownstreamComIDs(nil, 20, 30); got != nil {
		t.Errorf("expected nil, got %v", got)
	}
}

func TestSampleDownstreamComIDs_singleFeature(t *testing.T) {
	// one short feature — only the put-in comid should be emitted
	f := makeLineStringFeature("AAA", [][]float64{{-106.0, 39.0}, {-106.001, 39.0}})
	out := SampleDownstreamComIDs([]Feature{f}, 20, 30)
	if len(out) != 1 || out[0] != "AAA" {
		t.Errorf("expected [AAA], got %v", out)
	}
}

func TestSampleDownstreamComIDs_nilComIDSkipped(t *testing.T) {
	noComID := Feature{
		Geometry: Geometry{
			Type:        "LineString",
			Coordinates: func() json.RawMessage { b, _ := json.Marshal([][]float64{{-106.0, 39.0}, {-106.001, 39.0}}); return b }(),
		},
	}
	out := SampleDownstreamComIDs([]Feature{noComID}, 20, 30)
	if len(out) != 0 {
		t.Errorf("expected no comids for nil NhdplusComID, got %v", out)
	}
}

func TestSampleDownstreamComIDs_spacingRespected(t *testing.T) {
	// build a chain of 5 features each ~20 km long (1 degree lat ≈ 111 km → ~0.18 deg = 20 km)
	step := 0.18 // ≈ 20 km
	features := make([]Feature, 5)
	for i := 0; i < 5; i++ {
		lat := 39.0 + float64(i)*step
		features[i] = makeLineStringFeature(
			fmt.Sprintf("%03d", i+1),
			[][]float64{{-106.0, lat}, {-106.0, lat + step}},
		)
	}
	// spacing 20 km, max 10 — expect one sample per feature (5 total)
	out := SampleDownstreamComIDs(features, 20, 10)
	if len(out) == 0 {
		t.Fatal("expected at least 1 sampled comid")
	}
	if len(out) > 6 {
		t.Errorf("expected roughly 5 samples, got %d: %v", len(out), out)
	}
}

func TestSampleDownstreamComIDs_maxAnchorsRespected(t *testing.T) {
	step := 0.18
	features := make([]Feature, 20)
	for i := 0; i < 20; i++ {
		lat := 39.0 + float64(i)*step
		features[i] = makeLineStringFeature(
			fmt.Sprintf("%03d", i+1),
			[][]float64{{-106.0, lat}, {-106.0, lat + step}},
		)
	}
	out := SampleDownstreamComIDs(features, 20, 5)
	if len(out) > 5 {
		t.Errorf("maxAnchors not respected: got %d", len(out))
	}
}

func TestToGeoJSONLineString_empty(t *testing.T) {
	got := ToGeoJSONLineString(nil)
	want := `{"type":"LineString","coordinates":[]}`
	if got != want {
		t.Errorf("empty coords: got %s", got)
	}
}
