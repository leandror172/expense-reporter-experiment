package cmd

import (
	"fmt"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"

	"expense-reporter/internal/generate"
)

var (
	generateOutput        string
	generateTaxonomy      string
	generateEntries       string
	generateIncomeEntries string
	generateYear          int
	generateHeadroom      int
)

var generateWorkbookCmd = &cobra.Command{
	Use:   "generate-workbook",
	Short: "Generate a complete expense workbook from a taxonomy file",
	Long: `Builds a workbook (Listas de itens + Receitas + one sheet per expense group)
from a JSON taxonomy file; optionally fills it with entries from an expenses_log.jsonl file.
The workbook is regenerated from data, never inserted into.`,
	RunE: runGenerateWorkbook,
}

func init() {
	rootCmd.AddCommand(generateWorkbookCmd)

	generateWorkbookCmd.Flags().StringVarP(&generateOutput, "output", "o", "", "Output .xlsx path (required)")
	generateWorkbookCmd.Flags().StringVar(&generateTaxonomy, "taxonomy", "", "Taxonomy JSON file path (required)")
	generateWorkbookCmd.Flags().StringVar(&generateEntries, "entries", "", "Entries JSONL path (optional)")
	generateWorkbookCmd.Flags().StringVar(&generateIncomeEntries, "income-entries", "", "Income entries JSONL path (income_log.jsonl schema; optional)")
	generateWorkbookCmd.Flags().IntVar(&generateYear, "year", time.Now().Year(), "Year applied to entry dates")
	generateWorkbookCmd.Flags().IntVar(&generateHeadroom, "headroom", 0, "Spare data rows per block beyond busiest month")

	if err := generateWorkbookCmd.MarkFlagRequired("output"); err != nil {
		panic(err)
	}
	if err := generateWorkbookCmd.MarkFlagRequired("taxonomy"); err != nil {
		panic(err)
	}
}

func runGenerateWorkbook(cmd *cobra.Command, args []string) error {
	opts := generate.Options{
		TaxonomyPath:      generateTaxonomy,
		EntriesPath:       generateEntries,
		IncomeEntriesPath: generateIncomeEntries,
		OutPath:           generateOutput,
		Year:              generateYear,
		Headroom:          generateHeadroom,
	}

	if err := generate.Generate(opts); err != nil {
		return fmt.Errorf("generating workbook: %w", err)
	}

	absPath, err := filepath.Abs(generateOutput)
	if err != nil {
		return fmt.Errorf("getting absolute path: %w", err)
	}

	fmt.Fprintf(cmd.OutOrStdout(), "workbook written: %s\n", absPath)
	return nil
}
