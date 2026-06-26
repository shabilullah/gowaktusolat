package scraper

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"
)

func RunFullScrape(ctx context.Context, db *sql.DB, year int) {
	log.Printf("[scraper] Starting full scrape for year %d", year)

	if err := updateScraperStatus(db, "running"); err != nil {
		log.Printf("[scraper] Failed to update status to running: %v", err)
	}

	if err := SeedZones(db); err != nil {
		log.Printf("[scraper] Seed zones failed: %v", err)
		_ = updateScraperStatus(db, "failed")
		return
	}

	rows, err := db.QueryContext(ctx, "SELECT jakim_code FROM prayer_zones ORDER BY jakim_code")
	if err != nil {
		log.Printf("[scraper] Query zones failed: %v", err)
		_ = updateScraperStatus(db, "failed")
		return
	}
	defer rows.Close()

	var zones []string
	for rows.Next() {
		var code string
		if err := rows.Scan(&code); err != nil {
			log.Printf("[scraper] Scan zone failed: %v", err)
			continue
		}
		zones = append(zones, code)
	}

	successCount := 0
	failCount := 0
	for i, zone := range zones {
		select {
		case <-ctx.Done():
			log.Printf("[scraper] Cancelled after %d/%d zones", successCount+failCount, len(zones))
			_ = updateScraperStatus(db, "failed")
			return
		default:
		}

		times, err := FetchPrayerTimes(ctx, zone, year)
		if err != nil {
			log.Printf("[scraper] ✗ %s: %v", zone, err)
			failCount++
			continue
		}
		if times == nil {
			log.Printf("[scraper] - %s (no data for %d)", zone, year)
			continue
		}

		if err := SavePrayerTimes(db, zone, year, times); err != nil {
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
				_ = updateScraperStatus(db, "failed")
				return
			}
		}
	}

	status := fmt.Sprintf("success: %d/%d zones", successCount, successCount+failCount)
	if failCount > 0 {
		status = fmt.Sprintf("partial: %d/%d zones (%d failed)", successCount, successCount+failCount, failCount)
	}
	log.Printf("[scraper] Complete: %s", status)
	_ = updateScraperStatus(db, status)
}

func updateScraperStatus(db *sql.DB, status string) error {
	now := time.Now().UTC().Format(time.RFC3339)
	if _, err := db.Exec(
		"INSERT OR REPLACE INTO settings (key, value, updated_at) VALUES ('scraper.last_run', ?, ?)",
		now, now,
	); err != nil {
		return err
	}
	if _, err := db.Exec(
		"INSERT OR REPLACE INTO settings (key, value, updated_at) VALUES ('scraper.last_status', ?, ?)",
		status, now,
	); err != nil {
		return err
	}
	return nil
}
