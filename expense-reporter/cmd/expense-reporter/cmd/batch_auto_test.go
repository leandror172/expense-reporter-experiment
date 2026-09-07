package cmd

import (
	"encoding/csv"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"expense-reporter/internal/config"
	"expense-reporter/internal/parse"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Column positions in classified.csv / review.csv, which share one header:
// item;date;value;subcategory;category;confidence;auto_inserted;type
const (
	itemColumn  = 0
	dateColumn  = 1
	valueColumn = 2
)

// testParseOptions pins the year ladder so no assertion in this file depends on the
// calendar day it runs. Year outranks the clock, and Now is injected as well because
// the boundary refuses a date beyond the current year — leaving Now at the real clock
// would make these cases start failing on 1 January.
func testParseOptions() parse.Options {
	return parse.Options{Year: 2026, Now: time.Date(2026, 7, 30, 0, 0, 0, 0, time.UTC)}
}

// classifiedRowFor builds a row the way the command builds one: by running the line
// through the real parser. Hand-assembling the Expense would let a test assert against
// a combination the boundary rejects, so the test would be describing behavior the
// production path cannot actually produce.
//
// RawLine is deliberately left unset — it carries the original text only for a row that
// FAILED to parse, and a parsed row that also carried it would make displayItem's
// fallback untestable.
func classifiedRowFor(t *testing.T, line string) classifiedRow {
	t.Helper()
	pe, err := parse3FieldLine(line, testParseOptions())
	require.NoError(t, err, "parse3FieldLine(%q)", line)
	return classifiedRow{Expense: pe}
}

// classifiedRowWithPrediction is classifiedRowFor plus the classifier's verdict, for the
// CSV-writing tests whose subject is a fully classified row rather than the parse.
func classifiedRowWithPrediction(t *testing.T, line, subcategory, category string, confidence float64, autoInserted bool) classifiedRow {
	t.Helper()
	row := classifiedRowFor(t, line)
	row.Subcategory = subcategory
	row.Category = category
	row.Confidence = confidence
	row.AutoInserted = autoInserted
	return row
}

// TestAppendClassified_DowngradesRowOnAppendFailure guards the failure-honesty contract of
// the log-append pivot: when the expense log (the only durable persistence) cannot be
// written, the affected row must be downgraded in place — AutoInserted=false + Error set —
// so the summary count stays honest, the row lands in review.csv, and the command exits
// non-zero. The pre-flight normally makes a real append failure unreachable from the CLI,
// so this is verified here at unit level by pointing the log at a path whose parent dir
// does not exist (feedback.AppendExpense opens with O_CREATE but does not MkdirAll).
func TestAppendClassified_DowngradesRowOnAppendFailure(t *testing.T) {
	row := classifiedRowFor(t, "Uber Centro;15/04/2026;35,50")
	row.Subcategory = "Uber/Taxi"
	row.Category = "Transporte"
	row.Type = "Variáveis"
	row.Confidence = 0.95
	row.AutoInserted = true
	results := []classifiedRow{row}
	// Parent directory does not exist → the append fails; classifications_path is empty so
	// the (secondary) confirmed-feedback write is skipped.
	cfg := &config.Config{ExpensesLogPath: "/expense-reporter-nonexistent-dir/expenses_log.jsonl"}

	err := appendClassified(results, cfg, "my-classifier-q3", map[string]int{})

	require.Error(t, err, "appendClassified should return an error when a row fails to append")
	require.False(t, results[0].AutoInserted, "the failed row must be downgraded to AutoInserted=false")
	require.Error(t, results[0].Error, "the failed row must record its append error")
}

// TestParse3FieldLine pins the CSV reader's half of the contract: it owns the 3-field
// split and nothing else, handing the fields to the parse boundary for every rule about
// what they may contain.
//
// The expected dates are written as full DD/MM/YYYY literals even where the input is a
// bare DD/MM, because that IS the observable change — the reader used to pass a raw date
// string through untouched, and now returns a resolved date whose canonical form is the
// join id's only source. Comparing against a second call to the parser instead would
// assert the parser equals itself and pass even if it returned nothing.
//
// Error cases assert only THAT the line was rejected, never the wording: the message now
// comes from the boundary, which owns it and is free to reword it.
func TestParse3FieldLine(t *testing.T) {
	tests := []struct {
		name         string
		line         string
		wantItem     string
		wantDate     string
		wantRawValue string
		wantErr      bool
	}{
		{"valid line", "Uber Centro;15/04;35,50", "Uber Centro", "15/04/2026", "35,50", false},
		{"valid with period decimal", "Item;05/01;160.00", "Item", "05/01/2026", "160.00", false},
		{"installment value", "Uber Centro;15/04;35,50/3", "Uber Centro", "15/04/2026", "35,50/3", false},
		{"explicit year in the date outranks the ladder", "Item;15/04/2024;35,50", "Item", "15/04/2024", "35,50", false},
		{"too few fields", "Uber;35,50", "", "", "", true},
		{"empty item", ";15/04;35,50", "", "", "", true},
		{"invalid value", "Item;15/04;abc", "", "", "", true},
		{"leading whitespace trimmed", " Uber ;15/04;35,50", "Uber", "15/04/2026", "35,50", false},
		// failed.csv round-trip: the reason is written onto the row it explains, so a
		// repaired line must parse with the comment still attached (T-63).
		{"trailing comment ignored", "Item;15/04;35,50   # invalid date format, expected DD/MM", "Item", "15/04/2026", "35,50", false},
		{"trailing comment with no reason text", "Item;15/04;35,50 #", "Item", "15/04/2026", "35,50", false},
		// A '#' that is NOT whitespace-preceded is data, not a comment marker.
		{"hash inside the item is data", "Mesa #5;15/04;35,50", "Mesa #5", "15/04/2026", "35,50", false},
		// The strip must not widen the format: the 4-field add/correct form still fails
		// rather than silently losing its subcategory.
		{"four fields still rejected", "Item;15/04;35,50;Padaria", "", "", "", true},
		// A '#' glued to the value is a typo, not a comment: without the whitespace rule
		// this would be silently truncated to a valid 35,50 instead of being rejected.
		{"hash glued to the value is rejected", "Item;15/04;35,50#oops", "", "", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parse3FieldLine(tt.line, testParseOptions())
			if tt.wantErr {
				require.Error(t, err, "parse3FieldLine(%q) should have been rejected", tt.line)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantItem, got.Item, "Item")
			assert.Equal(t, tt.wantDate, got.DateString(), "canonical date")
			assert.Equal(t, tt.wantRawValue, got.RawValue, "RawValue")
		})
	}
}

