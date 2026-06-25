package taxonomy

import (
	"bufio"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestIncomeNestedParseFlatSkeleton verifies that the flat legacy format
// ("blocks":["Salário","13°"]) still parses correctly: each string becomes
// a RevenueBlock with Block==Label and Category from the parent name.
// This guards backward-compat with test/fixtures/generate-basic/taxonomy.json.
func TestIncomeNestedParseFlatSkeleton(t *testing.T) {
	sheets, incomeBlocks, err := LoadTaxonomy("../../test/fixtures/generate-basic/taxonomy.json", "", "", 0)
	require.NoError(t, err)
	_ = sheets

	require.Len(t, incomeBlocks, 2)
	assert.Equal(t, "Receita", incomeBlocks[0].Category)
	assert.Equal(t, "Salário", incomeBlocks[0].Block)
	assert.Equal(t, "Salário", incomeBlocks[0].Label)
	assert.Equal(t, "Receita", incomeBlocks[1].Category)
	assert.Equal(t, "13°", incomeBlocks[1].Block)
	assert.Equal(t, "13°", incomeBlocks[1].Label)
}

// TestIncomeNestedParseNestedSkeleton verifies that the new nested format
// ({block, sublines:[]}) parses to one RevenueBlock per (block, subline) triple.
// Uses test/fixtures/generate-income/taxonomy.json (the WS-C fixture).
func TestIncomeNestedParseNestedSkeleton(t *testing.T) {
	sheets, incomeBlocks, err := LoadTaxonomy("../../test/fixtures/generate-income/taxonomy.json", "", "", 0)
	require.NoError(t, err)
	_ = sheets

	// Expected leaves: Salário×3 + Férias×2 = 5 RevenueBlocks.
	require.Len(t, incomeBlocks, 5)

	// First block group: Salário
	assert.Equal(t, "Receitas", incomeBlocks[0].Category)
	assert.Equal(t, "Salário", incomeBlocks[0].Block)
	assert.Equal(t, "Salário", incomeBlocks[0].Label)

	assert.Equal(t, "Receitas", incomeBlocks[1].Category)
	assert.Equal(t, "Salário", incomeBlocks[1].Block)
	assert.Equal(t, "INSS", incomeBlocks[1].Label)

	assert.Equal(t, "Receitas", incomeBlocks[2].Category)
	assert.Equal(t, "Salário", incomeBlocks[2].Block)
	assert.Equal(t, "IRRF", incomeBlocks[2].Label)

	// Second block group: Férias
	assert.Equal(t, "Receitas", incomeBlocks[3].Category)
	assert.Equal(t, "Férias", incomeBlocks[3].Block)
	assert.Equal(t, "Férias Normais", incomeBlocks[3].Label)

	assert.Equal(t, "Receitas", incomeBlocks[4].Category)
	assert.Equal(t, "Férias", incomeBlocks[4].Block)
	assert.Equal(t, "IRRF Férias", incomeBlocks[4].Label)
}

// TestIncomeEntryRoutesCorrectLeafAndMonth verifies that scanIncomeEntries
// routes a line to the correct RevenueBlock leaf and the correct month slice.
// Uses the generate-income fixture taxonomy + income-entries JSONL.
func TestIncomeEntryRoutesCorrectLeafAndMonth(t *testing.T) {
	_, incomeBlocks, err := LoadTaxonomy(
		"../../test/fixtures/generate-income/taxonomy.json",
		"",
		"../../test/fixtures/generate-income/income-entries.jsonl",
		2026,
	)
	require.NoError(t, err)

	// Find each leaf by Block+Label and check its months.
	blk := findLeaf(t, incomeBlocks, "Salário", "Salário")
	assert.Len(t, blk.Months[0], 1, "Salário/Salário: 1 entry in Jan")
	assert.Equal(t, 5000.0, blk.Months[0][0].Value)
	assert.Len(t, blk.Months[1], 1, "Salário/Salário: 1 entry in Feb")

	inss := findLeaf(t, incomeBlocks, "Salário", "INSS")
	assert.Len(t, inss.Months[0], 1, "Salário/INSS: 1 entry in Jan")
	assert.Equal(t, -550.0, inss.Months[0][0].Value, "deductions must be negative")
	assert.Len(t, inss.Months[1], 1, "Salário/INSS: 1 entry in Feb")

	irrf := findLeaf(t, incomeBlocks, "Salário", "IRRF")
	assert.Len(t, irrf.Months[0], 1, "Salário/IRRF: 1 entry in Jan")
	assert.Equal(t, -300.0, irrf.Months[0][0].Value)
	assert.Nil(t, irrf.Months[1], "Salário/IRRF: no Feb entry")

	ferias := findLeaf(t, incomeBlocks, "Férias", "Férias Normais")
	assert.Len(t, ferias.Months[6], 1, "Férias Normais: 1 entry in Jul (index 6)")
	assert.Equal(t, 8000.0, ferias.Months[6][0].Value)

	irrfFerias := findLeaf(t, incomeBlocks, "Férias", "IRRF Férias")
	assert.Len(t, irrfFerias.Months[6], 1, "IRRF Férias: 1 entry in Jul")
	assert.Equal(t, -400.0, irrfFerias.Months[6][0].Value)
}

// TestIncomeSignedValuesKept guards that negative deduction values are stored
// as-is (no sign flip, no abs). The generator just SUMs — sign logic stays in data.
func TestIncomeSignedValuesKept(t *testing.T) {
	const taxonomyJSON = `{"types":[],"incomeCategories":[{"name":"Receitas","blocks":[{"block":"Sal","sublines":["Sal","Desc"]}]}]}`
	const entriesJSONL = `{"date":"01/01/2026","value":1000.0,"income_category":"Sal","income_label":"Sal","item_note":"Base"}
{"date":"01/01/2026","value":-200.0,"income_category":"Sal","income_label":"Desc","item_note":"Deducao"}`

	blocks := parseNestedBlocks(t, taxonomyJSON)
	idx := buildIncomeIndex(buildByPath(t, blocks))

	err := scanIncomeEntries(bufio.NewScanner(strings.NewReader(entriesJSONL)), idx, 2026)
	require.NoError(t, err)

	sal := findLeaf(t, blocks, "Sal", "Sal")
	require.Len(t, sal.Months[0], 1)
	assert.Equal(t, 1000.0, sal.Months[0][0].Value)

	desc := findLeaf(t, blocks, "Sal", "Desc")
	require.Len(t, desc.Months[0], 1)
	assert.Equal(t, -200.0, desc.Months[0][0].Value, "deduction must remain negative")
}

// TestIncomeWrongLabelWarnsAndSkips mirrors TestLoadTaxonomy_TypedEntryWrongPathSkipped:
// an entry with an income_label not in the taxonomy is warned and skipped, not crashed.
func TestIncomeWrongLabelWarnsAndSkips(t *testing.T) {
	const taxonomyJSON = `{"types":[],"incomeCategories":[{"name":"Receitas","blocks":[{"block":"Sal","sublines":["Sal"]}]}]}`
	const entriesJSONL = `{"date":"01/01/2026","value":1000.0,"income_category":"Sal","income_label":"DoesNotExist","item_note":"Bad"}`

	blocks := parseNestedBlocks(t, taxonomyJSON)
	idx := buildIncomeIndex(buildByPath(t, blocks))

	err := scanIncomeEntries(bufio.NewScanner(strings.NewReader(entriesJSONL)), idx, 2026)
	require.NoError(t, err, "wrong income_label must warn+skip, not error")

	sal := findLeaf(t, blocks, "Sal", "Sal")
	assert.Nil(t, sal.Months[0], "no entry should land when label is wrong")
}

// TestIncomeYearFilterSkipsOtherYear ensures that entries with a year != targetYear
// are filtered out (same contract as expense entry year filtering).
func TestIncomeYearFilterSkipsOtherYear(t *testing.T) {
	const taxonomyJSON = `{"types":[],"incomeCategories":[{"name":"Receitas","blocks":[{"block":"Sal","sublines":["Sal"]}]}]}`
	const entriesJSONL = `{"date":"01/01/2025","value":9999.0,"income_category":"Sal","income_label":"Sal","item_note":"OldYear"}`

	blocks := parseNestedBlocks(t, taxonomyJSON)
	idx := buildIncomeIndex(buildByPath(t, blocks))

	err := scanIncomeEntries(bufio.NewScanner(strings.NewReader(entriesJSONL)), idx, 2026)
	require.NoError(t, err)

	sal := findLeaf(t, blocks, "Sal", "Sal")
	assert.Nil(t, sal.Months[0], "entry from wrong year must be filtered")
}

// --- helpers ---

// findLeaf returns the RevenueBlock with matching Block and Label, fails the test if absent.
func findLeaf(t *testing.T, blocks []RevenueBlock, block, label string) *RevenueBlock {
	t.Helper()
	for i := range blocks {
		if blocks[i].Block == block && blocks[i].Label == label {
			return &blocks[i]
		}
	}
	t.Fatalf("leaf not found: block=%q label=%q", block, label)
	return nil
}

// parseNestedBlocks loads a taxonomy JSON string and returns its income blocks.
func parseNestedBlocks(t *testing.T, taxonomyJSON string) []RevenueBlock {
	t.Helper()
	tmp := t.TempDir()
	path := tmp + "/taxonomy.json"
	require.NoError(t, writeFile(path, taxonomyJSON))
	_, blocks, err := LoadTaxonomy(path, "", "", 0)
	require.NoError(t, err)
	return blocks
}

// buildByPath reconstructs the byPath routing map from a []RevenueBlock.
func buildByPath(t *testing.T, blocks []RevenueBlock) map[string]subcatTarget {
	t.Helper()
	byPath := make(map[string]subcatTarget)
	for i := range blocks {
		b := &blocks[i]
		key := incomePath(b.Category, b.Block, b.Label)
		byPath[key] = subcatTarget{kind: "income", income: b}
	}
	return byPath
}

// writeFile writes content to path (test helper).
func writeFile(path, content string) error {
	return os.WriteFile(path, []byte(content), 0600)
}
