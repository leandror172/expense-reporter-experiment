package cmd

import (
	"expense-reporter/internal/workflow"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

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
	rootCmd.AddCommand(addCmd)
}

func runAdd(cmd *cobra.Command, args []string) error {
	expenseString := args[0]

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
	return nil
}
