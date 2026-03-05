//go:build acceptance

package acceptance_test

import (
	"path/filepath"
	"testing"

	"expense-reporter/test/actions"
	"expense-reporter/test/harness"
	"expense-reporter/test/verify"
)

func TestBatchAuto_Basic(t *testing.T) {
	harness.RequireOllama(t, "")

	fixDir := filepath.Join(fixturesDir(), "batch-auto-basic")

	harness.Run(t, harness.Scenario{
		Name:  "batch-auto basic — 10 rows dry-run",
		Given: tenMixedExpensesReadyForBatch(fixDir),
		When:  actions.RunBatchAutoWithFixture(fixDir),
		Then:  classifiedAndReviewFilesProduced(),
	})
}

func TestBatchAuto_MixedConfidence(t *testing.T) {
	harness.RequireOllama(t, "")

	fixDir := filepath.Join(fixturesDir(), "batch-auto-basic")

	harness.Run(t, harness.Scenario{
		Name:  "batch-auto — classified.csv has 11 rows (1 header + 10 data), 7 columns",
		Given: tenMixedExpensesReadyForBatch(fixDir),
		When:  actions.RunBatchAutoWithFixture(fixDir),
		Then:  allInputExpensesClassified(11),
	})
}

func tenMixedExpensesReadyForBatch(fixDir string) func(*harness.Context) {
	return func(ctx *harness.Context) {
		ctx.BinaryPath = binaryPath
		ctx.FixtureDir = fixDir
		if err := harness.CopyFixtureToWorkDir(ctx, fixDir); err != nil {
			ctx.T.Fatalf("CopyFixtureToWorkDir: %v", err)
		}
	}
}

func classifiedAndReviewFilesProduced() []func(*harness.Context) {
	return []func(*harness.Context){
		verify.ExitCodeZero(),
		verify.FileExists("classified.csv"),
		verify.FileExists("review.csv"),
		verify.RowCountAtLeast("classified.csv", 1),
		verify.ColumnCount("classified.csv", 7),
		verify.AllConfidencesInRange("classified.csv", 5),
	}
}

func allInputExpensesClassified(rows int) []func(*harness.Context) {
	return []func(*harness.Context){
		verify.ExitCodeZero(),
		verify.FileExists("classified.csv"),
		verify.FileExists("review.csv"),
		verify.RowCount("classified.csv", rows),
		verify.ColumnCount("classified.csv", 7),
		verify.AllConfidencesInRange("classified.csv", 5),
	}
}
