package main

import (
	"context"
	"database/sql"
	"log"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/cors"
	_ "modernc.org/sqlite"

	"github.com/shabilullah/gowaktusolat/internal/api"
	"github.com/shabilullah/gowaktusolat/internal/config"
	"github.com/shabilullah/gowaktusolat/internal/db"
	"github.com/shabilullah/gowaktusolat/internal/geo"
	"github.com/shabilullah/gowaktusolat/internal/scraper"
)

func main() {
	cfg := config.Load()

	database, err := sql.Open("sqlite", cfg.DBPath)
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer database.Close()

	if err := db.InitDB(database); err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}

	// Override scheduler settings from SEEDER_SCHED env var.
	if cfg.SeederSched != "" {
		if _, err := database.Exec(
			"INSERT OR REPLACE INTO settings (key, value, updated_at) VALUES ('scraper.schedule', ?, datetime('now'))",
			cfg.SeederSched,
		); err != nil {
			log.Printf("Failed to set scraper.schedule: %v", err)
		}
		if _, err := database.Exec(
			"INSERT OR REPLACE INTO settings (key, value, updated_at) VALUES ('scraper.enabled', 'true', datetime('now'))",
		); err != nil {
			log.Printf("Failed to set scraper.enabled: %v", err)
		}
		log.Printf("SEEDER_SCHED set — auto-scraping enabled with schedule: %s", cfg.SeederSched)
	}

	detector, err := geo.NewDetector(database)
	if err != nil {
		log.Fatalf("Failed to initialize GPS detector: %v", err)
	}

	// First-run: seed zones and scrape current year data if DB is fresh.
	var zoneCount int
	if err := database.QueryRow("SELECT COUNT(*) FROM prayer_zones").Scan(&zoneCount); err != nil {
		log.Printf("Failed to check prayer_zones: %v", err)
	} else if zoneCount == 0 {
		log.Printf("First run detected — starting initial seed and scrape for %d", time.Now().Year())
		go scraper.RunFullScrape(context.Background(), database, time.Now().Year())
	}

	sched := scraper.NewScheduler(database)
	sched.Start()
	defer sched.Stop()

	app := fiber.New(fiber.Config{
		AppName: "Go Waktu Solat API",
	})

	app.Use(cors.New(cors.Config{
		AllowOrigins: cfg.CORSOriginsSlice(),
	}))

	api.RegisterRoutes(app, database, detector, cfg)

	log.Printf("Server starting on port %s (prefork=%v)", cfg.Port, cfg.Prefork)
	if err := app.Listen(":"+cfg.Port, fiber.ListenConfig{EnablePrefork: cfg.Prefork}); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
