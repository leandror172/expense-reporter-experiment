package capture

// The sink half of plan D3 / D7, as PURE functions: which line goes to which month
// file, and what the rejects file must say. No I/O here — the command layer owns
// paths, the overwrite refusal (D8) and the writes, so this is unit-testable with a
// slice of outcomes and nothing else.

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

// MonthFile is one batch-auto input: every Converted line whose RESOLVED expense date
// falls in that month (plan D7 — a purchase on 30/05 reported on 01/06 belongs to
// May), in stream order.
type MonthFile struct {
	// Name is "expenses-YYYY-MM.csv", the shape of the close's existing files.
	Name string
	// Lines are the CSV body lines: item;DD/MM/YYYY;value. A repaired line carries a
	// trailing "   # repaired from: <text as typed> (message <id>)" comment, which
	// batch-auto's reader strips (whitespace-preceded '#' after three complete
	// fields) — so the audit trail rides inside the very file batch-auto reads.
	Lines []string
}

// Reject is one attempted expense the converter could not take, in the terms the
// T-63 rejects file needs: the text as typed, and the reason a human repairs by.
type Reject struct {
	// Text is the message text with any line break flattened to a space, so the
	// file stays one row per message. Nothing else is changed: the human repairs
	// what they typed, not a re-serialization of it.
	Text string
	// Reason is the boundary's own message followed by the provenance a human needs
	// to fix a year by hand: "<error> (message <id>, sent <YYYY-MM-DD>)".
	Reason string
}

// repairedFromSeparator matches batch's reason separator on purpose: the leading
// whitespace is what lets the reader tell a comment from a '#' inside an item.
const repairedFromSeparator = "   # repaired from: "

// MonthFileName renders the file a date's month belongs to: "expenses-YYYY-MM.csv".
func MonthFileName(t time.Time) string {
	return fmt.Sprintf("expenses-%04d-%02d.csv", t.Year(), int(t.Month()))
}

// MonthFiles groups every Converted outcome by MonthFileName(outcome.Date), keeping
// stream order inside each file, and returns the files sorted by Name. Outcomes of
// any other class contribute nothing. A repaired outcome's line gets the
// "repaired from" comment appended.
func MonthFiles(outcomes []Outcome) []MonthFile {
	monthLines := make(map[string][]string)
	for _, o := range outcomes {
		if o.Class != Converted {
			continue
		}
		line := monthLine(o)
		name := MonthFileName(o.Date)
		monthLines[name] = append(monthLines[name], line)
	}

	return sortedMonthFiles(monthLines)
}

// Rejects lists every Rejected outcome, in stream order, as the rows of the rejects
// file. Receipts and conversation are NOT rejects and never appear here (plan D3).
func Rejects(outcomes []Outcome) []Reject {
	var rejects []Reject
	for _, o := range outcomes {
		if o.Class != Rejected {
			continue
		}
		text := flattenLineBreaks(strings.TrimSpace(o.Message.Text))
		reason := fmt.Sprintf("%v (message %d, sent %s)", o.Err, o.Message.ID, o.Message.SentAt.Format("2006-01-02"))
		rejects = append(rejects, Reject{Text: text, Reason: reason})
	}
	return rejects
}

// monthLine renders the line for a Converted outcome.
func monthLine(o Outcome) string {
	if o.Repair == "" {
		return o.Line
	}
	return o.Line + repairedFromSeparator + flattenLineBreaks(o.Message.Text) + fmt.Sprintf(" (message %d)", o.Message.ID)
}

// sortedMonthFiles converts a map of name→lines to a slice of MonthFile, sorted by name.
func sortedMonthFiles(byName map[string][]string) []MonthFile {
	names := make([]string, 0, len(byName))
	for name := range byName {
		names = append(names, name)
	}
	sort.Strings(names)

	files := make([]MonthFile, 0, len(names))
	for _, name := range names {
		files = append(files, MonthFile{Name: name, Lines: byName[name]})
	}
	return files
}

// flattenLineBreaks keeps a message on one physical line of a line-oriented file.
func flattenLineBreaks(s string) string {
	return strings.NewReplacer("\r\n", " ", "\r", " ", "\n", " ").Replace(s)
}