func TestBatchAutoCommand_Flags(t *testing.T) {
	for _, flag := range []string{"model", "data-dir", "ollama-url", "threshold", "top", "dry-run", "output-dir", "resume", "year"} {
		if batchAutoCmd.Flags().Lookup(flag) == nil {
			t.Errorf("flag %q not registered on batch-auto command", flag)
		}
	}
}

func TestWriteClassifiedCSV(t *testing.T) {
	f, err := os.CreateTemp("", "classified-*.csv")
	if err != nil {
		t.Fatal(err)
	}
	f.Close()
	defer os.Remove(f.Name())

	rows := []classifiedRow{
		classifiedRowWithPrediction(t, "Uber Centro;15/04;35,50", "Uber/Taxi", "Transporte", 0.95, true),
		classifiedRowWithPrediction(t, "Starbucks;16/04;25,00", "Cafe", "Alimentação", 0.70, false),
	}

	if err := writeClassifiedCSV(f.Name(), rows); err != nil {
		t.Fatalf("writeClassifiedCSV: %v", err)
	}

	content, err := os.ReadFile(f.Name())
	if err != nil {
		t.Fatal(err)
	}

	lines := strings.Split(strings.TrimSpace(string(content)), "\n")
	if len(lines) != 3 {
		t.Errorf("got %d lines, want 3 (header + 2 data)", len(lines))
	}
	if !strings.HasPrefix(lines[0], "item;date;value") {
		t.Errorf("unexpected header: %q", lines[0])
	}
}

