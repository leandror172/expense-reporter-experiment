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
		Given: noExpensesLoggedYet(fixDir),
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
		Given: noExpensesLoggedYet(fixDir),
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

// noExpensesLoggedYet: nothing has been logged — a fresh WorkDir with the taxonomy
// authored and the training corpus recorded. The empty log is the salient fact: every
// scenario here asserts on exactly what the auto run appended to it.
func noExpensesLoggedYet(fixDir string) func(*harness.Context) {
	return func(ctx *harness.Context) {
		ctx.BinaryPath = binaryPath
		domain.SetDataDir(ctx, dataDir)
		ctx.FixtureDir = fixDir
		withFeedbackAndTaxonomyConfig(ctx, fixDir)
	}
}

// TestAuto_JoinIDMatchesAcrossLogs guards the T-35 join-key divergence on the auto path.
// auto writes BOTH logs, and they share no key other than GenerateID — a hash of
// (item, date, value). auto passed the RAW date argument to the feedback log but the
// PARSED date to the expense log, so the same expense landed under two different ids and
// every cross-file join silently broke (including `correct`, which looks up the prior
// classification by the normalized-date id and therefore missed every auto-logged row).
//
// The short "15/04" input is what exposes it: with a full "15/04/2026" the raw and
// normalized strings are byte-identical, the hashes agree by accident, and the bug hides —
// which is precisely why the two neighbouring tests above, which pass full dates, never
// caught it.
//
// Single non-installment expense on purpose: installments rewrite the item to "X (i/N)"
// and shift each date by a month, so their ids legitimately differ between the two files
// and the join is only meaningful at count == 1.
func TestAuto_JoinIDMatchesAcrossLogs(t *testing.T) {
	extern.RequireOllama(t, "")

	fixDir := filepath.Join(fixturesDir(), "auto-log-append")

	harness.Run(t, harness.Scenario{
		Name:  "auto given a short DD/MM date logs one row to each log under the same join id",
		Given: noExpensesLoggedYet(fixDir),
		When:  actions.RunAuto("Posto Ipiranga", "35,50", "15/04"),
		Then: slices.Concat(
			commandSucceeded(),
			shortDateExpenseJoinableAcrossBothLogs(),
		),
	})
}

// shortDateExpenseJoinableAcrossBothLogs asserts the auto-appended row carries the same
// join id in classifications.jsonl and expenses_log.jsonl — i.e. both writers normalized
// the short date identically.
func shortDateExpenseJoinableAcrossBothLogs() []func(*harness.Context) {
	return []func(*harness.Context){
		expect.JoinIDMatchesAcrossLogs(),
	}
}
