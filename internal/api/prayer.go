package api

import (
	"database/sql"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/shabilullah/gowaktusolat/internal/db"
	"github.com/shabilullah/gowaktusolat/internal/geo"
)

type PrayerTime struct {
	DB       *sql.DB
	Detector *geo.Detector
	BasePath string
}

func (h *PrayerTime) FetchMonth(c fiber.Ctx) error {
	zone := c.Params("zone")
	year, month := parseYearMonth(c)

	rows, err := db.QueryPrayerTimes(h.DB, zone, year, month)
	if err == sql.ErrNoRows {
		return c.Status(404).JSON(fiber.Map{
			"message": fmt.Sprintf("No data found for zone: %s for %s/%d", zone, strings.ToUpper(monthName(month)), year),
		})
	}
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"message": err.Error()})
	}

	prayerTime := make([]fiber.Map, len(rows))
	for i, r := range rows {
		prayerTime[i] = mapPrayerTimeRow(r)
	}

	return c.JSON(fiber.Map{
		"prayerTime": prayerTime,
		"status":     "OK!",
		"serverTime": time.Now().Format("2006-01-02 15:04:05"),
		"periodType": "month",
		"lang":       "",
		"zone":       zone,
		"bearing":    "",
	})
}

func (h *PrayerTime) FetchDay(c fiber.Ctx) error {
	zone := c.Params("zone")
	dayStr := c.Params("day")
	day, err := strconv.Atoi(dayStr)
	if err != nil || day < 1 || day > 31 {
		return c.Status(400).JSON(fiber.Map{"message": "Invalid day parameter"})
	}

	year, month := parseYearMonth(c)

	rows, err := db.QueryPrayerTimes(h.DB, zone, year, month)
	if err == sql.ErrNoRows {
		return c.Status(404).JSON(fiber.Map{
			"message": fmt.Sprintf("No data found for zone: %s for %s/%d", zone, strings.ToUpper(monthName(month)), year),
		})
	}
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"message": err.Error()})
	}

	if day > len(rows) {
		return c.Status(400).JSON(fiber.Map{"message": fmt.Sprintf("Day %d out of range for %s/%d", day, monthName(month), year)})
	}

	return c.JSON(fiber.Map{
		"prayerTime": mapPrayerTimeRow(rows[day-1]),
		"status":     "OK!",
		"serverTime": time.Now().Format("2006-01-02 15:04:05"),
		"periodType": "day",
		"lang":       "",
		"zone":       zone,
		"bearing":    "",
	})
}

func (h *PrayerTime) FetchMonthByGPS(c fiber.Ctx) error {
	latStr := c.Params("lat")
	lngStr := c.Params("long")

	lat, err := strconv.ParseFloat(latStr, 64)
	if err != nil {
		return c.Status(422).JSON(fiber.Map{"message": "Invalid latitude"})
	}
	lng, err := strconv.ParseFloat(lngStr, 64)
	if err != nil {
		return c.Status(422).JSON(fiber.Map{"message": "Invalid longitude"})
	}

	result, err := h.Detector.DetectZone(lat, lng)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"message": err.Error()})
	}

	zone := result.Zone
	year, month := parseYearMonth(c)

	rows, err := db.QueryPrayerTimes(h.DB, zone, year, month)
	if err == sql.ErrNoRows {
		return c.Status(404).JSON(fiber.Map{
			"message": fmt.Sprintf("No data found for zone: %s for %s/%d", zone, strings.ToUpper(monthName(month)), year),
		})
	}
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"message": err.Error()})
	}

	prayerTime := make([]fiber.Map, len(rows))
	for i, r := range rows {
		prayerTime[i] = mapPrayerTimeRow(r)
	}

	return c.JSON(fiber.Map{
		"prayerTime": prayerTime,
		"status":     "OK!",
		"serverTime": time.Now().Format("2006-01-02 15:04:05"),
		"periodType": "month",
		"lang":       "",
		"zone":       zone,
		"bearing":    "",
	})
}

func mapPrayerTimeRow(r db.PrayerTimeRow) fiber.Map {
	t, err := time.Parse("2006-01-02", r.Date)
	dateStr := r.Date
	dayStr := ""
	if err == nil {
		dateStr = t.Format("02-Jan-2006")
		dayStr = t.Format("Monday")
	}

	return fiber.Map{
		"hijri":   nilIfEmpty(r.Hijri),
		"date":    dateStr,
		"day":     dayStr,
		"imsak":   nilIfEmpty(r.Imsak),
		"fajr":    nilIfEmpty(r.Fajr),
		"syuruk":  nilIfEmpty(r.Syuruk),
		"dhuha":   nilIfEmpty(r.Dhuha),
		"dhuhr":   nilIfEmpty(r.Dhuhr),
		"asr":     nilIfEmpty(r.Asr),
		"maghrib": nilIfEmpty(r.Maghrib),
		"isha":    nilIfEmpty(r.Isha),
	}
}

func nilIfEmpty(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
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
