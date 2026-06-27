package main

import (
	"context"
	"database/sql"
	"log"
	"strings"
	"time"

	json "github.com/goccy/go-json"
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/cors"
	"github.com/gofiber/fiber/v3/middleware/etag"
	"github.com/gofiber/fiber/v3/middleware/limiter"
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
		AppName:     "Go Waktu Solat API",
		JSONEncoder: json.Marshal,
		JSONDecoder: json.Unmarshal,
	})

	app.Use(cors.New(cors.Config{
		AllowOrigins: cfg.CORSOriginsSlice(),
	}))

	app.Use(etag.New())

	app.Use(limiter.New(limiter.Config{
		MaxFunc: func(c fiber.Ctx) int {
			path := c.Path()
			// PDF generation is expensive — tighter limit
			if strings.Contains(path, "jadual_solat") {
				return 10
			}
			return 60
		},
		ExpirationFunc: func(c fiber.Ctx) time.Duration {
			path := c.Path()
			if strings.Contains(path, "jadual_solat") {
				return 2 * time.Minute
			}
			return 1 * time.Minute
		},
		LimiterMiddleware: limiter.SlidingWindow{},
		LimitReached: func(c fiber.Ctx) error {
			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
				"message": "Too many requests. Please try again later.",
			})
		},
	}))
	api.RegisterRoutes(app, database, detector, cfg.APIKey)

	log.Printf("Server starting on port %s (prefork=%v)", cfg.Port, cfg.Prefork)
	if err := app.Listen(":"+cfg.Port, fiber.ListenConfig{EnablePrefork: cfg.Prefork}); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
