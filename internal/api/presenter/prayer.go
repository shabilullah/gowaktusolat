package presenter

import (
	"time"

	"github.com/shabilullah/gowaktusolat/internal/service"
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

// ItemFromDTO converts a service-layer DTO into a JSON-ready prayer time item.
func ItemFromDTO(d service.PrayerTimeDTO) PrayerTimeItem {
	t, err := time.Parse("2006-01-02", d.Date)
	dateStr := d.Date
	dayStr := ""
	if err == nil {
		dateStr = t.Format("02-Jan-2006")
		dayStr = t.Format("Monday")
	}

	return PrayerTimeItem{
		Hijri:   nilIfEmpty(d.Hijri),
		Date:    dateStr,
		Day:     dayStr,
		Imsak:   nilIfEmpty(d.Imsak),
		Fajr:    nilIfEmpty(d.Fajr),
		Syuruk:  nilIfEmpty(d.Syuruk),
		Dhuha:   nilIfEmpty(d.Dhuha),
		Dhuhr:   nilIfEmpty(d.Dhuhr),
		Asr:     nilIfEmpty(d.Asr),
		Maghrib: nilIfEmpty(d.Maghrib),
		Isha:    nilIfEmpty(d.Isha),
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
