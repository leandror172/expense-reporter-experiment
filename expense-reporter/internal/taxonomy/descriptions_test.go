package taxonomy

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/text/unicode/norm"
)

func TestLoadTypeDescriptions_ValidFile_RoundTripsEntries(t *testing.T) {
	t.Run("Valid file round trips entries", func(t *testing.T) {
		tempDir := t.TempDir()
		filePath := filepath.Join(tempDir, "types.json")

		// Write valid JSON content to the temp file
		validContent := []byte(`{
			"Fixas": "Recurring fixed monthly commitments.",
			"Variáveis": "Recurring essential spending that varies.",
			"Extras": "Irregular necessary expenses.",
			"Adicionais": "Discretionary optional spending."
		}`)

		if err := os.WriteFile(filePath, validContent, 0o644); err != nil {
			t.Fatalf("Failed to write test file: %v", err)
		}

		// Call LoadTypeDescriptions with the path
		descMap, err := LoadTypeDescriptions(filePath)
		require.NoError(t, err)

		// Assert that the returned map has exactly 4 entries
		assert.Len(t, descMap, 4)
		assert.Equal(t, "Recurring fixed monthly commitments.", descMap["Fixas"])
		assert.Equal(t, "Recurring essential spending that varies.", descMap["Variáveis"])
		assert.Equal(t, "Irregular necessary expenses.", descMap["Extras"])
		assert.Equal(t, "Discretionary optional spending.", descMap["Adicionais"])
	})
}

func TestLoadTypeDescriptions_NonexistentPath_ReturnsEmptyMapNoError(t *testing.T) {
	t.Run("Nonexistent path returns empty map and no error", func(t *testing.T) {
		tempDir := t.TempDir()
		filePath := filepath.Join(tempDir, "does-not-exist.json")

		descMap, err := LoadTypeDescriptions(filePath)
		require.NoError(t, err)

		assert.NotNil(t, descMap)
		assert.Empty(t, descMap)
	})
}

func TestLoadTypeDescriptions_EmptyPath_ReturnsEmptyMapNoError(t *testing.T) {
	t.Run("Empty path returns empty map and no error", func(t *testing.T) {
		descMap, err := LoadTypeDescriptions("")
		require.NoError(t, err)

		assert.NotNil(t, descMap)
		assert.Empty(t, descMap)
	})
}

func TestLoadTypeDescriptions_MalformedJSON_ReturnsError(t *testing.T) {
	t.Run("Malformed JSON returns error", func(t *testing.T) {
		tempDir := t.TempDir()
		filePath := filepath.Join(tempDir, "malformed.json")

		// Write malformed JSON content to the temp file
		malformedContent := []byte("{not valid json")

		if err := os.WriteFile(filePath, malformedContent, 0o644); err != nil {
			t.Fatalf("Failed to write test file: %v", err)
		}

		// Call LoadTypeDescriptions with the path
		descMap, err := LoadTypeDescriptions(filePath)
		require.Error(t, err)

		assert.Nil(t, descMap)
	})
}

func TestApplyDescriptions_SetsDescriptionByName(t *testing.T) {
	t.Run("Sets description by name", func(t *testing.T) {
		// Create a slice of ExpenseTypes
		types := []ExpenseType{
			{Name: "Fixas"},
			{Name: "Extras"},
		}

		// Define the map to apply descriptions from
		descMap := map[string]string{"Fixas": "desc A"}

		// Apply descriptions
		ApplyDescriptions(types, descMap)

		// Assert that only the first type has a description
		assert.Equal(t, "desc A", types[0].Description)
		assert.Empty(t, types[1].Description)
	})
}

func TestApplyDescriptions_MatchesAcrossNFDvsNFCAccentedName(t *testing.T) {
	t.Run("Matches across NFD vs NFC accented name", func(t *testing.T) {
		// Create a slice of ExpenseTypes
		types := []ExpenseType{
			{Name: norm.NFC.String("Variáveis")}, // NFC form (precomposed)
		}

		// Define the map to apply descriptions from with NFD key
		descMap := map[string]string{
			norm.NFD.String("Variáveis"): "desc B", // NFD form (decomposed)
		}

		// Apply descriptions
		ApplyDescriptions(types, descMap)

		// Assert that the description is set correctly
		assert.Equal(t, "desc B", types[0].Description)
	})

	t.Run("Matches across NFC vs NFD accented name", func(t *testing.T) {
		// Create a slice of ExpenseTypes
		types := []ExpenseType{
			{Name: norm.NFD.String("Variáveis")}, // NFD form (decomposed)
		}

		// Define the map to apply descriptions from with NFC key
		descMap := map[string]string{
			norm.NFC.String("Variáveis"): "desc C", // NFC form (precomposed)
		}

		// Apply descriptions
		ApplyDescriptions(types, descMap)

		// Assert that the description is set correctly
		assert.Equal(t, "desc C", types[0].Description)
	})
}
