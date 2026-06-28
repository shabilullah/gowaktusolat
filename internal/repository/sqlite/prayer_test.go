package sqlite

import (
	"context"
	"testing"
	"time"

	"github.com/shabilullah/gowaktusolat/internal/repository"
	"zombiezen.com/go/sqlite"
	"zombiezen.com/go/sqlite/sqlitex"
)

func setupQueryTestDB(t *testing.T) *PrayerTimeRepo {
	t.Helper()
	pool, err := sqlitex.NewPool("file::memory:?cache=shared", sqlitex.PoolOptions{
		Flags:    sqlite.OpenReadWrite | sqlite.OpenCreate | sqlite.OpenWAL | sqlite.OpenURI,
		PoolSize: 1,
	})
	if err != nil {
		t.Fatalf("open pool: %v", err)
	}
	t.Cleanup(func() { pool.Close() })

	conn, err := pool.Take(context.Background())
	if err != nil {
		t.Fatalf("take conn: %v", err)
	}
	defer pool.Put(conn)

	if err := sqlitex.Execute(conn, "PRAGMA journal_mode=WAL", nil); err != nil {
		t.Fatalf("wal: %v", err)
	}

	if err := sqlitex.Execute(conn, `CREATE TABLE IF NOT EXISTS prayer_times (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		date TEXT NOT NULL,
		location_code TEXT NOT NULL,
		hijri TEXT,
		imsak TEXT,
		fajr TEXT,
		syuruk TEXT,
		dhuha TEXT,
		dhuhr TEXT,
		asr TEXT,
		maghrib TEXT,
		isha TEXT,
		created_at TEXT NOT NULL,
		updated_at TEXT NOT NULL
	)`, nil); err != nil {
		t.Fatalf("create table: %v", err)
	}

	if err := sqlitex.Execute(conn, `CREATE UNIQUE INDEX IF NOT EXISTS idx_prayer_times_zone_date ON prayer_times(location_code, date)`, nil); err != nil {
		t.Fatalf("create index: %v", err)
	}

	now := time.Now().UTC().Format(time.RFC3339)
	testData := []struct {
		date, zone, fajr, dhuhr, asr, maghrib, isha string
	}{
		{"2026-06-01", "SGR01", "05:49:00", "13:14:00", "16:39:00", "19:22:00", "20:37:00"},
		{"2026-06-02", "SGR01", "05:49:00", "13:14:00", "16:39:00", "19:23:00", "20:38:00"},
		{"2026-06-03", "SGR01", "05:49:00", "13:14:00", "16:39:00", "19:23:00", "20:38:00"},
		{"2026-06-04", "SGR01", "05:49:00", "13:15:00", "16:40:00", "19:23:00", "20:38:00"},
		{"2026-06-05", "SGR01", "05:50:00", "13:15:00", "16:40:00", "19:23:00", "20:38:00"},
		{"2026-06-01", "JHR01", "05:39:00", "13:02:00", "16:27:00", "19:09:00", "20:24:00"},
	}

	for _, d := range testData {
		if err := sqlitex.Execute(conn,
			`INSERT INTO prayer_times (date, location_code, hijri, imsak, fajr, syuruk, dhuha, dhuhr, asr, maghrib, isha, created_at, updated_at)
			 VALUES (?, ?, '', '', ?, '', '', ?, ?, ?, ?, ?, ?)`,
			&sqlitex.ExecOptions{
				Args: []interface{}{
					d.date, d.zone, d.fajr, d.dhuhr, d.asr, d.maghrib, d.isha, now, now,
				},
			},
		); err != nil {
			t.Fatalf("insert %s/%s: %v", d.zone, d.date, err)
		}
	}

	return &PrayerTimeRepo{Pool: pool}
}

