//go:build acceptance

package acceptance_test

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"expense-reporter/test/actions"
	"expense-reporter/test/domain"
	"expense-reporter/test/expect"
	"github.com/leandror172/acceptance-harness/harness"
	"github.com/leandror172/acceptance-harness/verify"
)

func TestApply_IdempotencyAndFeedback(t *testing.T) {
	fixDir := filepath.Join(fixturesDir(), "apply-basic")

	harness.Run(t, harness.Scenario{
		Name:    "apply command handles idempotency and feedback correctly",
		Fixture: fixDir,
		Given:   expensesAutoInsertedBeforeReview(),
		When:    actions.RunApply(filepath.Join(fixDir, "reviewed.json")),
		Then: slices.Concat(
			commandSucceeded(),
			correctionsLoggedForAlreadyInserted(fixDir),
			noNewExpensesInserted(),
			summaryMentionsCorrections(),
		),
	})
}

// TestApply_DryRunWritesNothing guards the dry-run leak advisor finding #1: today
// handleActiveEntry's found+corrected branch calls feedback.Append UNCONDITIONALLY,
// before any dry-run check, so `apply --dry-run` against an already-found corrected
// entry (apply-basic entry 2, Diarista) writes a corrected feedback line even though
// the user asked for a preview. Slice 4 must thread dryRun into
// processEntries/handleActiveEntry and gate that write. Until then this test is RED:
// classifications.jsonl gains a 3rd line during a run that should write nothing to
// either log.
func TestApply_DryRunWritesNothing(t *testing.T) {
	fixDir := filepath.Join(fixturesDir(), "apply-basic")

	harness.Run(t, harness.Scenario{
		Name:    "apply --dry-run writes nothing to either log, even for a found+corrected row",
		Fixture: fixDir,
		Given:   expensesAutoInsertedBeforeReview(),
		When:    actions.RunApplyDryRun(filepath.Join(fixDir, "reviewed.json")),
		Then: slices.Concat(
			commandSucceeded(),
			dryRunLeftClassificationsUnchanged(fixDir),
			noNewExpensesInserted(),
		),
	})
}

// TestApply_UnwritableLogPath_FailsFast guards the non-dry-run pre-flight (plan §5):
// an unwritable/unconfigured expenses_log_path must abort BEFORE any row is
// processed, with a `Hint:` in the error. No pre-flight exists today — apply.go
// silently no-ops the expense-log write when expensesLogPath canot be resolved and
// only fails (without a Hint) once it tries to allocate workbook rows. RED until
// the pre-flight is added.
func TestApply_UnwritableLogPath_FailsFast(t *testing.T) {
	fixDir := filepath.Join(fixturesDir(), "apply-basic")

	harness.Run(t, harness.Scenario{
		Name:    "unwritable expense log fails fast with a Hint",
		Fixture: fixDir,
		Given:   applyEntriesSubmittedWithUnwritableLogPath(),
		When:    actions.RunApply(filepath.Join(fixDir, "reviewed.json")),
		Then:    commandFailedWithHint(),
	})
}

// TestApply_UnwritableClassificationsPath_FailsFast is the both-path-pre-flight
// sibling (advisor finding #2): apply uses classifications.jsonl as the dedup index
// for expenses_log.jsonl, so an unwritable classifications path must ALSO fail fast,
// not just the expense log (unlike batch-auto's log-only pre-flight). This is the
// case that makes the §2 duplicate-on-re-run scenario unreachable. RED until the
// pre-flight checks BOTH paths — if the main session mirrors batch-auto's
// log-only pre-flight, this test stays red.
func TestApply_UnwritableClassificationsPath_FailsFast(t *testing.T) {
	fixDir := filepath.Join(fixturesDir(), "apply-basic")

	harness.Run(t, harness.Scenario{
		Name:    "unwritable classifications path fails fast with a Hint",
		Fixture: fixDir,
		Given:   applyEntriesSubmittedWithUnwritableClassificationsPath(),
		When:    actions.RunApply(filepath.Join(fixDir, "reviewed.json")),
		Then:    commandFailedWithHint(),
	})
}

// --- Given helpers (Event Modeling style — past-tense events that happened) ---

