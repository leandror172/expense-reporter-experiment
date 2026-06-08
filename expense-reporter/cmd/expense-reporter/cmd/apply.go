package cmd

import (
	"fmt"
	"io"
	"time"

	"github.com/spf13/cobra"

	"expense-reporter/internal/apply"
	"expense-reporter/internal/classifier"
	internalconfig "expense-reporter/internal/config"
	"expense-reporter/internal/excel"
	"expense-reporter/internal/feedback"
	"expense-reporter/internal/models"
	"expense-reporter/pkg/utils"
)

var (
	applyWorkbook string
	applyYear     int
	applyDryRun   bool
)

var applyCmd = &cobra.Command{
	Use:   "apply <reviewed.json>",
	Short: "Apply reviewed entries to workbook and feedback logs",
	Long: `Ingests the UI's reviewed.json, inserts new rows into the Excel workbook,
and writes feedback entries to classifications.jsonl.

Pending and skipped entries are ignored. Confirmed and corrected entries already
present in classifications.jsonl receive feedback-only updates (no workbook write).`,
	Args: cobra.ExactArgs(1),
	RunE: runApply,
}

func init() {
	rootCmd.AddCommand(applyCmd)
	applyCmd.Flags().StringVar(&applyWorkbook, "workbook", "", "Workbook path (overrides config)")
	applyCmd.Flags().IntVar(&applyYear, "year", time.Now().Year(), "Year for parsing DD/MM dates")
	applyCmd.Flags().BoolVar(&applyDryRun, "dry-run", false, "Print what would be inserted without writing")
}

func runApply(cmd *cobra.Command, args []string) error {
	cfg, err := internalconfig.Load()
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	classifPath := cfg.ClassificationsFilePath()
	if classifPath == "" {
		return fmt.Errorf("classifications log path is not configured")
	}
	expensesLogPath := cfg.ExpensesLogFilePath()

	rf, err := apply.ReadReviewed(args[0])
	if err != nil {
		return fmt.Errorf("reading reviewed file: %w", err)
	}

	workbookPath := applyWorkbook
	if workbookPath == "" {
		workbookPath = cfg.WorkbookFilePath()
	}

	newRows, corrections, pending, skipped, err := processEntries(rf.Entries, classifPath)
	if err != nil {
		return fmt.Errorf("processing entries: %w", err)
	}

	var insertedConfirmed, insertedCorrected int
	if len(newRows) > 0 {
		if workbookPath == "" {
			return fmt.Errorf("workbook path not configured (set EXPENSE_WORKBOOK or use --workbook)")
		}
		if err := excel.ValidateWorkbook(workbookPath); err != nil {
			return fmt.Errorf("validating workbook: %w", err)
		}
		insertedConfirmed, insertedCorrected, err = insertNewRows(newRows, workbookPath, classifPath, expensesLogPath, applyYear, applyDryRun)
		if err != nil {
			return fmt.Errorf("inserting new rows: %w", err)
		}
	}

	printSummary(cmd.OutOrStdout(), rf.Source, len(rf.Entries), pending, skipped, insertedConfirmed, insertedCorrected, corrections)
	return nil
}

func processEntries(entries []apply.ReviewedEntry, classifPath string) (newRows, corrections []apply.ReviewedEntry, pending, skipped int, err error) {
	for _, entry := range entries {
		switch entry.Action {
		case apply.ActionPending:
			pending++
		case apply.ActionSkipped:
			skipped++
		case apply.ActionConfirmed, apply.ActionCorrected:
			if hErr := handleActiveEntry(entry, classifPath, &newRows, &corrections); hErr != nil {
				return nil, nil, 0, 0, hErr
			}
		}
	}
	return newRows, corrections, pending, skipped, nil
}

