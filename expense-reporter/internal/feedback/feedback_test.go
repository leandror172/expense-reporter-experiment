package feedback

import (
	"bufio"
	"encoding/json"
	"os"
	"regexp"
	"strings"
	"testing"
	"time"

	"expense-reporter/internal/classifier"
)

var hexPattern = regexp.MustCompile(`^[0-9a-f]{12}$`)

func TestGenerateID(t *testing.T) {
	tests := []struct {
		name       string
		item       string
		date       string
		value      float64
		wantLength int
		sameAs     string // if set, another item that should produce the same ID
	}{
		{"basic", "Uber Centro", "15/04/2025", 35.50, 12, ""},
		{"hex chars only", "Padaria", "01/01/2025", 10.00, 12, ""},
		{"leading/trailing spaces normalized", "  Uber Centro  ", "15/04/2025", 35.50, 12, "Uber Centro"},
		{"uppercase normalized", "UBER CENTRO", "15/04/2025", 35.50, 12, "Uber Centro"},
		{"mixed case normalized", "  UBER CENTRO  ", "15/04/2025", 35.50, 12, "Uber Centro"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id := GenerateID(tt.item, tt.date, tt.value)
			if len(id) != tt.wantLength {
				t.Errorf("GenerateID length = %d, want %d (got %q)", len(id), tt.wantLength, id)
			}
			if !hexPattern.MatchString(id) {
				t.Errorf("GenerateID %q is not 12-char hex", id)
			}
			if tt.sameAs != "" {
				canonical := GenerateID(tt.sameAs, tt.date, tt.value)
				if id != canonical {
					t.Errorf("GenerateID(%q) = %q, want same as GenerateID(%q) = %q", tt.item, id, tt.sameAs, canonical)
				}
			}
		})
	}
}

func TestGenerateID_Consistency(t *testing.T) {
	id1 := GenerateID("Supermercado", "10/03/2025", 120.50)
	id2 := GenerateID("Supermercado", "10/03/2025", 120.50)
	if id1 != id2 {
		t.Errorf("GenerateID not deterministic: %q != %q", id1, id2)
	}
}

func TestNewConfirmedEntry(t *testing.T) {
	fixedTime := time.Date(2026, 3, 13, 14, 30, 0, 0, time.UTC)
	orig := Now
	Now = func() time.Time { return fixedTime }
	defer func() { Now = orig }()

	predicted := classifier.Result{
		Subcategory: "Uber/Taxi",
		Category:    "Transporte",
		Confidence:  0.92,
	}

	e := NewConfirmedEntry("Uber Centro", "15/04/2025", 35.50, predicted, "my-classifier-q3")

	if e.Status != StatusConfirmed {
		t.Errorf("Status = %q, want %q", e.Status, StatusConfirmed)
	}
	if e.PredictedSubcategory != "Uber/Taxi" {
		t.Errorf("PredictedSubcategory = %q, want Uber/Taxi", e.PredictedSubcategory)
	}
	if e.PredictedCategory != "Transporte" {
		t.Errorf("PredictedCategory = %q, want Transporte", e.PredictedCategory)
	}
	if e.ActualSubcategory != "Uber/Taxi" {
		t.Errorf("ActualSubcategory = %q, want Uber/Taxi", e.ActualSubcategory)
	}
	if e.ActualCategory != "Transporte" {
		t.Errorf("ActualCategory = %q, want Transporte", e.ActualCategory)
	}
	if e.Confidence != 0.92 {
		t.Errorf("Confidence = %f, want 0.92", e.Confidence)
	}
	if e.Model != "my-classifier-q3" {
		t.Errorf("Model = %q, want my-classifier-q3", e.Model)
	}
	if !hexPattern.MatchString(e.ID) {
		t.Errorf("ID %q is not 12-char hex", e.ID)
	}
	wantTS := "2026-03-13T14:30:00Z"
	if e.Timestamp != wantTS {
		t.Errorf("Timestamp = %q, want %q", e.Timestamp, wantTS)
	}
	if e.Item != "Uber Centro" {
		t.Errorf("Item = %q, want Uber Centro", e.Item)
	}
}

