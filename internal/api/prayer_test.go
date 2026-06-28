package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/shabilullah/gowaktusolat/internal/api/presenter"
	"github.com/shabilullah/gowaktusolat/internal/service"
)

// mockPrayerService implements service.PrayerService for handler tests.
type mockPrayerService struct {
	getMonthFn func(ctx context.Context, zone string, year, month int) ([]service.PrayerTimeDTO, error)
	getDayFn   func(ctx context.Context, zone string, year, month, day int) (*service.PrayerTimeDTO, error)
	getByGPSFn func(ctx context.Context, lat, lng float64, year, month int) ([]service.PrayerTimeDTO, string, error)
}

func (m *mockPrayerService) GetMonth(ctx context.Context, zone string, year, month int) ([]service.PrayerTimeDTO, error) {
	return m.getMonthFn(ctx, zone, year, month)
}

func (m *mockPrayerService) GetDay(ctx context.Context, zone string, year, month, day int) (*service.PrayerTimeDTO, error) {
	return m.getDayFn(ctx, zone, year, month, day)
}

func (m *mockPrayerService) GetByGPS(ctx context.Context, lat, lng float64, year, month int) ([]service.PrayerTimeDTO, string, error) {
	return m.getByGPSFn(ctx, lat, lng, year, month)
}

func TestFetchMonth_OK(t *testing.T) {
	mock := &mockPrayerService{
		getMonthFn: func(ctx context.Context, zone string, year, month int) ([]service.PrayerTimeDTO, error) {
			if zone != "SGR01" || year != 2026 || month != 6 {
				t.Errorf("unexpected args: zone=%s year=%d month=%d", zone, year, month)
			}
			return []service.PrayerTimeDTO{
				{Date: "2026-06-01", Fajr: "05:49:00", Dhuhr: "13:14:00"},
				{Date: "2026-06-02", Fajr: "05:49:00", Dhuhr: "13:14:00"},
			}, nil
		},
	}

	handler := &PrayerTime{Service: mock}
	app := fiber.New()
	app.Get("/api/solat/:zone", handler.FetchMonth)

	req := httptest.NewRequest("GET", "/api/solat/SGR01?month=6&year=2026", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	if resp.StatusCode != 200 {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}

	var body presenter.PrayerTimesResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}

	if body.Status != "OK!" {
		t.Errorf("status = %q, want OK!", body.Status)
	}
	if body.PeriodType != "month" {
		t.Errorf("periodType = %q, want month", body.PeriodType)
	}
	if body.Zone != "SGR01" {
		t.Errorf("zone = %q, want SGR01", body.Zone)
	}
	if len(body.PrayerTime) != 2 {
		t.Fatalf("prayerTime len = %d, want 2", len(body.PrayerTime))
	}
	if body.PrayerTime[0].Date != "01-Jun-2026" {
		t.Errorf("date = %q, want 01-Jun-2026", body.PrayerTime[0].Date)
	}
}

