package service

import (
	"time"
)

// DateFormat constants
const (
	DateFormatYMD = "2006-01-02"
	DateFormatISO = "2006-01-02T15:04:05Z07:00"
)

// ParseDateOnly extracts YYYY-MM-DD from various date formats
func ParseDateOnly(dateStr string) string {
	if dateStr == "" {
		return ""
	}

	// Handle ISO format with time
	if len(dateStr) >= 10 {
		return dateStr[:10]
	}

	return dateStr
}

// ParseDateSafe parses a date string and returns empty string on failure
func ParseDateSafe(dateStr string, layouts ...string) string {
	if dateStr == "" {
		return ""
	}

	// Default layouts to try
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

	// Fallback: try to extract first 10 characters if they look like a date (YYYY-MM-DD)
	if len(dateStr) >= 10 {
		candidate := dateStr[:10]
		// Validate it looks like YYYY-MM-DD
		if len(candidate) == 10 && candidate[4] == '-' && candidate[7] == '-' {
			// Try to parse the candidate
			if _, err := time.Parse(DateFormatYMD, candidate); err == nil {
				return candidate
			}
		}
	}

	return ""
}

// FormatDateRange formats start and end dates for API queries
func FormatDateRange(startDate, endDate string) (start, end string) {
	start = startDate + "T00:00:00Z"
	end = endDate + "T23:59:59Z"
	return
}

// IsDateInRange checks if a date is within the given range
func IsDateInRange(date, startDate, endDate string) bool {
	dateOnly := ParseDateOnly(date)
	return dateOnly >= startDate && dateOnly <= endDate
}
