package db

import "database/sql"

func InitDB(db *sql.DB) error {
	if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		return err
	}

	statements := []string{
		`CREATE TABLE IF NOT EXISTS prayer_zones (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			jakim_code TEXT NOT NULL UNIQUE,
			negeri TEXT NOT NULL,
			daerah TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS prayer_times (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			date TEXT NOT NULL,
			location_code TEXT NOT NULL,
			hijri TEXT,
			imsak TEXT,
			fajr TEXT,
			syuruk TEXT,
			dhuha TEXT,
			dhuhr TEXT,
			asr TEXT,
			maghrib TEXT,
			isha TEXT,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL
		)`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_prayer_times_zone_date ON prayer_times(location_code, date)`,
		`CREATE TABLE IF NOT EXISTS zone_polygons (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			string_id TEXT NOT NULL UNIQUE,
			name TEXT,
			code_state INTEGER,
			state TEXT,
			jakim_code TEXT,
			polygon TEXT
		)`,
		`CREATE TABLE IF NOT EXISTS settings (
			key TEXT PRIMARY KEY,
			value TEXT NOT NULL,
			updated_at TEXT NOT NULL
		)`,
	}

	for _, stmt := range statements {
		if _, err := db.Exec(stmt); err != nil {
			return err
		}
	}

	return seedDefaultSettings(db)
}

func seedDefaultSettings(db *sql.DB) error {
	var count int
	if err := db.QueryRow("SELECT COUNT(*) FROM settings").Scan(&count); err != nil {
		return err
	}
	if count > 0 {
		return nil
	}

	defaults := map[string]string{
		"scraper.last_run":    "",
		"scraper.last_status": "",
	}

	for k, v := range defaults {
		if _, err := db.Exec(
			"INSERT INTO settings (key, value, updated_at) VALUES (?, ?, datetime('now'))",
			k, v,
		); err != nil {
			return err
		}
	}
	return nil
}