func expensesAutoInsertedBeforeReview() func(*harness.Context) {
	return func(ctx *harness.Context) {
		domain.SetDataDir(ctx, dataDir)
		withFeedbackConfig(ctx)
		if err := harness.SeedFileFromFixture(ctx, ctx.FixtureDir, "seed-classifications.jsonl", "classifications.jsonl"); err != nil {
			ctx.T.Fatalf("SeedFileFromFixture: %v", err)
		}
	}
}

// applyEntriesSubmittedWithUnwritableLogPath points expenses_log_path at a file whose
// PARENT is itself a regular file ("blocker"), so the pre-flight's directory creation
// fails and the run aborts before any entry is processed. classifications.jsonl is
// intentionally left unseeded (entries are "new" rather than "found"), matching the
// non-dry-run path that actually needs to write a log. Local-model generated
// (my-go-qcoder), mirrors batchSubmittedWithUnwritableLogPath in batch_auto_test.go.
func applyEntriesSubmittedWithUnwritableLogPath() func(*harness.Context) {
	return func(ctx *harness.Context) {
		domain.SetDataDir(ctx, dataDir)
		if err := harness.CopyFixtureToWorkDir(ctx, ctx.FixtureDir); err != nil {
			ctx.T.Fatalf("CopyFixtureToWorkDir: %v", err)
		}
		blocker := filepath.Join(ctx.WorkDir, "blocker")
		if err := os.WriteFile(blocker, []byte("x"), 0o644); err != nil {
			ctx.T.Fatalf("writing blocker file: %v", err)
		}
		if err := domain.SetupBinaryConfig(ctx, map[string]interface{}{
			"classifications_path": filepath.Join(ctx.WorkDir, "classifications.jsonl"),
			"expenses_log_path":    filepath.Join(blocker, "expenses_log.jsonl"),
		}); err != nil {
			ctx.T.Fatalf("SetupBinaryConfig: %v", err)
		}
	}
}

// applyEntriesSubmittedWithUnwritableClassificationsPath is the sibling of
// applyEntriesSubmittedWithUnwritableLogPath: classifications_path (not
// expenses_log_path) is the one nested under the blocker file. Proves the
// dedup-index pre-flight, not just the log pre-flight. Local-model generated
// (my-go-qcoder), mirrors batchSubmittedWithUnwritableLogPath in batch_auto_test.go.
func applyEntriesSubmittedWithUnwritableClassificationsPath() func(*harness.Context) {
	return func(ctx *harness.Context) {
		domain.SetDataDir(ctx, dataDir)
		if err := harness.CopyFixtureToWorkDir(ctx, ctx.FixtureDir); err != nil {
			ctx.T.Fatalf("CopyFixtureToWorkDir: %v", err)
		}
		blocker := filepath.Join(ctx.WorkDir, "blocker")
		if err := os.WriteFile(blocker, []byte("x"), 0o644); err != nil {
			ctx.T.Fatalf("writing blocker file: %v", err)
		}
		if err := domain.SetupBinaryConfig(ctx, map[string]interface{}{
			"classifications_path": filepath.Join(blocker, "classifications.jsonl"),
			"expenses_log_path":    filepath.Join(ctx.WorkDir, "expenses_log.jsonl"),
		}); err != nil {
			ctx.T.Fatalf("SetupBinaryConfig: %v", err)
		}
	}
}

// --- Then helpers (composable) ---

func correctionsLoggedForAlreadyInserted(fixDir string) []func(*harness.Context) {
	return []func(*harness.Context){
		expect.ClassificationsMatch(filepath.Join(fixDir, "expected-feedback.jsonl")),
	}
}

func noNewExpensesInserted() []func(*harness.Context) {
	return []func(*harness.Context){
		expect.ExpenseLogNotCreated(),
	}
}

// dryRunLeftClassificationsUnchanged asserts classifications.jsonl is byte-identical
// to the seed (no corrected-feedback line appended during --dry-run). Outcome-named
// per the PR #35 naming sweep — the seed file IS the expectation under dry-run.
func dryRunLeftClassificationsUnchanged(fixDir string) []func(*harness.Context) {
	return []func(*harness.Context){
		expect.ClassificationsMatch(filepath.Join(fixDir, "seed-classifications.jsonl")),
	}
}

func summaryMentionsCorrections() []func(*harness.Context) {
	return []func(*harness.Context){
		verify.OutputContains("no expense-log change"),
	}
}

