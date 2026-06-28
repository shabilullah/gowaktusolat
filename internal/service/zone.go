package service

import (
	"context"
	"fmt"

	"github.com/shabilullah/gowaktusolat/internal/geo"
	"github.com/shabilullah/gowaktusolat/internal/repository"
)

// ZoneDTO is a domain-level zone value returned by the service layer.
type ZoneDTO struct {
	JakimCode string
	Negeri    string
	Daerah    string
}

// ZoneService defines the use-case contract for zone operations.
type ZoneService interface {
	ListAll(ctx context.Context) ([]ZoneDTO, error)
	ListByState(ctx context.Context, statePrefix string) ([]ZoneDTO, error)
	GetByCoordinate(ctx context.Context, lat, lng float64) (*ZoneDTO, error)
	LookupDaerah(ctx context.Context, jakimCode string) (string, error)
}

type zoneService struct {
	repo     repository.ZoneRepository
	detector *geo.Detector
}

// NewZoneService creates a concrete ZoneService.
func NewZoneService(repo repository.ZoneRepository, detector *geo.Detector) ZoneService {
	return &zoneService{repo: repo, detector: detector}
}

func (s *zoneService) ListAll(ctx context.Context) ([]ZoneDTO, error) {
	rows, err := s.repo.ListAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("service list zones: %w", err)
	}

	dtos := make([]ZoneDTO, len(rows))
	for i, r := range rows {
		dtos[i] = ZoneDTO{JakimCode: r.JakimCode, Negeri: r.Negeri, Daerah: r.Daerah}
	}
	return dtos, nil
}

func (s *zoneService) ListByState(ctx context.Context, statePrefix string) ([]ZoneDTO, error) {
	rows, err := s.repo.ListByState(ctx, statePrefix)
	if err != nil {
		return nil, fmt.Errorf("service list zones by state: %w", err)
	}

	dtos := make([]ZoneDTO, len(rows))
	for i, r := range rows {
		dtos[i] = ZoneDTO{JakimCode: r.JakimCode, Negeri: r.Negeri, Daerah: r.Daerah}
	}
	return dtos, nil
}

func (s *zoneService) GetByCoordinate(ctx context.Context, lat, lng float64) (*ZoneDTO, error) {
	result, err := s.detector.DetectZone(lat, lng)
	if err != nil {
		return nil, fmt.Errorf("service get zone by coordinate: %w", err)
	}

	return &ZoneDTO{
		JakimCode: result.Zone,
		Negeri:    result.State,
		Daerah:    result.District,
	}, nil
}

func (s *zoneService) LookupDaerah(ctx context.Context, jakimCode string) (string, error) {
	return s.repo.LookupDaerah(ctx, jakimCode)
}
