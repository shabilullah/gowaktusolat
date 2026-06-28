package presenter

import (
	"time"

	"github.com/shabilullah/gowaktusolat/internal/repository"
)

// PrayerTimeItem is a single prayer-time entry in JSON responses.
type PrayerTimeItem struct {
	Hijri   any    `json:"hijri"`
	Date    string `json:"date"`
	Day     string `json:"day"`
	Imsak   any    `json:"imsak"`
	Fajr    any    `json:"fajr"`
	Syuruk  any    `json:"syuruk"`
	Dhuha   any    `json:"dhuha"`
	Dhuhr   any    `json:"dhuhr"`
	Asr     any    `json:"asr"`
	Maghrib any    `json:"maghrib"`
	Isha    any    `json:"isha"`
}

// PrayerTimesResponse is the JSON envelope for a month (list) response.
type PrayerTimesResponse struct {
	PrayerTime []PrayerTimeItem `json:"prayerTime"`
	Status     string           `json:"status"`
	ServerTime string           `json:"serverTime"`
	PeriodType string           `json:"periodType"`
	Lang       string           `json:"lang"`
	Zone       string           `json:"zone"`
	Bearing    string           `json:"bearing"`
}

// PrayerDayResponse is the JSON envelope for a single-day response.
type PrayerDayResponse struct {
	PrayerTime PrayerTimeItem `json:"prayerTime"`
	Status     string         `json:"status"`
	ServerTime string         `json:"serverTime"`
	PeriodType string         `json:"periodType"`
	Lang       string         `json:"lang"`
	Zone       string         `json:"zone"`
	Bearing    string         `json:"bearing"`
}

// ItemFromRow converts a DB prayer-time row into a JSON-ready item.
func ItemFromRow(r repository.PrayerTimeRow) PrayerTimeItem {
	t, err := time.Parse("2006-01-02", r.Date)
	dateStr := r.Date
	dayStr := ""
	if err == nil {
		dateStr = t.Format("02-Jan-2006")
		dayStr = t.Format("Monday")
	}

	return PrayerTimeItem{
		Hijri:   nilIfEmpty(r.Hijri),
		Date:    dateStr,
		Day:     dayStr,
		Imsak:   nilIfEmpty(r.Imsak),
		Fajr:    nilIfEmpty(r.Fajr),
		Syuruk:  nilIfEmpty(r.Syuruk),
		Dhuha:   nilIfEmpty(r.Dhuha),
		Dhuhr:   nilIfEmpty(r.Dhuhr),
		Asr:     nilIfEmpty(r.Asr),
		Maghrib: nilIfEmpty(r.Maghrib),
		Isha:    nilIfEmpty(r.Isha),
	}
}

// PrayerTimes builds a month response envelope.
func PrayerTimes(items []PrayerTimeItem, zone string) PrayerTimesResponse {
	return PrayerTimesResponse{
		PrayerTime: items,
		Status:     "OK!",
		ServerTime: time.Now().Format("2006-01-02 15:04:05"),
		PeriodType: "month",
		Zone:       zone,
	}
}

// PrayerDay builds a single-day response envelope.
func PrayerDay(item PrayerTimeItem, zone string) PrayerDayResponse {
	return PrayerDayResponse{
		PrayerTime: item,
		Status:     "OK!",
		ServerTime: time.Now().Format("2006-01-02 15:04:05"),
		PeriodType: "day",
		Zone:       zone,
	}
}

func nilIfEmpty(s string) any {
	if s == "" {
		return nil
	}
	return s
}
