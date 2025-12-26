package batch

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TDD RED: Test ambiguous expense CSV writing
func TestAmbiguousWriter_Write(t *testing.T) {
	tests := []struct {
		name               string
		entries            []AmbiguousEntry
		wantLines          []string
		wantFileCreated    bool
	}{
		{
			name: "single ambiguous entry with two sheets",
			entries: []AmbiguousEntry{
				{
					ExpenseString: "Consulta;15/04;100,00;Dentista",
					Subcategory:   "Dentista",
					SheetOptions:  []string{"Variáveis", "Extras"},
				},
			},
			wantLines: []string{
				"# Ambiguous expenses - choose the correct sheet and re-import",
				"# Format: <original_expense>,<sheet1>,<sheet2>,...",
				"# Edit this file to keep only the correct sheet, then re-import",
				"Consulta;15/04;100,00;Dentista,Variáveis,Extras",
			},
			wantFileCreated: true,
		},
		{
			name: "multiple ambiguous entries",
			entries: []AmbiguousEntry{
				{
					ExpenseString: "Consulta;15/04;100,00;Dentista",
					Subcategory:   "Dentista",
					SheetOptions:  []string{"Variáveis", "Extras"},
				},
				{
					ExpenseString: "Recarga;20/04;120,00;Gás",
					Subcategory:   "Gás",
					SheetOptions:  []string{"Variáveis", "Fixas"},
				},
			},
			wantLines: []string{
				"# Ambiguous expenses - choose the correct sheet and re-import",
				"# Format: <original_expense>,<sheet1>,<sheet2>,...",
				"# Edit this file to keep only the correct sheet, then re-import",
				"Consulta;15/04;100,00;Dentista,Variáveis,Extras",
				"Recarga;20/04;120,00;Gás,Variáveis,Fixas",
			},
			wantFileCreated: true,
		},
		{
			name: "entry with three sheet options",
			entries: []AmbiguousEntry{
				{
					ExpenseString: "Test;15/04;50,00;Orion",
					Subcategory:   "Orion",
					SheetOptions:  []string{"Variáveis", "Extras", "Adicionais"},
				},
			},
			wantLines: []string{
				"# Ambiguous expenses - choose the correct sheet and re-import",
				"# Format: <original_expense>,<sheet1>,<sheet2>,...",
				"# Edit this file to keep only the correct sheet, then re-import",
				"Test;15/04;50,00;Orion,Variáveis,Extras,Adicionais",
			},
			wantFileCreated: true,
		},
		{
			name:            "empty entries - should not create file",
			entries:         []AmbiguousEntry{},
			wantLines:       nil,
			wantFileCreated: false,
		},
		{
			name:            "nil entries - should not create file",
			entries:         nil,
			wantLines:       nil,
			wantFileCreated: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary directory for output
			tmpDir := t.TempDir()
			outputPath := filepath.Join(tmpDir, "ambiguous_expenses.csv")

			// Create writer and write
			writer := NewAmbiguousWriter(outputPath)
			err := writer.Write(tt.entries)

			if err != nil {
				t.Errorf("AmbiguousWriter.Write() error = %v, want nil", err)
				return
			}

			// Check if file was created
			_, statErr := os.Stat(outputPath)
			fileExists := statErr == nil

			if fileExists != tt.wantFileCreated {
				t.Errorf("AmbiguousWriter.Write() file created = %v, want %v", fileExists, tt.wantFileCreated)
				return
			}

			// If file should exist, verify content
			if tt.wantFileCreated {
				content, err := os.ReadFile(outputPath)
				if err != nil {
					t.Fatalf("Failed to read output file: %v", err)
				}

				lines := strings.Split(strings.TrimRight(string(content), "\n"), "\n")

				if len(lines) != len(tt.wantLines) {
					t.Errorf("AmbiguousWriter.Write() wrote %d lines, want %d", len(lines), len(tt.wantLines))
					t.Logf("Got:\n%s", content)
					return
				}

				for i, line := range lines {
					if line != tt.wantLines[i] {
						t.Errorf("AmbiguousWriter.Write() line %d = %q, want %q", i, line, tt.wantLines[i])
					}
				}
			}
		})
	}
}

// Test error handling when file cannot be created
func TestAmbiguousWriter_WriteError(t *testing.T) {
	// Try to write to invalid path
	writer := NewAmbiguousWriter("/invalid/path/that/does/not/exist/file.csv")
	entries := []AmbiguousEntry{
		{
			ExpenseString: "Test;15/04;50,00;Dentista",
			Subcategory:   "Dentista",
			SheetOptions:  []string{"Variáveis", "Extras"},
		},
	}

	err := writer.Write(entries)
	if err == nil {
		t.Error("AmbiguousWriter.Write() expected error for invalid path, got nil")
	}
}
