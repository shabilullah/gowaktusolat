package api

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/shabilullah/gowaktusolat/internal/api/presenter"
	"github.com/shabilullah/gowaktusolat/internal/db"
	"github.com/shabilullah/gowaktusolat/internal/pdf"
	"zombiezen.com/go/sqlite"
	"zombiezen.com/go/sqlite/sqlitex"
)

type JadualSolat struct {
	Pool *sqlitex.Pool
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
	rows, err := db.QueryPrayerTimes(h.Pool, c.Context(), zone, year, month)
	if err != nil {
		return c.Status(500).JSON(presenter.Message(err.Error()))
	}
	if len(rows) == 0 {
		return c.Status(404).JSON(presenter.Message(
			fmt.Sprintf("No data found for zone: %s for %s/%d", zone, strings.ToUpper(monthName(month)), year),
		))
	}

	daerah := lookupDaerah(h.Pool, zone)
	pdfBytes := pdf.GenerateMonth(zone, daerah, year, month, rows)

	c.Set("Content-Type", "application/pdf")
	c.Set("Content-Disposition", fmt.Sprintf("inline; filename=\"jadual-solat-%s-%d-%02d.pdf\"", zone, year, month))
	return c.Send(pdfBytes)
}

func (h *JadualSolat) fetchYear(c fiber.Ctx, zone string, year int) error {
	allRows, err := db.QueryPrayerTimesYear(h.Pool, c.Context(), zone, year)
	if err != nil {
		return c.Status(500).JSON(presenter.Message(err.Error()))
	}

	monthly := make([][]db.PrayerTimeRow, 13)
	for _, row := range allRows {
		t, err := time.Parse("2006-01-02", row.Date)
		if err != nil {
			continue
		}
		m := t.Month()
		monthly[m] = append(monthly[m], row)
	}

	daerah := lookupDaerah(h.Pool, zone)
	pdfBytes := pdf.GenerateYear(zone, daerah, year, monthly)

	c.Set("Content-Type", "application/pdf")
	c.Set("Content-Disposition", fmt.Sprintf("inline; filename=\"jadual-solat-%s-%d.pdf\"", zone, year))
	return c.Send(pdfBytes)
}

func lookupDaerah(pool *sqlitex.Pool, zone string) string {
	conn, err := pool.Take(context.Background())
	if err != nil {
		return ""
	}
	defer pool.Put(conn)

	var daerah string
	if err := sqlitex.ExecuteTransient(conn,
		"SELECT daerah FROM prayer_zones WHERE jakim_code = ?",
		&sqlitex.ExecOptions{
			Args: []interface{}{zone},
			ResultFunc: func(stmt *sqlite.Stmt) error {
				daerah = stmt.ColumnText(0)
				return nil
			},
		}); err != nil {
		return ""
	}
	return daerah
}
