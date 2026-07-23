//go:build acceptance

package acceptance_test

import (
	"path/filepath"
	"slices"
	"testing"

	"expense-reporter/test/actions"
	"expense-reporter/test/domain"
	"github.com/leandror172/acceptance-harness/harness"
	"github.com/leandror172/acceptance-harness/verify"
)

func TestCorrect_LogsCorrectedEntryWhenPredictionExists(t *testing.T) {
	fixDir := filepath.Join(fixturesDir(), "correct-overrides-confirmed")

	harness.Run(t, harness.Scenario{
		Name:    "correct command logs corrected entry when prediction exists",
		Fixture: fixDir,
		Given:   expenseAutoConfirmed(),
		When:    actions.RunCorrect("Uber Centro;15/04;35,50;Combustível"),
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
		Name:    "correct command uses most recent prediction when id repeats",
		Fixture: fixDir,
		Given:   expenseConfirmedThenCorrected(),
		When:    actions.RunCorrect("Uber Centro;15/04;35,50;Combustível"),
		Then: slices.Concat(
			commandSucceeded(),
			classificationsMatchExpected(fixDir),
		),
	})
}

// --- Given helpers (Event Modeling style — past-tense events that happened) ---

// priorClassificationsRecorded: the classification log already holds the history the
// fixture seeds. WHICH history is the fixture's business — see the wrappers below.
// Counterpart of noClassificationsRecorded().
//
// T-13: correct resolves the corrected entry's category from the taxonomy
// (CategoryForLeaf), so a taxonomy must be published for the success path.
func priorClassificationsRecorded() func(*harness.Context) {
	return harness.Events(
		taxonomyAuthoredWithTrainingData(),
		classificationsSeededFromFixture(),
	)
}

// classificationsSeededFromFixture: the fixture's seed file becomes the starting
// classifications.jsonl.
func classificationsSeededFromFixture() func(*harness.Context) {
	return func(ctx *harness.Context) {
		if err := harness.SeedFileFromFixture(ctx, ctx.FixtureDir, "seed-classifications.jsonl", "classifications.jsonl"); err != nil {
			ctx.T.Fatalf("SeedFileFromFixture: %v", err)
		}
	}
}

// expenseAutoConfirmed: the seeded history is one auto-confirmed classification.
func expenseAutoConfirmed() func(*harness.Context) {
	return priorClassificationsRecorded()
}

// expenseConfirmedThenCorrected: the seeded history is a confirmation followed by a
// correction of the same expense, so the id repeats across entries.
func expenseConfirmedThenCorrected() func(*harness.Context) {
	return priorClassificationsRecorded()
}

func noClassificationsRecorded() func(*harness.Context) {
	return func(ctx *harness.Context) {
		domain.SetDataDir(ctx, dataDir)
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
