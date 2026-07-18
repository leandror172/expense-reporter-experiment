package cmd

import (
	"encoding/csv"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"expense-reporter/internal/appender"
	"expense-reporter/internal/batch"
	"expense-reporter/internal/classifier"
	"expense-reporter/internal/config"
	"expense-reporter/internal/taxonomy"
	"expense-reporter/pkg/utils"

	"github.com/spf13/cobra"
)

var (
	batchAutoModel     string
	batchAutoDataDir   string
	batchAutoOllamaURL string
	batchAutoThreshold float64
	batchAutoTopN      int
	batchAutoDryRun    bool
	batchAutoOutputDir string
)

var batchAutoCmd = &cobra.Command{
	Use:   "batch-auto <csv_file>",
	Short: "Classify a CSV batch and auto-insert rows that pass the agreement gate",
	Long: `Read a 3-field semicolon-delimited CSV (item;DD/MM;value), classify each row,
and auto-insert rows that pass the agreement gate (the model's prediction agrees
with an unambiguous, high-specificity keyword match) into the workbook.

Output files are written to --output-dir (default: same directory as input):
  classified.csv  — all rows with classification results
  review.csv      — rows not auto-inserted (gate not met or excluded)
  rollover.csv    — installment rows whose later months fall into next year (if any)

Use --dry-run to skip workbook insertion and only produce the CSV outputs.

Use --resume for an idempotent re-run after a partial failure: each row's expense-log
entry ids are predicted up front, and any row whose ids are ALL already present in the log
is skipped before the model is called (printed as SKIP … already logged, recorded in
classified.csv, absent from review.csv). A row only PARTIALLY present in the log (e.g. some
installments of a series logged before a mid-run failure) is routed to review for manual
resolution rather than auto-completed, so a divergent re-classification can never split a
series across categories. Year note: bare DD/MM dates infer the current year, so a resume
crossing a year boundary (run started in December, resumed in January) may not match — pass
DD/MM/YYYY inputs for December batches.

Independently of --resume, an always-on warning is printed to stderr whenever an appended
entry's id already exists in the log, flagging a likely duplicate append.

Examples:
  expense-reporter batch-auto expenses.csv
  expense-reporter batch-auto expenses.csv --dry-run --output-dir /tmp/out
  expense-reporter batch-auto expenses.csv --resume   # re-run, skipping already-logged rows`,
	Args: cobra.ExactArgs(1),
	RunE: runBatchAuto,
}

func init() {
	rootCmd.AddCommand(batchAutoCmd)
	batchAutoCmd.Flags().StringVar(&batchAutoModel, "model", "my-classifier-q3", "Ollama model to use")
	batchAutoCmd.Flags().StringVar(&batchAutoDataDir, "data-dir", "data/classification", "Path to classification data directory")
	batchAutoCmd.Flags().StringVar(&batchAutoOllamaURL, "ollama-url", "http://localhost:11434", "Ollama API base URL")
	batchAutoCmd.Flags().Float64Var(&batchAutoThreshold, "threshold", 0.85, "Deprecated: ignored — the auto-insert gate now uses keyword agreement, not confidence")
	batchAutoCmd.Flags().IntVar(&batchAutoTopN, "top", 3, "Number of classification candidates")
	batchAutoCmd.Flags().BoolVar(&batchAutoDryRun, "dry-run", false, "Classify and write CSVs without inserting into workbook")
	batchAutoCmd.Flags().StringVar(&batchAutoOutputDir, "output-dir", "", "Directory for output CSV files (default: same as input file)")
	batchAutoCmd.Flags().BoolVar(&batchAutoThink, "think", true, "Allow the model to emit thinking tokens (false = faster, sends think:false)")
	batchAutoCmd.Flags().BoolVar(&batchAutoResume, "resume", false, "Skip rows already present in the expense log (idempotent re-run after a partial failure). Note: bare DD/MM dates infer the current year, so a resume crossing a year boundary may not match — use DD/MM/YYYY inputs for December batches.")
	// T-32: the agreement gate replaced the confidence threshold; keep the flag
	// accepted (so existing scripts/fixtures don't error) but mark it deprecated.
	_ = batchAutoCmd.Flags().MarkDeprecated("threshold", "ignored — the auto-insert gate now uses keyword agreement, not confidence")
}

