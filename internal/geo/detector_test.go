package geo

import (
	"testing"
)

func TestBBoxContains(t *testing.T) {
	bbox := [4]float64{100.0, 1.0, 104.0, 6.0}

	tests := []struct {
		name   string
		lng    float64
		lat    float64
		expect bool
	}{
		{"inside center", 102.0, 3.5, true},
		{"on left edge", 100.0, 3.5, true},
		{"on bottom edge", 102.0, 1.0, true},
		{"on top-right corner", 104.0, 6.0, true},
		{"left of bbox", 99.0, 3.5, false},
		{"right of bbox", 105.0, 3.5, false},
		{"below bbox", 102.0, 0.5, false},
		{"above bbox", 102.0, 7.0, false},
		{"diagonal outside", 99.0, 7.0, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := bboxContains(bbox, tt.lng, tt.lat)
			if got != tt.expect {
				t.Errorf("bboxContains(%v, %v, %v) = %v, want %v", bbox, tt.lng, tt.lat, got, tt.expect)
			}
		})
	}
}

func TestPointInPolygon(t *testing.T) {
	// Square: (0,0)-(2,0)-(2,2)-(0,2)
	square := [][2]float64{{0, 0}, {2, 0}, {2, 2}, {0, 2}}

	// Triangle: (0,0)-(4,0)-(2,4)
	triangle := [][2]float64{{0, 0}, {4, 0}, {2, 4}}

	// L-shaped polygon (concave)
	lshape := [][2]float64{{0, 0}, {3, 0}, {3, 1}, {1, 1}, {1, 3}, {0, 3}}

	tests := []struct {
		name     string
		polygon  [][2]float64
		px, py   float64
		expected bool
	}{
		// Square tests
		{"square inside", square, 1.0, 1.0, true},
		{"square near vertex", square, 1.999, 1.999, true},
		{"square on edge", square, 1.0, 0.0, true},
		{"square outside right", square, 3.0, 1.0, false},
		{"square outside top", square, 1.0, 3.0, false},
		{"square outside diagonal", square, 3.0, 3.0, false},

		// Triangle tests
		{"triangle inside", triangle, 2.0, 1.0, true},
		{"triangle near center", triangle, 2.0, 2.0, true},
		{"triangle outside", triangle, 0.0, 3.0, false},
		{"triangle far outside", triangle, 5.0, 5.0, false},

		// L-shape tests (concave polygon)
		{"lshape inside body", lshape, 0.5, 0.5, true},
		{"lshape inside arm", lshape, 0.5, 2.0, true},
		{"lshape in notch", lshape, 2.0, 2.0, false}, // the missing corner
		{"lshape outside", lshape, 2.5, 2.5, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := pointInPolygon(tt.polygon, tt.px, tt.py)
			if got != tt.expected {
				t.Errorf("pointInPolygon(..., %v, %v) = %v, want %v", tt.px, tt.py, got, tt.expected)
			}
		})
	}
}

func TestPointInPolygonEdgeCases(t *testing.T) {
	// Too few vertices
	if pointInPolygon([][2]float64{{0, 0}, {1, 1}}, 0.5, 0.5) {
		t.Error("polygon with < 3 vertices should never contain a point")
	}

	// Empty polygon
	if pointInPolygon(nil, 0, 0) {
		t.Error("nil polygon should never contain a point")
	}
}

func TestParsePolygonMultiPolygon(t *testing.T) {
	// GeoJSON MultiPolygon format as stored in DB
	multipolyJSON := `{
		"type": "MultiPolygon",
		"coordinates": [[[[101.0, 3.0], [102.0, 3.0], [102.0, 4.0], [101.0, 4.0], [101.0, 3.0]]]]
	}`

	r := polygonRecord{
		JakimCode: "TST01",
		State:     "TEST",
		Name:      "TestArea",
		Polygon:   multipolyJSON,
	}

	zp, err := parsePolygon(r)
	if err != nil {
		t.Fatalf("parsePolygon failed: %v", err)
	}

	if zp.code != "TST01" {
		t.Errorf("code = %s, want TST01", zp.code)
	}
	if zp.state != "TEST" {
		t.Errorf("state = %s, want TEST", zp.state)
	}
	if zp.district != "TestArea" {
		t.Errorf("district = %s, want TestArea", zp.district)
	}
	if len(zp.vertices) != 5 {
		t.Errorf("vertices len = %d, want 5", len(zp.vertices))
	}

	// Point inside the square (101.5, 3.5) should be inside
	if !pointInPolygon(zp.vertices, 101.5, 3.5) {
		t.Error("point in center of square should be inside")
	}
	// Point outside (100.0, 3.5) should be outside
	if pointInPolygon(zp.vertices, 100.0, 3.5) {
		t.Error("point far left should be outside")
	}
}
