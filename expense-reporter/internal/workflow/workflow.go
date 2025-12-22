package workflow

import (
	"expense-reporter/internal/excel"
	"expense-reporter/internal/models"
	"expense-reporter/internal/parser"
	"expense-reporter/internal/resolver"
	"fmt"
)

// InsertExpense is the main workflow function that parses, resolves, and inserts an expense
func InsertExpense(workbookPath, expenseString string) error {
	// Step 1: Parse the expense string
	expense, err := parser.ParseExpenseString(expenseString)
	if err != nil {
		return fmt.Errorf("failed to parse expense: %w", err)
	}

	// Step 2: Load reference sheet mappings
	mappings, err := excel.LoadReferenceSheet(workbookPath)
	if err != nil {
		return fmt.Errorf("failed to load reference sheet: %w", err)
	}

	// Step 3: Resolve subcategory to sheet location
	mapping, isAmbiguous, err := resolver.ResolveSubcategory(mappings, expense.Subcategory)
	if err != nil {
		return fmt.Errorf("failed to resolve subcategory: %w", err)
	}

	// Step 4: Handle ambiguous subcategories
	if isAmbiguous {
		// Try with parent if it's a detailed subcategory
		searchKey := expense.Subcategory
		parent := resolver.ExtractParentSubcategory(expense.Subcategory)
		if parent != expense.Subcategory {
			// Use parent for ambiguous lookup
			searchKey = parent
		}
		options := resolver.GetAmbiguousOptions(mappings, searchKey)
		return fmt.Errorf("subcategory '%s' is ambiguous, found in %d sheets: please specify which one to use",
			expense.Subcategory, len(options))
	}

	// Step 5: Find the target row in the sheet
	targetRow, err := excel.FindSubcategoryRow(workbookPath, mapping.SheetName, mapping.Subcategory)
	if err != nil {
		return fmt.Errorf("failed to find subcategory row: %w", err)
	}

	// Step 6: Get month columns for the expense date
	itemCol, _, _, err := excel.GetMonthColumns(expense.Date.Month())
	if err != nil {
		return fmt.Errorf("failed to get month columns: %w", err)
	}

	// Step 7: Find next empty row in the subcategory section
	nextEmptyRow, err := excel.FindNextEmptyRow(workbookPath, mapping.SheetName, itemCol, targetRow)
	if err != nil {
		return fmt.Errorf("failed to find empty row: %w", err)
	}

	// Step 8: Create sheet location
	location := &models.SheetLocation{
		SheetName:   mapping.SheetName,
		Category:    mapping.Category,
		SubcatRow:   targetRow,
		TargetRow:   nextEmptyRow,
		MonthColumn: itemCol,
	}

	// Step 9: Write the expense to Excel
	if err := excel.WriteExpense(workbookPath, expense, location); err != nil {
		return fmt.Errorf("failed to write expense: %w", err)
	}

	return nil
}
