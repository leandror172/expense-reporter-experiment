package batch

import (
	"errors"
	"expense-reporter/internal/models"
	"expense-reporter/internal/resolver"
	"testing"
)

// TDD RED: Test batch processor functionality

// createTestMappings creates a test mapping set for unit tests
func createTestMappings() map[string][]resolver.SubcategoryMapping {
	return map[string][]resolver.SubcategoryMapping{
		// Unambiguous mappings
		"Uber/Taxi": {
			{Subcategory: "Uber/Taxi", SheetName: "Variáveis", Category: "Transporte", RowNumber: 10},
		},
		"Supermercado": {
			{Subcategory: "Supermercado", SheetName: "Variáveis", Category: "Alimentação", RowNumber: 20},
		},
		"Padaria": {
			{Subcategory: "Padaria", SheetName: "Variáveis", Category: "Alimentação", RowNumber: 21},
		},
		// Ambiguous mapping - appears in multiple sheets
		"Dentista": {
			{Subcategory: "Dentista", SheetName: "Variáveis", Category: "Saúde", RowNumber: 30},
			{Subcategory: "Dentista", SheetName: "Extras", Category: "Saúde", RowNumber: 15},
		},
	}
}

// Mock insert function for testing
type mockInsertFunc struct {
	calls      []mockInsertCall
	shouldFail bool
	failError  error
}

type mockInsertCall struct {
	workbookPath string
	expense      *models.Expense
}

func (m *mockInsertFunc) Insert(workbookPath string, expense *models.Expense) error {
	m.calls = append(m.calls, mockInsertCall{
		workbookPath: workbookPath,
		expense:      expense,
	})
	if m.shouldFail {
		return m.failError
	}
	return nil
}

// Mock progress callback for testing
type mockProgressCallback struct {
	calls []mockProgressCall
}

type mockProgressCall struct {
	current int
	total   int
}

func (m *mockProgressCallback) Callback(current, total int) {
	m.calls = append(m.calls, mockProgressCall{
		current: current,
		total:   total,
	})
}

