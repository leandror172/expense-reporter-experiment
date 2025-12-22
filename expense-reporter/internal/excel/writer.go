package excel

import (
	"expense-reporter/internal/models"
	"fmt"
	"time"

	"github.com/xuri/excelize/v2"
)

// WriteExpense writes an expense to the specified location in the Excel workbook
func WriteExpense(workbookPath string, expense *models.Expense, location *models.SheetLocation) error {
	if expense == nil {
		return fmt.Errorf("expense cannot be nil")
	}
	if location == nil {
		return fmt.Errorf("location cannot be nil")
	}

	// Validate expense
	if err := expense.Validate(); err != nil {
		return fmt.Errorf("invalid expense: %w", err)
	}

	// Open workbook
	f, err := excelize.OpenFile(workbookPath)
	if err != nil {
		return fmt.Errorf("failed to open workbook: %w", err)
	}
	defer func() {
		if err := f.Close(); err != nil {
			// Log error but don't fail
		}
	}()

	// Get column letters for the expense's month
	itemCol, dateCol, valueCol, err := GetMonthColumns(expense.Date.Month())
	if err != nil {
		return fmt.Errorf("failed to get month columns: %w", err)
	}

	sheetName := location.SheetName
	targetRow := location.TargetRow

	// Write Item (description)
	itemCell := fmt.Sprintf("%s%d", itemCol, targetRow)
	if err := f.SetCellValue(sheetName, itemCell, expense.Item); err != nil {
		return fmt.Errorf("failed to write item to %s: %w", itemCell, err)
	}

	// Write Date (as Excel serial number)
	dateCell := fmt.Sprintf("%s%d", dateCol, targetRow)
	excelDate := TimeToExcelSerial(expense.Date)
	if err := f.SetCellValue(sheetName, dateCell, excelDate); err != nil {
		return fmt.Errorf("failed to write date to %s: %w", dateCell, err)
	}

	// Apply date format to the cell
	dateStyle, err := f.NewStyle(&excelize.Style{
		NumFmt: 14, // Excel date format: m/d/yy
	})
	if err != nil {
		return fmt.Errorf("failed to create date style: %w", err)
	}
	if err := f.SetCellStyle(sheetName, dateCell, dateCell, dateStyle); err != nil {
		return fmt.Errorf("failed to apply date style to %s: %w", dateCell, err)
	}

	// Write Value (numeric)
	valueCell := fmt.Sprintf("%s%d", valueCol, targetRow)
	if err := f.SetCellValue(sheetName, valueCell, expense.Value); err != nil {
		return fmt.Errorf("failed to write value to %s: %w", valueCell, err)
	}

	// Apply currency format to the cell
	currencyStyle, err := f.NewStyle(&excelize.Style{
		NumFmt: 4, // Excel currency format with 2 decimal places
	})
	if err != nil {
		return fmt.Errorf("failed to create currency style: %w", err)
	}
	if err := f.SetCellStyle(sheetName, valueCell, valueCell, currencyStyle); err != nil {
		return fmt.Errorf("failed to apply currency style to %s: %w", valueCell, err)
	}

	// Save the workbook
	if err := f.Save(); err != nil {
		return fmt.Errorf("failed to save workbook: %w", err)
	}

	return nil
}

// TimeToExcelSerial converts a Go time.Time to Excel serial number
// Excel dates are stored as days since 1899-12-30 (Excel's epoch)
func TimeToExcelSerial(t time.Time) float64 {
	// Excel epoch: December 30, 1899
	excelEpoch := time.Date(1899, 12, 30, 0, 0, 0, 0, time.UTC)

	// Calculate days difference
	duration := t.Sub(excelEpoch)
	days := duration.Hours() / 24

	return days
}
