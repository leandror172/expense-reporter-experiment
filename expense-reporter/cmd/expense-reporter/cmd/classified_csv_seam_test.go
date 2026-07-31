package cmd

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"expense-reporter/internal/feedback"
	"expense-reporter/internal/review"
)

// The seam between batch-auto and review had no test until T-54. writeClassifiedCSV is the
// only Go producer of classified.csv and review.ReadQueue is its only consumer, but every
// other test feeds ReadQueue a HAND-AUTHORED fixture written to satisfy the reader. Two
// halves each self-consistent, joined by nothing — so the suite stayed green while
// `expense-reporter review classified.csv` could not read real batch-auto output at all.
//
// These two tests are the join. They generate the producer's real bytes at runtime, which
// is what makes them representative without any fixture having to be kept in step.

// TestClassifiedCSV_ReviewReadsWhatBatchAutoWrote pins that the reader accepts the writer's
// actual output. It fails before T-54 because the writer renders auto_inserted as
// true/false (fmt %v on a bool) while the reader accepted only "1"/"0" and returned a hard
// error on anything else — a spelling nobody in the tree ever produced.
func TestClassifiedCSV_ReviewReadsWhatBatchAutoWrote(t *testing.T) {
	rows := []classifiedRow{
		classifiedRowWithPrediction(t, "Posto Ipiranga;15/04;280,00", "Combustível", "Transporte", 0.95, true),
		classifiedRowWithPrediction(t, "Uber Centro;22/06;38,50", "Uber/Taxi", "Transporte", 0.87, false),
	}

	entries := writeThenReadQueue(t, rows)

	require.Len(t, entries, 2, "every written row must survive into the review queue")
	assert.Equal(t, "Posto Ipiranga", entries[0].Item, "the item column round-trips unchanged")
	assert.True(t, entries[0].AutoInserted, "a row the gate admitted is still marked auto-inserted after the round trip")
	assert.Equal(t, "Uber Centro", entries[1].Item, "the item column round-trips unchanged")
	assert.False(t, entries[1].AutoInserted, "a row routed to review is still marked not-auto-inserted after the round trip")
}

// TestClassifiedCSV_QueueIDJoinsTheExpenseLogID pins the property T-41 slice 3 established
// and left untested: the id review.ReadQueue hashes out of the CSV is the SAME id the
// expense log hashes for that expense, so reviewed.json joins the logs instead of naming
// rows that exist nowhere.
//
// It holds only because writeClassifiedCSV emits the CANONICAL date (dateCell) rather than
// the raw input. Revert dateCell to the raw string and this test — and only this test —
// goes red.
func TestClassifiedCSV_QueueIDJoinsTheExpenseLogID(t *testing.T) {
	// The input date MUST stay bare DD/MM. A full date makes the raw and canonical strings
	// byte-identical, which would leave this test passing no matter what the code did —
	// silently disabled rather than broken. Same rule as the apply-join-id fixtures.
	const line = "Posto Ipiranga;15/04;280,00"

	row := classifiedRowWithPrediction(t, line, "Combustível", "Transporte", 0.95, true)
	// A single expense only. An installment SERIES legitimately diverges here, because the
	// appender suffixes the item per installment while the feedback side keeps the original
	// — asserting equality on a series would pin behavior the system deliberately does not have.
	require.Equal(t, 1, row.Expense.Installments, "the id-equality claim is meaningful only for a single expense")

	entries := writeThenReadQueue(t, []classifiedRow{row})

	require.Len(t, entries, 1, "one row in, one queue entry out")
	expenseLogID := feedback.GenerateID(row.Expense.Item, row.Expense.DateString(), row.Expense.Value)
	assert.Equal(t, expenseLogID, entries[0].ID,
		"the review queue and the expense log must name this expense with one id, not two")
}

// writeThenReadQueue runs rows through the real writer and the real reader, which is the
// whole point: neither side is simulated, so the bytes under test are the bytes that ship.
func writeThenReadQueue(t *testing.T, rows []classifiedRow) []review.QueueEntry {
	t.Helper()

	csvPath := filepath.Join(t.TempDir(), "classified.csv")
	require.NoError(t, writeClassifiedCSV(csvPath, rows), "writing classified.csv")

	entries, _, err := review.ReadQueue(csvPath)
	require.NoError(t, err, "review.ReadQueue must accept what writeClassifiedCSV produced")
	return entries
}
