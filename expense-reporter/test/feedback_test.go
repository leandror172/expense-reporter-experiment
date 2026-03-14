//go:build acceptance

package acceptance_test

import (
	"path/filepath"
	"slices"
	"testing"

	"expense-reporter/test/actions"
	"expense-reporter/test/harness"
	"expense-reporter/test/verify"
)

func TestAuto_FeedbackLoggedOnInsert(t *testing.T) {
	harness.RequireOllama(t, "")
	harness.RequireWorkbook(t, testWorkbook)

	fixDir := filepath.Join(fixturesDir(), "auto-basic")

	harness.Run(t, harness.Scenario{
		Name:  "auto command logs confirmed feedback entry on successful insert",
		Given: knownExpenseReadyForAutoInsert(fixDir),
		When:  actions.RunAuto("Uber Centro", "35,50", "15/04"),
		Then: slices.Concat(
			autoInsertSucceeded(),
			classificationsMatchExpected(fixDir),
			expenseLogMatchesExpected(fixDir),
		),
	})
}

func TestBatchAuto_FeedbackLoggedForInsertedRows(t *testing.T) {
	harness.RequireOllama(t, "")
	harness.RequireWorkbook(t, testWorkbook)

	fixDir := filepath.Join(fixturesDir(), "batch-auto-feedback")

	harness.Run(t, harness.Scenario{
		Name:  "batch-auto logs confirmed feedback for all auto-inserted rows",
		Given: knownExpenseBatchReadyForInsert(fixDir),
		When:  actions.RunBatchAutoWithFixture(fixDir),
		Then: slices.Concat(
			commandSucceeded(),
			classificationsMatchExpected(fixDir),
			expenseLogMatchesExpected(fixDir),
		),
	})
}

func TestBatchAuto_DryRunNoFeedbackLogged(t *testing.T) {
	harness.RequireOllama(t, "")

	// batch-auto-basic has --dry-run in extra_args — no workbook needed
	fixDir := filepath.Join(fixturesDir(), "batch-auto-basic")

	harness.Run(t, harness.Scenario{
		Name:  "batch-auto dry-run does not create feedback log",
		Given: mixedExpensesReadyForDryRun(fixDir),
		When:  actions.RunBatchAutoWithFixture(fixDir),
		Then: slices.Concat(
			commandSucceeded(),
			noLogsCreated(),
		),
	})
}

func TestAdd_ManualFeedbackLogged(t *testing.T) {
	harness.RequireWorkbook(t, testWorkbook)

	fixDir := filepath.Join(fixturesDir(), "add-feedback")

	harness.Run(t, harness.Scenario{
		Name:  "add command logs manual feedback entry",
		Given: singleExpenseReadyForManualAdd(),
		When:  actions.RunAdd("Padaria Maeda;15/03;27,50;Padaria"),
		Then: slices.Concat(
			commandSucceeded(),
			classificationsMatchExpected(fixDir),
			expenseLogMatchesExpected(fixDir),
		),
	})
}

// --- Given helpers ---

func knownExpenseReadyForAutoInsert(fixDir string) func(*harness.Context) {
	return func(ctx *harness.Context) {
		ctx.BinaryPath = binaryPath
		ctx.DataDir = dataDir
		ctx.FixtureDir = fixDir
		if err := harness.CopyWorkbookToWorkDir(ctx, testWorkbook); err != nil {
			ctx.T.Fatalf("CopyWorkbookToWorkDir: %v", err)
		}
		withFeedbackConfig(ctx)
	}
}

func knownExpenseBatchReadyForInsert(fixDir string) func(*harness.Context) {
	return func(ctx *harness.Context) {
		ctx.BinaryPath = binaryPath
		ctx.DataDir = dataDir
		ctx.FixtureDir = fixDir
		if err := harness.CopyFixtureToWorkDir(ctx, fixDir); err != nil {
			ctx.T.Fatalf("CopyFixtureToWorkDir: %v", err)
		}
		if err := harness.CopyWorkbookToWorkDir(ctx, testWorkbook); err != nil {
			ctx.T.Fatalf("CopyWorkbookToWorkDir: %v", err)
		}
		withFeedbackConfig(ctx)
	}
}

func mixedExpensesReadyForDryRun(fixDir string) func(*harness.Context) {
	return func(ctx *harness.Context) {
		ctx.BinaryPath = binaryPath
		ctx.DataDir = dataDir
		ctx.FixtureDir = fixDir
		if err := harness.CopyFixtureToWorkDir(ctx, fixDir); err != nil {
			ctx.T.Fatalf("CopyFixtureToWorkDir: %v", err)
		}
		withFeedbackConfig(ctx)
	}
}

func singleExpenseReadyForManualAdd() func(*harness.Context) {
	return func(ctx *harness.Context) {
		ctx.BinaryPath = binaryPath
		if err := harness.CopyWorkbookToWorkDir(ctx, testWorkbook); err != nil {
			ctx.T.Fatalf("CopyWorkbookToWorkDir: %v", err)
		}
		withFeedbackConfig(ctx)
	}
}

// withFeedbackConfig writes binary config with classifications_path + expenses_log_path and registers both artifacts.
func withFeedbackConfig(ctx *harness.Context) {
	classificationsPath := filepath.Join(ctx.WorkDir, "classifications.jsonl")
	expensesLogPath := filepath.Join(ctx.WorkDir, "expenses_log.jsonl")
	if err := harness.SetupBinaryConfig(ctx, map[string]interface{}{
		"classifications_path": classificationsPath,
		"expenses_log_path":    expensesLogPath,
	}); err != nil {
		ctx.T.Fatalf("SetupBinaryConfig: %v", err)
	}
	ctx.Artifacts["classifications.jsonl"] = classificationsPath
	ctx.Artifacts["expenses_log.jsonl"] = expensesLogPath
}

// --- Then helpers (composable) ---

func commandSucceeded() []func(*harness.Context) {
	return []func(*harness.Context){
		verify.CommandSucceeded(),
	}
}

func autoInsertSucceeded() []func(*harness.Context) {
	return []func(*harness.Context){
		verify.CommandSucceeded(),
		verify.OutputContains("✓ Inserted"),
	}
}

func classificationsMatchExpected(fixDir string) []func(*harness.Context) {
	return []func(*harness.Context){
		verify.ClassificationsMatch(filepath.Join(fixDir, "expected-feedback.jsonl")),
	}
}

func expenseLogMatchesExpected(fixDir string) []func(*harness.Context) {
	return []func(*harness.Context){
		verify.ExpenseLogMatches(filepath.Join(fixDir, "expected-expenses_log.jsonl")),
	}
}

func noLogsCreated() []func(*harness.Context) {
	return []func(*harness.Context){
		verify.ClassificationsNotCreated(),
		verify.ExpenseLogNotCreated(),
	}
}
