//go:build acceptance

package acceptance_test

import (
	"path/filepath"
	"slices"
	"testing"

	"expense-reporter/test/actions"
	"expense-reporter/test/extern"
	"github.com/leandror172/acceptance-harness/harness"
)

func TestBatchAuto_TypeEmittedInExpenseLog(t *testing.T) {
	extern.RequireOllama(t, "")

	fixDir := filepath.Join(fixturesDir(), "batch-auto-typed")

	harness.Run(t, harness.Scenario{
		Name:    "batch-auto with taxonomy logs entries with correct type field",
		Fixture: fixDir,
		Given:   typedExpenseBatchSubmittedForClassification(),
		When:    actions.RunBatchAutoWithFixture(fixDir),
		Then: slices.Concat(
			commandSucceeded(),
			classificationsMatchExpected(fixDir),
			expenseLogMatchesExpected(fixDir),
		),
	})
}

// typedExpenseBatchSubmittedForClassification is the canonical non-dry-run append
// anchor: the submitted batch carries expenses whose type the log must record. No
// workbook is involved (the log-append path needs none). This is the load-bearing
// coverage of the append path that the dry-run survivors do not provide.
func typedExpenseBatchSubmittedForClassification() func(*harness.Context) {
	return expenseBatchSubmittedForClassification()
}
