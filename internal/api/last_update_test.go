package api

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v3"
	"zombiezen.com/go/sqlite"
	"zombiezen.com/go/sqlite/sqlitex"
)

func setupLastUpdateTestDB(t *testing.T) *sqlitex.Pool {
	t.Helper()
	pool, err := sqlitex.NewPool("file::memory:?cache=shared", sqlitex.PoolOptions{
		Flags:    sqlite.OpenReadWrite | sqlite.OpenCreate | sqlite.OpenWAL | sqlite.OpenURI,
		PoolSize: 1,
	})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}

	conn, err := pool.Take(context.Background())
	if err != nil {
		t.Fatalf("take: %v", err)
	}
	defer pool.Put(conn)

	if err := sqlitex.Execute(conn, `CREATE TABLE IF NOT EXISTS settings (
		key TEXT PRIMARY KEY,
		value TEXT NOT NULL,
		updated_at TEXT NOT NULL
	)`, nil); err != nil {
		t.Fatalf("create table: %v", err)
	}

	defaults := map[string]string{
		"scraper.last_run":    "2026-06-27T12:00:00Z",
		"scraper.last_status": "success: 53/53 zones",
	}
	for k, v := range defaults {
		if err := sqlitex.Execute(conn, "INSERT INTO settings (key, value, updated_at) VALUES (?, ?, datetime('now'))", &sqlitex.ExecOptions{
			Args: []interface{}{k, v},
		}); err != nil {
			t.Fatalf("seed %s: %v", k, err)
		}
	}

	return pool
}

func TestLastUpdateGet(t *testing.T) {
	pool := setupLastUpdateTestDB(t)
	defer pool.Close()

	handler := &LastUpdate{DB: pool}

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
	pool, err := sqlitex.NewPool("file::memory:?cache=shared", sqlitex.PoolOptions{
		Flags:    sqlite.OpenReadWrite | sqlite.OpenCreate | sqlite.OpenWAL | sqlite.OpenURI,
		PoolSize: 1,
	})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer pool.Close()

	conn, err := pool.Take(context.Background())
	if err != nil {
		t.Fatalf("take: %v", err)
	}

	if err := sqlitex.Execute(conn, `CREATE TABLE IF NOT EXISTS settings (
		key TEXT PRIMARY KEY,
		value TEXT NOT NULL,
		updated_at TEXT NOT NULL
	)`, nil); err != nil {
		t.Fatalf("create table: %v", err)
	}

	pool.Put(conn)

	handler := &LastUpdate{DB: pool}

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
