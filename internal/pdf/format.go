package pdf

import "strings"

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
