package sqlite

import (
	"context"
	"fmt"
	"time"

	"github.com/shabilullah/gowaktusolat/internal/repository"
	"zombiezen.com/go/sqlite"
	"zombiezen.com/go/sqlite/sqlitex"
)

// PrayerTimeRepo implements repository.PrayerTimeRepository backed by SQLite.
type PrayerTimeRepo struct {
	Pool *sqlitex.Pool
}

func (r *PrayerTimeRepo) Query(ctx context.Context, zone string, year, month int) ([]repository.PrayerTimeRow, error) {
	conn, err := r.Pool.Take(ctx)
	if err != nil {
		return nil, fmt.Errorf("acquire db conn: %w", err)
	}
	defer r.Pool.Put(conn)

	startDate := fmt.Sprintf("%d-%02d-01", year, month)
	lastDay := daysInMonth(year, month)
	endDate := fmt.Sprintf("%d-%02d-%02d", year, month, lastDay)

	var results []repository.PrayerTimeRow
	err = sqlitex.Execute(conn,
		`SELECT date, hijri, imsak, fajr, syuruk, dhuha, dhuhr, asr, maghrib, isha
		 FROM prayer_times
		 WHERE location_code = ? AND date >= ? AND date <= ?
		 ORDER BY date ASC`,
		&sqlitex.ExecOptions{
			Args: []interface{}{zone, startDate, endDate},
			ResultFunc: func(stmt *sqlite.Stmt) error {
				results = append(results, repository.PrayerTimeRow{
					Date:    stmt.ColumnText(0),
					Hijri:   stmt.ColumnText(1),
					Imsak:   stmt.ColumnText(2),
					Fajr:    stmt.ColumnText(3),
					Syuruk:  stmt.ColumnText(4),
					Dhuha:   stmt.ColumnText(5),
					Dhuhr:   stmt.ColumnText(6),
					Asr:     stmt.ColumnText(7),
					Maghrib: stmt.ColumnText(8),
					Isha:    stmt.ColumnText(9),
				})
				return nil
			},
		},
	)
	if err != nil {
		return nil, fmt.Errorf("query prayer times: %w", err)
	}

	if len(results) == 0 {
		return nil, repository.ErrNoRows
	}

	return results, nil
}

// QueryYear returns all prayer times for a zone for the whole year,
// ordered by date. Used by the year-PDF endpoint to avoid N+1 monthly queries.
func (r *PrayerTimeRepo) QueryYear(ctx context.Context, zone string, year int) ([]repository.PrayerTimeRow, error) {
	conn, err := r.Pool.Take(ctx)
	if err != nil {
		return nil, fmt.Errorf("acquire db conn: %w", err)
	}
	defer r.Pool.Put(conn)

	startDate := fmt.Sprintf("%d-01-01", year)
	endDate := fmt.Sprintf("%d-12-31", year)

	var results []repository.PrayerTimeRow
	err = sqlitex.Execute(conn,
		`SELECT date, hijri, imsak, fajr, syuruk, dhuha, dhuhr, asr, maghrib, isha
		 FROM prayer_times
		 WHERE location_code = ? AND date >= ? AND date <= ?
		 ORDER BY date ASC`,
		&sqlitex.ExecOptions{
			Args: []interface{}{zone, startDate, endDate},
			ResultFunc: func(stmt *sqlite.Stmt) error {
				results = append(results, repository.PrayerTimeRow{
					Date:    stmt.ColumnText(0),
					Hijri:   stmt.ColumnText(1),
					Imsak:   stmt.ColumnText(2),
					Fajr:    stmt.ColumnText(3),
					Syuruk:  stmt.ColumnText(4),
					Dhuha:   stmt.ColumnText(5),
					Dhuhr:   stmt.ColumnText(6),
					Asr:     stmt.ColumnText(7),
					Maghrib: stmt.ColumnText(8),
					Isha:    stmt.ColumnText(9),
				})
				return nil
			},
		},
	)
	if err != nil {
		return nil, fmt.Errorf("query prayer times year: %w", err)
	}

	if len(results) == 0 {
		return nil, repository.ErrNoRows
	}

	return results, nil
}

// ScrapedYears returns the set of years where every prayer_zone has at least one
// row in prayer_times. A year is only "complete" when all zones are present.
func (r *PrayerTimeRepo) ScrapedYears(ctx context.Context) (map[int]bool, error) {
	conn, err := r.Pool.Take(ctx)
	if err != nil {
		return nil, fmt.Errorf("acquire db conn: %w", err)
	}
	defer r.Pool.Put(conn)

	years := make(map[int]bool)
	err = sqlitex.Execute(conn,
		`SELECT CAST(substr(date, 1, 4) AS INTEGER) AS year
		 FROM prayer_times
		 GROUP BY year
		 HAVING COUNT(DISTINCT location_code) = (SELECT COUNT(*) FROM prayer_zones)`,
		&sqlitex.ExecOptions{
			ResultFunc: func(stmt *sqlite.Stmt) error {
				years[stmt.ColumnInt(0)] = true
				return nil
			},
		},
	)
	if err != nil {
		return nil, fmt.Errorf("query scraped years: %w", err)
	}
	return years, nil
}

func daysInMonth(year, month int) int {
	t := time.Date(year, time.Month(month)+1, 0, 0, 0, 0, 0, time.UTC)
	return t.Day()
}
