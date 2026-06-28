package geo

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"zombiezen.com/go/sqlite/sqlitex"
)

// SeedFromGeoJSON fetches the Malaysia district GeoJSON from GitHub and
// populates the zone_polygons table. Run once before creating a Detector.
func SeedFromGeoJSON(pool *sqlitex.Pool) error {
	conn, err := pool.Take(context.Background())
	if err != nil {
		return fmt.Errorf("take conn: %w", err)
	}
	defer pool.Put(conn)

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
