//go:build acceptance

package acceptance_test

import (
	"path/filepath"
	"slices"
	"testing"
	"time"

	"expense-reporter/internal/appender"
	"expense-reporter/test/actions"
	"expense-reporter/test/expect"
	"expense-reporter/test/extern"

	"github.com/leandror172/acceptance-harness/harness"
	"github.com/leandror172/acceptance-harness/verify"
)

// TestBatchAutoResume_AllRowsAlreadyLogged is the deterministic core of --resume idempotency:
// every input row (a plain expense AND a full 3-installment series) is pre-seeded into the
// expense log, so a re-run must skip all of them BEFORE any classification. No Ollama is
// contacted because the skip decision happens before the model call. This exercises the
// highest-risk id path: the installment (i/N) suffix + per-month dates + per-installment value
// must predict the exact ids ExpandAndAppend wrote. The seed is built via the REAL append path
// (appender.ExpandAndAppend), so any normalization drift between prediction and append fails
// loudly here rather than silently skipping nothing.
//
// Asserts: command succeeds; both rows reported as already-logged; the log is byte-for-byte
// unchanged (still exactly the 4 seeded entries — a wrongful re-append would double the count);
// classifications.jsonl is never created.
func TestBatchAutoResume_AllRowsAlreadyLogged(t *testing.T) {
	fixDir := filepath.Join(fixturesDir(), "batch-auto-resume-all-seeded")

	harness.Run(t, harness.Scenario{
		Name:    "batch-auto --resume skips every pre-logged row (plain + installment series) before classifying",
		Fixture: fixDir,
		Given:   allInputRowsAlreadyLogged(),
		When:    actions.RunBatchAutoWithFixture(fixDir),
		Then: slices.Concat(
			commandSucceeded(),
			bothRowsReportedAsAlreadyLogged(),
			logUnchangedByResumeSkip(fixDir),
			noClassificationsLogWritten(),
		),
	})
}

// TestBatchAutoResume_OneOfTwoDuplicatesSkipped proves count-consumption: two identical
// legitimate rows with the log seeded with exactly ONE copy. --resume must skip exactly one
// (consuming the single seeded count) and let the other proceed to classification. To keep the
// outcome deterministic, the item is the gate-FAILING "Uber Centro", so the non-skipped row
// deterministically routes to review (never appends) — what matters is skip-count 1, not the
// append. Ollama is required only to classify the second row.
//
// Asserts: command succeeds; exactly ONE row skipped; review.csv has exactly the one
// non-skipped row (the skipped row is excluded from review.csv); the log is unchanged (the
// non-skipped row gate-failed, so nothing appended).
func TestBatchAutoResume_OneOfTwoDuplicatesSkipped(t *testing.T) {
	extern.RequireOllama(t, "")

	fixDir := filepath.Join(fixturesDir(), "batch-auto-resume-one-of-two")

	harness.Run(t, harness.Scenario{
		Name:    "batch-auto --resume skips exactly one of two identical rows when the log holds one copy",
		Fixture: fixDir,
		Given:   oneOfTwoDuplicatesAlreadyLogged(),
		When:    actions.RunBatchAutoWithFixture(fixDir),
		Then: slices.Concat(
			commandSucceeded(),
			exactlyOneRowSkipped(),
			onlyTheNonSkippedRowRoutedToReview(),
			logUnchangedByResumeSkip(fixDir),
		),
	})
}

// TestBatchAutoResume_OnlyNewRowClassifies pairs a pre-logged row with a brand-new
// gate-passing row under --resume: only the new row classifies and appends; the log ends as
// seed + 1 with no duplicate of the seeded row. Ollama-gated (the new row must classify).
func TestBatchAutoResume_OnlyNewRowClassifies(t *testing.T) {
	extern.RequireOllama(t, "")

	fixDir := filepath.Join(fixturesDir(), "batch-auto-resume-one-new")

	harness.Run(t, harness.Scenario{
		Name:    "batch-auto --resume classifies only the new row and appends it once, skipping the pre-logged row",
		Fixture: fixDir,
		Given:   oneRowAlreadyLoggedOtherIsNew(),
		When:    actions.RunBatchAutoWithFixture(fixDir),
		Then: slices.Concat(
			commandSucceeded(),
			newRowAppendedSeededRowSkipped(fixDir),
		),
	})
}

// TestBatchAutoDuplicateWarning_FiresWithoutResume proves the always-on duplicate warning:
// WITHOUT --resume, a row whose id already exists in the log appends AGAIN (append-only default
// preserved) and emits exactly one stderr warning. Ollama-gated (the row must gate-pass to
// reach the append path).
func TestBatchAutoDuplicateWarning_FiresWithoutResume(t *testing.T) {
	extern.RequireOllama(t, "")

	fixDir := filepath.Join(fixturesDir(), "batch-auto-dup-warning")

	harness.Run(t, harness.Scenario{
		Name:    "batch-auto without --resume re-appends a pre-logged row and warns exactly once",
		Fixture: fixDir,
		Given:   expenseAlreadyLoggedThenReappended(),
		When:    actions.RunBatchAutoWithFixture(fixDir),
		Then: slices.Concat(
			commandSucceeded(),
			rowAppendedAgainWithSingleDuplicateWarning(fixDir),
		),
	})
}

