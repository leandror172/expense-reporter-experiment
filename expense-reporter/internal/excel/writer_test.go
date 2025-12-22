package excel

import (
	"expense-reporter/internal/models"
	"os"
	"testing"
	"time"

	"github.com/xuri/excelize/v2"
)

// TDD RED: Test writing expense to Excel
func TestWriteExpense(t *testing.T) {
	// Create a test copy of the workbook
	originalPath := "Z:\\Meu Drive\\controle\\code\\Planilha_BMeFBovespa_Leandro_OrcamentoPessoal-2025.xlsx"
	testPath := "Z:\\Meu Drive\\controle\\code\\expense-reporter\\test_workbook.xlsx"

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
	// Excel may format as "35.5" or "35.50" depending on currency formatting
	if valueNum != "35.5" && valueNum != "35.50" {
		t.Errorf("Value cell %s = %v, want '35.5' or '35.50'", valueCell, valueNum)
	}

	// Check Date column (N98) - should be Excel serial number
	dateCell := "N98"
	dateValue, err := f.GetCellValue("Variáveis", dateCell)
	if err != nil {
		t.Errorf("Failed to read date cell: %v", err)
	}
	if dateValue == "" {
		t.Errorf("Date cell %s is empty", dateCell)
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

// Helper function to copy file
func copyFile(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, data, 0644)
}
