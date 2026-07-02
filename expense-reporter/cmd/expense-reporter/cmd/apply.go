package cmd

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"

	"expense-reporter/internal/appender"
	"expense-reporter/internal/apply"
	"expense-reporter/internal/classifier"
	internalconfig "expense-reporter/internal/config"
	"expense-reporter/internal/feedback"
	"expense-reporter/pkg/utils"
)

var (
	applyYear   int
	applyDryRun bool
)

var applyCmd = &cobra.Command{
	Use:   "apply <reviewed.json>",
	Short: "Apply reviewed entries to the expense log and feedback logs",
	Long: `Ingests the UI's reviewed.json and appends new confirmed/corrected entries
to expenses_log.jsonl, recording feedback in classifications.jsonl. The workbook
is produced separately by generate-workbook.

Pending and skipped entries are ignored. Confirmed and corrected entries already
present in classifications.jsonl receive feedback-only updates (no new append).`,
	Args: cobra.ExactArgs(1),
	RunE: runApply,
}

func init() {
	rootCmd.AddCommand(applyCmd)
	applyCmd.Flags().IntVar(&applyYear, "year", time.Now().Year(), "Year for parsing DD/MM dates")
	applyCmd.Flags().BoolVar(&applyDryRun, "dry-run", false, "Print what would be appended without writing")
}

func runApply(cmd *cobra.Command, args []string) error {
	cfg, err := internalconfig.Load()
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	classifPath := cfg.ClassificationsFilePath()
	if classifPath == "" {
		return fmt.Errorf("classifications log path is not configured")
	}
	expensesLogPath := cfg.ExpensesLogFilePath()

	rf, err := apply.ReadReviewed(args[0])
	if err != nil {
		return fmt.Errorf("reading reviewed file: %w", err)
	}

	// Pre-flight the classifications log BEFORE processing: processEntries reads it
	// (the dedup index) and the corrected branch writes to it. Skipped under
	// --dry-run, which writes nothing.
	if !applyDryRun {
		if err := ensureLogWritable(classifPath, true); err != nil {
			return fmt.Errorf("classifications log not writable: %w\n  Hint: ensure %s is writable, or use --dry-run", err, classifPath)
		}
	}

	newRows, corrections, pendingEntries, skippedEntries, err := processEntries(rf.Entries, classifPath, applyDryRun)
	if err != nil {
		return fmt.Errorf("processing entries: %w", err)
	}

	// Pre-flight the expense log only when there are new rows to append, and
	// non-destructively (never create it on probe) — a found-only run appends
	// nothing and must not leave an empty log behind.
	if !applyDryRun && len(newRows) > 0 {
		if expensesLogPath == "" {
			return fmt.Errorf("expense log path not configured\n  Hint: set expenses_log_path in config, or use --dry-run")
		}
		if err := ensureLogWritable(expensesLogPath, false); err != nil {
			return fmt.Errorf("expense log not writable: %w\n  Hint: ensure %s is writable, or use --dry-run", err, expensesLogPath)
		}
	}

	appendedConfirmed, appendedCorrected, failed, appendErr := appendNewRows(newRows, classifPath, expensesLogPath, applyYear, applyDryRun)

	printSummary(cmd.OutOrStdout(), rf.Source, len(rf.Entries), pendingEntries, skippedEntries, appendedConfirmed, appendedCorrected, corrections, failed, applyDryRun)
	if appendErr != nil {
		return appendErr
	}
	return nil
}

// ensureLogWritable verifies a log file can be appended to. With allowCreate it may
// create the file (append mode never truncates), used for the classifications log.
// Without it the probe is NON-DESTRUCTIVE: it ensures the parent dir is writable but
// opens the file only if it already exists — so pre-flighting a log that may never
// be written does not leave an empty file behind.
func ensureLogWritable(path string, allowCreate bool) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	flags := os.O_APPEND | os.O_WRONLY
	if allowCreate {
		flags |= os.O_CREATE
	} else if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil // dir is writable; don't create the file on probe
	}
	f, err := os.OpenFile(path, flags, 0o644)
	if err != nil {
		return err
	}
	return f.Close()
}

func processEntries(entries []apply.ReviewedEntry, classifPath string, dryRun bool) (newRows, corrections, pendingEntries, skippedEntries []apply.ReviewedEntry, err error) {
	for _, entry := range entries {
		switch entry.Action {
		case apply.ActionPending:
			pendingEntries = append(pendingEntries, entry)
		case apply.ActionSkipped:
			skippedEntries = append(skippedEntries, entry)
		case apply.ActionConfirmed, apply.ActionCorrected:
			if hErr := handleActiveEntry(entry, classifPath, dryRun, &newRows, &corrections); hErr != nil {
				return nil, nil, nil, nil, hErr
			}
		}
	}
	return newRows, corrections, pendingEntries, skippedEntries, nil
}

func handleActiveEntry(entry apply.ReviewedEntry, classifPath string, dryRun bool, newRows, corrections *[]apply.ReviewedEntry) error {
	prior, found, err := feedback.FindLatestEntry(classifPath, entry.ID)
	if err != nil {
		return fmt.Errorf("finding prior entry for %q: %w", entry.ID, err)
	}

	if !found {
		*newRows = append(*newRows, entry)
		return nil
	}

	// Already applied — record corrected feedback only when the action is a
	// correction. The write is gated on !dryRun (preview only); the row is still
	// counted in corrections so the summary reflects what would change.
	if entry.Action == apply.ActionCorrected && entry.Reviewed != nil {
		if !dryRun {
			predicted := classifier.Result{
				Subcategory: prior.PredictedSubcategory,
				Category:    prior.PredictedCategory,
				Confidence:  prior.Confidence,
			}
			corrEntry := feedback.NewCorrectedEntry(
				entry.Item, entry.Date, entry.Value,
				predicted, prior.Model,
				entry.Reviewed.Subcategory, entry.Reviewed.Category,
			)
			corrEntry.Type = entry.Reviewed.Type
			if err := feedback.Append(classifPath, corrEntry); err != nil {
				return fmt.Errorf("appending corrected entry: %w", err)
			}
		}
		*corrections = append(*corrections, entry)
	}
	// confirmed+found: no-op (already applied)
	return nil
}

