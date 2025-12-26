package batch

import (
	"expense-reporter/internal/models"
	"expense-reporter/internal/resolver"
)

// BatchResult represents the outcome of processing a single expense line
type BatchResult struct {
	LineNumber    int                // Line number in CSV (1-based)
	ExpenseString string             // Original expense string from CSV
	Success       bool               // True if successfully inserted
	IsAmbiguous   bool               // True if subcategory was ambiguous
	Error         error              // Error if failed (nil if success or ambiguous)
	Expense       *models.Expense    // Parsed expense (nil if parse failed)
	AmbiguousOpts []resolver.SubcategoryMapping // Sheet options if ambiguous
}

// BatchSummary aggregates results from batch processing
type BatchSummary struct {
	TotalLines      int           // Total lines read from CSV
	SuccessCount    int           // Expenses successfully inserted
	ErrorCount      int           // Expenses that failed with errors
	AmbiguousCount  int           // Expenses skipped due to ambiguity
	SkippedCount    int           // Lines skipped (comments, empty)
	Results         []BatchResult // Detailed results for each line
}

// AmbiguousEntry represents an expense with multiple sheet options
type AmbiguousEntry struct {
	ExpenseString string   // Original expense string
	Subcategory   string   // The ambiguous subcategory name
	SheetOptions  []string // Available sheet names
}
