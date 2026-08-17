package apply

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReadReviewed(t *testing.T) {
	t.Run("valid file with all four actions", func(t *testing.T) {
		tempDir := t.TempDir()
		filePath := filepath.Join(tempDir, "reviewed.json")

		content := `{
  "reviewedAt": "2026-05-29T10:00:00Z",
  "source": "classified.csv",
  "entries": [
    {
      "id": "f0c3bf1293f3",
      "item": "Uber Centro",
      "date": "15/04",
      "value": 35.50,
      "installments": 1,
      "confidence": 0.91,
      "predicted": {"category": "Transporte", "subcategory": "Uber/Taxi"},
      "action": "confirmed",
      "reviewed": {"sheet": "Variáveis", "category": "Transporte", "subcategory": "Uber/Taxi"}
    },
    {
      "id": "733224d39e01",
      "item": "Diarista Letícia",
      "date": "10/05",
      "value": 160.00,
      "installments": 1,
      "confidence": 0.87,
      "predicted": {"category": "Habitação", "subcategory": "Diarista"},
      "action": "corrected",
      "reviewed": {"sheet": "Fixas", "category": "Habitação", "subcategory": "Aluguel"}
    },
    {
      "id": "24c75fff9223",
      "item": "Academia Smart Fit",
      "date": "05/05",
      "value": 99.90,
      "installments": 1,
      "confidence": 0.62,
      "predicted": {"category": "Saúde", "subcategory": "Academia"},
      "action": "pending",
      "reviewed": null
    }
  ]
}`

		err := os.WriteFile(filePath, []byte(content), 0644)
		require.NoError(t, err)

		result, err := ReadReviewed(filePath)
		require.NoError(t, err)

		assert.Len(t, result.Entries, 3)

		assert.Equal(t, ActionConfirmed, result.Entries[0].Action)
		assert.Equal(t, "Uber Centro", result.Entries[0].Item)

		assert.Equal(t, ActionCorrected, result.Entries[1].Action)
		assert.NotNil(t, result.Entries[1].Reviewed)
		assert.Equal(t, "Aluguel", result.Entries[1].Reviewed.Subcategory)

		assert.Equal(t, ActionPending, result.Entries[2].Action)
		assert.Nil(t, result.Entries[2].Reviewed)
	})

	t.Run("unknown action returns error", func(t *testing.T) {
		tempDir := t.TempDir()
		filePath := filepath.Join(tempDir, "reviewed.json")

		content := `{
  "reviewedAt": "2026-05-29T10:00:00Z",
  "source": "classified.csv",
  "entries": [
    {
      "id": "f0c3bf1293f3",
      "item": "Uber Centro",
      "date": "15/04",
      "value": 35.50,
      "confidence": 0.91,
      "predicted": {"category": "Transporte", "subcategory": "Uber/Taxi"},
      "action": "invalid-action",
      "reviewed": null
    }
  ]
}`

		err := os.WriteFile(filePath, []byte(content), 0644)
		require.NoError(t, err)

		_, err = ReadReviewed(filePath)
		assert.ErrorContains(t, err, "unknown action")
	})

	t.Run("invalid JSON returns error", func(t *testing.T) {
		tempDir := t.TempDir()
		filePath := filepath.Join(tempDir, "reviewed.json")

		content := "{not valid json"
		err := os.WriteFile(filePath, []byte(content), 0644)
		require.NoError(t, err)

		_, err = ReadReviewed(filePath)
		assert.Error(t, err)
	})

	t.Run("file not found returns error", func(t *testing.T) {
		_, err := ReadReviewed("/path/that/does/not/exist/reviewed.json")
		assert.Error(t, err)
	})
}

func TestReviewedLocation_TypeFieldUnmarshal(t *testing.T) {
	tests := []struct {
		name       string
		jsonInput  string
		wantType   string
		wantCat    string
		wantSubcat string
	}{
		{
			"new type key",
			`{"type":"Fixas","category":"Habitação","subcategory":"Aluguel"}`,
			"Fixas",
			"Habitação",
			"Aluguel",
		},
		{
			"empty type",
			`{"category":"Habitação","subcategory":"Aluguel"}`,
			"",
			"Habitação",
			"Aluguel",
		},
		{
			"Variáveis type",
			`{"type":"Variáveis","category":"Despesas","subcategory":"Internet"}`,
			"Variáveis",
			"Despesas",
			"Internet",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var loc ReviewedLocation
			err := json.Unmarshal([]byte(tt.jsonInput), &loc)
			require.NoError(t, err)
			assert.Equal(t, tt.wantType, loc.Type)
			assert.Equal(t, tt.wantCat, loc.Category)
			assert.Equal(t, tt.wantSubcat, loc.Subcategory)
		})
	}
}

