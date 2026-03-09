package cmd

import (
	"os"
	"strings"
	"testing"
)

func TestParse3FieldLine(t *testing.T) {
	tests := []struct {
		name         string
		line         string
		wantItem     string
		wantDate     string
		wantRawValue string
		wantErr      bool
	}{
		{"valid line", "Uber Centro;15/04;35,50", "Uber Centro", "15/04", "35,50", false},
		{"valid with period decimal", "Item;05/01;160.00", "Item", "05/01", "160.00", false},
		{"installment value", "Uber Centro;15/04;35,50/3", "Uber Centro", "15/04", "35,50/3", false},
		{"too few fields", "Uber;35,50", "", "", "", true},
		{"empty item", ";15/04;35,50", "", "", "", true},
		{"invalid value", "Item;15/04;abc", "", "", "", true},
		{"leading whitespace trimmed", " Uber ;15/04;35,50", "Uber", "15/04", "35,50", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parse3FieldLine(tt.line)
			if (err != nil) != tt.wantErr {
				t.Fatalf("parse3FieldLine(%q) error=%v, wantErr=%v", tt.line, err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}
			if got.Item != tt.wantItem {
				t.Errorf("Item: got %q, want %q", got.Item, tt.wantItem)
			}
			if got.Date != tt.wantDate {
				t.Errorf("Date: got %q, want %q", got.Date, tt.wantDate)
			}
			if got.RawValue != tt.wantRawValue {
				t.Errorf("RawValue: got %q, want %q", got.RawValue, tt.wantRawValue)
			}
		})
	}
}

func TestBatchAutoCommand_Flags(t *testing.T) {
	for _, flag := range []string{"model", "data-dir", "ollama-url", "threshold", "top", "dry-run", "output-dir"} {
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
		{Item: "Uber Centro", Date: "15/04", RawValue: "35,50", Subcategory: "Uber/Taxi", Category: "Transporte", Confidence: 0.95, AutoInserted: true},
		{Item: "Starbucks", Date: "16/04", RawValue: "25,00", Subcategory: "Cafe", Category: "Alimentação", Confidence: 0.70, AutoInserted: false},
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
		{Item: "Uber", AutoInserted: true},
		{Item: "Starbucks", AutoInserted: false},
		{Item: "McDonald's", AutoInserted: false},
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
