//go:build acceptance

package acceptance_test

import (
	"path/filepath"
	"slices"
	"testing"

	"expense-reporter/test/actions"
	"expense-reporter/test/harness"
	"expense-reporter/test/verify"
)

// This test is RED by design: `--income-entries` flag is not implemented; oracle freeze is WS-C Step 5.
// Fixture arithmetic: Salário block Jan=4150, Fev=4450; Férias block Jul=7600
func TestGenerateWorkbook_IncomeRoute(t *testing.T) {
	fixDir := filepath.Join(fixturesDir(), "generate-income")

	harness.Run(t, harness.Scenario{
		Name: "generate-workbook command produces income route structure when entries are provided",
		Given: incomeEntriesRecorded(fixDir),
		When:  actions.RunGenerateWorkbook(filepath.Join(fixDir, "taxonomy.json"), filepath.Join(fixDir, "entries.jsonl"), "--year", "2026", "--income-entries", filepath.Join(fixDir, "income-entries.jsonl")),
		Then:  slices.Concat(
			commandSucceeded(),
			incomeRouteStructureGenerated(fixDir),
		),
	})
}

// --- Given helpers (Event Modeling style — past-tense events that happened) ---

func incomeEntriesRecorded(fixDir string) func(*harness.Context) {
	return func(ctx *harness.Context) {
		ctx.BinaryPath = binaryPath
		ctx.FixtureDir = fixDir
	}
}

// --- Then helpers (composable) ---

func incomeRouteStructureGenerated(fixDir string) []func(*harness.Context) {
	return []func(*harness.Context){
		verify.WorkbookStructureMatches(filepath.Join(fixDir, "expected-dump-data")), // This will fail because the flag doesn't exist
	}
}
