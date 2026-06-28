package service

import (
	"context"
	"fmt"

	"github.com/shabilullah/gowaktusolat/internal/geo"
	"github.com/shabilullah/gowaktusolat/internal/repository"
)

// PrayerTimeDTO is a domain-level prayer-time value returned by the service layer.
type PrayerTimeDTO struct {
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

// PrayerService defines the use-case contract for prayer-time operations.
type PrayerService interface {
	GetMonth(ctx context.Context, zone string, year, month int) ([]PrayerTimeDTO, error)
	GetDay(ctx context.Context, zone string, year, month, day int) (*PrayerTimeDTO, error)
	GetByGPS(ctx context.Context, lat, lng float64, year, month int) ([]PrayerTimeDTO, string, error)
}

type prayerService struct {
	repo     repository.PrayerTimeRepository
	detector *geo.Detector
}

// NewPrayerService creates a concrete PrayerService.
func NewPrayerService(repo repository.PrayerTimeRepository, detector *geo.Detector) PrayerService {
	return &prayerService{repo: repo, detector: detector}
}

func (s *prayerService) GetMonth(ctx context.Context, zone string, year, month int) ([]PrayerTimeDTO, error) {
	rows, err := s.repo.Query(ctx, zone, year, month)
	if err != nil {
		return nil, fmt.Errorf("service get month: %w", err)
	}

	dtos := make([]PrayerTimeDTO, len(rows))
	for i, r := range rows {
		dtos[i] = PrayerTimeDTO{
			Date:    r.Date,
			Hijri:   r.Hijri,
			Imsak:   r.Imsak,
			Fajr:    r.Fajr,
			Syuruk:  r.Syuruk,
			Dhuha:   r.Dhuha,
			Dhuhr:   r.Dhuhr,
			Asr:     r.Asr,
			Maghrib: r.Maghrib,
			Isha:    r.Isha,
		}
	}
	return dtos, nil
}

func (s *prayerService) GetDay(ctx context.Context, zone string, year, month, day int) (*PrayerTimeDTO, error) {
	rows, err := s.repo.Query(ctx, zone, year, month)
	if err != nil {
		return nil, fmt.Errorf("service get day: %w", err)
	}
	if day < 1 || day > len(rows) {
		return nil, fmt.Errorf("day %d out of range", day)
	}

	r := rows[day-1]
	return &PrayerTimeDTO{
		Date:    r.Date,
		Hijri:   r.Hijri,
		Imsak:   r.Imsak,
		Fajr:    r.Fajr,
		Syuruk:  r.Syuruk,
		Dhuha:   r.Dhuha,
		Dhuhr:   r.Dhuhr,
		Asr:     r.Asr,
		Maghrib: r.Maghrib,
		Isha:    r.Isha,
	}, nil
}

func (s *prayerService) GetByGPS(ctx context.Context, lat, lng float64, year, month int) ([]PrayerTimeDTO, string, error) {
	result, err := s.detector.DetectZone(lat, lng)
	if err != nil {
		return nil, "", fmt.Errorf("service get by gps: %w", err)
	}

	dtos, err := s.GetMonth(ctx, result.Zone, year, month)
	if err != nil {
		return nil, "", err
	}
	return dtos, result.Zone, nil
}
