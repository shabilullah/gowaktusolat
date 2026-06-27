package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"

	"zombiezen.com/go/sqlite"
	"zombiezen.com/go/sqlite/sqlitex"

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

	switch cmd {
	case "seed-zones":
		if err := scraper.SeedZones(pool); err != nil {
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

		scraper.RunFullScrape(context.Background(), pool, *year)

	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", cmd)
		os.Exit(1)
	}
}
