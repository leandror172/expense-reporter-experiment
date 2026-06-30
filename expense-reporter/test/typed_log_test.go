//go:build acceptance

package acceptance_test

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"expense-reporter/test/actions"
	"expense-reporter/test/harness"
)

func TestBatchAuto_TypeEmittedInExpenseLog(t *testing.T) {
	harness.RequireOllama(t, "")

	fixDir := filepath.Join(fixturesDir(), "batch-auto-typed")

	harness.Run(t, harness.Scenario{
		Name:  "batch-auto with taxonomy logs entries with correct type field",
		Given: typedBatchReadyForLogAppend(fixDir),
		When:  actions.RunBatchAutoWithFixture(fixDir),
		Then: slices.Concat(
			commandSucceeded(),
			classificationsMatchExpected(fixDir),
			expenseLogMatchesExpected(fixDir),
		),
	})
}

// typedBatchReadyForLogAppend sets up the canonical non-dry-run append anchor: fixture +
// taxonomy config, no workbook (the log-append path needs no workbook). This is the
// load-bearing coverage of the append path that the dry-run survivors do not provide.
func typedBatchReadyForLogAppend(fixDir string) func(*harness.Context) {
	return func(ctx *harness.Context) {
		ctx.BinaryPath = binaryPath
		ctx.DataDir = dataDir
		ctx.FixtureDir = fixDir
		if err := harness.CopyFixtureToWorkDir(ctx, fixDir); err != nil {
			ctx.T.Fatalf("CopyFixtureToWorkDir: %v", err)
		}
		withFeedbackAndTaxonomyConfig(ctx, fixDir)
	}
}

// withFeedbackAndTaxonomyConfig writes config with feedback paths + taxonomy path,
// and copies the fixture taxonomy into the work dir so the binary can load it.
func withFeedbackAndTaxonomyConfig(ctx *harness.Context, fixDir string) {
	classificationsPath := filepath.Join(ctx.WorkDir, "classifications.jsonl")
	expensesLogPath := filepath.Join(ctx.WorkDir, "expenses_log.jsonl")
	taxonomyDest := filepath.Join(ctx.WorkDir, "taxonomy.json")

	taxData, err := os.ReadFile(filepath.Join(fixDir, "fixture-taxonomy.json"))
	if err != nil {
		ctx.T.Fatalf("reading fixture taxonomy: %v", err)
	}
	if err := os.WriteFile(taxonomyDest, taxData, 0o644); err != nil {
		ctx.T.Fatalf("writing taxonomy to workdir: %v", err)
	}

	if err := harness.SetupBinaryConfig(ctx, map[string]interface{}{
		"classifications_path": classificationsPath,
		"expenses_log_path":    expensesLogPath,
		"taxonomy_path":        taxonomyDest,
	}); err != nil {
		ctx.T.Fatalf("SetupBinaryConfig: %v", err)
	}
	ctx.Artifacts["classifications.jsonl"] = classificationsPath
	ctx.Artifacts["expenses_log.jsonl"] = expensesLogPath
}
