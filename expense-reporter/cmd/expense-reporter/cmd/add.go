package cmd

import (
	"expense-reporter/internal/appender"
	"expense-reporter/internal/classifier"
	"expense-reporter/internal/config"
	"expense-reporter/internal/feedback"
	"expense-reporter/pkg/utils"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

var addDryRun bool
var addDataDir string

var addPredictedSubcategory string
var addPredictedCategory string
var addClassificationID string
var addConfidence float64
var addModel string

var addCmd = &cobra.Command{
	Use:   "add \"<item>;<DD/MM[/YYYY]>;<##,##[/N]>;<subcat>\"",
	Short: "Add a single expense",
	Long: `Add a single expense to the expense log.

The expense format is: <item_description>;<DD/MM or DD/MM/YYYY>;<value>;<sub_category>

Installment notation: append /N to the value to expand into N monthly log entries.

Examples:
  expense-reporter add "Uber Centro;15/04/2026;35,50;Uber/Taxi"
  expense-reporter add "Compras Carrefour;03/01/2026;150,00;Supermercado"
  expense-reporter add "Curso online;15/11/2026;90,00/3;Amazon"

Notes:
  - Date accepts DD/MM (defaults to current year) or DD/MM/YYYY
  - Value uses Brazilian format (##,## with comma as decimal)
  - Installments expand into N dated log entries — cross-year dates use the real next-year date
  - Subcategory must exist in the taxonomy`,
	Args: cobra.ExactArgs(1),
	RunE: runAdd,
}

func init() {
	addCmd.Flags().BoolVar(&addDryRun, "dry-run", false, "Validate and parse without inserting into log")
	addCmd.Flags().StringVar(&addDataDir, "data-dir", "data/classification", "Path to classification data directory")
	addCmd.Flags().StringVar(&addPredictedSubcategory, "predicted-subcategory", "", "Model's top prediction for subcategory")
	addCmd.Flags().StringVar(&addPredictedCategory, "predicted-category", "", "Model's predicted category")
	addCmd.Flags().StringVar(&addClassificationID, "classification-id", "", "ID from the prior classify call (for cross-reference)")
	addCmd.Flags().Float64Var(&addConfidence, "confidence", 0.0, "Model's confidence score")
	addCmd.Flags().StringVar(&addModel, "model", "", "Model name used for classification")
	rootCmd.AddCommand(addCmd)
}

func runAdd(cmd *cobra.Command, args []string) error {
	expenseString := args[0]

	item, dateStr, parsedDate, value, installmentCount, subcategory, ok := parseExpenseForFeedback(expenseString)
	if !ok {
		return fmt.Errorf("invalid expense format: expected \"item;DD/MM[/YYYY];value[/N];subcategory\"")
	}

	category := resolveCategoryFromTaxonomy(subcategory, addDataDir)

	if addDryRun {
		return runAddDryRun(cmd, item, dateStr, value, subcategory, category)
	}

	appCfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	typ := resolveExpenseType(loadTypeIndex(appCfg), category, subcategory)

	if logPath := appCfg.ExpensesLogFilePath(); logPath != "" {
		if err := appender.ExpandAndAppend(logPath, item, parsedDate, value, installmentCount, typ, category, subcategory); err != nil {
			fmt.Fprintf(os.Stderr, "⚠  expense log: %v\n", err)
		}
	}

	if addPredictedSubcategory != "" {
		logPredictedFeedback(appCfg, item, dateStr, value, subcategory, category,
			addPredictedSubcategory, addPredictedCategory, addClassificationID,
			addConfidence, addModel)
	} else {
		logManualFeedback(appCfg, item, dateStr, value, subcategory, category)
	}

	fmt.Println("✓ Expense added successfully!")
	return nil
}

// AddOutput represents the structured output of an add --dry-run command.
type AddOutput struct {
	Item        string  `json:"item"`
	Value       float64 `json:"value"`
	Date        string  `json:"date"`
	Subcategory string  `json:"subcategory"`
	Category    string  `json:"category"`
	Action      string  `json:"action"`
}

func runAddDryRun(cmd *cobra.Command, item, date string, value float64, subcategory, category string) error {
	jsonMode, _ := cmd.Flags().GetBool("json")

	if jsonMode {
		return printJSON(AddOutput{
			Item:        item,
			Value:       value,
			Date:        date,
			Subcategory: subcategory,
			Category:    category,
			Action:      "would_insert",
		})
	}

	fmt.Printf("Dry run — would insert:\n")
	fmt.Printf("  Item:        %s\n", item)
	fmt.Printf("  Date:        %s\n", date)
	fmt.Printf("  Value:       %.2f\n", value)
	fmt.Printf("  Subcategory: %s\n", subcategory)
	if category != "" {
		fmt.Printf("  Category:    %s\n", category)
	}
	return nil
}

// logParsedManualFeedback appends feedback log entry using pre-parsed values.
// Non-fatal: any failure silently skips logging.
func logParsedManualFeedback(item, date string, value float64, subcategory, category string) {
	appCfg, err := config.Load()
	if err != nil {
		return
	}
	logManualFeedback(appCfg, item, date, value, subcategory, category)
}

// parseExpenseForFeedback splits "item;DD/MM[/YYYY];value[/N];subcategory" and parses date + value.
// Returns the formatted dateStr (DD/MM/YYYY), parsed time.Time, per-installment value, and installment count.
func parseExpenseForFeedback(expenseString string) (item, dateStr string, parsedDate time.Time, value float64, installmentCount int, subcategory string, ok bool) {
	parts := strings.SplitN(expenseString, ";", 4)
	if len(parts) != 4 {
		return
	}
	item = strings.TrimSpace(parts[0])
	rawDate := strings.TrimSpace(parts[1])
	valueStr := strings.TrimSpace(parts[2])
	subcategory = strings.TrimSpace(parts[3])
	if item == "" || rawDate == "" || subcategory == "" {
		return
	}
	t, err := utils.ParseDateFlexible(rawDate)
	if err != nil {
		return
	}
	parsedDate = t
	dateStr = utils.FormatDate(t)
	v, count, err := utils.ParseCurrencyWithInstallments(valueStr)
	if err != nil {
		return
	}
	value = v
	installmentCount = count
	ok = true
	return
}

// resolveCategoryFromTaxonomy looks up subcategory in the taxonomy to find its parent category.
// Returns empty string if the taxonomy cannot be loaded or the subcategory is not found.
func resolveCategoryFromTaxonomy(subcategory, dataDir string) string {
	taxonomy, err := classifier.LoadTaxonomy(dataDir)
	if err != nil {
		return ""
	}
	return taxonomy[subcategory]
}

// logManualFeedback appends a manual entry to classifications.jsonl.
// Non-fatal: warns on stderr if writing fails.
func logManualFeedback(appCfg *config.Config, item, date string, value float64, subcategory, category string) {
	path := appCfg.ClassificationsFilePath()
	if path == "" {
		return
	}
	entry := feedback.NewManualEntry(item, date, value, subcategory, category)
	if err := feedback.Append(path, entry); err != nil {
		fmt.Fprintf(os.Stderr, "⚠  feedback log: %v\n", err)
	}
}

// logPredictedFeedback writes a confirmed or corrected feedback entry to classifications.jsonl,
// depending on whether the user's chosen subcategory matches the model's prediction.
// Non-fatal: warns on stderr if the classification-id cross-reference misses or if the write fails.
func logPredictedFeedback(appCfg *config.Config, item, date string, value float64,
	chosenSubcategory, chosenCategory, predictedSubcategory, predictedCategory, classificationID string,
	confidence float64, model string) {

	path := appCfg.ClassificationsFilePath()

	if classificationID != "" && path != "" {
		_, found, err := feedback.FindLatestEntry(path, classificationID)
		if !found || err != nil {
			fmt.Fprintf(os.Stderr, "⚠  feedback log: classification-id %q not found, continuing without cross-reference\n", classificationID)
		}
	}

	if path == "" {
		return
	}

	predicted := classifier.Result{
		Subcategory: predictedSubcategory,
		Category:    predictedCategory,
		Confidence:  confidence,
	}

	var entry feedback.Entry
	if chosenSubcategory == predictedSubcategory {
		entry = feedback.NewConfirmedEntry(item, date, value, predicted, model)
	} else {
		entry = feedback.NewCorrectedEntry(item, date, value, predicted, model, chosenSubcategory, chosenCategory)
	}

	if err := feedback.Append(path, entry); err != nil {
		fmt.Fprintf(os.Stderr, "⚠  feedback log: %v\n", err)
	}
}
