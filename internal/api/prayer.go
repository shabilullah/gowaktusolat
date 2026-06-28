package api

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/shabilullah/gowaktusolat/internal/api/presenter"
	"github.com/shabilullah/gowaktusolat/internal/geo"
	"github.com/shabilullah/gowaktusolat/internal/repository"
	reposqlite "github.com/shabilullah/gowaktusolat/internal/repository/sqlite"
)

type PrayerTime struct {
	Repo     *reposqlite.PrayerTimeRepo
	Detector *geo.Detector
	BasePath string
}

func (h *PrayerTime) FetchMonth(c fiber.Ctx) error {
	zone := c.Params("zone")
	year, month := parseYearMonth(c)

	rows, err := h.Repo.Query(c.Context(), zone, year, month)
	if err == repository.ErrNoRows {
		return c.Status(404).JSON(presenter.Message(
			fmt.Sprintf("No data found for zone: %s for %s/%d", zone, strings.ToUpper(monthName(month)), year),
		))
	}
	if err != nil {
		return c.Status(500).JSON(presenter.Message(err.Error()))
	}

	items := make([]presenter.PrayerTimeItem, len(rows))
	for i, r := range rows {
		items[i] = presenter.ItemFromRow(r)
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

	rows, err := h.Repo.Query(c.Context(), zone, year, month)
	if err == repository.ErrNoRows {
		return c.Status(404).JSON(presenter.Message(
			fmt.Sprintf("No data found for zone: %s for %s/%d", zone, strings.ToUpper(monthName(month)), year),
		))
	}
	if err != nil {
		return c.Status(500).JSON(presenter.Message(err.Error()))
	}

	if day > len(rows) {
		return c.Status(400).JSON(presenter.Message(
			fmt.Sprintf("Day %d out of range for %s/%d", day, monthName(month), year),
		))
	}

	return c.JSON(presenter.PrayerDay(presenter.ItemFromRow(rows[day-1]), zone))
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

	result, err := h.Detector.DetectZone(lat, lng)
	if err != nil {
		return c.Status(404).JSON(presenter.Message(err.Error()))
	}

	zone := result.Zone
	year, month := parseYearMonth(c)

	rows, err := h.Repo.Query(c.Context(), zone, year, month)
	if err == repository.ErrNoRows {
		return c.Status(404).JSON(presenter.Message(
			fmt.Sprintf("No data found for zone: %s for %s/%d", zone, strings.ToUpper(monthName(month)), year),
		))
	}
	if err != nil {
		return c.Status(500).JSON(presenter.Message(err.Error()))
	}

	items := make([]presenter.PrayerTimeItem, len(rows))
	for i, r := range rows {
		items[i] = presenter.ItemFromRow(r)
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