func TestProcessor_Process(t *testing.T) {
	tests := []struct {
		name               string
		expenseStrings     []string
		workbookPath       string
		insertShouldFail   bool
		insertFailError    error
		wantSuccessCount   int
		wantErrorCount     int
		wantAmbiguousCount int
		wantInsertCalls    int
		wantProgressCalls  int
	}{
		{
			name: "single valid expense - should parse, resolve, and insert",
			expenseStrings: []string{
				"Uber Centro;15/04;35,50;Uber/Taxi",
			},
			workbookPath:       "test.xlsx",
			insertShouldFail:   false,
			wantSuccessCount:   1,
			wantErrorCount:     0,
			wantAmbiguousCount: 0,
			wantInsertCalls:    1,
			wantProgressCalls:  1,
		},
		{
			name: "multiple valid expenses - should process all",
			expenseStrings: []string{
				"Uber Centro;15/04;35,50;Uber/Taxi",
				"Compras;03/01;150,00;Supermercado",
				"Pão;22/12;8,50;Padaria",
			},
			workbookPath:       "test.xlsx",
			insertShouldFail:   false,
			wantSuccessCount:   3,
			wantErrorCount:     0,
			wantAmbiguousCount: 0,
			wantInsertCalls:    3,
			wantProgressCalls:  3,
		},
		{
			name: "invalid expense format - should collect error and continue",
			expenseStrings: []string{
				"Uber Centro;15/04;35,50;Uber/Taxi",
				"Invalid format here",
				"Compras;03/01;150,00;Supermercado",
			},
			workbookPath:       "test.xlsx",
			insertShouldFail:   false,
			wantSuccessCount:   2,
			wantErrorCount:     1,
			wantAmbiguousCount: 0,
			wantInsertCalls:    2, // Only valid expenses inserted
			wantProgressCalls:  3, // Progress called for all
		},
		{
			name: "insert failure - should collect error and continue",
			expenseStrings: []string{
				"Uber Centro;15/04;35,50;Uber/Taxi",
				"Compras;03/01;150,00;Supermercado",
			},
			workbookPath:       "test.xlsx",
			insertShouldFail:   true,
			insertFailError:    errors.New("Excel write failed"),
			wantSuccessCount:   0,
			wantErrorCount:     2,
			wantAmbiguousCount: 0,
			wantInsertCalls:    2, // Attempted both
			wantProgressCalls:  2,
		},
		{
			name: "ambiguous subcategory - should skip insert and mark as ambiguous",
			expenseStrings: []string{
				// "Dentista" appears in both Variáveis and Extras sheets
				"Consulta;15/04;100,00;Dentista",
			},
			workbookPath:       "test.xlsx",
			insertShouldFail:   false,
			wantSuccessCount:   0,
			wantErrorCount:     0,
			wantAmbiguousCount: 1,
			wantInsertCalls:    0, // Ambiguous expenses not inserted
			wantProgressCalls:  1,
		},
		{
			name: "mixed success, error, and ambiguous",
			expenseStrings: []string{
				"Uber Centro;15/04;35,50;Uber/Taxi",       // Success
				"Invalid",                                  // Error
				"Consulta;15/04;100,00;Dentista",          // Ambiguous
				"Compras;03/01;150,00;Supermercado",       // Success
			},
			workbookPath:       "test.xlsx",
			insertShouldFail:   false,
			wantSuccessCount:   2,
			wantErrorCount:     1,
			wantAmbiguousCount: 1,
			wantInsertCalls:    2,
			wantProgressCalls:  4,
		},
		{
			name:               "empty expense strings - should return empty summary",
			expenseStrings:     []string{},
			workbookPath:       "test.xlsx",
			insertShouldFail:   false,
			wantSuccessCount:   0,
			wantErrorCount:     0,
			wantAmbiguousCount: 0,
			wantInsertCalls:    0,
			wantProgressCalls:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock insert function
			mockInsert := &mockInsertFunc{
				shouldFail: tt.insertShouldFail,
				failError:  tt.insertFailError,
			}

			// Create mock progress callback
			mockProgress := &mockProgressCallback{}

			// Create processor with test mappings
			processor := NewProcessor(tt.workbookPath)
			processor.SetMappings(createTestMappings())

			summary, err := processor.Process(
				tt.expenseStrings,
				mockInsert.Insert,
				mockProgress.Callback,
			)

			// Should not return error - errors are collected in results
			if err != nil {
				t.Errorf("Processor.Process() returned unexpected error: %v", err)
				return
			}

			// Verify summary counts
			if summary.TotalLines != len(tt.expenseStrings) {
				t.Errorf("summary.TotalLines = %d, want %d", summary.TotalLines, len(tt.expenseStrings))
			}
			if summary.SuccessCount != tt.wantSuccessCount {
				t.Errorf("summary.SuccessCount = %d, want %d", summary.SuccessCount, tt.wantSuccessCount)
			}
			if summary.ErrorCount != tt.wantErrorCount {
				t.Errorf("summary.ErrorCount = %d, want %d", summary.ErrorCount, tt.wantErrorCount)
			}
			if summary.AmbiguousCount != tt.wantAmbiguousCount {
				t.Errorf("summary.AmbiguousCount = %d, want %d", summary.AmbiguousCount, tt.wantAmbiguousCount)
			}

			// Verify insert was called correct number of times
			if len(mockInsert.calls) != tt.wantInsertCalls {
				t.Errorf("Insert called %d times, want %d times", len(mockInsert.calls), tt.wantInsertCalls)
			}

			// Verify progress callback was called correct number of times
			if len(mockProgress.calls) != tt.wantProgressCalls {
				t.Errorf("Progress callback called %d times, want %d times", len(mockProgress.calls), tt.wantProgressCalls)
			}

			// Verify progress callback called with correct values
			for i, call := range mockProgress.calls {
				expectedCurrent := i + 1
				expectedTotal := len(tt.expenseStrings)
				if call.current != expectedCurrent || call.total != expectedTotal {
					t.Errorf("Progress call %d: got (%d, %d), want (%d, %d)",
						i, call.current, call.total, expectedCurrent, expectedTotal)
				}
			}

			// Verify results slice length
			if len(summary.Results) != len(tt.expenseStrings) {
				t.Errorf("len(summary.Results) = %d, want %d", len(summary.Results), len(tt.expenseStrings))
			}

			// Verify each result has correct line number
			for i, result := range summary.Results {
				expectedLineNum := i + 1
				if result.LineNumber != expectedLineNum {
					t.Errorf("Result[%d].LineNumber = %d, want %d", i, result.LineNumber, expectedLineNum)
				}
			}
		})
	}
}

