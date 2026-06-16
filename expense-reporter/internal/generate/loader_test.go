package generate

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadTaxonomy_SkeletonOnly(t *testing.T) {
	taxonomyPath := "../../test/fixtures/generate-basic/taxonomy.json"
	sheets, incomeBlocks, err := LoadTaxonomy(taxonomyPath, "")
	require.NoError(t, err)

	assert.Len(t, sheets, 2)
	assert.Equal(t, "Fixas", sheets[0].Name)
	assert.Equal(t, "Variáveis", sheets[1].Name)

	fixas := sheets[0]
	assert.Len(t, fixas.Cats, 2)
	assert.Equal(t, "Habitação", fixas.Cats[0].Name)
	assert.Len(t, fixas.Cats[0].Subs, 2)
	assert.Equal(t, "Diarista", fixas.Cats[0].Subs[0].Name)
	assert.Equal(t, "Aluguel", fixas.Cats[0].Subs[1].Name)
	assert.Equal(t, "Lazer", fixas.Cats[1].Name)
	assert.Len(t, fixas.Cats[1].Subs, 1)
	assert.Equal(t, "Netflix", fixas.Cats[1].Subs[0].Name)

	variaveis := sheets[1]
	assert.Len(t, variaveis.Cats, 1)
	assert.Equal(t, "Transporte", variaveis.Cats[0].Name)
	assert.Len(t, variaveis.Cats[0].Subs, 1)
	assert.Equal(t, "Metrô", variaveis.Cats[0].Subs[0].Name)

	assert.Len(t, incomeBlocks, 2)
	assert.Equal(t, "Receita", incomeBlocks[0].Category)
	assert.Equal(t, "Salário", incomeBlocks[0].Label)
	assert.Equal(t, "Receita", incomeBlocks[1].Category)
	assert.Equal(t, "13°", incomeBlocks[1].Label)

	for _, sheet := range sheets {
		for _, cat := range sheet.Cats {
			for _, sub := range cat.Subs {
				assert.Zero(t, sub.MaxEntries())
			}
		}
	}

	for _, block := range incomeBlocks {
		assert.Zero(t, block.MaxEntries())
	}
}

func TestLoadTaxonomy_WithEntries(t *testing.T) {
	taxonomyPath := "../../test/fixtures/generate-basic/taxonomy.json"
	entriesPath := "../../test/fixtures/generate-basic/entries.jsonl"

	sheets, incomeBlocks, err := LoadTaxonomy(taxonomyPath, entriesPath)
	require.NoError(t, err)

	// Check Aluguel in Fixas.Habitação
	fixas := sheets[0]
	aluguelSub := fixas.Cats[0].Subs[1] // Aluguel is second subcategory in Habitação

	assert.Len(t, aluguelSub.Months[0], 2) // Jan has 2 entries
	assert.Equal(t, "Aluguel", aluguelSub.Months[0][0].Item)
	assert.Equal(t, 10, aluguelSub.Months[0][0].Day)
	assert.Equal(t, 2200.0, aluguelSub.Months[0][0].Value)
	assert.Equal(t, "Aluguel ajuste", aluguelSub.Months[0][1].Item)
	assert.Equal(t, 25, aluguelSub.Months[0][1].Day)
	assert.Equal(t, 150.0, aluguelSub.Months[0][1].Value)

	assert.Len(t, aluguelSub.Months[1], 1) // Feb has 1 entry
	assert.Equal(t, "Aluguel", aluguelSub.Months[1][0].Item)
	assert.Equal(t, 10, aluguelSub.Months[1][0].Day)
	assert.Equal(t, 2200.0, aluguelSub.Months[1][0].Value)

	assert.Equal(t, 2, aluguelSub.MaxEntries())

	// Check Netflix in Fixas.Lazer
	netflixSub := fixas.Cats[1].Subs[0]
	assert.Len(t, netflixSub.Months[0], 1)
	assert.Equal(t, "Netflix", netflixSub.Months[0][0].Item)
	assert.Equal(t, 15, netflixSub.Months[0][0].Day)
	assert.Equal(t, 55.9, netflixSub.Months[0][0].Value)

	// Check Metrô in Variáveis.Transporte
	variaveis := sheets[1]
	metroSub := variaveis.Cats[0].Subs[0]
	assert.Len(t, metroSub.Months[0], 1)
	assert.Equal(t, "Metrô recarga", metroSub.Months[0][0].Item)
	assert.Equal(t, 3, metroSub.Months[0][0].Day)
	assert.Equal(t, 50.0, metroSub.Months[0][0].Value)

	assert.Len(t, metroSub.Months[1], 1)
	assert.Equal(t, "Metrô recarga", metroSub.Months[1][0].Item)
	assert.Equal(t, 4, metroSub.Months[1][0].Day)
	assert.Equal(t, 50.0, metroSub.Months[1][0].Value)

	// Check income blocks
	salarioBlock := incomeBlocks[0]
	assert.Len(t, salarioBlock.Months[0], 1)
	assert.Equal(t, "Salário", salarioBlock.Months[0][0].Item)
	assert.Equal(t, 5, salarioBlock.Months[0][0].Day)
	assert.Equal(t, 5000.0, salarioBlock.Months[0][0].Value)

	assert.Len(t, salarioBlock.Months[1], 1)
	assert.Equal(t, "Salário", salarioBlock.Months[1][0].Item)
	assert.Equal(t, 5, salarioBlock.Months[1][0].Day)
	assert.Equal(t, 5000.0, salarioBlock.Months[1][0].Value)

	// Check 13° block
	degreeBlock := incomeBlocks[1]
	assert.Len(t, degreeBlock.Months[0], 1)
	assert.Equal(t, "13° primeira parcela", degreeBlock.Months[0][0].Item)
	assert.Equal(t, 20, degreeBlock.Months[0][0].Day)
	assert.Equal(t, 2500.0, degreeBlock.Months[0][0].Value)
}