func TestNewManualEntry(t *testing.T) {
	fixedTime := time.Date(2026, 3, 13, 10, 0, 0, 0, time.UTC)
	orig := Now
	Now = func() time.Time { return fixedTime }
	defer func() { Now = orig }()

	e := NewManualEntry("Padaria Maeda", "15/03/2025", 27.50, "Padaria", "Alimentação")

	if e.Status != StatusManual {
		t.Errorf("Status = %q, want %q", e.Status, StatusManual)
	}
	if e.PredictedSubcategory != "" {
		t.Errorf("PredictedSubcategory = %q, want empty", e.PredictedSubcategory)
	}
	if e.PredictedCategory != "" {
		t.Errorf("PredictedCategory = %q, want empty", e.PredictedCategory)
	}
	if e.Confidence != 0.0 {
		t.Errorf("Confidence = %f, want 0.0", e.Confidence)
	}
	if e.Model != "" {
		t.Errorf("Model = %q, want empty", e.Model)
	}
	if e.ActualSubcategory != "Padaria" {
		t.Errorf("ActualSubcategory = %q, want Padaria", e.ActualSubcategory)
	}
	if e.ActualCategory != "Alimentação" {
		t.Errorf("ActualCategory = %q, want Alimentação", e.ActualCategory)
	}
	if !hexPattern.MatchString(e.ID) {
		t.Errorf("ID %q is not 12-char hex", e.ID)
	}
}

func TestAppend(t *testing.T) {
	dir := t.TempDir()
	f, err := os.CreateTemp(dir, "feedback-*.jsonl")
	if err != nil {
		t.Fatalf("CreateTemp: %v", err)
	}
	path := f.Name()
	f.Close()

	entry1 := Entry{
		ID:     "aabbccddeeff",
		Item:   "Uber Centro",
		Date:   "15/04/2025",
		Value:  35.50,
		Status: StatusConfirmed,
	}
	if err := Append(path, entry1); err != nil {
		t.Fatalf("Append first entry: %v", err)
	}

	entry2 := Entry{
		ID:     "112233445566",
		Item:   "Padaria",
		Date:   "01/01/2025",
		Value:  10.00,
		Status: StatusManual,
	}
	if err := Append(path, entry2); err != nil {
		t.Fatalf("Append second entry: %v", err)
	}

	// Read back both lines
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	lines := strings.Split(strings.TrimRight(string(data), "\n"), "\n")
	if len(lines) != 2 {
		t.Fatalf("expected 2 lines, got %d", len(lines))
	}

	var got1, got2 Entry
	if err := json.Unmarshal([]byte(lines[0]), &got1); err != nil {
		t.Errorf("line 1 not valid JSON: %v", err)
	}
	if got1.Item != "Uber Centro" {
		t.Errorf("line 1 Item = %q, want Uber Centro", got1.Item)
	}
	if err := json.Unmarshal([]byte(lines[1]), &got2); err != nil {
		t.Errorf("line 2 not valid JSON: %v", err)
	}
	if got2.Item != "Padaria" {
		t.Errorf("line 2 Item = %q, want Padaria", got2.Item)
	}
}

func TestAppend_CreatesFile(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/new-classifications.jsonl"

	// File must not exist yet
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("precondition: file should not exist")
	}

	entry := Entry{
		ID:     "aabbccddeeff",
		Item:   "Test Item",
		Date:   "01/01/2025",
		Value:  99.00,
		Status: StatusConfirmed,
	}
	if err := Append(path, entry); err != nil {
		t.Fatalf("Append to new file: %v", err)
	}

	// File now exists and has exactly 1 line
	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("Open created file: %v", err)
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	lineCount := 0
	var lastLine string
	for scanner.Scan() {
		lineCount++
		lastLine = scanner.Text()
	}
	if lineCount != 1 {
		t.Fatalf("expected 1 line, got %d", lineCount)
	}
	var got Entry
	if err := json.Unmarshal([]byte(lastLine), &got); err != nil {
		t.Errorf("line not valid JSON: %v", err)
	}
	if got.Item != "Test Item" {
		t.Errorf("Item = %q, want Test Item", got.Item)
	}
}
