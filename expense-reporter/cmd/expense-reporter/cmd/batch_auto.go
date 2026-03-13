package cmd

import (
	"encoding/csv"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"expense-reporter/internal/batch"
	"expense-reporter/internal/classifier"
	"expense-reporter/internal/config"
	"expense-reporter/internal/feedback"
	"expense-reporter/internal/models"
	"expense-reporter/internal/workflow"
	"expense-reporter/pkg/utils"

	"github.com/spf13/cobra"
)

var (
	batchAutoModel     string
	batchAutoDataDir   string
	batchAutoOllamaURL string
	batchAutoThreshold float64
	batchAutoTopN      int
	batchAutoDryRun    bool
	batchAutoOutputDir string
)

var batchAutoCmd = &cobra.Command{
	Use:   "batch-auto <csv_file>",
	Short: "Classify a CSV batch and auto-insert high-confidence expenses",
	Long: `Read a 3-field semicolon-delimited CSV (item;DD/MM;value), classify each row,
and auto-insert rows that exceed the confidence threshold into the workbook.

Output files are written to --output-dir (default: same directory as input):
  classified.csv  — all rows with classification results
  review.csv      — rows not auto-inserted (low confidence or excluded)
  rollover.csv    — installment rows whose later months fall into next year (if any)

Use --dry-run to skip workbook insertion and only produce the CSV outputs.

Examples:
  expense-reporter batch-auto expenses.csv
  expense-reporter batch-auto expenses.csv --dry-run --output-dir /tmp/out`,
	Args: cobra.ExactArgs(1),
	RunE: runBatchAuto,
}

func init() {
	rootCmd.AddCommand(batchAutoCmd)
	batchAutoCmd.Flags().StringVar(&batchAutoModel, "model", "my-classifier-q3", "Ollama model to use")
	batchAutoCmd.Flags().StringVar(&batchAutoDataDir, "data-dir", "data/classification", "Path to classification data directory")
	batchAutoCmd.Flags().StringVar(&batchAutoOllamaURL, "ollama-url", "http://localhost:11434", "Ollama API base URL")
	batchAutoCmd.Flags().Float64Var(&batchAutoThreshold, "threshold", 0.85, "Minimum confidence for auto-insert")
	batchAutoCmd.Flags().IntVar(&batchAutoTopN, "top", 3, "Number of classification candidates")
	batchAutoCmd.Flags().BoolVar(&batchAutoDryRun, "dry-run", false, "Classify and write CSVs without inserting into workbook")
	batchAutoCmd.Flags().StringVar(&batchAutoOutputDir, "output-dir", "", "Directory for output CSV files (default: same as input file)")
}

// classifiedRow holds the result of classifying a single input row.
type classifiedRow struct {
	Item         string
	Date         string
	RawValue     string // original value string, preserves installment notation (e.g. "99,90/3")
	Subcategory  string
	Category     string
	Confidence   float64
	AutoInserted bool
	Error        error
}

func runBatchAuto(cmd *cobra.Command, args []string) error {
	csvPath := args[0]

	outputDir, err := resolveOutputDir(csvPath, batchAutoOutputDir)
	if err != nil {
		return err
	}

	lines, err := loadInputLines(csvPath)
	if err != nil {
		return err
	}

	taxonomy, appCfg, err := loadBatchAutoDeps(batchAutoDataDir)
	if err != nil {
		return err
	}

	cfg := classifier.Config{
		OllamaURL: batchAutoOllamaURL,
		Model:     batchAutoModel,
		TopN:      batchAutoTopN,
	}

	results := classifyLines(lines, taxonomy, appCfg, cfg, batchAutoThreshold)

	var rollovers []workflow.RolloverExpense
	if !batchAutoDryRun {
		rollovers, err = insertClassified(results, appCfg, batchAutoModel)
		if err != nil {
			return err
		}
	}

	classifiedPath := filepath.Join(outputDir, "classified.csv")
	reviewPath := filepath.Join(outputDir, "review.csv")

	if err := writeClassifiedCSV(classifiedPath, results); err != nil {
		return fmt.Errorf("writing classified.csv: %w", err)
	}
	if err := writeReviewCSV(reviewPath, results); err != nil {
		return fmt.Errorf("writing review.csv: %w", err)
	}

	rolloverPath := ""
	if len(rollovers) > 0 {
		rolloverPath = filepath.Join(outputDir, "rollover.csv")
		if err := writeRolloverCSV(rolloverPath, rollovers); err != nil {
			return fmt.Errorf("writing rollover.csv: %w", err)
		}
	}

	printBatchSummary(results, rollovers, batchAutoDryRun, classifiedPath, reviewPath, rolloverPath)
	return nil
}

func resolveOutputDir(csvPath, outputDir string) (string, error) {
	if outputDir == "" {
		outputDir = filepath.Dir(csvPath)
	}
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return "", fmt.Errorf("creating output dir: %w", err)
	}
	return outputDir, nil
}

