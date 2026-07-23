//go:build acceptance

package acceptance_test

import (
	"path/filepath"
	"testing"

	"expense-reporter/test/actions"
	"expense-reporter/test/extern"
	"github.com/leandror172/acceptance-harness/harness"
	"github.com/leandror172/acceptance-harness/verify"
)

func TestAuto_KnownExpenseIsClassifiedWithConfidence(t *testing.T) {
	extern.RequireOllama(t, "")

	fixDir := filepath.Join(fixturesDir(), "auto-basic")

	harness.Run(t, harness.Scenario{
		Name:  "Uber Centro classified as transport with confidence score",
		Given: taxonomyAuthoredWithTrainingData(fixDir),
		When:  actions.RunAuto("Uber Centro", "35,50", "15/04"),
		Then:  expenseClassifiedWithConfidence(),
	})
}

func TestAuto_AmbiguousExpenseKeptForManualReview(t *testing.T) {
	extern.RequireOllama(t, "")

	fixDir := filepath.Join(fixturesDir(), "auto-basic")

	harness.Run(t, harness.Scenario{
		Name:  "vague expense description must not be auto-inserted",
		Given: taxonomyAuthoredWithTrainingData(fixDir),
		When:  actions.RunAuto("Outros gastos aleatorios xyz", "10.00", "01/01"),
		Then:  expenseKeptForManualReview(),
	})
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
		verify.OutputNotContains("✓ Appended",
			"vague expense should not be auto-appended (may go to review or fail resolution gracefully)"),
	}
}