var batchAutoThink bool
var batchAutoResume bool

// classifiedRow holds the result of classifying a single input row.
type classifiedRow struct {
	Item         string
	Date         string
	RawValue     string // original value string, preserves installment notation (e.g. "99,90/3")
	Subcategory  string
	Category     string
	Confidence   float64
	AutoInserted bool
	Type         string // resolved expense type name (empty if not found or ambiguous)
	// Skipped marks a row that --resume matched entirely against the pre-existing expense
	// log and therefore did NOT classify or append. Skipped rows land in classified.csv
	// (with skippedMarker in the subcategory column) but never in review.csv, and are
	// counted separately in the summary. Distinct from an error and from a review row.
	Skipped bool
	Error   error
}

func runBatchAuto(cmd *cobra.Command, args []string) error {
	csvPath := args[0]

	outputDir, err := resolveOutputDir(csvPath, batchAutoOutputDir)
	if err != nil {
		return err
	}

	lines, err := loadInputLines(csvPath)
	if err != nil {
		return err
	}

	sheets, appCfg, err := loadBatchAutoDeps()
	if err != nil {
		return err
	}

	cfg := classifier.Config{
		OllamaURL:    batchAutoOllamaURL,
		Model:        batchAutoModel,
		DataDir:      batchAutoDataDir,
		FeedbackPath: appCfg.ClassificationsFilePath(),
		TopN:         batchAutoTopN,
		NoThink:      !batchAutoThink,
	}

	// Log-append pivot: the expense log is now the only durable persistence, so
	// fail fast if it is unwritable before spending ~12 s/row on the model.
	if !batchAutoDryRun {
		if err := preflightLogPath(appCfg); err != nil {
			return err
		}
	}

	// One shared ID-ledger threaded through the classify phase (full-skip consumption under
	// --resume) and the append phase (per-entry duplicate warnings). Loaded whenever the log
	// exists, not only under --resume. Consumption order (invariant): classify-phase full
	// skips consume first, then append-phase appends/warnings; partial rows consume nothing.
	ledger, err := loadResumeLedger(appCfg)
	if err != nil {
		return fmt.Errorf("loading expense log ledger: %w", err)
	}

	results := classifyLines(lines, sheets, appCfg, cfg, ledger, batchAutoResume)

	var appendErr error
	if !batchAutoDryRun {
		appendErr = appendClassified(results, appCfg, batchAutoModel, ledger)
	}

	classifiedPath := filepath.Join(outputDir, "classified.csv")
	reviewPath := filepath.Join(outputDir, "review.csv")

	// CSVs are written AFTER appendClassified so they reflect any rows it
	// downgraded on append failure (a failed row lands in review.csv, not as a
	// false "appended").
	if err := writeClassifiedCSV(classifiedPath, results); err != nil {
		return fmt.Errorf("writing classified.csv: %w", err)
	}
	if err := writeReviewCSV(reviewPath, results); err != nil {
		return fmt.Errorf("writing review.csv: %w", err)
	}

	printBatchSummary(results, batchAutoDryRun, classifiedPath, reviewPath)
	if appendErr != nil {
		return fmt.Errorf("log append failed (classification CSVs preserved at %s): %w", outputDir, appendErr)
	}
	return nil
}

func resolveOutputDir(csvPath, outputDir string) (string, error) {
	if outputDir == "" {
		outputDir = filepath.Dir(csvPath)
	}
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return "", fmt.Errorf("creating output dir: %w", err)
	}
	return outputDir, nil
}

func loadInputLines(csvPath string) ([]string, error) {
	lines, err := batch.NewCSVReader(csvPath).Read()
	if err != nil {
		return nil, fmt.Errorf("reading CSV: %w", err)
	}
	if len(lines) == 0 {
		return nil, fmt.Errorf("input CSV is empty")
	}
	return lines, nil
}

