//go:build acceptance

package acceptance_test

import (
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
		Then: []func(*harness.Context){
			verify.CommandSucceeded(),
			verify.OutputIsValidJSON(),
			verify.OutputJSONHasKey("item"),
			verify.OutputJSONHasKey("candidates"),
		},
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
		Then: []func(*harness.Context){
			verify.CommandSucceeded(),
			verify.OutputIsValidJSON(),
			verify.OutputJSONHasKey("action"),
			verify.OutputJSONHasKey("candidates"),
			verify.OutputNotContains("✓ Inserted", "JSON mode must not insert into workbook"),
		},
	})
}

// classifierForJSON sets up the context for JSON output tests.
// No workbook needed since --json mode is read-only.
func classifierForJSON() func(*harness.Context) {
	return func(ctx *harness.Context) {
		ctx.BinaryPath = binaryPath
		ctx.DataDir = dataDir
	}
}
