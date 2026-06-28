package service

import (
	"context"
	"testing"

	"github.com/shabilullah/gowaktusolat/internal/repository"
)

// mockPrayerRepo implements repository.PrayerTimeRepository for testing.
type mockPrayerRepo struct {
	queryFn     func(ctx context.Context, zone string, year, month int) ([]repository.PrayerTimeRow, error)
	queryYearFn func(ctx context.Context, zone string, year int) ([]repository.PrayerTimeRow, error)
	scrapedYrFn func(ctx context.Context) (map[int]bool, error)
}

func (m *mockPrayerRepo) Query(ctx context.Context, zone string, year, month int) ([]repository.PrayerTimeRow, error) {
	return m.queryFn(ctx, zone, year, month)
}

func (m *mockPrayerRepo) QueryYear(ctx context.Context, zone string, year int) ([]repository.PrayerTimeRow, error) {
	return m.queryYearFn(ctx, zone, year)
}

func (m *mockPrayerRepo) ScrapedYears(ctx context.Context) (map[int]bool, error) {
	return m.scrapedYrFn(ctx)
}

func TestPrayerService_GetMonth(t *testing.T) {
	repo := &mockPrayerRepo{
		queryFn: func(ctx context.Context, zone string, year, month int) ([]repository.PrayerTimeRow, error) {
			if zone != "SGR01" || year != 2026 || month != 6 {
				t.Errorf("unexpected args: zone=%s year=%d month=%d", zone, year, month)
			}
			return []repository.PrayerTimeRow{
				{Date: "2026-06-01", Fajr: "05:49:00", Dhuhr: "13:14:00"},
				{Date: "2026-06-02", Fajr: "05:49:00", Dhuhr: "13:14:00"},
			}, nil
		},
	}

	svc := NewPrayerService(repo, nil)
	dtos, err := svc.GetMonth(context.Background(), "SGR01", 2026, 6)
	if err != nil {
		t.Fatalf("GetMonth failed: %v", err)
	}

	if len(dtos) != 2 {
		t.Fatalf("got %d dtos, want 2", len(dtos))
	}
	if dtos[0].Date != "2026-06-01" {
		t.Errorf("first date = %s, want 2026-06-01", dtos[0].Date)
	}
	if dtos[0].Fajr != "05:49:00" {
		t.Errorf("first fajr = %s, want 05:49:00", dtos[0].Fajr)
	}
}

func TestPrayerService_GetMonth_EmptyResult(t *testing.T) {
	repo := &mockPrayerRepo{
		queryFn: func(ctx context.Context, zone string, year, month int) ([]repository.PrayerTimeRow, error) {
			return nil, repository.ErrNoRows
		},
	}

	svc := NewPrayerService(repo, nil)
	_, err := svc.GetMonth(context.Background(), "XXXXX", 2026, 6)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestPrayerService_GetDay(t *testing.T) {
	repo := &mockPrayerRepo{
		queryFn: func(ctx context.Context, zone string, year, month int) ([]repository.PrayerTimeRow, error) {
			return []repository.PrayerTimeRow{
				{Date: "2026-06-01", Fajr: "05:49:00"},
				{Date: "2026-06-02", Fajr: "05:50:00"},
				{Date: "2026-06-03", Fajr: "05:51:00"},
			}, nil
		},
	}

	svc := NewPrayerService(repo, nil)
	dto, err := svc.GetDay(context.Background(), "SGR01", 2026, 6, 2)
	if err != nil {
		t.Fatalf("GetDay failed: %v", err)
	}
	if dto.Date != "2026-06-02" {
		t.Errorf("date = %s, want 2026-06-02", dto.Date)
	}
	if dto.Fajr != "05:50:00" {
		t.Errorf("fajr = %s, want 05:50:00", dto.Fajr)
	}
}

func TestPrayerService_GetDay_OutOfRange(t *testing.T) {
	repo := &mockPrayerRepo{
		queryFn: func(ctx context.Context, zone string, year, month int) ([]repository.PrayerTimeRow, error) {
			return []repository.PrayerTimeRow{{Date: "2026-06-01"}}, nil
		},
	}

	svc := NewPrayerService(repo, nil)
	_, err := svc.GetDay(context.Background(), "SGR01", 2026, 6, 5)
	if err == nil {
		t.Fatal("expected error for out-of-range day, got nil")
	}
}
