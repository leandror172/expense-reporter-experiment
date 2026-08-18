package batch

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// A clean run must leave NOTHING behind: callers use the file's existence as the
// signal that rows were rejected, so an empty file would report a problem that
// did not happen.
func TestWriteFailedRows_EmptySliceWritesNoFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "failed.csv")

	require.NoError(t, WriteFailedRows(path, nil))

	_, err := os.Stat(path)
	assert.True(t, os.IsNotExist(err), "a run with no rejects must not create %s", path)
}

// The reason rides on the SAME line as the data it explains, as a trailing comment.
// This is the whole point of the format: the human repairs the data in place without
// having to delete a column first.
func TestWriteFailedRows_ReasonSharesTheLineWithItsData(t *testing.T) {
	path := filepath.Join(t.TempDir(), "failed.csv")

	require.NoError(t, WriteFailedRows(path, []FailedRow{
		{OriginalLine: "Comida cinema;109,39;14/01", Reason: "invalid date format, expected DD/MM"},
	}))

	line := dataLinesOf(t, path)[0]
	assert.True(t, strings.HasPrefix(line, "Comida cinema;109,39;14/01"),
		"the data must come first so the line stays repairable: %q", line)
	assert.Contains(t, line, "# invalid date format, expected DD/MM")
}

// Every row is preserved, in order. Losing one silently would be worse than the
// stderr-only behaviour this file replaces.
func TestWriteFailedRows_KeepsEveryRowInOrder(t *testing.T) {
	path := filepath.Join(t.TempDir(), "failed.csv")

	require.NoError(t, WriteFailedRows(path, []FailedRow{
		{OriginalLine: "first;01/01;1,00", Reason: "reason one"},
		{OriginalLine: "second;02/01;2,00", Reason: "reason two"},
		{OriginalLine: "third;03/01;3,00", Reason: "reason three"},
	}))

	lines := dataLinesOf(t, path)
	require.Len(t, lines, 3)
	for i, wantItem := range []string{"first", "second", "third"} {
		assert.True(t, strings.HasPrefix(lines[i], wantItem), "row %d: %q", i, lines[i])
	}
}

// A newline inside the reason would push its remainder to column 0, where the reader
// would treat it as a data line and re-import a bogus expense.
func TestWriteFailedRows_MultilineReasonCollapsesToOneLine(t *testing.T) {
	path := filepath.Join(t.TempDir(), "failed.csv")

	require.NoError(t, WriteFailedRows(path, []FailedRow{
		{OriginalLine: "Item;01/01;1,00", Reason: "first part\nsecond part\r\nthird part"},
	}))

	lines := dataLinesOf(t, path)
	require.Len(t, lines, 1, "a multi-line reason must not become extra lines")
	assert.Contains(t, lines[0], "first part second part third part")
}

// The header is entirely '#' comments so the file re-imports without editing the
// explanation away.
func TestWriteFailedRows_HeaderIsAllComments(t *testing.T) {
	path := filepath.Join(t.TempDir(), "failed.csv")

	require.NoError(t, WriteFailedRows(path, []FailedRow{
		{OriginalLine: "Item;01/01;1,00", Reason: "why"},
	}))

	content := readFile(t, path)
	require.True(t, strings.HasPrefix(content, "#"), "file must open with the explanation")
	assert.True(t, strings.HasSuffix(content, "\n"), "file must end with a newline")
}

// dataLinesOf returns the lines a re-import would actually consume: comments and
// blanks dropped, exactly as batch.CSVReader does.
func dataLinesOf(t *testing.T, path string) []string {
	t.Helper()
	var out []string
	for _, line := range strings.Split(readFile(t, path), "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		out = append(out, line)
	}
	return out
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	return string(data)
}
