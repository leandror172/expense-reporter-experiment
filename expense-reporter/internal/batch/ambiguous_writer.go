package batch

import (
	"fmt"
	"os"
	"strings"
)

// AmbiguousWriter writes ambiguous expenses to a CSV file
type AmbiguousWriter struct {
	filePath string
}

// NewAmbiguousWriter creates a new ambiguous expense writer
func NewAmbiguousWriter(filePath string) *AmbiguousWriter {
	return &AmbiguousWriter{filePath: filePath}
}

// Write writes ambiguous entries to the CSV file
// Format: <original_expense>,<sheet1>,<sheet2>,...
// If entries is empty or nil, no file is created
func (w *AmbiguousWriter) Write(entries []AmbiguousEntry) error {
	// Don't create file if no ambiguous entries
	if len(entries) == 0 {
		return nil
	}

	// Build file content
	var content strings.Builder

	// Write header comments
	content.WriteString("# Ambiguous expenses - choose the correct sheet and re-import\n")
	content.WriteString("# Format: <original_expense>,<sheet1>,<sheet2>,...\n")
	content.WriteString("# Edit this file to keep only the correct sheet, then re-import\n")

	// Write each ambiguous entry
	for _, entry := range entries {
		// Format: original_expense,sheet1,sheet2,...
		line := entry.ExpenseString + "," + strings.Join(entry.SheetOptions, ",")
		content.WriteString(line + "\n")
	}

	// Write to file
	err := os.WriteFile(w.filePath, []byte(content.String()), 0644)
	if err != nil {
		return fmt.Errorf("failed to write ambiguous expenses file: %w", err)
	}

	return nil
}