// TestApply_JoinIDMatchesAcrossLogs guards the T-35 join-key divergence: apply writes
// BOTH logs, and they share no foreign key other than the sha256[:12] id built from
// (item, date, value). apply normalized the date to DD/MM/YYYY for expenses_log.jsonl
// (via ParseDateWithYear) but handed the RAW review-queue string to the feedback entry,
// so the same expense landed under two different ids and every cross-file join silently
// broke — nothing errored, the rows just stopped corresponding.
//
// The fixture's short "15/04" date is what exposes it: with a full "15/04/2026" the raw
// and normalized strings are byte-identical, the hashes agree by accident, and the bug
// hides. The assertion compares only the two ids — never a literal date — so it cannot
// rot at year rollover.
func TestApply_JoinIDMatchesAcrossLogs(t *testing.T) {
	fixDir := filepath.Join(fixturesDir(), "apply-join-id")

	harness.Run(t, harness.Scenario{
		Name:    "apply writes one row to each log under the same join id",
		Fixture: fixDir,
		Given:   reviewQueueSubmittedWithNoPriorClassifications(),
		When:    actions.RunApply(filepath.Join(fixDir, "reviewed.json")),
		Then: slices.Concat(
			commandSucceeded(),
			newRowJoinableAcrossBothLogs(),
		),
	})
}

// reviewQueueSubmittedWithNoPriorClassifications seeds nothing: the row must be NEW so
// it takes the append path and lands exactly once in each log, making the join unambiguous.
func reviewQueueSubmittedWithNoPriorClassifications() func(*harness.Context) {
	return func(ctx *harness.Context) {
		domain.SetDataDir(ctx, dataDir)
		withFeedbackConfig(ctx)
	}
}

// newRowJoinableAcrossBothLogs asserts the applied row carries the same join id in
// classifications.jsonl and expenses_log.jsonl — i.e. the two logs still correspond.
func newRowJoinableAcrossBothLogs() []func(*harness.Context) {
	return []func(*harness.Context){
		expect.JoinIDMatchesAcrossLogs(),
	}
}

// TestApply_StaleReviewedIDIsRecomputedNotTrusted proves apply derives an entry's
// identity from the entry's own data rather than trusting the id it arrived with.
//
// Why this needs its own fixture: apply rewrites a bare date to canonical form before
// anything else, so any id supplied by whoever produced reviewed.json was hashed from a
// date apply is about to change. Trusting it means looking up an id apply will never
// itself write, and that lookup miss is SILENT — handleActiveEntry treats an unfound
// entry as new and appends it. The symptom is therefore a duplicate expense in the log,
// not an error anyone sees.
//
// The sibling apply-basic fixture cannot catch this. It is internally self-consistent,
// which is right for what it tests, but that means the lookup succeeds whether or not
// the id was recomputed — verified by mutation: with the recompute removed, apply-basic
// stays green and only this scenario goes red.
func TestApply_StaleReviewedIDIsRecomputedNotTrusted(t *testing.T) {
	fixDir := filepath.Join(fixturesDir(), "apply-stale-id")

	harness.Run(t, harness.Scenario{
		Name:    "apply recognises a row whose reviewed.json id was hashed from the pre-canonical date",
		Fixture: fixDir,
		Given:   expensesAutoInsertedBeforeReview(),
		When:    actions.RunApply(filepath.Join(fixDir, "reviewed.json")),
		Then: slices.Concat(
			commandSucceeded(),
			noNewExpensesInserted(),
		),
	})
}

// TestApply_ReviewedInstallmentExpandsToOneRowPerInstallment guards T-21: a reviewed
// installment purchase must be recorded as one dated row per installment, not as one row.
//
// apply was the only caller of appender.ExpandAndAppend passing a literal 1 as the
// installment count — not because it chose to, but because it had nothing else to pass:
// the count was discarded two hops upstream in review.ReadQueue and never existed in the
// reviewed.json contract at all. add, auto and batch-auto all passed the real count.
//
// Nothing downstream could notice. A three-installment purchase recorded as a single row
// is indistinguishable from a legitimate one-off expense of the same per-installment
// value, so the workbook simply came out short and no check anywhere went red.
func TestApply_ReviewedInstallmentExpandsToOneRowPerInstallment(t *testing.T) {
	fixDir := filepath.Join(fixturesDir(), "apply-reviewed-installments")

	harness.Run(t, harness.Scenario{
		Name:    "apply expands a reviewed installment purchase into one row per installment",
		Fixture: fixDir,
		Given:   reviewQueueSubmittedWithNoPriorClassifications(),
		When:    actions.RunApply(filepath.Join(fixDir, "reviewed.json")),
		Then: slices.Concat(
			commandSucceeded(),
			reviewedInstallmentExpandedToNDatedLogLines(fixDir),
		),
	})
}

