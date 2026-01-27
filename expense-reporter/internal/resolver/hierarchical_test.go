package resolver

import (
	"testing"
)

// TestResolveSubcategoryWithPath tests hierarchical path resolution
func TestResolveSubcategoryWithPath(t *testing.T) {
	// Create mock data with ambiguous subcategory
	flatMappings := map[string][]SubcategoryMapping{
		"diarista": {
			{
				Subcategory: "Diarista",
				SheetName:   "Fixas",
				Category:    "Habitação",
				RowNumber:   10,
				TotalRow:    20,
			},
			{
				Subcategory: "Diarista",
				SheetName:   "Variáveis",
				Category:    "Casa",
				RowNumber:   30,
				TotalRow:    40,
			},
		},
		"uber/taxi": {
			{
				Subcategory: "Uber/Taxi",
				SheetName:   "Variáveis",
				Category:    "Transporte",
				RowNumber:   50,
				TotalRow:    60,
			},
		},
	}

	// Build hierarchical index
	index := &PathIndex{
		BySubcategory: flatMappings,
		ByFullPath:    make(map[string]*SubcategoryMapping),
		By2Level:      make(map[string][]SubcategoryMapping),
		By1Level:      make(map[string][]SubcategoryMapping),
	}

	for _, mappings := range flatMappings {
		for i := range mappings {
			m := mappings[i]
			subKey := NormalizePath(m.Subcategory)
			index.By1Level[subKey] = append(index.By1Level[subKey], m)

			if m.Category != "" {
				twoKey := NormalizePath(m.Category + "," + m.Subcategory)
				index.By2Level[twoKey] = append(index.By2Level[twoKey], m)
			}

			fullKey := NormalizePath(m.SheetName + "," + m.Category + "," + m.Subcategory)
			index.ByFullPath[fullKey] = &m
		}
	}

	tests := []struct {
		name          string
		input         string
		wantSheet     string
		wantCategory  string
		wantRow       int
		wantAmbiguous bool
		wantErr       bool
	}{
		// Progressive resolution
		{"1-level ambiguous", "Diarista", "", "", 0, true, false},
		{"2-level Habitação", "Habitação,Diarista", "Fixas", "Habitação", 10, false, false},
		{"2-level Casa", "Casa,Diarista", "Variáveis", "Casa", 30, false, false},
		{"3-level Fixas", "Fixas,Habitação,Diarista", "Fixas", "Habitação", 10, false, false},
		{"3-level Variáveis", "Variáveis,Casa,Diarista", "Variáveis", "Casa", 30, false, false},

		// Case insensitive
		{"uppercase", "FIXAS,HABITAÇÃO,DIARISTA", "Fixas", "Habitação", 10, false, false},
		{"mixed case", "FiXaS,HaBiTaÇãO,DiArIsTa", "Fixas", "Habitação", 10, false, false},

		// Space normalization
		{"spaces", "Fixas , Habitação , Diarista", "Fixas", "Habitação", 10, false, false},

		// Unique subcategory
		{"unique", "Uber/Taxi", "Variáveis", "Transporte", 50, false, false},

		// Errors
		{"not found", "NonExistent", "", "", 0, false, true},
		{"wrong path", "WrongCat,Diarista", "", "", 0, false, true},
		{"too deep", "A,B,C,D", "", "", 0, false, true},
		{"empty", "", "", "", 0, false, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, isAmbiguous, err := ResolveSubcategoryWithPath(index, tt.input)

			if (err != nil) != tt.wantErr {
				t.Errorf("error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr {
				return
			}

			if isAmbiguous != tt.wantAmbiguous {
				t.Errorf("isAmbiguous = %v, want %v", isAmbiguous, tt.wantAmbiguous)
			}

			if tt.wantAmbiguous {
				return
			}

			if result.SheetName != tt.wantSheet {
				t.Errorf("SheetName = %v, want %v", result.SheetName, tt.wantSheet)
			}
			if result.Category != tt.wantCategory {
				t.Errorf("Category = %v, want %v", result.Category, tt.wantCategory)
			}
			if result.RowNumber != tt.wantRow {
				t.Errorf("RowNumber = %v, want %v", result.RowNumber, tt.wantRow)
			}
		})
	}
}
