package main

import (
	"expense-reporter/internal/cli"
	"expense-reporter/internal/excel"
	"expense-reporter/internal/resolver"
	"expense-reporter/internal/workflow"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

func main() {
	if err := runCLI(os.Stdout); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

// runCLI executes the CLI logic (separated for testing)
func runCLI(output io.Writer) error {
	// Parse arguments (skip program name)
	command, input, err := cli.ParseArgs(os.Args[1:])
	if err != nil {
		cli.PrintHelp(output)
		return err
	}

	// Handle commands
	switch command {
	case "help":
		cli.PrintHelp(output)
		return nil

	case "version":
		cli.PrintVersion(output)
		return nil

	case "add":
		return handleAddExpense(input, output)

	default:
		cli.PrintHelp(output)
		return fmt.Errorf("unknown command: %s", command)
	}
}

// handleAddExpense processes the add expense command
func handleAddExpense(expenseString string, output io.Writer) error {
	// Get workbook path
	workbookPath, err := GetWorkbookPath()
	if err != nil {
		return fmt.Errorf("failed to get workbook path: %w", err)
	}

	// Verify workbook exists
	if _, err := os.Stat(workbookPath); os.IsNotExist(err) {
		return fmt.Errorf("workbook not found at: %s", workbookPath)
	}

	// Try to insert the expense
	err = workflow.InsertExpense(workbookPath, expenseString)

	// Check if it's an ambiguous subcategory error
	if err != nil && isAmbiguousError(err) {
		return handleAmbiguousSubcategory(workbookPath, expenseString, output)
	}

	if err != nil {
		return fmt.Errorf("failed to insert expense: %w", err)
	}

	fmt.Fprintf(output, "✓ Expense added successfully!\n")
	return nil
}

// handleAmbiguousSubcategory prompts user to select from multiple options
func handleAmbiguousSubcategory(workbookPath, expenseString string, output io.Writer) error {
	// This is a simplified version - in a real implementation, we would:
	// 1. Parse the expense to get the subcategory
	// 2. Load mappings and get ambiguous options
	// 3. Prompt user to select
	// 4. Re-insert with the selected sheet

	// For now, just return the error with guidance
	return fmt.Errorf("ambiguous subcategory detected. Please specify the full path in the format: <sheet>/<category>/<subcategory>")
}

// isAmbiguousError checks if an error is due to ambiguous subcategory
func isAmbiguousError(err error) bool {
	if err == nil {
		return false
	}
	return containsString(err.Error(), "ambiguous")
}

// GetWorkbookPath returns the path to the Excel workbook
// Uses environment variable EXPENSE_WORKBOOK_PATH if set, otherwise uses default
func GetWorkbookPath() (string, error) {
	// Check environment variable
	if path := os.Getenv("EXPENSE_WORKBOOK_PATH"); path != "" {
		return path, nil
	}

	// Use default path (relative to executable location)
	execPath, err := os.Executable()
	if err != nil {
		// Fallback to hardcoded path
		return "Z:\\Meu Drive\\controle\\code\\Planilha_BMeFBovespa_Leandro_OrcamentoPessoal-2025.xlsx", nil
	}

	execDir := filepath.Dir(execPath)
	defaultPath := filepath.Join(execDir, "..", "Planilha_BMeFBovespa_Leandro_OrcamentoPessoal-2025.xlsx")

	return defaultPath, nil
}

// containsString checks if a string contains a substring (case-insensitive helper)
func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// Enhanced version that handles ambiguous subcategories with user prompt
func handleAmbiguousWithPrompt(workbookPath, expenseString string, output io.Writer) error {
	// Parse expense to extract subcategory
	// (We would need to add this to parser or workflow package)

	// Load reference sheet
	mappings, err := excel.LoadReferenceSheet(workbookPath)
	if err != nil {
		return fmt.Errorf("failed to load reference sheet: %w", err)
	}

	// Get subcategory from expense string
	// For now, we'll extract it manually (this should be in parser)
	parts := splitExpenseString(expenseString)
	if len(parts) != 4 {
		return fmt.Errorf("invalid expense format")
	}
	subcategory := parts[3]

	// Try to extract parent if it's a detailed subcategory
	parent := resolver.ExtractParentSubcategory(subcategory)
	searchKey := parent

	// Get ambiguous options
	options := resolver.GetAmbiguousOptions(mappings, searchKey)
	if len(options) == 0 {
		return fmt.Errorf("no options found for subcategory: %s", subcategory)
	}

	// Format options for display
	optionStrings := make([]string, len(options))
	for i, opt := range options {
		optionStrings[i] = fmt.Sprintf("%s - %s", opt.SheetName, opt.Subcategory)
	}

	// Prompt user to select
	selected, err := cli.PromptForSelection(optionStrings, os.Stdin, output)
	if err != nil {
		return fmt.Errorf("failed to get user selection: %w", err)
	}

	// TODO: Re-insert expense with selected option
	// This would require modifying workflow.InsertExpense to accept a specific sheet
	fmt.Fprintf(output, "\nSelected: %s\n", optionStrings[selected])
	fmt.Fprintf(output, "Note: Ambiguous resolution with selection not yet fully implemented.\n")

	return nil
}

// splitExpenseString splits expense string by semicolon
func splitExpenseString(s string) []string {
	result := []string{}
	current := ""
	for _, ch := range s {
		if ch == ';' {
			result = append(result, current)
			current = ""
		} else {
			current += string(ch)
		}
	}
	if current != "" {
		result = append(result, current)
	}
	return result
}
