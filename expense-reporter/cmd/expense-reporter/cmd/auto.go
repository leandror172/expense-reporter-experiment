package cmd

import (
	"bufio"
	"expense-reporter/internal/classifier"
	"expense-reporter/internal/config"
	"expense-reporter/internal/workflow"
	"expense-reporter/pkg/utils"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

const highConfidenceThreshold = 0.85

var (
	autoModel   string
	autoDataDir string
	autoConfirm bool
)

var autoCmd = &cobra.Command{
	Use:   "auto <item> <value> <DD/MM>",
	Short: "Classify and auto-insert if confident",
	Long: `Classify an expense and insert it automatically if confidence is high (≥85%).
If confidence is below the threshold, prints candidates for manual review.

Examples:
  expense-reporter auto "Uber Centro" 35.50 15/04
  expense-reporter auto "Diarista Letícia" 160,00 05/01 --confirm`,
	Args: cobra.ExactArgs(3),
	RunE: runAuto,
}

func init() {
	rootCmd.AddCommand(autoCmd)
	autoCmd.Flags().StringVar(&autoModel, "model", "my-classifier-q3", "Ollama model to use")
	autoCmd.Flags().StringVar(&autoDataDir, "data-dir", "data/classification", "Path to classification data directory")
	autoCmd.Flags().BoolVar(&autoConfirm, "confirm", false, "Always ask for confirmation before inserting")
}

func runAuto(cmd *cobra.Command, args []string) error {
	item := args[0]

	value, err := utils.ParseCurrency(args[1])
	if err != nil {
		return fmt.Errorf("invalid value %q: expected a number (e.g. 35.50 or 35,50)", args[1])
	}

	date := args[2]

	taxonomy, err := classifier.LoadTaxonomy(autoDataDir)
	if err != nil {
		return fmt.Errorf("loading taxonomy: %w", err)
	}

	appCfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	cfg := classifier.Config{
		OllamaURL: "http://localhost:11434",
		Model:     autoModel,
		TopN:      3,
	}

	results, err := classifier.Classify(item, value, date, taxonomy, cfg)
	if err != nil {
		return fmt.Errorf("classification failed: %w", err)
	}

	if len(results) == 0 {
		return fmt.Errorf("classifier returned no results")
	}

	top := results[0]

	if isAutoInsertable(top, appCfg.AutoInsertExcluded) {
		if autoConfirm {
			fmt.Printf("Top match: %s (%s) — %.0f%% confidence\n", top.Subcategory, top.Category, top.Confidence*100)
			fmt.Printf("Insert? [y/N] ")
			if !confirmInsert(os.Stdin) {
				printCandidates(item, value, date, results)
				fmt.Println("\n⚠  Not inserted — cancelled by user.")
				return nil
			}
		}
		return insertExpense(item, date, value, top)
	}

	printCandidates(item, value, date, results)
	if top.Confidence >= highConfidenceThreshold {
		fmt.Printf("\n⚠  Not inserted — \"%s\" is excluded from auto-insert.\n", top.Subcategory)
	} else {
		fmt.Printf("\n⚠  Not inserted — top confidence %.0f%% is below threshold %.0f%%.\n",
			top.Confidence*100, highConfidenceThreshold*100)
	}
	return nil
}

// isAutoInsertable returns true if the result meets the confidence threshold
// and its subcategory is not in the exclusion list.
func isAutoInsertable(result classifier.Result, excluded []string) bool {
	if result.Confidence < highConfidenceThreshold {
		return false
	}
	for _, ex := range excluded {
		if result.Subcategory == ex {
			return false
		}
	}
	return true
}

func insertExpense(item, date string, value float64, result classifier.Result) error {
	workbook, err := GetWorkbookPath()
	if err != nil {
		return fmt.Errorf("failed to get workbook path: %w", err)
	}

	if _, err := os.Stat(workbook); os.IsNotExist(err) {
		return fmt.Errorf("workbook not found at: %s", workbook)
	}

	insertStr := buildInsertString(item, date, value, result.Subcategory)
	if err := workflow.InsertExpense(workbook, insertStr); err != nil {
		return fmt.Errorf("failed to insert expense: %w", err)
	}

	fmt.Printf("✓ Inserted: %s → %s (%s) — %.0f%% confidence\n",
		item, result.Subcategory, result.Category, result.Confidence*100)
	return nil
}

func printCandidates(item string, value float64, date string, results []classifier.Result) {
	fmt.Printf("Classifying: %s  R$ %.2f  %s\n\n", item, value, date)
	for i, r := range results {
		bar := confidenceBar(r.Confidence)
		fmt.Printf("  %d. %-30s %-20s %s %.0f%%\n", i+1, r.Subcategory, r.Category, bar, r.Confidence*100)
	}
}

// formatBRValue formats a float64 as a Brazilian decimal string (comma separator).
func formatBRValue(v float64) string {
	s := fmt.Sprintf("%.2f", v)
	return strings.Replace(s, ".", ",", 1)
}

// buildInsertString constructs the semicolon-delimited string expected by workflow.InsertExpense.
func buildInsertString(item, date string, value float64, subcategory string) string {
	return fmt.Sprintf("%s;%s;%s;%s", item, date, formatBRValue(value), subcategory)
}

// confirmInsert reads a line from r and returns true only for "y" or "yes" (case-insensitive).
func confirmInsert(r io.Reader) bool {
	scanner := bufio.NewScanner(r)
	if !scanner.Scan() {
		return false
	}
	answer := strings.TrimSpace(strings.ToLower(scanner.Text()))
	return answer == "y" || answer == "yes"
}