func handleActiveEntry(entry apply.ReviewedEntry, classifPath string, newRows, corrections *[]apply.ReviewedEntry) error {
	prior, found, err := feedback.FindLatestEntry(classifPath, entry.ID)
	if err != nil {
		return fmt.Errorf("finding prior entry for %q: %w", entry.ID, err)
	}

	if !found {
		*newRows = append(*newRows, entry)
		return nil
	}

	// Already in workbook — write corrected feedback only if action is corrected.
	if entry.Action == apply.ActionCorrected && entry.Reviewed != nil {
		predicted := classifier.Result{
			Subcategory: prior.PredictedSubcategory,
			Category:    prior.PredictedCategory,
			Confidence:  prior.Confidence,
		}
		corrEntry := feedback.NewCorrectedEntry(
			entry.Item, entry.Date, entry.Value,
			predicted, prior.Model,
			entry.Reviewed.Subcategory, entry.Reviewed.Category,
		)
		if err := feedback.Append(classifPath, corrEntry); err != nil {
			return fmt.Errorf("appending corrected entry: %w", err)
		}
		*corrections = append(*corrections, entry)
	}
	// confirmed+found: no-op (already logged and in workbook)
	return nil
}

func insertNewRows(newRows []apply.ReviewedEntry, workbookPath, classifPath, expensesLogPath string, year int, dryRun bool) (insertedConfirmed, insertedCorrected int, err error) {
	subcatRows, err := excel.FindSubcategoryRowBatch(workbookPath, buildSubcatRequests(newRows))
	if err != nil {
		return 0, 0, fmt.Errorf("finding subcategory rows: %w", err)
	}

	emptyReqs, parsedDates, err := buildEmptyRowRequests(newRows, subcatRows, year)
	if err != nil {
		return 0, 0, err
	}

	targetRows, err := excel.AllocateEmptyRows(workbookPath, emptyReqs)
	if err != nil {
		return 0, 0, fmt.Errorf("allocating empty rows: %w", err)
	}

	batch, writtenIndices := buildExpenseBatch(newRows, parsedDates, subcatRows, emptyReqs, targetRows)
	if dryRun {
		confirmed, corrected := 0, 0
		for _, i := range writtenIndices {
			if newRows[i].Action == apply.ActionConfirmed {
				confirmed++
			} else {
				corrected++
			}
		}
		return confirmed, corrected, nil
	}
	if len(batch) > 0 {
		if err := excel.WriteBatchExpenses(workbookPath, batch); err != nil {
			return 0, 0, fmt.Errorf("writing batch expenses: %w", err)
		}
		return writeFeedbackForNewRows(newRows, writtenIndices, classifPath, expensesLogPath)
	}
	return 0, 0, nil
}

func buildSubcatRequests(newRows []apply.ReviewedEntry) []excel.SubcategoryLookupRequest {
	reqs := make([]excel.SubcategoryLookupRequest, len(newRows))
	for i, entry := range newRows {
		reqs[i] = excel.SubcategoryLookupRequest{
			SheetName:   entry.Reviewed.Sheet,
			Subcategory: entry.Reviewed.Subcategory,
		}
	}
	return reqs
}

func buildEmptyRowRequests(newRows []apply.ReviewedEntry, subcatRows map[string]map[string]int, year int) ([]excel.EmptyRowRequest, []time.Time, error) {
	dates := make([]time.Time, len(newRows))
	var reqs []excel.EmptyRowRequest
	for i, entry := range newRows {
		t, err := utils.ParseDateWithYear(entry.Date, year)
		if err != nil {
			return nil, nil, fmt.Errorf("parsing date for %q: %w", entry.Item, err)
		}
		dates[i] = t

		subcatRow, ok := subcatRows[entry.Reviewed.Sheet][entry.Reviewed.Subcategory]
		if !ok {
			continue
		}
		itemCol, _, _, err := excel.GetMonthColumns(t.Month())
		if err != nil {
			return nil, nil, fmt.Errorf("getting month columns for %q: %w", entry.Item, err)
		}
		reqs = append(reqs, excel.EmptyRowRequest{
			SheetName:       entry.Reviewed.Sheet,
			ColumnLetter:    itemCol,
			StartRow:        subcatRow + 1,
			SubcategoryName: entry.Reviewed.Subcategory,
			ExpenseIndex:    i,
		})
	}
	return reqs, dates, nil
}

