package classifier

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadTrainingExamples(t *testing.T) {
	const validJSON = `{
		"metadata": {},
		"expenses": [
			{"id":1,"item":"Uber","date":"2024-03-15","value":25.50,"subcategory":"Uber","category":"Transporte"},
			{"id":2,"item":"Extra","date":"2024-01-05","value":89.20,"subcategory":"Supermercado","category":"Alimentação"}
		]
	}`

	tests := []struct {
		name    string
		setup   func(dir string)
		wantLen int
		wantNil bool
		wantErr bool
		// spot-check first entry
		wantDate string
		wantSrc  ExampleSource
	}{
		{
			name:     "valid file",
			setup:    func(dir string) { writeFile(t, dir+"/training_data_complete.json", validJSON) },
			wantLen:  2,
			wantDate: "15/03",
			wantSrc:  SourceTraining,
		},
		{
			name:    "missing file returns nil nil",
			setup:   func(dir string) {},
			wantNil: true,
		},
		{
			name:    "malformed json returns error",
			setup:   func(dir string) { writeFile(t, dir+"/training_data_complete.json", "not json") },
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			tt.setup(dir)

			result, err := LoadTrainingExamples(dir)

			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			if tt.wantNil {
				assert.Nil(t, result)
				return
			}
			require.Len(t, result, tt.wantLen)
			assert.Equal(t, tt.wantDate, result[0].Date)
			assert.Equal(t, tt.wantSrc, result[0].Source)
		})
	}
}

func TestLoadFeedbackExamples(t *testing.T) {
	const confirmedLine = `{"id":"a1","item":"Uber Centro","date":"15/04","value":35.50,"predicted_subcategory":"Uber","predicted_category":"Transporte","actual_subcategory":"Uber","actual_category":"Transporte","confidence":0.92,"status":"confirmed","model":"m","timestamp":"t"}`
	const correctedLine = `{"id":"a2","item":"Rappi","date":"10/03","value":25.00,"predicted_subcategory":"Restaurante","predicted_category":"Alimentação","actual_subcategory":"Delivery","actual_category":"Alimentação","confidence":0.70,"status":"corrected","model":"m","timestamp":"t"}`
	const manualLine = `{"id":"a3","item":"Manual entry","date":"01/01","value":100.00,"predicted_subcategory":"","predicted_category":"","actual_subcategory":"Aluguel","actual_category":"Habitação","confidence":0.0,"status":"manual","model":"","timestamp":"t"}`

	tests := []struct {
		name    string
		content string
		wantNil bool
		wantErr bool
		wantLen int
		check   func(t *testing.T, result []Example)
	}{
		{
			name:    "confirmed and corrected loaded, manual skipped",
			content: confirmedLine + "\n" + correctedLine + "\n" + manualLine + "\n",
			wantLen: 2,
			check: func(t *testing.T, result []Example) {
				assert.Equal(t, SourceConfirmed, result[0].Source)
				assert.Equal(t, "Uber", result[0].Subcategory)
				assert.Equal(t, "Transporte", result[0].Category)
				assert.Equal(t, SourceCorrected, result[1].Source)
				assert.Equal(t, "Delivery", result[1].Subcategory)
				assert.Equal(t, "Alimentação", result[1].Category)
			},
		},
		{
			name:    "missing file returns nil nil",
			wantNil: true,
		},
		{
			name:    "blank lines skipped",
			content: confirmedLine + "\n\n" + correctedLine + "\n\n",
			wantLen: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var path string
			if tt.content != "" {
				dir := t.TempDir()
				path = filepath.Join(dir, "classifications.jsonl")
				writeFile(t, path, tt.content)
			} else {
				path = t.TempDir() + "/classifications.jsonl" // non-existent
			}

			result, err := LoadFeedbackExamples(path)

			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			if tt.wantNil {
				assert.Nil(t, result)
				return
			}
			require.Len(t, result, tt.wantLen)
			if tt.check != nil {
				tt.check(t, result)
			}
		})
	}
}

func TestLoadKeywordIndex(t *testing.T) {
	const validDict = `{
		"lexical_features": {
			"keywords": {
				"uber": {"dominant_subcategory":"Uber","specificity":0.95,"subcategories":["Uber"]},
				"rappi": {"dominant_subcategory":"Delivery","specificity":0.88,"subcategories":["Delivery","Restaurante"]}
			}
		}
	}`

	tests := []struct {
		name    string
		setup   func(dir string)
		wantLen int
		wantErr bool
	}{
		{
			name:    "loads keywords correctly",
			setup:   func(dir string) { writeFile(t, dir+"/feature_dictionary_enhanced.json", validDict) },
			wantLen: 2,
		},
		{
			name:    "missing file returns error",
			setup:   func(dir string) {},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			tt.setup(dir)

			index, err := LoadKeywordIndex(dir)

			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Len(t, index, tt.wantLen)
			entry, ok := index["uber"]
			require.True(t, ok)
			assert.Equal(t, "Uber", entry.DominantSubcategory)
			assert.Equal(t, 0.95, entry.Specificity)
		})
	}
}

func TestMergeExamplePools(t *testing.T) {
	training2 := []Example{
		{Item: "Uber Centro", Source: SourceTraining},
		{Item: "Supermercado", Source: SourceTraining},
	}
	feedback2 := []Example{
		{Item: "Rappi", Source: SourceConfirmed},
		{Item: "Drogaria", Source: SourceCorrected},
	}

	tests := []struct {
		name      string
		training  []Example
		feedback  []Example
		wantNil   bool
		wantLen   int
		checkFunc func(t *testing.T, result []Example)
	}{
		{
			name:    "both nil returns nil",
			wantNil: true,
		},
		{
			name:     "only training",
			training: training2,
			wantLen:  2,
		},
		{
			name:    "only feedback",
			feedback: feedback2,
			wantLen:  2,
		},
		{
			name:     "feedback wins on dedup",
			training: []Example{{Item: "Uber", Source: SourceTraining}},
			feedback: []Example{{Item: "Uber", Source: SourceCorrected}},
			wantLen:  1,
			checkFunc: func(t *testing.T, result []Example) {
				assert.Equal(t, SourceCorrected, result[0].Source)
			},
		},
		{
			name:     "dedup is case-insensitive",
			training: []Example{{Item: "UBER CENTRO", Source: SourceTraining}},
			feedback: []Example{{Item: "uber centro", Source: SourceCorrected}},
			wantLen:  1,
			checkFunc: func(t *testing.T, result []Example) {
				assert.Equal(t, SourceCorrected, result[0].Source)
			},
		},
		{
			name:     "feedback appears first in result",
			training: []Example{{Item: "B", Source: SourceTraining}},
			feedback: []Example{{Item: "A", Source: SourceConfirmed}},
			wantLen:  2,
			checkFunc: func(t *testing.T, result []Example) {
				assert.Equal(t, "A", result[0].Item)
				assert.Equal(t, "B", result[1].Item)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := MergeExamplePools(tt.training, tt.feedback)
			if tt.wantNil {
				assert.Nil(t, result)
				return
			}
			require.Len(t, result, tt.wantLen)
			if tt.checkFunc != nil {
				tt.checkFunc(t, result)
			}
		})
	}
}

// writeFile is a test helper that writes content to path, failing the test on error.
func writeFile(t *testing.T, path, content string) {
	t.Helper()
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
}
