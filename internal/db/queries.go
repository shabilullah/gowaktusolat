package db

import (
	"database/sql"
	"fmt"
	"time"
)

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

func QueryPrayerTimes(database *sql.DB, zone string, year, month int) ([]PrayerTimeRow, error) {
	startDate := fmt.Sprintf("%d-%02d-01", year, month)
	lastDay := daysInMonth(year, month)
	endDate := fmt.Sprintf("%d-%02d-%02d", year, month, lastDay)

	rows, err := database.Query(
		`SELECT date, hijri, imsak, fajr, syuruk, dhuha, dhuhr, asr, maghrib, isha
		 FROM prayer_times
		 WHERE location_code = ? AND date >= ? AND date <= ?
		 ORDER BY date ASC`,
		zone, startDate, endDate,
	)
	if err != nil {
		return nil, fmt.Errorf("query prayer times: %w", err)
	}
	defer rows.Close()

	var results []PrayerTimeRow
	for rows.Next() {
		var pt PrayerTimeRow
		if err := rows.Scan(
			&pt.Date, &pt.Hijri,
			&pt.Imsak, &pt.Fajr, &pt.Syuruk, &pt.Dhuha,
			&pt.Dhuhr, &pt.Asr, &pt.Maghrib, &pt.Isha,
		); err != nil {
			return nil, fmt.Errorf("scan prayer time: %w", err)
		}
		results = append(results, pt)
	}

	if len(results) == 0 {
		return nil, sql.ErrNoRows
	}

	return results, nil
}

func daysInMonth(year, month int) int {
	t := time.Date(year, time.Month(month)+1, 0, 0, 0, 0, 0, time.UTC)
	return t.Day()
}