func loadBatchAutoDeps() ([]taxonomy.ExpenseType, *config.Config, error) {
	appCfg, err := config.Load()
	if err != nil {
		return nil, nil, fmt.Errorf("loading config: %w", err)
	}
	sheets, err := loadTaxonomyTree(appCfg)
	if err != nil {
		return nil, nil, err
	}
	return sheets, appCfg, nil
}

func classifyLines(lines []string, sheets []taxonomy.ExpenseType, appCfg *config.Config, cfg classifier.Config, ledger map[string]int, resume bool) []classifiedRow {
	total := len(lines)
	results := make([]classifiedRow, 0, total)

	// Load the keyword index once for the whole batch. On failure, proceed with a
	// nil index: MatchStrength treats every row as a keyword miss → REVIEW, rather
	// than aborting the run.
	keywords, err := classifier.LoadKeywordIndex(cfg.DataDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "⚠  keyword index unavailable (%v); all rows route to REVIEW\n", err)
		keywords = nil
	}

	for i, line := range lines {
		row, err := parse3FieldLine(line)
		if err != nil {
			fmt.Fprintf(os.Stderr, "[%d/%d] SKIP  %q: %v\n", i+1, total, line, err)
			results = append(results, classifiedRow{Item: line, Error: err})
			continue
		}

		// --resume: parse date/value up front, predict this row's entry ids, and consult the
		// ledger BEFORE the model call. A full match skips (consuming the ledger); a parse
		// failure routes to an error row (never a skip); a partial match forces review below.
		partial := false
		if resume {
			handled, forceReview := applyResumeDecision(ledger, row, i, total, &results)
			if handled {
				continue
			}
			partial = forceReview
		}

		classResults, err := classifier.Classify(row.Item, row.Value, row.Date, sheets, cfg)
		if err != nil || len(classResults) == 0 {
			fmt.Fprintf(os.Stderr, "[%d/%d] REVIEW %q: classifier error: %v\n", i+1, total, row.Item, err)
			results = append(results, classifiedRow{Item: row.Item, Date: row.Date, RawValue: row.RawValue, Error: err})
			continue
		}

		top := classResults[0]
		signal := classifier.MatchStrength(row.Item, keywords)
		autoInsert := classifier.IsAutoInsertable(top, signal, appCfg.AutoInsertExcluded)
		if partial {
			// A partially-logged series must be resolved by hand, never auto-completed:
			// a divergent re-classification would split the series across categories.
			autoInsert = false
			fmt.Fprintf(os.Stderr, "[%d/%d] REVIEW %q: partially logged — resolve manually\n", i+1, total, row.Item)
		}
		status := "REVIEW"
		if autoInsert {
			status = "AUTO  "
		}
		fmt.Printf("[%d/%d] %s %s → %s (%.0f%%)\n", i+1, total, status, row.Item, top.Subcategory, top.Confidence*100)

		results = append(results, classifiedRow{
			Item:         row.Item,
			Date:         row.Date,
			RawValue:     row.RawValue,
			Subcategory:  top.Subcategory,
			Category:     top.Category,
			Confidence:   top.Confidence,
			AutoInserted: autoInsert,
			Type:         top.Type, // T-13: type comes from the predicted full path
		})
	}
	return results
}

// applyResumeDecision runs the --resume pre-check for one row and records any terminal outcome
// directly into results. It returns handled=true when the row is fully resolved here (skipped
// or routed to an error row) so the caller must `continue`; when handled=false the row proceeds
// to classification, and forceReview=true means it is partially logged and must go to review.
func applyResumeDecision(ledger map[string]int, row inputRow, i, total int, results *[]classifiedRow) (handled, forceReview bool) {
	outcome, err := classifyResumeDecision(ledger, row)
	switch outcome {
	case resumeParseErr:
		fmt.Fprintf(os.Stderr, "[%d/%d] ERROR  %q: %v\n", i+1, total, row.Item, err)
		*results = append(*results, classifiedRow{Item: row.Item, Date: row.Date, RawValue: row.RawValue, Error: err})
		return true, false
	case resumeSkipFull:
		fmt.Printf("[%d/%d] SKIP  %s (already logged)\n", i+1, total, row.Item)
		*results = append(*results, classifiedRow{Item: row.Item, Date: row.Date, RawValue: row.RawValue, Subcategory: skippedMarker, Skipped: true})
		return true, false
	case resumePartial:
		return false, true
	default:
		return false, false
	}
}

