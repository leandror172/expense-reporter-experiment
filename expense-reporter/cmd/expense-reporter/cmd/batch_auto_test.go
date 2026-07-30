package cmd

import (
	"os"
	"strings"
	"testing"
	"time"

	"expense-reporter/internal/config"
	"expense-reporter/internal/parse"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

	// Header must end with ;type
	if !strings.HasSuffix(lines[0], ";type") {
		t.Errorf("header missing type column: %q", lines[0])
	}

	// First data row: type = "Fixas"
	fields0 := strings.Split(lines[1], ";")
	if len(fields0) != 8 {
		t.Fatalf("data row has %d fields, want 8: %q", len(fields0), lines[1])
	}
	if fields0[7] != "Fixas" {
		t.Errorf("type field: got %q, want %q", fields0[7], "Fixas")
	}

	// Second data row: type = "" (empty)
	fields1 := strings.Split(lines[2], ";")
	if len(fields1) != 8 {
		t.Fatalf("data row has %d fields, want 8: %q", len(fields1), lines[2])
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
	if !strings.HasSuffix(lines[0], ";type") {
		t.Errorf("header missing type column: %q", lines[0])
	}

	fields := strings.Split(lines[1], ";")
	if len(fields) != 8 {
		t.Fatalf("data row has %d fields, want 8: %q", len(fields), lines[1])
	}
	if fields[7] != "Extras" {
		t.Errorf("type field: got %q, want %q", fields[7], "Extras")
	}
}
