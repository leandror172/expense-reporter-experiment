package batch

import (
	"expense-reporter/internal/excel"
	"expense-reporter/internal/models"
	"expense-reporter/internal/parser"
	"expense-reporter/internal/resolver"
	"fmt"
)

// InsertFunc is a function type for inserting expenses into the workbook
// This allows dependency injection for testing
type InsertFunc func(workbookPath string, expense *models.Expense) error

// Processor handles batch processing of expense strings
type Processor struct {
	workbookPath string
	mappings     map[string][]resolver.SubcategoryMapping
}

// NewProcessor creates a new batch processor
func NewProcessor(workbookPath string) *Processor {
	return &Processor{
		workbookPath: workbookPath,
	}
}

// LoadMappings loads subcategory mappings from the Excel reference sheet
// This should be called once before processing to cache mappings
func (p *Processor) LoadMappings() error {
	mappings, err := excel.LoadReferenceSheet(p.workbookPath)
	if err != nil {
		return fmt.Errorf("failed to load reference sheet: %w", err)
	}
	p.mappings = mappings
	return nil
}

// SetMappings sets mappings directly (for testing)
func (p *Processor) SetMappings(mappings map[string][]resolver.SubcategoryMapping) {
	p.mappings = mappings
}

// Process processes a batch of expense strings
// - Parses each expense string
// - Resolves subcategory (detecting ambiguous cases)
// - Inserts non-ambiguous expenses via insertFunc
// - Collects results and errors
// - Calls progressCallback after each expense (if provided)
// Returns complete BatchSummary with all results
func (p *Processor) Process(
	expenseStrings []string,
	insertFunc InsertFunc,
	progressCallback func(current, total int),
) (*BatchSummary, error) {
	total := len(expenseStrings)
	summary := &BatchSummary{
		TotalLines: total,
		Results:    make([]BatchResult, 0, total),
	}

	for i, expenseString := range expenseStrings {
		lineNumber := i + 1
		result := p.processOne(expenseString, lineNumber, insertFunc)
		summary.Results = append(summary.Results, result)

		// Update counters
		if result.Success {
			summary.SuccessCount++
		} else if result.IsAmbiguous {
			summary.AmbiguousCount++
		} else {
			summary.ErrorCount++
		}

		// Call progress callback if provided
		if progressCallback != nil {
			progressCallback(lineNumber, total)
		}
	}

	return summary, nil
}

// processOne processes a single expense string
func (p *Processor) processOne(expenseString string, lineNumber int, insertFunc InsertFunc) BatchResult {
	result := BatchResult{
		LineNumber:    lineNumber,
		ExpenseString: expenseString,
	}

	// Step 1: Parse expense string
	expense, err := parser.ParseExpenseString(expenseString)
	if err != nil {
		result.Error = err
		return result
	}
	result.Expense = expense

	// Step 2: Resolve subcategory
	mapping, isAmbiguous, err := resolver.ResolveSubcategory(p.mappings, expense.Subcategory)
	if err != nil {
		result.Error = err
		return result
	}

	// Step 3: Handle ambiguous subcategory
	if isAmbiguous {
		result.IsAmbiguous = true
		// Get all options for this subcategory
		searchKey := expense.Subcategory
		parent := resolver.ExtractParentSubcategory(expense.Subcategory)
		if parent != expense.Subcategory {
			searchKey = parent
		}
		result.AmbiguousOpts = resolver.GetAmbiguousOptions(p.mappings, searchKey)
		// Don't insert ambiguous expenses - user must resolve manually
		return result
	}

	// Step 4: Single mapping found - insert the expense
	// Note: mapping is already resolved to the correct sheet by resolver
	_ = mapping // Will be used in future for validation
	err = insertFunc(p.workbookPath, expense)
	if err != nil {
		result.Error = err
		return result
	}

	// Success!
	result.Success = true
	return result
}
