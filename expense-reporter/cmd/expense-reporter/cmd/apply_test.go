package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"expense-reporter/internal/apply"
)

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
		classifPath, expensesLogPath, 2026, false,
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
		classifPath, expensesLogPath, 2026, false,
	)

	require.Error(t, err)
	assert.Equal(t, 0, confirmed)
	assert.Equal(t, 0, corrected)
	require.Len(t, failed, 1)
	assert.Equal(t, "Mystery", failed[0].Item)
}
