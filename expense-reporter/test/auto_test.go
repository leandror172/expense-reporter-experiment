//go:build acceptance

package acceptance_test

import (
	"path/filepath"
	"testing"

	"expense-reporter/test/actions"
	"expense-reporter/test/harness"
	"expense-reporter/test/verify"
)

func TestAuto_KnownExpenseIsClassifiedWithConfidence(t *testing.T) {
	harness.RequireOllama(t, "")
	harness.RequireWorkbook(t, testWorkbook)

	fixDir := filepath.Join(fixturesDir(), "auto-basic")

	harness.Run(t, harness.Scenario{
		Name:  "Uber Centro classified as transport with confidence score",
		Given: expenseTaxonomyAvailable(fixDir),
		When:  actions.RunAuto("Uber Centro", "35,50", "15/04"),
		Then:  expenseClassifiedWithConfidence(),
	})
}

func TestAuto_AmbiguousExpenseKeptForManualReview(t *testing.T) {
	harness.RequireOllama(t, "")
	harness.RequireWorkbook(t, testWorkbook)

	harness.Run(t, harness.Scenario{
		Name:  "vague expense description must not be auto-inserted",
		Given: expenseClassifierAvailable(),
		When:  actions.RunAuto("Outros gastos aleatorios xyz", "10.00", "01/01"),
		Then:  expenseKeptForManualReview(),
	})
}

func expenseTaxonomyAvailable(fixDir string) func(*harness.Context) {
	return func(ctx *harness.Context) {
		ctx.BinaryPath = binaryPath
		ctx.FixtureDir = fixDir
		ctx.DataDir = dataDir
		if err := harness.CopyWorkbookToWorkDir(ctx, testWorkbook); err != nil {
			ctx.T.Fatalf("CopyWorkbookToWorkDir: %v", err)
		}
	}
}

func expenseClassifierAvailable() func(*harness.Context) {
	return func(ctx *harness.Context) {
		ctx.BinaryPath = binaryPath
		ctx.DataDir = dataDir
		if err := harness.CopyWorkbookToWorkDir(ctx, testWorkbook); err != nil {
			ctx.T.Fatalf("CopyWorkbookToWorkDir: %v", err)
		}
	}
}

func expenseClassifiedWithConfidence() []func(*harness.Context) {
	return []func(*harness.Context){
		verify.CommandSucceeded(),
		verify.OutputContains("%", "confidence score should be shown after classification"),
	}
}

func expenseKeptForManualReview() []func(*harness.Context) {
	return []func(*harness.Context){
		verify.CommandSucceeded(),
		verify.OutputNotContains("✓ Inserted", "vague expense should not be auto-inserted"),
	}
}
