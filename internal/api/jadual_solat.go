package api

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/shabilullah/gowaktusolat/internal/db"
)

type JadualSolat struct {
	DB *sql.DB
}

func (h *JadualSolat) FetchMonth(c fiber.Ctx) error {
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

	pdf, err := generateJadualPDF(zone, year, month, rows)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"message": fmt.Sprintf("PDF generation failed: %v", err)})
	}

	c.Set("Content-Type", "application/pdf")
	return c.Send(pdf)
}

func generateJadualPDF(zone string, year, month int, rows []db.PrayerTimeRow) ([]byte, error) {
	var b strings.Builder

	headers := []string{"Tarikh", "Hijri", "Imsak", "Subuh", "Syuruk", "Zohor", "Asar", "Maghrib", "Isyak"}
	colWidths := []float64{24, 32, 24, 24, 24, 24, 24, 24, 24}
	totalWidth := 0.0
	for _, w := range colWidths {
		totalWidth += w
	}

	pageW := 297.0
	pageH := 210.0
	marginLeft := (pageW - totalWidth) / 2
	marginTop := 15.0
	rowH := 6.0
	headerH := 7.0
	fontSize := 8

	var stream strings.Builder
	y := pageH - marginTop

	stream.WriteString(fmt.Sprintf("BT /F1 14 Tf %.1f %.1f Td (Jadual Waktu Solat) Tj ET\n",
		mmToPt(pageW/2-40), mmToPt(y)))
	y -= 12
	stream.WriteString(fmt.Sprintf("BT /F1 10 Tf %.1f %.1f Td (Zon: %s  |  %s %d) Tj ET\n",
		mmToPt(pageW/2-35), mmToPt(y), escapePDF(zone), monthName(month), year))
	y -= 10

	stream.WriteString(fmt.Sprintf("BT /F1 %d Tf\n", fontSize))
	x := marginLeft
	for i, h := range headers {
		tw := float64(len(h)) * float64(fontSize) * 0.35
		stream.WriteString(fmt.Sprintf("%.1f %.1f Td (%s) Tj\n",
			mmToPt(x+(colWidths[i]-tw)/2), mmToPt(y-headerH/2-float64(fontSize)*0.2), escapePDF(h)))
		x += colWidths[i]
	}
	stream.WriteString("ET\n")
	stream.WriteString(fmt.Sprintf("%.3f %.3f %.3f RG 0.5 w\n", 0.5, 0.5, 0.5))
	stream.WriteString(fmt.Sprintf("%.1f %.1f %.1f %.1f re S\n",
		mmToPt(marginLeft), mmToPt(y-headerH), mmToPt(totalWidth), mmToPt(headerH)))
	y -= headerH

	for _, row := range rows {
		stream.WriteString(fmt.Sprintf("BT /F1 %d Tf\n", fontSize))

		t, err := time.Parse("2006-01-02", row.Date)
		dateStr := row.Date
		if err == nil {
			dateStr = t.Format("02/01")
		}

		cells := []string{
			dateStr,
			shortenHijri(row.Hijri),
			formatTimeShort(row.Imsak),
			formatTimeShort(row.Fajr),
			formatTimeShort(row.Syuruk),
			formatTimeShort(row.Dhuhr),
			formatTimeShort(row.Asr),
			formatTimeShort(row.Maghrib),
			formatTimeShort(row.Isha),
		}

		x = marginLeft
		for i, cell := range cells {
			escaped := escapePDF(cell)
			tw := float64(len(cell)) * float64(fontSize) * 0.35
			stream.WriteString(fmt.Sprintf("%.1f %.1f Td (%s) Tj\n",
				mmToPt(x+(colWidths[i]-tw)/2), mmToPt(y-rowH/2-float64(fontSize)*0.2), escaped))
			x += colWidths[i]
		}
		stream.WriteString("ET\n")
		stream.WriteString(fmt.Sprintf("%.3f %.3f %.3f RG 0.3 w\n", 0.7, 0.7, 0.7))
		stream.WriteString(fmt.Sprintf("%.1f %.1f %.1f %.1f re S\n",
			mmToPt(marginLeft), mmToPt(y-rowH), mmToPt(totalWidth), mmToPt(rowH)))
		y -= rowH
	}

	streamContent := stream.String()

	catalog := "1 0 obj\n<< /Type /Catalog /Pages 2 0 R >>\nendobj\n"
	pages := "2 0 obj\n<< /Type /Pages /Kids [3 0 R] /Count 1 >>\nendobj\n"
	page := fmt.Sprintf("3 0 obj\n<< /Type /Page /Parent 2 0 R /MediaBox [0 0 %.1f %.1f] /Contents 4 0 R /Resources << /Font << /F1 5 0 R >> >> >>\nendobj\n",
		mmToPt(pageW), mmToPt(pageH))
	contentObj := fmt.Sprintf("4 0 obj\n<< /Length %d >>\nstream\n%s\nendstream\nendobj\n",
		len(streamContent), streamContent)
	font := "5 0 obj\n<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica >>\nendobj\n"

	objects := []string{catalog, pages, page, contentObj, font}

	b.WriteString("%PDF-1.4\n")
	for _, obj := range objects {
		b.WriteString(obj)
	}

	xrefOffset := b.Len()
	b.WriteString("xref\n")
	b.WriteString(fmt.Sprintf("0 %d\n", len(objects)+1))
	b.WriteString("0000000000 65535 f \n")
	b.WriteString(fmt.Sprintf("%010d 00000 n \n", len("%PDF-1.4\n")))
	b.WriteString(fmt.Sprintf("%010d 00000 n \n", len("%PDF-1.4\n")+len(catalog)))
	b.WriteString(fmt.Sprintf("%010d 00000 n \n", len("%PDF-1.4\n")+len(catalog)+len(pages)))
	b.WriteString(fmt.Sprintf("%010d 00000 n \n", len("%PDF-1.4\n")+len(catalog)+len(pages)+len(page)))
	b.WriteString(fmt.Sprintf("%010d 00000 n \n", len("%PDF-1.4\n")+len(catalog)+len(pages)+len(page)+len(contentObj)))

	b.WriteString("trailer\n")
	b.WriteString(fmt.Sprintf("<< /Size %d /Root 1 0 R >>\n", len(objects)+1))
	b.WriteString("startxref\n")
	b.WriteString(fmt.Sprintf("%d\n", xrefOffset))
	b.WriteString("%%EOF\n")

	return []byte(b.String()), nil
}

func formatTimeShort(t string) string {
	if t == "" {
		return "-"
	}
	if len(t) >= 5 {
		return t[:5]
	}
	return t
}

func shortenHijri(hijri string) string {
	if len(hijri) > 25 {
		return hijri[:22] + "..."
	}
	return hijri
}

func mmToPt(mm float64) float64 {
	return mm * 2.834645669
}

func escapePDF(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "(", "\\(")
	s = strings.ReplaceAll(s, ")", "\\)")
	return s
}
