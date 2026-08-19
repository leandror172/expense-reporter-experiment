package cmd

import (
	"encoding/csv"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Column indices for the classified/review CSVs. itemColumn, dateColumn and valueColumn
// already live in batch_auto_test.go; these complete the row so no assertion below has to
// spell a bare integer.
const (
	subcategoryColumn  = 3
	categoryColumn     = 4
	confidenceColumn   = 5
	autoInsertedColumn = 6
	typeColumn         = 7
	keywordHintColumn  = 8
)

// expectedCSVHeader is the full contract, stated once. keyword_hint is APPENDED — every
// prior column keeps its index, which is what lets expect.NoneWereAutoInserted go on
// reading auto_inserted positionally at 6.
var expectedCSVHeader = []string{
	"item", "date", "value", "subcategory", "category",
	"confidence", "auto_inserted", "type", "keyword_hint",
}

type csvWriter struct {
	name  string
	write func(string, []classifiedRow) error
}

// bothCSVWriters is the point of these tests: classified.csv and review.csv must satisfy
// the SAME column contract, and that is only proved by running identical assertions
// through each. review.csv drifting from classified.csv is the T-54 shape — one consumer,
// two producers, no test joining them.
func bothCSVWriters() []csvWriter {
	return []csvWriter{
		{"classified.csv", writeClassifiedCSV},
		{"review.csv", writeReviewCSV},
	}
}

// csvHeaderRow runs a writer over no rows and returns its header. csvDataRows drops the
// header, so the contract this file is about needs its own reader.
func csvHeaderRow(t *testing.T, write func(string, []classifiedRow) error) []string {
	t.Helper()
	f, err := os.CreateTemp("", "csv-header-*.csv")
	require.NoError(t, err)
	f.Close()
	defer os.Remove(f.Name())

	require.NoError(t, write(f.Name(), nil))

	handle, err := os.Open(f.Name())
	require.NoError(t, err)
	defer handle.Close()

	reader := csv.NewReader(handle)
	reader.Comma = ';'
	records, err := reader.ReadAll()
	require.NoError(t, err)
	require.NotEmpty(t, records, "the writer produced no header")
	return records[0]
}

// hintedRow is a row the keyword layer had a second opinion about. Deliberately NOT
// auto-inserted: writeReviewCSV skips auto-inserted rows, so an auto-inserted row could
// never be compared across both writers.
func hintedRow(t *testing.T, hint string) classifiedRow {
	t.Helper()
	row := classifiedRowWithPrediction(t, "Uber Centro;15/04/2026;35,50", "Uber/Taxi", "Transporte", 0.95, false)
	row.KeywordHint = hint
	return row
}

// TestCSVWriters_KeywordHintIsAppendedAsTheNinthColumn pins the position, not just the
// presence. Inserting the column anywhere earlier would shift auto_inserted out from under
// expect.NoneWereAutoInserted, which indexes it by number and would then silently assert
// against the wrong field rather than fail to compile.
func TestCSVWriters_KeywordHintIsAppendedAsTheNinthColumn(t *testing.T) {
	for _, w := range bothCSVWriters() {
		t.Run(w.name, func(t *testing.T) {
			assert.Equal(t, expectedCSVHeader, csvHeaderRow(t, w.write),
				"%s must append keyword_hint without disturbing the eight columns before it", w.name)
		})
	}
}

// TestCSVWriters_KeywordHintReachesTheReviewQueue: a second opinion the keyword layer
// offered must survive into the file the reviewer's page is built from. This is the whole
// transport contract — the predicate is pinned in classifier.TestKeywordHint.
func TestCSVWriters_KeywordHintReachesTheReviewQueue(t *testing.T) {
	const hint = "Lilly"
	for _, w := range bothCSVWriters() {
		t.Run(w.name, func(t *testing.T) {
			data := csvDataRows(t, w.write, []classifiedRow{hintedRow(t, hint)})
			require.Len(t, data, 1)
			assert.Equal(t, hint, data[0][keywordHintColumn],
				"%s dropped the keyword's suggestion", w.name)
		})
	}
}

// TestCSVWriters_AbsentKeywordHintIsAnEmptyCell: no second opinion must be spelled as the
// empty string, never a placeholder. The consumer treats PRESENCE as the signal, so a
// sentinel like "none" would render as a real suggestion named "none".
func TestCSVWriters_AbsentKeywordHintIsAnEmptyCell(t *testing.T) {
	for _, w := range bothCSVWriters() {
		t.Run(w.name, func(t *testing.T) {
			data := csvDataRows(t, w.write, []classifiedRow{hintedRow(t, "")})
			require.Len(t, data, 1)
			assert.Empty(t, data[0][keywordHintColumn],
				"%s must leave the cell empty when the keyword layer had nothing to add", w.name)
		})
	}
}

// TestCSVWriters_AppendingTheHintLeavesEveryPriorColumnInPlace is the regression guard for
// the shift itself. The header test above would catch a reordered HEADER; this catches a
// reordered ROW, which is a separate mistake and the one that corrupts data silently.
func TestCSVWriters_AppendingTheHintLeavesEveryPriorColumnInPlace(t *testing.T) {
	for _, w := range bothCSVWriters() {
		t.Run(w.name, func(t *testing.T) {
			data := csvDataRows(t, w.write, []classifiedRow{hintedRow(t, "Lilly")})
			require.Len(t, data, 1)
			row := data[0]

			assert.Equal(t, "Uber Centro", row[itemColumn], "item column moved")
			assert.Equal(t, "15/04/2026", row[dateColumn], "date column moved")
			assert.Equal(t, "35,50", row[valueColumn], "value column moved")
			assert.Equal(t, "Uber/Taxi", row[subcategoryColumn], "subcategory column moved")
			assert.Equal(t, "Transporte", row[categoryColumn], "category column moved")
			assert.Equal(t, "0.9500", row[confidenceColumn], "confidence column moved")
			assert.Equal(t, "false", row[autoInsertedColumn], "auto_inserted column moved")
			assert.Equal(t, "Lilly", row[keywordHintColumn], "keyword_hint is not last")
		})
	}
}
