package resolver

import (
	"testing"
)

// TDD RED: Test actual subcategory resolution logic
func TestResolveSubcategory(t *testing.T) {
	// Mock reference data simulating what we'd get from Excel
	mockMappings := map[string][]SubcategoryMapping{
		"Uber/Taxi": {
			{
				Subcategory: "Uber/Taxi",
				SheetName:   "Variáveis",
				Category:    "Transporte",
				RowNumber:   97,
			},
		},
		"Supermercado": {
			{
				Subcategory: "Supermercado",
				SheetName:   "Variáveis",
				Category:    "Alimentação / Limpeza",
				RowNumber:   17,
			},
		},
		// Ambiguous - appears in two places
		"Dentista": {
			{
				Subcategory: "Dentista",
				SheetName:   "Variáveis",
				Category:    "Saúde",
				RowNumber:   168,
			},
			{
				Subcategory: "Dentista",
				SheetName:   "Extras",
				Category:    "Saúde",
				RowNumber:   9,
			},
		},
		// Smart matching - parent with children
		"Orion": {
			{
				Subcategory: "Orion",
				SheetName:   "Variáveis",
				Category:    "Pets",
				RowNumber:   65,
			},
		},
	}

	tests := []struct {
		name          string
		subcategory   string
		userChoice    int // For ambiguous cases
		wantSheetName string
		wantCategory  string
		wantRow       int
		wantAmbiguous bool
		wantErr       bool
	}{
		{
			name:          "unique subcategory - Uber/Taxi",
			subcategory:   "Uber/Taxi",
			wantSheetName: "Variáveis",
			wantCategory:  "Transporte",
			wantRow:       97,
			wantAmbiguous: false,
			wantErr:       false,
		},
		{
			name:          "unique subcategory - Supermercado",
			subcategory:   "Supermercado",
			wantSheetName: "Variáveis",
			wantCategory:  "Alimentação / Limpeza",
			wantRow:       17,
			wantAmbiguous: false,
			wantErr:       false,
		},
		{
			name:          "ambiguous - Dentista (detect only)",
			subcategory:   "Dentista",
			wantAmbiguous: true,
			wantErr:       false, // Not error, just needs user choice
		},
		{
			name:          "smart match - Orion - Consultas finds Orion",
			subcategory:   "Orion - Consultas",
			wantSheetName: "Variáveis",
			wantCategory:  "Pets",
			wantRow:       65,
			wantAmbiguous: false,
			wantErr:       false,
		},
		{
			name:        "not found subcategory",
			subcategory: "NonExistent",
			wantErr:     true,
		},
		{
			name:        "empty subcategory",
			subcategory: "",
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, isAmbiguous, err := ResolveSubcategory(mockMappings, tt.subcategory)

			if (err != nil) != tt.wantErr {
				t.Errorf("ResolveSubcategory() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr {
				return
			}

			if isAmbiguous != tt.wantAmbiguous {
				t.Errorf("ResolveSubcategory() isAmbiguous = %v, want %v", isAmbiguous, tt.wantAmbiguous)
			}

			if tt.wantAmbiguous {
				// For ambiguous, just verify we got multiple options
				options := mockMappings[tt.subcategory]
				if len(options) <= 1 {
					t.Errorf("Expected multiple options for ambiguous subcategory, got %d", len(options))
				}
				return
			}

			// Verify non-ambiguous result
			if result.SheetName != tt.wantSheetName {
				t.Errorf("ResolveSubcategory() SheetName = %v, want %v", result.SheetName, tt.wantSheetName)
			}
			if result.Category != tt.wantCategory {
				t.Errorf("ResolveSubcategory() Category = %v, want %v", result.Category, tt.wantCategory)
			}
			if result.RowNumber != tt.wantRow {
				t.Errorf("ResolveSubcategory() RowNumber = %v, want %v", result.RowNumber, tt.wantRow)
			}
		})
	}
}

// Test extracting parent from detailed subcategory
func TestExtractParentSubcategory(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "detailed with dash - Orion - Consultas",
			input: "Orion - Consultas",
			want:  "Orion",
		},
		{
			name:  "detailed with dash - Lilly - Ração",
			input: "Lilly - Ração",
			want:  "Lilly",
		},
		{
			name:  "no dash - Uber/Taxi",
			input: "Uber/Taxi",
			want:  "Uber/Taxi",
		},
		{
			name:  "no dash - Supermercado",
			input: "Supermercado",
			want:  "Supermercado",
		},
		{
			name:  "multiple dashes - take first",
			input: "Orion - Consultas - Extra",
			want:  "Orion",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExtractParentSubcategory(tt.input)
			if got != tt.want {
				t.Errorf("ExtractParentSubcategory() = %v, want %v", got, tt.want)
			}
		})
	}
}

// Test getting all options for ambiguous subcategory
func TestGetAmbiguousOptions(t *testing.T) {
	mockMappings := map[string][]SubcategoryMapping{
		"Dentista": {
			{
				Subcategory: "Dentista",
				SheetName:   "Variáveis",
				Category:    "Saúde",
			},
			{
				Subcategory: "Dentista",
				SheetName:   "Extras",
				Category:    "Saúde",
			},
		},
	}

	options := GetAmbiguousOptions(mockMappings, "Dentista")

	if len(options) != 2 {
		t.Errorf("GetAmbiguousOptions() returned %d options, want 2", len(options))
	}

	// Verify we got both options
	hasVariaveis := false
	hasExtras := false
	for _, opt := range options {
		if opt.SheetName == "Variáveis" {
			hasVariaveis = true
		}
		if opt.SheetName == "Extras" {
			hasExtras = true
		}
	}

	if !hasVariaveis || !hasExtras {
		t.Errorf("GetAmbiguousOptions() missing expected sheet options")
	}
}

// Test NormalizePath function
func TestNormalizePath(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"lowercase conversion", "Fixas,Habitação,Diarista", "fixas,habitação,diarista"},
		{"uppercase conversion", "FIXAS,HABITAÇÃO,DIARISTA", "fixas,habitação,diarista"},
		{"mixed case", "FiXaS,HaBiTaÇãO,DiArIsTa", "fixas,habitação,diarista"},
		{"trim spaces", "Fixas , Habitação , Diarista", "fixas,habitação,diarista"},
		{"extra spaces", "  Fixas  ,  Habitação  ", "fixas,habitação"},
		{"single level", "Diarista", "diarista"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NormalizePath(tt.input)
			if got != tt.expected {
				t.Errorf("NormalizePath(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

// Test SplitPath function
func TestSplitPath(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{"three levels", "Fixas,Habitação,Diarista", []string{"fixas", "habitação", "diarista"}},
		{"two levels", "Habitação,Diarista", []string{"habitação", "diarista"}},
		{"one level", "Diarista", []string{"diarista"}},
		{"empty", "", []string{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SplitPath(tt.input)
			if len(got) != len(tt.expected) {
				t.Errorf("SplitPath(%q) length = %d, want %d", tt.input, len(got), len(tt.expected))
				return
			}
			for i := range got {
				if got[i] != tt.expected[i] {
					t.Errorf("SplitPath(%q)[%d] = %q, want %q", tt.input, i, got[i], tt.expected[i])
				}
			}
		})
	}
}