// buildExpenseBatch iterates by emptyReqs position (matching AllocateEmptyRows key space)
// and uses req.ExpenseIndex to retrieve the original newRows entry and its parsed date.
// This is necessary because buildEmptyRowRequests may skip rows (subcategory not in
// workbook), making emptyReqs shorter than newRows and shifting AllocateEmptyRows keys.
func buildExpenseBatch(newRows []apply.ReviewedEntry, dates []time.Time, subcatRows map[string]map[string]int, emptyReqs []excel.EmptyRowRequest, targetRows map[int]int) ([]excel.ExpenseWithLocation, []int) {
	var batch []excel.ExpenseWithLocation
	var indices []int
	for pos, req := range emptyReqs {
		targetRow, ok := targetRows[pos] // pos = position in emptyReqs (AllocateEmptyRows key)
		if !ok {
			continue
		}
		i := req.ExpenseIndex // original index into newRows
		entry := newRows[i]
		subcatRow := subcatRows[entry.Reviewed.Sheet][entry.Reviewed.Subcategory]
		itemCol, _, _, _ := excel.GetMonthColumns(dates[i].Month())
		exp := &models.Expense{
			Item:        entry.Item,
			Date:        dates[i],
			Value:       entry.Value,
			Subcategory: entry.Reviewed.Subcategory,
		}
		loc := &models.SheetLocation{
			SheetName:   entry.Reviewed.Sheet,
			Category:    entry.Reviewed.Category,
			SubcatRow:   subcatRow,
			TargetRow:   targetRow,
			MonthColumn: itemCol,
		}
		batch = append(batch, excel.ExpenseWithLocation{Expense: exp, Location: loc})
		indices = append(indices, i)
	}
	return batch, indices
}

func writeFeedbackForNewRows(newRows []apply.ReviewedEntry, indices []int, classifPath, expensesLogPath string) (insertedConfirmed, insertedCorrected int, err error) {
	for _, i := range indices {
		entry := newRows[i]
		fbEntry, isConfirmed := buildFeedbackEntry(entry)
		if isConfirmed {
			insertedConfirmed++
		} else {
			insertedCorrected++
		}
		if err := feedback.Append(classifPath, fbEntry); err != nil {
			return insertedConfirmed, insertedCorrected, fmt.Errorf("appending feedback: %w", err)
		}
		if expensesLogPath != "" {
			expEntry := feedback.NewExpenseEntry(entry.Item, entry.Date, entry.Value, entry.Reviewed.Subcategory, entry.Reviewed.Category)
			if err := feedback.AppendExpense(expensesLogPath, expEntry); err != nil {
				return insertedConfirmed, insertedCorrected, fmt.Errorf("appending expense log: %w", err)
			}
		}
	}
	return insertedConfirmed, insertedCorrected, nil
}

func buildFeedbackEntry(entry apply.ReviewedEntry) (feedback.Entry, bool) {
	if entry.Action == apply.ActionConfirmed {
		predicted := classifier.Result{
			Subcategory: entry.Reviewed.Subcategory,
			Category:    entry.Reviewed.Category,
			Confidence:  entry.Confidence,
		}
		return feedback.NewConfirmedEntry(entry.Item, entry.Date, entry.Value, predicted, "review"), true
	}
	predicted := classifier.Result{
		Subcategory: entry.Predicted.Subcategory,
		Category:    entry.Predicted.Category,
		Confidence:  entry.Confidence,
	}
	return feedback.NewCorrectedEntry(entry.Item, entry.Date, entry.Value, predicted, "review",
		entry.Reviewed.Subcategory, entry.Reviewed.Category), false
}

func printSummary(w io.Writer, source string, total, pending, skipped, insertedConfirmed, insertedCorrected int, corrections []apply.ReviewedEntry) {
	inserted := insertedConfirmed + insertedCorrected
	fmt.Fprintf(w, "Applied %s (%d entries)\n\n", source, total)
	fmt.Fprintf(w, "Inserted:  %d rows (%d confirmed, %d corrected)\n", inserted, insertedConfirmed, insertedCorrected)
	fmt.Fprintf(w, "Skipped:   %d rows\n", skipped)
	fmt.Fprintf(w, "Pending:   %d rows\n", pending)

	if len(corrections) == 0 {
		return
	}
	fmt.Fprintf(w, "\n⚠  %d already-inserted rows were corrected — workbook not updated:\n", len(corrections))
	for _, c := range corrections {
		fmt.Fprintf(w, "   %s (%s, R$%.2f) %s → %s  [logged]\n",
			c.Item, c.Date, c.Value, c.Predicted.Subcategory, c.Reviewed.Subcategory)
	}
}