// TestBatchAutoResume_DryRunShowsSkipsWritesNoLog composes --dry-run with --resume: the skip
// decision is still shown, and no log is written. Deterministic (the row is pre-seeded, so it
// skips before any model call).
func TestBatchAutoResume_DryRunShowsSkipsWritesNoLog(t *testing.T) {
	fixDir := filepath.Join(fixturesDir(), "batch-auto-resume-dryrun")

	harness.Run(t, harness.Scenario{
		Name:    "batch-auto --dry-run --resume shows the skips and writes nothing to the log",
		Fixture: fixDir,
		Given:   allInputRowsAlreadyLogged(),
		When:    actions.RunBatchAutoWithFixture(fixDir),
		Then: slices.Concat(
			commandSucceeded(),
			bothRowsReportedAsAlreadyLogged(),
			logUnchangedByResumeSkip(fixDir),
			noClassificationsLogWritten(),
		),
	})
}

// --- Given helpers (event-style: an expense was already recorded in the log) ---

// seedRow is a logical input row to pre-seed into the expense log via the real append path.
type seedRow struct {
	item        string
	date        time.Time
	value       float64 // per-installment value
	count       int     // installment count (1 = plain)
	expenseType string
	category    string
	subcategory string
}

// expenseLogSeededWith copies the fixture to the work dir, configures feedback + taxonomy
// paths, then writes the given rows into the expense log using appender.ExpandAndAppend — the
// exact code path batch-auto's append phase uses — so seeded ids match the binary's predictions.
func expenseLogSeededWith(rows []seedRow) func(*harness.Context) {
	return func(ctx *harness.Context) {
		expenseBatchSubmittedForClassification()(ctx)

		logPath := ctx.Artifacts["expenses_log.jsonl"]
		for _, r := range rows {
			if err := appender.ExpandAndAppend(logPath, r.item, r.date, r.value, r.count, r.expenseType, r.category, r.subcategory); err != nil {
				ctx.T.Fatalf("seeding expense log: %v", err)
			}
		}
	}
}

func allInputRowsAlreadyLogged() func(*harness.Context) {
	return expenseLogSeededWith([]seedRow{
		{item: "Netflix", date: date(2026, 3, 10), value: 55.90, count: 1, expenseType: "Fixas", category: "Lazer", subcategory: "Netflix"},
		{item: "Posto Ipiranga", date: date(2026, 4, 1), value: 30.00, count: 3, expenseType: "Variáveis", category: "Transporte", subcategory: "Combustível"},
	})
}

func oneOfTwoDuplicatesAlreadyLogged() func(*harness.Context) {
	return expenseLogSeededWith([]seedRow{
		{item: "Uber Centro", date: date(2026, 4, 15), value: 35.50, count: 1, expenseType: "Variáveis", category: "Transporte", subcategory: "Uber/Taxi"},
	})
}

func oneRowAlreadyLoggedOtherIsNew() func(*harness.Context) {
	return expenseLogSeededWith([]seedRow{
		{item: "Netflix", date: date(2026, 3, 10), value: 55.90, count: 1, expenseType: "Fixas", category: "Lazer", subcategory: "Netflix"},
	})
}

func expenseAlreadyLoggedThenReappended() func(*harness.Context) {
	return expenseLogSeededWith([]seedRow{
		{item: "Posto Ipiranga", date: date(2026, 4, 15), value: 35.50, count: 1, expenseType: "Variáveis", category: "Transporte", subcategory: "Combustível"},
	})
}

// date is a small helper for midnight-UTC dates matching ParseDateFlexible's output.
func date(year int, month time.Month, day int) time.Time {
	return time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
}

// --- Then helpers (composable, named by expected result) ---

func bothRowsReportedAsAlreadyLogged() []func(*harness.Context) {
	return []func(*harness.Context){
		verify.OutputContains("SKIP"),
		expect.ResumeSkipCount(2),
	}
}

func exactlyOneRowSkipped() []func(*harness.Context) {
	return []func(*harness.Context){
		expect.ResumeSkipCount(1),
	}
}

func logUnchangedByResumeSkip(fixDir string) []func(*harness.Context) {
	return []func(*harness.Context){
		expect.ExpenseLogMatches(filepath.Join(fixDir, "expected-expenses_log.jsonl")),
	}
}

func noClassificationsLogWritten() []func(*harness.Context) {
	return []func(*harness.Context){
		expect.ClassificationsNotCreated(),
	}
}

// onlyTheNonSkippedRowRoutedToReview asserts review.csv holds exactly one data row (header +
// 1): the second, non-skipped Uber Centro that gate-failed. The skipped first copy must be
// excluded from review.csv.
func onlyTheNonSkippedRowRoutedToReview() []func(*harness.Context) {
	return []func(*harness.Context){
		expect.OutputFileHasRows("review.csv", 2),
	}
}

func newRowAppendedSeededRowSkipped(fixDir string) []func(*harness.Context) {
	return []func(*harness.Context){
		verify.OutputContains("SKIP"),
		expect.ResumeSkipCount(1),
		expect.ExpenseLogMatches(filepath.Join(fixDir, "expected-expenses_log.jsonl")),
	}
}

func rowAppendedAgainWithSingleDuplicateWarning(fixDir string) []func(*harness.Context) {
	return []func(*harness.Context){
		expect.DuplicateWarningCount(1),
		expect.ExpenseLogMatches(filepath.Join(fixDir, "expected-expenses_log.jsonl")),
	}
}
