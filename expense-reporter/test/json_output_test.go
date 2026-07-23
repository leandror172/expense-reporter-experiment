//go:build acceptance

package acceptance_test

import (
	"slices"
	"testing"

	"expense-reporter/test/actions"
	"expense-reporter/test/expect"
	"expense-reporter/test/extern"
	"github.com/leandror172/acceptance-harness/harness"
	"github.com/leandror172/acceptance-harness/verify"
)

// TestClassifyJSON_ReturnsValidJSONWithCandidates verifies that classify --json
// produces valid JSON output with the expected top-level keys.
func TestClassifyJSON_ReturnsValidJSONWithCandidates(t *testing.T) {
	extern.RequireOllama(t, "")

	harness.Run(t, harness.Scenario{
		Name:  "classify --json returns valid JSON with candidates array",
		Given: taxonomyAuthoredWithTrainingData(jsonOutputFixture()),
		When:  actions.RunClassify("--json", "Uber Centro", "35,50", "15/04"),
		Then: slices.Concat(
			thenJSONSucceeded(),
			thenJSONHasCandidates(),
		),
	})
}

// TestAutoJSON_ReturnsRecommendationWithoutInserting verifies that auto --json
// returns a recommendation (action field) but never inserts into the workbook.
func TestAutoJSON_ReturnsRecommendationWithoutInserting(t *testing.T) {
	extern.RequireOllama(t, "")

	harness.Run(t, harness.Scenario{
		Name:  "auto --json returns action recommendation without inserting",
		Given: taxonomyAuthoredWithTrainingData(jsonOutputFixture()),
		When:  actions.RunAuto("--json", "Uber Centro", "35,50", "15/04"),
		Then: slices.Concat(
			thenJSONSucceeded(),
			thenJSONHasActionAndCandidates(),
			thenJSONModeDidNotInsert(),
		),
	})
}

// TestAddDryRunJSON_ReturnsValidJSONWithAction verifies that add --dry-run --json
// produces valid JSON with parsed expense fields and "would_insert" action.
// Does NOT require Ollama — no classification involved.
func TestAddDryRunJSON_ReturnsValidJSONWithAction(t *testing.T) {
	harness.Run(t, harness.Scenario{
		Name:  "add --dry-run --json returns valid JSON with would_insert action",
		Given: taxonomyAuthoredWithoutTrainingData(jsonOutputFixture()),
		When:  actions.RunAddDryRun("Uber Centro;15/04;35,50;Uber/Taxi", "--json"),
		Then: slices.Concat(
			thenJSONSucceeded(),
			thenJSONHasExpenseFields(),
			thenJSONActionIs("would_insert"),
			thenDryRunDidNotInsert(),
		),
	})
}

// TestAddDryRunJSON_ResolvesCategory verifies that add --dry-run --json resolves
// the parent category from taxonomy when --data-dir is provided.
func TestAddDryRunJSON_ResolvesCategory(t *testing.T) {
	harness.Run(t, harness.Scenario{
		Name:  "add --dry-run --json resolves category from taxonomy",
		Given: taxonomyAuthoredWithTrainingData(jsonOutputFixture()),
		When:  actions.RunAddDryRun("Uber Centro;15/04;35,50;Uber/Taxi", "--json"),
		Then: slices.Concat(
			thenJSONSucceeded(),
			thenJSONCategoryIs("Transporte"),
		),
	})
}

// TestAddDryRunJSON_SurfacesResolvedType verifies that add --dry-run --json now
// emits the expense type it already resolves from the taxonomy full path (T-13),
// not just the category. Uber/Taxi is an unambiguous leaf under Variáveis/Transporte,
// so the resolved type is deterministic — no Ollama involved.
func TestAddDryRunJSON_SurfacesResolvedType(t *testing.T) {
	harness.Run(t, harness.Scenario{
		Name:  "add --dry-run --json surfaces the type resolved from taxonomy",
		Given: taxonomyAuthoredWithTrainingData(jsonOutputFixture()),
		When:  actions.RunAddDryRun("Uber Centro;15/04;35,50;Uber/Taxi", "--json"),
		Then: slices.Concat(
			thenJSONSucceeded(),
			thenJSONTypeIs("Variáveis"),
		),
	})
}

