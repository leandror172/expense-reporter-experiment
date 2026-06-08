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

func TestApply_IdempotencyAndFeedback(t *testing.T) {
	fixDir := filepath.Join(fixturesDir(), "apply-basic")

	harness.Run(t, harness.Scenario{
		Name:  "apply command handles idempotency and feedback correctly",
		Given: expensesAutoInsertedBeforeReview(fixDir),
		When:  actions.RunApply(filepath.Join(fixDir, "reviewed.json")),
		Then: slices.Concat(
			commandSucceeded(),
			correctionsLoggedForAlreadyInserted(fixDir),
			noNewExpensesInserted(),
			summaryMentionsCorrections(),
		),
	})
}

// --- Given helpers (Event Modeling style — past-tense events that happened) ---

func expensesAutoInsertedBeforeReview(fixDir string) func(*harness.Context) {
	return func(ctx *harness.Context) {
		ctx.BinaryPath = binaryPath
		ctx.DataDir = dataDir
		ctx.FixtureDir = fixDir
		withFeedbackConfig(ctx)
		if err := harness.SeedFileFromFixture(ctx, fixDir, "seed-classifications.jsonl", "classifications.jsonl"); err != nil {
			ctx.T.Fatalf("SeedFileFromFixture: %v", err)
		}
	}
}

// --- Then helpers (composable) ---

func correctionsLoggedForAlreadyInserted(fixDir string) []func(*harness.Context) {
	return []func(*harness.Context){
		verify.ClassificationsMatch(filepath.Join(fixDir, "expected-feedback.jsonl")),
	}
}

func noNewExpensesInserted() []func(*harness.Context) {
	return []func(*harness.Context){
		verify.ExpenseLogNotCreated(),
	}
}

func summaryMentionsCorrections() []func(*harness.Context) {
	return []func(*harness.Context){
		verify.OutputContains("workbook not updated"),
	}
}
