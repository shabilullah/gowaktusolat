package main

import (
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
)

func TestServerShutdown(t *testing.T) {
	app := fiber.New()
	app.Get("/ping", func(c fiber.Ctx) error {
		return c.SendString("pong")
	})

	// Override the shutdown wait so test doesn't block on OS signal.
	waitForShutdown = func() {}

	// Verify routing works (in-memory request, no port needed).
	req := httptest.NewRequest("GET", "/ping", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("status = %d, want 200", resp.StatusCode)
	}

	// Shutdown with timeout — must complete without error.
	if err := app.ShutdownWithTimeout(5 * time.Second); err != nil {
		t.Fatalf("Shutdown failed: %v", err)
	}
}
