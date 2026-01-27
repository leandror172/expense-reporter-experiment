package workflow

import (
	"expense-reporter/internal/excel"
	"expense-reporter/internal/models"
	"expense-reporter/internal/parser"
	"expense-reporter/internal/resolver"
	"fmt"
	"strings"
	"time"
)

// InsertExpense is the main workflow function that parses, resolves, and inserts an expense
func InsertExpense(workbookPath, expenseString string) error {
	// Step 1: Parse the expense string
	expense, err := parser.ParseExpenseString(expenseString)
	if err != nil {
		return fmt.Errorf("failed to parse expense: %w", err)
	}

	// Step 2: Load reference sheet mappings and build hierarchical index
	mappings, err := excel.LoadReferenceSheet(workbookPath)
	if err != nil {
		return fmt.Errorf("failed to load reference sheet: %w", err)
	}
	pathIndex := excel.BuildPathIndex(mappings)

	// Step 3: Resolve subcategory to sheet location using hierarchical path
	mapping, isAmbiguous, err := resolver.ResolveSubcategoryWithPath(pathIndex, expense.Subcategory)
	if err != nil {
		return fmt.Errorf("failed to resolve subcategory: %w", err)
	}

	// Step 4: Handle ambiguous subcategories
	if isAmbiguous {
		options := resolver.GetAmbiguousOptions(pathIndex.BySubcategory, expense.Subcategory)
		sheetNames := make([]string, len(options))
		for i, opt := range options {
			sheetNames[i] = opt.SheetName
		}
		return fmt.Errorf("ambiguous subcategory '%s' found in sheets: [%s]. Use hierarchical path like 'Sheet,Category,Subcategory' to disambiguate",
			expense.Subcategory, strings.Join(sheetNames, ", "))
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
// Also returns rollover expenses that cross year boundary
func InsertBatchExpenses(workbookPath string, expenseStrings []string) ([]*models.BatchError, []RolloverExpense) {
	// Initialize results slice (same size as input)
	originalErrors := make([]*models.BatchError, len(expenseStrings))

	// Handle empty batch
	if len(expenseStrings) == 0 {
		return originalErrors, nil
	}

	// Step 1: Parse all expenses first (fail fast before file operations)
	parsedExpenses := make([]*models.Expense, len(expenseStrings))
	for i, expenseString := range expenseStrings {
		expense, err := parser.ParseExpenseString(expenseString)
		if err != nil {
			originalErrors[i] = models.NewParseError(err.Error(), err)
			continue
		}
		parsedExpenses[i] = expense
	}

	// Step 1.5: Expand installments
	// Track mapping: originalIndex → [expandedIndices]
	// Track rollover installments for next year
	expenses := []*models.Expense{}
	indexMapping := make(map[int][]int) // original → expanded indices
	allRollovers := []RolloverExpense{}

	for i, parsedExpense := range parsedExpenses {
		if originalErrors[i] != nil || parsedExpense == nil {
			// Keep original index mapping for errors
			indexMapping[i] = []int{len(expenses)}
			expenses = append(expenses, nil)
			continue
		}

		if parsedExpense.IsInstallment() {
			// Expand into multiple expenses
			expandedList, rollovers := expandInstallments(parsedExpense, i)
			startIdx := len(expenses)

			// Add this-year installments to processing queue
			for j := range expandedList {
				expenses = append(expenses, expandedList[j])
				if indexMapping[i] == nil {
					indexMapping[i] = []int{}
				}
				indexMapping[i] = append(indexMapping[i], startIdx+j)
			}

			// Collect next-year installments for rollover file
			allRollovers = append(allRollovers, rollovers...)
		} else {
			// Regular expense
			indexMapping[i] = []int{len(expenses)}
			expenses = append(expenses, parsedExpense)
		}
	}

	// Initialize errors array for EXPANDED expenses
	errors := make([]*models.BatchError, len(expenses))

	// Step 2: Load reference sheet mappings ONCE and build hierarchical index
	mappings, err := excel.LoadReferenceSheet(workbookPath)
	if err != nil {
		// If reference sheet fails, all expenses fail
		for i := range errors {
			if errors[i] == nil { // Don't overwrite parse errors
				errors[i] = models.NewIOError("load reference sheet", err)
			}
		}
		return aggregateErrors(originalErrors, errors, indexMapping), allRollovers
	}
	pathIndex := excel.BuildPathIndex(mappings)

	// Step 3: Resolve all subcategories using hierarchical paths (in-memory, no file I/O)
	// Track valid expenses for batch processing
	validIndices := []int{}
	resolvedMappings := make([]*resolver.SubcategoryMapping, len(expenses))

	for i, expense := range expenses {
		if errors[i] != nil || expense == nil {
			continue // Skip already failed expenses
		}

		mapping, isAmbiguous, err := resolver.ResolveSubcategoryWithPath(pathIndex, expense.Subcategory)
		if err != nil {
			errors[i] = models.NewResolutionError(expense.Subcategory)
			continue
		}

		if isAmbiguous {
			options := resolver.GetAmbiguousOptions(pathIndex.BySubcategory, expense.Subcategory)
			errors[i] = models.NewAmbiguousError(expense.Subcategory, len(options))
			continue
		}

		resolvedMappings[i] = mapping
		validIndices = append(validIndices, i)
	}

	// If no valid expenses, return early
	if len(validIndices) == 0 {
		return aggregateErrors(originalErrors, errors, indexMapping), allRollovers
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
		return aggregateErrors(originalErrors, errors, indexMapping), allRollovers
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
		return aggregateErrors(originalErrors, errors, indexMapping), allRollovers
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
		return aggregateErrors(originalErrors, errors, indexMapping), allRollovers
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
		return aggregateErrors(originalErrors, errors, indexMapping), allRollovers
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
			return aggregateErrors(originalErrors, errors, indexMapping), allRollovers
		}
	}

	return aggregateErrors(originalErrors, errors, indexMapping), allRollovers
}

// aggregateErrors combines errors from expanded installments back to original indices
func aggregateErrors(originalErrors []*models.BatchError, expandedErrors []*models.BatchError, indexMapping map[int][]int) []*models.BatchError {
	// Aggregate errors back to original indices
	for origIdx, expandedIndices := range indexMapping {
		// Skip if already has a parse error
		if originalErrors[origIdx] != nil {
			continue
		}

		// Collect errors for this original expense's installments
		var firstError *models.BatchError
		failedCount := 0
		totalCount := len(expandedIndices)

		for _, expIdx := range expandedIndices {
			if expIdx < len(expandedErrors) && expandedErrors[expIdx] != nil {
				failedCount++
				if firstError == nil {
					firstError = expandedErrors[expIdx]
				}
			}
		}

		// Report combined result
		if failedCount > 0 {
			if failedCount == totalCount {
				// All installments failed - report first error
				originalErrors[origIdx] = firstError
			} else {
				// Partial failure - create informative error
				originalErrors[origIdx] = &models.BatchError{
					Category:   models.ErrorCategoryCapacity,
					Message:    fmt.Sprintf("%d/%d installments failed", failedCount, totalCount),
					OutputFile: models.OutputFileFailed,
					GroupLabel: "Partial Installment Failure",
					Retriable:  true,
				}
			}
		}
	}

	return originalErrors
}

// addMonths adds n months to a date, keeping the same day
// If target month has fewer days, adjusts to last day of month
func addMonths(t time.Time, months int) time.Time {
	year := t.Year()
	month := t.Month()
	day := t.Day()

	// Add months
	month += time.Month(months)

	// Handle year overflow
	for month > 12 {
		month -= 12
		year++
	}

	// Create target date (may overflow if day doesn't exist in target month)
	target := time.Date(year, month, day, t.Hour(), t.Minute(), t.Second(), t.Nanosecond(), t.Location())

	// If date overflow (e.g., Jan 31 + 1 month = Mar 3), adjust to last day of target month
	if target.Month() != month {
		// Go back to last day of intended month
		target = time.Date(year, month+1, 0, t.Hour(), t.Minute(), t.Second(), t.Nanosecond(), t.Location())
	}

	return target
}

// RolloverExpense represents an expense that rolls over to next year
type RolloverExpense struct {
	Expense       *models.Expense
	OriginalIndex int
}

// expandInstallments converts a single installment expense into multiple individual expenses
// Returns: expanded expenses, rollover expenses (for next year)
func expandInstallments(expense *models.Expense, originalIndex int) ([]*models.Expense, []RolloverExpense) {
	if !expense.IsInstallment() {
		return []*models.Expense{expense}, nil
	}

	currentYear := expense.Date.Year()
	expanded := make([]*models.Expense, 0, expense.Installment.Count)
	rollovers := []RolloverExpense{}

	for i := 0; i < expense.Installment.Count; i++ {
		installmentDate := addMonths(expense.Date, i)

		newExpense := &models.Expense{
			Item:        expense.Item, // Will be formatted with (N/M) during write
			Date:        installmentDate,
			Value:       expense.Value, // Already divided
			Subcategory: expense.Subcategory,
			Installment: &models.Installment{
				Total:   expense.Installment.Total,
				Count:   expense.Installment.Count,
				Current: i + 1, // 1-based
			},
		}

		// Track if this installment rolls over to next year
		if installmentDate.Year() > currentYear {
			rollovers = append(rollovers, RolloverExpense{
				Expense:       newExpense,
				OriginalIndex: originalIndex,
			})
		} else {
			// Normal installment for this year
			expanded = append(expanded, newExpense)
		}
	}

	return expanded, rollovers
}
