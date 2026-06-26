package api

import (
	"database/sql"
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v3"
	_ "modernc.org/sqlite"
)

func setupSettingsTestDB(t *testing.T) *sql.DB {
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
		"scraper.schedule":    "0 2 1 1 *",
		"scraper.enabled":     "false",
		"scraper.last_run":    "",
		"scraper.last_status": "",
	}
	for k, v := range defaults {
		if _, err := db.Exec("INSERT INTO settings (key, value, updated_at) VALUES (?, ?, datetime('now'))", k, v); err != nil {
			t.Fatalf("seed %s: %v", k, err)
		}
	}

	return db
}

func TestSettingsGet(t *testing.T) {
	db := setupSettingsTestDB(t)
	defer db.Close()

	handler := &Settings{DB: db}

	app := fiber.New()
	app.Get("/api/settings", handler.Get)

	req := httptest.NewRequest("GET", "/api/settings", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	if resp.StatusCode != 200 {
		t.Errorf("status = %d, want 200", resp.StatusCode)
	}

	var body map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}

	scraper, ok := body["scraper"].(map[string]interface{})
	if !ok {
		t.Fatal("missing scraper key")
	}

	if scraper["enabled"] != false {
		t.Error("enabled should default to false")
	}
	if scraper["schedule"] != "0 2 1 1 *" {
		t.Errorf("schedule = %v, want '0 2 1 1 *'", scraper["schedule"])
	}
}

func TestSettingsPutValid(t *testing.T) {
	db := setupSettingsTestDB(t)
	defer db.Close()

	handler := &Settings{DB: db}

	app := fiber.New()
	app.Put("/api/settings", handler.Put)

	req := httptest.NewRequest("PUT", "/api/settings",
		strings.NewReader(`{"scraper":{"enabled":true,"schedule":"*/5 * * * *"}}`))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	if resp.StatusCode != 200 {
		t.Errorf("status = %d, want 200", resp.StatusCode)
	}

	var body map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&body)
	scraper := body["scraper"].(map[string]interface{})

	if scraper["enabled"] != true {
		t.Error("enabled should be true after PUT")
	}
	if scraper["schedule"] != "*/5 * * * *" {
		t.Errorf("schedule = %v, want '*/5 * * * *'", scraper["schedule"])
	}
}

func TestSettingsPutInvalidCron(t *testing.T) {
	db := setupSettingsTestDB(t)
	defer db.Close()

	handler := &Settings{DB: db}
	app := fiber.New()
	app.Put("/api/settings", handler.Put)

	req := httptest.NewRequest("PUT", "/api/settings",
		strings.NewReader(`{"scraper":{"schedule":"invalid"}}`))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	if resp.StatusCode != 400 {
		t.Errorf("status = %d, want 400", resp.StatusCode)
	}
}

func TestSettingsPutUnknownKey(t *testing.T) {
	db := setupSettingsTestDB(t)
	defer db.Close()

	handler := &Settings{DB: db}
	app := fiber.New()
	app.Put("/api/settings", handler.Put)

	req := httptest.NewRequest("PUT", "/api/settings",
		strings.NewReader(`{"unknown":"value"}`))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	if resp.StatusCode != 400 {
		t.Errorf("status = %d, want 400", resp.StatusCode)
	}
}
