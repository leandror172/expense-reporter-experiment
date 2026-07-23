//go:build acceptance

package acceptance_test

import (
	"path/filepath"
	"slices"
	"testing"

	"expense-reporter/test/actions"
	"expense-reporter/test/domain"
	"expense-reporter/test/expect"
	"expense-reporter/test/extern"
	"github.com/leandror172/acceptance-harness/harness"
	"github.com/leandror172/acceptance-harness/verify"
)

func TestAuto_FeedbackLoggedOnInsert(t *testing.T) {
	extern.RequireOllama(t, "")

	fixDir := filepath.Join(fixturesDir(), "auto-basic")

	harness.Run(t, harness.Scenario{
		Name:  "auto command logs confirmed feedback entry on successful append",
		Given: knownExpenseNotYetLogged(fixDir),
		When:  actions.RunAuto("Posto Ipiranga", "35,50", "15/04/2026"),
		Then: slices.Concat(
			autoAppendSucceeded(),
			classificationsMatchExpected(fixDir),
			expenseLogMatchesExpected(fixDir),
		),
	})
}

func TestBatchAuto_FeedbackLoggedForAppendedRows(t *testing.T) {
	extern.RequireOllama(t, "")

	fixDir := filepath.Join(fixturesDir(), "batch-auto-feedback")

	harness.Run(t, harness.Scenario{
		Name:  "batch-auto logs confirmed feedback for all auto-appended rows",
		Given: knownExpenseBatchSubmittedForClassification(fixDir),
		When:  actions.RunBatchAutoWithFixture(fixDir),
		Then: slices.Concat(
			commandSucceeded(),
			classificationsMatchExpected(fixDir),
			expenseLogMatchesExpected(fixDir),
		),
	})
}

func TestBatchAuto_DryRunNoFeedbackLogged(t *testing.T) {
	extern.RequireOllama(t, "")

	// batch-auto-basic has --dry-run in extra_args — no workbook needed
	fixDir := filepath.Join(fixturesDir(), "batch-auto-basic")

	harness.Run(t, harness.Scenario{
		Name:  "batch-auto dry-run does not create feedback log",
		Given: tenMixedExpensesSubmittedForClassification(fixDir),
		When:  actions.RunBatchAutoWithFixture(fixDir),
		Then: slices.Concat(
			commandSucceeded(),
			noLogsCreated(),
		),
	})
}

func TestAdd_ManualFeedbackLogged(t *testing.T) {
	fixDir := filepath.Join(fixturesDir(), "add-feedback")

	harness.Run(t, harness.Scenario{
		Name:  "add command logs manual feedback entry",
		Given: taxonomyAuthoredWithoutTrainingData(fixDir),
		When:  actions.RunAdd("Padaria Maeda;15/03/2026;27,50;Padaria"),
		Then: slices.Concat(
			commandSucceeded(),
			classificationsMatchExpected(fixDir),
			expenseLogMatchesExpected(fixDir),
		),
	})
}

// --- Given helpers ---

// knownExpenseNotYetLogged: an expense the keyword dictionary already knows (so the
// agreement gate will pass) has never been logged.
func knownExpenseNotYetLogged(fixDir string) func(*harness.Context) {
	return taxonomyAuthoredWithTrainingData(fixDir)
}

// knownExpenseBatchSubmittedForClassification submits a batch whose expenses the
// keyword dictionary already knows, so the agreement gate passes and rows are
// appended — the precondition for asserting feedback was logged for them.
func knownExpenseBatchSubmittedForClassification(fixDir string) func(*harness.Context) {
	return expenseBatchSubmittedForClassification(fixDir)
}

// withFeedbackConfig writes binary config with classifications_path + expenses_log_path and registers both artifacts.
func withFeedbackConfig(ctx *harness.Context) {
	classificationsPath := filepath.Join(ctx.WorkDir, "classifications.jsonl")
	expensesLogPath := filepath.Join(ctx.WorkDir, "expenses_log.jsonl")
	if err := domain.SetupBinaryConfig(ctx, map[string]interface{}{
		"classifications_path": classificationsPath,
		"expenses_log_path":    expensesLogPath,
	}); err != nil {
		ctx.T.Fatalf("SetupBinaryConfig: %v", err)
	}
	ctx.Artifacts["classifications.jsonl"] = classificationsPath
	ctx.Artifacts["expenses_log.jsonl"] = expensesLogPath
}

