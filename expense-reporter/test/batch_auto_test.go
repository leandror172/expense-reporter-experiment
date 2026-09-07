//go:build acceptance

package acceptance_test

import (
	"os"
	"path/filepath"
	"testing"

	"expense-reporter/test/actions"
	"expense-reporter/test/domain"
	"expense-reporter/test/expect"
	"expense-reporter/test/extern"
	"github.com/leandror172/acceptance-harness/harness"
	"github.com/leandror172/acceptance-harness/verify"
)

func TestBatchAuto_Basic(t *testing.T) {
	extern.RequireOllama(t, "")

	fixDir := filepath.Join(fixturesDir(), "batch-auto-basic")

	harness.Run(t, harness.Scenario{
		Name:    "batch-auto basic — 10 rows dry-run",
		Fixture: fixDir,
		Given:   tenMixedExpensesSubmittedForClassification(),
		When:    actions.RunBatchAutoWithFixture(),
		Then:    classifiedAndReviewFilesProduced(),
	})
}

func TestBatchAuto_MixedConfidence(t *testing.T) {
	extern.RequireOllama(t, "")

	fixDir := filepath.Join(fixturesDir(), "batch-auto-basic")

	harness.Run(t, harness.Scenario{
		Name:    "batch-auto — classified.csv has 11 rows (1 header + 10 data), 9 columns",
		Fixture: fixDir,
		Given:   tenMixedExpensesSubmittedForClassification(),
		When:    actions.RunBatchAutoWithFixture(),
		Then:    allInputExpensesClassified(11),
	})
}

func TestBatchAuto_ExcludedCategoriesGoToReview(t *testing.T) {
	extern.RequireOllama(t, "")

	fixDir := filepath.Join(fixturesDir(), "batch-auto-exclusions")

	harness.Run(t, harness.Scenario{
		Name:    "batch pipeline runs cleanly with mixed confidence and exclusion markers",
		Fixture: fixDir,
		Given:   expensesWithExcludedCategoryMarkers(),
		When:    actions.RunBatchAutoWithFixture(),
		Then:    classifiedAndReviewFilesProduced(),
	})
}

func TestBatchAuto_ClassificationAccuracy(t *testing.T) {
	extern.RequireOllama(t, "")

	fixDir := filepath.Join(fixturesDir(), "batch-auto-basic")
	expectedPath := filepath.Join(fixDir, "expected-classified.csv")
	resultsDir := filepath.Join(fixturesDir(), "..", "results")

	harness.Run(t, harness.Scenario{
		Name:    "batch-auto accuracy >= 50% against expected classifications",
		Fixture: fixDir,
		Given:   tenMixedExpensesSubmittedForClassification(),
		When:    actions.RunBatchAutoWithFixture(),
		Then:    classificationMatchesExpectedWithMinAccuracy(expectedPath, resultsDir),
	})
}

func TestBatchAuto_OutputDirFlag(t *testing.T) {
	extern.RequireOllama(t, "")

	fixDir := filepath.Join(fixturesDir(), "batch-auto-basic")

	harness.Run(t, harness.Scenario{
		Name:    "output CSVs are written to --output-dir, not input file directory",
		Fixture: fixDir,
		Given:   tenMixedExpensesWithCustomOutputDirectory(),
		When:    actions.RunBatchAutoIntoArtifactDir("outDir"),
		Then:    classifiedAndReviewFilesProduced(),
	})
}

func TestBatchAuto_SameYearInstallmentsExpandedInLog(t *testing.T) {
	extern.RequireOllama(t, "")

	fixtureDir := filepath.Join(fixturesDir(), "batch-auto-installments")

	harness.Run(t, harness.Scenario{
		Name:    "same-year installments expand into one dated log line each",
		Fixture: fixtureDir,
		Given:   expenseBatchSubmittedForClassification(),
		When:    actions.RunBatchAutoWithInput("midyear-input.csv"),
		Then:    installmentsExpandedInLog(fixtureDir),
	})
}

