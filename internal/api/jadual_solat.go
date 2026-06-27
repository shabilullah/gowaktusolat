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

	var daerah string
	if err := h.DB.QueryRow(
		"SELECT daerah FROM prayer_zones WHERE jakim_code = ?", zone,
	).Scan(&daerah); err != nil {
		daerah = ""
	}

	pdf, err := generateJadualPDF(zone, daerah, year, month, rows)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"message": fmt.Sprintf("PDF generation failed: %v", err)})
	}

	c.Set("Content-Type", "application/pdf")
	c.Set("Content-Disposition", fmt.Sprintf("inline; filename=\"jadual-solat-%s-%d-%02d.pdf\"", zone, year, month))
	return c.Send(pdf)
}

func generateJadualPDF(zone, daerah string, year, month int, rows []db.PrayerTimeRow) ([]byte, error) {
	var b strings.Builder

	headers := []string{"Tarikh", "Subuh", "Syuruk", "Zohor", "Asar", "Maghrib", "Isyak"}
	numCols := len(headers)

	pageW := 841.890
	pageH := 595.280
	marginLeft := 15.0
	colW := 115.127 // Equal width columns matching reference
	totalW := colW * float64(numCols)
	rowH := 14.920 // Matching reference row height
	headerH := 14.921

	var stream strings.Builder

	// --- Title: left "Jadual Waktu Solat" + right "Month Year" in Bold 18pt ---
	y := pageH - 15.0 - 18.0*0.8
	stream.WriteString("0 0 0 rg\n")
	stream.WriteString(fmt.Sprintf("BT /F2 18 Tf %.3f %.3f Td (Jadual Waktu Solat) Tj ET\n",
		marginLeft, y))
	monthYear := fmt.Sprintf("%s %d", malayMonthName(month), year)
	stream.WriteString(fmt.Sprintf("BT /F2 18 Tf %.3f %.3f Td (%s) Tj ET\n",
		marginLeft+totalW-float64(len(monthYear))*9.0, y, escapePDF(monthYear)))
	y -= 25.0

	// --- Subtitle: yellow bar with rounded corners + oblique zone/daerah ---
	barY := y
	barH := 16.0
	stream.WriteString("1 1 0 rg\n") // Yellow fill
	// Rounded rectangle using Bezier curves
	r := 2.0 // corner radius
	stream.WriteString(fmt.Sprintf("%.3f %.3f m\n", marginLeft+r, barY))
	stream.WriteString(fmt.Sprintf("%.3f %.3f %.3f %.3f %.3f %.3f c\n",
		marginLeft+r, barY+barH-r, marginLeft, barY+barH-r, marginLeft, barY+barH))
	stream.WriteString(fmt.Sprintf("%.3f %.3f l\n", marginLeft, barY+r))
	stream.WriteString(fmt.Sprintf("%.3f %.3f %.3f %.3f %.3f %.3f c\n",
		marginLeft, barY, marginLeft+r, barY, marginLeft+r, barY))
	stream.WriteString(fmt.Sprintf("%.3f %.3f l\n", marginLeft+totalW-r, barY))
	stream.WriteString(fmt.Sprintf("%.3f %.3f %.3f %.3f %.3f %.3f c\n",
		marginLeft+totalW, barY, marginLeft+totalW, barY+r, marginLeft+totalW, barY+r))
	stream.WriteString(fmt.Sprintf("%.3f %.3f l\n", marginLeft+totalW, barY+barH))
	stream.WriteString(fmt.Sprintf("%.3f %.3f %.3f %.3f %.3f %.3f c f\n",
		marginLeft+totalW, barY+barH, marginLeft+totalW-r, barY+barH, marginLeft+totalW-r, barY+barH))

	subtitle := zone
	if daerah != "" {
		subtitle = zone + " " + daerah
	}
	stream.WriteString("0 0 0 rg\n")
	stream.WriteString(fmt.Sprintf("BT /F3 12 Tf %.3f %.3f Td (%s) Tj ET\n",
		marginLeft+3.0, barY+barH/2-4.0, escapePDF(subtitle)))
	y = barY - 5.0

	// --- Header row: dark grey (#222) background, white bold text ---
	headerTop := y
	stream.WriteString("0.133 0.133 0.133 rg\n")
	stream.WriteString(fmt.Sprintf("%.3f %.3f %.3f %.3f re f\n",
		marginLeft, headerTop-headerH, totalW, headerH))

	// Column borders (0.75pt)
	stream.WriteString("0.2 0.2 0.2 RG 0.75 w\n")
	colX := marginLeft
	for range numCols + 1 {
		stream.WriteString(fmt.Sprintf("%.3f %.3f m %.3f %.3f l S\n",
			colX, headerTop, colX, headerTop-headerH))
		colX += colW
	}
	// Horizontal border below header
	stream.WriteString(fmt.Sprintf("%.3f %.3f m %.3f %.3f l S\n",
		marginLeft, headerTop-headerH, marginLeft+totalW, headerTop-headerH))

	stream.WriteString("1 1 1 rg\n")
	colX = marginLeft
	for _, h := range headers {
		tw := float64(len(h)) * 5.25 // 10.5pt bold char ≈ 5.25pt
		stream.WriteString(fmt.Sprintf("BT /F2 10.5 Tf %.3f %.3f Td (%s) Tj ET\n",
			colX+(colW-tw)/2, headerTop-headerH/2-3.5, escapePDF(h)))
		colX += colW
	}

	y = headerTop - headerH

	// --- Data rows ---
	for rowIdx, row := range rows {
		// Alternating background: white (#fff) or light grey (#f2f2f2)
		if rowIdx%2 == 0 {
			stream.WriteString("1 1 1 rg\n")
		} else {
			stream.WriteString("0.949 0.949 0.949 rg\n")
		}
		stream.WriteString(fmt.Sprintf("%.3f %.3f %.3f %.3f re f\n",
			marginLeft, y-rowH, totalW, rowH))

		t, err := time.Parse("2006-01-02", row.Date)
		dateStr := row.Date
		if err == nil {
			dateStr = t.Format("02-01-2006")
		}

		cells := []string{
			dateStr,
			formatTimeShort(row.Fajr),
			formatTimeShort(row.Syuruk),
			formatTimeShort(row.Dhuhr),
			formatTimeShort(row.Asr),
			formatTimeShort(row.Maghrib),
			formatTimeShort(row.Isha),
		}

		stream.WriteString("0 0 0 rg\n")
		colX = marginLeft
		for i, cell := range cells {
			// Date column left-aligned, time columns centered
			if i == 0 {
				stream.WriteString(fmt.Sprintf("BT /F1 10.5 Tf %.3f %.3f Td (%s) Tj ET\n",
					colX+3.0, y-rowH/2-3.5, escapePDF(cell)))
			} else {
				tw := float64(len(cell)) * 5.25
				stream.WriteString(fmt.Sprintf("BT /F1 10.5 Tf %.3f %.3f Td (%s) Tj ET\n",
					colX+(colW-tw)/2, y-rowH/2-3.5, escapePDF(cell)))
			}
			colX += colW
		}

		// Row borders
		stream.WriteString("0.2 0.2 0.2 RG 0.75 w\n")
		colX = marginLeft
		for range numCols + 1 {
			stream.WriteString(fmt.Sprintf("%.3f %.3f m %.3f %.3f l S\n",
				colX, y, colX, y-rowH))
			colX += colW
		}
		stream.WriteString(fmt.Sprintf("%.3f %.3f m %.3f %.3f l S\n",
			marginLeft, y-rowH, marginLeft+totalW, y-rowH))

		y -= rowH
	}

	// --- Footer ---
	y -= 8.0
	footerY := y
	footerH := 12.821

	// Left section: white background, "Dijana pada ..."
	stream.WriteString("1 1 1 rg\n")
	stream.WriteString(fmt.Sprintf("%.3f %.3f %.3f %.3f re f\n",
		marginLeft, footerY-footerH, totalW/2, footerH))
	now := time.Now()
	footerLeft := fmt.Sprintf("Dijana pada %s", now.Format("02/01/2006 15:04:05"))
	stream.WriteString("0.333 0.333 0.333 rg\n")
	stream.WriteString(fmt.Sprintf("BT /F1 9 Tf %.3f %.3f Td (%s) Tj ET\n",
		marginLeft, footerY-footerH/2-3.0, escapePDF(footerLeft)))

	// Right section: white background, "(logo) Waktu Solat Malaysia"
	rightX := marginLeft + totalW/2
	stream.WriteString("1 1 1 rg\n")
	stream.WriteString(fmt.Sprintf("%.3f %.3f %.3f %.3f re f\n",
		rightX, footerY-footerH, totalW/2, footerH))
	stream.WriteString("0.333 0.333 0.333 rg\n")
	brandText := "Waktu Solat Malaysia"
	stream.WriteString(fmt.Sprintf("BT /F2 9 Tf %.3f %.3f Td (%s) Tj ET\n",
		rightX+totalW/2-float64(len(brandText))*4.5, footerY-footerH/2-3.0, escapePDF(brandText)))

	streamContent := stream.String()

	// --- PDF objects ---
	catalog := "1 0 obj\n<< /Type /Catalog /Pages 2 0 R >>\nendobj\n"
	pages := "2 0 obj\n<< /Type /Pages /Kids [3 0 R] /Count 1 >>\nendobj\n"
	page := fmt.Sprintf("3 0 obj\n<< /Type /Page /Parent 2 0 R /MediaBox [0 0 %.3f %.3f] /Contents 4 0 R /Resources << /Font << /F1 5 0 R /F2 6 0 R /F3 7 0 R >> >> >>\nendobj\n",
		pageW, pageH)
	contentObj := fmt.Sprintf("4 0 obj\n<< /Length %d >>\nstream\n%s\nendstream\nendobj\n",
		len(streamContent), streamContent)
	fontNormal := "5 0 obj\n<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica /Encoding /WinAnsiEncoding >>\nendobj\n"
	fontBold := "6 0 obj\n<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica-Bold /Encoding /WinAnsiEncoding >>\nendobj\n"
	fontOblique := "7 0 obj\n<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica-Oblique /Encoding /WinAnsiEncoding >>\nendobj\n"

	objects := []string{catalog, pages, page, contentObj, fontNormal, fontBold, fontOblique}

	b.WriteString("%PDF-1.4\n")
	for _, obj := range objects {
		b.WriteString(obj)
	}

	xrefOffset := b.Len()
	b.WriteString("xref\n")
	b.WriteString(fmt.Sprintf("0 %d\n", len(objects)+1))
	b.WriteString("0000000000 65535 f \n")

	offsets := []int{len("%PDF-1.4\n")}
	for i, obj := range objects {
		offsets = append(offsets, offsets[i]+len(obj))
	}
	for _, off := range offsets[:len(offsets)-1] {
		b.WriteString(fmt.Sprintf("%010d 00000 n \n", off))
	}

	b.WriteString("trailer\n")
	b.WriteString(fmt.Sprintf("<< /Size %d /Root 1 0 R >>\n", len(objects)+1))
	b.WriteString("startxref\n")
	b.WriteString(fmt.Sprintf("%d\n", xrefOffset))
	b.WriteString("%%EOF\n")

	return []byte(b.String()), nil
}

var malayMonths = []string{
	"Januari", "Februari", "Mac", "April", "Mei", "Jun",
	"Julai", "Ogos", "September", "Oktober", "November", "Disember",
}

func malayMonthName(m int) string {
	if m < 1 || m > 12 {
		return ""
	}
	return malayMonths[m-1]
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

func escapePDF(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "(", "\\(")
	s = strings.ReplaceAll(s, ")", "\\)")
	return s
}
