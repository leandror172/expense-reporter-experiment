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
	Value        float64
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

	if !batchAutoDryRun {
		if err := insertClassified(results); err != nil {
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

	printBatchSummary(results, batchAutoDryRun, classifiedPath, reviewPath)
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
			results = append(results, classifiedRow{Item: row.Item, Date: row.Date, Value: row.Value, Error: err})
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
			Value:        row.Value,
			Subcategory:  top.Subcategory,
			Category:     top.Category,
			Confidence:   top.Confidence,
			AutoInserted: autoInsert,
		})
	}
	return results
}

func insertClassified(results []classifiedRow) error {
	workbook, err := GetWorkbookPath()
	if err != nil {
		return fmt.Errorf("getting workbook path: %w", err)
	}
	if _, err := os.Stat(workbook); os.IsNotExist(err) {
		return fmt.Errorf("workbook not found: %s", workbook)
	}

	// Backup before any modifications
	if _, err := batch.NewBackupManager().CreateBackup(workbook); err != nil {
		return fmt.Errorf("creating backup: %w", err)
	}

	// Build insert strings for high-confidence rows, track their source indices
	var srcIdx []int
	var insertStrings []string
	for i, r := range results {
		if r.AutoInserted && r.Error == nil {
			srcIdx = append(srcIdx, i)
			insertStrings = append(insertStrings, utils.BuildInsertString(r.Item, r.Date, r.Value, r.Subcategory))
		}
	}
	if len(insertStrings) == 0 {
		return nil
	}

	// Use Processor.ProcessBatch for optimised single-open batch insert
	p := batch.NewProcessor(workbook)
	if err := p.LoadMappings(); err != nil {
		return fmt.Errorf("loading workbook mappings: %w", err)
	}
	summary, _, err := p.ProcessBatch(insertStrings, nil)
	if err != nil {
		return fmt.Errorf("batch insert: %w", err)
	}

	// Map failures back to classified rows
	for i, result := range summary.Results {
		if !result.Success {
			results[srcIdx[i]].AutoInserted = false
			results[srcIdx[i]].Error = result.Error
			fmt.Fprintf(os.Stderr, "  INSERT ERROR %q: %v\n", results[srcIdx[i]].Item, result.Error)
		}
	}
	return nil
}

func printBatchSummary(results []classifiedRow, dryRun bool, classifiedPath, reviewPath string) {
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
	fmt.Printf("  classified.csv: %s\n", classifiedPath)
	fmt.Printf("  review.csv    : %s\n", reviewPath)
}

// inputRow is a parsed 3-field line.
type inputRow struct {
	Item  string
	Date  string
	Value float64
}

// parse3FieldLine splits "item;DD/MM;value" and parses currency.
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
	v, err := utils.ParseCurrency(valueStr)
	if err != nil {
		return inputRow{}, fmt.Errorf("parsing value %q: %w", valueStr, err)
	}
	return inputRow{Item: item, Date: date, Value: v}, nil
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
			utils.FormatBRValue(r.Value),
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
			utils.FormatBRValue(r.Value),
			r.Subcategory,
			r.Category,
			fmt.Sprintf("%.4f", r.Confidence),
			"false",
		})
	}
	w.Flush()
	return w.Error()
}
