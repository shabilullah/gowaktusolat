package scraper

import (
	"database/sql"
	_ "embed"
	"encoding/json"
	"fmt"
)

//go:embed zones_data.json
var zonesJSON []byte

type Zone struct {
	JakimCode string `json:"jakim_code"`
	Negeri    string `json:"negeri"`
	Daerah    string `json:"daerah"`
}

func LoadZones() ([]Zone, error) {
	var zones []Zone
	if err := json.Unmarshal(zonesJSON, &zones); err != nil {
		return nil, fmt.Errorf("unmarshal zones: %w", err)
	}
	return zones, nil
}

func SeedZones(db *sql.DB) error {
	zones, err := LoadZones()
	if err != nil {
		return err
	}

	for _, z := range zones {
		if _, err := db.Exec(
			"INSERT OR REPLACE INTO prayer_zones (jakim_code, negeri, daerah) VALUES (?, ?, ?)",
			z.JakimCode, z.Negeri, z.Daerah,
		); err != nil {
			return fmt.Errorf("seed zone %s: %w", z.JakimCode, err)
		}
	}
	return nil
}
