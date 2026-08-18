//go:build acceptance

package acceptance_test

import (
	"path/filepath"
	"slices"
	"testing"

	"expense-reporter/test/actions"
	"expense-reporter/test/expect"

	"github.com/leandror172/acceptance-harness/harness"
)

// A batch whose every row is malformed still SUCCEEDS, and leaves behind a failed.csv
// the human can repair and re-run. Before T-63 those rows reached no durable artifact
// at all: they were named on stderr and then lost.
//
// Deterministic and Ollama-free by construction — the parse precedes classification, so
// a batch of only-bad rows never calls the model. Same reason batch-auto-resume-all-seeded
// is deterministic. Do NOT add a parseable row to the fixture; it would call the model and
// move this scenario out of the fast group. See the fixture README.
func TestBatchAuto_UnparseableRowsAreWrittenToAFileTheHumanCanRepair(t *testing.T) {
	fixDir := filepath.Join(fixturesDir(), "batch-auto-failed-rows")

	harness.Run(t, harness.Scenario{
		Name:    "a batch of only-malformed rows still succeeds and reports each rejection",
		Fixture: fixDir,
		Given:   expenseBatchSubmittedForClassification(),
		When:    actions.RunBatchAutoWithFixture(),
		Then: slices.Concat(
			commandSucceeded(),
			everyRejectedLineListedWithItsReason(),
		),
	})
}

// everyRejectedLineListedWithItsReason pins the repair contract: each rejected row is
// reproduced verbatim, in input order, with the reason on the same line. The expected
// lines are the four real rejects from the first monthly close on 2026 data.
func everyRejectedLineListedWithItsReason() []func(*harness.Context) {
	return []func(*harness.Context){
		expect.FailedRowsCarryTheirReason("failed.csv", []string{
			"Anita;Elô ADM;09/01;405,25 - 1/4",
			"Comida cinema;109,39;14/01",
			"Café padaria;201/01;13,00",
			"Anita compra chocolate Ruby 299,00 e cacau 49,90;646,25 4x",
		}),
	}
}
