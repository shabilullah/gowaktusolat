package main

import (
	"context"
	"log"
	"strings"
	"time"

	json "github.com/goccy/go-json"
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/cors"
	"github.com/gofiber/fiber/v3/middleware/etag"
	"github.com/gofiber/fiber/v3/middleware/limiter"
	"github.com/gofiber/fiber/v3/middleware/recover"
	"zombiezen.com/go/sqlite"
	"zombiezen.com/go/sqlite/sqlitex"

	"github.com/shabilullah/gowaktusolat/internal/api"
	"github.com/shabilullah/gowaktusolat/internal/config"
	"github.com/shabilullah/gowaktusolat/internal/db"
	"github.com/shabilullah/gowaktusolat/internal/geo"
	"github.com/shabilullah/gowaktusolat/internal/scraper"
)

func main() {
	cfg := config.Load()

	pool, err := sqlitex.NewPool(cfg.DBPath, sqlitex.PoolOptions{
		Flags:    sqlite.OpenReadWrite | sqlite.OpenCreate | sqlite.OpenWAL | sqlite.OpenURI,
		PoolSize: 4,
	})
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer pool.Close()

	if err := db.InitPool(pool); err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}

	// SEEDER_SCHED env var controls the cron schedule for auto-scraping.
	// When set, the scheduler runs with the given schedule.
	// When empty, no scheduled scraping occurs.
	sched := scraper.NewScheduler(pool, cfg.SeederSched)
	if cfg.SeederSched != "" {
		log.Printf("Auto-scraping enabled with schedule: %s", cfg.SeederSched)
	}
	sched.Start()
	defer sched.Stop()

	detector, err := geo.NewDetector(pool)
	if err != nil {
		log.Fatalf("Failed to initialize GPS detector: %v", err)
	}

	// First-run: seed zones and scrape current year data if DB is fresh.
	var zoneCount int64
	execConn, err := pool.Take(context.Background())
	if err != nil {
		log.Printf("Failed to take connection: %v", err)
	} else {
		err = sqlitex.Execute(execConn, "SELECT COUNT(*) FROM prayer_zones", &sqlitex.ExecOptions{
			ResultFunc: func(stmt *sqlite.Stmt) error {
				zoneCount = stmt.ColumnInt64(0)
				return nil
			},
		})
		pool.Put(execConn)
		if err != nil {
			log.Printf("Failed to check prayer_zones: %v", err)
		} else if zoneCount == 0 {
			log.Printf("First run detected — starting initial seed and scrape for %d", time.Now().Year())
			go scraper.RunFullScrape(context.Background(), pool, time.Now().Year())
		}
	}

	app := fiber.New(fiber.Config{
		AppName:     "Go Waktu Solat API",
		JSONEncoder: json.Marshal,
		JSONDecoder: json.Unmarshal,
	})

	app.Use(recover.New())
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
	api.RegisterRoutes(app, pool, detector, cfg.APIKey)

	log.Printf("Server starting on port %s (prefork=%v)", cfg.Port, cfg.Prefork)
	if err := app.Listen(":"+cfg.Port, fiber.ListenConfig{EnablePrefork: cfg.Prefork}); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