func TestFetchMonth_DefaultYearMonth(t *testing.T) {
	mock := &mockPrayerService{
		getMonthFn: func(ctx context.Context, zone string, year, month int) ([]service.PrayerTimeDTO, error) {
			// Year/month should default to current time when not passed
			if year == 0 || month == 0 {
				t.Error("year/month should have defaults")
			}
			return []service.PrayerTimeDTO{
				{Date: "2026-06-01", Fajr: "05:49:00"},
			}, nil
		},
	}

	handler := &PrayerTime{Service: mock}
	app := fiber.New()
	app.Get("/api/solat/:zone", handler.FetchMonth)

	req := httptest.NewRequest("GET", "/api/solat/SGR01", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	if resp.StatusCode != 200 {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
}

func TestFetchMonth_NoData(t *testing.T) {
	mock := &mockPrayerService{
		getMonthFn: func(ctx context.Context, zone string, year, month int) ([]service.PrayerTimeDTO, error) {
			return nil, errors.New("no rows in result set") // service wraps ErrNoRows
		},
	}

	handler := &PrayerTime{Service: mock}
	app := fiber.New()
	app.Get("/api/solat/:zone", handler.FetchMonth)

	req := httptest.NewRequest("GET", "/api/solat/XXXXX?month=6&year=2026", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	// Service errors that aren't ErrNoRows → 500
	if resp.StatusCode != 500 {
		t.Errorf("status = %d, want 500", resp.StatusCode)
	}
}

func TestFetchDay_OK(t *testing.T) {
	mock := &mockPrayerService{
		getDayFn: func(ctx context.Context, zone string, year, month, day int) (*service.PrayerTimeDTO, error) {
			if day != 15 {
				t.Errorf("day = %d, want 15", day)
			}
			return &service.PrayerTimeDTO{
				Date: "2026-06-15", Fajr: "05:50:00", Dhuhr: "13:16:00",
			}, nil
		},
	}

	handler := &PrayerTime{Service: mock}
	app := fiber.New()
	app.Get("/api/solat/:zone/:day", handler.FetchDay)

	req := httptest.NewRequest("GET", "/api/solat/SGR01/15?month=6&year=2026", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	if resp.StatusCode != 200 {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}

	var body presenter.PrayerDayResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}

	if body.PeriodType != "day" {
		t.Errorf("periodType = %q, want day", body.PeriodType)
	}
	if body.PrayerTime.Date != "15-Jun-2026" {
		t.Errorf("date = %q, want 15-Jun-2026", body.PrayerTime.Date)
	}
}

func TestFetchDay_InvalidDay(t *testing.T) {
	handler := &PrayerTime{Service: &mockPrayerService{}}
	app := fiber.New()
	app.Get("/api/solat/:zone/:day", handler.FetchDay)

	tests := []struct {
		day string
	}{
		{"abc"},
		{"0"},
		{"32"},
	}

	for _, tt := range tests {
		req := httptest.NewRequest("GET", "/api/solat/SGR01/"+tt.day+"?month=6&year=2026", nil)
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		if resp.StatusCode != 400 {
			t.Errorf("day=%s: status = %d, want 400", tt.day, resp.StatusCode)
		}
	}
}

func TestFetchMonthByGPS_OK(t *testing.T) {
	mock := &mockPrayerService{
		getByGPSFn: func(ctx context.Context, lat, lng float64, year, month int) ([]service.PrayerTimeDTO, string, error) {
			if lat != 3.068498 || lng != 101.630263 {
				t.Errorf("lat=%f lng=%f, want 3.068498/101.630263", lat, lng)
			}
			return []service.PrayerTimeDTO{
				{Date: "2026-06-01", Fajr: "05:49:00"},
			}, "SGR01", nil
		},
	}

	handler := &PrayerTime{Service: mock}
	app := fiber.New()
	app.Get("/api/solat/gps/:lat/:long", handler.FetchMonthByGPS)

	req := httptest.NewRequest("GET", "/api/solat/gps/3.068498/101.630263?month=6&year=2026", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	if resp.StatusCode != 200 {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}

	var body presenter.PrayerTimesResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}

	if body.Zone != "SGR01" {
		t.Errorf("zone = %q, want SGR01", body.Zone)
	}
}

func TestFetchMonthByGPS_InvalidLat(t *testing.T) {
	mock := &mockPrayerService{}
	handler := &PrayerTime{Service: mock}
	app := fiber.New()
	app.Get("/api/solat/gps/:lat/:long", handler.FetchMonthByGPS)

	req := httptest.NewRequest("GET", "/api/solat/gps/abc/101.630263?month=6&year=2026", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	if resp.StatusCode != 422 {
		t.Errorf("status = %d, want 422", resp.StatusCode)
	}
}

func TestFetchMonthByGPS_InvalidLng(t *testing.T) {
	mock := &mockPrayerService{}
	handler := &PrayerTime{Service: mock}
	app := fiber.New()
	app.Get("/api/solat/gps/:lat/:long", handler.FetchMonthByGPS)

	req := httptest.NewRequest("GET", "/api/solat/gps/3.068498/xyz?month=6&year=2026", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	if resp.StatusCode != 422 {
		t.Errorf("status = %d, want 422", resp.StatusCode)
	}
}