// --- Then helpers (composable) ---

func commandSucceeded() []func(*harness.Context) {
	return []func(*harness.Context){
		verify.CommandSucceeded(),
	}
}

func autoAppendSucceeded() []func(*harness.Context) {
	return []func(*harness.Context){
		verify.CommandSucceeded(),
		verify.OutputContains("✓ Appended"),
	}
}

func classificationsMatchExpected(fixDir string) []func(*harness.Context) {
	return []func(*harness.Context){
		expect.ClassificationsMatch(filepath.Join(fixDir, "expected-feedback.jsonl")),
	}
}

func expenseLogMatchesExpected(fixDir string) []func(*harness.Context) {
	return []func(*harness.Context){
		expect.ExpenseLogMatches(filepath.Join(fixDir, "expected-expenses_log.jsonl")),
	}
}

func noLogsCreated() []func(*harness.Context) {
	return []func(*harness.Context){
		expect.ClassificationsNotCreated(),
		expect.ExpenseLogNotCreated(),
	}
}

// --- Tests: add with prediction flags (MCP-layer corrections) ---

// TestAdd_ConfirmedFeedbackWhenPredictionMatches covers the Telegram flow where the user
// accepted the model's top candidate — add writes confirmed feedback (same as auto auto-accept).
func TestAdd_ConfirmedFeedbackWhenPredictionMatches(t *testing.T) {
	fixDir := filepath.Join(fixturesDir(), "add-with-prediction-match")

	harness.Run(t, harness.Scenario{
		Name:  "add with --predicted-subcategory matching chosen subcategory logs confirmed feedback",
		Given: expenseClassifiedByModel(fixDir),
		When: actions.RunAdd(
			"Uber Centro;15/04/2026;35,50;Uber/Taxi",
			"--predicted-subcategory", "Uber/Taxi",
			"--predicted-category", "Transporte",
			"--confidence", "0.92",
			"--model", "my-classifier-q3",
		),
		Then: slices.Concat(
			commandSucceeded(),
			classificationsMatchExpected(fixDir),
			expenseLogMatchesExpected(fixDir),
		),
	})
}

// TestAdd_CorrectedFeedbackWhenPredictionMismatches covers the Telegram flow where the user
// rejected the top candidate and picked a different subcategory.
func TestAdd_CorrectedFeedbackWhenPredictionMismatches(t *testing.T) {
	fixDir := filepath.Join(fixturesDir(), "add-with-prediction-mismatch")

	harness.Run(t, harness.Scenario{
		Name:  "add with --predicted-subcategory differing from chosen subcategory logs corrected feedback",
		Given: expenseClassifiedByModel(fixDir),
		When: actions.RunAdd(
			"Uber Centro;15/04/2026;35,50;Combustível",
			"--predicted-subcategory", "Uber/Taxi",
			"--predicted-category", "Transporte",
			"--confidence", "0.92",
			"--model", "my-classifier-q3",
		),
		Then: slices.Concat(
			commandSucceeded(),
			classificationsMatchExpected(fixDir),
			expenseLogMatchesExpected(fixDir),
		),
	})
}

// TestAdd_ManualFeedbackWithoutPredictionFlags is a backwards-compat check:
// add without prediction flags must continue to write a manual entry, not a confirmed/corrected one.
// Note: this scenario is also covered by TestAdd_ManualFeedbackLogged in the same file —
// it is duplicated here as an explicit regression guard for the new flag-branching logic.
func TestAdd_ManualFeedbackWithoutPredictionFlags(t *testing.T) {
	fixDir := filepath.Join(fixturesDir(), "add-feedback")

	harness.Run(t, harness.Scenario{
		Name:  "add without prediction flags continues to write manual feedback entry",
		Given: taxonomyAuthoredWithoutTrainingData(fixDir),
		When:  actions.RunAdd("Padaria Maeda;15/03/2026;27,50;Padaria"),
		Then: slices.Concat(
			commandSucceeded(),
			classificationsMatchExpected(fixDir),
			expenseLogMatchesExpected(fixDir),
		),
	})
}

// expenseClassifiedByModel: the model already produced a prediction for this expense, arriving on the add command as --predicted-* flags.
func expenseClassifiedByModel(fixDir string) func(*harness.Context) {
	return taxonomyAuthoredWithTrainingData(fixDir)
}