// preflightLogPath fails fast when the expense log is unwritable, before the
// batch spends ~12 s/row on the model. Because the log is now the only durable
// persistence, an unwritable path must abort the run rather than silently lose
// every auto-classified row.
func preflightLogPath(appCfg *config.Config) error {
	logPath := appCfg.ExpensesLogFilePath()
	if logPath == "" {
		return fmt.Errorf("expense log path not configured\n  Hint: set expenses_log_path in config, or use --dry-run")
	}
	if err := verifyAppendable(logPath); err != nil {
		return fmt.Errorf("expense log not writable: %w\n  Hint: ensure %s is writable, or use --dry-run", err, logPath)
	}
	return nil
}

// verifyAppendable confirms the log can actually be opened for append — the same
// mode feedback.AppendExpense uses — so a read-only existing file is caught, not
// just a missing directory.
func verifyAppendable(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	return f.Close()
}

// appendClassified appends every auto-classified row to the expense log. Because
// the log is the only durable persistence, a per-row failure (value/date parse
// or append error) downgrades that row in place — AutoInserted=false + Error set —
// so the summary count stays honest, the row falls into review.csv, and the
// command exits non-zero (the returned error is wrapped by the caller).
func appendClassified(results []classifiedRow, appCfg *config.Config, model string, ledger map[string]int) error {
	logPath := appCfg.ExpensesLogFilePath()
	var failCount int
	for idx := range results {
		r := results[idx]
		if !r.AutoInserted || r.Error != nil {
			continue
		}
		if err := appendOneRow(logPath, r, ledger); err != nil {
			results[idx].AutoInserted = false
			results[idx].Error = err
			fmt.Fprintf(os.Stderr, "  APPEND ERROR %q: %v\n", r.Item, err)
			failCount++
			continue
		}
		logConfirmedFeedbackForRow(appCfg, r, model)
	}
	if failCount > 0 {
		return fmt.Errorf("%d row(s) failed to append to the expense log", failCount)
	}
	return nil
}

// appendOneRow expands installments and appends a single classified row to the
// expense log. Returns an error if the value/date cannot be parsed or the append
// fails — any of which means the row was not persisted. Before appending, it emits
// the always-on duplicate warning for any entry id already present in the ledger
// (consuming that count), so a re-append over a pre-existing log line is flagged.
func appendOneRow(logPath string, r classifiedRow, ledger map[string]int) error {
	perInstallment, installmentCount, err := utils.ParseCurrencyWithInstallments(r.RawValue)
	if err != nil {
		return fmt.Errorf("parsing value %q: %w", r.RawValue, err)
	}
	parsedDate, err := utils.ParseDateFlexible(r.Date)
	if err != nil {
		return fmt.Errorf("parsing date %q: %w", r.Date, err)
	}
	warnDuplicateEntries(ledger, r.Item, appender.PredictEntryIDs(r.Item, parsedDate, perInstallment, installmentCount))
	return appender.ExpandAndAppend(logPath, r.Item, parsedDate, perInstallment, installmentCount, r.Type, r.Category, r.Subcategory)
}

// logConfirmedFeedbackForRow records the confirmed classification to
// classifications.jsonl for a successfully appended row. Secondary to the expense
// log: a failure here is non-fatal (logConfirmedFeedback warns internally).
func logConfirmedFeedbackForRow(appCfg *config.Config, r classifiedRow, model string) {
	perInstallment, _, err := utils.ParseCurrencyWithInstallments(r.RawValue)
	if err != nil {
		return
	}
	predicted := classifier.Result{
		Type:        r.Type,
		Category:    r.Category,
		Subcategory: r.Subcategory,
		Confidence:  r.Confidence,
	}
	logConfirmedFeedback(appCfg, r.Item, r.Date, perInstallment, predicted, model)
}

