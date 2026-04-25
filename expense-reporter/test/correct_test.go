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

func TestCorrect_LogsCorrectedEntryWhenPredictionExists(t *testing.T) {
	fixDir := filepath.Join(fixturesDir(), "correct-overrides-confirmed")

	harness.Run(t, harness.Scenario{
		Name:  "correct command logs corrected entry when prediction exists",
		Given: expenseAutoConfirmed(fixDir),
		When:  actions.RunCorrect("Uber Centro;15/04;35,50;Combustível"),
		Then: slices.Concat(
			commandSucceeded(),
			classificationsMatchExpected(fixDir),
		),
	})
}

func TestCorrect_FailsWhenNoPriorPredictionToOverride(t *testing.T) {
	harness.Run(t, harness.Scenario{
		Name:  "correct command fails when no prior prediction exists to override",
		Given: noClassificationsRecorded(),
		When:  actions.RunCorrect("Uber Centro;15/04;35,50;Combustível"),
		Then: slices.Concat(
			commandFailed(),
			correctionHintShownToUser(),
			noLogsCreated(),
		),
	})
}

func TestCorrect_UsesMostRecentPredictionWhenIdRepeats(t *testing.T) {
	fixDir := filepath.Join(fixturesDir(), "correct-uses-latest-entry")

	harness.Run(t, harness.Scenario{
		Name:  "correct command uses most recent prediction when id repeats",
		Given: expenseConfirmedThenCorrected(fixDir),
		When:  actions.RunCorrect("Uber Centro;15/04;35,50;Combustível"),
		Then: slices.Concat(
			commandSucceeded(),
			classificationsMatchExpected(fixDir),
		),
	})
}

// --- Given helpers (Event Modeling style — past-tense events that happened) ---

func expenseAutoConfirmed(fixDir string) func(*harness.Context) {
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

func expenseConfirmedThenCorrected(fixDir string) func(*harness.Context) {
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

func noClassificationsRecorded() func(*harness.Context) {
	return func(ctx *harness.Context) {
		ctx.BinaryPath = binaryPath
		ctx.DataDir = dataDir
		withFeedbackConfig(ctx)
	}
}

// --- Then helpers (composable) ---

func commandFailed() []func(*harness.Context) {
	return []func(*harness.Context){
		verify.CommandFailed(),
	}
}

func correctionHintShownToUser() []func(*harness.Context) {
	return []func(*harness.Context){
		verify.OutputContains("use 'add'"),
	}
}
