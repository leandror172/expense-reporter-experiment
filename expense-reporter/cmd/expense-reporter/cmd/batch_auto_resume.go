package cmd

import (
	"fmt"
	"os"

	"expense-reporter/internal/appender"
	"expense-reporter/internal/config"
	"expense-reporter/internal/feedback"
	"expense-reporter/internal/parse"
)

const skippedMarker = "(already logged)"

// loadResumeLedger loads the expense ID counts from the log file.
// The ledger is loaded whenever the log exists (the always-on duplicate warning reads it too), not only under --resume.
func loadResumeLedger(appCfg *config.Config) (map[string]int, error) {
	logPath := appCfg.ExpensesLogFilePath()
	if logPath == "" {
		return make(map[string]int), nil
	}
	return feedback.LoadExpenseIDCounts(logPath)
}

// evaluateResumeSkip determines if a row should be skipped based on the ledger and predicted IDs.
// The ledger invariant: full skips consume here in the classify phase; partial/absent rows consume nothing.
func evaluateResumeSkip(ledger map[string]int, predictedIDs []string) (skip, partial bool) {
	need := make(map[string]int)
	for _, id := range predictedIDs {
		need[id]++
	}

	allPresent := true
	anyPresent := false

	for id, count := range need {
		if ledger[id] < count {
			allPresent = false
		}
		if ledger[id] > 0 {
			anyPresent = true
		}
	}

	if allPresent {
		for id, count := range need {
			ledger[id] -= count
		}
		return true, false
	}

	return false, anyPresent
}

// warnIfDuplicate checks if an ID has been used before and warns if so.
// If ledger[id] <= 0: return false, no mutation.
// Else: decrement ledger[id] by 1, print to os.Stderr a warning line that INCLUDES the exact substring "already in expense log" and includes item and id.
func warnIfDuplicate(ledger map[string]int, item, id string) bool {
	if ledger[id] <= 0 {
		return false
	}
	ledger[id]--
	fmt.Fprintf(os.Stderr, "⚠  duplicate: %q already in expense log (id %s)\n", item, id)
	return true
}

// warnDuplicateEntries warns about duplicates for each ID in the list.
func warnDuplicateEntries(ledger map[string]int, item string, ids []string) {
	for _, id := range ids {
		warnIfDuplicate(ledger, item, id)
	}
}

type resumeOutcome int

const (
	resumeProceed resumeOutcome = iota
	resumeSkipFull
	resumePartial
)

// classifyResumeDecision makes a decision about whether to skip or proceed with a row.
// Consumes ledger counts only on a full skip; partial/absent consume nothing.
//
// It cannot fail. The row arrives already parsed, so the value and date rejections this
// used to report — and the resumeParseErr outcome that carried them — have no way to
// occur here; a line that does not parse never reaches the resume check at all. Predicting
// from the same ParsedExpense the append will later use is also what keeps the prediction
// honest: the ids compared against the ledger are derived exactly as the written ones are.
func classifyResumeDecision(ledger map[string]int, pe parse.ParsedExpense) resumeOutcome {
	predictedIDs := appender.PredictEntryIDs(pe.Item, pe.Date, pe.Value, pe.Installments)
	skip, partial := evaluateResumeSkip(ledger, predictedIDs)

	if skip {
		return resumeSkipFull
	}
	if partial {
		return resumePartial
	}
	return resumeProceed
}
