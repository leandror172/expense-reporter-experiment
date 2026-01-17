package batch

import (
	"expense-reporter/internal/workflow"
	"fmt"
	"os"
	"strings"
	"time"
)

// RolloverWriter writes rollover installments to CSV
type RolloverWriter struct {
	baseDir string
}

// NewRolloverWriter creates a new rollover writer
func NewRolloverWriter(baseDir string) *RolloverWriter {
	if baseDir == "" {
		baseDir = "."
	}
	return &RolloverWriter{baseDir: baseDir}
}

// Write writes rollover entries to CSV
// Returns the file path if entries were written, empty string if no entries
func (w *RolloverWriter) Write(entries []workflow.RolloverExpense) (string, error) {
	if len(entries) == 0 {
		return "", nil
	}

	timestamp := time.Now().Format("20060102_150405")
	filePath := fmt.Sprintf("%s/expenses_rollover_%s.csv", w.baseDir, timestamp)

	var content strings.Builder

	// Header
	content.WriteString("# Installments for next year - import in January\n")
	content.WriteString("# Format: <item>;<date>;<value>;<subcategory>\n")
	content.WriteString("# These expenses are ready to import as-is\n\n")

	// Write entries
	for _, entry := range entries {
		expense := entry.Expense

		// Format date as DD/MM
		dateStr := fmt.Sprintf("%02d/%02d", expense.Date.Day(), expense.Date.Month())

		// Format value in PT-BR format (comma as decimal separator)
		valueStr := fmt.Sprintf("%.2f", expense.Value)
		valueStr = strings.ReplaceAll(valueStr, ".", ",")

		// Format item with installment info
		itemStr := expense.FormattedItem()

		// Write CSV line
		line := fmt.Sprintf("%s;%s;%s;%s\n", itemStr, dateStr, valueStr, expense.Subcategory)
		content.WriteString(line)
	}

	if err := os.WriteFile(filePath, []byte(content.String()), 0644); err != nil {
		return "", err
	}

	return filePath, nil
}
