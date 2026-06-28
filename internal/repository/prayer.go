package repository

import (
	"context"
	"errors"
)

// ErrNoRows is returned when a query returns no results.
var ErrNoRows = errors.New("repository: no rows in result set")

// PrayerTimeRow is a single prayer-time record from the database.
type PrayerTimeRow struct {
	Date    string
	Hijri   string
	Imsak   string
	Fajr    string
	Syuruk  string
	Dhuha   string
	Dhuhr   string
	Asr     string
	Maghrib string
	Isha    string
}

// PrayerTimeRepository defines the data-access contract for prayer times.
type PrayerTimeRepository interface {
	Query(ctx context.Context, zone string, year, month int) ([]PrayerTimeRow, error)
	QueryYear(ctx context.Context, zone string, year int) ([]PrayerTimeRow, error)
	ScrapedYears(ctx context.Context) (map[int]bool, error)
}
