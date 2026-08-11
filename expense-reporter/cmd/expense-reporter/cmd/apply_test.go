package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"expense-reporter/internal/apply"
)

// newInstallmentRow builds a confirmed new row for a purchase split into n installments.
func newInstallmentRow(item string, n int) apply.ReviewedEntry {
	row := newConfirmedRow(item)
	row.Installments = n
	return row
}

// TestAppendNewRows_CountsRowsWrittenNotEntries pins the UNIT of apply's summary.
//
// printSummary prints "Appended: %d rows", and until T-21 that label was true by
// accident: every entry produced exactly one log row, so counting entries and counting
// rows were the same number. An installment purchase now produces N, so a per-entry
// counter reports "1 rows" for a write of three — the same class of defect T-21 itself
// was, a count that quietly stops meaning what it says.
func TestAppendNewRows_CountsRowsWrittenNotEntries(t *testing.T) {
	dir := t.TempDir()
	classifPath := filepath.Join(dir, "classifications.jsonl")
	expensesLogPath := filepath.Join(dir, "expenses_log.jsonl")

	confirmed, corrected, failed, err := appendNewRows(
		[]apply.ReviewedEntry{newInstallmentRow("Tratamento dentário", 3)},
		classifPath, expensesLogPath, false,
	)

	require.NoError(t, err)
	require.Empty(t, failed)
	assert.Equal(t, 0, corrected)
	assert.Equal(t, 3, confirmed, "a three-installment purchase writes three rows")

	data, readErr := os.ReadFile(expensesLogPath)
	require.NoError(t, readErr)
	assert.Equal(t, 3, strings.Count(string(data), "\n"),
		"the count must match the lines that actually landed, not the entries processed")
}

// TestAppendNewRows_DryRunPreviewsTheRowsARealRunWouldWrite is the paired inverse: a
// preview whose number disagrees with the run it previews is worse than no preview,
// because the user approves the write on the strength of it.
func TestAppendNewRows_DryRunPreviewsTheRowsARealRunWouldWrite(t *testing.T) {
	dir := t.TempDir()
	expensesLogPath := filepath.Join(dir, "expenses_log.jsonl")

	previewed, _, _, err := appendNewRows(
		[]apply.ReviewedEntry{newInstallmentRow("Tratamento dentário", 3)},
		filepath.Join(dir, "classifications.jsonl"), expensesLogPath, true,
	)

	require.NoError(t, err)
	assert.Equal(t, 3, previewed, "the dry-run must promise the same three rows the real run writes")

	_, statErr := os.Stat(expensesLogPath)
	assert.True(t, os.IsNotExist(statErr), "a dry run must still write nothing")
}

// newConfirmedRow builds a confirmed new row with a valid reviewed location.
func newConfirmedRow(item string) apply.ReviewedEntry {
	return apply.ReviewedEntry{
		Item:   item,
		Date:   "15/04/2026",
		Value:  35.50,
		Action: apply.ActionConfirmed,
		Reviewed: &apply.ReviewedLocation{
			Type:        "Variáveis",
			Category:    "Transporte",
			Subcategory: "Uber/Taxi",
		},
	}
}

// TestAppendNewRows_DowngradesRowOnAppendFailure: when the expense-log append
// fails, the row is downgraded into `failed` (not appended), no feedback is
// written for it, and appendNewRows returns a non-nil error so the command
// exits non-zero. Guards the §2 failure-honesty rule at the unit level (the
// pre-flight makes this unreachable at acceptance level).
func TestAppendNewRows_DowngradesRowOnAppendFailure(t *testing.T) {
	dir := t.TempDir()
	// A regular file used as a directory component → OpenFile under it fails.
	blocker := filepath.Join(dir, "blocker")
	require.NoError(t, os.WriteFile(blocker, []byte("x"), 0o644))

	classifPath := filepath.Join(dir, "classifications.jsonl")
	expensesLogPath := filepath.Join(blocker, "expenses_log.jsonl") // unwritable

	confirmed, corrected, failed, err := appendNewRows(
		[]apply.ReviewedEntry{newConfirmedRow("Uber Centro")},
		classifPath, expensesLogPath, false,
	)

	require.Error(t, err, "an unpersisted row must surface a non-zero exit")
	assert.Equal(t, 0, confirmed)
	assert.Equal(t, 0, corrected)
	require.Len(t, failed, 1)
	assert.Equal(t, "Uber Centro", failed[0].Item)

	// Feedback must NOT have been written for the failed row (log-first ordering).
	_, statErr := os.Stat(classifPath)
	assert.True(t, os.IsNotExist(statErr), "no feedback should be written when the durable append failed")
}

// TestAppendNewRows_MalformedReviewedRoutedToFailed: a confirmed/corrected entry
// with a nil Reviewed location is routed to `failed` rather than panicking on a
// nil dereference, and yields a non-zero exit. Guards advisor finding #4.
func TestAppendNewRows_MalformedReviewedRoutedToFailed(t *testing.T) {
	dir := t.TempDir()
	classifPath := filepath.Join(dir, "classifications.jsonl")
	expensesLogPath := filepath.Join(dir, "expenses_log.jsonl")

	malformed := apply.ReviewedEntry{
		Item:     "Mystery",
		Date:     "15/04/2026",
		Value:    10,
		Action:   apply.ActionConfirmed,
		Reviewed: nil, // malformed reviewed.json
	}

	confirmed, corrected, failed, err := appendNewRows(
		[]apply.ReviewedEntry{malformed},
		classifPath, expensesLogPath, false,
	)

	require.Error(t, err)
	assert.Equal(t, 0, confirmed)
	assert.Equal(t, 0, corrected)
	require.Len(t, failed, 1)
	assert.Equal(t, "Mystery", failed[0].Item)
}