// TestAdd_ReadsBrazilianThousandsAmount pins the documented BR value format at
// the CLI contract: "1.234,56" (dot thousands + comma decimal) is understood as
// 1234.56. Unsupported anywhere before the T-41 boundary — the old parser's
// naive comma→dot swap produced "1.234.56" and errored.
func TestAdd_ReadsBrazilianThousandsAmount(t *testing.T) {
	harness.Run(t, harness.Scenario{
		Name:  "add --dry-run --json parses BR thousands-separator value",
		Given: taxonomyAuthoredWithTrainingData(jsonOutputFixture()),
		When:  actions.RunAddDryRun("Passagem Aérea;15/04/2026;1.234,56;Uber/Taxi", "--json"),
		Then: slices.Concat(
			thenJSONSucceeded(),
			thousandsAmountReadAs(1234.56),
		),
	})
}

// --- Then helpers ---

// thenJSONSucceeded is the base for all JSON output tests: command exits 0 and stdout is valid JSON.
// Builds on commandSucceeded() from feedback_test.go (same package).
func thenJSONSucceeded() []func(*harness.Context) {
	return slices.Concat(
		commandSucceeded(),
		[]func(*harness.Context){verify.OutputIsValidJSON()},
	)
}

// thenJSONHasExpenseFields checks the four core fields present in any add-command JSON output.
func thenJSONHasExpenseFields() []func(*harness.Context) {
	return []func(*harness.Context){
		verify.OutputJSONHasKey("item"),
		verify.OutputJSONHasKey("value"),
		verify.OutputJSONHasKey("date"),
		verify.OutputJSONHasKey("subcategory"),
	}
}

// thenJSONHasCandidates checks the shape of classify --json output.
func thenJSONHasCandidates() []func(*harness.Context) {
	return []func(*harness.Context){
		verify.OutputJSONHasKey("item"),
		verify.OutputJSONHasKey("candidates"),
	}
}

// thenJSONHasActionAndCandidates checks the shape of auto --json output.
func thenJSONHasActionAndCandidates() []func(*harness.Context) {
	return []func(*harness.Context){
		verify.OutputJSONHasKey("action"),
		verify.OutputJSONHasKey("candidates"),
	}
}

// thenJSONActionIs checks that the action field is present and has the expected value.
func thenJSONActionIs(action string) []func(*harness.Context) {
	return []func(*harness.Context){
		verify.OutputJSONHasKey("action"),
		expect.OutputJSONHasAction(action),
	}
}

// thenJSONCategoryIs checks that the category field has the expected value.
func thenJSONCategoryIs(category string) []func(*harness.Context) {
	return []func(*harness.Context){
		expect.OutputJSONHasCategory(category),
	}
}

// thenJSONTypeIs checks that the surfaced expense type has the expected value.
func thenJSONTypeIs(typ string) []func(*harness.Context) {
	return []func(*harness.Context){
		expect.OutputJSONHasType(typ),
	}
}

// thousandsAmountReadAs asserts the BR thousands-formatted input amount was
// understood as the given decimal value.
func thousandsAmountReadAs(value float64) []func(*harness.Context) {
	return []func(*harness.Context){
		verify.OutputJSONHasValue("value", value),
	}
}

// thenJSONModeDidNotInsert asserts --json mode never writes to the workbook.
func thenJSONModeDidNotInsert() []func(*harness.Context) {
	return []func(*harness.Context){
		verify.OutputNotContains("✓ Appended", "JSON mode must not append to the expense log"),
	}
}

// thenDryRunDidNotInsert asserts --dry-run mode never writes to the workbook.
func thenDryRunDidNotInsert() []func(*harness.Context) {
	return []func(*harness.Context){
		verify.OutputNotContains("✓ Expense added", "Dry-run mode must not insert"),
	}
}