func loadInputLines(csvPath string) ([]string, error) {
	lines, err := batch.NewCSVReader(csvPath).Read()
	if err != nil {
		return nil, fmt.Errorf("reading CSV: %w", err)
	}
	if len(lines) == 0 {
		return nil, fmt.Errorf("input CSV is empty")
	}
	return lines, nil
}

func loadBatchAutoDeps(dataDir string) (classifier.Taxonomy, *config.Config, error) {
	taxonomy, err := classifier.LoadTaxonomy(dataDir)
	if err != nil {
		return nil, nil, fmt.Errorf("loading taxonomy: %w", err)
	}
	appCfg, err := config.Load()
	if err != nil {
		return nil, nil, fmt.Errorf("loading config: %w", err)
	}
	return taxonomy, appCfg, nil
}

func classifyLines(lines []string, taxonomy classifier.Taxonomy, appCfg *config.Config, cfg classifier.Config, threshold float64) []classifiedRow {
	total := len(lines)
	results := make([]classifiedRow, 0, total)

	for i, line := range lines {
		row, err := parse3FieldLine(line)
		if err != nil {
			fmt.Fprintf(os.Stderr, "[%d/%d] SKIP  %q: %v\n", i+1, total, line, err)
			results = append(results, classifiedRow{Item: line, Error: err})
			continue
		}

		classResults, err := classifier.Classify(row.Item, row.Value, row.Date, taxonomy, cfg)
		if err != nil || len(classResults) == 0 {
			fmt.Fprintf(os.Stderr, "[%d/%d] REVIEW %q: classifier error: %v\n", i+1, total, row.Item, err)
			results = append(results, classifiedRow{Item: row.Item, Date: row.Date, RawValue: row.RawValue, Error: err})
			continue
		}

		top := classResults[0]
		autoInsert := classifier.IsAutoInsertable(top, threshold, appCfg.AutoInsertExcluded)
		status := "REVIEW"
		if autoInsert {
			status = "AUTO  "
		}
		fmt.Printf("[%d/%d] %s %s → %s (%.0f%%)\n", i+1, total, status, row.Item, top.Subcategory, top.Confidence*100)

		results = append(results, classifiedRow{
			Item:         row.Item,
			Date:         row.Date,
			RawValue:     row.RawValue,
			Subcategory:  top.Subcategory,
			Category:     top.Category,
			Confidence:   top.Confidence,
			AutoInserted: autoInsert,
		})
	}
	return results
}

func insertClassified(results []classifiedRow, appCfg *config.Config, model string) ([]workflow.RolloverExpense, error) {
	workbook, err := GetWorkbookPath()
	if err != nil {
		return nil, fmt.Errorf("getting workbook path: %w", err)
	}
	if _, err := os.Stat(workbook); os.IsNotExist(err) {
		return nil, fmt.Errorf("workbook not found: %s", workbook)
	}

	if _, err := batch.NewBackupManager().CreateBackup(workbook); err != nil {
		return nil, fmt.Errorf("creating backup: %w", err)
	}

	// Build ClassifiedExpense slice for auto-insertable rows; track source indices
	var srcIdx []int
	var toInsert []models.ClassifiedExpense
	for i, r := range results {
		if r.AutoInserted && r.Error == nil {
			srcIdx = append(srcIdx, i)
			toInsert = append(toInsert, models.ClassifiedExpense{
				Item:        r.Item,
				Date:        r.Date,
				RawValue:    r.RawValue,
				Subcategory: r.Subcategory,
				Category:    r.Category,
				Confidence:  r.Confidence,
			})
		}
	}
	if len(toInsert) == 0 {
		return nil, nil
	}

	batchErrors, rollovers := workflow.InsertBatchExpensesFromClassified(workbook, toInsert)

	for i, bErr := range batchErrors {
		if bErr != nil {
			results[srcIdx[i]].AutoInserted = false
			results[srcIdx[i]].Error = fmt.Errorf("%s", bErr.Message)
			fmt.Fprintf(os.Stderr, "  INSERT ERROR %q: %s\n", results[srcIdx[i]].Item, bErr.Message)
		}
	}
	logBatchFeedback(appCfg, results, srcIdx, batchErrors, model)
	return rollovers, nil
}

// logBatchFeedback appends confirmed feedback entries for all successfully inserted rows.
// Non-fatal: logs a warning to stderr if any individual write fails.
func logBatchFeedback(appCfg *config.Config, results []classifiedRow, srcIdx []int, batchErrors []*models.BatchError, model string) {
	path := appCfg.ClassificationsFilePath()
	if path == "" {
		return
	}
	// Iterate source indices — log rows where AutoInserted is still true (no insert error)
	for i, idx := range srcIdx {
		if batchErrors[i] != nil {
			continue
		}
		r := results[idx]
		if !r.AutoInserted {
			continue
		}
		// Parse per-installment value for ID generation
		perInstallment, _, err := utils.ParseCurrencyWithInstallments(r.RawValue)
		if err != nil {
			continue
		}
		predicted := classifier.Result{
			Subcategory: r.Subcategory,
			Category:    r.Category,
			Confidence:  r.Confidence,
		}
		entry := feedback.NewConfirmedEntry(r.Item, r.Date, perInstallment, predicted, model)
		if err := feedback.Append(path, entry); err != nil {
			fmt.Fprintf(os.Stderr, "⚠  feedback log %q: %v\n", r.Item, err)
		}
	}
}

