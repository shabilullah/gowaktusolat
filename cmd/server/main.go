package main

import (
	"context"
	"database/sql"
	"log"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/cors"
	"github.com/gofiber/fiber/v3/middleware/etag"
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

	// SEEDER_SCHED env var controls the cron schedule for auto-scraping.
	// When set, the scheduler runs with the given schedule.
	// When empty, no scheduled scraping occurs.
	sched := scraper.NewScheduler(database, cfg.SeederSched)
	if cfg.SeederSched != "" {
		log.Printf("Auto-scraping enabled with schedule: %s", cfg.SeederSched)
	}
	sched.Start()
	defer sched.Stop()

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

	app := fiber.New(fiber.Config{
		AppName: "Go Waktu Solat API",
	})

	app.Use(cors.New(cors.Config{
		AllowOrigins: cfg.CORSOriginsSlice(),
	}))

	app.Use(etag.New())

	api.RegisterRoutes(app, database, detector)

	log.Printf("Server starting on port %s (prefork=%v)", cfg.Port, cfg.Prefork)
	if err := app.Listen(":"+cfg.Port, fiber.ListenConfig{EnablePrefork: cfg.Prefork}); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
