package cmd

import (
	"errors"
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

// writeThenReadQueue is the queue-only form, for scenarios with no unparsed rows.
func writeThenReadQueue(t *testing.T, rows []classifiedRow) []review.QueueEntry {
	t.Helper()
	entries, _ := writeThenReadQueueWithUnreviewable(t, rows)
	return entries
}

// writeThenReadQueueWithUnreviewable runs rows through the real writer and the real reader,
// which is the whole point: neither side is simulated, so the bytes under test are the bytes
// that ship. Returns the unreviewable raw lines alongside the queue.
func writeThenReadQueueWithUnreviewable(t *testing.T, rows []classifiedRow) ([]review.QueueEntry, []string) {
	t.Helper()

	csvPath := filepath.Join(t.TempDir(), "classified.csv")
	require.NoError(t, writeClassifiedCSV(csvPath, rows), "writing classified.csv")

	entries, unreviewable, err := review.ReadQueue(csvPath)
	require.NoError(t, err, "review.ReadQueue must accept what writeClassifiedCSV produced")
	return entries, unreviewable
}

// TestClassifiedCSV_UnparsedRowSurvivesTheSeam closes the gap the T-42 scout found in this
// very file. The two tests above feed ONLY successfully-parsed rows, so they passed while
// the seam was still broken for the row shape batch-auto deliberately writes when a line
// does not parse — a guard built from the happy path certifies the happy path.
//
// The producer's contract is TestWriteClassifiedCSV_UnparsedRowKeepsItsRawLine: raw text in
// the item column, empty date and value. The consumer's contract (S1) is that such a row is
// returned as unreviewable rather than failing the read. This pins them TOGETHER, against
// the real writer's bytes, so neither side can be changed alone.
func TestClassifiedCSV_UnparsedRowSurvivesTheSeam(t *testing.T) {
	const badLine = "Anita;Elô ADM;09/01;405,25"
	rows := []classifiedRow{
		classifiedRowWithPrediction(t, "Posto Ipiranga;15/04;280,00", "Combustível", "Transporte", 0.95, true),
		{RawLine: badLine, Error: errors.New("unparseable")},
	}

	entries, unreviewable := writeThenReadQueueWithUnreviewable(t, rows)

	assert.Len(t, entries, 1, "the parsed row is still queued — one bad row must not cost the good ones")
	require.Len(t, unreviewable, 1, "the unparsed row is reported, not silently dropped")
	assert.Equal(t, badLine, unreviewable[0],
		"the caller gets the original text, which is the only thing that identifies the row")
}

// TestClassifiedCSV_KeywordHintSurvivesTheSeam pins the ninth column across the same join.
//
// The writer's own test proves the column is emitted and the reader's own test proves it is
// read — but each was verified against its own idea of the format, which is exactly the
// arrangement that let T-54 ship: a reader accepting a spelling no producer emitted, both
// halves green. The field count is a hard error in the reader, so the two can only move
// together, and this is where that is enforced.
//
// The data is the real case the feature was built for (session 72): the model routed a pet
// consultation to Dentista, a HUMAN dental leaf, while `lilly` mapped to Lilly at
// specificity 1.00 unambiguously. Seven such rows appeared in one month.
//
// Both halves matter. Absence must survive as absence — the page treats PRESENCE as the
// signal for whether to show anything, so a hint that arrived as "none" or "-" would render
// as a real suggestion by that name.
//
// The hint stops here by design: it is input TO the human, not part of their decision, and
// it is deliberately not carried back out through exportReviewed(). Nothing downstream of
// the review page should look for it.
func TestClassifiedCSV_KeywordHintSurvivesTheSeam(t *testing.T) {
	hinted := classifiedRowWithPrediction(t, "Consulta Lilly nefro;15/04;180,00", "Dentista", "Saúde", 0.95, false)
	hinted.KeywordHint = "Lilly"
	unhinted := classifiedRowWithPrediction(t, "Uber Centro;22/06;38,50", "Uber/Taxi", "Transporte", 0.87, false)

	entries := writeThenReadQueue(t, []classifiedRow{hinted, unhinted})

	require.Len(t, entries, 2, "both rows must survive the round trip")
	assert.Equal(t, "Lilly", entries[0].KeywordHint,
		"the keyword's competing suggestion must reach the reviewer's page")
	assert.Empty(t, entries[1].KeywordHint,
		"no second opinion must arrive as an empty hint, never a placeholder")
}

// TestClassifiedCSV_AlreadyLoggedMarkerSurvivesTheSeam pins the tenth column across the
// same join.
//
// The writer's own test proves the column is emitted and the reader's own test proves it is
// read — but each was verified against its own idea of the format, which is exactly the
// arrangement that let T-54 ship: a reader accepting a spelling no producer emitted, both
// halves green. The field count is a hard error in the reader, so the two can only move
// together, and this is where that is enforced.
//
// The two markers must stay DISTINGUISHABLE, which is why this is a marker string and not a
// bool. They demand different things of the human who acts on them: a fully-logged row is
// already recorded, so the reviewer has to decide whether this is a genuinely separate
// purchase (same-day duplicates hash alike by construction, and the live log holds eight
// legitimate pairs); a partially-logged row has some installments recorded and some not, and
// is the only one of the two that can be legitimately completed. Collapse them into one flag
// and the page can no longer say which question it is asking.
//
// Absence must survive as absence — the page treats PRESENCE as the signal for whether to
// show anything, so a marker arriving as "none" or "-" would render as a real state by that
// name. Same rule the keyword hint above follows.
//
// The marker stops here by design: it is input TO the human, not part of their decision, and
// it is deliberately not carried back out through exportReviewed(). Nothing downstream of
// the review page should look for it.
func TestClassifiedCSV_AlreadyLoggedMarkerSurvivesTheSeam(t *testing.T) {
	full := classifiedRowWithPrediction(t, "Posto Ipiranga;15/04;280,00", "Combustível", "Transporte", 0.95, false)
	full.AlreadyLogged = alreadyLoggedFull

	partial := classifiedRowWithPrediction(t, "Anita Elô ADM;09/01;405,25 x4", "Empréstimo", "Anita", 0.91, false)
	partial.AlreadyLogged = alreadyLoggedPartial

	none := classifiedRowWithPrediction(t, "Uber Centro;22/06;38,50", "Uber/Taxi", "Transporte", 0.87, false)

	entries := writeThenReadQueue(t, []classifiedRow{full, partial, none})

	require.Len(t, entries, 3, "every written row must survive into the review queue")
	assert.Equal(t, alreadyLoggedFull, entries[0].AlreadyLogged,
		"a row already recorded in full must reach the reviewer saying so")
	assert.Equal(t, alreadyLoggedPartial, entries[1].AlreadyLogged,
		"a partially-logged series must stay distinguishable from a full duplicate — they ask the reviewer different questions")
	assert.Empty(t, entries[2].AlreadyLogged,
		"a row that is not in the log must arrive as an empty marker, never a placeholder")
}

// TestClassifiedCSV_ThousandsValueSurvivesTheSeam closes a live divergence found in s73,
// by measurement rather than by reading: writeClassifiedCSV emits "1.234,56" and
// review.ReadQueue returned a HARD error on it, so ONE such row anywhere in a month's
// input killed the entire review step.
//
// The cause is the T-54 shape a third time — one function, two callers, and a
// normalization step only one of them applied. internal/parse strips BR thousands dots
// before calling utils.ParseCurrencyWithInstallments; ReadQueue called that function
// directly and so never did. Both sides were individually green.
//
// It is reachable, not theoretical: the value column deliberately keeps the RAW token so
// the installment count survives into the queue (s67), and TestAdd_ReadsBrazilianThousandsAmount
// already pins "1.234,56" as legal CLI input.
func TestClassifiedCSV_ThousandsValueSurvivesTheSeam(t *testing.T) {
	row := classifiedRowWithPrediction(t, "Notebook Dell;15/04;1.234,56", "Eletrônicos", "Diversos", 0.95, false)

	entries := writeThenReadQueue(t, []classifiedRow{row})

	require.Len(t, entries, 1, "a thousands-separated value must not cost the whole review step")
	assert.Equal(t, "1.234,56", entries[0].RawValue,
		"the raw token round-trips unchanged — it is what the reviewer reads")
	assert.InDelta(t, 1234.56, entries[0].Value, 0.001,
		"both sides of the seam must agree that a dot before three digits is a thousands separator")
	assert.Equal(t, 1, entries[0].Installments, "a plain value is a single installment")
}

// TestClassifiedCSV_MultiplierNotationSurvivesTheSeam is the T-64 half of the same join,
// and the reason the notation could not simply be taught to internal/parse alone.
//
// The two value notations mean OPPOSITE things — "405,25/4" divides, "405,25 x4"
// multiplies — so a count that dies at this boundary is not a missing feature, it is a
// budget wrong by a factor of four. That is precisely the T-21 bug: the auto route
// expanded a reviewed installment while the review route silently did not, and a 3×
// purchase recorded as one row is indistinguishable from a legitimate one-off.
//
// Both callers therefore have to learn the notation TOGETHER, and this is the only test
// that can tell whether they did: each side's own tests are written against its own idea
// of the format.
func TestClassifiedCSV_MultiplierNotationSurvivesTheSeam(t *testing.T) {
	multiplied := classifiedRowWithPrediction(t, "Anita Elô ADM;09/01;405,25 x4", "Diversos", "Diversos", 0.95, false)
	divided := classifiedRowWithPrediction(t, "Geladeira;10/01;405,25/4", "Eletrodomésticos", "Diversos", 0.95, false)

	entries := writeThenReadQueue(t, []classifiedRow{multiplied, divided})

	require.Len(t, entries, 2, "both installment notations must survive the round trip")

	assert.Equal(t, "405,25 x4", entries[0].RawValue, "the raw token is what the reviewer reads")
	assert.Equal(t, 4, entries[0].Installments, "the count must reach apply, which is the only consumer that expands it")
	assert.InDelta(t, 405.25, entries[0].Value, 0.001,
		"the multiplier form states the PER-INSTALLMENT value, so it passes through unchanged")

	// The pair is asserted together on purpose. Identical digits, four times apart: if one
	// day both notations were read the same way, only a test holding them side by side
	// would notice.
	assert.Equal(t, 4, entries[1].Installments, "the divisor form carries the same count")
	assert.InDelta(t, 101.3125, entries[1].Value, 0.001,
		"the divisor form states the TOTAL, so the per-installment value is a quarter of it")
}
