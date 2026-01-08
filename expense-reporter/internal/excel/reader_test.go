package excel

import (
	"testing"
)

// TDD RED: Test loading reference sheet from actual Excel file
func TestLoadReferenceSheet(t *testing.T) {
	// This will test with the actual Excel file
	workbookPath := "Z:\\Meu Drive\\controle\\code\\Planilha_BMeFBovespa_Leandro_OrcamentoPessoal-2025.xlsx"

	mappings, err := LoadReferenceSheet(workbookPath)
	if err != nil {
		t.Fatalf("LoadReferenceSheet() error = %v", err)
	}

	// Test that we got expected number of entries
	if len(mappings) == 0 {
		t.Error("LoadReferenceSheet() returned empty mappings")
	}

	// Test specific known subcategories
	knownSubcats := []string{"Uber/Taxi", "Supermercado", "Diarista"}
	for _, subcat := range knownSubcats {
		if _, exists := mappings[subcat]; !exists {
			t.Errorf("LoadReferenceSheet() missing expected subcategory: %s", subcat)
		}
	}

	// Test that Uber/Taxi maps to correct sheet
	if options, exists := mappings["Uber/Taxi"]; exists {
		if len(options) == 0 {
			t.Error("Uber/Taxi has no mapping options")
		} else if options[0].SheetName != "Variáveis" {
			t.Errorf("Uber/Taxi SheetName = %v, want Variáveis", options[0].SheetName)
		}
	}

	// Test ambiguous subcategory (should have multiple entries)
	if options, exists := mappings["Dentista"]; exists {
		if len(options) < 2 {
			t.Errorf("Dentista should be ambiguous with at least 2 options, got %d", len(options))
		}
	}
}

// Test finding subcategory row in actual Excel sheet
func TestFindSubcategoryRow(t *testing.T) {
	workbookPath := "Z:\\Meu Drive\\controle\\code\\Planilha_BMeFBovespa_Leandro_OrcamentoPessoal-2025.xlsx"

	tests := []struct {
		name        string
		sheetName   string
		subcategory string
		wantFound   bool
		minRow      int // Should be at least this row
	}{
		{
			name:        "Uber/Taxi in Variáveis",
			sheetName:   "Variáveis",
			subcategory: "Uber/Taxi",
			wantFound:   true,
			minRow:      3, // Should be after headers
		},
		{
			name:        "Supermercado in Variáveis",
			sheetName:   "Variáveis",
			subcategory: "Supermercado",
			wantFound:   true,
			minRow:      3,
		},
		{
			name:        "Diarista in Fixas",
			sheetName:   "Fixas",
			subcategory: "Diarista",
			wantFound:   true,
			minRow:      3,
		},
		{
			name:        "NonExistent in Variáveis",
			sheetName:   "Variáveis",
			subcategory: "NonExistent",
			wantFound:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			row, err := FindSubcategoryRow(workbookPath, tt.sheetName, tt.subcategory)

			if tt.wantFound {
				if err != nil {
					t.Errorf("FindSubcategoryRow() unexpected error = %v", err)
					return
				}
				if row < tt.minRow {
					t.Errorf("FindSubcategoryRow() row = %v, want >= %v", row, tt.minRow)
				}
			} else {
				if err == nil {
					t.Errorf("FindSubcategoryRow() expected error for non-existent subcategory, got row %d", row)
				}
			}
		})
	}
}

// TDD RED: Test batch finding subcategory rows
func TestFindSubcategoryRowBatch(t *testing.T) {
	workbookPath := "Z:\\Meu Drive\\controle\\code\\Planilha_BMeFBovespa_Leandro_OrcamentoPessoal-2025.xlsx"

	tests := []struct {
		name     string
		requests []SubcategoryLookupRequest
		wantErr  bool
		validate func(t *testing.T, results map[string]map[string]int)
	}{
		{
			name:     "empty requests",
			requests: []SubcategoryLookupRequest{},
			wantErr:  false,
			validate: func(t *testing.T, results map[string]map[string]int) {
				if len(results) != 0 {
					t.Errorf("Expected empty results, got %d entries", len(results))
				}
			},
		},
		{
			name: "single request",
			requests: []SubcategoryLookupRequest{
				{SheetName: "Variáveis", Subcategory: "Uber/Taxi"},
			},
			wantErr: false,
			validate: func(t *testing.T, results map[string]map[string]int) {
				if row, ok := results["Variáveis"]["Uber/Taxi"]; !ok || row < 3 {
					t.Errorf("Expected Uber/Taxi row >= 3, got %v", row)
				}
			},
		},
		{
			name: "multiple same sheet",
			requests: []SubcategoryLookupRequest{
				{SheetName: "Variáveis", Subcategory: "Uber/Taxi"},
				{SheetName: "Variáveis", Subcategory: "Supermercado"},
			},
			wantErr: false,
			validate: func(t *testing.T, results map[string]map[string]int) {
				if _, ok := results["Variáveis"]["Uber/Taxi"]; !ok {
					t.Error("Expected Uber/Taxi in results")
				}
				if _, ok := results["Variáveis"]["Supermercado"]; !ok {
					t.Error("Expected Supermercado in results")
				}
			},
		},
		{
			name: "multiple different sheets",
			requests: []SubcategoryLookupRequest{
				{SheetName: "Variáveis", Subcategory: "Uber/Taxi"},
				{SheetName: "Fixas", Subcategory: "Diarista"},
			},
			wantErr: false,
			validate: func(t *testing.T, results map[string]map[string]int) {
				if _, ok := results["Variáveis"]["Uber/Taxi"]; !ok {
					t.Error("Expected Uber/Taxi in Variáveis")
				}
				if _, ok := results["Fixas"]["Diarista"]; !ok {
					t.Error("Expected Diarista in Fixas")
				}
			},
		},
		{
			name: "missing subcategory",
			requests: []SubcategoryLookupRequest{
				{SheetName: "Variáveis", Subcategory: "NonExistent"},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results, err := FindSubcategoryRowBatch(workbookPath, tt.requests)

			if (err != nil) != tt.wantErr {
				t.Errorf("FindSubcategoryRowBatch() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if err == nil && tt.validate != nil {
				tt.validate(t, results)
			}
		})
	}
}
