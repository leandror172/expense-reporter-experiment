package review

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReadQueue(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		csvContent    string
		missingFile   bool
		wantCount     int
		wantError     bool
		errorContains string
		assertions    func(t *testing.T, entries []QueueEntry)
	}{
		{
			name:       "good row without type",
			csvContent: "item;date;value;subcategory;category;confidence;auto_inserted;type\nUber Centro;15/05;35,50;Taxi;Transporte;0.95;1;",
			wantCount:  1,
			assertions: func(t *testing.T, entries []QueueEntry) {
				e := entries[0]
				assert.Equal(t, "Uber Centro", e.Item)
				assert.Equal(t, "15/05", e.Date)
				assert.Equal(t, "35,50", e.RawValue)
				assert.Equal(t, 35.50, e.Value)
				assert.Equal(t, 0.95, e.Confidence)
				assert.True(t, e.AutoInserted)
				assert.Equal(t, "Transporte", e.Predicted.Category)
				assert.Equal(t, "Taxi", e.Predicted.Subcategory)
				assert.Empty(t, e.Predicted.Type)
				assert.NotEmpty(t, e.ID)
			},
		},
		{
			name:       "good row with type populates Predicted.Type",
			csvContent: "item;date;value;subcategory;category;confidence;auto_inserted;type\nAluguel;05/01;2500,00;Aluguel;Moradia;0.95;0;Fixas",
			wantCount:  1,
			assertions: func(t *testing.T, entries []QueueEntry) {
				e := entries[0]
				assert.Equal(t, "Fixas", e.Predicted.Type)
				assert.Equal(t, "Moradia", e.Predicted.Category)
				assert.Equal(t, "Aluguel", e.Predicted.Subcategory)
			},
		},
		{
			name:      "blank lines skipped",
			csvContent: "item;date;value;subcategory;category;confidence;auto_inserted;type\n\nUber Centro;15/05;35,50;Taxi;Transporte;0.95;1;\n\nUber Centro 2;16/05;40,00;Taxi;Transporte;0.90;0;\n\nUber Centro 3;17/05;45,00;Taxi;Transporte;0.85;1;",
			wantCount: 3,
		},
		{
			name:       "installment value parsed",
			csvContent: "item;date;value;subcategory;category;confidence;auto_inserted;type\nTest Item;15/05;250,00/2;Taxi;Transporte;0.95;1;",
			wantCount:  1,
			assertions: func(t *testing.T, entries []QueueEntry) {
				assert.Equal(t, "250,00/2", entries[0].RawValue)
				assert.Equal(t, 125.0, entries[0].Value)
			},
		},
		{
			name:          "malformed confidence value",
			csvContent:    "item;date;value;subcategory;category;confidence;auto_inserted;type\nTest Item;15/05;35,50;Taxi;Transporte;abc;1;",
			wantError:     true,
			errorContains: "confidence",
		},
		{
			name:          "bad auto_inserted",
			csvContent:    "item;date;value;subcategory;category;confidence;auto_inserted;type\nTest Item;15/05;35,50;Taxi;Transporte;0.95;X;",
			wantError:     true,
			errorContains: "auto_inserted",
		},
		{
			name:          "wrong field count",
			csvContent:    "item;date;value;subcategory;category;confidence;auto_inserted;type\nTest Item;15/05;35,50;Taxi;Transporte",
			wantError:     true,
			errorContains: "expected 8 fields",
		},
		{
			name:       "header only returns empty slice",
			csvContent: "item;date;value;subcategory;category;confidence;auto_inserted;type",
			wantCount:  0,
		},
		{
			name:        "missing file returns error",
			missingFile: true,
			wantError:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var csvPath string
			if tt.missingFile {
				csvPath = filepath.Join(t.TempDir(), "nonexistent.csv")
			} else {
				csvPath = filepath.Join(t.TempDir(), "test.csv")
				require.NoError(t, os.WriteFile(csvPath, []byte(tt.csvContent), 0o644))
			}

			entries, err := ReadQueue(csvPath)

			if tt.wantError {
				require.Error(t, err)
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains)
				}
				return
			}

			require.NoError(t, err)
			assert.Len(t, entries, tt.wantCount)
			if tt.assertions != nil {
				tt.assertions(t, entries)
			}
		})
	}
}
