package utils

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// ParseDate parses a date string in DD/MM format and returns a time.Time for year 2025
func ParseDate(dateStr string) (time.Time, error) {
	if dateStr == "" {
		return time.Time{}, errors.New("date string cannot be empty")
	}

	parts := strings.Split(dateStr, "/")
	if len(parts) != 2 {
		return time.Time{}, fmt.Errorf("invalid date format, expected DD/MM, got: %s", dateStr)
	}

	day, err := strconv.Atoi(strings.TrimSpace(parts[0]))
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid day: %s", parts[0])
	}

	month, err := strconv.Atoi(strings.TrimSpace(parts[1]))
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid month: %s", parts[1])
	}

	// Validate ranges
	if day < 1 || day > 31 {
		return time.Time{}, fmt.Errorf("day must be between 1 and 31, got: %d", day)
	}

	if month < 1 || month > 12 {
		return time.Time{}, fmt.Errorf("month must be between 1 and 12, got: %d", month)
	}

	// Create date with year 2025
	date := time.Date(2025, time.Month(month), day, 0, 0, 0, 0, time.UTC)

	// Validate that the date is actually valid (e.g., not Feb 30)
	if date.Day() != day || date.Month() != time.Month(month) {
		return time.Time{}, fmt.Errorf("invalid date: day %d does not exist in month %d", day, month)
	}

	return date, nil
}

// ParseDateWithYear parses a date string in DD/MM format and returns a time.Time for the given year.
func ParseDateWithYear(dateStr string, year int) (time.Time, error) {
	if dateStr == "" {
		return time.Time{}, errors.New("date string cannot be empty")
	}

	parts := strings.Split(dateStr, "/")
	if len(parts) != 2 {
		return time.Time{}, fmt.Errorf("invalid date format, expected DD/MM, got: %s", dateStr)
	}

	day, err := strconv.Atoi(strings.TrimSpace(parts[0]))
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid day: %s", parts[0])
	}

	month, err := strconv.Atoi(strings.TrimSpace(parts[1]))
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid month: %s", parts[1])
	}

	if day < 1 || day > 31 {
		return time.Time{}, fmt.Errorf("day must be between 1 and 31, got: %d", day)
	}

	if month < 1 || month > 12 {
		return time.Time{}, fmt.Errorf("month must be between 1 and 12, got: %d", month)
	}

	date := time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC)

	if date.Day() != day || date.Month() != time.Month(month) {
		return time.Time{}, fmt.Errorf("invalid date: day %d does not exist in month %d", day, month)
	}

	return date, nil
}

// ParseDateFlexible parses a date string in DD/MM or DD/MM/YYYY format.
// When year is omitted (DD/MM), the current calendar year is used.
// Returns a time.Time in UTC at midnight.
func ParseDateFlexible(dateStr string) (time.Time, error) {
	if dateStr == "" {
		return time.Time{}, errors.New("date string cannot be empty")
	}
	parts := strings.Split(dateStr, "/")
	if len(parts) != 2 && len(parts) != 3 {
		return time.Time{}, fmt.Errorf("invalid date format, expected DD/MM or DD/MM/YYYY, got: %s", dateStr)
	}
	day, err := strconv.Atoi(strings.TrimSpace(parts[0]))
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid day: %s", parts[0])
	}
	month, err := strconv.Atoi(strings.TrimSpace(parts[1]))
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid month: %s", parts[1])
	}
	year := time.Now().Year()
	if len(parts) == 3 {
		year, err = strconv.Atoi(strings.TrimSpace(parts[2]))
		if err != nil {
			return time.Time{}, fmt.Errorf("invalid year: %s", parts[2])
		}
	}
	if day < 1 || day > 31 {
		return time.Time{}, fmt.Errorf("day must be between 1 and 31, got: %d", day)
	}
	if month < 1 || month > 12 {
		return time.Time{}, fmt.Errorf("month must be between 1 and 12, got: %d", month)
	}
	date := time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC)
	if date.Day() != day || date.Month() != time.Month(month) {
		return time.Time{}, fmt.Errorf("invalid date: day %d does not exist in month %d", day, month)
	}
	return date, nil
}

// FormatDate formats a time.Time as DD/MM/YYYY (Brazilian format).
func FormatDate(t time.Time) string {
	return fmt.Sprintf("%02d/%02d/%04d", t.Day(), int(t.Month()), t.Year())
}

// TimeToExcelDate converts a time.Time to Excel serial date number
// Excel epoch is December 30, 1899
func TimeToExcelDate(t time.Time) float64 {
	epoch := time.Date(1899, 12, 30, 0, 0, 0, 0, time.UTC)
	duration := t.Sub(epoch)
	days := duration.Hours() / 24
	return days
}
