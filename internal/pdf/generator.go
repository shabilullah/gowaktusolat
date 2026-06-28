package pdf

import (
	"fmt"
	"strings"
	"time"

	"github.com/shabilullah/gowaktusolat/internal/db"
)

// GenerateMonth produces a single-page PDF for one zone/month.
func GenerateMonth(zone, daerah string, year, month int, rows []db.PrayerTimeRow) []byte {
	content := generatePageContent(zone, daerah, year, month, rows)
	return buildSinglePagePDF(content)
}

// GenerateYear produces a multi-page PDF for one zone across 12 months.
// monthly is indexed 1–12; monthly[0] is ignored.
func GenerateYear(zone, daerah string, year int, monthly [][]db.PrayerTimeRow) []byte {
	contents := make([]string, 0, 12)
	for month := 1; month <= 12; month++ {
		contents = append(contents, generatePageContent(zone, daerah, year, month, monthly[month]))
	}
	return buildMultiPagePDF(contents)
}

// generatePageContent returns the PDF content stream for one month page.
func generatePageContent(zone, daerah string, year, month int, rows []db.PrayerTimeRow) string {
	headers := []string{"Tarikh", "Subuh", "Syuruk", "Zohor", "Asar", "Maghrib", "Isyak"}
	numCols := len(headers)

	pageH := 595.280
	marginLeft := 15.0
	colW := 115.127
	totalW := colW * float64(numCols)
	rowH := 14.920
	headerH := 14.921

	var s strings.Builder

	// Title
	y := pageH - 15.0 - 18.0*0.8
	s.WriteString("0 0 0 rg\n")
	s.WriteString(fmt.Sprintf("BT /F2 18 Tf %.3f %.3f Td (Jadual Waktu Solat) Tj ET\n", marginLeft, y))
	monthYear := fmt.Sprintf("%s %d", malayMonthName(month), year)
	s.WriteString(fmt.Sprintf("BT /F2 18 Tf %.3f %.3f Td (%s) Tj ET\n",
		marginLeft+totalW-float64(len(monthYear))*9.0, y, escapePDF(monthYear)))
	y -= 25.0

	// Subtitle yellow bar
	barY := y
	barH := 16.0
	r := 2.0
	s.WriteString("1 1 0 rg\n")
	s.WriteString(fmt.Sprintf("%.3f %.3f m\n", marginLeft+r, barY))
	s.WriteString(fmt.Sprintf("%.3f %.3f %.3f %.3f %.3f %.3f c\n",
		marginLeft+r, barY+barH-r, marginLeft, barY+barH-r, marginLeft, barY+barH))
	s.WriteString(fmt.Sprintf("%.3f %.3f l\n", marginLeft, barY+r))
	s.WriteString(fmt.Sprintf("%.3f %.3f %.3f %.3f %.3f %.3f c\n",
		marginLeft, barY, marginLeft+r, barY, marginLeft+r, barY))
	s.WriteString(fmt.Sprintf("%.3f %.3f l\n", marginLeft+totalW-r, barY))
	s.WriteString(fmt.Sprintf("%.3f %.3f %.3f %.3f %.3f %.3f c\n",
		marginLeft+totalW, barY, marginLeft+totalW, barY+r, marginLeft+totalW, barY+r))
	s.WriteString(fmt.Sprintf("%.3f %.3f l\n", marginLeft+totalW, barY+barH))
	s.WriteString(fmt.Sprintf("%.3f %.3f %.3f %.3f %.3f %.3f c f\n",
		marginLeft+totalW, barY+barH, marginLeft+totalW-r, barY+barH, marginLeft+totalW-r, barY+barH))

	subtitle := zone
	if daerah != "" {
		subtitle = zone + " " + daerah
	}
	s.WriteString("0 0 0 rg\n")
	s.WriteString(fmt.Sprintf("BT /F3 12 Tf %.3f %.3f Td (%s) Tj ET\n",
		marginLeft+3.0, barY+barH/2-4.0, escapePDF(subtitle)))
	y = barY - 5.0

	// Header row
	headerTop := y
	s.WriteString("0.133 0.133 0.133 rg\n")
	s.WriteString(fmt.Sprintf("%.3f %.3f %.3f %.3f re f\n",
		marginLeft, headerTop-headerH, totalW, headerH))

	s.WriteString("0.2 0.2 0.2 RG 0.75 w\n")
	colX := marginLeft
	for range numCols + 1 {
		s.WriteString(fmt.Sprintf("%.3f %.3f m %.3f %.3f l S\n", colX, headerTop, colX, headerTop-headerH))
		colX += colW
	}
	s.WriteString(fmt.Sprintf("%.3f %.3f m %.3f %.3f l S\n",
		marginLeft, headerTop-headerH, marginLeft+totalW, headerTop-headerH))

	s.WriteString("1 1 1 rg\n")
	colX = marginLeft
	for _, h := range headers {
		tw := float64(len(h)) * 5.25
		s.WriteString(fmt.Sprintf("BT /F2 10.5 Tf %.3f %.3f Td (%s) Tj ET\n",
			colX+(colW-tw)/2, headerTop-headerH/2-3.5, escapePDF(h)))
		colX += colW
	}

	y = headerTop - headerH

	// Data rows
	for rowIdx, row := range rows {
		if rowIdx%2 == 0 {
			s.WriteString("1 1 1 rg\n")
		} else {
			s.WriteString("0.949 0.949 0.949 rg\n")
		}
		s.WriteString(fmt.Sprintf("%.3f %.3f %.3f %.3f re f\n", marginLeft, y-rowH, totalW, rowH))

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

		s.WriteString("0 0 0 rg\n")
		colX = marginLeft
		for i, cell := range cells {
			if i == 0 {
				s.WriteString(fmt.Sprintf("BT /F1 10.5 Tf %.3f %.3f Td (%s) Tj ET\n",
					colX+3.0, y-rowH/2-3.5, escapePDF(cell)))
			} else {
				tw := float64(len(cell)) * 5.25
				s.WriteString(fmt.Sprintf("BT /F1 10.5 Tf %.3f %.3f Td (%s) Tj ET\n",
					colX+(colW-tw)/2, y-rowH/2-3.5, escapePDF(cell)))
			}
			colX += colW
		}

		s.WriteString("0.2 0.2 0.2 RG 0.75 w\n")
		colX = marginLeft
		for range numCols + 1 {
			s.WriteString(fmt.Sprintf("%.3f %.3f m %.3f %.3f l S\n", colX, y, colX, y-rowH))
			colX += colW
		}
		s.WriteString(fmt.Sprintf("%.3f %.3f m %.3f %.3f l S\n",
			marginLeft, y-rowH, marginLeft+totalW, y-rowH))

		y -= rowH
	}

	// Footer
	y -= 8.0
	footerY := y
	footerH := 12.821

	s.WriteString("1 1 1 rg\n")
	s.WriteString(fmt.Sprintf("%.3f %.3f %.3f %.3f re f\n", marginLeft, footerY-footerH, totalW/2, footerH))
	now := time.Now()
	footerLeft := fmt.Sprintf("Dijana pada %s", now.Format("02/01/2006 15:04:05"))
	s.WriteString("0.333 0.333 0.333 rg\n")
	s.WriteString(fmt.Sprintf("BT /F1 9 Tf %.3f %.3f Td (%s) Tj ET\n",
		marginLeft, footerY-footerH/2-3.0, escapePDF(footerLeft)))

	rightX := marginLeft + totalW/2
	s.WriteString("1 1 1 rg\n")
	s.WriteString(fmt.Sprintf("%.3f %.3f %.3f %.3f re f\n", rightX, footerY-footerH, totalW/2, footerH))
	s.WriteString("0.333 0.333 0.333 rg\n")
	brandText := "Waktu Solat Malaysia"
	s.WriteString(fmt.Sprintf("BT /F2 9 Tf %.3f %.3f Td (%s) Tj ET\n",
		rightX+totalW/2-float64(len(brandText))*4.5, footerY-footerH/2-3.0, escapePDF(brandText)))

	return s.String()
}

