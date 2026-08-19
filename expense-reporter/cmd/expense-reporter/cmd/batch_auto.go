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
	"expense-reporter/internal/parse"
	"expense-reporter/internal/taxonomy"

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

Use --dry-run to skip workbook insertion and only produce the CSV outputs.

Bare DD/MM dates resolve their year through the parse boundary's precedence ladder:
a year written into the date wins, then --year, then config date_year, then the most
recent occurrence that is not in the future.

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
	batchAutoCmd.Flags().BoolVar(&batchAutoThink, "think", false, "Allow the model to emit thinking tokens (~10x slower for a marginal accuracy gain)")
	batchAutoCmd.Flags().BoolVar(&batchAutoResume, "resume", false, "Skip rows already present in the expense log (idempotent re-run after a partial failure). Bare DD/MM dates take their year from the precedence ladder, so a resume must resolve the same year as the original run — pass --year (or set config date_year) when re-running a batch across a year boundary.")
	batchAutoCmd.Flags().IntVar(&batchAutoYear, "year", 0, "Fallback year for bare DD/MM dates (outranks config date_year; an explicit year in the date always wins)")
	// T-32: the agreement gate replaced the confidence threshold; keep the flag
	// accepted (so existing scripts/fixtures don't error) but mark it deprecated.
	_ = batchAutoCmd.Flags().MarkDeprecated("threshold", "ignored — the auto-insert gate now uses keyword agreement, not confidence")
}

var batchAutoThink bool
var batchAutoResume bool
var batchAutoYear int

// classifiedRow holds the result of classifying a single input row.
type classifiedRow struct {
	// Expense is the row's input, parsed ONCE at the top of the loop. Everything
	// downstream — the resume prediction, the append, the feedback write, the CSVs —
	// reads it instead of re-parsing the line, which is what let those consumers
	// disagree about a row's date and split one expense across two join ids (T-35).
	//
	// A row whose parse failed carries the zero value. That is detectable without a
	// second flag: YearSource is YearSourceUnknown, which the boundary defines as
	// "not produced by a successful parse".
	Expense parse.ParsedExpense
	// RawLine is the original CSV line, kept only so a row that never parsed can still
	// be named in the output — it is the sole identity such a row has.
	RawLine      string
	Subcategory  string
	Category     string
	Confidence   float64
	AutoInserted bool
	Type         string // resolved expense type name (empty if not found or ambiguous)
	// KeywordHint is the keyword layer's competing suggestion, non-empty ONLY when it
	// unambiguously disagrees with the model (classifier.KeywordHint). It is advisory: it
	// rides the CSVs so the review page can show the reviewer a second opinion, and
	// nothing downstream may act on it. Empty on most rows, and empty is the signal for
	// "no second opinion" — never a placeholder.
	KeywordHint string
	// Skipped marks a row that --resume matched entirely against the pre-existing expense
	// log and therefore did NOT classify or append. Skipped rows land in classified.csv
	// (with skippedMarker in the subcategory column) but never in review.csv, and are
	// counted separately in the summary. Distinct from an error and from a review row.
	Skipped bool
	Error   error
}

// displayItem names the row in output. A row that never parsed has no item, so it
// falls back to the raw line — which is also what the CSV's item column has always
// carried for such rows.
func (r classifiedRow) displayItem() string {
	if r.Expense.Item != "" {
		return r.Expense.Item
	}
	return r.RawLine
}

// dateCell renders the row's date for the CSV writers, canonical DD/MM/YYYY, or empty
// for a row that never parsed.
//
// Canonical rather than the raw input is deliberate. review.ReadQueue hashes this exact
// column into the id it puts in reviewed.json, while both JSONL logs hash the canonical
// form — so emitting the raw "15/04" here handed the review queue an id that apply would
// never itself write, and an id miss in apply is silent (it treats the entry as new and
// appends it). Writing the canonical date is the same one-boundary normalization T-35
// applied to the two log writers, extended to their third reader.
//
// The empty case is read off YearSourceUnknown rather than a separate "parsed" flag,
// because that is already the boundary's own marker for "not produced by a successful
// parse" — one fact with one home. Without the guard a failed row would render its zero
// time as 01/01/0001.
func (r classifiedRow) dateCell() string {
	if r.Expense.YearSource == parse.YearSourceUnknown {
		return ""
	}
	return r.Expense.DateString()
}

