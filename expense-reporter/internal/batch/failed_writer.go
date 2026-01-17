package batch

import (
	"expense-reporter/internal/models"
	"fmt"
	"os"
	"strings"
	"time"
)

// FailedEntry represents a failed expense with error message
type FailedEntry struct {
	ExpenseString string
	ErrorMessage  string
	ErrorCategory models.ErrorCategory
}

// FailedWriter writes failed expenses to CSV files
type FailedWriter struct {
	baseDir string
}

// NewFailedWriter creates a new failed expense writer
func NewFailedWriter(baseDir string) *FailedWriter {
	if baseDir == "" {
		baseDir = "."
	}
	return &FailedWriter{baseDir: baseDir}
}

// Write writes failed and ambiguous entries to separate CSV files
// Returns paths to created files (empty string if no file created)
func (w *FailedWriter) Write(entries []FailedEntry, ambiguousEntries []AmbiguousEntry) (failedPath, ambiguousPath string, err error) {
	timestamp := time.Now().Format("20060102_150405")

	// Write failed CSV (non-ambiguous errors)
	failedEntries := []FailedEntry{}
	for _, entry := range entries {
		if entry.ErrorCategory != models.ErrorCategoryAmbiguous {
			failedEntries = append(failedEntries, entry)
		}
	}

	if len(failedEntries) > 0 {
		failedPath = fmt.Sprintf("%s/expenses_failed_%s.csv", w.baseDir, timestamp)
		if err := w.writeFailedCSV(failedPath, failedEntries); err != nil {
			return "", "", fmt.Errorf("failed to write failed CSV: %w", err)
		}
	}

	// Write ambiguous CSV (existing mechanism)
	if len(ambiguousEntries) > 0 {
		ambiguousPath = fmt.Sprintf("%s/expenses_ambiguous_%s.csv", w.baseDir, timestamp)
		writer := NewAmbiguousWriter(ambiguousPath)
		if err := writer.Write(ambiguousEntries); err != nil {
			return failedPath, "", fmt.Errorf("failed to write ambiguous CSV: %w", err)
		}
	}

	return failedPath, ambiguousPath, nil
}

// writeFailedCSV writes failed entries to CSV with semicolon delimiter
func (w *FailedWriter) writeFailedCSV(filePath string, entries []FailedEntry) error {
	var content strings.Builder

	// Write header comments
	content.WriteString("# Failed expenses - fix errors and re-import\n")
	content.WriteString("# Format: <original_expense>;<error_message>\n")
	content.WriteString("# Fix the issues and re-import this file\n\n")

	// Write each entry with semicolon delimiter (PT-BR format)
	for _, entry := range entries {
		// Escape semicolons in error message (replace with comma)
		errorMsg := strings.ReplaceAll(entry.ErrorMessage, ";", ",")
		line := fmt.Sprintf("%s;%s\n", entry.ExpenseString, errorMsg)
		content.WriteString(line)
	}

	// Write to file
	if err := os.WriteFile(filePath, []byte(content.String()), 0644); err != nil {
		return err
	}

	return nil
}
