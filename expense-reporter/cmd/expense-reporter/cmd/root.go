package cmd

import (
	"expense-reporter/internal/logger"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

var workbookPath string
var verbose bool

var rootCmd = &cobra.Command{
	Use:   "expense-reporter",
	Short: "Automate expense reporting to Excel budget",
	Long: `expense-reporter is a CLI tool that automates the tedious
task of entering expenses into your Excel budget spreadsheet.

The tool supports:
  - Adding single expenses
  - Batch importing from CSV files
  - Smart subcategory matching
  - Automatic backup creation`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		logger.SetDebug(verbose)
		return nil
	},
}

// Execute runs the root command
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVar(&workbookPath, "workbook", "",
		"Path to Excel workbook (or use EXPENSE_WORKBOOK_PATH env)")
	rootCmd.PersistentFlags().BoolVar(&verbose, "verbose", false,
		"Verbose output")
}

// GetWorkbookPath returns the path to the Excel workbook
// Uses flag value if set, otherwise environment variable, otherwise default
func GetWorkbookPath() (string, error) {
	// Check flag value first
	if workbookPath != "" {
		return workbookPath, nil
	}

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
