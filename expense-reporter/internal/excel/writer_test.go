package excel

import (
	"expense-reporter/internal/models"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/xuri/excelize/v2"
)

// TDD RED: Test writing expense to Excel
func TestWriteExpense(t *testing.T) {
	// Create a test copy of the workbook
	originalPath := getTestWorkbookPath(t)
	testPath := "test_workbook.xlsx"

	// Copy original to test file
	copyFile(originalPath, testPath)
	defer os.Remove(testPath) // Clean up after test

	expense := &models.Expense{
		Item:        "Test Uber",
		Date:        time.Date(2025, 4, 15, 0, 0, 0, 0, time.UTC),
		Value:       35.50,
		Subcategory: "Uber/Taxi",
	}

	location := &models.SheetLocation{
		SheetName:   "Variáveis",
		Category:    "Transporte",
		SubcatRow:   97,
		TargetRow:   98, // Next available row
		MonthColumn: "M", // April
	}

	err := WriteExpense(testPath, expense, location)
	if err != nil {
		t.Fatalf("WriteExpense() error = %v", err)
	}

	// Verify the data was written
	f, err := excelize.OpenFile(testPath)
	if err != nil {
		t.Fatalf("Failed to open test workbook: %v", err)
	}
	defer f.Close()

	// Check Item column (M98)
	itemCell := "M98"
	itemValue, err := f.GetCellValue("Variáveis", itemCell)
	if err != nil {
		t.Errorf("Failed to read item cell: %v", err)
	}
	if itemValue != "Test Uber" {
		t.Errorf("Item cell %s = %v, want 'Test Uber'", itemCell, itemValue)
	}

	// Check Value column (O98)
	valueCell := "O98"
	valueNum, err := f.GetCellValue("Variáveis", valueCell)
	if err != nil {
		t.Errorf("Failed to read value cell: %v", err)
	}
	// GetCellValue returns the formatted string; with our R$ currency style that's "R$ 35.50"
	if valueNum != "35.5" && valueNum != "35.50" && valueNum != "R$ 35.50" {
		t.Errorf("Value cell %s = %v, want '35.5', '35.50', or 'R$ 35.50'", valueCell, valueNum)
	}

	// Check Date column (N98) - written as serial number, displayed as dd/mm
	dateCell := "N98"
	dateValue, err := f.GetCellValue("Variáveis", dateCell)
	if err != nil {
		t.Errorf("Failed to read date cell: %v", err)
	}
	if dateValue == "" {
		t.Errorf("Date cell %s is empty", dateCell)
	}
	// With dd/mm custom format, April 15 should render as "15/04"
	if dateValue != "15/04" {
		t.Logf("Date cell %s = %q (expected dd/mm format like '15/04')", dateCell, dateValue)
	}
}

// Test rollback on error
func TestWriteExpenseRollback(t *testing.T) {
	// This should test that if writing fails, the file is not corrupted
	// For now, just verify error handling works

	invalidPath := "Z:\\NonExistent\\path\\workbook.xlsx"

	expense := &models.Expense{
		Item:        "Test",
		Date:        time.Date(2025, 4, 15, 0, 0, 0, 0, time.UTC),
		Value:       35.50,
		Subcategory: "Test",
	}

	location := &models.SheetLocation{
		SheetName:   "Variáveis",
		TargetRow:   98,
		MonthColumn: "M",
	}

	err := WriteExpense(invalidPath, expense, location)
	if err == nil {
		t.Error("WriteExpense() expected error for invalid path, got nil")
	}
}

// TDD RED: Test batch writing expenses
func TestWriteBatchExpenses(t *testing.T) {
	tests := []struct {
		name    string
		prepare func() []ExpenseWithLocation
		wantErr bool
	}{
		{
			name: "empty batch",
			prepare: func() []ExpenseWithLocation {
				return []ExpenseWithLocation{}
			},
			wantErr: false,
		},
		{
			name: "single expense",
			prepare: func() []ExpenseWithLocation {
				return []ExpenseWithLocation{
					{
						Expense: &models.Expense{
							Item:        "Test Single",
							Date:        time.Date(2025, 4, 15, 0, 0, 0, 0, time.UTC),
							Value:       50.00,
							Subcategory: "Uber/Taxi",
						},
						Location: &models.SheetLocation{
							SheetName:   "Variáveis",
							Category:    "Transporte",
							SubcatRow:   97,
							TargetRow:   98,
							MonthColumn: "M",
						},
					},
				}
			},
			wantErr: false,
		},
		{
			name: "multiple expenses same sheet",
			prepare: func() []ExpenseWithLocation {
				return []ExpenseWithLocation{
					{
						Expense: &models.Expense{
							Item:        "Uber 1",
							Date:        time.Date(2025, 4, 15, 0, 0, 0, 0, time.UTC),
							Value:       30.00,
							Subcategory: "Uber/Taxi",
						},
						Location: &models.SheetLocation{
							SheetName:   "Variáveis",
							TargetRow:   98,
							MonthColumn: "M",
						},
					},
					{
						Expense: &models.Expense{
							Item:        "Uber 2",
							Date:        time.Date(2025, 4, 16, 0, 0, 0, 0, time.UTC),
							Value:       25.00,
							Subcategory: "Uber/Taxi",
						},
						Location: &models.SheetLocation{
							SheetName:   "Variáveis",
							TargetRow:   99,
							MonthColumn: "M",
						},
					},
				}
			},
			wantErr: false,
		},
		{
			name: "nil expense",
			prepare: func() []ExpenseWithLocation {
				return []ExpenseWithLocation{
					{
						Expense:  nil,
						Location: &models.SheetLocation{},
					},
				}
			},
			wantErr: true,
		},
		{
			name: "nil location",
			prepare: func() []ExpenseWithLocation {
				return []ExpenseWithLocation{
					{
						Expense: &models.Expense{
							Item:        "Test",
							Date:        time.Date(2025, 4, 15, 0, 0, 0, 0, time.UTC),
							Value:       10.00,
							Subcategory: "Test",
						},
						Location: nil,
					},
				}
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test workbook copy
			originalPath := getTestWorkbookPath(t)
			testPath := "Z:\\Meu Drive\\controle\\code\\expense-reporter\\test_batch_workbook.xlsx"

			if err := copyFile(originalPath, testPath); err != nil {
				t.Fatalf("Failed to copy test workbook: %v", err)
			}
			defer os.Remove(testPath)

			expensesWithLocations := tt.prepare()
			err := WriteBatchExpenses(testPath, expensesWithLocations)

			if (err != nil) != tt.wantErr {
				t.Errorf("WriteBatchExpenses() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if err == nil && len(expensesWithLocations) > 0 {
				// Verify expenses were written
				f, err := excelize.OpenFile(testPath)
				if err != nil {
					t.Fatalf("Failed to open test workbook: %v", err)
				}
				defer f.Close()

				for i, ewl := range expensesWithLocations {
					if ewl.Expense == nil || ewl.Location == nil {
						continue
					}

					itemCol, _, _, _ := GetMonthColumns(ewl.Expense.Date.Month())
					itemCell := fmt.Sprintf("%s%d", itemCol, ewl.Location.TargetRow)

					itemValue, err := f.GetCellValue(ewl.Location.SheetName, itemCell)
					if err != nil {
						t.Errorf("Expense %d: failed to read item cell: %v", i, err)
					}
					if itemValue != ewl.Expense.Item {
						t.Errorf("Expense %d: item = %v, want %v", i, itemValue, ewl.Expense.Item)
					}
				}
			}
		})
	}
}

// Helper function to copy file
func copyFile(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, data, 0644)
}
