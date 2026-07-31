package cmd

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"

	internalconfig "expense-reporter/internal/config"
	"expense-reporter/internal/review"
	"expense-reporter/internal/taxonomy"
)

var (
	reviewOutput string
	reviewForce  bool
)

var reviewCmd = &cobra.Command{
	Use:   "review <classified.csv>",
	Short: "Generate HTML review page from classified CSV",
	Long: `Generate an interactive HTML review page from a classified CSV file.
The output file contains the full expense queue and taxonomy, ready to open in a browser.

The output file is NOT overwritten without --force. Use --force to replace an existing file.
Use -o - to write to stdout; the summary line is written to stderr in that case.

The picker's categories come from the configured taxonomy (config/taxonomy.json) — the
same file classify, auto, batch-auto and generate-workbook use. No workbook is read.

A row batch-auto could not parse is recorded in classified.csv with empty date/value
cells; those rows are left out of the queue and named on stderr, not treated as fatal.

Examples:
  expense-reporter review classified.csv
  expense-reporter review classified.csv --output review.html
  expense-reporter review classified.csv -o -`,
	Args: cobra.ExactArgs(1),
	RunE: runReview,
}

func init() {
	rootCmd.AddCommand(reviewCmd)
	reviewCmd.Flags().StringVarP(&reviewOutput, "output", "o", "review.html", "Output HTML file path")
	reviewCmd.Flags().BoolVarP(&reviewForce, "force", "f", false, "Overwrite output file if it exists")
}

func runReview(cmd *cobra.Command, args []string) error {
	cfg, err := internalconfig.Load()
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	taxonomyPath := cfg.TaxonomyFilePath()
	if taxonomyPath == "" {
		return fmt.Errorf("taxonomy path not configured (set taxonomy_path in config.json)")
	}

	// Entries and income are irrelevant here — the picker needs the tree, not the data —
	// so both paths are empty and the year filter is 0 (keep everything).
	types, _, err := taxonomy.LoadTaxonomy(taxonomyPath, "", "", 0)
	if err != nil {
		return fmt.Errorf("loading taxonomy: %w", err)
	}

	pickerTaxonomy := review.BuildTaxonomy(types)

	queue, unreviewable, err := review.ReadQueue(args[0])
	if err != nil {
		return fmt.Errorf("reading queue: %w", err)
	}
	reportUnreviewableRows(cmd.ErrOrStderr(), unreviewable)
	if len(queue) == 0 {
		return fmt.Errorf("no rows to review")
	}

	data := review.ReviewData{
		Source:      filepath.Base(args[0]),
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
		Queue:       queue,
		Taxonomy:    pickerTaxonomy,
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

// reportUnreviewableRows names every row batch-auto could not parse, so a row dropped from
// the queue is visible rather than merely absent.
//
// The message lives here rather than in ReadQueue for the reason parse reports YearSource
// instead of printing: a library that owns a stderr contract cannot be reused by a caller
// wanting a different one, and its output can only be tested by capturing stderr.
//
// Per-row lines are right here, unlike the T-49 stale-year warning that had to collapse to
// one counted line. That one fired once per GOOD row (300 lines for a 300-row batch); this
// fires once per PROBLEM, and the raw text is exactly what the user needs to fix the source
// and re-run. stderr because --json owns stdout.
func reportUnreviewableRows(w io.Writer, rows []string) {
	if len(rows) == 0 {
		return
	}
	fmt.Fprintf(w, "warning: %d row(s) could not be parsed upstream and are NOT in the review queue; fix them in the source CSV and re-run:\n", len(rows))
	for _, raw := range rows {
		fmt.Fprintf(w, "  unreviewable: %s\n", raw)
	}
}
