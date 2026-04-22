package cmd

import (
	"expense-reporter/internal/classifier"
	"expense-reporter/internal/config"
	"expense-reporter/internal/feedback"
	"fmt"

	"github.com/spf13/cobra"
)

var correctDataDir string

var correctCmd = &cobra.Command{
	Use:   "correct \"<item>;<DD/MM>;<##,##>;<corrected_subcat>\"",
	Short: "Log a correction overriding a prior auto-classification",
	Long: `Log a correction overriding a prior auto-classification.

This command logs a corrected feedback entry to classifications.jsonl when a
previously confirmed (auto-inserted) classification was wrong. It requires a
prior entry — for expenses with no prior model prediction, use 'add' instead.

Examples:
  expense-reporter correct "Uber Centro;15/04;35,50;Combustível"
  expense-reporter correct "Compras Carrefour;03/01;150,00;Supermercado"

Note: This command does NOT modify the workbook — it only writes to the feedback log.`,
	Args: cobra.ExactArgs(1),
	RunE: runCorrect,
}

func init() {
	correctCmd.Flags().StringVar(&correctDataDir, "data-dir", "data/classification", "Path to classification data directory")
	rootCmd.AddCommand(correctCmd)
}

func runCorrect(cmd *cobra.Command, args []string) error {
	item, date, value, actualSubcategory, ok := parseExpenseForFeedback(args[0])
	if !ok {
		return fmt.Errorf("invalid expense format: expected \"item;DD/MM;value;subcategory\"")
	}

	appCfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	path := appCfg.ClassificationsFilePath()
	if path == "" {
		return fmt.Errorf("classifications log path is not configured")
	}

	id := feedback.GenerateID(item, date, value)
	prior, found, err := feedback.FindLatestEntry(path, id)
	if err != nil {
		return fmt.Errorf("looking up prior classification: %w", err)
	}
	if !found {
		return fmt.Errorf("no prior classification found for %q on %s — use 'add' to log a manual entry instead", item, date)
	}

	actualCategory := resolveCategoryFromTaxonomy(actualSubcategory, correctDataDir)

	predicted := classifier.Result{
		Subcategory: prior.PredictedSubcategory,
		Category:    prior.PredictedCategory,
		Confidence:  prior.Confidence,
	}
	entry := feedback.NewCorrectedEntry(item, date, value, predicted, prior.Model, actualSubcategory, actualCategory)

	if err := feedback.Append(path, entry); err != nil {
		return fmt.Errorf("writing corrected entry: %w", err)
	}

	fmt.Printf("✓ Correction logged: %s → %s (was %s)\n", item, actualSubcategory, prior.PredictedSubcategory)
	return nil
}