func printBatchSummary(results []classifiedRow, rollovers []workflow.RolloverExpense, dryRun bool, classifiedPath, reviewPath, rolloverPath string) {
	autoCount, reviewCount, errorCount := 0, 0, 0
	for _, r := range results {
		switch {
		case r.Error != nil:
			errorCount++
		case r.AutoInserted:
			autoCount++
		default:
			reviewCount++
		}
	}
	dryTag := ""
	if dryRun {
		dryTag = " (dry-run)"
	}
	fmt.Printf("\n--- Summary%s ---\n", dryTag)
	fmt.Printf("  Auto-inserted : %d\n", autoCount)
	fmt.Printf("  For review    : %d\n", reviewCount)
	fmt.Printf("  Errors        : %d\n", errorCount)
	if len(rollovers) > 0 {
		fmt.Printf("  Rollovers     : %d (next-year installments → %s)\n", len(rollovers), rolloverPath)
	}
	fmt.Printf("  classified.csv: %s\n", classifiedPath)
	fmt.Printf("  review.csv    : %s\n", reviewPath)
}

// inputRow is a parsed 3-field line.
type inputRow struct {
	Item     string
	Date     string
	Value    float64 // per-installment value, used for classifier display
	RawValue string  // original string, preserves installment notation (e.g. "99,90/3")
}

// parse3FieldLine splits "item;DD/MM;value" and parses currency.
// value may include installment notation (e.g. "99,90/3"); RawValue preserves it.
func parse3FieldLine(line string) (inputRow, error) {
	parts := strings.SplitN(line, ";", 3)
	if len(parts) != 3 {
		return inputRow{}, fmt.Errorf("expected 3 fields (item;DD/MM;value), got %d", len(parts))
	}
	item := strings.TrimSpace(parts[0])
	date := strings.TrimSpace(parts[1])
	valueStr := strings.TrimSpace(parts[2])
	if item == "" {
		return inputRow{}, fmt.Errorf("empty item field")
	}
	perInstallment, _, err := utils.ParseCurrencyWithInstallments(valueStr)
	if err != nil {
		return inputRow{}, fmt.Errorf("parsing value %q: %w", valueStr, err)
	}
	return inputRow{Item: item, Date: date, Value: perInstallment, RawValue: valueStr}, nil
}

// writeClassifiedCSV writes all classified rows to path.
// Format: item;date;value;subcategory;category;confidence;auto_inserted
func writeClassifiedCSV(path string, rows []classifiedRow) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	w := csv.NewWriter(f)
	w.Comma = ';'
	if err := w.Write([]string{"item", "date", "value", "subcategory", "category", "confidence", "auto_inserted"}); err != nil {
		return err
	}
	for _, r := range rows {
		w.Write([]string{ //nolint:errcheck
			r.Item,
			r.Date,
			r.RawValue,
			r.Subcategory,
			r.Category,
			fmt.Sprintf("%.4f", r.Confidence),
			fmt.Sprintf("%v", r.AutoInserted),
		})
	}
	w.Flush()
	return w.Error()
}

// writeReviewCSV writes only rows where auto_inserted == false.
func writeReviewCSV(path string, rows []classifiedRow) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	w := csv.NewWriter(f)
	w.Comma = ';'
	if err := w.Write([]string{"item", "date", "value", "subcategory", "category", "confidence", "auto_inserted"}); err != nil {
		return err
	}
	for _, r := range rows {
		if r.AutoInserted {
			continue
		}
		w.Write([]string{ //nolint:errcheck
			r.Item,
			r.Date,
			r.RawValue,
			r.Subcategory,
			r.Category,
			fmt.Sprintf("%.4f", r.Confidence),
			"false",
		})
	}
	w.Flush()
	return w.Error()
}

// writeRolloverCSV writes next-year installment expenses that could not be inserted
// into the current workbook. Format: item;date;value;subcategory
func writeRolloverCSV(path string, rollovers []workflow.RolloverExpense) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	w := csv.NewWriter(f)
	w.Comma = ';'
	if err := w.Write([]string{"item", "date", "value", "subcategory"}); err != nil {
		return err
	}
	for _, r := range rollovers {
		w.Write([]string{ //nolint:errcheck
			r.Expense.FormattedItem(),
			r.Expense.Date.Format("02/01"),
			utils.FormatBRValue(r.Expense.Value),
			r.Expense.Subcategory,
		})
	}
	w.Flush()
	return w.Error()
}
