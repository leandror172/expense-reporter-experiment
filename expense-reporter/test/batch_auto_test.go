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
		Name:  "batch-auto — classified.csv has 11 rows (1 header + 10 data), 8 columns",
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

func TestBatchAuto_SameYearInstallmentsExpandedInLog(t *testing.T) {
	harness.RequireOllama(t, "")

	fixtureDir := filepath.Join(fixturesDir(), "batch-auto-installments")

	harness.Run(t, harness.Scenario{
		Name:  "same-year installments expand into one dated log line each",
		Given: batchReadyForLogAppend(fixtureDir),
		When:  actions.RunBatchAutoWithInput(fixtureDir, "midyear-input.csv"),
		Then:  installmentsExpandedInLog(fixtureDir),
	})
}

func TestBatchAuto_CrossYearInstallmentsLoggedNotRolledOver(t *testing.T) {
	harness.RequireOllama(t, "")

	fixtureDir := filepath.Join(fixturesDir(), "batch-auto-rollover")

	harness.Run(t, harness.Scenario{
		Name:  "cross-year installments are logged with their real next-year date — no rollover.csv",
		Given: batchReadyForLogAppend(fixtureDir),
		When:  actions.RunBatchAutoWithFixture(fixtureDir),
		Then:  crossYearInstallmentsLoggedNotRolledOver(fixtureDir),
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
		withFeedbackAndTaxonomyConfig(ctx, fixDir) // T-13: batch-auto requires a configured taxonomy
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
		withFeedbackAndTaxonomyConfig(ctx, fixDir) // T-13: batch-auto requires a configured taxonomy
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
		withFeedbackAndTaxonomyConfig(ctx, fixDir) // T-13: batch-auto requires a configured taxonomy
		outDir := filepath.Join(ctx.WorkDir, "out")
		if err := os.MkdirAll(outDir, 0o755); err != nil {
			ctx.T.Fatalf("mkdir out: %v", err)
		}
		ctx.Artifacts["outDir"] = outDir
	}
}

// batchReadyForLogAppend sets up a batch-auto run for the log-append world: fixture +
// taxonomy config, no workbook. Used by the installment and cross-year scenarios, which
// now assert against expenses_log.jsonl rather than workbook rows.
func batchReadyForLogAppend(fixtureDir string) func(*harness.Context) {
	return func(ctx *harness.Context) {
		ctx.BinaryPath = binaryPath
		ctx.DataDir = dataDir
		ctx.FixtureDir = fixtureDir
		if err := harness.CopyFixtureToWorkDir(ctx, fixtureDir); err != nil {
			ctx.T.Fatalf("CopyFixtureToWorkDir: %v", err)
		}
		withFeedbackAndTaxonomyConfig(ctx, fixtureDir)
	}
}

func classifiedAndReviewFilesProduced() []func(*harness.Context) {
	return []func(*harness.Context){
		verify.CommandSucceeded(),
		verify.OutputFileExists("classified.csv"),
		verify.OutputFileExists("review.csv"),
		verify.OutputFileHasAtLeastRows("classified.csv", 1),
		verify.OutputFileHasColumns("classified.csv", 8),
		verify.AllClassificationScoresValid("classified.csv"),
	}
}

func allInputExpensesClassified(rows int) []func(*harness.Context) {
	return []func(*harness.Context){
		verify.CommandSucceeded(),
		verify.OutputFileExists("classified.csv"),
		verify.OutputFileExists("review.csv"),
		verify.OutputFileHasRows("classified.csv", rows),
		verify.OutputFileHasColumns("classified.csv", 8),
		verify.AllClassificationScoresValid("classified.csv"),
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
		verify.ExpenseLogMatches(filepath.Join(fixDir, "expected-expenses_log.jsonl")),
	}
}

// crossYearInstallmentsLoggedNotRolledOver verifies the rollover-retirement: installments
// crossing the year boundary are logged as normal lines carrying their real next-year date,
// and NO rollover.csv is produced. expected-expenses_log.jsonl pins all N lines, including
// the next-year ones (e.g. 01/01/2027).
func crossYearInstallmentsLoggedNotRolledOver(fixDir string) []func(*harness.Context) {
	return []func(*harness.Context){
		verify.CommandSucceeded(),
		verify.NoRolloverFileCreated(),
		verify.ExpenseLogMatches(filepath.Join(fixDir, "expected-expenses_log.jsonl")),
	}
}

func classificationMatchesExpectedWithMinAccuracy(expectedPath, resultsDir string) []func(*harness.Context) {
	return []func(*harness.Context){
		verify.CommandSucceeded(),
		verify.ClassificationAccuracyAtLeast("classified.csv", expectedPath, 0.5, resultsDir),
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
		Name:  "unwritable expense log fails fast before classification",
		Given: batchSubmittedWithUnwritableLogPath(fixDir),
		When:  actions.RunBatchAutoWithFixture(fixDir),
		Then:  commandFailedWithHint(),
	})
}

// batchSubmittedWithUnwritableLogPath points expenses_log_path at a file whose PARENT is
// itself a regular file, so the pre-flight's MkdirAll fails and the run aborts. Taxonomy is
// configured (it loads before the pre-flight) but its contents are irrelevant — no row is
// ever classified. Self-contained config (SetupBinaryConfig replaces the whole file).
func batchSubmittedWithUnwritableLogPath(fixDir string) func(*harness.Context) {
	return func(ctx *harness.Context) {
		ctx.BinaryPath = binaryPath
		ctx.DataDir = dataDir
		ctx.FixtureDir = fixDir
		if err := harness.CopyFixtureToWorkDir(ctx, fixDir); err != nil {
			ctx.T.Fatalf("CopyFixtureToWorkDir: %v", err)
		}
		taxonomyDest := filepath.Join(ctx.WorkDir, "taxonomy.json")
		taxData, err := os.ReadFile(filepath.Join(fixDir, "fixture-taxonomy.json"))
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
		if err := harness.SetupBinaryConfig(ctx, map[string]interface{}{
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
