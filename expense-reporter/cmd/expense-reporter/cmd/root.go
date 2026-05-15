package cmd

import (
	"expense-reporter/internal/config"
	"expense-reporter/internal/logger"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var workbookPath string
var verbose bool
var outputJSON bool

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
	rootCmd.PersistentFlags().BoolVar(&outputJSON, "json", false,
		"Output in JSON format (machine-readable)")
}

// GetWorkbookPath returns the path to the Excel workbook.
// Resolution order: --workbook flag → EXPENSE_WORKBOOK_PATH env → config.json workbook_path → error.
func GetWorkbookPath() (string, error) {
	if workbookPath != "" {
		return workbookPath, nil
	}

	if path := os.Getenv("EXPENSE_WORKBOOK_PATH"); path != "" {
		return path, nil
	}

	cfg, err := config.Load()
	if err != nil {
		return "", fmt.Errorf("loading config: %w", err)
	}
	if p := cfg.WorkbookFilePath(); p != "" {
		return p, nil
	}

	return "", fmt.Errorf("workbook path not configured: use --workbook <path> or set EXPENSE_WORKBOOK_PATH")
}
