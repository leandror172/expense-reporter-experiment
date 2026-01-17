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
	nextEmptyRow, err := excel.FindNextEmptyRow(workbookPath, mapping.SheetName, itemCol, targetRow, mapping.Subcategory)
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

// InsertBatchExpenses inserts multiple expense strings in a single batch operation
// Returns a slice of BatchErrors (one per expense) - nil for success, error for failure
// This is dramatically faster than calling InsertExpense repeatedly (20-28x speedup)
func InsertBatchExpenses(workbookPath string, expenseStrings []string) []*models.BatchError {
	// Initialize results slice (same size as input)
	errors := make([]*models.BatchError, len(expenseStrings))

	// Handle empty batch
	if len(expenseStrings) == 0 {
		return errors
	}

	// Step 1: Parse all expenses first (fail fast before file operations)
	expenses := make([]*models.Expense, len(expenseStrings))
	for i, expenseString := range expenseStrings {
		expense, err := parser.ParseExpenseString(expenseString)
		if err != nil {
			errors[i] = models.NewParseError(err.Error(), err)
			continue
		}
		expenses[i] = expense
	}

	// Step 2: Load reference sheet mappings ONCE
	mappings, err := excel.LoadReferenceSheet(workbookPath)
	if err != nil {
		// If reference sheet fails, all expenses fail
		for i := range errors {
			if errors[i] == nil { // Don't overwrite parse errors
				errors[i] = models.NewIOError("load reference sheet", err)
			}
		}
		return errors
	}

	// Step 3: Resolve all subcategories (in-memory, no file I/O)
	// Track valid expenses for batch processing
	validIndices := []int{}
	resolvedMappings := make([]*resolver.SubcategoryMapping, len(expenses))

	for i, expense := range expenses {
		if errors[i] != nil || expense == nil {
			continue // Skip already failed expenses
		}

		mapping, isAmbiguous, err := resolver.ResolveSubcategory(mappings, expense.Subcategory)
		if err != nil {
			errors[i] = models.NewResolutionError(expense.Subcategory)
			continue
		}

		if isAmbiguous {
			searchKey := expense.Subcategory
			parent := resolver.ExtractParentSubcategory(expense.Subcategory)
			if parent != expense.Subcategory {
				searchKey = parent
			}
			options := resolver.GetAmbiguousOptions(mappings, searchKey)
			errors[i] = models.NewAmbiguousError(expense.Subcategory, len(options))
			continue
		}

		resolvedMappings[i] = mapping
		validIndices = append(validIndices, i)
	}

	// If no valid expenses, return early
	if len(validIndices) == 0 {
		return errors
	}

	// Step 4: Build batch lookup requests for subcategory rows
	subcatRequests := []excel.SubcategoryLookupRequest{}
	for _, i := range validIndices {
		mapping := resolvedMappings[i]
		subcatRequests = append(subcatRequests, excel.SubcategoryLookupRequest{
			SheetName:   mapping.SheetName,
			Subcategory: mapping.Subcategory,
		})
	}

	// Step 5: Find all subcategory rows in ONE file open
	subcatRows, err := excel.FindSubcategoryRowBatch(workbookPath, subcatRequests)
	if err != nil {
		// If batch lookup fails, mark all remaining valid expenses as failed
		for _, i := range validIndices {
			errors[i] = models.NewIOError("find subcategory rows", err)
		}
		return errors
	}

	// Step 5.5: Check capacity for all subcategories (NEW - capacity detection)
	capacityRequests := []excel.CapacityCheckRequest{}
	for _, i := range validIndices {
		expense := expenses[i]
		mapping := resolvedMappings[i]
		subcatRow := subcatRows[mapping.SheetName][mapping.Subcategory]

		// Get month column
		itemCol, _, _, err := excel.GetMonthColumns(expense.Date.Month())
		if err != nil {
			errors[i] = models.NewParseError(fmt.Sprintf("invalid month: %v", err), err)
			continue
		}

		capacityRequests = append(capacityRequests, excel.CapacityCheckRequest{
			SheetName:      mapping.SheetName,
			SubcategoryRow: subcatRow,
			TotalRow:       mapping.TotalRow,
			MonthColumn:    itemCol,
		})
	}

	// Check capacity for all in ONE file open
	capacityResults, err := excel.CheckCapacityBatch(workbookPath, capacityRequests)
	if err != nil {
		// If capacity check fails, mark all as IO errors
		for _, i := range validIndices {
			if errors[i] == nil {
				errors[i] = models.NewIOError("check capacity", err)
			}
		}
		return errors
	}

	// Filter out full subcategories
	validAfterCapacity := []int{}
	for _, i := range validIndices {
		if errors[i] != nil {
			continue // Already failed (e.g., month column error)
		}

		mapping := resolvedMappings[i]
		subcatRow := subcatRows[mapping.SheetName][mapping.Subcategory]
		capacityInfo := capacityResults[mapping.SheetName][subcatRow]

		if capacityInfo.IsFull {
			// Create capacity error
			errors[i] = models.NewCapacityError(
				mapping.Subcategory,
				mapping.SheetName,
				capacityInfo.AvailableRows,
			)
			continue
		}

		validAfterCapacity = append(validAfterCapacity, i)
	}

	// If no valid expenses after capacity check, return early
	if len(validAfterCapacity) == 0 {
		return errors
	}

	// Step 6: Build batch requests for empty rows (ONLY for non-full subcategories)
	emptyRowRequests := []excel.EmptyRowRequest{}
	for _, i := range validAfterCapacity {
		expense := expenses[i]
		mapping := resolvedMappings[i]

		// Get subcategory row from batch results
		subcatRow := subcatRows[mapping.SheetName][mapping.Subcategory]

		// Get month column
		itemCol, _, _, err := excel.GetMonthColumns(expense.Date.Month())
		if err != nil {
			errors[i] = models.NewParseError(fmt.Sprintf("invalid month: %v", err), err)
			continue
		}

		emptyRowRequests = append(emptyRowRequests, excel.EmptyRowRequest{
			SheetName:      mapping.SheetName,
			ColumnLetter:   itemCol,
			StartRow:       subcatRow,
			SubcategoryName: mapping.Subcategory,
		})
	}

	// Step 7: Find all empty rows in ONE file open
	emptyRows, err := excel.FindNextEmptyRowBatch(workbookPath, emptyRowRequests)
	if err != nil {
		// If batch lookup fails, mark all remaining valid expenses as failed
		for _, i := range validAfterCapacity {
			if errors[i] == nil { // Don't overwrite month column errors
				errors[i] = models.NewIOError("find empty rows", err)
			}
		}
		return errors
	}

	// Step 8: Build expense-location pairs for batch write
	expensesWithLocations := []excel.ExpenseWithLocation{}
	finalValidIndices := []int{}

	for _, i := range validAfterCapacity {
		if errors[i] != nil {
			continue // Skip expenses that failed in month column lookup
		}

		expense := expenses[i]
		mapping := resolvedMappings[i]

		subcatRow := subcatRows[mapping.SheetName][mapping.Subcategory]
		emptyRow := emptyRows[mapping.SheetName][subcatRow]

		itemCol, _, _, _ := excel.GetMonthColumns(expense.Date.Month())

		location := &models.SheetLocation{
			SheetName:   mapping.SheetName,
			Category:    mapping.Category,
			SubcatRow:   subcatRow,
			TargetRow:   emptyRow,
			MonthColumn: itemCol,
		}

		expensesWithLocations = append(expensesWithLocations, excel.ExpenseWithLocation{
			Expense:  expense,
			Location: location,
		})
		finalValidIndices = append(finalValidIndices, i)
	}

	// Step 9: Write all valid expenses in ONE file open/save cycle
	if len(expensesWithLocations) > 0 {
		if err := excel.WriteBatchExpenses(workbookPath, expensesWithLocations); err != nil {
			// If batch write fails, mark all expenses that were going to be written as failed
			for _, i := range finalValidIndices {
				errors[i] = models.NewIOError("write expenses", err)
			}
			return errors
		}
	}

	return errors
}
