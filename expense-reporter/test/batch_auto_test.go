//go:build acceptance

package acceptance_test

import (
	"os"
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

func TestBatchAuto_ExcludedCategoriesGoToReview(t *testing.T) {
	harness.RequireOllama(t, "")

	fixDir := filepath.Join(fixturesDir(), "batch-auto-exclusions")

	harness.Run(t, harness.Scenario{
		Name:  "items classified as Diversos must land in review.csv regardless of confidence",
		Given: expensesWithExcludedCategoryMarkers(fixDir),
		When:  actions.RunBatchAutoWithFixture(fixDir),
		Then:  noneAutoInsertedDueToExclusions(),
	})
}

func TestBatchAuto_ClassificationAccuracy(t *testing.T) {
	harness.RequireOllama(t, "")

	fixDir := filepath.Join(fixturesDir(), "batch-auto-basic")
	expectedPath := filepath.Join(fixDir, "expected-classified.csv")
	resultsDir := filepath.Join(fixturesDir(), "..", "results")

	harness.Run(t, harness.Scenario{
		Name:  "batch-auto accuracy >= 50% against expected classifications",
		Given: tenMixedExpensesReadyForBatch(fixDir),
		When:  actions.RunBatchAutoWithFixture(fixDir),
		Then:  classificationMatchesExpectedWithMinAccuracy(expectedPath, resultsDir),
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

func expensesWithExcludedCategoryMarkers(fixDir string) func(*harness.Context) {
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

func noneAutoInsertedDueToExclusions() []func(*harness.Context) {
	return []func(*harness.Context){
		verify.ExitCodeZero(),
		verify.FileExists("classified.csv"),
		verify.FileExists("review.csv"),
		verify.AllInReview("classified.csv", 6), // no row was auto-inserted
		verify.RowCount("review.csv", 4),        // 1 header + 3 data rows
	}
}

func TestBatchAuto_OutputDirFlag(t *testing.T) {
	harness.RequireOllama(t, "")

	fixDir := filepath.Join(fixturesDir(), "batch-auto-basic")

	harness.Run(t, harness.Scenario{
		Name:  "output CSVs are written to --output-dir, not input file directory",
		Given: tenMixedExpensesWithCustomOutputDirectory(fixDir),
		When:  actions.RunBatchAutoIntoArtifactDir(fixDir, "outDir"),
		Then:  classifiedAndReviewFilesProduced(),
	})
}

func tenMixedExpensesWithCustomOutputDirectory(fixDir string) func(*harness.Context) {
	return func(ctx *harness.Context) {
		ctx.BinaryPath = binaryPath
		ctx.FixtureDir = fixDir
		if err := harness.CopyFixtureToWorkDir(ctx, fixDir); err != nil {
			ctx.T.Fatalf("CopyFixtureToWorkDir: %v", err)
		}
		outDir := filepath.Join(ctx.WorkDir, "out")
		if err := os.MkdirAll(outDir, 0o755); err != nil {
			ctx.T.Fatalf("mkdir out: %v", err)
		}
		ctx.Artifacts["outDir"] = outDir
	}
}

func classificationMatchesExpectedWithMinAccuracy(expectedPath, resultsDir string) []func(*harness.Context) {
	return []func(*harness.Context){
		verify.ExitCodeZero(),
		verify.SoftAccuracy("classified.csv", expectedPath, 0.5, 3, resultsDir),
	}
}