// TestApply_AppliedInstallmentSeriesIsRecognisedByResume proves the ids apply writes for
// an installment series are the ids the rest of the system will look for.
//
// Asserting that directly — comparing apply's output against appender.PredictEntryIDs —
// would be circular: both route through the same expandEntries, so they agree by
// construction and the test would pass with the arithmetic arbitrarily wrong. (The same
// circularity got a slice-3 test rejected in session 66: a parser asserted against a
// second call to itself.)
//
// So the claim is expressed as a behavior spanning two independent value-derivation
// paths. apply takes its per-installment value as an already-typed JSON float out of
// reviewed.json; batch-auto derives the same value from the raw "100,00/3" token through
// internal/parse. If the series apply wrote is the series batch-auto predicts, --resume
// reports the row as already logged. Before T-21 it could not: apply wrote one row under
// the UNSUFFIXED id while --resume predicts three suffixed ones, so the two shared no id
// at all and every re-run sent the row back to review.
//
// Ollama-free while green — a full resume match is consumed before the model call.
func TestApply_AppliedInstallmentSeriesIsRecognisedByResume(t *testing.T) {
	fixDir := filepath.Join(fixturesDir(), "apply-installment-resume-join")

	harness.Run(t, harness.Scenario{
		Name:    "batch-auto --resume recognises an installment series that apply wrote",
		Fixture: fixDir,
		Given:   installmentExpenseReviewedAndApplied(),
		When:    actions.RunBatchAutoWithFixture(),
		Then: slices.Concat(
			commandSucceeded(),
			exactlyOneRowSkipped(),
		),
	})
}

// Deliberately NOT asserted here: that classifications.jsonl was never created. The
// seeding apply run writes it — that is what apply does — so the assertion would be
// checking the Given rather than the When. The skip count is the whole claim: a row is
// only skipped when every id the resume ladder predicts is already in the expense log.

// installmentExpenseReviewedAndApplied: the expense in this fixture's input.csv was
// reviewed in the browser and applied, so the expense log already holds its installment
// series. The seeding runs apply ITSELF rather than calling appender directly — seeding
// through the library would prove only that appender agrees with appender, which is the
// circularity this scenario exists to avoid.
//
// The apply run is registered as a BeforeWhen hook rather than executed inline, because
// SetupBinaryConfig does not write config.json when called: it accumulates keys and
// flushes them from its own BeforeWhen, so during the Given phase the binary still has no
// config to read (apply fails with "classifications log path is not configured"). Hooks
// fire in registration order and the canonical Given registers the config flush first, so
// this ordering is config → seed → When.
func installmentExpenseReviewedAndApplied() func(*harness.Context) {
	return func(ctx *harness.Context) {
		expenseBatchSubmittedForClassification()(ctx)
		ctx.BeforeWhen(func() { seedExpenseLogByApplying(ctx) })
	}
}

// seedExpenseLogByApplying runs apply over the fixture's reviewed.json and fails the
// scenario if it did not succeed — a silently failed seed would leave the expense log
// empty, and an empty log skips nothing, which is the very outcome under test.
func seedExpenseLogByApplying(ctx *harness.Context) {
	actions.RunApply(filepath.Join(ctx.FixtureDir, "reviewed.json"))(ctx)
	if ctx.ExitCode != 0 {
		ctx.T.Fatalf("seeding apply run failed (exit %d): %s", ctx.ExitCode, ctx.Stderr)
	}
}

// reviewedInstallmentExpandedToNDatedLogLines asserts the reviewed purchase became one
// log line per installment, each suffixed (i/N) and dated a month after the last.
func reviewedInstallmentExpandedToNDatedLogLines(fixDir string) []func(*harness.Context) {
	return []func(*harness.Context){
		expect.ExpenseLogMatches(filepath.Join(fixDir, "expected-expenses_log.jsonl")),
	}
}
