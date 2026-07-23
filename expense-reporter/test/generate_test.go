//go:build acceptance

package acceptance_test

import (
	"path/filepath"
	"slices"
	"testing"

	"expense-reporter/test/actions"
	"expense-reporter/test/expect"
	"github.com/leandror172/acceptance-harness/harness"
	"github.com/leandror172/acceptance-harness/verify"
)

// These tests are the Phase G3 acceptance contract for the generate-workbook command
// and were born RED (command not implemented yet); expected dumps were frozen from
// the scratch template-builder oracle.
func TestGenerateWorkbook_Skeleton(t *testing.T) {
	fixDir := filepath.Join(fixturesDir(), "generate-basic")

	harness.Run(t, harness.Scenario{
		Name:    "generate-workbook command produces skeleton structure when no entries provided",
		Fixture: fixDir,
		When:    actions.RunGenerateWorkbook(filepath.Join(fixDir, "taxonomy.json"), "", "--year", "2026"),
		Then: slices.Concat(
			commandSucceeded(),
			skeletonStructureGenerated(fixDir),
		),
	})
}

func TestGenerateWorkbook_WithEntries(t *testing.T) {
	fixDir := filepath.Join(fixturesDir(), "generate-basic")

	harness.Run(t, harness.Scenario{
		Name:    "generate-workbook command produces data-bearing structure when entries are provided",
		Fixture: fixDir,
		When:    actions.RunGenerateWorkbook(filepath.Join(fixDir, "taxonomy.json"), filepath.Join(fixDir, "entries.jsonl"), "--year", "2026"),
		Then: slices.Concat(
			commandSucceeded(),
			dataBearingStructureGenerated(fixDir),
		),
	})
}

func TestGenerateWorkbook_UnmappedSubcategorySkipped(t *testing.T) {
	fixDir := filepath.Join(fixturesDir(), "generate-basic")

	harness.Run(t, harness.Scenario{
		Name:    "generate-workbook command skips unmapped subcategories and warns about them",
		Fixture: fixDir,
		When:    actions.RunGenerateWorkbook(filepath.Join(fixDir, "taxonomy.json"), filepath.Join(fixDir, "entries-with-unmapped.jsonl"), "--year", "2026"),
		Then: slices.Concat(
			commandSucceeded(),
			unmappedEntryWarnedAndSkipped(),
		),
	})
}

func TestGenerateWorkbook_MultiYearLogFiltersToYear(t *testing.T) {
	fixDir := filepath.Join(fixturesDir(), "generate-basic")

	harness.Run(t, harness.Scenario{
		Name:    "generate-workbook command filters multi-year log to target year",
		Fixture: fixDir,
		When:    actions.RunGenerateWorkbook(filepath.Join(fixDir, "taxonomy.json"), filepath.Join(fixDir, "entries-multiyear.jsonl"), "--year", "2026"),
		Then: slices.Concat(
			commandSucceeded(),
			dataBearingStructureGenerated(fixDir),
		),
	})
}

// --- Given helpers (Event Modeling style — past-tense events that happened) ---

// --- Then helpers (composable) ---

func skeletonStructureGenerated(fixDir string) []func(*harness.Context) {
	return []func(*harness.Context){
		expect.WorkbookStructureMatches(filepath.Join(fixDir, "expected-dump-skeleton")),
	}
}

func dataBearingStructureGenerated(fixDir string) []func(*harness.Context) {
	return []func(*harness.Context){
		expect.WorkbookStructureMatches(filepath.Join(fixDir, "expected-dump-data")),
	}
}

func unmappedEntryWarnedAndSkipped() []func(*harness.Context) {
	return []func(*harness.Context){
		verify.OutputContains("Esportes"),
	}
}
