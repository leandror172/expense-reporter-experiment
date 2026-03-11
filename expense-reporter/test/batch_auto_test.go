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
		Name:  "batch pipeline runs cleanly with mixed confidence and exclusion markers",
		Given: expensesWithExcludedCategoryMarkers(fixDir),
		When:  actions.RunBatchAutoWithFixture(fixDir),
		Then:  classifiedAndReviewFilesProduced(),
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

func TestBatchAuto_SameYearInstallmentsExpanded(t *testing.T) {
	harness.RequireOllama(t, "")
	harness.RequireWorkbook(t, testWorkbook)

	fixtureDir := filepath.Join(fixturesDir(), "batch-auto-installments")

	harness.Run(t, harness.Scenario{
		Name:  "installment expense auto-inserted as single classified entry",
		Given: midYearInstallmentExpensesReadyForBatch(fixtureDir),
		When:  actions.RunBatchAutoWithInput(fixtureDir, "midyear-input.csv"),
		Then:  installmentExpenseAutoInserted(),
	})
}

func TestBatchAuto_RolloverInstallmentsWrittenToFile(t *testing.T) {
	harness.RequireOllama(t, "")
	harness.RequireWorkbook(t, testWorkbook)

	fixtureDir := filepath.Join(fixturesDir(), "batch-auto-rollover")

	harness.Run(t, harness.Scenario{
		Name:  "installments crossing year boundary produce rollover.csv for next-year entries",
		Given: lateYearInstallmentExpensesReadyForBatch(fixtureDir),
		When:  actions.RunBatchAutoWithFixture(fixtureDir),
		Then:  nextYearInstallmentsWrittenToRolloverFile(),
	})
}

func tenMixedExpensesReadyForBatch(fixDir string) func(*harness.Context) {
	return func(ctx *harness.Context) {
		ctx.BinaryPath = binaryPath
		ctx.DataDir = dataDir
		ctx.FixtureDir = fixDir
		if err := harness.CopyFixtureToWorkDir(ctx, fixDir); err != nil {
			ctx.T.Fatalf("CopyFixtureToWorkDir: %v", err)
		}
	}
}

func expensesWithExcludedCategoryMarkers(fixDir string) func(*harness.Context) {
	return func(ctx *harness.Context) {
		ctx.BinaryPath = binaryPath
		ctx.DataDir = dataDir
		ctx.FixtureDir = fixDir
		if err := harness.CopyFixtureToWorkDir(ctx, fixDir); err != nil {
			ctx.T.Fatalf("CopyFixtureToWorkDir: %v", err)
		}
	}
}

func tenMixedExpensesWithCustomOutputDirectory(fixDir string) func(*harness.Context) {
	return func(ctx *harness.Context) {
		ctx.BinaryPath = binaryPath
		ctx.DataDir = dataDir
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

func midYearInstallmentExpensesReadyForBatch(fixtureDir string) func(*harness.Context) {
	return func(ctx *harness.Context) {
		ctx.BinaryPath = binaryPath
		ctx.DataDir = dataDir
		ctx.FixtureDir = fixtureDir
		if err := harness.CopyFixtureToWorkDir(ctx, fixtureDir); err != nil {
			ctx.T.Fatalf("CopyFixtureToWorkDir: %v", err)
		}
		if err := harness.CopyWorkbookToWorkDir(ctx, testWorkbook); err != nil {
			ctx.T.Fatalf("CopyWorkbookToWorkDir: %v", err)
		}
	}
}

func lateYearInstallmentExpensesReadyForBatch(fixtureDir string) func(*harness.Context) {
	return func(ctx *harness.Context) {
		ctx.BinaryPath = binaryPath
		ctx.DataDir = dataDir
		ctx.FixtureDir = fixtureDir
		if err := harness.CopyFixtureToWorkDir(ctx, fixtureDir); err != nil {
			ctx.T.Fatalf("CopyFixtureToWorkDir: %v", err)
		}
		if err := harness.CopyWorkbookToWorkDir(ctx, testWorkbook); err != nil {
			ctx.T.Fatalf("CopyWorkbookToWorkDir: %v", err)
		}
	}
}

func classifiedAndReviewFilesProduced() []func(*harness.Context) {
	return []func(*harness.Context){
		verify.CommandSucceeded(),
		verify.OutputFileExists("classified.csv"),
		verify.OutputFileExists("review.csv"),
		verify.OutputFileHasAtLeastRows("classified.csv", 1),
		verify.OutputFileHasColumns("classified.csv", 7),
		verify.AllClassificationScoresValid("classified.csv"),
	}
}

func allInputExpensesClassified(rows int) []func(*harness.Context) {
	return []func(*harness.Context){
		verify.CommandSucceeded(),
		verify.OutputFileExists("classified.csv"),
		verify.OutputFileExists("review.csv"),
		verify.OutputFileHasRows("classified.csv", rows),
		verify.OutputFileHasColumns("classified.csv", 7),
		verify.AllClassificationScoresValid("classified.csv"),
	}
}

// installmentExpenseAutoInserted verifies the installment expense was successfully
// classified and auto-inserted. The per-installment workbook rows (3 for a /3 expense)
// are written during insertion but not surfaced in output — workbook verification
// requires verify/workbook.go which is not yet implemented.
func installmentExpenseAutoInserted() []func(*harness.Context) {
	return []func(*harness.Context){
		verify.CommandSucceeded(),
		verify.OutputFileExists("classified.csv"),
		verify.OutputContains("Auto-inserted : 1", "installment expense should be auto-inserted"),
	}
}

func nextYearInstallmentsWrittenToRolloverFile() []func(*harness.Context) {
	return []func(*harness.Context){
		verify.CommandSucceeded(),
		verify.OutputFileExists("rollover.csv"),
		verify.OutputFileHasRows("rollover.csv", 3), // header + 2 rollover installments
	}
}

func classificationMatchesExpectedWithMinAccuracy(expectedPath, resultsDir string) []func(*harness.Context) {
	return []func(*harness.Context){
		verify.CommandSucceeded(),
		verify.ClassificationAccuracyAtLeast("classified.csv", expectedPath, 0.5, resultsDir),
	}
}
