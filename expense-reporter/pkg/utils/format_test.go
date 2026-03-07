package utils

import "testing"

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
		got := FormatBRValue(tt.value)
		if got != tt.want {
			t.Errorf("FormatBRValue(%v) = %q, want %q", tt.value, got, tt.want)
		}
	}
}

func TestBuildInsertString(t *testing.T) {
	tests := []struct {
		item        string
		date        string
		value       float64
		subcategory string
		want        string
	}{
		{"Uber Centro", "15/04", 35.50, "Uber/Taxi", "Uber Centro;15/04;35,50;Uber/Taxi"},
		{"Diarista Letícia", "05/01", 160.0, "Diarista", "Diarista Letícia;05/01;160,00;Diarista"},
		{"Supermercado Extra", "03/01", 210.99, "Supermercado", "Supermercado Extra;03/01;210,99;Supermercado"},
	}
	for _, tt := range tests {
		got := BuildInsertString(tt.item, tt.date, tt.value, tt.subcategory)
		if got != tt.want {
			t.Errorf("BuildInsertString() = %q, want %q", got, tt.want)
		}
	}
}
