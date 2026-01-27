package cmd

import (
	"expense-reporter/internal/batch"
	"expense-reporter/internal/models"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

var (
	backupFlag bool
	reportPath string
	silentFlag bool
)

var batchCmd = &cobra.Command{
	Use:   "batch <csv_file>",
	Short: "Import multiple expenses from a CSV file",
	Long: `Import multiple expenses from a CSV file.

CSV Format (semicolon-delimited, one expense per line):
  <item_description>;<DD/MM>;<value_in_##,##>;<subcategory>

Example:
  Uber Centro;15/04;35,50;Uber/Taxi
  Compras Carrefour;03/01;150,00;Supermercado

Comments (lines starting with #) and empty lines are ignored.

The command will:
  1. Optionally create a backup of the workbook (--backup)
  2. Process all expenses from the CSV file
  3. Skip ambiguous subcategories (saved to ambiguous_expenses.csv)
  4. Generate a detailed report (--report)
  5. Display a summary of the results`,
	Args: cobra.ExactArgs(1),
	RunE: runBatch,
}

func init() {
	rootCmd.AddCommand(batchCmd)
	batchCmd.Flags().BoolVar(&backupFlag, "backup", false, "Create backup before processing")
	batchCmd.Flags().StringVar(&reportPath, "report", "batch_report.txt", "Report output path (empty to skip)")
	batchCmd.Flags().BoolVar(&silentFlag, "silent", false, "Suppress progress bar output")
}

func runBatch(cmd *cobra.Command, args []string) error {
	csvPath := args[0]

	// Print header
	fmt.Println("Batch Import")
	fmt.Println("============")

	// Step 1: Validate CSV file exists
	if _, err := os.Stat(csvPath); err != nil {
		return fmt.Errorf("CSV file not found: %s", csvPath)
	}
	fmt.Printf("Source: %s\n", csvPath)

	// Step 2: Get workbook path
	workbookPath, err := GetWorkbookPath()
	if err != nil {
		return fmt.Errorf("failed to get workbook path: %w", err)
	}

	// Step 3: Validate workbook exists
	if _, err := os.Stat(workbookPath); err != nil {
		return fmt.Errorf("workbook not found: %s", workbookPath)
	}
	fmt.Printf("Workbook: %s\n\n", workbookPath)

	// Step 4: Create backup if requested
	var backupPath string
	if backupFlag {
		backupMgr := batch.NewBackupManager()
		backupPath, err = backupMgr.CreateBackup(workbookPath)
		if err != nil {
			return fmt.Errorf("backup failed: %w", err)
		}
		fmt.Printf("✓ Backup created: %s\n", filepath.Base(backupPath))
	}

	// Step 5: Create processor and load mappings
	processor := batch.NewProcessor(workbookPath)
	err = processor.LoadMappings()
	if err != nil {
		return fmt.Errorf("failed to load subcategory mappings: %w", err)
	}
	fmt.Println("✓ Loaded subcategory mappings")

	// Step 6: Read CSV
	reader := batch.NewCSVReader(csvPath)
	expenseStrings, err := reader.Read()
	if err != nil {
		return fmt.Errorf("failed to read CSV: %w", err)
	}

	if len(expenseStrings) == 0 {
		fmt.Println("\n⚠ No expenses found in CSV file")
		return nil
	}

	// Step 7: Process batch with progress (optimized - 20-28x faster)
	fmt.Printf("\nProcessing %d expenses...\n", len(expenseStrings))
	progress := batch.NewProgressReporter(len(expenseStrings), silentFlag)

	summary, rolloverExpenses, err := processor.ProcessBatch(
		expenseStrings,
		progress.Update,
	)
	progress.Finish()

	if err != nil {
		return fmt.Errorf("batch processing failed: %w", err)
	}

	// Step 8: Collect error entries by type
	failedEntries := []batch.FailedEntry{}
	ambiguousEntries := []batch.AmbiguousEntry{}

	for _, result := range summary.Results {
		if result.IsAmbiguous {
			// Collect ambiguous entries with hierarchical path options
			sheetNames := make([]string, len(result.AmbiguousOpts))
			pathOptions := make([]string, len(result.AmbiguousOpts))
			for i, opt := range result.AmbiguousOpts {
				sheetNames[i] = opt.SheetName
				pathOptions[i] = opt.SheetName + "," + opt.Category + "," + opt.Subcategory
			}
			ambiguousEntries = append(ambiguousEntries, batch.AmbiguousEntry{
				ExpenseString: result.ExpenseString,
				Subcategory:   result.Expense.Subcategory,
				SheetOptions:  sheetNames,
				PathOptions:   pathOptions,
			})
		} else if !result.Success {
			// Collect failed entries (new)
			errorCategory := models.ErrorCategoryIO // default
			if batchErr, ok := result.Error.(*models.BatchError); ok {
				errorCategory = batchErr.Category
			}

			failedEntries = append(failedEntries, batch.FailedEntry{
				ExpenseString: result.ExpenseString,
				ErrorMessage:  result.Error.Error(),
				ErrorCategory: errorCategory,
			})
		}
	}

	// Step 9: Write failed and ambiguous files
	failedWriter := batch.NewFailedWriter(".")
	failedPath, ambiguousPath, err := failedWriter.Write(failedEntries, ambiguousEntries)
	if err != nil {
		fmt.Printf("\n⚠ Warning: Failed to write output files: %v\n", err)
	}

	if failedPath != "" {
		fmt.Printf("\n✗ Failed expenses saved to: %s\n", failedPath)
		fmt.Println("  Fix the errors and re-import this file.")
	}

	if ambiguousPath != "" {
		fmt.Printf("\n⚠ Ambiguous expenses saved to: %s\n", ambiguousPath)
		fmt.Println("  Choose the correct sheet for each entry and re-import.")
	}

	// Step 9.5: Write rollover file if any
	if len(rolloverExpenses) > 0 {
		rolloverWriter := batch.NewRolloverWriter(".")
		rolloverPath, err := rolloverWriter.Write(rolloverExpenses)
		if err != nil {
			fmt.Printf("\n⚠ Warning: Failed to write rollover file: %v\n", err)
		} else if rolloverPath != "" {
			fmt.Printf("\n→ Rollover expenses (next year) saved to: %s\n", rolloverPath)
			fmt.Println("  Import these in January of next year.")
		}
	}

	// Step 10: Write report if path specified
	if reportPath != "" {
		reportWriter := batch.NewReportWriter(reportPath)
		err = reportWriter.Write(summary, csvPath)
		if err != nil {
			fmt.Printf("\n⚠ Warning: Failed to write report: %v\n", err)
		} else {
			fmt.Printf("\n✓ Report saved to: %s\n", reportPath)
		}
	}

	// Step 11: Display summary
	fmt.Println("\nResults")
	fmt.Println("-------")
	fmt.Printf("✓ Successfully inserted: %d\n", summary.SuccessCount)
	fmt.Printf("✗ Failed: %d\n", summary.ErrorCount)
	fmt.Printf("⚠ Ambiguous (review required): %d\n", summary.AmbiguousCount)

	// Show grouped errors if any
	if summary.ErrorCount > 0 {
		displayGroupedErrors(summary)
	}

	fmt.Println()
	return nil
}

// collectAmbiguousEntries extracts ambiguous entries from summary for writing to CSV
func collectAmbiguousEntries(summary *batch.BatchSummary) []batch.AmbiguousEntry {
	var entries []batch.AmbiguousEntry

	for _, result := range summary.Results {
		if result.IsAmbiguous {
			// Extract sheet names and hierarchical paths from mapping options
			sheetNames := make([]string, len(result.AmbiguousOpts))
			pathOptions := make([]string, len(result.AmbiguousOpts))
			for i, opt := range result.AmbiguousOpts {
				sheetNames[i] = opt.SheetName
				pathOptions[i] = opt.SheetName + "," + opt.Category + "," + opt.Subcategory
			}

			entries = append(entries, batch.AmbiguousEntry{
				ExpenseString: result.ExpenseString,
				Subcategory:   result.Expense.Subcategory,
				SheetOptions:  sheetNames,
				PathOptions:   pathOptions,
			})
		}
	}

	return entries
}

// displayGroupedErrors shows errors grouped by category with fix instructions
func displayGroupedErrors(summary *batch.BatchSummary) {
	// Group errors by category
	errorGroups := make(map[models.ErrorCategory][]batch.BatchResult)

	for _, result := range summary.Results {
		if !result.Success && !result.IsAmbiguous {
			category := models.ErrorCategoryIO // default
			if batchErr, ok := result.Error.(*models.BatchError); ok {
				category = batchErr.Category
			}
			errorGroups[category] = append(errorGroups[category], result)
		}
	}

	// Display each group
	fmt.Println("\nErrors by Category:")
	for category, results := range errorGroups {
		groupLabel := getGroupLabel(category)
		fmt.Printf("\n%s (%d):\n", groupLabel, len(results))

		// Show first 3 errors
		for i, result := range results {
			if i >= 3 {
				fmt.Printf("  ... and %d more\n", len(results)-3)
				break
			}
			fmt.Printf("  Line %d: %s\n", result.LineNumber, result.Error.Error())
		}

		// Show fix instructions
		fmt.Printf("  Fix: %s\n", getFixInstructions(category))
	}
}

// getGroupLabel returns human-readable label for error category
func getGroupLabel(category models.ErrorCategory) string {
	switch category {
	case models.ErrorCategoryParse:
		return "Parse Errors"
	case models.ErrorCategoryResolution:
		return "Subcategory Not Found"
	case models.ErrorCategoryCapacity:
		return "Capacity Full"
	case models.ErrorCategoryAmbiguous:
		return "Ambiguous Subcategories"
	case models.ErrorCategoryIO:
		return "File I/O Errors"
	default:
		return "Other Errors"
	}
}

// getFixInstructions returns fix instructions for each category
func getFixInstructions(category models.ErrorCategory) string {
	switch category {
	case models.ErrorCategoryParse:
		return "Check CSV format: <item>;<DD/MM>;<value>;<subcategory>"
	case models.ErrorCategoryResolution:
		return "Verify subcategory exists in reference sheet"
	case models.ErrorCategoryCapacity:
		return "Manually expand subcategory rows in Excel before re-importing"
	case models.ErrorCategoryAmbiguous:
		return "Specify sheet in ambiguous CSV file"
	case models.ErrorCategoryIO:
		return "Check file permissions and disk space"
	default:
		return "See failed CSV for details"
	}
}
