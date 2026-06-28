package api

import (
	"fmt"
	"strings"

	"github.com/gofiber/fiber/v3"
	"github.com/shabilullah/gowaktusolat/internal/api/presenter"
	"github.com/shabilullah/gowaktusolat/internal/service"
)

type JadualSolat struct {
	PrayerService service.PrayerService
	ZoneService   service.ZoneService
	PDFService    service.PDFService
}

func (h *JadualSolat) FetchMonth(c fiber.Ctx) error {
	zone := c.Params("zone")
	year, month := parseYearMonth(c)

	if c.Query("month") == "" {
		return h.fetchYear(c, zone, year)
	}

	return h.fetchSingleMonth(c, zone, year, month)
}

func (h *JadualSolat) fetchSingleMonth(c fiber.Ctx, zone string, year, month int) error {
	dtos, err := h.PrayerService.GetMonth(c.Context(), zone, year, month)
	if err != nil {
		return c.Status(500).JSON(presenter.Message(err.Error()))
	}
	if len(dtos) == 0 {
		return c.Status(404).JSON(presenter.Message(
			fmt.Sprintf("No data found for zone: %s for %s/%d", zone, strings.ToUpper(monthName(month)), year),
		))
	}

	daerah, _ := h.ZoneService.LookupDaerah(c.Context(), zone)
	pdfBytes := h.PDFService.GenerateMonth(zone, daerah, year, month, dtos)

	c.Set("Content-Type", "application/pdf")
	c.Set("Content-Disposition", fmt.Sprintf("inline; filename=\"jadual-solat-%s-%d-%02d.pdf\"", zone, year, month))
	return c.Send(pdfBytes)
}

func (h *JadualSolat) fetchYear(c fiber.Ctx, zone string, year int) error {
	monthly := make([][]service.PrayerTimeDTO, 13)
	for month := 1; month <= 12; month++ {
		dtos, err := h.PrayerService.GetMonth(c.Context(), zone, year, month)
		if err != nil {
			return c.Status(500).JSON(presenter.Message(err.Error()))
		}
		monthly[month] = dtos
	}

	daerah, _ := h.ZoneService.LookupDaerah(c.Context(), zone)
	pdfBytes := h.PDFService.GenerateYear(zone, daerah, year, monthly)

	c.Set("Content-Type", "application/pdf")
	c.Set("Content-Disposition", fmt.Sprintf("inline; filename=\"jadual-solat-%s-%d.pdf\"", zone, year))
	return c.Send(pdfBytes)
}
