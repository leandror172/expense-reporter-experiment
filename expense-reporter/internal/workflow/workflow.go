package workflow

import (
	"expense-reporter/internal/excel"
	"expense-reporter/internal/logger"
	"expense-reporter/internal/models"
	"expense-reporter/internal/parser"
	"expense-reporter/internal/resolver"
	"fmt"
	"time"
)

// InsertExpense inserts a single expense into the workbook.
// Delegates to InsertBatchExpenses so both paths share identical pipeline logic.
func InsertExpense(workbookPath, expenseString string) error {
	errs, _ := InsertBatchExpenses(workbookPath, []string{expenseString})
	if len(errs) > 0 && errs[0] != nil {
		return fmt.Errorf("%s", errs[0].Message)
	}
	return nil
}

// InsertBatchExpenses inserts multiple expense strings in a single batch operation
// Returns a slice of BatchErrors (one per expense) - nil for success, error for failure
// This is dramatically faster than calling InsertExpense repeatedly (20-28x speedup)
// Also returns rollover expenses that cross year boundary
func InsertBatchExpenses(workbookPath string, expenseStrings []string) ([]*models.BatchError, []RolloverExpense) {
	if len(expenseStrings) == 0 {
		return make([]*models.BatchError, 0), nil
	}

	// Parse all expenses first (fail fast before file operations)
	parsedExpenses, originalErrors := parseExpenseStrings(expenseStrings)

	return insertParsedExpenses(workbookPath, parsedExpenses, originalErrors)
}

// InsertBatchExpensesFromClassified inserts pre-classified expenses using the same batch
// pipeline as InsertBatchExpenses. RawValue is preserved as-is so installment notation
// (e.g. "99,90/3") is not lost during the string conversion.
func InsertBatchExpensesFromClassified(workbookPath string, rows []models.ClassifiedExpense) ([]*models.BatchError, []RolloverExpense) {

	// Parse all expenses first (fail fast before file operations)
	parsedExpenses, originalErrors := parseClassifiedExpenses(rows)
	return insertParsedExpenses(workbookPath, parsedExpenses, originalErrors)
}

func insertParsedExpenses(workbookPath string, parsedExpenses []*models.Expense, originalErrors []*models.BatchError) ([]*models.BatchError, []RolloverExpense) {
	// Step 1: Expand installments; track original→expanded index mapping
	expenses, indexMapping, allRollovers := expandAllInstallments(parsedExpenses, originalErrors)

	// Initialize errors array for EXPANDED expenses
	errors := make([]*models.BatchError, len(expenses))

	// Step 2: Load reference sheet mappings ONCE and build hierarchical index
	mappings, err := excel.LoadReferenceSheet(workbookPath)
	if err != nil {
		for i := range errors {
			if errors[i] == nil {
				errors[i] = models.NewIOError("load reference sheet", err)
			}
		}
		return aggregateErrors(originalErrors, errors, indexMapping), allRollovers
	}
	pathIndex := excel.BuildPathIndex(mappings)

	// Step 3: Resolve all subcategories (in-memory, no file I/O)
	resolvedMappings, validIndices := resolveAllSubcategories(expenses, errors, pathIndex)
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
		for _, i := range validIndices {
			errors[i] = models.NewIOError("find subcategory rows", err)
		}
		return aggregateErrors(originalErrors, errors, indexMapping), allRollovers
	}

	// Step 6: Build empty row requests (one per valid expense)
	emptyRowRequests := buildEmptyRowRequests(expenses, validIndices, resolvedMappings, subcatRows, errors)

	// Step 7: Allocate rows — one per request, respecting order within each subcategory
	allocatedRows, err := excel.AllocateEmptyRows(workbookPath, emptyRowRequests)
	if err != nil {
		for _, i := range validIndices {
			if errors[i] == nil {
				errors[i] = models.NewIOError("find empty rows", err)
			}
		}
		return aggregateErrors(originalErrors, errors, indexMapping), allRollovers
	}

	// Step 8: Build expense-location pairs for batch write
	expensesWithLocations, finalValidIndices := buildExpensesWithLocations(
		emptyRowRequests, allocatedRows, expenses, resolvedMappings, subcatRows, errors,
	)

	// Step 9: Write all valid expenses in ONE file open/save cycle
	if len(expensesWithLocations) > 0 {
		if err := excel.WriteBatchExpenses(workbookPath, expensesWithLocations); err != nil {
			for _, i := range finalValidIndices {
				errors[i] = models.NewIOError("write expenses", err)
			}
			return aggregateErrors(originalErrors, errors, indexMapping), allRollovers
		}
	}

	return aggregateErrors(originalErrors, errors, indexMapping), allRollovers
}

