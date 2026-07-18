package appender

import (
	"fmt"
	"time"

	"expense-reporter/internal/feedback"
)

// ExpandAndAppend expands installments and appends typed expense entries to expenses_log.jsonl.
func ExpandAndAppend(logPath, item string, date time.Time, perInstallmentValue float64, installmentCount int, expenseType, category, subcategory string) error {
	entries := expandEntries(item, date, perInstallmentValue, installmentCount, expenseType, category, subcategory)
	for _, entry := range entries {
		if err := feedback.AppendExpense(logPath, entry); err != nil {
			return err
		}
	}
	return nil
}

// PredictEntryIDs returns the IDs of all expense entries that would be written by ExpandAndAppend
// for the same parameters. It shares the same logic as expandEntries to ensure consistency.
func PredictEntryIDs(item string, date time.Time, perInstallmentValue float64, installmentCount int) []string {
	entries := expandEntries(item, date, perInstallmentValue, installmentCount, "", "", "")
	ids := make([]string, len(entries))
	for i, entry := range entries {
		ids[i] = entry.ID
	}
	return ids
}

// expandEntries returns the full ordered slice of entries that should be logged.
func expandEntries(item string, date time.Time, perInstallmentValue float64, installmentCount int, expenseType, category, subcategory string) []feedback.ExpenseEntry {
	if installmentCount <= 1 {
		entry := buildEntry(item, formatDate(date), perInstallmentValue, expenseType, category, subcategory)
		return []feedback.ExpenseEntry{entry}
	}

	entries := make([]feedback.ExpenseEntry, 0, installmentCount)
	for i := 1; i <= installmentCount; i++ {
		newItem := formatInstallmentItem(item, i, installmentCount)
		newDate := addMonths(date, i-1)
		entry := buildEntry(newItem, formatDate(newDate), perInstallmentValue, expenseType, category, subcategory)
		entries = append(entries, entry)
	}
	return entries
}

func addMonths(t time.Time, n int) time.Time {
	year := t.Year()
	month := t.Month()
	day := t.Day()

	for i := 0; i < n; i++ {
		if month == 12 {
			year++
			month = 1
		} else {
			month++
		}
	}

	// Handle day overflow (e.g., Jan 31 -> Feb 28)
	lastDayOfMonth := time.Date(year, month+1, 0, 0, 0, 0, 0, time.UTC).Day()
	if day > lastDayOfMonth {
		day = lastDayOfMonth
	}

	return time.Date(year, month, day, t.Hour(), t.Minute(), t.Second(), t.Nanosecond(), t.Location())
}

func formatInstallmentItem(item string, current, total int) string {
	return item + fmt.Sprintf(" (%d/%d)", current, total)
}

func formatDate(t time.Time) string {
	return t.Format("02/01/2006")
}

func buildEntry(item, dateStr string, value float64, expenseType, category, subcategory string) feedback.ExpenseEntry {
	entry := feedback.NewExpenseEntry(item, dateStr, value, subcategory, category)
	entry.Type = expenseType
	return entry
}
