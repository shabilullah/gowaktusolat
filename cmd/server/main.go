package main

import (
	"database/sql"
	"log"

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

	detector, err := geo.NewDetector(database)
	if err != nil {
		log.Fatalf("Failed to initialize GPS detector: %v", err)
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
