package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"log"
	"os"

	_ "modernc.org/sqlite"

	"github.com/shabilullah/gowaktusolat/internal/config"
	"github.com/shabilullah/gowaktusolat/internal/db"
	"github.com/shabilullah/gowaktusolat/internal/scraper"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "Usage: scraper <command> [options]\n")
		fmt.Fprintf(os.Stderr, "Commands:\n")
		fmt.Fprintf(os.Stderr, "  seed-zones          Seed prayer_zones table from embedded data\n")
		fmt.Fprintf(os.Stderr, "  scrape --year=YYYY  Scrape prayer times for all zones\n")
		os.Exit(1)
	}

	cmd := os.Args[1]

	cfg := config.Load()

	database, err := sql.Open("sqlite", cfg.DBPath)
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer database.Close()

	if err := db.InitDB(database); err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}

	switch cmd {
	case "seed-zones":
		if err := scraper.SeedZones(database); err != nil {
			log.Fatalf("Failed to seed zones: %v", err)
		}
		fmt.Println("Zones seeded successfully.")
		zones, _ := scraper.LoadZones()
		for _, z := range zones {
			fmt.Printf("  ✓ %s (%s - %s)\n", z.JakimCode, z.Negeri, z.Daerah)
		}

	case "scrape":
		fs := flag.NewFlagSet("scrape", flag.ExitOnError)
		year := fs.Int("year", 0, "Year to scrape")
		fs.Parse(os.Args[2:])

		if *year == 0 {
			log.Fatal("--year is required")
		}

		scraper.RunFullScrape(context.Background(), database, *year)

	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", cmd)
		os.Exit(1)
	}
}