// buildSinglePagePDF assembles a single-page PDF from the content stream.
func buildSinglePagePDF(content string) []byte {
	pageW := "841.890"
	pageH := "595.280"
	objects := []string{
		"1 0 obj\n<< /Type /Catalog /Pages 2 0 R >>\nendobj\n",
		"2 0 obj\n<< /Type /Pages /Kids [3 0 R] /Count 1 >>\nendobj\n",
		fmt.Sprintf("3 0 obj\n<< /Type /Page /Parent 2 0 R /MediaBox [0 0 %s %s] /Contents 4 0 R /Resources << /Font << /F1 5 0 R /F2 6 0 R /F3 7 0 R >> >> >>\nendobj\n", pageW, pageH),
		fmt.Sprintf("4 0 obj\n<< /Length %d >>\nstream\n%s\nendstream\nendobj\n", len(content), content),
		"5 0 obj\n<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica /Encoding /WinAnsiEncoding >>\nendobj\n",
		"6 0 obj\n<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica-Bold /Encoding /WinAnsiEncoding >>\nendobj\n",
		"7 0 obj\n<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica-Oblique /Encoding /WinAnsiEncoding >>\nendobj\n",
	}
	return writePDF(objects)
}

// buildMultiPagePDF assembles a multi-page PDF with shared font resources.
func buildMultiPagePDF(contents []string) []byte {
	pageW := "841.890"
	pageH := "595.280"
	numPages := len(contents)

	var objects []string

	// 1: Catalog
	objects = append(objects, "1 0 obj\n<< /Type /Catalog /Pages 2 0 R >>\nendobj\n")

	// 2: Pages
	kids := make([]string, numPages)
	firstPageObj := 3
	for i := range kids {
		kids[i] = fmt.Sprintf("%d 0 R", firstPageObj+i*2)
	}
	objects = append(objects, fmt.Sprintf("2 0 obj\n<< /Type /Pages /Kids [%s] /Count %d >>\nendobj\n",
		strings.Join(kids, " "), numPages))

	// Page objects + Content stream objects
	fontRefObj := firstPageObj + 2*numPages
	fontRes := fmt.Sprintf("/Resources << /Font << /F1 %d 0 R /F2 %d 0 R /F3 %d 0 R >> >>",
		fontRefObj, fontRefObj+1, fontRefObj+2)

	for i, content := range contents {
		pageObj := firstPageObj + i*2
		contentObj := pageObj + 1
		objects = append(objects,
			fmt.Sprintf("%d 0 obj\n<< /Type /Page /Parent 2 0 R /MediaBox [0 0 %s %s] /Contents %d 0 R %s >>\nendobj\n",
				pageObj, pageW, pageH, contentObj, fontRes))
		objects = append(objects,
			fmt.Sprintf("%d 0 obj\n<< /Length %d >>\nstream\n%s\nendstream\nendobj\n",
				contentObj, len(content), content))
	}

	// Font objects
	objects = append(objects,
		fmt.Sprintf("%d 0 obj\n<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica /Encoding /WinAnsiEncoding >>\nendobj\n", fontRefObj),
		fmt.Sprintf("%d 0 obj\n<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica-Bold /Encoding /WinAnsiEncoding >>\nendobj\n", fontRefObj+1),
		fmt.Sprintf("%d 0 obj\n<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica-Oblique /Encoding /WinAnsiEncoding >>\nendobj\n", fontRefObj+2),
	)

	return writePDF(objects)
}

func writePDF(objects []string) []byte {
	var b strings.Builder
	b.WriteString("%PDF-1.4\n")
	for _, obj := range objects {
		b.WriteString(obj)
	}

	xrefOffset := b.Len()
	b.WriteString("xref\n")
	b.WriteString(fmt.Sprintf("0 %d\n", len(objects)+1))
	b.WriteString("0000000000 65535 f \n")

	offset := len("%PDF-1.4\n")
	for _, obj := range objects {
		b.WriteString(fmt.Sprintf("%010d 00000 n \n", offset))
		offset += len(obj)
	}

	b.WriteString("trailer\n")
	b.WriteString(fmt.Sprintf("<< /Size %d /Root 1 0 R >>\n", len(objects)+1))
	b.WriteString("startxref\n")
	b.WriteString(fmt.Sprintf("%d\n", xrefOffset))
	b.WriteString("%%EOF\n")
	return []byte(b.String())
}