func TestBatchAuto_CrossYearInstallmentsLoggedNotRolledOver(t *testing.T) {
	extern.RequireOllama(t, "")

	fixtureDir := filepath.Join(fixturesDir(), "batch-auto-rollover")

	harness.Run(t, harness.Scenario{
		Name:    "cross-year installments are logged with their real next-year date — no rollover.csv",
		Fixture: fixtureDir,
		Given:   expenseBatchSubmittedForClassification(),
		When:    actions.RunBatchAutoWithFixture(),
		Then:    crossYearInstallmentsLoggedNotRolledOver(fixtureDir),
	})
}

// tenMixedExpensesSubmittedForClassification: the batch-auto-basic batch — ten expenses
// of mixed familiarity, some the keyword dictionary knows and some it does not.
func tenMixedExpensesSubmittedForClassification() func(*harness.Context) {
	return expenseBatchSubmittedForClassification()
}

// expensesWithExcludedCategoryMarkers: the submitted batch contains rows whose
// subcategory sits in auto_insert_excluded, so the gate must route them to review.
func expensesWithExcludedCategoryMarkers() func(*harness.Context) {
	return expenseBatchSubmittedForClassification()
}

// tenMixedExpensesWithCustomOutputDirectory: the same batch, plus a separate directory
// the run is told to write its output CSVs into instead of beside the input.
func tenMixedExpensesWithCustomOutputDirectory() func(*harness.Context) {
	return harness.Events(
		expenseBatchSubmittedForClassification(),
		separateOutputDirectoryPrepared(),
	)
}

// separateOutputDirectoryPrepared: a directory outside the input's own exists for the
// run to write into, registered as the "outDir" artifact.
func separateOutputDirectoryPrepared() func(*harness.Context) {
	return func(ctx *harness.Context) {
		outDir := filepath.Join(ctx.WorkDir, "out")
		if err := os.MkdirAll(outDir, 0o755); err != nil {
			ctx.T.Fatalf("mkdir out: %v", err)
		}
		ctx.Artifacts["outDir"] = outDir
	}
}

func classifiedAndReviewFilesProduced() []func(*harness.Context) {
	return []func(*harness.Context){
		verify.CommandSucceeded(),
		verify.OutputFileExists("classified.csv"),
		verify.OutputFileExists("review.csv"),
		expect.OutputFileHasAtLeastRows("classified.csv", 1),
		expect.OutputFileHasColumns("classified.csv", 10),
		expect.AllClassificationScoresValid("classified.csv"),
	}
}

func allInputExpensesClassified(rows int) []func(*harness.Context) {
	return []func(*harness.Context){
		verify.CommandSucceeded(),
		verify.OutputFileExists("classified.csv"),
		verify.OutputFileExists("review.csv"),
		expect.OutputFileHasRows("classified.csv", rows),
		expect.OutputFileHasColumns("classified.csv", 10),
		expect.AllClassificationScoresValid("classified.csv"),
	}
}

// installmentsExpandedInLog verifies a /N installment auto-row expands into N dated lines
// in expenses_log.jsonl (the harness can read the log directly; it could not read workbook
// cells — the old test could only assert a summary string). expected-expenses_log.jsonl
// pins the per-installment item suffix, incremented dates, and split value.
func installmentsExpandedInLog(fixDir string) []func(*harness.Context) {
	return []func(*harness.Context){
		verify.CommandSucceeded(),
		verify.OutputFileExists("classified.csv"),
		expect.ExpenseLogMatches(filepath.Join(fixDir, "expected-expenses_log.jsonl")),
	}
}

