package api

import (
	"database/sql"
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v3"
	_ "modernc.org/sqlite"
)

func setupLastUpdateTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}

	if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		t.Fatalf("wal: %v", err)
	}

	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS settings (
		key TEXT PRIMARY KEY,
		value TEXT NOT NULL,
		updated_at TEXT NOT NULL
	)`); err != nil {
		t.Fatalf("create table: %v", err)
	}

	defaults := map[string]string{
		"scraper.last_run":    "2026-06-27T12:00:00Z",
		"scraper.last_status": "success: 53/53 zones",
	}
	for k, v := range defaults {
		if _, err := db.Exec("INSERT INTO settings (key, value, updated_at) VALUES (?, ?, datetime('now'))", k, v); err != nil {
			t.Fatalf("seed %s: %v", k, err)
		}
	}

	return db
}

func TestLastUpdateGet(t *testing.T) {
	db := setupLastUpdateTestDB(t)
	defer db.Close()

	handler := &LastUpdate{DB: db}

	app := fiber.New()
	app.Get("/api/last-update", handler.Get)

	req := httptest.NewRequest("GET", "/api/last-update", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	if resp.StatusCode != 200 {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var body lastUpdateResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}

	if body.LastRun != "2026-06-27T12:00:00Z" {
		t.Errorf("last_run = %q, want %q", body.LastRun, "2026-06-27T12:00:00Z")
	}
	if body.LastStatus != "success: 53/53 zones" {
		t.Errorf("last_status = %q, want %q", body.LastStatus, "success: 53/53 zones")
	}
}

func TestLastUpdateGetEmpty(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()

	if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		t.Fatalf("wal: %v", err)
	}
	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS settings (
		key TEXT PRIMARY KEY,
		value TEXT NOT NULL,
		updated_at TEXT NOT NULL
	)`); err != nil {
		t.Fatalf("create table: %v", err)
	}

	handler := &LastUpdate{DB: db}

	app := fiber.New()
	app.Get("/api/last-update", handler.Get)

	req := httptest.NewRequest("GET", "/api/last-update", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	if resp.StatusCode != 200 {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var body lastUpdateResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}

	if body.LastRun != "" {
		t.Errorf("last_run = %q, want empty", body.LastRun)
	}
	if body.LastStatus != "" {
		t.Errorf("last_status = %q, want empty", body.LastStatus)
	}
}
