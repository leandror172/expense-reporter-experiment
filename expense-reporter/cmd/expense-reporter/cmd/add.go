package cmd

import (
	"expense-reporter/internal/classifier"
	"expense-reporter/internal/config"
	"expense-reporter/internal/feedback"
	"expense-reporter/internal/workflow"
	"expense-reporter/pkg/utils"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

var addDryRun bool

var addCmd = &cobra.Command{
	Use:   "add \"<item>;<DD/MM>;<##,##>;<subcat>\"",
	Short: "Add a single expense",
	Long: `Add a single expense to the budget spreadsheet.

The expense format is: <item_description>;<DD/MM>;<value>;<sub_category>

Examples:
  expense-reporter add "Uber Centro;15/04;35,50;Uber/Taxi"
  expense-reporter add "Compras Carrefour;03/01;150,00;Supermercado"
  expense-reporter add "Consulta veterinária;10/03;180,00;Orion - Consultas"

Notes:
  - Date year is always 2025
  - Value uses Brazilian format (##,## with comma as decimal)
  - Subcategory must exist in the reference sheet
  - Smart matching: "Orion - Consultas" finds "Orion" if exact match not found`,
	Args: cobra.ExactArgs(1),
	RunE: runAdd,
}

func init() {
	addCmd.Flags().BoolVar(&addDryRun, "dry-run", false, "Validate and parse without inserting into workbook")
	rootCmd.AddCommand(addCmd)
}

func runAdd(cmd *cobra.Command, args []string) error {
	expenseString := args[0]

	item, date, value, subcategory, ok := parseExpenseForFeedback(expenseString)
	if !ok {
		return fmt.Errorf("invalid expense format: expected \"item;DD/MM;value;subcategory\"")
	}

	category := resolveCategoryFromTaxonomy(subcategory, "data/classification")

	if addDryRun {
		return runAddDryRun(cmd, item, date, value, subcategory, category)
	}

	workbook, err := GetWorkbookPath()
	if err != nil {
		return fmt.Errorf("failed to get workbook path: %w", err)
	}

	// Verify workbook exists
	if _, err := os.Stat(workbook); os.IsNotExist(err) {
		return fmt.Errorf("workbook not found at: %s", workbook)
	}

	err = workflow.InsertExpense(workbook, expenseString)
	if err != nil {
		return fmt.Errorf("failed to insert expense: %w", err)
	}

	fmt.Println("✓ Expense added successfully!")
	logParsedManualFeedback(item, date, value, subcategory, category)
	return nil
}

// AddOutput represents the structured output of an add --dry-run command.
type AddOutput struct {
	Item        string `json:"item"`
	Value       float64 `json:"value"`
	Date        string `json:"date"`
	Subcategory string `json:"subcategory"`
	Category    string `json:"category"`
	Action      string `json:"action"`
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

// logParsedManualFeedback appends feedback and expense log entries using pre-parsed values.
// Non-fatal: any failure silently skips logging.
func logParsedManualFeedback(item, date string, value float64, subcategory, category string) {
	appCfg, err := config.Load()
	if err != nil {
		return
	}
	logManualFeedback(appCfg, item, date, value, subcategory, category)
	logExpense(appCfg, item, date, value, subcategory, category)
}

// parseExpenseForFeedback splits "item;DD/MM;value;subcategory" and parses the value.
func parseExpenseForFeedback(expenseString string) (item, date string, value float64, subcategory string, ok bool) {
	parts := strings.SplitN(expenseString, ";", 4)
	if len(parts) != 4 {
		return "", "", 0, "", false
	}
	item = strings.TrimSpace(parts[0])
	date = strings.TrimSpace(parts[1])
	valueStr := strings.TrimSpace(parts[2])
	subcategory = strings.TrimSpace(parts[3])
	if item == "" || date == "" || subcategory == "" {
		return "", "", 0, "", false
	}
	v, err := utils.ParseCurrency(valueStr)
	if err != nil {
		return "", "", 0, "", false
	}
	return item, date, v, subcategory, true
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
