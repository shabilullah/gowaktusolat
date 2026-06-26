package geo

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"sync"
)

type ZoneResult struct {
	Zone     string `json:"zone"`
	State    string `json:"state"`
	District string `json:"district"`
}

type polygonRecord struct {
	JakimCode string
	State     string
	Name      string
	Polygon   string
}

type zonePolygon struct {
	code     string
	state    string
	district string
	vertices [][2]float64
	bbox     [4]float64
}

type Detector struct {
	polygons []zonePolygon
	mu       sync.RWMutex
}

func NewDetector(database *sql.DB) (*Detector, error) {
	d := &Detector{}
	if err := d.load(database); err != nil {
		return nil, fmt.Errorf("load detector: %w", err)
	}
	return d, nil
}

func (d *Detector) load(database *sql.DB) error {
	rows, err := database.Query("SELECT jakim_code, state, name, polygon FROM zone_polygons")
	if err != nil {
		return fmt.Errorf("query zone_polygons: %w", err)
	}
	defer rows.Close()

	var records []polygonRecord
	for rows.Next() {
		var r polygonRecord
		if err := rows.Scan(&r.JakimCode, &r.State, &r.Name, &r.Polygon); err != nil {
			return fmt.Errorf("scan polygon: %w", err)
		}
		records = append(records, r)
	}

	if len(records) == 0 {
		if err := d.seedFromGeoJSON(database); err != nil {
			return fmt.Errorf("seed polygons: %w", err)
		}
		return d.load(database)
	}

	for _, r := range records {
		zp, err := parsePolygon(r)
		if err != nil {
			continue
		}
		d.polygons = append(d.polygons, zp)
	}

	return nil
}

func (d *Detector) seedFromGeoJSON(database *sql.DB) error {
	url := "https://raw.githubusercontent.com/mptwaktusolat/jakim.geojson/refs/heads/master/malaysia.district-jakim.geojson"
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("fetch GeoJSON: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read GeoJSON: %w", err)
	}

	var fc struct {
		Features []struct {
			Properties struct {
				Name      string  `json:"name"`
				CodeState float64 `json:"code_state"`
				State     string  `json:"state"`
				JakimCode string  `json:"jakim_code"`
			} `json:"properties"`
			Geometry struct {
				Type        string          `json:"type"`
				Coordinates json.RawMessage `json:"coordinates"`
			} `json:"geometry"`
		} `json:"features"`
	}

	if err := json.Unmarshal(body, &fc); err != nil {
		return fmt.Errorf("unmarshal GeoJSON: %w", err)
	}

	tx, err := database.Begin()
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	for _, feat := range fc.Features {
		polygonJSON, err := json.Marshal(feat.Geometry)
		if err != nil {
			continue
		}

		stringID := fmt.Sprintf("%s/%s", feat.Properties.JakimCode, feat.Properties.Name)

		_, err = tx.Exec(
			`INSERT OR REPLACE INTO zone_polygons (string_id, name, code_state, state, jakim_code, polygon)
			 VALUES (?, ?, ?, ?, ?, ?)`,
			stringID,
			feat.Properties.Name,
			int(feat.Properties.CodeState),
			feat.Properties.State,
			feat.Properties.JakimCode,
			string(polygonJSON),
		)
		if err != nil {
			return fmt.Errorf("insert polygon: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit tx: %w", err)
	}

	return nil
}

func parsePolygon(r polygonRecord) (zonePolygon, error) {
	var geom struct {
		Type        string          `json:"type"`
		Coordinates json.RawMessage `json:"coordinates"`
	}
	if err := json.Unmarshal([]byte(r.Polygon), &geom); err != nil {
		return zonePolygon{}, err
	}

	var coords [][][]float64
	if err := json.Unmarshal(geom.Coordinates, &coords); err != nil {
		// Try MultiPolygon
		var multiCoords [][][][]float64
		if err := json.Unmarshal(geom.Coordinates, &multiCoords); err != nil {
			return zonePolygon{}, fmt.Errorf("unmarshal coordinates: %w", err)
		}
		if len(multiCoords) == 0 || len(multiCoords[0]) == 0 {
			return zonePolygon{}, fmt.Errorf("empty multipolygon")
		}
		coords = multiCoords[0]
	}

	if len(coords) == 0 || len(coords[0]) == 0 {
		return zonePolygon{}, fmt.Errorf("empty coordinates")
	}

	ring := coords[0]
	vertices := make([][2]float64, len(ring))
	minLng, minLat := math.MaxFloat64, math.MaxFloat64
	maxLng, maxLat := -math.MaxFloat64, -math.MaxFloat64

	for i, pt := range ring {
		lng, lat := pt[0], pt[1]
		vertices[i] = [2]float64{lng, lat}
		if lng < minLng {
			minLng = lng
		}
		if lat < minLat {
			minLat = lat
		}
		if lng > maxLng {
			maxLng = lng
		}
		if lat > maxLat {
			maxLat = lat
		}
	}

	return zonePolygon{
		code:     r.JakimCode,
		state:    r.State,
		district: r.Name,
		vertices: vertices,
		bbox:     [4]float64{minLng, minLat, maxLng, maxLat},
	}, nil
}

func (d *Detector) DetectZone(lat, lng float64) (*ZoneResult, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	for _, p := range d.polygons {
		if !bboxContains(p.bbox, lng, lat) {
			continue
		}
		if pointInPolygon(p.vertices, lng, lat) {
			return &ZoneResult{
				Zone:     p.code,
				State:    p.state,
				District: p.district,
			}, nil
		}
	}
	return nil, fmt.Errorf("no zone found for the given coordinates")
}

func bboxContains(bbox [4]float64, lng, lat float64) bool {
	return lng >= bbox[0] && lng <= bbox[2] && lat >= bbox[1] && lat <= bbox[3]
}

func pointInPolygon(vertices [][2]float64, px, py float64) bool {
	n := len(vertices)
	if n < 3 {
		return false
	}

	inside := false
	j := n - 1
	for i := 0; i < n; i++ {
		xi, yi := vertices[i][0], vertices[i][1]
		xj, yj := vertices[j][0], vertices[j][1]

		if ((yi > py) != (yj > py)) && (px < (xj-xi)*(py-yi)/(yj-yi)+xi) {
			inside = !inside
		}
		j = i
	}
	return inside
}