func TestWriteReviewCSV_OnlyLowConfidence(t *testing.T) {
	f, err := os.CreateTemp("", "review-*.csv")
	if err != nil {
		t.Fatal(err)
	}
	f.Close()
	defer os.Remove(f.Name())

	rows := []classifiedRow{
		classifiedRowWithPrediction(t, "Uber;15/04;35,50", "Uber/Taxi", "Transporte", 0.95, true),
		classifiedRowWithPrediction(t, "Starbucks;16/04;25,00", "Cafe", "Alimentação", 0.70, false),
		classifiedRowWithPrediction(t, "McDonald's;17/04;30,00", "Restaurante", "Alimentação", 0.60, false),
	}

	if err := writeReviewCSV(f.Name(), rows); err != nil {
		t.Fatalf("writeReviewCSV: %v", err)
	}

	content, err := os.ReadFile(f.Name())
	if err != nil {
		t.Fatal(err)
	}

	lines := strings.Split(strings.TrimSpace(string(content)), "\n")
	// 1 header + 2 data rows (auto_inserted=true excluded)
	if len(lines) != 3 {
		t.Errorf("got %d lines, want 3 (header + 2 review rows)", len(lines))
	}
}

// Note: TestResolveType (and its sampleExpenseTypes fixture) was removed in T-13.
// The command-layer resolveExpenseType wrapper is gone — classifiedRow.Type is now
// populated from the classifier's predicted full path, not a (category,subcategory)
// TypeIndex lookup. The CSV tests below still verify the type column is emitted.

// TestWriteClassifiedCSV_TypeColumn verifies that writeClassifiedCSV includes the type
// column in the header and that a row with a non-empty Type field has it in column 8.
func TestWriteClassifiedCSV_TypeColumn(t *testing.T) {
	f, err := os.CreateTemp("", "classified-type-*.csv")
	if err != nil {
		t.Fatal(err)
	}
	f.Close()
	defer os.Remove(f.Name())

	typed := classifiedRowWithPrediction(t, "Aluguel;05/01;2500,00", "Aluguel", "Moradia", 0.95, true)
	typed.Type = "Fixas"
	rows := []classifiedRow{
		typed,
		classifiedRowWithPrediction(t, "Uber Centro;15/04;35,50", "Uber/Taxi", "Transporte", 0.80, false),
	}

	if err := writeClassifiedCSV(f.Name(), rows); err != nil {
		t.Fatalf("writeClassifiedCSV: %v", err)
	}

	content, err := os.ReadFile(f.Name())
	if err != nil {
		t.Fatal(err)
	}

	lines := strings.Split(strings.TrimSpace(string(content)), "\n")
	if len(lines) != 3 {
		t.Fatalf("got %d lines, want 3", len(lines))
	}

	// Type must sit at its established index. Asserted positionally rather than as a
	// header suffix: type stopped being the LAST column when keyword_hint was appended,
	// and this test's claim was always about WHERE type is, not what trails it.
	headerFields := strings.Split(lines[0], ";")
	if len(headerFields) <= 7 || headerFields[7] != "type" {
		t.Errorf("header missing type column at index 7: %q", lines[0])
	}

	// First data row: type = "Fixas"
	fields0 := strings.Split(lines[1], ";")
	if len(fields0) != 10 {
		t.Fatalf("data row has %d fields, want 10: %q", len(fields0), lines[1])
	}
	if fields0[7] != "Fixas" {
		t.Errorf("type field: got %q, want %q", fields0[7], "Fixas")
	}

	// Second data row: type = "" (empty)
	fields1 := strings.Split(lines[2], ";")
	if len(fields1) != 10 {
		t.Fatalf("data row has %d fields, want 10: %q", len(fields1), lines[2])
	}
	if fields1[7] != "" {
		t.Errorf("type field for unresolved row: got %q, want empty", fields1[7])
	}
}

