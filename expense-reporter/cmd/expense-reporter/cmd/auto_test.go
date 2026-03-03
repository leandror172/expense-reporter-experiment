package cmd

import (
	"expense-reporter/internal/classifier"
	"strings"
	"testing"
)

// --- formatBRValue ---

func TestFormatBRValue(t *testing.T) {
	tests := []struct {
		value float64
		want  string
	}{
		{35.50, "35,50"},
		{35.5, "35,50"},
		{160.0, "160,00"},
		{1234.56, "1234,56"},
		{0.99, "0,99"},
		{1000.0, "1000,00"},
	}
	for _, tt := range tests {
		got := formatBRValue(tt.value)
		if got != tt.want {
			t.Errorf("formatBRValue(%v) = %q, want %q", tt.value, got, tt.want)
		}
	}
}

// --- buildInsertString ---

func TestBuildInsertString(t *testing.T) {
	tests := []struct {
		item        string
		date        string
		value       float64
		subcategory string
		want        string
	}{
		{
			item:        "Uber Centro",
			date:        "15/04",
			value:       35.50,
			subcategory: "Uber/Taxi",
			want:        "Uber Centro;15/04;35,50;Uber/Taxi",
		},
		{
			item:        "Diarista Letícia",
			date:        "05/01",
			value:       160.0,
			subcategory: "Diarista",
			want:        "Diarista Letícia;05/01;160,00;Diarista",
		},
		{
			item:        "Supermercado Extra",
			date:        "03/01",
			value:       210.99,
			subcategory: "Supermercado",
			want:        "Supermercado Extra;03/01;210,99;Supermercado",
		},
	}
	for _, tt := range tests {
		got := buildInsertString(tt.item, tt.date, tt.value, tt.subcategory)
		if got != tt.want {
			t.Errorf("buildInsertString() = %q, want %q", got, tt.want)
		}
	}
}

// --- isAutoInsertable ---

func TestIsAutoInsertable(t *testing.T) {
	tests := []struct {
		name        string
		subcategory string
		confidence  float64
		excluded    []string
		want        bool
	}{
		{"high confidence, specific subcategory", "Uber/Taxi", 0.92, []string{"Diversos"}, true},
		{"high confidence, Diarista", "Diarista", 0.95, []string{"Diversos"}, true},
		{"exactly at threshold", "Supermercado", 0.85, []string{"Diversos"}, true},
		{"high confidence, Diversos — must not auto-insert", "Diversos", 0.95, []string{"Diversos"}, false},
		{"high confidence, Diversos at 100%", "Diversos", 1.0, []string{"Diversos"}, false},
		{"below threshold, specific subcategory", "Uber/Taxi", 0.84, []string{"Diversos"}, false},
		{"below threshold, Diversos", "Diversos", 0.60, []string{"Diversos"}, false},
		{"zero confidence", "Diarista", 0.0, []string{"Diversos"}, false},
		{"empty exclusion list — Diversos allowed through", "Diversos", 0.95, []string{}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := classifier.Result{Subcategory: tt.subcategory, Confidence: tt.confidence}
			got := isAutoInsertable(r, tt.excluded)
			if got != tt.want {
				t.Errorf("isAutoInsertable(%q, %.2f, %v) = %v, want %v",
					tt.subcategory, tt.confidence, tt.excluded, got, tt.want)
			}
		})
	}
}

// --- confirmInsert (interactive prompt with reader) ---

func TestConfirmInsert_Yes(t *testing.T) {
	for _, input := range []string{"y\n", "Y\n", "yes\n", "YES\n"} {
		r := strings.NewReader(input)
		if !confirmInsert(r) {
			t.Errorf("confirmInsert(%q) = false, want true", input)
		}
	}
}

func TestConfirmInsert_No(t *testing.T) {
	for _, input := range []string{"n\n", "N\n", "no\n", "\n", "anything\n"} {
		r := strings.NewReader(input)
		if confirmInsert(r) {
			t.Errorf("confirmInsert(%q) = true, want false", input)
		}
	}
}
