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
		Name:  "apply command handles idempotency and feedback correctly",
		Given: expensesAutoInsertedBeforeReview(fixDir),
		When:  actions.RunApply(filepath.Join(fixDir, "reviewed.json")),
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
		Name:  "apply --dry-run writes nothing to either log, even for a found+corrected row",
		Given: expensesAutoInsertedBeforeReview(fixDir),
		When:  actions.RunApplyDryRun(filepath.Join(fixDir, "reviewed.json")),
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
		Name:  "unwritable expense log fails fast with a Hint",
		Given: applyEntriesSubmittedWithUnwritableLogPath(fixDir),
		When:  actions.RunApply(filepath.Join(fixDir, "reviewed.json")),
		Then:  commandFailedWithHint(),
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
		Name:  "unwritable classifications path fails fast with a Hint",
		Given: applyEntriesSubmittedWithUnwritableClassificationsPath(fixDir),
		When:  actions.RunApply(filepath.Join(fixDir, "reviewed.json")),
		Then:  commandFailedWithHint(),
	})
}

// --- Given helpers (Event Modeling style — past-tense events that happened) ---

func expensesAutoInsertedBeforeReview(fixDir string) func(*harness.Context) {
	return func(ctx *harness.Context) {
		ctx.BinaryPath = binaryPath
		domain.SetDataDir(ctx, dataDir)
		ctx.FixtureDir = fixDir
		withFeedbackConfig(ctx)
		if err := harness.SeedFileFromFixture(ctx, fixDir, "seed-classifications.jsonl", "classifications.jsonl"); err != nil {
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
func applyEntriesSubmittedWithUnwritableLogPath(fixDir string) func(*harness.Context) {
	return func(ctx *harness.Context) {
		ctx.BinaryPath = binaryPath
		domain.SetDataDir(ctx, dataDir)
		ctx.FixtureDir = fixDir
		if err := harness.CopyFixtureToWorkDir(ctx, fixDir); err != nil {
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
func applyEntriesSubmittedWithUnwritableClassificationsPath(fixDir string) func(*harness.Context) {
	return func(ctx *harness.Context) {
		ctx.BinaryPath = binaryPath
		domain.SetDataDir(ctx, dataDir)
		ctx.FixtureDir = fixDir
		if err := harness.CopyFixtureToWorkDir(ctx, fixDir); err != nil {
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
		Name:  "apply writes one row to each log under the same join id",
		Given: reviewQueueSubmittedWithNoPriorClassifications(fixDir),
		When:  actions.RunApply(filepath.Join(fixDir, "reviewed.json")),
		Then: slices.Concat(
			commandSucceeded(),
			newRowJoinableAcrossBothLogs(),
		),
	})
}

// reviewQueueSubmittedWithNoPriorClassifications seeds nothing: the row must be NEW so
// it takes the append path and lands exactly once in each log, making the join unambiguous.
func reviewQueueSubmittedWithNoPriorClassifications(fixDir string) func(*harness.Context) {
	return func(ctx *harness.Context) {
		ctx.BinaryPath = binaryPath
		domain.SetDataDir(ctx, dataDir)
		ctx.FixtureDir = fixDir
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