func printBatchSummary(results []classifiedRow, dryRun bool, classifiedPath, reviewPath string) {
	autoCount, reviewCount, errorCount, skippedCount := 0, 0, 0, 0
	for _, r := range results {
		switch {
		case r.Error != nil:
			errorCount++
		case r.Skipped:
			skippedCount++
		case r.AutoInserted:
			autoCount++
		default:
			reviewCount++
		}
	}
	// Dry-run appends nothing, so the count is what *would* be appended.
	dryTag, appendLine := "", "  Auto-appended : %d\n"
	if dryRun {
		dryTag, appendLine = " (dry-run)", "  Would append  : %d\n"
	}
	fmt.Printf("\n--- Summary%s ---\n", dryTag)
	fmt.Printf(appendLine, autoCount)
	fmt.Printf("  Skipped       : %d\n", skippedCount)
	fmt.Printf("  For review    : %d\n", reviewCount)
	fmt.Printf("  Errors        : %d\n", errorCount)
	fmt.Printf("  classified.csv: %s\n", classifiedPath)
	fmt.Printf("  review.csv    : %s\n", reviewPath)
}

// inputRow is a parsed 3-field line.
type inputRow struct {
	Item     string
	Date     string
	Value    float64 // per-installment value, used for classifier display
	RawValue string  // original string, preserves installment notation (e.g. "99,90/3")
}

// parse3FieldLine splits "item;DD/MM;value" and parses currency.
// value may include installment notation (e.g. "99,90/3"); RawValue preserves it.
func parse3FieldLine(line string) (inputRow, error) {
	parts := strings.SplitN(line, ";", 3)
	if len(parts) != 3 {
		return inputRow{}, fmt.Errorf("expected 3 fields (item;DD/MM;value), got %d", len(parts))
	}
	item := strings.TrimSpace(parts[0])
	date := strings.TrimSpace(parts[1])
	valueStr := strings.TrimSpace(parts[2])
	if item == "" {
		return inputRow{}, fmt.Errorf("empty item field")
	}
	perInstallment, _, err := utils.ParseCurrencyWithInstallments(valueStr)
	if err != nil {
		return inputRow{}, fmt.Errorf("parsing value %q: %w", valueStr, err)
	}
	return inputRow{Item: item, Date: date, Value: perInstallment, RawValue: valueStr}, nil
}

// writeClassifiedCSV writes all classified rows to path.
// Format: item;date;value;subcategory;category;confidence;auto_inserted;type
func writeClassifiedCSV(path string, rows []classifiedRow) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	w := csv.NewWriter(f)
	w.Comma = ';'
	if err := w.Write([]string{"item", "date", "value", "subcategory", "category", "confidence", "auto_inserted", "type"}); err != nil {
		return err
	}
	for _, r := range rows {
		w.Write([]string{ //nolint:errcheck
			r.Item,
			r.Date,
			r.RawValue,
			r.Subcategory,
			r.Category,
			fmt.Sprintf("%.4f", r.Confidence),
			fmt.Sprintf("%v", r.AutoInserted),
			r.Type,
		})
	}
	w.Flush()
	return w.Error()
}

// writeReviewCSV writes only rows where auto_inserted == false.
// Format: item;date;value;subcategory;category;confidence;auto_inserted;type
func writeReviewCSV(path string, rows []classifiedRow) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	w := csv.NewWriter(f)
	w.Comma = ';'
	if err := w.Write([]string{"item", "date", "value", "subcategory", "category", "confidence", "auto_inserted", "type"}); err != nil {
		return err
	}
	for _, r := range rows {
		// Auto-inserted rows are already in the log; skipped rows (--resume matched them
		// against the pre-existing log) are recorded only in classified.csv. Neither belongs
		// in the review queue.
		if r.AutoInserted || r.Skipped {
			continue
		}
		w.Write([]string{ //nolint:errcheck
			r.Item,
			r.Date,
			r.RawValue,
			r.Subcategory,
			r.Category,
			fmt.Sprintf("%.4f", r.Confidence),
			"false",
			r.Type,
		})
	}
	w.Flush()
	return w.Error()
}
