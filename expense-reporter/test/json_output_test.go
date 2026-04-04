//go:build acceptance

package acceptance_test

import (
	"slices"
	"testing"

	"expense-reporter/test/actions"
	"expense-reporter/test/harness"
	"expense-reporter/test/verify"
)

// TestClassifyJSON_ReturnsValidJSONWithCandidates verifies that classify --json
// produces valid JSON output with the expected top-level keys.
func TestClassifyJSON_ReturnsValidJSONWithCandidates(t *testing.T) {
	harness.RequireOllama(t, "")

	harness.Run(t, harness.Scenario{
		Name:  "classify --json returns valid JSON with candidates array",
		Given: classifierForJSON(),
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
	harness.RequireOllama(t, "")

	harness.Run(t, harness.Scenario{
		Name:  "auto --json returns action recommendation without inserting",
		Given: classifierForJSON(),
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
		Given: binaryOnly(),
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
		Given: classifierForJSON(),
		When:  actions.RunAddDryRun("Uber Centro;15/04;35,50;Uber/Taxi", "--json"),
		Then: slices.Concat(
			thenJSONSucceeded(),
			thenJSONCategoryIs("Transporte"),
		),
	})
}

// binaryOnly sets up context with just the binary path — no Ollama, no workbook, no data dir.
func binaryOnly() func(*harness.Context) {
	return func(ctx *harness.Context) {
		ctx.BinaryPath = binaryPath
	}
}

// classifierForJSON sets up the context for JSON output tests.
// No workbook needed since --json mode is read-only.
func classifierForJSON() func(*harness.Context) {
	return func(ctx *harness.Context) {
		ctx.BinaryPath = binaryPath
		ctx.DataDir = dataDir
	}
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
		verify.OutputJSONHasAction(action),
	}
}

// thenJSONCategoryIs checks that the category field has the expected value.
func thenJSONCategoryIs(category string) []func(*harness.Context) {
	return []func(*harness.Context){
		verify.OutputJSONHasCategory(category),
	}
}

// thenJSONModeDidNotInsert asserts --json mode never writes to the workbook.
func thenJSONModeDidNotInsert() []func(*harness.Context) {
	return []func(*harness.Context){
		verify.OutputNotContains("✓ Inserted", "JSON mode must not insert into workbook"),
	}
}

// thenDryRunDidNotInsert asserts --dry-run mode never writes to the workbook.
func thenDryRunDidNotInsert() []func(*harness.Context) {
	return []func(*harness.Context){
		verify.OutputNotContains("✓ Expense added", "Dry-run mode must not insert"),
	}
}