// TestWriteReviewCSV_TypeColumn verifies that writeReviewCSV includes the type column.
func TestWriteReviewCSV_TypeColumn(t *testing.T) {
	f, err := os.CreateTemp("", "review-type-*.csv")
	if err != nil {
		t.Fatal(err)
	}
	f.Close()
	defer os.Remove(f.Name())

	autoInserted := classifiedRowWithPrediction(t, "Uber;15/04;35,50", "Uber/Taxi", "Transporte", 0.95, true)
	autoInserted.Type = "Variáveis" // excluded from review.csv
	typed := classifiedRowWithPrediction(t, "Cinema;16/04;40,00", "Cinema", "Lazer", 0.70, false)
	typed.Type = "Extras" // included
	untyped := classifiedRowWithPrediction(t, "Unknown;17/04;10,00", "???", "", 0.30, false)
	// included, type empty

	rows := []classifiedRow{autoInserted, typed, untyped}

	if err := writeReviewCSV(f.Name(), rows); err != nil {
		t.Fatalf("writeReviewCSV: %v", err)
	}

	content, err := os.ReadFile(f.Name())
	if err != nil {
		t.Fatal(err)
	}

	lines := strings.Split(strings.TrimSpace(string(content)), "\n")
	// header + 2 data rows (auto_inserted excluded)
	if len(lines) != 3 {
		t.Fatalf("got %d lines, want 3", len(lines))
	}
	// Type must sit at its established index. Asserted positionally rather than as a
	// header suffix: type stopped being the LAST column when keyword_hint was appended,
	// and this test's claim was always about WHERE type is, not what trails it.
	headerFields := strings.Split(lines[0], ";")
	if len(headerFields) <= 7 || headerFields[7] != "type" {
		t.Errorf("header missing type column at index 7: %q", lines[0])
	}

	fields := strings.Split(lines[1], ";")
	if len(fields) != 10 {
		t.Fatalf("data row has %d fields, want 10: %q", len(fields), lines[1])
	}
	if fields[7] != "Extras" {
		t.Errorf("type field: got %q, want %q", fields[7], "Extras")
	}
}

// csvDataRows writes rows through one of the CSV writers and reads the result back with a
// real CSV reader, returning the data rows only. Parsing rather than splitting on ';'
// matters: a field containing the delimiter is quoted on the way out, so a naive split
// would mis-align exactly the malformed-input row one of these tests is about.
func csvDataRows(t *testing.T, write func(string, []classifiedRow) error, rows []classifiedRow) [][]string {
	t.Helper()
	f, err := os.CreateTemp("", "csv-rows-*.csv")
	require.NoError(t, err)
	f.Close()
	defer os.Remove(f.Name())

	require.NoError(t, write(f.Name(), rows))

	handle, err := os.Open(f.Name())
	require.NoError(t, err)
	defer handle.Close()

	reader := csv.NewReader(handle)
	reader.Comma = ';'
	records, err := reader.ReadAll()
	require.NoError(t, err)
	require.NotEmpty(t, records, "the writer produced no header")
	return records[1:]
}

// TestCSVWriters_PreserveInstallmentNotation pins the plan's one flagged hazard (D4).
//
// The assertion looks trivial — a string round-trips — but the value column is the only
// place an installment count survives into the review queue: review.ReadQueue re-derives
// the count by re-parsing this token. Writing the parsed per-installment float instead
// would be silent, with no error and no failing test anywhere else, and three installments
// would quietly become one (T-21, made worse). Do NOT "simplify" this column to the
// numeric value.
func TestCSVWriters_PreserveInstallmentNotation(t *testing.T) {
	const rawValue = "99,90/3"
	row := classifiedRowWithPrediction(t, "Notebook;15/04;"+rawValue, "Eletrônicos", "Casa", 0.95, false)

	writers := []struct {
		name  string
		write func(string, []classifiedRow) error
	}{
		{"classified.csv", writeClassifiedCSV},
		{"review.csv", writeReviewCSV},
	}
	for _, w := range writers {
		t.Run(w.name, func(t *testing.T) {
			data := csvDataRows(t, w.write, []classifiedRow{row})
			require.Len(t, data, 1)
			assert.Equal(t, rawValue, data[0][valueColumn],
				"%s must carry the original value token, not the per-installment float", w.name)
		})
	}
}

// TestWriteClassifiedCSV_UnparsedRowKeepsItsRawLine covers the row shape that has no
// parsed expense at all.
//
// Such a row must still name itself, so the item column falls back to the original line.
// The date and value cells must be EMPTY rather than rendered: a zero time.Time formats as
// the well-formed but meaningless 01/01/0001, and a plausible-looking date is worse than a
// blank one because nothing downstream can tell it was never real.
func TestWriteClassifiedCSV_UnparsedRowKeepsItsRawLine(t *testing.T) {
	const line = "Uber Centro;not-a-date;35,50"
	rows := []classifiedRow{{RawLine: line, Error: errors.New("unparseable")}}

	data := csvDataRows(t, writeClassifiedCSV, rows)

	require.Len(t, data, 1)
	assert.Equal(t, line, data[0][itemColumn], "an unparsed row identifies itself by its raw line")
	assert.Empty(t, data[0][dateColumn], "a row that never parsed has no date to render")
	assert.Empty(t, data[0][valueColumn], "a row that never parsed has no value to render")
}
