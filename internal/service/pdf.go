package service

import (
	"github.com/shabilullah/gowaktusolat/internal/pdf"
	"github.com/shabilullah/gowaktusolat/internal/repository"
)

// PDFService defines the use-case contract for PDF generation.
type PDFService interface {
	GenerateMonth(zone, daerah string, year, month int, dtos []PrayerTimeDTO) []byte
	GenerateYear(zone, daerah string, year int, monthly [][]PrayerTimeDTO) []byte
}

type pdfService struct{}

// NewPDFService creates a concrete PDFService.
func NewPDFService() PDFService {
	return &pdfService{}
}

func (s *pdfService) GenerateMonth(zone, daerah string, year, month int, dtos []PrayerTimeDTO) []byte {
	rows := dtosToRows(dtos)
	return pdf.GenerateMonth(zone, daerah, year, month, rows)
}

func (s *pdfService) GenerateYear(zone, daerah string, year int, monthly [][]PrayerTimeDTO) []byte {
	monthlyRows := make([][]repository.PrayerTimeRow, 13)
	for m := 1; m <= 12; m++ {
		monthlyRows[m] = dtosToRows(monthly[m])
	}
	return pdf.GenerateYear(zone, daerah, year, monthlyRows)
}

func dtosToRows(dtos []PrayerTimeDTO) []repository.PrayerTimeRow {
	rows := make([]repository.PrayerTimeRow, len(dtos))
	for i, d := range dtos {
		rows[i] = repository.PrayerTimeRow{
			Date:    d.Date,
			Hijri:   d.Hijri,
			Imsak:   d.Imsak,
			Fajr:    d.Fajr,
			Syuruk:  d.Syuruk,
			Dhuha:   d.Dhuha,
			Dhuhr:   d.Dhuhr,
			Asr:     d.Asr,
			Maghrib: d.Maghrib,
			Isha:    d.Isha,
		}
	}
	return rows
}
