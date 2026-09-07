package cmd

import (
	"errors"
	"fmt"

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

// errDuplicateEntry marks a row that reached the append phase while its id was still in
// the ledger. A sentinel so callers can match on the KIND of failure, not on message text.
var errDuplicateEntry = errors.New("entry is already in the expense log")

// refuseDuplicateAppend refuses to append a row whose id the ledger still holds.
//
// This used to WARN and append anyway (T-80). It is an error now for a reason that only
// became true when the duplicate check moved into the classify phase: that check consumes
// the ledger, so by the time a row reaches the append phase every count it could match has
// already been spent. A hit here therefore means the earlier check MISSED — an invariant
// violation, not a routine notice.
//
// Keeping it as a warning would have left a call that can only ever print nothing, and a
// check that cannot fail is not a guard. The caller already downgrades a row whose append
// returns an error and exits non-zero, so returning it is the whole fix.
func refuseDuplicateAppend(ledger map[string]int, item string, ids []string) error {
	for _, id := range ids {
		if ledger[id] > 0 {
			return fmt.Errorf("%q (id %s): %w", item, id, errDuplicateEntry)
		}
	}
	return nil
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

// Markers for classifiedRow.AlreadyLogged. The empty string means the row is NOT in the
// log; these two say how it is.
const (
	alreadyLoggedFull    = "logged"
	alreadyLoggedPartial = "partial"
)

// ledgerOutcomeFor reports why a row is already in the expense log, and whether it can be
// dropped without ever being classified.
//
// The ledger is consulted on EVERY run. --resume decides only what a FULL match does — it
// does NOT decide whether to look. Before T-80 the lookup itself sat behind the flag, so a
// plain run classified a row the log already held, let it pass the auto-insert gate, and
// appended it a second time; because both logs are append-only, undoing that meant
// discarding a whole month of work, and because an auto-inserted row shows on the review
// page as already handled, no human ever saw it.
//
// What the flag still legitimately decides: --resume exists to re-run an interrupted batch,
// where a fully-logged row is noise and dropping it is right. Without the flag the same row
// is a surprise, and a surprise belongs in front of a human — classified, so the reviewer
// has the model's suggestion when judging whether this is a second genuine purchase.
//
// classifyResumeDecision is called exactly once per row because it CONSUMES the ledger
// budget on a full match; a second call would spend a count that no longer exists.
func ledgerOutcomeFor(resume bool, ledger map[string]int, pe parse.ParsedExpense) (marker string, abandon bool) {
	switch classifyResumeDecision(ledger, pe) {
	case resumeSkipFull:
		return alreadyLoggedFull, resume
	case resumePartial:
		return alreadyLoggedPartial, false
	default:
		return "", false
	}
}
