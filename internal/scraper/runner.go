package scraper

import (
	"context"
	"fmt"
	"log"
	"time"

	"zombiezen.com/go/sqlite"
	"zombiezen.com/go/sqlite/sqlitex"
)

func RunFullScrape(ctx context.Context, pool *sqlitex.Pool, year int) {
	log.Printf("[scraper] Starting full scrape for year %d", year)

	if err := updateScraperStatus(pool, "running"); err != nil {
		log.Printf("[scraper] Failed to update status to running: %v", err)
	}

	if err := SeedZones(pool); err != nil {
		log.Printf("[scraper] Seed zones failed: %v", err)
		_ = updateScraperStatus(pool, "failed")
		return
	}

	conn, err := pool.Take(context.Background())
	if err != nil {
		log.Printf("[scraper] Take conn failed: %v", err)
		_ = updateScraperStatus(pool, "failed")
		return
	}
	defer pool.Put(conn)

	var zones []string
	if err := sqlitex.ExecuteTransient(conn, "SELECT jakim_code FROM prayer_zones ORDER BY jakim_code", &sqlitex.ExecOptions{
		ResultFunc: func(stmt *sqlite.Stmt) error {
			zones = append(zones, stmt.ColumnText(0))
			return nil
		},
	}); err != nil {
		log.Printf("[scraper] Query zones failed: %v", err)
		_ = updateScraperStatus(pool, "failed")
		return
	}

	successCount := 0
	failCount := 0
	for i, zone := range zones {
		select {
		case <-ctx.Done():
			log.Printf("[scraper] Cancelled after %d/%d zones", successCount+failCount, len(zones))
			_ = updateScraperStatus(pool, "failed")
			return
		default:
		}

		times, err := FetchPrayerTimes(zone, year)
		if err != nil {
			log.Printf("[scraper] ✗ %s: %v", zone, err)
			failCount++
			continue
		}
		if times == nil {
			log.Printf("[scraper] - %s (no data for %d)", zone, year)
			continue
		}

		if err := SavePrayerTimes(pool, zone, year, times); err != nil {
			log.Printf("[scraper] ✗ %s (save): %v", zone, err)
			failCount++
			continue
		}

		log.Printf("[scraper] ✓ %s (%d days)", zone, len(times))
		successCount++

		if i < len(zones)-1 {
			select {
			case <-time.After(1200 * time.Millisecond):
			case <-ctx.Done():
				log.Printf("[scraper] Cancelled during delay")
				_ = updateScraperStatus(pool, "failed")
				return
			}
		}
	}

	status := fmt.Sprintf("success: %d/%d zones", successCount, successCount+failCount)
	if failCount > 0 {
		status = fmt.Sprintf("partial: %d/%d zones (%d failed)", successCount, successCount+failCount, failCount)
	}
	log.Printf("[scraper] Complete: %s", status)
	_ = updateScraperStatus(pool, status)
}

func updateScraperStatus(pool *sqlitex.Pool, status string) error {
	conn, err := pool.Take(context.Background())
	if err != nil {
		return fmt.Errorf("take conn: %w", err)
	}
	defer pool.Put(conn)

	now := time.Now().UTC().Format(time.RFC3339)
	if err := sqlitex.Exec(conn,
		"INSERT OR REPLACE INTO settings (key, value, updated_at) VALUES ('scraper.last_run', ?, ?)",
		nil, now, now,
	); err != nil {
		return err
	}
	if err := sqlitex.Exec(conn,
		"INSERT OR REPLACE INTO settings (key, value, updated_at) VALUES ('scraper.last_status', ?, ?)",
		nil, status, now,
	); err != nil {
		return err
	}
	return nil
}
