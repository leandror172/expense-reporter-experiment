package batch

import (
	"fmt"
	"os"
	"strings"
)

// reasonSeparator sits between a repaired data line and its rejection reason.
// The leading whitespace is load-bearing: the re-import strip looks for a '#'
// preceded by whitespace, so an item containing a bare '#' is not mistaken for
// the start of a comment.
const reasonSeparator = "   # "

// FailedRow is one input line batch-auto could not parse, paired with the reason
// the parse boundary rejected it.
type FailedRow struct {
	// OriginalLine is the input line exactly as read. The file the human edits is
	// therefore the text they typed, not a re-serialization of it — round-tripping
	// through a CSV writer would quote and escape it into something else.
	OriginalLine string
	// Reason is the rejection message, written as a TRAILING COMMENT rather than as
	// a fourth field. As a field it would land inside the value (the row parser
	// splits into at most 3 parts), so repairing the data would also mean deleting
	// a column — which is why the retired batch command's file never re-imported.
	Reason string
}

// WriteFailedRows writes rows to path as a file the human can edit in place and feed
// straight back to batch-auto.
//
// An empty slice writes NO file, rather than an empty one: existence alone is the
// signal that a run had rejects, so a clean run must leave nothing behind.
func WriteFailedRows(path string, rows []FailedRow) error {
	if len(rows) == 0 {
		return nil
	}
	if err := os.WriteFile(path, []byte(renderFailedRows(rows)), 0o644); err != nil {
		return fmt.Errorf("writing failed rows to %s: %w", path, err)
	}
	return nil
}

// renderFailedRows builds the whole file so it is written in one call.
func renderFailedRows(rows []FailedRow) string {
	var b strings.Builder
	b.WriteString(failedRowsHeader())
	for _, row := range rows {
		b.WriteString(renderFailedRow(row))
	}
	return b.String()
}

// failedRowsHeader explains the file to whoever opens it months from now. Every line
// is a '#' comment, so the reader skips the block and the file re-imports unedited.
func failedRowsHeader() string {
	return strings.Join([]string{
		"# Rows batch-auto could not parse.",
		"# Fix the data on each line below, then re-run THIS FILE through batch-auto.",
		"# Expected shape: item;DD/MM;value",
		"# Text after '#' is the rejection reason and is ignored on re-import.",
		"",
		"",
	}, "\n")
}

// renderFailedRow puts the reason on the SAME line as its data, which is what makes
// the file repairable in place: the reader sees the failure and the text that caused
// it without moving between lines.
func renderFailedRow(row FailedRow) string {
	return row.OriginalLine + reasonSeparator + flattenReason(row.Reason) + "\n"
}

// flattenReason collapses a multi-line reason onto one physical line. An embedded
// newline would push the remainder of the message to column 0, where it would read
// as a data line and be re-imported as a bogus expense.
func flattenReason(reason string) string {
	return strings.NewReplacer("\r\n", " ", "\r", " ", "\n", " ").Replace(reason)
}
