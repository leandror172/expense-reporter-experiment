package cmd

import (
	"bytes"
	"errors"
	"os"
	"testing"

	"expense-reporter/internal/config"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// failed.csv must contain ONLY the rows the boundary rejected. This is the claim the
// acceptance test cannot make cheaply: proving that a PARSEABLE row stays out of the
// file needs a row that classifies, which needs Ollama. The projection is pure, so
// the discrimination is pinned here instead.
func TestRejectedRows_KeepsOnlyRowsThatFailedToParse(t *testing.T) {
	rows := []classifiedRow{
		{RawLine: "bad one;nope;1,00", Error: errors.New("invalid date")},
		{Subcategory: "Padaria"},
		{RawLine: "bad two;;;", Error: errors.New("expected 3 fields")},
		{Subcategory: "Combustível", AutoInserted: true},
		{Skipped: true},
	}

	rejected := rejectedRows(rows)

	require.Len(t, rejected, 2, "only the two unparseable rows belong in failed.csv")
	assert.Equal(t, "bad one;nope;1,00", rejected[0].OriginalLine)
	assert.Equal(t, "bad two;;;", rejected[1].OriginalLine, "input order is preserved")
	assert.Equal(t, "invalid date", rejected[0].Reason)
}

// A run with nothing rejected must project to nothing, because WriteFailedRows treats
// an empty slice as "write no file" — that is how a clean run leaves no artifact.
func TestRejectedRows_CleanRunProjectsToNothing(t *testing.T) {
	rejected := rejectedRows([]classifiedRow{
		{Subcategory: "Padaria"},
		{Subcategory: "Combustível", AutoInserted: true},
	})

	assert.Empty(t, rejected)
}

// Re-running an unrepaired failed.csv must not stack a second copy of the reason onto
// each line. Found by dogfooding the feature: the reason is regenerated every run, so
// carrying the old one through grew the line on each pass.
func TestRejectedRows_ReRunDoesNotAccumulateComments(t *testing.T) {
	alreadyAnnotated := "Comida cinema;109,39;14/01   # invalid date: expected DD/MM, got: 109,39"

	rejected := rejectedRows([]classifiedRow{
		{RawLine: alreadyAnnotated, Error: errors.New("invalid date: expected DD/MM, got: 109,39")},
	})

	require.Len(t, rejected, 1)
	assert.Equal(t, "Comida cinema;109,39;14/01", rejected[0].OriginalLine,
		"the previous run's reason must be dropped, not carried into the next file")
}

// The summary names failed.csv ONLY when a row was rejected. Pointing at a path that was
// deliberately not written would read as an empty reject list rather than as no rejects,
// which is the opposite of what the file's existence is supposed to mean.
func TestPrintBatchSummary_NamesFailedFileOnlyWhenRowsWereRejected(t *testing.T) {
	tests := []struct {
		name     string
		rows     []classifiedRow
		wantPath bool
	}{
		{"a clean run stays silent about it", []classifiedRow{{Subcategory: "Padaria"}}, false},
		{"a run with a reject points at it", []classifiedRow{{RawLine: "bad", Error: errors.New("nope")}}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out := captureStdout(t, func() {
				printBatchSummary(tt.rows, true, "/tmp/classified.csv", "/tmp/review.csv", "/tmp/failed.csv", &config.Config{})
			})
			if tt.wantPath {
				assert.Contains(t, out, "/tmp/failed.csv")
				assert.Contains(t, out, "re-run this file", "the line must say what to DO with it")
				return
			}
			assert.NotContains(t, out, "failed.csv")
		})
	}
}

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	old := os.Stdout
	r, w, err := os.Pipe()
	require.NoError(t, err)
	os.Stdout = w
	fn()
	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, err = buf.ReadFrom(r)
	require.NoError(t, err)
	r.Close()
	return buf.String()
}
