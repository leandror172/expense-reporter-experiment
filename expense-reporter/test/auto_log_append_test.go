//go:build acceptance

package acceptance_test

import (
	"path/filepath"
	"testing"

	"expense-reporter/test/actions"
	"expense-reporter/test/domain"
	"expense-reporter/test/expect"
	"expense-reporter/test/extern"
	"github.com/leandror172/acceptance-harness/harness"
	"github.com/leandror172/acceptance-harness/verify"
)

// TestAuto_KeywordAgreementAppendsToLog asserts that auto, when the prediction passes the
// agreement gate, appends a confirmed expense entry to expenses_log.jsonl and a confirmed
// classification entry to classifications.jsonl — without touching a workbook.
//
// Before: auto called workflow.InsertBatchExpenses (workbook write) + logExpense.
// After: auto calls appender.ExpandAndAppend (log-append only); no workbook required.
//
// "Posto Ipiranga" → Combustível (Transporte) is the canonical gate-passing item
// (keyword spec 1.0, unambiguous, model agrees — T-32). "Uber Centro" no longer works
// here: spec 0.8 + ambiguous fails the agreement gate and routes to review.
//
// Named for the AGREEMENT gate, not confidence: T-32 removed confidence from the
// auto-insert decision entirely (the 649-replay measured it uninformative), so the old
// "HighConfidence" name described a gate that no longer exists.
func TestAuto_KeywordAgreementAppendsToLog(t *testing.T) {
	extern.RequireOllama(t, "")

	fixDir := filepath.Join(fixturesDir(), "auto-log-append")

	harness.Run(t, harness.Scenario{
		Name:  "auto HIGH confidence appends typed expense log entry without workbook",
		Given: autoLogAppendReady(fixDir),
		When:  actions.RunAuto("Posto Ipiranga", "35,50", "15/04/2026"),
		Then: []func(*harness.Context){
			verify.CommandSucceeded(),
			expect.NoRolloverFileCreated(),
			// classifications.jsonl: confirmed entry from the Ollama classification
			expect.FeedbackContainsStatus("classifications.jsonl", "confirmed"),
			expect.FeedbackContainsItem("classifications.jsonl", "Posto Ipiranga"),
			// expenses_log.jsonl: exactly one typed entry appended
			expect.FeedbackEntryCount("expenses_log.jsonl", 1),
			expect.FeedbackContainsItem("expenses_log.jsonl", "Posto Ipiranga"),
		},
	})
}

// TestAuto_InstallmentsExpandToNEntries asserts that auto with installment
// notation in the value (e.g. "90,00/3") expands into N dated log entries — same as add.
// Each entry carries the "(i/N)" suffix in the item name.
//
// Before: auto did not support installment notation (ParseCurrency, no expansion).
// After: ParseCurrencyWithInstallments + ExpandAndAppend emits N entries to the log.
//
// "Posto Ipiranga" passes the T-32 agreement gate; the installment notation in the
// value string is what drives the expansion under test.
func TestAuto_InstallmentsExpandToNEntries(t *testing.T) {
	extern.RequireOllama(t, "")

	fixDir := filepath.Join(fixturesDir(), "auto-log-append")

	harness.Run(t, harness.Scenario{
		Name:  "auto with installment value notation appends N dated log entries",
		Given: autoLogAppendReady(fixDir),
		When:  actions.RunAuto("Posto Ipiranga", "90,00/3", "15/04/2026"),
		Then: []func(*harness.Context){
			verify.CommandSucceeded(),
			expect.NoRolloverFileCreated(),
			// Three installment entries should be in the log
			expect.FeedbackEntryCount("expenses_log.jsonl", 3),
		},
	})
}

// --- Given helpers ---

func autoLogAppendReady(fixDir string) func(*harness.Context) {
	return func(ctx *harness.Context) {
		ctx.BinaryPath = binaryPath
		domain.SetDataDir(ctx, dataDir)
		ctx.FixtureDir = fixDir
		withFeedbackAndTaxonomyConfig(ctx, fixDir)
	}
}
