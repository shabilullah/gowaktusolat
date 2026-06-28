package scraper

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	json "github.com/goccy/go-json"
	"zombiezen.com/go/sqlite/sqlitex"
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

var httpClient = &http.Client{Timeout: 30 * time.Second}

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

func FetchPrayerTimes(zoneCode string, year int) ([]PrayerTime, error) {
	u := fmt.Sprintf(
		"https://www.e-solat.gov.my/index.php?r=esolatApi%%2Ftakwimsolat&period=duration&zone=%s",
		zoneCode,
	)

	body := strings.NewReader(fmt.Sprintf("datestart=%d-01-01&dateend=%d-12-31", year, year))
	req, err := http.NewRequest("POST", u, body)
	if err != nil {
		return nil, fmt.Errorf("create request for %s: %w", zoneCode, err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch prayer times for %s: %w", zoneCode, err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response for %s: %w", zoneCode, err)
	}

	var esolatResp esolatResponse
	if err := json.Unmarshal(respBytes, &esolatResp); err != nil {
		return nil, fmt.Errorf("unmarshal response for %s: %w", zoneCode, err)
	}

	if esolatResp.Status == "NO_RECORD!" {
		return nil, nil
	}

	var prayerTimes []esolatPrayerTime
	if err := json.Unmarshal(esolatResp.PrayerTime, &prayerTimes); err != nil {
		return nil, fmt.Errorf("unmarshal prayerTime for %s: %w", zoneCode, err)
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

func SavePrayerTimes(pool *sqlitex.Pool, zoneCode string, year int, times []PrayerTime) error {
	conn, err := pool.Take(context.Background())
	if err != nil {
		return fmt.Errorf("take conn: %w", err)
	}
	defer pool.Put(conn)

	const batchSize = 500
	now := time.Now().UTC().Format(time.RFC3339)

	for i := 0; i < len(times); i += batchSize {
		end := i + batchSize
		if end > len(times) {
			end = len(times)
		}
		batch := times[i:end]

		stmt, err := conn.Prepare(`INSERT OR REPLACE INTO prayer_times
			(date, location_code, hijri, imsak, fajr, syuruk, dhuha, dhuhr, asr, maghrib, isha, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`)
		if err != nil {
			return fmt.Errorf("prepare: %w", err)
		}
		for _, pt := range batch {
			stmt.BindText(1, pt.Date)
			stmt.BindText(2, pt.LocationCode)
			stmt.BindText(3, pt.Hijri)
			stmt.BindText(4, pt.Imsak)
			stmt.BindText(5, pt.Fajr)
			stmt.BindText(6, pt.Syuruk)
			stmt.BindText(7, pt.Dhuha)
			stmt.BindText(8, pt.Dhuhr)
			stmt.BindText(9, pt.Asr)
			stmt.BindText(10, pt.Maghrib)
			stmt.BindText(11, pt.Isha)
			stmt.BindText(12, now)
			stmt.BindText(13, now)
			if _, err := stmt.Step(); err != nil {
				stmt.Reset()
				return fmt.Errorf("save prayer time %s/%s: %w", zoneCode, pt.Date, err)
			}
			if err := stmt.Reset(); err != nil {
				return fmt.Errorf("reset: %w", err)
			}
		}
	}
	return nil
}
