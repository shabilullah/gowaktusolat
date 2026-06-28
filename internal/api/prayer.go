package api

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/shabilullah/gowaktusolat/internal/api/presenter"
	"github.com/shabilullah/gowaktusolat/internal/repository"
	"github.com/shabilullah/gowaktusolat/internal/service"
)

type PrayerTime struct {
	Service service.PrayerService
}

func (h *PrayerTime) FetchMonth(c fiber.Ctx) error {
	zone := c.Params("zone")
	year, month := parseYearMonth(c)

	dtos, err := h.Service.GetMonth(c.Context(), zone, year, month)
	if errors.Is(err, repository.ErrNoRows) {
		return c.Status(404).JSON(presenter.Message(
			fmt.Sprintf("No data found for zone: %s for %s/%d", zone, strings.ToUpper(monthName(month)), year),
		))
	}
	if err != nil {
		return c.Status(500).JSON(presenter.Message(err.Error()))
	}

	items := make([]presenter.PrayerTimeItem, len(dtos))
	for i, d := range dtos {
		items[i] = presenter.ItemFromDTO(d)
	}

	return c.JSON(presenter.PrayerTimes(items, zone))
}

func (h *PrayerTime) FetchDay(c fiber.Ctx) error {
	zone := c.Params("zone")
	dayStr := c.Params("day")
	day, err := strconv.Atoi(dayStr)
	if err != nil || day < 1 || day > 31 {
		return c.Status(400).JSON(presenter.Message("Invalid day parameter"))
	}

	year, month := parseYearMonth(c)

	dto, err := h.Service.GetDay(c.Context(), zone, year, month, day)
	if errors.Is(err, repository.ErrNoRows) {
		return c.Status(404).JSON(presenter.Message(
			fmt.Sprintf("No data found for zone: %s for %s/%d", zone, strings.ToUpper(monthName(month)), year),
		))
	}
	if err != nil {
		return c.Status(500).JSON(presenter.Message(err.Error()))
	}

	return c.JSON(presenter.PrayerDay(presenter.ItemFromDTO(*dto), zone))
}

func (h *PrayerTime) FetchMonthByGPS(c fiber.Ctx) error {
	latStr := c.Params("lat")
	lngStr := c.Params("long")

	lat, err := strconv.ParseFloat(latStr, 64)
	if err != nil {
		return c.Status(422).JSON(presenter.Message("Invalid latitude"))
	}
	lng, err := strconv.ParseFloat(lngStr, 64)
	if err != nil {
		return c.Status(422).JSON(presenter.Message("Invalid longitude"))
	}

	year, month := parseYearMonth(c)
	dtos, zone, err := h.Service.GetByGPS(c.Context(), lat, lng, year, month)
	if err != nil {
		return c.Status(404).JSON(presenter.Message(err.Error()))
	}

	items := make([]presenter.PrayerTimeItem, len(dtos))
	for i, d := range dtos {
		items[i] = presenter.ItemFromDTO(d)
	}

	return c.JSON(presenter.PrayerTimes(items, zone))
}

func parseYearMonth(c fiber.Ctx) (year, month int) {
	now := time.Now()
	year = now.Year()
	month = int(now.Month())

	if yStr := c.Query("year"); yStr != "" {
		if y, err := strconv.Atoi(yStr); err == nil && y >= 2020 {
			year = y
		}
	}
	if mStr := c.Query("month"); mStr != "" {
		if m, err := strconv.Atoi(mStr); err == nil && m >= 1 && m <= 12 {
			month = m
		}
	}
	return
}

func monthName(m int) string {
	return time.Month(m).String()
}
