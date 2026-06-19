package cmd

import (
	"os"
	"strings"
	"testing"

	"expense-reporter/internal/taxonomy"
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

// sampleExpenseTypes builds a small expense-type tree mirroring the taxonomy.sampleTypes
// fixture used in taxonomy/lookup_test.go — Fixas/Moradia/Aluguel, Variáveis/Mercado/Feira,
// and an ambiguous pair Variáveis+Extras/Saúde/Dentista.
func sampleExpenseTypes() []taxonomy.ExpenseType {
	return []taxonomy.ExpenseType{
		{
			Name: "Fixas",
			Cats: []taxonomy.Category{
				{Name: "Moradia", Subs: []taxonomy.Subcat{{Name: "Aluguel"}, {Name: "Condomínio"}}},
			},
		},
		{
			Name: "Variáveis",
			Cats: []taxonomy.Category{
				{Name: "Mercado", Subs: []taxonomy.Subcat{{Name: "Feira"}}},
				{Name: "Saúde", Subs: []taxonomy.Subcat{{Name: "Dentista"}}},
			},
		},
		{
			Name: "Extras",
			Cats: []taxonomy.Category{
				{Name: "Saúde", Subs: []taxonomy.Subcat{{Name: "Dentista"}}},
			},
		},
	}
}

// TestResolveType verifies the pure helper that maps (category, subcategory) → type
// string via a taxonomy.TypeIndex, covering the three outcomes: unique resolution,
// absent pair, and ambiguous pair. The last two must both return "".
func TestResolveType(t *testing.T) {
	idx := taxonomy.BuildTypeIndex(sampleExpenseTypes())

	tests := []struct {
		name        string
		category    string
		subcategory string
		wantType    string
	}{
		{"unique pair resolves to type", "Moradia", "Aluguel", "Fixas"},
		{"unique pair resolves to variáveis type", "Mercado", "Feira", "Variáveis"},
		{"absent pair returns empty", "Lazer", "Cinema", ""},
		{"ambiguous pair returns empty", "Saúde", "Dentista", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := resolveExpenseType(idx, tt.category, tt.subcategory)
			if got != tt.wantType {
				t.Errorf("resolveExpenseType(%q, %q) = %q, want %q", tt.category, tt.subcategory, got, tt.wantType)
			}
		})
	}
}

// TestWriteClassifiedCSV_TypeColumn verifies that writeClassifiedCSV includes the type
// column in the header and that a row with a non-empty Type field has it in column 8.
func TestWriteClassifiedCSV_TypeColumn(t *testing.T) {
	f, err := os.CreateTemp("", "classified-type-*.csv")
	if err != nil {
		t.Fatal(err)
	}
	f.Close()
	defer os.Remove(f.Name())

	rows := []classifiedRow{
		{Item: "Aluguel", Date: "05/01", RawValue: "2500,00", Subcategory: "Aluguel", Category: "Moradia", Confidence: 0.95, AutoInserted: true, Type: "Fixas"},
		{Item: "Uber Centro", Date: "15/04", RawValue: "35,50", Subcategory: "Uber/Taxi", Category: "Transporte", Confidence: 0.80, AutoInserted: false, Type: ""},
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

	rows := []classifiedRow{
		{Item: "Uber", AutoInserted: true, Type: "Variáveis"},                              // excluded from review.csv
		{Item: "Cinema", AutoInserted: false, Subcategory: "Cinema", Type: "Extras"},       // included
		{Item: "Unknown", AutoInserted: false, Subcategory: "???", Category: "", Type: ""}, // included, type empty
	}

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
