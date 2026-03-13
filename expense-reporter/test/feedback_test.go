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
		Given: autoFeedbackSetup(fixDir),
		When:  actions.RunAuto("Uber Centro", "35,50", "15/04"),
		Then: []func(*harness.Context){
			verify.CommandSucceeded(),
			verify.OutputContains("✓ Inserted"),
			verify.FeedbackFileExists("classifications.jsonl"),
			verify.FeedbackEntryCount("classifications.jsonl", 1),
			verify.FeedbackContainsStatus("classifications.jsonl", "confirmed"),
			verify.FeedbackContainsItem("classifications.jsonl", "Uber Centro"),
		},
	})
}

func TestBatchAuto_FeedbackLoggedForInsertedRows(t *testing.T) {
	harness.RequireOllama(t, "")
	harness.RequireWorkbook(t, testWorkbook)

	fixDir := filepath.Join(fixturesDir(), "batch-auto-feedback")

	harness.Run(t, harness.Scenario{
		Name:  "batch-auto logs confirmed feedback for all auto-inserted rows",
		Given: batchFeedbackSetup(fixDir),
		When:  actions.RunBatchAutoWithFixture(fixDir),
		Then: []func(*harness.Context){
			verify.CommandSucceeded(),
			verify.FeedbackFileExists("classifications.jsonl"),
			verify.FeedbackAllConfirmed("classifications.jsonl"),
			verify.FeedbackContainsStatus("classifications.jsonl", "confirmed"),
		},
	})
}

func TestBatchAuto_DryRunNoFeedbackLogged(t *testing.T) {
	harness.RequireOllama(t, "")

	// batch-auto-basic has --dry-run in extra_args — no workbook needed
	fixDir := filepath.Join(fixturesDir(), "batch-auto-basic")

	harness.Run(t, harness.Scenario{
		Name:  "batch-auto dry-run does not create feedback log",
		Given: batchFeedbackDryRunSetup(fixDir),
		When:  actions.RunBatchAutoWithFixture(fixDir),
		Then: []func(*harness.Context){
			verify.CommandSucceeded(),
			verify.FeedbackFileNotExists("classifications.jsonl"),
		},
	})
}

func TestAdd_ManualFeedbackLogged(t *testing.T) {
	harness.RequireWorkbook(t, testWorkbook)

	harness.Run(t, harness.Scenario{
		Name:  "add command logs manual feedback entry",
		Given: addFeedbackSetup(),
		When:  actions.RunAdd("Padaria Maeda;15/03;27,50;Padaria"),
		Then: []func(*harness.Context){
			verify.CommandSucceeded(),
			verify.FeedbackFileExists("classifications.jsonl"),
			verify.FeedbackEntryCount("classifications.jsonl", 1),
			verify.FeedbackContainsStatus("classifications.jsonl", "manual"),
			verify.FeedbackContainsItem("classifications.jsonl", "Padaria Maeda"),
		},
	})
}

// --- Given helpers ---

func autoFeedbackSetup(fixDir string) func(*harness.Context) {
	return func(ctx *harness.Context) {
		ctx.BinaryPath = binaryPath
		ctx.DataDir = dataDir
		ctx.FixtureDir = fixDir
		if err := harness.CopyWorkbookToWorkDir(ctx, testWorkbook); err != nil {
			ctx.T.Fatalf("CopyWorkbookToWorkDir: %v", err)
		}
		setupFeedbackConfig(ctx)
	}
}

func batchFeedbackSetup(fixDir string) func(*harness.Context) {
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
		setupFeedbackConfig(ctx)
	}
}

func batchFeedbackDryRunSetup(fixDir string) func(*harness.Context) {
	return func(ctx *harness.Context) {
		ctx.BinaryPath = binaryPath
		ctx.DataDir = dataDir
		ctx.FixtureDir = fixDir
		if err := harness.CopyFixtureToWorkDir(ctx, fixDir); err != nil {
			ctx.T.Fatalf("CopyFixtureToWorkDir: %v", err)
		}
		setupFeedbackConfig(ctx)
	}
}

func addFeedbackSetup() func(*harness.Context) {
	return func(ctx *harness.Context) {
		ctx.BinaryPath = binaryPath
		if err := harness.CopyWorkbookToWorkDir(ctx, testWorkbook); err != nil {
			ctx.T.Fatalf("CopyWorkbookToWorkDir: %v", err)
		}
		setupFeedbackConfig(ctx)
	}
}

// setupFeedbackConfig writes binary config with classifications_path and registers the artifact.
func setupFeedbackConfig(ctx *harness.Context) {
	jsonlPath := filepath.Join(ctx.WorkDir, "classifications.jsonl")
	if err := harness.SetupBinaryConfig(ctx, map[string]interface{}{
		"classifications_path": jsonlPath,
	}); err != nil {
		ctx.T.Fatalf("SetupBinaryConfig: %v", err)
	}
	ctx.Artifacts["classifications.jsonl"] = jsonlPath
}
