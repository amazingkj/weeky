package service

import (
	"time"
)

const (
	DateFormatYMD = "2006-01-02"
	DateFormatISO = "2006-01-02T15:04:05Z07:00"
)

func ParseDateOnly(dateStr string) string {
	if dateStr == "" {
		return ""
	}
	if len(dateStr) >= 10 {
		return dateStr[:10]
	}

	return dateStr
}

func ParseDateSafe(dateStr string, layouts ...string) string {
	if dateStr == "" {
		return ""
	}

	if len(layouts) == 0 {
		layouts = []string{
			DateFormatISO,
			DateFormatYMD,
			"2006-01-02 15:04:05",
			time.RFC3339,
		}
	}

	for _, layout := range layouts {
		if t, err := time.Parse(layout, dateStr); err == nil {
			return t.Format(DateFormatYMD)
		}
	}

	if len(dateStr) >= 10 {
		candidate := dateStr[:10]
		if candidate[4] == '-' && candidate[7] == '-' {
			if _, err := time.Parse(DateFormatYMD, candidate); err == nil {
				return candidate
			}
		}
	}

	return ""
}

func FormatDateRange(startDate, endDate string) (start, end string) {
	start = startDate + "T00:00:00Z"
	end = endDate + "T23:59:59Z"
	return
}

func IsDateInRange(date, startDate, endDate string) bool {
	dateOnly := ParseDateOnly(date)
	return dateOnly >= startDate && dateOnly <= endDate
}
