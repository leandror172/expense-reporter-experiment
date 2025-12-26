package cmd

import (
	"expense-reporter/internal/batch"
	"expense-reporter/internal/models"
	"expense-reporter/internal/resolver"
	"testing"
	"time"
)

// Test collectAmbiguousEntries function
func TestCollectAmbiguousEntries(t *testing.T) {
	tests := []struct {
		name      string
		summary   *batch.BatchSummary
		wantCount int
	}{
		{
			name: "single ambiguous entry",
			summary: &batch.BatchSummary{
				AmbiguousCount: 1,
				Results: []batch.BatchResult{
					{
						LineNumber:    1,
						ExpenseString: "Consulta;15/04;100,00;Dentista",
						IsAmbiguous:   true,
						Expense: &models.Expense{
							Item:        "Consulta",
							Date:        time.Date(2025, 4, 15, 0, 0, 0, 0, time.UTC),
							Value:       100.00,
							Subcategory: "Dentista",
						},
						AmbiguousOpts: []resolver.SubcategoryMapping{
							{Subcategory: "Dentista", SheetName: "Variáveis"},
							{Subcategory: "Dentista", SheetName: "Extras"},
						},
					},
				},
			},
			wantCount: 1,
		},
		{
			name: "multiple ambiguous entries",
			summary: &batch.BatchSummary{
				AmbiguousCount: 2,
				Results: []batch.BatchResult{
					{
						LineNumber:    1,
						ExpenseString: "Consulta;15/04;100,00;Dentista",
						IsAmbiguous:   true,
						Expense: &models.Expense{
							Item:        "Consulta",
							Date:        time.Date(2025, 4, 15, 0, 0, 0, 0, time.UTC),
							Value:       100.00,
							Subcategory: "Dentista",
						},
						AmbiguousOpts: []resolver.SubcategoryMapping{
							{Subcategory: "Dentista", SheetName: "Variáveis"},
							{Subcategory: "Dentista", SheetName: "Extras"},
						},
					},
					{
						LineNumber:    3,
						ExpenseString: "Recarga;20/04;120,00;Gás",
						IsAmbiguous:   true,
						Expense: &models.Expense{
							Item:        "Recarga",
							Date:        time.Date(2025, 4, 20, 0, 0, 0, 0, time.UTC),
							Value:       120.00,
							Subcategory: "Gás",
						},
						AmbiguousOpts: []resolver.SubcategoryMapping{
							{Subcategory: "Gás", SheetName: "Variáveis"},
							{Subcategory: "Gás", SheetName: "Fixas"},
						},
					},
				},
			},
			wantCount: 2,
		},
		{
			name: "mixed results - only extract ambiguous",
			summary: &batch.BatchSummary{
				SuccessCount:   1,
				ErrorCount:     1,
				AmbiguousCount: 1,
				Results: []batch.BatchResult{
					{
						LineNumber:    1,
						ExpenseString: "Uber;15/04;35,50;Uber/Taxi",
						Success:       true,
					},
					{
						LineNumber:    2,
						ExpenseString: "Invalid",
						Success:       false,
					},
					{
						LineNumber:    3,
						ExpenseString: "Consulta;15/04;100,00;Dentista",
						IsAmbiguous:   true,
						Expense: &models.Expense{
							Subcategory: "Dentista",
						},
						AmbiguousOpts: []resolver.SubcategoryMapping{
							{Subcategory: "Dentista", SheetName: "Variáveis"},
							{Subcategory: "Dentista", SheetName: "Extras"},
						},
					},
				},
			},
			wantCount: 1,
		},
		{
			name: "no ambiguous entries",
			summary: &batch.BatchSummary{
				SuccessCount: 2,
				Results: []batch.BatchResult{
					{LineNumber: 1, ExpenseString: "Uber;15/04;35,50;Uber/Taxi", Success: true},
					{LineNumber: 2, ExpenseString: "Compras;03/01;150,00;Supermercado", Success: true},
				},
			},
			wantCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entries := collectAmbiguousEntries(tt.summary)

			if len(entries) != tt.wantCount {
				t.Errorf("collectAmbiguousEntries() returned %d entries, want %d", len(entries), tt.wantCount)
			}

			// Verify each entry has the required fields
			for i, entry := range entries {
				if entry.ExpenseString == "" {
					t.Errorf("Entry %d: ExpenseString is empty", i)
				}
				if entry.Subcategory == "" {
					t.Errorf("Entry %d: Subcategory is empty", i)
				}
				if len(entry.SheetOptions) < 2 {
					t.Errorf("Entry %d: Should have at least 2 SheetOptions, got %d", i, len(entry.SheetOptions))
				}
			}
		})
	}
}

// Test that collectAmbiguousEntries extracts correct sheet names
func TestCollectAmbiguousEntries_SheetNames(t *testing.T) {
	summary := &batch.BatchSummary{
		AmbiguousCount: 1,
		Results: []batch.BatchResult{
			{
				LineNumber:    1,
				ExpenseString: "Consulta;15/04;100,00;Dentista",
				IsAmbiguous:   true,
				Expense: &models.Expense{
					Subcategory: "Dentista",
				},
				AmbiguousOpts: []resolver.SubcategoryMapping{
					{Subcategory: "Dentista", SheetName: "Variáveis", Category: "Saúde"},
					{Subcategory: "Dentista", SheetName: "Extras", Category: "Saúde"},
				},
			},
		},
	}

	entries := collectAmbiguousEntries(summary)

	if len(entries) != 1 {
		t.Fatalf("Expected 1 entry, got %d", len(entries))
	}

	entry := entries[0]

	// Verify sheet names are extracted correctly
	if len(entry.SheetOptions) != 2 {
		t.Fatalf("Expected 2 sheet options, got %d", len(entry.SheetOptions))
	}

	if entry.SheetOptions[0] != "Variáveis" {
		t.Errorf("First sheet option = %q, want %q", entry.SheetOptions[0], "Variáveis")
	}

	if entry.SheetOptions[1] != "Extras" {
		t.Errorf("Second sheet option = %q, want %q", entry.SheetOptions[1], "Extras")
	}

	// Verify other fields
	if entry.Subcategory != "Dentista" {
		t.Errorf("Subcategory = %q, want %q", entry.Subcategory, "Dentista")
	}

	if entry.ExpenseString != "Consulta;15/04;100,00;Dentista" {
		t.Errorf("ExpenseString = %q, want %q", entry.ExpenseString, "Consulta;15/04;100,00;Dentista")
	}
}

// Test batch command flags are registered
func TestBatchCommand_Flags(t *testing.T) {
	// Verify backup flag exists
	backupFlagVal := batchCmd.Flags().Lookup("backup")
	if backupFlagVal == nil {
		t.Error("--backup flag not registered")
	}

	// Verify report flag exists
	reportFlagVal := batchCmd.Flags().Lookup("report")
	if reportFlagVal == nil {
		t.Error("--report flag not registered")
	}

	// Verify silent flag exists
	silentFlagVal := batchCmd.Flags().Lookup("silent")
	if silentFlagVal == nil {
		t.Error("--silent flag not registered")
	}
}

// Test batch command usage and help text
func TestBatchCommand_UsageText(t *testing.T) {
	if batchCmd.Use != "batch <csv_file>" {
		t.Errorf("Batch command Use = %q, want %q", batchCmd.Use, "batch <csv_file>")
	}

	if batchCmd.Short == "" {
		t.Error("Batch command should have Short description")
	}

	if batchCmd.Long == "" {
		t.Error("Batch command should have Long description")
	}
}