// parseExpenseStrings parses raw expense strings into Expense structs.
// Returns parallel slices: parsed[i] is nil when errors[i] is non-nil.
func parseExpenseStrings(expenseStrings []string) ([]*models.Expense, []*models.BatchError) {
	parsedExpenses := make([]*models.Expense, len(expenseStrings))
	errors := make([]*models.BatchError, len(expenseStrings))
	for i, expenseString := range expenseStrings {
		expense, err := parser.ParseExpenseString(expenseString)
		if err != nil {
			errors[i] = models.NewParseError(err.Error(), err)
			continue
		}
		parsedExpenses[i] = expense
	}
	return parsedExpenses, errors
}

// parseClassifiedExpenses parses classified expense structs into Expense structs.
// Returns parallel slices: parsed[i] is nil when errors[i] is non-nil.
func parseClassifiedExpenses(rows []models.ClassifiedExpense) ([]*models.Expense, []*models.BatchError) {
	parsedExpenses := make([]*models.Expense, len(rows))
	errors := make([]*models.BatchError, len(rows))
	for i, expense := range rows {
		expense, err := parser.ParseExpense(expense)
		if err != nil {
			errors[i] = models.NewParseError(err.Error(), err)
			continue
		}
		parsedExpenses[i] = expense
	}
	return parsedExpenses, errors
}

// expandAllInstallments expands installment expenses into per-month entries.
// Returns the expanded expense list, original→expanded index mapping, and rollover expenses.
func expandAllInstallments(parsedExpenses []*models.Expense, parseErrors []*models.BatchError) ([]*models.Expense, map[int][]int, []RolloverExpense) {
	expenses := []*models.Expense{}
	indexMapping := make(map[int][]int)
	allRollovers := []RolloverExpense{}

	for i, parsedExpense := range parsedExpenses {
		if parseErrors[i] != nil || parsedExpense == nil {
			indexMapping[i] = []int{len(expenses)}
			expenses = append(expenses, nil)
			continue
		}

		if parsedExpense.IsInstallment() {
			expandedList, rollovers := expandInstallments(parsedExpense, i)
			startIdx := len(expenses)
			for j := range expandedList {
				expenses = append(expenses, expandedList[j])
				if indexMapping[i] == nil {
					indexMapping[i] = []int{}
				}
				indexMapping[i] = append(indexMapping[i], startIdx+j)
			}
			allRollovers = append(allRollovers, rollovers...)
		} else {
			indexMapping[i] = []int{len(expenses)}
			expenses = append(expenses, parsedExpense)
		}
	}
	return expenses, indexMapping, allRollovers
}