// Test that processor works without progress callback (optional)
func TestProcessor_ProcessWithoutProgressCallback(t *testing.T) {
	expenseStrings := []string{
		"Uber Centro;15/04;35,50;Uber/Taxi",
		"Compras;03/01;150,00;Supermercado",
	}

	mockInsert := &mockInsertFunc{}
	processor := NewProcessor("test.xlsx")
	processor.SetMappings(createTestMappings())

	// Pass nil for progress callback
	summary, err := processor.Process(expenseStrings, mockInsert.Insert, nil)

	if err != nil {
		t.Errorf("Processor.Process() with nil callback returned error: %v", err)
	}

	if summary.SuccessCount != 2 {
		t.Errorf("summary.SuccessCount = %d, want 2", summary.SuccessCount)
	}
}

// Test that ambiguous results contain mapping options
func TestProcessor_AmbiguousResultsContainOptions(t *testing.T) {
	expenseStrings := []string{
		"Consulta;15/04;100,00;Dentista", // Ambiguous: appears in Variáveis and Extras
	}

	mockInsert := &mockInsertFunc{}
	processor := NewProcessor("test.xlsx")
	processor.SetMappings(createTestMappings())

	summary, err := processor.Process(expenseStrings, mockInsert.Insert, nil)

	if err != nil {
		t.Fatalf("Processor.Process() returned error: %v", err)
	}

	if summary.AmbiguousCount != 1 {
		t.Fatalf("summary.AmbiguousCount = %d, want 1", summary.AmbiguousCount)
	}

	// Find the ambiguous result
	var ambiguousResult *BatchResult
	for i := range summary.Results {
		if summary.Results[i].IsAmbiguous {
			ambiguousResult = &summary.Results[i]
			break
		}
	}

	if ambiguousResult == nil {
		t.Fatal("Expected to find ambiguous result in summary.Results")
	}

	// Verify it has mapping options
	if len(ambiguousResult.AmbiguousOpts) < 2 {
		t.Errorf("ambiguousResult.AmbiguousOpts has %d options, want at least 2",
			len(ambiguousResult.AmbiguousOpts))
	}

	// Verify options contain sheet names
	hasVariaveis := false
	hasExtras := false
	for _, opt := range ambiguousResult.AmbiguousOpts {
		if opt.SheetName == "Variáveis" {
			hasVariaveis = true
		}
		if opt.SheetName == "Extras" {
			hasExtras = true
		}
	}

	if !hasVariaveis || !hasExtras {
		t.Error("Expected ambiguous options to contain both 'Variáveis' and 'Extras' sheets")
	}
}

// Test error result details
func TestProcessor_ErrorResultDetails(t *testing.T) {
	expenseStrings := []string{
		"Invalid format",
	}

	mockInsert := &mockInsertFunc{}
	processor := NewProcessor("test.xlsx")
	processor.SetMappings(createTestMappings())

	summary, err := processor.Process(expenseStrings, mockInsert.Insert, nil)

	if err != nil {
		t.Fatalf("Processor.Process() returned error: %v", err)
	}

	if summary.ErrorCount != 1 {
		t.Fatalf("summary.ErrorCount = %d, want 1", summary.ErrorCount)
	}

	result := summary.Results[0]
	if result.Success {
		t.Error("Error result should have Success = false")
	}
	if result.Error == nil {
		t.Error("Error result should have Error != nil")
	}
	if result.ExpenseString != "Invalid format" {
		t.Errorf("result.ExpenseString = %q, want %q", result.ExpenseString, "Invalid format")
	}
}
