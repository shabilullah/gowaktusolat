package geo

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"sync"

	"zombiezen.com/go/sqlite"
	"zombiezen.com/go/sqlite/sqlitex"
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
	pool     *sqlitex.Pool
	polygons []zonePolygon
	mu       sync.RWMutex
}

func NewDetector(pool *sqlitex.Pool) (*Detector, error) {
	d := &Detector{pool: pool}
	if err := d.load(); err != nil {
		return nil, fmt.Errorf("load detector: %w", err)
	}
	return d, nil
}

func (d *Detector) load() error {
	conn, err := d.pool.Take(context.Background())
	if err != nil {
		return fmt.Errorf("take conn: %w", err)
	}
	defer d.pool.Put(conn)

	var records []polygonRecord
	err = sqlitex.Execute(conn, "SELECT jakim_code, state, name, polygon FROM zone_polygons", &sqlitex.ExecOptions{
		ResultFunc: func(stmt *sqlite.Stmt) error {
			records = append(records, polygonRecord{
				JakimCode: stmt.ColumnText(0),
				State:     stmt.ColumnText(1),
				Name:      stmt.ColumnText(2),
				Polygon:   stmt.ColumnText(3),
			})
			return nil
		},
	})
	if err != nil {
		return fmt.Errorf("query zone_polygons: %w", err)
	}

	if len(records) == 0 {
		if err := d.seedFromGeoJSON(); err != nil {
			return fmt.Errorf("seed polygons: %w", err)
		}
		return d.load()
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
func (d *Detector) seedFromGeoJSON() error {
	conn, err := d.pool.Take(context.Background())
	if err != nil {
		return fmt.Errorf("take conn: %w", err)
	}
	defer d.pool.Put(conn)

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

	if err := sqlitex.Execute(conn, "BEGIN IMMEDIATE", nil); err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}

	for _, feat := range fc.Features {
		polygonJSON, err := json.Marshal(feat.Geometry)
		if err != nil {
			continue
		}

		stringID := fmt.Sprintf("%s/%s", feat.Properties.JakimCode, feat.Properties.Name)

		if err := sqlitex.Execute(conn,
			`INSERT OR REPLACE INTO zone_polygons (string_id, name, code_state, state, jakim_code, polygon)
			 VALUES (?, ?, ?, ?, ?, ?)`,
			&sqlitex.ExecOptions{
				Args: []interface{}{
					stringID,
					feat.Properties.Name,
					int(feat.Properties.CodeState),
					feat.Properties.State,
					feat.Properties.JakimCode,
					string(polygonJSON),
				},
			},
		); err != nil {
			sqlitex.Execute(conn, "ROLLBACK", nil)
			return fmt.Errorf("insert polygon: %w", err)
		}
	}

	if err := sqlitex.Execute(conn, "COMMIT", nil); err != nil {
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