func TestQueryPrayerTimes(t *testing.T) {
	repo := setupQueryTestDB(t)

	rows, err := repo.Query(context.Background(), "SGR01", 2026, 6)
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}

	if len(rows) != 5 {
		t.Errorf("got %d rows, want 5", len(rows))
	}

	if rows[0].Date != "2026-06-01" {
		t.Errorf("first date = %s, want 2026-06-01", rows[0].Date)
	}
	if rows[0].Fajr != "05:49:00" {
		t.Errorf("first fajr = %s, want 05:49:00", rows[0].Fajr)
	}

	if rows[4].Date != "2026-06-05" {
		t.Errorf("last date = %s, want 2026-06-05", rows[4].Date)
	}
}

func TestQueryPrayerTimesDifferentZone(t *testing.T) {
	repo := setupQueryTestDB(t)

	rows, err := repo.Query(context.Background(), "JHR01", 2026, 6)
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}

	if len(rows) != 1 {
		t.Errorf("got %d rows for JHR01, want 1", len(rows))
	}
	_ = rows[0]
}

func TestQueryPrayerTimesNoData(t *testing.T) {
	repo := setupQueryTestDB(t)

	_, err := repo.Query(context.Background(), "XXXXX", 2026, 6)
	if err != repository.ErrNoRows {
		t.Errorf("expected ErrNoRows, got %v", err)
	}
}

func TestQueryPrayerTimesDifferentMonth(t *testing.T) {
	repo := setupQueryTestDB(t)

	_, err := repo.Query(context.Background(), "SGR01", 2026, 1)
	if err != repository.ErrNoRows {
		t.Errorf("expected ErrNoRows for January (no data), got %v", err)
	}
}

func TestDaysInMonth(t *testing.T) {
	tests := []struct {
		year, month, expected int
	}{
		{2026, 1, 31},
		{2026, 2, 28},
		{2024, 2, 29},
		{2026, 4, 30},
		{2026, 6, 30},
		{2026, 12, 31},
	}

	for _, tt := range tests {
		got := daysInMonth(tt.year, tt.month)
		if got != tt.expected {
			t.Errorf("daysInMonth(%d, %d) = %d, want %d", tt.year, tt.month, got, tt.expected)
		}
	}
}

func TestQueryPrayerTimesYear(t *testing.T) {
	repo := setupQueryTestDB(t)

	rows, err := repo.QueryYear(context.Background(), "SGR01", 2026)
	if err != nil {
		t.Fatalf("QueryYear failed: %v", err)
	}

	if len(rows) != 5 {
		t.Errorf("got %d rows for SGR01 2026, want 5", len(rows))
	}

	for _, r := range rows {
		if len(r.Date) < 7 || r.Date[:7] != "2026-06" {
			t.Errorf("row date %s is outside expected month 2026-06", r.Date)
		}
	}

	if rows[0].Date != "2026-06-01" {
		t.Errorf("first date = %s, want 2026-06-01", rows[0].Date)
	}
	if rows[4].Date != "2026-06-05" {
		t.Errorf("last date = %s, want 2026-06-05", rows[4].Date)
	}
}

func TestQueryPrayerTimesYearMultipleZones(t *testing.T) {
	repo := setupQueryTestDB(t)

	rows, err := repo.QueryYear(context.Background(), "JHR01", 2026)
	if err != nil {
		t.Fatalf("QueryYear failed: %v", err)
	}

	if len(rows) != 1 {
		t.Errorf("got %d rows for JHR01 2026, want 1", len(rows))
	}
	if rows[0].Date != "2026-06-01" {
		t.Errorf("date = %s, want 2026-06-01", rows[0].Date)
	}
}

func TestQueryPrayerTimesYearNoData(t *testing.T) {
	repo := setupQueryTestDB(t)

	_, err := repo.QueryYear(context.Background(), "XXXXX", 2026)
	if err != repository.ErrNoRows {
		t.Errorf("expected ErrNoRows, got %v", err)
	}
}

func TestQueryPrayerTimesYearDifferentYear(t *testing.T) {
	repo := setupQueryTestDB(t)

	_, err := repo.QueryYear(context.Background(), "SGR01", 2025)
	if err != repository.ErrNoRows {
		t.Errorf("expected ErrNoRows for 2025 (no data), got %v", err)
	}
}