// crossYearInstallmentsLoggedNotRolledOver verifies the rollover-retirement: installments
// crossing the year boundary are logged as normal lines carrying their real next-year date,
// and NO rollover.csv is produced. expected-expenses_log.jsonl pins all N lines, including
// the next-year ones (e.g. 01/01/2027).
func crossYearInstallmentsLoggedNotRolledOver(fixDir string) []func(*harness.Context) {
	return []func(*harness.Context){
		verify.CommandSucceeded(),
		expect.NoRolloverFileCreated(),
		expect.ExpenseLogMatches(filepath.Join(fixDir, "expected-expenses_log.jsonl")),
	}
}

func classificationMatchesExpectedWithMinAccuracy(expectedPath, resultsDir string) []func(*harness.Context) {
	return []func(*harness.Context){
		verify.CommandSucceeded(),
		expect.ClassificationAccuracyAtLeast("classified.csv", expectedPath, 0.5, resultsDir),
	}
}

// TestBatchAuto_UnwritableLogPath_FailsFastBeforeClassification verifies the log-append
// pivot's pre-flight. Because expenses_log.jsonl is now the only durable persistence, an
// unwritable log must abort the run BEFORE the (slow ~12 s/row) classification rather than
// classify everything and then fail to persist. Deterministic — fails before any Ollama
// call, so there is no RequireOllama gate. Replaces the old missing-workbook fast-fail test
// (there is no workbook pre-flight anymore) and absorbs the "fail fast" value of the
// deleted corrupt-workbook InsertFailure test.
func TestBatchAuto_UnwritableLogPath_FailsFastBeforeClassification(t *testing.T) {
	fixDir := filepath.Join(fixturesDir(), "batch-auto-corrupt-workbook")

	harness.Run(t, harness.Scenario{
		Name:    "unwritable expense log fails fast before classification",
		Fixture: fixDir,
		Given:   batchSubmittedWithUnwritableLogPath(),
		When:    actions.RunBatchAutoWithFixture(),
		Then:    commandFailedWithHint(),
	})
}

// batchSubmittedWithUnwritableLogPath points expenses_log_path at a file whose PARENT is
// itself a regular file, so the pre-flight's MkdirAll fails and the run aborts. Taxonomy is
// configured (it loads before the pre-flight) but its contents are irrelevant — no row is
// ever classified. Self-contained config (SetupBinaryConfig replaces the whole file).
func batchSubmittedWithUnwritableLogPath() func(*harness.Context) {
	return func(ctx *harness.Context) {
		domain.SetDataDir(ctx, dataDir)
		if err := harness.CopyFixtureToWorkDir(ctx, ctx.FixtureDir); err != nil {
			ctx.T.Fatalf("CopyFixtureToWorkDir: %v", err)
		}
		taxonomyDest := filepath.Join(ctx.WorkDir, "taxonomy.json")
		taxData, err := os.ReadFile(filepath.Join(ctx.FixtureDir, "fixture-taxonomy.json"))
		if err != nil {
			ctx.T.Fatalf("reading fixture taxonomy: %v", err)
		}
		if err := os.WriteFile(taxonomyDest, taxData, 0o644); err != nil {
			ctx.T.Fatalf("writing taxonomy: %v", err)
		}
		blocker := filepath.Join(ctx.WorkDir, "blocker")
		if err := os.WriteFile(blocker, []byte("x"), 0o644); err != nil {
			ctx.T.Fatalf("writing blocker file: %v", err)
		}
		if err := domain.SetupBinaryConfig(ctx, map[string]interface{}{
			"classifications_path": filepath.Join(ctx.WorkDir, "classifications.jsonl"),
			"expenses_log_path":    filepath.Join(blocker, "expenses_log.jsonl"),
			"taxonomy_path":        taxonomyDest,
		}); err != nil {
			ctx.T.Fatalf("SetupBinaryConfig: %v", err)
		}
	}
}

func commandFailedWithHint() []func(*harness.Context) {
	return []func(*harness.Context){
		verify.CommandFailed(),
		verify.OutputContains("Hint:"),
	}
}
