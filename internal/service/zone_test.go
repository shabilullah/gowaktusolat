package service

import (
	"context"
	"testing"

	"github.com/shabilullah/gowaktusolat/internal/repository"
)

type mockZoneRepo struct {
	listAllFn      func(ctx context.Context) ([]repository.ZoneRow, error)
	listByStateFn  func(ctx context.Context, statePrefix string) ([]repository.ZoneRow, error)
	lookupDaerahFn func(ctx context.Context, jakimCode string) (string, error)
}

func (m *mockZoneRepo) ListAll(ctx context.Context) ([]repository.ZoneRow, error) {
	return m.listAllFn(ctx)
}

func (m *mockZoneRepo) ListByState(ctx context.Context, statePrefix string) ([]repository.ZoneRow, error) {
	return m.listByStateFn(ctx, statePrefix)
}

func (m *mockZoneRepo) LookupDaerah(ctx context.Context, jakimCode string) (string, error) {
	return m.lookupDaerahFn(ctx, jakimCode)
}

func TestZoneService_ListAll(t *testing.T) {
	repo := &mockZoneRepo{
		listAllFn: func(ctx context.Context) ([]repository.ZoneRow, error) {
			return []repository.ZoneRow{
				{JakimCode: "SGR01", Negeri: "Selangor", Daerah: "Gombak"},
				{JakimCode: "JHR01", Negeri: "Johor", Daerah: "Johor Bahru"},
			}, nil
		},
	}

	svc := NewZoneService(repo, nil)
	dtos, err := svc.ListAll(context.Background())
	if err != nil {
		t.Fatalf("ListAll failed: %v", err)
	}

	if len(dtos) != 2 {
		t.Fatalf("got %d zones, want 2", len(dtos))
	}
	if dtos[0].JakimCode != "SGR01" {
		t.Errorf("first code = %s, want SGR01", dtos[0].JakimCode)
	}
	if dtos[1].Daerah != "Johor Bahru" {
		t.Errorf("second daerah = %s, want Johor Bahru", dtos[1].Daerah)
	}
}

func TestZoneService_ListByState(t *testing.T) {
	repo := &mockZoneRepo{
		listByStateFn: func(ctx context.Context, statePrefix string) ([]repository.ZoneRow, error) {
			if statePrefix != "SGR" {
				t.Errorf("statePrefix = %s, want SGR", statePrefix)
			}
			return []repository.ZoneRow{
				{JakimCode: "SGR01", Negeri: "Selangor", Daerah: "Gombak"},
			}, nil
		},
	}

	svc := NewZoneService(repo, nil)
	dtos, err := svc.ListByState(context.Background(), "SGR")
	if err != nil {
		t.Fatalf("ListByState failed: %v", err)
	}

	if len(dtos) != 1 {
		t.Fatalf("got %d zones, want 1", len(dtos))
	}
}

func TestZoneService_LookupDaerah(t *testing.T) {
	repo := &mockZoneRepo{
		lookupDaerahFn: func(ctx context.Context, jakimCode string) (string, error) {
			if jakimCode != "SGR01" {
				t.Errorf("jakimCode = %s, want SGR01", jakimCode)
			}
			return "Gombak", nil
		},
	}

	svc := NewZoneService(repo, nil)
	daerah, err := svc.LookupDaerah(context.Background(), "SGR01")
	if err != nil {
		t.Fatalf("LookupDaerah failed: %v", err)
	}
	if daerah != "Gombak" {
		t.Errorf("daerah = %s, want Gombak", daerah)
	}
}
