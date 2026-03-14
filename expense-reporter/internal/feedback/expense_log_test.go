package feedback

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"
)

func TestNewExpenseEntry(t *testing.T) {
	fixedTime := time.Date(2026, 3, 14, 10, 0, 0, 0, time.UTC)
	orig := Now
	Now = func() time.Time { return fixedTime }
	defer func() { Now = orig }()

	e := NewExpenseEntry("Uber Centro", "15/04/2025", 35.50, "Uber/Taxi", "Transporte")

	if !hexPattern.MatchString(e.ID) {
		t.Errorf("ID %q is not 12-char hex", e.ID)
	}
	// ID must match GenerateID directly
	wantID := GenerateID("Uber Centro", "15/04/2025", 35.50)
	if e.ID != wantID {
		t.Errorf("ID = %q, want %q", e.ID, wantID)
	}
	if e.Item != "Uber Centro" {
		t.Errorf("Item = %q, want Uber Centro", e.Item)
	}
	if e.Subcategory != "Uber/Taxi" {
		t.Errorf("Subcategory = %q, want Uber/Taxi", e.Subcategory)
	}
	if e.Category != "Transporte" {
		t.Errorf("Category = %q, want Transporte", e.Category)
	}
	if e.Value != 35.50 {
		t.Errorf("Value = %f, want 35.50", e.Value)
	}
	wantTS := "2026-03-14T10:00:00Z"
	if e.Timestamp != wantTS {
		t.Errorf("Timestamp = %q, want %q", e.Timestamp, wantTS)
	}
}

func TestNewExpenseEntry_IDMatchesFeedbackEntry(t *testing.T) {
	// Same item/date/value must produce the same ID in both entry types.
	expenseEntry := NewExpenseEntry("Padaria Maeda", "01/03/2025", 27.50, "Padaria", "Alimentação")
	feedbackID := GenerateID("Padaria Maeda", "01/03/2025", 27.50)
	if expenseEntry.ID != feedbackID {
		t.Errorf("ExpenseEntry.ID %q does not match feedback GenerateID %q — IDs must be consistent across log files", expenseEntry.ID, feedbackID)
	}
}

func TestAppendExpense(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/expenses_log.jsonl"

	entry1 := NewExpenseEntry("Uber Centro", "15/04/2025", 35.50, "Uber/Taxi", "Transporte")
	if err := AppendExpense(path, entry1); err != nil {
		t.Fatalf("AppendExpense first entry: %v", err)
	}

	entry2 := NewExpenseEntry("Padaria Maeda", "01/03/2025", 27.50, "Padaria", "Alimentação")
	if err := AppendExpense(path, entry2); err != nil {
		t.Fatalf("AppendExpense second entry: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	lines := strings.Split(strings.TrimRight(string(data), "\n"), "\n")
	if len(lines) != 2 {
		t.Fatalf("expected 2 lines, got %d", len(lines))
	}

	var got1, got2 ExpenseEntry
	if err := json.Unmarshal([]byte(lines[0]), &got1); err != nil {
		t.Errorf("line 1 not valid JSON: %v", err)
	}
	if got1.Item != "Uber Centro" {
		t.Errorf("line 1 Item = %q, want Uber Centro", got1.Item)
	}
	if err := json.Unmarshal([]byte(lines[1]), &got2); err != nil {
		t.Errorf("line 2 not valid JSON: %v", err)
	}
	if got2.Item != "Padaria Maeda" {
		t.Errorf("line 2 Item = %q, want Padaria Maeda", got2.Item)
	}
}

func TestAppendExpense_CreatesFile(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/new-expenses.jsonl"

	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("precondition: file should not exist")
	}

	entry := NewExpenseEntry("Supermercado", "10/03/2025", 120.50, "Supermercado", "Alimentação")
	if err := AppendExpense(path, entry); err != nil {
		t.Fatalf("AppendExpense to new file: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	lines := strings.Split(strings.TrimRight(string(data), "\n"), "\n")
	if len(lines) != 1 {
		t.Fatalf("expected 1 line, got %d", len(lines))
	}
	var got ExpenseEntry
	if err := json.Unmarshal([]byte(lines[0]), &got); err != nil {
		t.Errorf("line not valid JSON: %v", err)
	}
	if got.Item != "Supermercado" {
		t.Errorf("Item = %q, want Supermercado", got.Item)
	}
}
