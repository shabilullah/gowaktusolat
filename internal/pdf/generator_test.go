package pdf

import (
	"testing"

	"github.com/shabilullah/gowaktusolat/internal/repository"
)

func TestGenerateMonth_ProducesValidPDFHeader(t *testing.T) {
	rows := []repository.PrayerTimeRow{
		{Date: "2026-06-01", Fajr: "05:49:00", Syuruk: "07:08:00", Dhuhr: "13:14:00", Asr: "16:39:00", Maghrib: "19:22:00", Isha: "20:37:00"},
		{Date: "2026-06-02", Fajr: "05:49:00", Syuruk: "07:08:00", Dhuhr: "13:14:00", Asr: "16:39:00", Maghrib: "19:23:00", Isha: "20:38:00"},
	}

	pdf := GenerateMonth("SGR01", "Gombak", 2026, 6, rows)

	s := string(pdf)
	checks := []string{
		"%PDF-1.4",
		"/Type /Catalog",
		"/Type /Pages",
		"/Type /Page",
		"/Type /Font",
		"/BaseFont /Helvetica",
		"/BaseFont /Helvetica-Bold",
		"/BaseFont /Helvetica-Oblique",
		"Jadual Waktu Solat",
		"Jun 2026",
		"SGR01 Gombak",
		"Subuh",
		"Zohor",
		"05:49",
		"19:22",
		"Waktu Solat Malaysia",
		"xref",
		"trailer",
		"%%EOF",
	}

	for _, check := range checks {
		if !contains(s, check) {
			t.Errorf("PDF output missing expected string %q", check)
		}
	}
}

func TestGenerateMonth_NoDaerah(t *testing.T) {
	rows := []repository.PrayerTimeRow{
		{Date: "2026-06-01", Fajr: "05:49:00", Syuruk: "07:08:00", Dhuhr: "13:14:00", Asr: "16:39:00", Maghrib: "19:22:00", Isha: "20:37:00"},
	}

	pdf := GenerateMonth("SGR01", "", 2026, 6, rows)

	s := string(pdf)
	if !contains(s, "SGR01") {
		t.Error("PDF missing zone code")
	}
	// Daerah should not appear doubled
	if contains(s, "SGR01  SGR01") {
		t.Error("PDF has doubled zone when daerah is empty")
	}
}

func TestGenerateMonth_EmptyRows(t *testing.T) {
	pdf := GenerateMonth("SGR01", "Gombak", 2026, 6, nil)

	s := string(pdf)
	checks := []string{"%PDF-1.4", "xref", "%%EOF"}
	for _, check := range checks {
		if !contains(s, check) {
			t.Errorf("empty-rows PDF missing %q", check)
		}
	}
}

func TestGenerateYear_ProducesMultiPagePDF(t *testing.T) {
	monthly := make([][]repository.PrayerTimeRow, 13)
	for m := 1; m <= 12; m++ {
		monthly[m] = []repository.PrayerTimeRow{
			{Date: "2026-06-01", Fajr: "05:49:00", Syuruk: "07:08:00", Dhuhr: "13:14:00", Asr: "16:39:00", Maghrib: "19:22:00", Isha: "20:37:00"},
		}
	}

	pdf := GenerateYear("SGR01", "Gombak", 2026, monthly)

	s := string(pdf)
	checks := []string{
		"%PDF-1.4",
		"/Count 12",
		"/Type /Font",
		"/BaseFont /Helvetica",
	}

	for _, check := range checks {
		if !contains(s, check) {
			t.Errorf("multi-page PDF missing %q", check)
		}
	}
}

func TestFormatTimeShort(t *testing.T) {
	tests := []struct {
		input, expected string
	}{
		{"", "-"},
		{"05:49:00", "05:49"},
		{"05:49", "05:49"},
		{"5", "5"},
	}

	for _, tt := range tests {
		got := formatTimeShort(tt.input)
		if got != tt.expected {
			t.Errorf("formatTimeShort(%q) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}

func TestMalayMonthName(t *testing.T) {
	tests := []struct {
		m        int
		expected string
	}{
		{1, "Januari"},
		{6, "Jun"},
		{12, "Disember"},
		{0, ""},
		{13, ""},
	}

	for _, tt := range tests {
		got := malayMonthName(tt.m)
		if got != tt.expected {
			t.Errorf("malayMonthName(%d) = %q, want %q", tt.m, got, tt.expected)
		}
	}
}

func TestEscapePDF(t *testing.T) {
	tests := []struct {
		input, expected string
	}{
		{"hello", "hello"},
		{"test (paren)", "test \\(paren\\)"},
		{`c:\path`, `c:\\path`},
		{`mix (test) \ ok`, `mix \(test\) \\ ok`},
	}

	for _, tt := range tests {
		got := escapePDF(tt.input)
		if got != tt.expected {
			t.Errorf("escapePDF(%q) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}

func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