// resolveAllSubcategories resolves each expanded expense to its sheet mapping.
// Sets errors[i] for resolution failures; returns resolved mappings and valid indices.
func resolveAllSubcategories(expenses []*models.Expense, errors []*models.BatchError, pathIndex *resolver.PathIndex) ([]*resolver.SubcategoryMapping, []int) {
	resolvedMappings := make([]*resolver.SubcategoryMapping, len(expenses))
	validIndices := []int{}

	for i, expense := range expenses {
		if errors[i] != nil || expense == nil {
			continue
		}

		mapping, isAmbiguous, err := resolver.ResolveSubcategoryWithPath(pathIndex, expense.Subcategory)
		if err != nil {
			errors[i] = models.NewResolutionError(expense.Subcategory)
			continue
		}

		if isAmbiguous {
			options := resolver.GetAmbiguousOptions(pathIndex.BySubcategory, expense.Subcategory)
			sheetNames := make([]string, len(options))
			for j, opt := range options {
				sheetNames[j] = opt.SheetName
			}
			errors[i] = models.NewAmbiguousError(expense.Subcategory, sheetNames)
			continue
		}

		resolvedMappings[i] = mapping
		validIndices = append(validIndices, i)
	}
	return resolvedMappings, validIndices
}

// buildEmptyRowRequests constructs the row allocation requests for all valid expenses.
// Sets errors[i] for invalid month; each request is tagged with ExpenseIndex for reverse lookup.
func buildEmptyRowRequests(
	expenses []*models.Expense,
	validIndices []int,
	resolvedMappings []*resolver.SubcategoryMapping,
	subcatRows map[string]map[string]int,
	errors []*models.BatchError,
) []excel.EmptyRowRequest {
	emptyRowRequests := []excel.EmptyRowRequest{}
	for _, i := range validIndices {
		expense := expenses[i]
		mapping := resolvedMappings[i]
		subcatRow := subcatRows[mapping.SheetName][mapping.Subcategory]

		itemCol, _, _, err := excel.GetMonthColumns(expense.Date.Month())
		if err != nil {
			errors[i] = models.NewParseError(fmt.Sprintf("invalid month: %v", err), err)
			continue
		}

		logger.Debug("batch: empty row request", "index", i, "subcategory", mapping.Subcategory,
			"sheet", mapping.SheetName, "subcatRow", subcatRow, "itemCol", itemCol)

		emptyRowRequests = append(emptyRowRequests, excel.EmptyRowRequest{
			SheetName:       mapping.SheetName,
			ColumnLetter:    itemCol,
			StartRow:        subcatRow,
			SubcategoryName: mapping.Subcategory,
			ExpenseIndex:    i,
		})
	}
	return emptyRowRequests
}

// buildExpensesWithLocations pairs each allocated row with its expense and sheet location.
// Sets errors[i] for capacity failures (no row allocated); returns pairs and their expense indices.
func buildExpensesWithLocations(
	emptyRowRequests []excel.EmptyRowRequest,
	allocatedRows map[int]int,
	expenses []*models.Expense,
	resolvedMappings []*resolver.SubcategoryMapping,
	subcatRows map[string]map[string]int,
	errors []*models.BatchError,
) ([]excel.ExpenseWithLocation, []int) {
	expensesWithLocations := []excel.ExpenseWithLocation{}
	finalValidIndices := []int{}

	for idx, req := range emptyRowRequests {
		i := req.ExpenseIndex
		if errors[i] != nil {
			continue
		}

		targetRow, ok := allocatedRows[idx]
		if !ok {
			errors[i] = models.NewCapacityError(
				resolvedMappings[i].Subcategory,
				resolvedMappings[i].SheetName,
				0,
			)
			continue
		}

		expense := expenses[i]
		mapping := resolvedMappings[i]
		itemCol, _, _, _ := excel.GetMonthColumns(expense.Date.Month())

		logger.Debug("batch: allocated row", "index", i, "subcategory", mapping.Subcategory,
			"sheet", mapping.SheetName, "targetRow", targetRow)

		location := &models.SheetLocation{
			SheetName:   mapping.SheetName,
			Category:    mapping.Category,
			SubcatRow:   subcatRows[mapping.SheetName][mapping.Subcategory],
			TargetRow:   targetRow,
			MonthColumn: itemCol,
		}

		expensesWithLocations = append(expensesWithLocations, excel.ExpenseWithLocation{
			Expense:  expense,
			Location: location,
		})
		finalValidIndices = append(finalValidIndices, i)
	}
	return expensesWithLocations, finalValidIndices
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