// valueCell renders the row's value for the CSV writers.
//
// It MUST be the raw token, never the parsed per-installment float: "99,90/3" is how the
// installment count survives into review.csv, and review.ReadQueue re-reads this column.
// Writing the float would erase the count from the review queue silently — no error, just
// three installments quietly becoming one (T-21).
func (r classifiedRow) valueCell() string {
	return r.Expense.RawValue
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

	results := classifyLines(lines, sheets, appCfg, cfg, ledger, batchAutoResume, parseOptions(batchAutoYear, appCfg))

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
	// failed.csv is written only when something was rejected, so its mere existence
	// is the signal that this run has rows needing a human. Rejected rows reach no
	// other durable artifact — classified.csv keeps the raw line but review.html
	// cannot render a row with no date (T-56).
	failedPath := filepath.Join(outputDir, "failed.csv")
	if err := batch.WriteFailedRows(failedPath, rejectedRows(results)); err != nil {
		return fmt.Errorf("writing failed.csv: %w", err)
	}

	printBatchSummary(results, batchAutoDryRun, classifiedPath, reviewPath, failedPath, appCfg)
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

func classifyLines(lines []string, sheets []taxonomy.ExpenseType, appCfg *config.Config, cfg classifier.Config, ledger map[string]int, resume bool, opts parse.Options) []classifiedRow {
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
		pe, err := parse3FieldLine(line, opts)
		if err != nil {
			// describeParseFailure rather than a second phrasing of the same failure:
			// the row number is already this line's prefix, and the helper is what
			// knows the accepted value spellings.
			err = describeParseFailure(err)
			fmt.Fprintf(os.Stderr, "[%d/%d] SKIP  %q: %v\n", i+1, total, line, err)
			results = append(results, classifiedRow{RawLine: line, Error: err})
			continue
		}

		// --resume: predict this row's entry ids from the already-parsed expense and
		// consult the ledger BEFORE the model call. A full match skips (consuming the
		// ledger); a partial match forces review below.
		partial := false
		if resume {
			handled, forceReview := applyResumeDecision(ledger, pe, i, total, &results)
			if handled {
				continue
			}
			partial = forceReview
		}

		classResults, err := classifier.Classify(pe.Item, pe.Value, pe.DateString(), sheets, cfg)
		if err != nil || len(classResults) == 0 {
			fmt.Fprintf(os.Stderr, "[%d/%d] REVIEW %q: classifier error: %v\n", i+1, total, pe.Item, err)
			results = append(results, classifiedRow{Expense: pe, Error: err})
			continue
		}

		top := classResults[0]
		signal := classifier.MatchStrength(pe.Item, keywords)
		autoInsert := classifier.IsAutoInsertable(top, signal, appCfg.AutoInsertExcluded)
		if partial {
			// A partially-logged series must be resolved by hand, never auto-completed:
			// a divergent re-classification would split the series across categories.
			autoInsert = false
			fmt.Fprintf(os.Stderr, "[%d/%d] REVIEW %q: partially logged — resolve manually\n", i+1, total, pe.Item)
		}
		status := "REVIEW"
		if autoInsert {
			status = "AUTO  "
		}
		fmt.Printf("[%d/%d] %s %s → %s (%.0f%%)\n", i+1, total, status, pe.Item, top.Subcategory, top.Confidence*100)

		results = append(results, classifiedRowFromPrediction(pe, top, signal, autoInsert))
	}
	return results
}

