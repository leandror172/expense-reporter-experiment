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
	content.WriteString("# Ambiguous expenses - choose correct path and re-import\n")
	content.WriteString("# Replace the subcategory with the full hierarchical path\n")
	content.WriteString("# Format: Sheet,Category,Subcategory OR Category,Subcategory\n")
	content.WriteString("#\n")
	content.WriteString("# Example: Change 'Diarista' to 'Habitação,Diarista' or 'Fixas,Habitação,Diarista'\n")
	content.WriteString("#\n\n")

	// Write each ambiguous entry
	for _, entry := range entries {
		// Original expense string
		content.WriteString(entry.ExpenseString + "\n")

		// Add path suggestions as comments
		content.WriteString("# Path options:\n")
		for _, path := range entry.PathOptions {
			content.WriteString(fmt.Sprintf("#   %s\n", path))
		}
		content.WriteString("#\n")
	}

	// Write to file
	err := os.WriteFile(w.filePath, []byte(content.String()), 0644)
	if err != nil {
		return fmt.Errorf("failed to write ambiguous expenses file: %w", err)
	}

	return nil
}