func TestLoadTaxonomy_UnmappedSubcategory(t *testing.T) {
	taxonomyPath := "../../test/fixtures/generate-basic/taxonomy.json"
	entriesPath := "../../test/fixtures/generate-basic/entries-with-unmapped.jsonl"

	sheets, incomeBlocks, err := LoadTaxonomy(taxonomyPath, entriesPath)
	require.NoError(t, err)

	// Diarista should be loaded
	fixas := sheets[0]
	diaristaSub := fixas.Cats[0].Subs[0] // Diarista is first subcategory in Habitação

	assert.Len(t, diaristaSub.Months[0], 1)
	assert.Equal(t, "Diarista Ana", diaristaSub.Months[0][0].Item)

	// Entry with "Esportes" should be skipped (no error)
	for _, sheet := range sheets {
		found := false
		for _, cat := range sheet.Cats {
			for _, sub := range cat.Subs {
				if sub.Name == "Esportes" {
					found = true
					break
				}
			}
			if found {
				break
			}
		}
		assert.False(t, found, "Esportes should not be in any sheet")
	}

	for _, block := range incomeBlocks {
		assert.NotEqual(t, "Esportes", block.Label)
	}
}

func TestLoadTaxonomy_DuplicateSubcategory(t *testing.T) {
	tempDir := t.TempDir()
	taxonomyPath := filepath.Join(tempDir, "taxonomy.json")

	// Create a taxonomy with duplicate subcategory
	content := `{
    "sheets": [
        {
            "name": "Sheet1",
            "categories": [
                {
                    "name": "Category1",
                    "subcategories": ["Diarista"]
                }
            ]
        },
        {
            "name": "Sheet2",
            "categories": [
                {
                    "name": "Category2",
                    "subcategories": ["Diarista"]
                }
            ]
        }
    ],
    "incomeCategories": []
}`

	err := os.WriteFile(taxonomyPath, []byte(content), 0644)
	require.NoError(t, err)

	_, _, err = LoadTaxonomy(taxonomyPath, "")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Diarista")
}

func TestParseDate_Malformed(t *testing.T) {
	cases := []struct {
		input    string
		dayErr   bool
		monthErr bool
	}{
		{"5/13", false, true},     // Invalid month
		{"32/01", true, false},     // Invalid day
		{"x/y", true, true},       // Non-numeric values
		{"05/01", false, false},   // Valid date
		{"2026-01-05", true, true}, // Wrong format
	}

	for _, tc := range cases {
		day, month, err := parseDate(tc.input)
		if tc.dayErr || tc.monthErr {
			assert.Error(t, err)
		} else {
			assert.NoError(t, err)
			assert.Equal(t, 5, day)
			assert.Equal(t, 1, month)
		}
	}
}