func appendNewRows(newRows []apply.ReviewedEntry, classifPath, expensesLogPath string, year int, dryRun bool) (appendedConfirmed, appendedCorrected int, failed []apply.ReviewedEntry, err error) {
	for _, entry := range newRows {
		if entry.Reviewed == nil {
			failed = append(failed, entry)
			continue
		}

		parsedDate, err := utils.ParseDateWithYear(entry.Date, year)
		if err != nil {
			failed = append(failed, entry)
			continue
		}

		if dryRun {
			if entry.Action == apply.ActionConfirmed {
				appendedConfirmed++
			} else {
				appendedCorrected++
			}
			continue
		}

		err = appender.ExpandAndAppend(expensesLogPath, entry.Item, parsedDate, entry.Value, 1, entry.Reviewed.Type, entry.Reviewed.Category, entry.Reviewed.Subcategory)
		if err != nil {
			failed = append(failed, entry)
			continue
		}

		if entry.Action == apply.ActionConfirmed {
			appendedConfirmed++
		} else {
			appendedCorrected++
		}

		fbEntry, _ := buildFeedbackEntry(entry)
		if err := feedback.Append(classifPath, fbEntry); err != nil {
			fmt.Fprintf(os.Stderr, "  FEEDBACK ERROR %q: %v\n", entry.Item, err)
		}
	}

	if len(failed) > 0 {
		err = fmt.Errorf("%d row(s) failed to append to the expense log", len(failed))
	}

	return appendedConfirmed, appendedCorrected, failed, err
}

func buildFeedbackEntry(entry apply.ReviewedEntry) (feedback.Entry, bool) {
	if entry.Action == apply.ActionConfirmed {
		predicted := classifier.Result{
			Subcategory: entry.Reviewed.Subcategory,
			Category:    entry.Reviewed.Category,
			Confidence:  entry.Confidence,
		}
		fbEntry := feedback.NewConfirmedEntry(entry.Item, entry.Date, entry.Value, predicted, "review")
		fbEntry.Type = entry.Reviewed.Type
		return fbEntry, true
	}
	predicted := classifier.Result{
		Subcategory: entry.Predicted.Subcategory,
		Category:    entry.Predicted.Category,
		Confidence:  entry.Confidence,
	}
	fbEntry := feedback.NewCorrectedEntry(entry.Item, entry.Date, entry.Value, predicted, "review",
		entry.Reviewed.Subcategory, entry.Reviewed.Category)
	fbEntry.Type = entry.Reviewed.Type
	return fbEntry, false
}

func printSummary(w io.Writer, source string, total int, pendingEntries, skippedEntries []apply.ReviewedEntry, appendedConfirmed, appendedCorrected int, corrections, failed []apply.ReviewedEntry, dryRun bool) {
	appended := appendedConfirmed + appendedCorrected
	tag, appendedLabel := "", "Appended:    "
	if dryRun {
		tag, appendedLabel = " (dry-run)", "Would append:"
	}
	fmt.Fprintf(w, "Applied %s (%d entries)%s\n\n", source, total, tag)
	fmt.Fprintf(w, "%s %d rows (%d confirmed, %d corrected)\n", appendedLabel, appended, appendedConfirmed, appendedCorrected)
	fmt.Fprintf(w, "Failed:       %d rows\n", len(failed))
	fmt.Fprintf(w, "Skipped:      %d rows\n", len(skippedEntries))
	fmt.Fprintf(w, "Pending:      %d rows\n", len(pendingEntries))

	if verbose {
		if len(skippedEntries) > 0 {
			fmt.Fprintf(w, "\nSkipped:\n")
			for _, e := range skippedEntries {
				fmt.Fprintf(w, "   %s (%s, R$%.2f)\n", e.Item, e.Date, e.Value)
			}
		}
		if len(pendingEntries) > 0 {
			fmt.Fprintf(w, "\nPending:\n")
			for _, e := range pendingEntries {
				fmt.Fprintf(w, "   %s (%s, R$%.2f)\n", e.Item, e.Date, e.Value)
			}
		}
	}

	if len(failed) > 0 {
		fmt.Fprintf(w, "\n⚠  %d rows could not be appended to the expense log:\n", len(failed))
		for _, u := range failed {
			cat, sub, typ := "?", "?", "?"
			if u.Reviewed != nil {
				cat, sub, typ = u.Reviewed.Category, u.Reviewed.Subcategory, u.Reviewed.Type
			}
			fmt.Fprintf(w, "   %s (%s, R$%.2f) — %s / %s [%s]\n", u.Item, u.Date, u.Value, cat, sub, typ)
		}
	}

	if len(corrections) == 0 {
		return
	}
	fmt.Fprintf(w, "\n⚠  %d already-applied rows were corrected — feedback logged, no expense-log change:\n", len(corrections))
	for _, c := range corrections {
		fmt.Fprintf(w, "   %s (%s, R$%.2f) %s → %s  [logged]\n",
			c.Item, c.Date, c.Value, c.Predicted.Subcategory, c.Reviewed.Subcategory)
	}
}
