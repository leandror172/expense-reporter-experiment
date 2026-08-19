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
			csvContent: "item;date;value;subcategory;category;confidence;auto_inserted;type;keyword_hint\nUber Centro;15/05;35,50;Taxi;Transporte;0.95;true;;",
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
			csvContent: "item;date;value;subcategory;category;confidence;auto_inserted;type;keyword_hint\nAluguel;05/01;2500,00;Aluguel;Moradia;0.95;false;Fixas;",
			wantCount:  1,
			assertions: func(t *testing.T, entries []QueueEntry) {
				e := entries[0]
				assert.Equal(t, "Fixas", e.Predicted.Type)
				assert.Equal(t, "Moradia", e.Predicted.Category)
				assert.Equal(t, "Aluguel", e.Predicted.Subcategory)
			},
		},
		{
			name:       "blank lines skipped",
			csvContent: "item;date;value;subcategory;category;confidence;auto_inserted;type;keyword_hint\n\nUber Centro;15/05;35,50;Taxi;Transporte;0.95;true;;\n\nUber Centro 2;16/05;40,00;Taxi;Transporte;0.90;false;;\n\nUber Centro 3;17/05;45,00;Taxi;Transporte;0.85;true;;",
			wantCount:  3,
		},
		{
			name:       "installment value parsed",
			csvContent: "item;date;value;subcategory;category;confidence;auto_inserted;type;keyword_hint\nTest Item;15/05;250,00/2;Taxi;Transporte;0.95;true;;",
			wantCount:  1,
			assertions: func(t *testing.T, entries []QueueEntry) {
				assert.Equal(t, "250,00/2", entries[0].RawValue)
				assert.Equal(t, 125.0, entries[0].Value)
			},
		},
		{
			name:          "malformed confidence value",
			csvContent:    "item;date;value;subcategory;category;confidence;auto_inserted;type;keyword_hint\nTest Item;15/05;35,50;Taxi;Transporte;abc;true;;",
			wantError:     true,
			errorContains: "confidence",
		},
		{
			// The 1/0 spelling was accepted here for months while no producer ever wrote
			// it — the reader's only inputs were fixtures authored to satisfy the reader
			// (T-54). Pinning the rejection keeps it from creeping back as a second
			// accepted spelling that nothing emits and nothing covers.
			name:          "legacy 1/0 spelling is rejected",
			csvContent:    "item;date;value;subcategory;category;confidence;auto_inserted;type;keyword_hint\nTest Item;15/05;35,50;Taxi;Transporte;0.95;1;;",
			wantError:     true,
			errorContains: `invalid auto_inserted value "1"`,
		},
		{
			name:          "bad auto_inserted",
			csvContent:    "item;date;value;subcategory;category;confidence;auto_inserted;type;keyword_hint\nTest Item;15/05;35,50;Taxi;Transporte;0.95;X;;",
			wantError:     true,
			errorContains: "auto_inserted",
		},
		{
			name:          "wrong field count",
			csvContent:    "item;date;value;subcategory;category;confidence;auto_inserted;type;keyword_hint\nTest Item;15/05;35,50;Taxi;Transporte",
			wantError:     true,
			errorContains: "expected 9 fields",
		},
		{
			name:       "header only returns empty slice",
			csvContent: "item;date;value;subcategory;category;confidence;auto_inserted;type;keyword_hint",
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

			entries, _, err := ReadQueue(csvPath)

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

// TestReadQueue_UnparsedRowsAreReturnedNotFatal pins the S1 contract at unit level: the
// shape batch-auto writes for a row it could not parse (raw text in the item column, empty
// date and value) is returned as unreviewable instead of failing the whole read.
//
// The tolerance is deliberately narrow — see the sibling table's "malformed confidence",
// "bad auto_inserted" and "wrong field count" cases, which still hard-error. An empty
// date/value pair is a documented producer output; corruption is not.
func TestReadQueue_UnparsedRowsAreReturnedNotFatal(t *testing.T) {
	t.Parallel()

	const content = "item;date;value;subcategory;category;confidence;auto_inserted;type;keyword_hint\n" +
		"Uber Centro;15/05;35,50;Taxi;Transporte;0.95;true;;\n" +
		`"Anita;Elô ADM;09/01;405,25";;;;;0.0000;false;;` + "\n" +
		"Aluguel;05/01;2500,00;Aluguel;Moradia;0.95;false;Fixas;"

	csvPath := filepath.Join(t.TempDir(), "classified.csv")
	require.NoError(t, os.WriteFile(csvPath, []byte(content), 0o644))

	entries, unreviewable, err := ReadQueue(csvPath)

	require.NoError(t, err, "an unparsed row must not fail the whole read")
	assert.Len(t, entries, 2, "the reviewable rows are still queued")
	require.Len(t, unreviewable, 1, "the unparsed row is reported back to the caller")
	assert.Equal(t, "Anita;Elô ADM;09/01;405,25", unreviewable[0],
		"the caller gets the original text, which is what the user needs to fix the source")
}