func TestReviewedLocation_LegacySheetBackwardCompat(t *testing.T) {
	tests := []struct {
		name      string
		jsonInput string
		wantType  string
	}{
		{
			"legacy sheet key only",
			`{"sheet":"Fixas","category":"Habitação","subcategory":"Aluguel"}`,
			"Fixas",
		},
		{
			"legacy sheet with empty type",
			`{"type":"","sheet":"Variáveis","category":"Despesas","subcategory":"Internet"}`,
			"Variáveis",
		},
		{
			"Extras via legacy sheet",
			`{"sheet":"Extras","category":"Lazer","subcategory":"Cinema"}`,
			"Extras",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var loc ReviewedLocation
			err := json.Unmarshal([]byte(tt.jsonInput), &loc)
			require.NoError(t, err)
			assert.Equal(t, tt.wantType, loc.Type, "Type should load from legacy sheet key")
		})
	}
}

func TestReviewedLocation_TypeWinsOverSheet(t *testing.T) {
	t.Run("type key takes precedence when both present and non-empty", func(t *testing.T) {
		jsonInput := `{"type":"Fixas","sheet":"Variáveis","category":"Habitação","subcategory":"Aluguel"}`
		var loc ReviewedLocation
		err := json.Unmarshal([]byte(jsonInput), &loc)
		require.NoError(t, err)
		assert.Equal(t, "Fixas", loc.Type)
	})

	t.Run("sheet fallback when type is empty", func(t *testing.T) {
		jsonInput := `{"type":"","sheet":"Variáveis","category":"Habitação","subcategory":"Aluguel"}`
		var loc ReviewedLocation
		err := json.Unmarshal([]byte(jsonInput), &loc)
		require.NoError(t, err)
		assert.Equal(t, "Variáveis", loc.Type)
	})
}

// TestReadReviewed_RejectsEntryWithoutInstallmentCount pins the installments field as
// REQUIRED rather than optional (T-21).
//
// The temptation is to treat an absent key as "1 installment", since that is what almost
// every row is. That would make the contract unfalsifiable: a missing key decodes to Go's
// zero value 0, so the reader could never distinguish "the producer did not send it" from
// "the producer sent zero", and a producer that silently stopped emitting the field would
// keep working until someone noticed the workbook was short. Requiring it means one
// producer, one spelling — the same rule T-54 established for auto_inserted after a
// reader spent months accepting a spelling nothing emitted.
func TestReadReviewed_RejectsEntryWithoutInstallmentCount(t *testing.T) {
	rejected := []struct {
		name   string
		clause string
	}{
		{"key absent entirely", ""},
		{"explicitly zero", `,"installments": 0`},
		{"explicitly negative", `,"installments": -1`},
	}

	for _, tc := range rejected {
		t.Run(tc.name, func(t *testing.T) {
			_, err := ReadReviewed(reviewedFileWithInstallments(t, tc.clause))

			require.Error(t, err)
			assert.Contains(t, err.Error(), "installments")
			assert.Contains(t, err.Error(), "entry 0")
		})
	}

	accepted := []struct {
		name string
		want int
	}{
		{"a one-off purchase is 1", 1},
		{"a three-installment purchase is 3", 3},
	}

	for _, tc := range accepted {
		t.Run(tc.name, func(t *testing.T) {
			clause := fmt.Sprintf(`,"installments": %d`, tc.want)

			result, err := ReadReviewed(reviewedFileWithInstallments(t, clause))

			require.NoError(t, err)
			assert.Equal(t, tc.want, result.Entries[0].Installments)
		})
	}
}

// reviewedFileWithInstallments writes a one-entry reviewed.json whose installments clause
// is supplied verbatim, so "the key is absent" is expressible as an empty string — the
// case a typed int parameter could not reach.
func reviewedFileWithInstallments(t *testing.T, installmentsClause string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "reviewed.json")
	content := fmt.Sprintf(`{
  "reviewedAt": "2026-05-29T10:00:00Z",
  "source": "classified.csv",
  "entries": [
    {
      "id": "f0c3bf1293f3",
      "item": "Uber Centro",
      "date": "15/04",
      "value": 35.50,
      "confidence": 0.91,
      "predicted": {"category": "Transporte", "subcategory": "Uber/Taxi"},
      "action": "confirmed",
      "reviewed": {"type": "Variáveis", "category": "Transporte", "subcategory": "Uber/Taxi"}%s
    }
  ]
}`, installmentsClause)
	require.NoError(t, os.WriteFile(path, []byte(content), 0644))

	return path
}
