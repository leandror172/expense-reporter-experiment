package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"

	internalconfig "expense-reporter/internal/config"
	"expense-reporter/internal/excel"
	"expense-reporter/internal/review"
)

var (
	reviewOutput   string
	reviewWorkbook string
	reviewForce    bool
)

var reviewCmd = &cobra.Command{
	Use:   "review <classified.csv>",
	Short: "Generate HTML review page from classified CSV",
	Long: `Generate an interactive HTML review page from a classified CSV file.
The output file contains the full expense queue and taxonomy, ready to open in a browser.

The output file is NOT overwritten without --force. Use --force to replace an existing file.
Use -o - to write to stdout; the summary line is written to stderr in that case.

Examples:
  expense-reporter review classified.csv
  expense-reporter review classified.csv --output review.html
  expense-reporter review classified.csv --workbook /path/to/workbook.xlsx
  expense-reporter review classified.csv -o -`,
	Args: cobra.ExactArgs(1),
	RunE: runReview,
}

func init() {
	rootCmd.AddCommand(reviewCmd)
	reviewCmd.Flags().StringVarP(&reviewOutput, "output", "o", "review.html", "Output HTML file path")
	reviewCmd.Flags().StringVar(&reviewWorkbook, "workbook", "", "Workbook path (overrides config)")
	reviewCmd.Flags().BoolVarP(&reviewForce, "force", "f", false, "Overwrite output file if it exists")
}

func runReview(cmd *cobra.Command, args []string) error {
	cfg, err := internalconfig.Load()
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	workbookPath := reviewWorkbook
	if workbookPath == "" {
		workbookPath = cfg.WorkbookFilePath()
	}
	if workbookPath == "" {
		return fmt.Errorf("workbook path not configured (set EXPENSE_WORKBOOK or use --workbook)")
	}

	if err := excel.ValidateWorkbook(workbookPath); err != nil {
		return fmt.Errorf("validating workbook: %w", err)
	}

	mappings, err := excel.LoadReferenceSheet(workbookPath)
	if err != nil {
		return fmt.Errorf("loading reference sheet: %w", err)
	}

	taxonomy := review.BuildTaxonomy(mappings)

	queue, err := review.ReadQueue(args[0])
	if err != nil {
		return fmt.Errorf("reading queue: %w", err)
	}
	if len(queue) == 0 {
		return fmt.Errorf("no rows to review")
	}

	data := review.ReviewData{
		Source:      filepath.Base(args[0]),
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
		Queue:       queue,
		Taxonomy:    taxonomy,
	}

	html, err := review.Render(review.TemplateHTML, data)
	if err != nil {
		return fmt.Errorf("rendering HTML: %w", err)
	}

	needsReview := 0
	for _, entry := range queue {
		if !entry.AutoInserted {
			needsReview++
		}
	}

	if reviewOutput == "-" {
		if _, err := fmt.Fprint(cmd.OutOrStdout(), html); err != nil {
			return fmt.Errorf("writing to stdout: %w", err)
		}
		fmt.Fprintf(cmd.ErrOrStderr(), "wrote to stdout — %d rows (%d need review)\n", len(queue), needsReview)
	} else {
		if _, err := os.Stat(reviewOutput); err == nil && !reviewForce {
			return fmt.Errorf("output file %q already exists (use --force to overwrite)", reviewOutput)
		}
		if err := os.WriteFile(reviewOutput, []byte(html), 0o644); err != nil {
			return fmt.Errorf("writing output file: %w", err)
		}
		fmt.Fprintf(cmd.OutOrStdout(), "wrote %s — %d rows (%d need review)\n", reviewOutput, len(queue), needsReview)
	}

	return nil
}
