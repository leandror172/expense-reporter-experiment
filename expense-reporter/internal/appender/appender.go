package appender

import (
	"fmt"
	"time"

	"expense-reporter/internal/feedback"
)

// ExpandAndAppend expands installments and appends typed expense entries to expenses_log.jsonl.
func ExpandAndAppend(logPath, item string, date time.Time, perInstallmentValue float64, installmentCount int, expenseType, category, subcategory string) error {
	if installmentCount <= 1 {
		entry := buildEntry(item, formatDate(date), perInstallmentValue, expenseType, category, subcategory)
		return feedback.AppendExpense(logPath, entry)
	}

	for i := 1; i <= installmentCount; i++ {
		newItem := formatInstallmentItem(item, i, installmentCount)
		newDate := addMonths(date, i-1)
		entry := buildEntry(newItem, formatDate(newDate), perInstallmentValue, expenseType, category, subcategory)
		if err := feedback.AppendExpense(logPath, entry); err != nil {
			return err
		}
	}
	return nil
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