// classifiedRowFromPrediction assembles the row for one successfully classified expense.
//
// Extracted so the field wiring is reachable without a model. Every other path into this
// struct is behind classifier.Classify, so a swapped argument here — passing the keyword's
// own subcategory as the model's, say — would compile, silently disable the hint for every
// row, and stay invisible until someone read a month of output. It is a pure function of
// what the classifier returned, so a unit test pins it directly.
func classifiedRowFromPrediction(pe parse.ParsedExpense, top classifier.Result, signal classifier.MatchSignal, autoInsert bool) classifiedRow {
	return classifiedRow{
		Expense:      pe,
		Subcategory:  top.Subcategory,
		Category:     top.Category,
		Confidence:   top.Confidence,
		AutoInserted: autoInsert,
		Type:         top.Type, // T-13: type comes from the predicted full path
		// The same signal the gate just consulted, read the other way round: the gate fires
		// on agreement, the hint on disagreement. Both take the MODEL's subcategory as the
		// thing being agreed or disagreed with.
		KeywordHint: classifier.KeywordHint(signal, top.Subcategory),
	}
}

// applyResumeDecision runs the --resume pre-check for one row and records any terminal outcome
// directly into results. It returns handled=true when the row is fully resolved here (skipped)
// so the caller must `continue`; when handled=false the row proceeds to classification, and
// forceReview=true means it is partially logged and must go to review.
//
// There is no parse-failure branch any more: the line is parsed once before this is reached,
// so a row that could not be parsed never arrives here.
func applyResumeDecision(ledger map[string]int, pe parse.ParsedExpense, i, total int, results *[]classifiedRow) (handled, forceReview bool) {
	switch classifyResumeDecision(ledger, pe) {
	case resumeSkipFull:
		fmt.Printf("[%d/%d] SKIP  %s (already logged)\n", i+1, total, pe.Item)
		*results = append(*results, classifiedRow{Expense: pe, Subcategory: skippedMarker, Skipped: true})
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
			fmt.Fprintf(os.Stderr, "  APPEND ERROR %q: %v\n", r.displayItem(), err)
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
// expense log. Returns an error if the append fails, which means the row was not
// persisted. Before appending, it emits the always-on duplicate warning for any entry
// id already present in the ledger (consuming that count), so a re-append over a
// pre-existing log line is flagged.
//
// It no longer re-parses anything: the row carries the expense parsed at read time, so
// the ids predicted here and the entries written below are derived from one value.
func appendOneRow(logPath string, r classifiedRow, ledger map[string]int) error {
	pe := r.Expense
	warnDuplicateEntries(ledger, pe.Item, appender.PredictEntryIDs(pe.Item, pe.Date, pe.Value, pe.Installments))
	return appender.ExpandAndAppend(logPath, pe.Item, pe.Date, pe.Value, pe.Installments, r.Type, r.Category, r.Subcategory)
}

// logConfirmedFeedbackForRow records the confirmed classification to
// classifications.jsonl for a successfully appended row. Secondary to the expense
// log: a failure here is non-fatal (logConfirmedFeedback warns internally).
//
// The two logs join on a hash of the date string, and this function used to re-parse
// and re-format the row's raw date to match what the expense log had written (T-35).
// That canonicalization is gone because the divergence it repaired is now
// unrepresentable: both writers read the same ParsedExpense, and DateString() is the
// single place those identity bytes are produced.
func logConfirmedFeedbackForRow(appCfg *config.Config, r classifiedRow, model string) {
	predicted := classifier.Result{
		Type:        r.Type,
		Category:    r.Category,
		Subcategory: r.Subcategory,
		Confidence:  r.Confidence,
	}
	logConfirmedFeedback(appCfg, r.Expense.Item, r.Expense.DateString(), r.Expense.Value, predicted, model)
}

func printBatchSummary(results []classifiedRow, dryRun bool, classifiedPath, reviewPath, failedPath string, appCfg *config.Config) {
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
	// Named only when it exists: pointing at a file that was deliberately not written
	// would read as an empty reject list rather than as no rejects at all.
	if errorCount > 0 {
		fmt.Printf("  failed.csv    : %s  <- fix these lines and re-run this file\n", failedPath)
	}
	warnIfStaleConfiguredYearInBatch(results, appCfg)
}

// rejectedRows projects the rows the parse boundary refused, in input order, into the
// shape failed.csv is written from. The raw line is carried through untouched: it is
// the text the human typed and will edit, so re-serializing it would hand them back
// something subtly different from what they wrote.
func rejectedRows(rows []classifiedRow) []batch.FailedRow {
	var rejected []batch.FailedRow
	for _, row := range rows {
		if row.Error == nil {
			continue
		}
		rejected = append(rejected, batch.FailedRow{
			// Stripped, not raw: the reason is metadata regenerated on every run, so
			// carrying the previous one through would append a second copy each time
			// an unrepaired file is re-run, and the line would grow without bound.
			// Uses the SAME strip as the re-import rather than a second rule free to
			// drift from it.
			OriginalLine: stripTrailingComment(row.RawLine),
			Reason:       row.Error.Error(),
		})
	}
	return rejected
}

// parse3FieldLine splits the CSV's "item;DD/MM;value" form and hands the three fields
// to the parse boundary, which owns every rule about what they may contain.
//
// The split stays here rather than moving into the boundary because this is a different
// INPUT FORMAT from the 4-field semicolon CLI form, not a drifted copy of it. Merging
// them into one 3-or-4-field function would cost error locality: `add "Item;15/04;35,50"`
// would stop failing at the parse with a field-count message and instead fail later,
// inside taxonomy resolution, with a message about something else entirely.
func parse3FieldLine(line string, opts parse.Options) (parse.ParsedExpense, error) {
	parts := strings.SplitN(stripTrailingComment(line), ";", 3)
	if len(parts) != 3 {
		return parse.ParsedExpense{}, fmt.Errorf("expected 3 fields (item;DD/MM;value), got %d", len(parts))
	}
	return parse.Fields(parts[0], parts[1], parts[2], opts)
}

// stripTrailingComment drops a trailing "# ..." note so failed.csv round-trips: the
// rejection reason is written onto the row it explains, and the human re-runs that
// same file once the data is repaired.
//
// A '#' only opens a comment when it is whitespace-preceded AND the three data fields
// are already complete before it. Whitespace alone is not enough — "Mesa #5;15/04;35,50"
// is an item containing a hash, and stripping there would leave one field. Requiring the
// separators first also keeps the format from widening: a genuine 4-field line (the
// add/correct form "item;date;value;subcategory") still fails loudly rather than being
// truncated to three, so batch-auto can never silently discard a subcategory.
func stripTrailingComment(line string) string {
	for i := 1; i < len(line); i++ {
		if line[i] != '#' || !isSpaceOrTab(line[i-1]) {
			continue
		}
		if strings.Count(line[:i], ";") >= 2 {
			return strings.TrimRight(line[:i], " \t")
		}
	}
	return line
}

func isSpaceOrTab(b byte) bool { return b == ' ' || b == '\t' }

// writeClassifiedCSV writes all classified rows to path.
// Format: item;date;value;subcategory;category;confidence;auto_inserted;type;keyword_hint
func writeClassifiedCSV(path string, rows []classifiedRow) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	w := csv.NewWriter(f)
	w.Comma = ';'
	if err := w.Write([]string{"item", "date", "value", "subcategory", "category", "confidence", "auto_inserted", "type", "keyword_hint"}); err != nil {
		return err
	}
	for _, r := range rows {
		w.Write([]string{ //nolint:errcheck
			r.displayItem(),
			r.dateCell(),
			r.valueCell(),
			r.Subcategory,
			r.Category,
			fmt.Sprintf("%.4f", r.Confidence),
			fmt.Sprintf("%v", r.AutoInserted),
			r.Type,
			r.KeywordHint,
		})
	}
	w.Flush()
	return w.Error()
}

// writeReviewCSV writes only rows where auto_inserted == false.
// Format: item;date;value;subcategory;category;confidence;auto_inserted;type;keyword_hint
func writeReviewCSV(path string, rows []classifiedRow) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	w := csv.NewWriter(f)
	w.Comma = ';'
	if err := w.Write([]string{"item", "date", "value", "subcategory", "category", "confidence", "auto_inserted", "type", "keyword_hint"}); err != nil {
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
			r.displayItem(),
			r.dateCell(),
			r.valueCell(),
			r.Subcategory,
			r.Category,
			fmt.Sprintf("%.4f", r.Confidence),
			"false",
			r.Type,
			r.KeywordHint,
		})
	}
	w.Flush()
	return w.Error()
}
