//go:build acceptance

package acceptance_test

import (
	"io"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"expense-reporter/test/actions"
	"expense-reporter/test/harness"
	"expense-reporter/test/verify"
)

// TestFewShot_ClassifyWithTrainingDataShowsFewShot verifies that when training data
// is available, the classify command injects few-shot examples and logs it under --verbose.
func TestFewShot_ClassifyWithTrainingDataShowsFewShot(t *testing.T) {
	harness.RequireOllama(t, "")

	harness.Run(t, harness.Scenario{
		Name:  "classify with training data logs few-shot injection under --verbose",
		Given: classifierWithDataDir(),
		When:  actions.RunClassify("--verbose", "Uber Centro", "35,50", "15/04"),
		Then: thenFewShotLineVisible(),
	})
}

// TestFewShot_ClassifyWithoutTrainingDataSucceeds verifies graceful degradation:
// when training_data_complete.json is absent, classify still succeeds (no error)
// and the few-shot debug line still appears (with count=0).
func TestFewShot_ClassifyWithoutTrainingDataSucceeds(t *testing.T) {
	harness.RequireOllama(t, "")

	harness.Run(t, harness.Scenario{
		Name:  "classify without training data degrades gracefully — no error, few-shot count 0",
		Given: classifierWithKeywordsOnly(),
		When:  actions.RunClassify("--verbose", "Uber Centro", "35,50", "15/04"),
		Then: slices.Concat(
			thenFewShotLineVisible(),
			thenNoTrainingErrorShown(),
		),
	})
}

// TestFewShot_BatchAutoProducesOutputFiles verifies that the batch-auto command
// with few-shot wiring still produces classified.csv and review.csv correctly.
func TestFewShot_BatchAutoProducesOutputFiles(t *testing.T) {
	harness.RequireOllama(t, "")

	fixDir := filepath.Join(fixturesDir(), "batch-auto-basic")

	harness.Run(t, harness.Scenario{
		Name:  "batch-auto with few-shot still produces classified and review CSV files",
		Given: mixedExpensesReadyForDryRun(fixDir),
		When:  actions.RunBatchAutoWithFixture(fixDir),
		Then: classifiedAndReviewFilesProduced(),
	})
}

// --- Given helpers ---

// classifierWithDataDir sets up the real data directory (training data + keyword index).
func classifierWithDataDir() func(*harness.Context) {
	return func(ctx *harness.Context) {
		ctx.BinaryPath = binaryPath
		ctx.DataDir = dataDir
	}
}

// classifierWithKeywordsOnly creates a stripped data dir containing only
// feature_dictionary_enhanced.json (no training_data_complete.json).
// This simulates a cold-start environment where training data is absent.
func classifierWithKeywordsOnly() func(*harness.Context) {
	return func(ctx *harness.Context) {
		ctx.BinaryPath = binaryPath

		// Build a temp dir with only the keyword index.
		tmpDir := ctx.T.TempDir()
		src := filepath.Join(dataDir, "feature_dictionary_enhanced.json")
		dst := filepath.Join(tmpDir, "feature_dictionary_enhanced.json")
		if err := copyFile(src, dst); err != nil {
			ctx.T.Fatalf("classifierWithKeywordsOnly: copy feature dict: %v", err)
		}
		// Intentionally omit training_data_complete.json.
		ctx.DataDir = tmpDir
	}
}

// --- Then helpers ---

// thenFewShotLineVisible asserts the command succeeded and --verbose shows few-shot injection.
func thenFewShotLineVisible() []func(*harness.Context) {
	return slices.Concat(
		commandSucceeded(),
		[]func(*harness.Context){
			verify.OutputContains("few-shot", "few-shot debug line should appear in --verbose output"),
		},
	)
}

// thenNoTrainingErrorShown asserts no error about missing training data appears in output.
func thenNoTrainingErrorShown() []func(*harness.Context) {
	return []func(*harness.Context){
		verify.OutputNotContains("training", "no error about missing training file should appear"),
	}
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	return err
}
