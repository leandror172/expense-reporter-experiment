//go:build acceptance

package acceptance_test

import (
	"path/filepath"
	"testing"

	"expense-reporter/test/actions"
	"expense-reporter/test/harness"
	"expense-reporter/test/verify"
)

// TestAuto_HighConfidenceAppendsToLog asserts that auto, when the classifier returns
// HIGH confidence (≥85%), appends a confirmed expense entry to expenses_log.jsonl
// and a confirmed classification entry to classifications.jsonl — without touching a workbook.
//
// Before: auto called workflow.InsertBatchExpenses (workbook write) + logExpense.
// After: auto calls appender.ExpandAndAppend (log-append only); no workbook required.
//
// "Uber Centro" → Uber/Taxi (Transporte) at 100% confidence is stable against my-classifier-q3.
func TestAuto_HighConfidenceAppendsToLog(t *testing.T) {
	harness.RequireOllama(t, "")

	fixDir := filepath.Join(fixturesDir(), "auto-log-append")

	harness.Run(t, harness.Scenario{
		Name:  "auto HIGH confidence appends typed expense log entry without workbook",
		Given: autoLogAppendReady(fixDir),
		When:  actions.RunAuto("Uber Centro", "35,50", "15/04/2026"),
		Then: []func(*harness.Context){
			verify.CommandSucceeded(),
			verify.NoRolloverFileCreated(),
			// classifications.jsonl: confirmed entry from the Ollama classification
			verify.FeedbackContainsStatus("classifications.jsonl", "confirmed"),
			verify.FeedbackContainsItem("classifications.jsonl", "Uber Centro"),
			// expenses_log.jsonl: exactly one typed entry appended
			verify.FeedbackEntryCount("expenses_log.jsonl", 1),
			verify.FeedbackContainsItem("expenses_log.jsonl", "Uber Centro"),
		},
	})
}

// TestAuto_HighConfidenceInstallmentsExpandToNEntries asserts that auto with installment
// notation in the value (e.g. "90,00/3") expands into N dated log entries — same as add.
// Each entry carries the "(i/N)" suffix in the item name.
//
// Before: auto did not support installment notation (ParseCurrency, no expansion).
// After: ParseCurrencyWithInstallments + ExpandAndAppend emits N entries to the log.
//
// "Netflix" → Netflix (Lazer) is reliably classified at high confidence.
func TestAuto_HighConfidenceInstallmentsExpandToNEntries(t *testing.T) {
	harness.RequireOllama(t, "")

	fixDir := filepath.Join(fixturesDir(), "auto-log-append")

	harness.Run(t, harness.Scenario{
		Name:  "auto with installment value notation appends N dated log entries",
		Given: autoLogAppendReady(fixDir),
		When:  actions.RunAuto("Uber Centro", "90,00/3", "15/04/2026"),
		Then: []func(*harness.Context){
			verify.CommandSucceeded(),
			verify.NoRolloverFileCreated(),
			// Three installment entries should be in the log
			verify.FeedbackEntryCount("expenses_log.jsonl", 3),
		},
	})
}

// --- Given helpers ---

func autoLogAppendReady(fixDir string) func(*harness.Context) {
	return func(ctx *harness.Context) {
		ctx.BinaryPath = binaryPath
		ctx.DataDir = dataDir
		ctx.FixtureDir = fixDir
		withFeedbackAndTaxonomyConfig(ctx, fixDir)
	}
}
