package service

import (
	"testing"
)

func TestParseDateOnly(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{"ISOFormat", "2024-01-15T10:30:00Z", "2024-01-15"},
		{"DateOnly", "2024-01-15", "2024-01-15"},
		{"WithTimezone", "2024-01-15T10:30:00+09:00", "2024-01-15"},
		{"Empty", "", ""},
		{"Short", "2024", "2024"},
		{"LongISO", "2024-01-15T10:30:00.123456Z", "2024-01-15"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := ParseDateOnly(tc.input)
			if result != tc.expected {
				t.Errorf("ParseDateOnly(%q) = %q, want %q", tc.input, result, tc.expected)
			}
		})
	}
}

func TestParseDateSafe(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{"ISOFormat", "2024-01-15T10:30:00Z", "2024-01-15"},
		{"DateOnly", "2024-01-15", "2024-01-15"},
		{"RFC3339", "2024-01-15T10:30:00+09:00", "2024-01-15"},
		{"DateTime", "2024-01-15 10:30:00", "2024-01-15"},
		{"Empty", "", ""},
		{"Invalid", "not-a-date", ""},
		{"PartialDate", "2024-01-15T", "2024-01-15"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := ParseDateSafe(tc.input)
			if result != tc.expected {
				t.Errorf("ParseDateSafe(%q) = %q, want %q", tc.input, result, tc.expected)
			}
		})
	}
}

func TestFormatDateRange(t *testing.T) {
	testCases := []struct {
		name          string
		startDate     string
		endDate       string
		expectedStart string
		expectedEnd   string
	}{
		{
			"Normal",
			"2024-01-01",
			"2024-01-07",
			"2024-01-01T00:00:00Z",
			"2024-01-07T23:59:59Z",
		},
		{
			"SameDay",
			"2024-01-15",
			"2024-01-15",
			"2024-01-15T00:00:00Z",
			"2024-01-15T23:59:59Z",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			start, end := FormatDateRange(tc.startDate, tc.endDate)
			if start != tc.expectedStart {
				t.Errorf("FormatDateRange start = %q, want %q", start, tc.expectedStart)
			}
			if end != tc.expectedEnd {
				t.Errorf("FormatDateRange end = %q, want %q", end, tc.expectedEnd)
			}
		})
	}
}

func TestIsDateInRange(t *testing.T) {
	testCases := []struct {
		name      string
		date      string
		startDate string
		endDate   string
		expected  bool
	}{
		{"InRange", "2024-01-15", "2024-01-01", "2024-01-31", true},
		{"AtStart", "2024-01-01", "2024-01-01", "2024-01-31", true},
		{"AtEnd", "2024-01-31", "2024-01-01", "2024-01-31", true},
		{"BeforeRange", "2023-12-31", "2024-01-01", "2024-01-31", false},
		{"AfterRange", "2024-02-01", "2024-01-01", "2024-01-31", false},
		{"ISODate", "2024-01-15T10:30:00Z", "2024-01-01", "2024-01-31", true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := IsDateInRange(tc.date, tc.startDate, tc.endDate)
			if result != tc.expected {
				t.Errorf("IsDateInRange(%q, %q, %q) = %v, want %v",
					tc.date, tc.startDate, tc.endDate, result, tc.expected)
			}
		})
	}
}
