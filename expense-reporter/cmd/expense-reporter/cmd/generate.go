package cmd

import (
	"fmt"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"

	internalconfig "expense-reporter/internal/config"
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
	generateWorkbookCmd.Flags().StringVar(&generateTaxonomy, "taxonomy", "", "Taxonomy JSON file path (default: config taxonomy_path)")
	generateWorkbookCmd.Flags().StringVar(&generateEntries, "entries", "", "Entries JSONL path (optional)")
	generateWorkbookCmd.Flags().StringVar(&generateIncomeEntries, "income-entries", "", "Income entries JSONL path (income_log.jsonl schema; optional)")
	generateWorkbookCmd.Flags().IntVar(&generateYear, "year", time.Now().Year(), "Year applied to entry dates")
	generateWorkbookCmd.Flags().IntVar(&generateHeadroom, "headroom", 0, "Spare data rows per block beyond busiest month")

	if err := generateWorkbookCmd.MarkFlagRequired("output"); err != nil {
		panic(err)
	}
	// taxonomy is NOT MarkFlagRequired: it is a required VALUE, which config can supply.
	// resolveTaxonomyPath enforces it.
}

// resolveTaxonomyPath prefers the flag and falls back to config, so generate-workbook reads
// the same taxonomy_path every other command does. It used to REQUIRE the flag — the one
// command whose output IS the deliverable was the only one ignoring the configured
// taxonomy, which the T-42 scout hit as friction mid-close.
//
// --entries deliberately does NOT get the same treatment. Omitting it is the documented way
// to generate an empty skeleton for a new year (TestGenerateWorkbook_Skeleton, the
// type-routing-cycle Given, and the T-03 rollover plan all rely on that), so defaulting it
// from config would silently make "give me an empty skeleton" impossible to express.
func resolveTaxonomyPath(cfg *internalconfig.Config) (string, error) {
	if generateTaxonomy != "" {
		return generateTaxonomy, nil
	}
	if path := cfg.TaxonomyFilePath(); path != "" {
		return path, nil
	}
	return "", fmt.Errorf("no taxonomy: pass --taxonomy or set taxonomy_path in config.json")
}

func runGenerateWorkbook(cmd *cobra.Command, args []string) error {
	cfg, err := internalconfig.Load()
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}
	taxonomyPath, err := resolveTaxonomyPath(cfg)
	if err != nil {
		return err
	}

	opts := generate.Options{
		TaxonomyPath:      taxonomyPath,
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
