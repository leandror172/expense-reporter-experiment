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

// TestAdd_LogAppendsTypedEntry asserts that add — without a workbook — appends a
// single typed ExpenseEntry line to expenses_log.jsonl in DD/MM/YYYY format.
// Before: add called workflow.InsertExpense (workbook write) then logExpense (type-less DD/MM).
// After: add appends a typed DD/MM/YYYY entry via internal/appender; no workbook touched.
func TestAdd_LogAppendsTypedEntry(t *testing.T) {
	fixDir := filepath.Join(fixturesDir(), "add-log-append")

	harness.Run(t, harness.Scenario{
		Name:  "add appends a typed expense-log entry without touching the workbook",
		Given: expenseManuallyAdded(fixDir),
		When:  actions.RunAdd("Padaria Maeda;15/03/2026;27,50;Padaria"),
		Then: slices.Concat(
			commandSucceeded(),
			classificationsMatchExpected(fixDir),
			typedExpenseRecordedAsSingleLogLine(fixDir),
		),
	})
}

// TestAdd_InstallmentsExpandToNEntriesInLog asserts that an installment value
// notation (e.g. "90,00/3") causes add to append N separate dated log entries —
// one per installment month — each carrying the (i/N) suffix in the item text.
// Before: add did not support installment notation at all (ParseCurrency, no expansion).
// After: ParseCurrencyWithInstallments + ExpandAndAppend in internal/appender emits N lines.
func TestAdd_InstallmentsExpandToNEntriesInLog(t *testing.T) {
	fixDir := filepath.Join(fixturesDir(), "add-log-append-installments")

	harness.Run(t, harness.Scenario{
		Name:  "add with installment notation appends N dated entries to expenses_log.jsonl",
		Given: expenseWithInstallmentsAdded(fixDir),
		When:  actions.RunAdd("Assinatura Netflix;15/03/2026;90,00/3;Netflix"),
		Then: slices.Concat(
			commandSucceeded(),
			classificationsMatchExpected(fixDir),
			installmentExpandedToNDatedLogLines(fixDir),
		),
	})
}

// TestAdd_CrossYearInstallmentLogsNextYearDate asserts that an installment whose
// expansion crosses a year boundary does NOT divert to rollover.csv but instead
// logs normal entries with next-year dates in DD/MM/YYYY format.
// Before: workflow.expandInstallments diverted cross-year installments to rollover.csv.
// After: appender.ExpandAndAppend emits all N entries as dated log lines; no rollover file.
func TestAdd_CrossYearInstallmentLogsNextYearDate(t *testing.T) {
	fixDir := filepath.Join(fixturesDir(), "add-log-append-crossyear")

	harness.Run(t, harness.Scenario{
		Name:  "add with cross-year installments logs all entries with correct dates including next year",
		Given: crossYearInstallmentAdded(fixDir),
		When:  actions.RunAdd("Assinatura Netflix;15/11/2026;90,00/3;Netflix"),
		Then: slices.Concat(
			commandSucceeded(),
			crossYearInstallmentNotDivertedToRollover(),
			classificationsMatchExpected(fixDir),
			crossYearInstallmentLoggedWithNextYearDate(fixDir),
		),
	})
}

// --- Given helpers (event-modeling style) ---

func expenseManuallyAdded(fixDir string) func(*harness.Context) {
	return func(ctx *harness.Context) {
		ctx.BinaryPath = binaryPath
		ctx.DataDir = dataDir
		ctx.FixtureDir = fixDir
		withFeedbackAndTaxonomyConfig(ctx, fixDir)
	}
}

func expenseWithInstallmentsAdded(fixDir string) func(*harness.Context) {
	return func(ctx *harness.Context) {
		ctx.BinaryPath = binaryPath
		ctx.DataDir = dataDir
		ctx.FixtureDir = fixDir
		withFeedbackAndTaxonomyConfig(ctx, fixDir)
	}
}

func crossYearInstallmentAdded(fixDir string) func(*harness.Context) {
	return func(ctx *harness.Context) {
		ctx.BinaryPath = binaryPath
		ctx.DataDir = dataDir
		ctx.FixtureDir = fixDir
		withFeedbackAndTaxonomyConfig(ctx, fixDir)
	}
}

// --- Then helpers (name the specific expected result of each scenario) ---

// typedExpenseRecordedAsSingleLogLine asserts the manual add produced exactly the
// one typed DD/MM/YYYY expense-log line captured in the fixture's expected log.
func typedExpenseRecordedAsSingleLogLine(fixDir string) []func(*harness.Context) {
	return expenseLogMatchesExpected(fixDir)
}

// installmentExpandedToNDatedLogLines asserts the installment notation fanned out
// into the N separate dated log lines captured in the fixture's expected log.
func installmentExpandedToNDatedLogLines(fixDir string) []func(*harness.Context) {
	return expenseLogMatchesExpected(fixDir)
}

// crossYearInstallmentLoggedWithNextYearDate asserts the year-crossing installment
// logged every entry as a dated line — including the next-year dates in the fixture's
// expected log — instead of diverting to a rollover file.
func crossYearInstallmentLoggedWithNextYearDate(fixDir string) []func(*harness.Context) {
	return expenseLogMatchesExpected(fixDir)
}

// crossYearInstallmentNotDivertedToRollover asserts the retired rollover.csv path
// stayed retired: no rollover file was produced for the year-crossing installment.
func crossYearInstallmentNotDivertedToRollover() []func(*harness.Context) {
	return []func(*harness.Context){verify.NoRolloverFileCreated()}
}
