package apply

import (
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
