package main

import (
	"context"
	"errors"
	"log"
	"net"
	"os"
	"os/signal"
	"strings"
	"syscall"
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
	reposqlite "github.com/shabilullah/gowaktusolat/internal/repository/sqlite"
	"github.com/shabilullah/gowaktusolat/internal/scraper"
	"github.com/shabilullah/gowaktusolat/internal/service"
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

	// Create concrete repository implementations.
	prayerRepo := &reposqlite.PrayerTimeRepo{Pool: pool}
	zoneRepo := &reposqlite.ZoneRepo{Pool: pool}

	// SEEDER_SCHED env var controls the cron schedule for auto-scraping.
	sched := scraper.NewScheduler(pool, cfg.SeederSched)
	if cfg.SeederSched != "" {
		log.Printf("Auto-scraping enabled with schedule: %s", cfg.SeederSched)
	}
	sched.Start()
	defer sched.Stop()

	// Seed GeoJSON polygon data (first-run, idempotent).
	if err := geo.SeedFromGeoJSON(pool); err != nil {
		log.Printf("Warning: failed to seed GeoJSON polygons: %v", err)
	}

	detector, err := geo.NewDetector(pool)
	if err != nil {
		log.Fatalf("Failed to initialize GPS detector: %v", err)
	}

	// Create service layer — handlers depend on these interfaces.
	prayerSvc := service.NewPrayerService(prayerRepo, detector)
	zoneSvc := service.NewZoneService(zoneRepo, detector)
	pdfSvc := service.NewPDFService()

	// Seed prayer_zones table if empty (first run).
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
			log.Print("First run detected — seeding zones")
			if err := scraper.SeedZones(pool); err != nil {
				log.Fatalf("Failed to seed zones on first run: %v", err)
			}
		}
	}

	// Determine which years need scraping.
	wanted := make(map[int]bool)
	wanted[time.Now().Year()] = true
	for _, y := range cfg.Years {
		wanted[y] = true
	}
	if scraped, err := prayerRepo.ScrapedYears(context.Background()); err != nil {
		log.Printf("Failed to check scraped years: %v", err)
	} else {
		for y := range scraped {
			delete(wanted, y)
		}
	}
	for year := range wanted {
		log.Printf("Year %d not yet scraped — starting scrape", year)
		go scraper.RunFullScrape(context.Background(), pool, year)
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
	api.RegisterRoutes(app, prayerSvc, zoneSvc, pdfSvc, pool, cfg.APIKey)

	go func() {
		log.Printf("Server starting on port %s (prefork=%v)", cfg.Port, cfg.Prefork)
		if err := app.Listen(":"+cfg.Port, fiber.ListenConfig{EnablePrefork: cfg.Prefork}); err != nil && !errors.Is(err, net.ErrClosed) {
			log.Fatalf("Server failed: %v", err)
		}
	}()

	// Wait for interrupt signal for graceful shutdown.
	waitForShutdown()
	log.Print("Shutting down server...")

	if err := app.ShutdownWithTimeout(10 * time.Second); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}
}

var waitForShutdown = func() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(c)
	<-c
}
