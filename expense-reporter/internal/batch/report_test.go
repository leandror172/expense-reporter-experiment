package batch

import (
	"errors"
	"expense-reporter/internal/resolver"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TDD RED: Test report writing functionality

func TestReportWriter_Write(t *testing.T) {
	tests := []struct {
		name          string
		summary       *BatchSummary
		sourcePath    string
		wantSections  []string // Sections that must appear in report
		wantNotInFile []string // Strings that should NOT appear
	}{
		{
			name: "successful batch - all insertions succeeded",
			summary: &BatchSummary{
				TotalLines:   3,
				SuccessCount: 3,
				ErrorCount:   0,
				AmbiguousCount: 0,
				Results: []BatchResult{
					{LineNumber: 1, ExpenseString: "Uber Centro;15/04;35,50;Uber/Taxi", Success: true},
					{LineNumber: 2, ExpenseString: "Compras;03/01;150,00;Supermercado", Success: true},
					{LineNumber: 3, ExpenseString: "Pão;22/12;8,50;Padaria", Success: true},
				},
			},
			sourcePath: "expenses.csv",
			wantSections: []string{
				"Batch Import Report",
				"Summary Statistics",
				"Total lines processed: 3",
				"Successful: 3",
				"Failed: 0",
				"Ambiguous: 0",
				"Successful Insertions (3)",
				"Line 1: Uber Centro;15/04;35,50;Uber/Taxi",
				"Line 2: Compras;03/01;150,00;Supermercado",
				"Line 3: Pão;22/12;8,50;Padaria",
			},
			wantNotInFile: []string{
				"Errors",
				"Ambiguous Entries",
			},
		},
		{
			name: "batch with errors",
			summary: &BatchSummary{
				TotalLines:   3,
				SuccessCount: 1,
				ErrorCount:   2,
				AmbiguousCount: 0,
				Results: []BatchResult{
					{LineNumber: 1, ExpenseString: "Uber Centro;15/04;35,50;Uber/Taxi", Success: true},
					{LineNumber: 2, ExpenseString: "Invalid", Success: false, Error: errors.New("expense string must have 4 fields")},
					{LineNumber: 3, ExpenseString: "Bad;Format", Success: false, Error: errors.New("invalid date format")},
				},
			},
			sourcePath: "expenses.csv",
			wantSections: []string{
				"Summary Statistics",
				"Successful: 1",
				"Failed: 2",
				"Errors (2)",
				"Line 2: Invalid",
				"Error: expense string must have 4 fields",
				"Line 3: Bad;Format",
				"Error: invalid date format",
			},
		},
		{
			name: "batch with ambiguous entries",
			summary: &BatchSummary{
				TotalLines:   2,
				SuccessCount: 1,
				ErrorCount:   0,
				AmbiguousCount: 1,
				Results: []BatchResult{
					{LineNumber: 1, ExpenseString: "Uber Centro;15/04;35,50;Uber/Taxi", Success: true},
					{
						LineNumber:    2,
						ExpenseString: "Consulta;15/04;100,00;Dentista",
						IsAmbiguous:   true,
						AmbiguousOpts: []resolver.SubcategoryMapping{
							{Subcategory: "Dentista", SheetName: "Variáveis"},
							{Subcategory: "Dentista", SheetName: "Extras"},
						},
					},
				},
			},
			sourcePath: "expenses.csv",
			wantSections: []string{
				"Ambiguous: 1",
				"Ambiguous Entries (1)",
				"Line 2: Consulta;15/04;100,00;Dentista",
				"Options: Variáveis, Extras",
			},
		},
		{
			name: "mixed success, errors, and ambiguous",
			summary: &BatchSummary{
				TotalLines:   5,
				SuccessCount: 2,
				ErrorCount:   2,
				AmbiguousCount: 1,
				Results: []BatchResult{
					{LineNumber: 1, ExpenseString: "Uber Centro;15/04;35,50;Uber/Taxi", Success: true},
					{LineNumber: 2, ExpenseString: "Invalid", Success: false, Error: errors.New("parse error")},
					{
						LineNumber:    3,
						ExpenseString: "Consulta;15/04;100,00;Dentista",
						IsAmbiguous:   true,
						AmbiguousOpts: []resolver.SubcategoryMapping{
							{Subcategory: "Dentista", SheetName: "Variáveis"},
							{Subcategory: "Dentista", SheetName: "Extras"},
						},
					},
					{LineNumber: 4, ExpenseString: "Compras;03/01;150,00;Supermercado", Success: true},
					{LineNumber: 5, ExpenseString: "Bad", Success: false, Error: errors.New("another error")},
				},
			},
			sourcePath: "expenses.csv",
			wantSections: []string{
				"Total lines processed: 5",
				"Successful: 2",
				"Failed: 2",
				"Ambiguous: 1",
				"Successful Insertions (2)",
				"Errors (2)",
				"Ambiguous Entries (1)",
			},
		},
		{
			name: "empty batch - no results",
			summary: &BatchSummary{
				TotalLines:   0,
				SuccessCount: 0,
				ErrorCount:   0,
				AmbiguousCount: 0,
				Results:      []BatchResult{},
			},
			sourcePath: "empty.csv",
			wantSections: []string{
				"Batch Import Report",
				"Total lines processed: 0",
				"Successful: 0",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary directory for output
			tmpDir := t.TempDir()
			reportPath := filepath.Join(tmpDir, "report.txt")

			// Create writer and write
			writer := NewReportWriter(reportPath)
			err := writer.Write(tt.summary, tt.sourcePath)

			if err != nil {
				t.Errorf("ReportWriter.Write() error = %v, want nil", err)
				return
			}

			// Read the report file
			content, err := os.ReadFile(reportPath)
			if err != nil {
				t.Fatalf("Failed to read report file: %v", err)
			}

			reportText := string(content)

			// Verify all expected sections appear
			for _, section := range tt.wantSections {
				if !strings.Contains(reportText, section) {
					t.Errorf("Report missing expected section: %q\nReport:\n%s", section, reportText)
				}
			}

			// Verify unwanted sections don't appear
			for _, unwanted := range tt.wantNotInFile {
				if strings.Contains(reportText, unwanted) {
					t.Errorf("Report contains unwanted section: %q", unwanted)
				}
			}
		})
	}
}

// Test report includes timestamp
func TestReportWriter_IncludesTimestamp(t *testing.T) {
	summary := &BatchSummary{
		TotalLines:   1,
		SuccessCount: 1,
		Results: []BatchResult{
			{LineNumber: 1, ExpenseString: "Test;15/04;35,50;Uber/Taxi", Success: true},
		},
	}

	tmpDir := t.TempDir()
	reportPath := filepath.Join(tmpDir, "report.txt")

	writer := NewReportWriter(reportPath)
	beforeWrite := time.Now()
	err := writer.Write(summary, "test.csv")

	if err != nil {
		t.Fatalf("ReportWriter.Write() error = %v", err)
	}

	content, err := os.ReadFile(reportPath)
	if err != nil {
		t.Fatalf("Failed to read report: %v", err)
	}

	reportText := string(content)

	// Check that report contains a date line
	if !strings.Contains(reportText, "Date:") {
		t.Error("Report should contain 'Date:' timestamp")
	}

	// Verify timestamp is reasonable (between before and after)
	// Just check that it contains year and common time format elements
	currentYear := beforeWrite.Format("2006")
	if !strings.Contains(reportText, currentYear) {
		t.Errorf("Report timestamp should contain current year %s", currentYear)
	}
}

// Test report percentages are calculated correctly
func TestReportWriter_Percentages(t *testing.T) {
	summary := &BatchSummary{
		TotalLines:     10,
		SuccessCount:   6,
		ErrorCount:     3,
		AmbiguousCount: 1,
		Results:        []BatchResult{}, // Don't need actual results for this test
	}

	tmpDir := t.TempDir()
	reportPath := filepath.Join(tmpDir, "report.txt")

	writer := NewReportWriter(reportPath)
	err := writer.Write(summary, "test.csv")

	if err != nil {
		t.Fatalf("ReportWriter.Write() error = %v", err)
	}

	content, err := os.ReadFile(reportPath)
	if err != nil {
		t.Fatalf("Failed to read report: %v", err)
	}

	reportText := string(content)

	// Check for percentage values
	// Success: 6/10 = 60%
	if !strings.Contains(reportText, "60.00%") {
		t.Error("Report should show 60.00% for success rate")
	}

	// Failed: 3/10 = 30%
	if !strings.Contains(reportText, "30.00%") {
		t.Error("Report should show 30.00% for error rate")
	}

	// Ambiguous: 1/10 = 10%
	if !strings.Contains(reportText, "10.00%") {
		t.Error("Report should show 10.00% for ambiguous rate")
	}
}

// Test error handling when file cannot be created
func TestReportWriter_WriteError(t *testing.T) {
	summary := &BatchSummary{
		TotalLines:   1,
		SuccessCount: 1,
		Results:      []BatchResult{{LineNumber: 1, ExpenseString: "Test", Success: true}},
	}

	// Try to write to invalid path
	writer := NewReportWriter("/invalid/path/that/does/not/exist/report.txt")
	err := writer.Write(summary, "test.csv")

	if err == nil {
		t.Error("ReportWriter.Write() expected error for invalid path, got nil")
	}
}
