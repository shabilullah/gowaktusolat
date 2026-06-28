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

// mockZoneService implements service.ZoneService for handler tests.
type mockZoneService struct {
	listAllFn      func(ctx context.Context) ([]service.ZoneDTO, error)
	listByStateFn  func(ctx context.Context, statePrefix string) ([]service.ZoneDTO, error)
	getByCoordFn   func(ctx context.Context, lat, lng float64) (*service.ZoneDTO, error)
	lookupDaerahFn func(ctx context.Context, jakimCode string) (string, error)
}

func (m *mockZoneService) ListAll(ctx context.Context) ([]service.ZoneDTO, error) {
	return m.listAllFn(ctx)
}

func (m *mockZoneService) ListByState(ctx context.Context, statePrefix string) ([]service.ZoneDTO, error) {
	return m.listByStateFn(ctx, statePrefix)
}

func (m *mockZoneService) GetByCoordinate(ctx context.Context, lat, lng float64) (*service.ZoneDTO, error) {
	return m.getByCoordFn(ctx, lat, lng)
}

func (m *mockZoneService) LookupDaerah(ctx context.Context, jakimCode string) (string, error) {
	return m.lookupDaerahFn(ctx, jakimCode)
}

func TestZonesIndex_OK(t *testing.T) {
	mock := &mockZoneService{
		listAllFn: func(ctx context.Context) ([]service.ZoneDTO, error) {
			return []service.ZoneDTO{
				{JakimCode: "SGR01", Negeri: "Selangor", Daerah: "Gombak"},
				{JakimCode: "JHR01", Negeri: "Johor", Daerah: "Johor Bahru"},
			}, nil
		},
	}

	handler := &Zones{Service: mock}
	app := fiber.New()
	app.Get("/api/zones", handler.Index)

	req := httptest.NewRequest("GET", "/api/zones", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	if resp.StatusCode != 200 {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}

	var body []presenter.ZoneItem
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}

	if len(body) != 2 {
		t.Fatalf("len = %d, want 2", len(body))
	}
	if body[0].JakimCode != "SGR01" {
		t.Errorf("first code = %q, want SGR01", body[0].JakimCode)
	}
	if body[1].Daerah != "Johor Bahru" {
		t.Errorf("second daerah = %q, want Johor Bahru", body[1].Daerah)
	}
}

func TestZonesIndex_Error(t *testing.T) {
	mock := &mockZoneService{
		listAllFn: func(ctx context.Context) ([]service.ZoneDTO, error) {
			return nil, errors.New("database error")
		},
	}

	handler := &Zones{Service: mock}
	app := fiber.New()
	app.Get("/api/zones", handler.Index)

	req := httptest.NewRequest("GET", "/api/zones", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	if resp.StatusCode != 500 {
		t.Errorf("status = %d, want 500", resp.StatusCode)
	}
}

func TestZonesGetByState_OK(t *testing.T) {
	mock := &mockZoneService{
		listByStateFn: func(ctx context.Context, statePrefix string) ([]service.ZoneDTO, error) {
			if statePrefix != "SGR" {
				t.Errorf("statePrefix = %q, want SGR", statePrefix)
			}
			return []service.ZoneDTO{
				{JakimCode: "SGR01", Negeri: "Selangor", Daerah: "Gombak"},
			}, nil
		},
	}

	handler := &Zones{Service: mock}
	app := fiber.New()
	app.Get("/api/zones/:state", handler.GetByState)

	req := httptest.NewRequest("GET", "/api/zones/sgr", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	if resp.StatusCode != 200 {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}

	var body []presenter.ZoneItem
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}

	if len(body) != 1 {
		t.Fatalf("len = %d, want 1", len(body))
	}
}

func TestZonesGetByCoordinate_OK(t *testing.T) {
	mock := &mockZoneService{
		getByCoordFn: func(ctx context.Context, lat, lng float64) (*service.ZoneDTO, error) {
			if lat != 3.068498 || lng != 101.630263 {
				t.Errorf("lat=%f lng=%f, want 3.068498/101.630263", lat, lng)
			}
			return &service.ZoneDTO{
				JakimCode: "SGR01",
				Negeri:    "Selangor",
				Daerah:    "Gombak",
			}, nil
		},
	}

	handler := &Zones{Service: mock}
	app := fiber.New()
	app.Get("/api/zones/:lat/:long", handler.GetByCoordinate)

	req := httptest.NewRequest("GET", "/api/zones/3.068498/101.630263", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	if resp.StatusCode != 200 {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}

	var body presenter.ZoneByCoordinateResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}

	if body.Zone != "SGR01" {
		t.Errorf("zone = %q, want SGR01", body.Zone)
	}
	if body.State != "Selangor" {
		t.Errorf("state = %q, want Selangor", body.State)
	}
}

func TestZonesGetByCoordinate_InvalidLat(t *testing.T) {
	handler := &Zones{Service: &mockZoneService{}}
	app := fiber.New()
	app.Get("/api/zones/:lat/:long", handler.GetByCoordinate)

	req := httptest.NewRequest("GET", "/api/zones/abc/101.630263", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	if resp.StatusCode != 422 {
		t.Errorf("status = %d, want 422", resp.StatusCode)
	}
}

func TestZonesGetByCoordinate_NotFound(t *testing.T) {
	mock := &mockZoneService{
		getByCoordFn: func(ctx context.Context, lat, lng float64) (*service.ZoneDTO, error) {
			return nil, errors.New("no zone found")
		},
	}

	handler := &Zones{Service: mock}
	app := fiber.New()
	app.Get("/api/zones/:lat/:long", handler.GetByCoordinate)

	req := httptest.NewRequest("GET", "/api/zones/1.0/1.0", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	if resp.StatusCode != 404 {
		t.Errorf("status = %d, want 404", resp.StatusCode)
	}
}
