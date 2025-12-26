package batch

import (
	"os"
	"path/filepath"
	"testing"
)

// TDD RED: Test CSV reading functionality
func TestCSVReader_Read(t *testing.T) {
	tests := []struct {
		name        string
		csvContent  string
		wantLines   []string
		wantErr     bool
	}{
		{
			name: "valid CSV with multiple lines",
			csvContent: `Uber Centro;15/04;35,50;Uber/Taxi
Compras Carrefour;03/01;150,00;Supermercado
Pão francês;22/12;8,50;Padaria`,
			wantLines: []string{
				"Uber Centro;15/04;35,50;Uber/Taxi",
				"Compras Carrefour;03/01;150,00;Supermercado",
				"Pão francês;22/12;8,50;Padaria",
			},
			wantErr: false,
		},
		{
			name: "CSV with comments - should skip lines starting with #",
			csvContent: `# This is a comment
Uber Centro;15/04;35,50;Uber/Taxi
# Another comment
Compras;03/01;150,00;Supermercado`,
			wantLines: []string{
				"Uber Centro;15/04;35,50;Uber/Taxi",
				"Compras;03/01;150,00;Supermercado",
			},
			wantErr: false,
		},
		{
			name: "CSV with empty lines - should skip them",
			csvContent: `Uber Centro;15/04;35,50;Uber/Taxi

Compras;03/01;150,00;Supermercado

`,
			wantLines: []string{
				"Uber Centro;15/04;35,50;Uber/Taxi",
				"Compras;03/01;150,00;Supermercado",
			},
			wantErr: false,
		},
		{
			name: "CSV with whitespace lines - should skip",
			csvContent: `Uber Centro;15/04;35,50;Uber/Taxi


Compras;03/01;150,00;Supermercado`,
			wantLines: []string{
				"Uber Centro;15/04;35,50;Uber/Taxi",
				"Compras;03/01;150,00;Supermercado",
			},
			wantErr: false,
		},
		{
			name: "CSV with mixed comments and empty lines",
			csvContent: `# Header comment
Uber Centro;15/04;35,50;Uber/Taxi

# Mid section comment
Compras;03/01;150,00;Supermercado
# Another comment

Pão;22/12;8,50;Padaria`,
			wantLines: []string{
				"Uber Centro;15/04;35,50;Uber/Taxi",
				"Compras;03/01;150,00;Supermercado",
				"Pão;22/12;8,50;Padaria",
			},
			wantErr: false,
		},
		{
			name:        "empty CSV file - should return empty slice",
			csvContent:  "",
			wantLines:   []string{},
			wantErr:     false,
		},
		{
			name: "CSV with only comments - should return empty slice",
			csvContent: `# Comment 1
# Comment 2
# Comment 3`,
			wantLines: []string{},
			wantErr:   false,
		},
		{
			name: "CSV with only empty lines - should return empty slice",
			csvContent: `


`,
			wantLines: []string{},
			wantErr:   false,
		},
		{
			name: "single line CSV",
			csvContent: `Uber Centro;15/04;35,50;Uber/Taxi`,
			wantLines: []string{
				"Uber Centro;15/04;35,50;Uber/Taxi",
			},
			wantErr: false,
		},
		{
			name: "CSV with trailing spaces - should preserve",
			csvContent: `Uber Centro   ;15/04;35,50;Uber/Taxi
Compras  ;03/01;150,00;Supermercado  `,
			wantLines: []string{
				"Uber Centro   ;15/04;35,50;Uber/Taxi",
				"Compras  ;03/01;150,00;Supermercado",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary CSV file
			tmpFile := createTempCSV(t, tt.csvContent)
			defer os.Remove(tmpFile)

			// Create CSV reader and read
			reader := NewCSVReader(tmpFile)
			lines, err := reader.Read()

			// Check error
			if (err != nil) != tt.wantErr {
				t.Errorf("CSVReader.Read() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			// Check number of lines
			if len(lines) != len(tt.wantLines) {
				t.Errorf("CSVReader.Read() returned %d lines, want %d", len(lines), len(tt.wantLines))
				return
			}

			// Check each line
			for i, line := range lines {
				if line != tt.wantLines[i] {
					t.Errorf("CSVReader.Read() line %d = %q, want %q", i, line, tt.wantLines[i])
				}
			}
		})
	}
}

// Test file not found error
func TestCSVReader_FileNotFound(t *testing.T) {
	reader := NewCSVReader("nonexistent_file.csv")
	_, err := reader.Read()

	if err == nil {
		t.Error("CSVReader.Read() expected error for non-existent file, got nil")
	}
}

// Helper function to create temporary CSV file for testing
func createTempCSV(t *testing.T, content string) string {
	t.Helper()

	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.csv")

	err := os.WriteFile(tmpFile, []byte(content), 0644)
	if err != nil {
		t.Fatalf("Failed to create temp CSV file: %v", err)
	}

	return tmpFile
}
