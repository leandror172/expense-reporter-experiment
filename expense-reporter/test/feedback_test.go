//go:build acceptance

package acceptance_test

import (
	"path/filepath"
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
		Then:  autoInsertConfirmedInFeedback(fixDir),
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
		Then:  batchInsertionsConfirmedInFeedback(fixDir),
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
		Then:  noFeedbackFileCreated(),
	})
}

func TestAdd_ManualFeedbackLogged(t *testing.T) {
	harness.RequireWorkbook(t, testWorkbook)

	fixDir := filepath.Join(fixturesDir(), "add-feedback")

	harness.Run(t, harness.Scenario{
		Name:  "add command logs manual feedback entry",
		Given: singleExpenseReadyForManualAdd(),
		When:  actions.RunAdd("Padaria Maeda;15/03;27,50;Padaria"),
		Then:  manualEntryLoggedInFeedback(fixDir),
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

// withFeedbackConfig writes binary config with classifications_path and registers the artifact.
func withFeedbackConfig(ctx *harness.Context) {
	jsonlPath := filepath.Join(ctx.WorkDir, "classifications.jsonl")
	if err := harness.SetupBinaryConfig(ctx, map[string]interface{}{
		"classifications_path": jsonlPath,
	}); err != nil {
		ctx.T.Fatalf("SetupBinaryConfig: %v", err)
	}
	ctx.Artifacts["classifications.jsonl"] = jsonlPath
}

// --- Then helpers ---

func autoInsertConfirmedInFeedback(fixDir string) []func(*harness.Context) {
	return []func(*harness.Context){
		verify.CommandSucceeded(),
		verify.OutputContains("✓ Inserted"),
		verify.FeedbackMatchesExpected("classifications.jsonl", filepath.Join(fixDir, "expected-feedback.jsonl")),
	}
}

func batchInsertionsConfirmedInFeedback(fixDir string) []func(*harness.Context) {
	return []func(*harness.Context){
		verify.CommandSucceeded(),
		verify.FeedbackMatchesExpected("classifications.jsonl", filepath.Join(fixDir, "expected-feedback.jsonl")),
	}
}

func noFeedbackFileCreated() []func(*harness.Context) {
	return []func(*harness.Context){
		verify.CommandSucceeded(),
		verify.FeedbackFileNotExists("classifications.jsonl"),
	}
}

func manualEntryLoggedInFeedback(fixDir string) []func(*harness.Context) {
	return []func(*harness.Context){
		verify.CommandSucceeded(),
		verify.FeedbackMatchesExpected("classifications.jsonl", filepath.Join(fixDir, "expected-feedback.jsonl")),
	}
}
