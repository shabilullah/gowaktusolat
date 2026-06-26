package scraper

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

var malayMonths = map[string]string{
	"Jan":  "Jan",
	"Feb":  "Feb",
	"Mac":  "Mar",
	"Apr":  "Apr",
	"Mei":  "May",
	"Jun":  "Jun",
	"Jul":  "Jul",
	"Ogos": "Aug",
	"Sep":  "Sep",
	"Okt":  "Oct",
	"Nov":  "Nov",
	"Dis":  "Dec",
}

type esolatResponse struct {
	PrayerTime json.RawMessage `json:"prayerTime"`
	Status     string          `json:"status"`
}

type esolatPrayerTime struct {
	Date    string `json:"date"`
	Day     string `json:"day"`
	Hijri   string `json:"hijri"`
	Imsak   string `json:"imsak"`
	Fajr    string `json:"fajr"`
	Syuruk  string `json:"syuruk"`
	Dhuha   string `json:"dhuha"`
	Dhuhr   string `json:"dhuhr"`
	Asr     string `json:"asr"`
	Maghrib string `json:"maghrib"`
	Isha    string `json:"isha"`
}

type PrayerTime struct {
	Date         string
	Hijri        string
	Imsak        string
	Fajr         string
	Syuruk       string
	Dhuha        string
	Dhuhr        string
	Asr          string
	Maghrib      string
	Isha         string
	LocationCode string
}

func FetchPrayerTimes(ctx context.Context, zoneCode string, year int) ([]PrayerTime, error) {
	u := fmt.Sprintf(
		"https://www.e-solat.gov.my/index.php?r=esolatApi%%2Ftakwimsolat&period=duration&zone=%s",
		zoneCode,
	)
	formData := url.Values{
		"datestart": {fmt.Sprintf("%d-01-01", year)},
		"dateend":   {fmt.Sprintf("%d-12-31", year)},
	}

	var lastErr error
	for attempt := 0; attempt < 3; attempt++ {
		if attempt > 0 {
			select {
			case <-time.After(2 * time.Second):
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodPost, u, strings.NewReader(formData.Encode()))
		if err != nil {
			return nil, fmt.Errorf("create request: %w", err)
		}
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		client := &http.Client{Timeout: 30 * time.Second}
		resp, err := client.Do(req)
		if err != nil {
			lastErr = err
			continue
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			lastErr = err
			continue
		}

		var esolatResp esolatResponse
		if err := json.Unmarshal(body, &esolatResp); err != nil {
			lastErr = fmt.Errorf("unmarshal response: %w", err)
			continue
		}

		// Handle NO_RECORD — zone has no data for this year
		if esolatResp.Status == "NO_RECORD!" {
			return nil, nil
		}

		var prayerTimes []esolatPrayerTime
		if err := json.Unmarshal(esolatResp.PrayerTime, &prayerTimes); err != nil {
			lastErr = fmt.Errorf("unmarshal prayerTime: %w", err)
			continue
		}

		times := make([]PrayerTime, 0, len(prayerTimes))
		for _, pt := range prayerTimes {
			times = append(times, PrayerTime{
				Date:         parseDate(pt.Date),
				Hijri:        pt.Hijri,
				Imsak:        pt.Imsak,
				Fajr:         pt.Fajr,
				Syuruk:       pt.Syuruk,
				Dhuha:        pt.Dhuha,
				Dhuhr:        pt.Dhuhr,
				Asr:          pt.Asr,
				Maghrib:      pt.Maghrib,
				Isha:         pt.Isha,
				LocationCode: zoneCode,
			})
		}
		return times, nil
	}
	return nil, fmt.Errorf("fetch prayer times for %s after 3 attempts: %w", zoneCode, lastErr)
}

func parseDate(malayDate string) string {
	for malay, eng := range malayMonths {
		malayDate = strings.Replace(malayDate, malay, eng, 1)
	}
	t, err := time.Parse("02-Jan-2006", malayDate)
	if err != nil {
		return malayDate
	}
	return t.Format("2006-01-02")
}

func SavePrayerTimes(db *sql.DB, zoneCode string, year int, times []PrayerTime) error {
	const batchSize = 500
	now := time.Now().UTC().Format(time.RFC3339)

	for i := 0; i < len(times); i += batchSize {
		end := i + batchSize
		if end > len(times) {
			end = len(times)
		}
		batch := times[i:end]

		stmt := `INSERT OR REPLACE INTO prayer_times
			(date, location_code, hijri, imsak, fajr, syuruk, dhuha, dhuhr, asr, maghrib, isha, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
		for _, pt := range batch {
			if _, err := db.Exec(stmt,
				pt.Date, pt.LocationCode, pt.Hijri,
				pt.Imsak, pt.Fajr, pt.Syuruk, pt.Dhuha,
				pt.Dhuhr, pt.Asr, pt.Maghrib, pt.Isha,
				now, now,
			); err != nil {
				return fmt.Errorf("save prayer time %s/%s: %w", zoneCode, pt.Date, err)
			}
		}
	}
	return nil
}
