package cmd

import (
	"expense-reporter/internal/classifier"
	"expense-reporter/internal/config"
	"expense-reporter/internal/feedback"
	"expense-reporter/internal/parse"
	taxonomy "expense-reporter/internal/taxonomy"
	"fmt"

	"github.com/spf13/cobra"
)

var correctDataDir string
var correctYear int

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
	correctCmd.Flags().StringVar(&correctDataDir, "data-dir", "data/classification", "(deprecated, no longer used: category resolves via config/taxonomy.json since T-13)")
	correctCmd.Flags().IntVar(&correctYear, "year", 0, "Fallback year for bare DD/MM dates (outranks config date_year; an explicit year in the date always wins)")
	rootCmd.AddCommand(correctCmd)
}

func runCorrect(cmd *cobra.Command, args []string) error {
	appCfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	// T-41: parse at the boundary; config loads first so date_year feeds the
	// year ladder. The canonical DateString keeps GenerateID's lookup bytes
	// aligned with what add/auto wrote.
	pe, actualSubcategory, err := parse.ExpenseString(args[0], parseOptions(correctYear, appCfg))
	if err != nil {
		return fmt.Errorf("invalid expense format: expected \"item;DD/MM;value;subcategory\": %w", err)
	}
	warnIfStaleConfiguredYear(pe, appCfg)
	item, date, value := pe.Item, pe.DateString(), pe.Value

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

	// Category is resolved from taxonomy.json (the single source of truth), but the
	// whole resolution is best-effort: a corrected entry is feedback-only and never
	// feeds generate-workbook, so neither an unloadable taxonomy nor an unknown
	// subcategory should block logging the correction — both degrade to an empty
	// category (matching the prior feature-dict behavior).
	actualCategory := ""
	if sheets, terr := loadTaxonomyTree(appCfg); terr == nil {
		actualCategory, _ = taxonomy.CategoryForLeaf(sheets, actualSubcategory)
	}

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
