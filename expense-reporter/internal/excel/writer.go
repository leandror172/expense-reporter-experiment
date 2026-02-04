package excel

import (
	"expense-reporter/internal/logger"
	"expense-reporter/internal/models"
	"fmt"
	"time"

	"github.com/xuri/excelize/v2"
)

// boolPtr is a tiny helper because excelize CalcPropsOptions fields are *bool.
func boolPtr(b bool) *bool { return &b }

// setFullCalcOnLoad marks the workbook so that Excel / LibreOffice will
// recalculate all formulas (e.g. SUM totals) the next time it is opened.
// excelize does not update cached formula values when cells change; this is the
// correct, lightweight fix — it costs nothing at runtime and is idempotent.
func setFullCalcOnLoad(f *excelize.File) {
	if err := f.SetCalcProps(&excelize.CalcPropsOptions{
		FullCalcOnLoad: boolPtr(true),
	}); err != nil {
		logger.Warn("setFullCalcOnLoad: failed to set calc props", "error", err)
	}
}

// dateStyle creates the dd/mm custom number format style and returns its ID.
// The workbook already uses "d/m" and "d/m/yyyy" in hand-entered cells;
// "dd/mm" gives the same layout with zero-padded day and month.
func dateStyle(f *excelize.File) (int, error) {
	return f.NewStyle(&excelize.Style{
		CustomNumFmt: strPtr("dd/mm"),
	})
}

func strPtr(s string) *string { return &s }

// ExpenseWithLocation pairs an expense with its target location
// This is used by batch operations to precompute locations before writing
type ExpenseWithLocation struct {
	Expense  *models.Expense
	Location *models.SheetLocation
}

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

	// Write Item (description) - use FormattedItem() to support installments
	itemCell := fmt.Sprintf("%s%d", itemCol, targetRow)
	if err := f.SetCellValue(sheetName, itemCell, expense.FormattedItem()); err != nil {
		return fmt.Errorf("failed to write item to %s: %w", itemCell, err)
	}

	// Write Date (as Excel serial number)
	dateCell := fmt.Sprintf("%s%d", dateCol, targetRow)
	excelDate := TimeToExcelSerial(expense.Date)
	if err := f.SetCellValue(sheetName, dateCell, excelDate); err != nil {
		return fmt.Errorf("failed to write date to %s: %w", dateCell, err)
	}

	// Apply dd/mm date format
	dStyle, err := dateStyle(f)
	if err != nil {
		return fmt.Errorf("failed to create date style: %w", err)
	}
	if err := f.SetCellStyle(sheetName, dateCell, dateCell, dStyle); err != nil {
		return fmt.Errorf("failed to apply date style to %s: %w", dateCell, err)
	}

	// Write Value (numeric)
	valueCell := fmt.Sprintf("%s%d", valueCol, targetRow)
	if err := f.SetCellValue(sheetName, valueCell, expense.Value); err != nil {
		return fmt.Errorf("failed to write value to %s: %w", valueCell, err)
	}

	// Apply R$ currency format (matches the workbook's existing "R$ "#,##0.00)
	currencyStyle, err := f.NewStyle(&excelize.Style{
		CustomNumFmt: strPtr(`"R$ "#,##0.00`),
	})
	if err != nil {
		return fmt.Errorf("failed to create currency style: %w", err)
	}
	if err := f.SetCellStyle(sheetName, valueCell, valueCell, currencyStyle); err != nil {
		return fmt.Errorf("failed to apply currency style to %s: %w", valueCell, err)
	}

	// Tell Excel/LibreOffice to recalculate all formulas (SUM totals etc.) on open
	setFullCalcOnLoad(f)

	// Save the workbook
	if err := f.Save(); err != nil {
		return fmt.Errorf("failed to save workbook: %w", err)
	}

	return nil
}

// WriteBatchExpenses writes multiple expenses in a single open/save cycle
// This is dramatically faster than calling WriteExpense repeatedly (50-100x speedup)
// The file is opened once, all expenses are written, then saved once
func WriteBatchExpenses(workbookPath string, expensesWithLocations []ExpenseWithLocation) error {
	// Validate all inputs before opening file (fail fast)
	if len(expensesWithLocations) == 0 {
		return nil // Nothing to do - not an error
	}

	for i, ewl := range expensesWithLocations {
		if ewl.Expense == nil {
			return fmt.Errorf("expense at index %d is nil", i)
		}
		if ewl.Location == nil {
			return fmt.Errorf("location at index %d is nil", i)
		}
		if err := ewl.Expense.Validate(); err != nil {
			return fmt.Errorf("invalid expense at index %d: %w", i, err)
		}
	}

	// Open workbook ONCE
	f, err := excelize.OpenFile(workbookPath)
	if err != nil {
		return fmt.Errorf("failed to open workbook: %w", err)
	}
	defer func() {
		if err := f.Close(); err != nil {
			// Log but don't fail on close error
		}
	}()

	// Create reusable styles ONCE
	dStyle, err := dateStyle(f)
	if err != nil {
		return fmt.Errorf("failed to create date style: %w", err)
	}

	currencyStyle, err := f.NewStyle(&excelize.Style{
		CustomNumFmt: strPtr(`"R$ "#,##0.00`),
	})
	if err != nil {
		return fmt.Errorf("failed to create currency style: %w", err)
	}

	// Process each expense (in memory, without closing/reopening file)
	for i, ewl := range expensesWithLocations {
		expense := ewl.Expense
		location := ewl.Location

		// Get column letters for the expense's month
		itemCol, dateCol, valueCol, err := GetMonthColumns(expense.Date.Month())
		if err != nil {
			return fmt.Errorf("failed to get month columns for expense %d: %w", i, err)
		}

		sheetName := location.SheetName
		targetRow := location.TargetRow

		// Write Item (description) - use FormattedItem() to support installments
		itemCell := fmt.Sprintf("%s%d", itemCol, targetRow)
		if err := f.SetCellValue(sheetName, itemCell, expense.FormattedItem()); err != nil {
			return fmt.Errorf("failed to write item for expense %d to %s: %w", i, itemCell, err)
		}

		// Write Date (as Excel serial number)
		dateCell := fmt.Sprintf("%s%d", dateCol, targetRow)
		excelDate := TimeToExcelSerial(expense.Date)
		if err := f.SetCellValue(sheetName, dateCell, excelDate); err != nil {
			return fmt.Errorf("failed to write date for expense %d to %s: %w", i, dateCell, err)
		}

		// Apply dd/mm date style (using cached style ID)
		if err := f.SetCellStyle(sheetName, dateCell, dateCell, dStyle); err != nil {
			return fmt.Errorf("failed to apply date style for expense %d to %s: %w", i, dateCell, err)
		}

		// Write Value (numeric)
		valueCell := fmt.Sprintf("%s%d", valueCol, targetRow)
		if err := f.SetCellValue(sheetName, valueCell, expense.Value); err != nil {
			return fmt.Errorf("failed to write value for expense %d to %s: %w", i, valueCell, err)
		}

		// Apply currency style (using cached style ID)
		if err := f.SetCellStyle(sheetName, valueCell, valueCell, currencyStyle); err != nil {
			return fmt.Errorf("failed to apply currency style for expense %d to %s: %w", i, valueCell, err)
		}
	}

	// Tell Excel/LibreOffice to recalculate all formulas (SUM totals etc.) on open
	setFullCalcOnLoad(f)

	// Save the workbook ONCE
	if err := f.Save(); err != nil {
		return fmt.Errorf("failed to save workbook after writing %d expenses: %w",
			len(expensesWithLocations), err)
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
