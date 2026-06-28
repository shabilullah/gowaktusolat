package db

import (
	"context"

	"zombiezen.com/go/sqlite"
	"zombiezen.com/go/sqlite/sqlitex"
)

func InitPool(pool *sqlitex.Pool) error {
	conn, err := pool.Take(context.Background())
	if err != nil {
		return err
	}
	defer pool.Put(conn)

	if err := sqlitex.Execute(conn, "PRAGMA journal_mode=WAL", nil); err != nil {
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
		if err := sqlitex.Execute(conn, stmt, nil); err != nil {
			return err
		}
	}

	return seedDefaultSettings(conn)
}

func seedDefaultSettings(conn *sqlite.Conn) error {
	var count int
	if err := sqlitex.Execute(conn, "SELECT COUNT(*) FROM settings", &sqlitex.ExecOptions{
		ResultFunc: func(stmt *sqlite.Stmt) error {
			count = stmt.ColumnInt(0)
			return nil
		},
	}); err != nil {
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
		if err := sqlitex.Execute(conn,
			"INSERT INTO settings (key, value, updated_at) VALUES (?, ?, datetime('now'))",
			&sqlitex.ExecOptions{
				Args: []interface{}{k, v},
			},
		); err != nil {
			return err
		}
	}
	return nil
}
