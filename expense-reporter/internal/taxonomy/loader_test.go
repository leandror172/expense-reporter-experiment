package taxonomy

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/text/unicode/norm"
)

func TestLoadTaxonomy_SkeletonOnly(t *testing.T) {
	taxonomyPath := "../../test/fixtures/generate-basic/taxonomy.json"
	sheets, incomeBlocks, err := LoadTaxonomy(taxonomyPath, "", "", 0)
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

	sheets, incomeBlocks, err := LoadTaxonomy(taxonomyPath, entriesPath, "", 0)
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

	sheets, incomeBlocks, err := LoadTaxonomy(taxonomyPath, entriesPath, "", 0)
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

// writeTempFile writes content to a temp file and returns its path.
func writeTempFile(t *testing.T, name, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	require.NoError(t, os.WriteFile(path, []byte(content), 0644))
	return path
}

// TestLoadTaxonomy_SamePathDuplicate is the surviving half of the old duplicate
// guard. Behavior change (full-path identity, task #5 routing decision): a
// subcategory's identity is now its full sheet/category/subcategory path, so only
// an EXACT repeat of that path is a validation error. Here "Diarista" is listed
// twice in the same sheet+category -> still an error.
func TestLoadTaxonomy_SamePathDuplicate(t *testing.T) {
	taxonomyPath := writeTempFile(t, "taxonomy.json", `{
    "types": [
        { "name": "Sheet1", "categories": [
            { "name": "Category1", "subcategories": ["Diarista", "Diarista"] } ] }
    ],
    "incomeCategories": []
}`)

	_, _, err := LoadTaxonomy(taxonomyPath, "", "", 0)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Diarista")
}

// TestLoadTaxonomy_CrossPathDuplicateAllowed documents the relaxed invariant.
// Previous behavior: the same bare name in two different sheets/categories was a
// hard error. New behavior: cross-path repeats are legal — the real taxonomy
// legitimately repeats leaf names (Orion across Pet blocks; Aluguel as both a
// Fixas expense and a Receitas income block). Reason: identity is the full path.
func TestLoadTaxonomy_CrossPathDuplicateAllowed(t *testing.T) {
	taxonomyPath := writeTempFile(t, "taxonomy.json", `{
    "types": [
        { "name": "Sheet1", "categories": [
            { "name": "Category1", "subcategories": ["Diarista"] } ] },
        { "name": "Sheet2", "categories": [
            { "name": "Category2", "subcategories": ["Diarista"] } ] }
    ],
    "incomeCategories": []
}`)

	sheets, _, err := LoadTaxonomy(taxonomyPath, "", "", 0)
	require.NoError(t, err)
	assert.Equal(t, "Diarista", sheets[0].Cats[0].Subs[0].Name)
	assert.Equal(t, "Diarista", sheets[1].Cats[0].Subs[0].Name)
}

// TestLoadTaxonomy_AmbiguousEntrySkipped is the real coverage for the
// ambiguous-routing safety: a bare name that maps to 3+ full paths (Orion in
// Pet/Pets across three sheets) must NOT route an entry to any of them while the
// full-path routing redesign (task #5) is deferred. The entry is skipped (warned
// to stderr, exit 0), never silently misrouted. This guards the 3x re-add trap:
// a naive delete-on-collision would re-add Orion on the third occurrence.
func TestLoadTaxonomy_AmbiguousEntrySkipped(t *testing.T) {
	dir := t.TempDir()
	taxonomyPath := filepath.Join(dir, "taxonomy.json")
	require.NoError(t, os.WriteFile(taxonomyPath, []byte(`{
    "types": [
        { "name": "Fixas", "categories": [
            { "name": "Pet", "subcategories": ["Orion"] } ] },
        { "name": "Variáveis", "categories": [
            { "name": "Pets", "subcategories": ["Orion"] } ] },
        { "name": "Extras", "categories": [
            { "name": "Pets", "subcategories": ["Orion"] } ] }
    ],
    "incomeCategories": []
}`), 0644))
	entriesPath := filepath.Join(dir, "entries.jsonl")
	require.NoError(t, os.WriteFile(entriesPath,
		[]byte(`{"item":"Ração","date":"05/01","value":120.0,"subcategory":"Orion"}`+"\n"), 0644))

	sheets, _, err := LoadTaxonomy(taxonomyPath, entriesPath, "", 0)
	require.NoError(t, err)

	for _, sheet := range sheets {
		for _, cat := range sheet.Cats {
			for _, sub := range cat.Subs {
				assert.Zero(t, sub.MaxEntries(),
					"ambiguous entry must not route into %s/%s/%s", sheet.Name, cat.Name, sub.Name)
			}
		}
	}
}

// orionTaxonomy is the three-sheet taxonomy where the leaf "Orion" is ambiguous by
// bare name (Fixas/Pet, Variáveis/Pets, Extras/Pets) — reused by the typed-routing tests.
const orionTaxonomy = `{
    "types": [
        { "name": "Fixas", "categories": [
            { "name": "Pet", "subcategories": ["Orion"] } ] },
        { "name": "Variáveis", "categories": [
            { "name": "Pets", "subcategories": ["Orion"] } ] },
        { "name": "Extras", "categories": [
            { "name": "Pets", "subcategories": ["Orion"] } ] }
    ],
    "incomeCategories": []
}`

// TestLoadTaxonomy_AmbiguousEntryRoutedByFullPath is the new capability (Plan B): when
// entries carry a type, the ambiguous leaf "Orion" routes to EXACTLY the block named by
// its full path — type/category/subcategory — and to none of the others. This is also
// the discriminating test for the full-path string-equality assumption: the entry's
// type+category must byte-match the taxonomy spelling, or it would route nowhere.
func TestLoadTaxonomy_AmbiguousEntryRoutedByFullPath(t *testing.T) {
	dir := t.TempDir()
	taxonomyPath := filepath.Join(dir, "taxonomy.json")
	require.NoError(t, os.WriteFile(taxonomyPath, []byte(orionTaxonomy), 0644))

	entriesPath := filepath.Join(dir, "entries.jsonl")
	require.NoError(t, os.WriteFile(entriesPath, []byte(
		`{"item":"Ração Fixas","date":"05/01","value":120.0,"type":"Fixas","category":"Pet","subcategory":"Orion"}`+"\n"+
			`{"item":"Ração Var","date":"06/02","value":121.0,"type":"Variáveis","category":"Pets","subcategory":"Orion"}`+"\n"+
			`{"item":"Ração Extra","date":"07/03","value":122.0,"type":"Extras","category":"Pets","subcategory":"Orion"}`+"\n"),
		0644))

	sheets, _, err := LoadTaxonomy(taxonomyPath, entriesPath, "", 0)
	require.NoError(t, err)

	// Each Orion block holds exactly its own entry, in the entry's month.
	fixasOrion := sheets[0].Cats[0].Subs[0]
	require.Len(t, fixasOrion.Months[0], 1) // Jan
	assert.Equal(t, "Ração Fixas", fixasOrion.Months[0][0].Item)

	varOrion := sheets[1].Cats[0].Subs[0]
	require.Len(t, varOrion.Months[1], 1) // Feb
	assert.Equal(t, "Ração Var", varOrion.Months[1][0].Item)

	extraOrion := sheets[2].Cats[0].Subs[0]
	require.Len(t, extraOrion.Months[2], 1) // Mar
	assert.Equal(t, "Ração Extra", extraOrion.Months[2][0].Item)

	// No cross-contamination: each block has exactly one entry total.
	assert.Equal(t, 1, fixasOrion.MaxEntries())
	assert.Equal(t, 1, varOrion.MaxEntries())
	assert.Equal(t, 1, extraOrion.MaxEntries())
}

// TestLoadTaxonomy_TypedEntryWrongPathSkipped guards the string-equality contract from
// the other side: a typed entry whose category does NOT match the taxonomy spelling
// fails to route (warn+skip, exit 0) rather than silently landing in the wrong block.
func TestLoadTaxonomy_TypedEntryWrongPathSkipped(t *testing.T) {
	dir := t.TempDir()
	taxonomyPath := filepath.Join(dir, "taxonomy.json")
	require.NoError(t, os.WriteFile(taxonomyPath, []byte(orionTaxonomy), 0644))

	entriesPath := filepath.Join(dir, "entries.jsonl")
	// category "Petz" (wrong spelling) under a valid type — must not route anywhere.
	require.NoError(t, os.WriteFile(entriesPath, []byte(
		`{"item":"Ração","date":"05/01","value":120.0,"type":"Fixas","category":"Petz","subcategory":"Orion"}`+"\n"),
		0644))

	sheets, _, err := LoadTaxonomy(taxonomyPath, entriesPath, "", 0)
	require.NoError(t, err)

	for _, sheet := range sheets {
		for _, cat := range sheet.Cats {
			for _, sub := range cat.Subs {
				assert.Zero(t, sub.MaxEntries(),
					"typed entry with wrong category must not route into %s/%s/%s", sheet.Name, cat.Name, sub.Name)
			}
		}
	}
}

// TestLoadTaxonomy_TypelessUnambiguousEntryRoutes guards the no-regression-on-auto-path
// promise: a type-less entry with a unique (unambiguous) leaf still routes via the
// retained bare-name fallback, exactly as before Plan B.
func TestLoadTaxonomy_TypelessUnambiguousEntryRoutes(t *testing.T) {
	dir := t.TempDir()
	taxonomyPath := filepath.Join(dir, "taxonomy.json")
	require.NoError(t, os.WriteFile(taxonomyPath, []byte(`{
    "types": [
        { "name": "Fixas", "categories": [
            { "name": "Casa", "subcategories": ["Aluguel"] } ] }
    ],
    "incomeCategories": []
}`), 0644))

	entriesPath := filepath.Join(dir, "entries.jsonl")
	require.NoError(t, os.WriteFile(entriesPath,
		[]byte(`{"item":"Aluguel Jan","date":"05/01","value":2000.0,"subcategory":"Aluguel"}`+"\n"), 0644))

	sheets, _, err := LoadTaxonomy(taxonomyPath, entriesPath, "", 0)
	require.NoError(t, err)

	aluguel := sheets[0].Cats[0].Subs[0]
	require.Len(t, aluguel.Months[0], 1)
	assert.Equal(t, "Aluguel Jan", aluguel.Months[0][0].Item)
}

// TestLoadTaxonomy_NFDEntryRoutesToNFCTaxonomy guards the Unicode-normalization
// safeguard: the apply path (workbook-derived) and config/taxonomy.json are authored
// independently and may differ in accent encoding. Here the taxonomy uses composed
// (NFC) accents and the entry uses decomposed (NFD) accents for the SAME human-visible
// names ("Variáveis"/"Transporte" — wait, those carry the accent on Variáveis). The
// entry must still route, not silently warn+skip.
func TestLoadTaxonomy_NFDEntryRoutesToNFCTaxonomy(t *testing.T) {
	dir := t.TempDir()

	// NFC taxonomy: "Variáveis" with composed á.
	taxonomyPath := filepath.Join(dir, "taxonomy.json")
	require.NoError(t, os.WriteFile(taxonomyPath, []byte(`{
    "types": [
        { "name": "Variáveis", "categories": [
            { "name": "Alimentação", "subcategories": ["Feira"] } ] }
    ],
    "incomeCategories": []
}`), 0644))

	// NFD entry: decompose the type+category accents (á → a +  ́, ç → c +  ̧).
	nfdType := norm.NFD.String("Variáveis")
	nfdCat := norm.NFD.String("Alimentação")
	require.NotEqual(t, "Variáveis", nfdType, "precondition: NFD form must differ byte-wise from NFC")

	entriesPath := filepath.Join(dir, "entries.jsonl")
	line := `{"item":"Feira sem","date":"05/01","value":80.0,"type":"` + nfdType +
		`","category":"` + nfdCat + `","subcategory":"Feira"}` + "\n"
	require.NoError(t, os.WriteFile(entriesPath, []byte(line), 0644))

	sheets, _, err := LoadTaxonomy(taxonomyPath, entriesPath, "", 0)
	require.NoError(t, err)

	feira := sheets[0].Cats[0].Subs[0]
	require.Len(t, feira.Months[0], 1, "NFD-encoded typed entry must route despite NFC taxonomy")
	assert.Equal(t, "Feira sem", feira.Months[0][0].Item)
}

func TestParseDate_Malformed(t *testing.T) {
	cases := []struct {
		input    string
		dayErr   bool
		monthErr bool
	}{
		{"5/13", false, true},      // Invalid month
		{"32/01", true, false},     // Invalid day
		{"x/y", true, true},        // Non-numeric values
		{"05/01", false, false},    // Valid date
		{"2026-01-05", true, true}, // Wrong format
	}

	for _, tc := range cases {
		day, month, _, err := parseDate(tc.input)
		if tc.dayErr || tc.monthErr {
			assert.Error(t, err)
		} else {
			assert.NoError(t, err)
			assert.Equal(t, 5, day)
			assert.Equal(t, 1, month)
		}
	}
}

func TestParseDate_MultiYear(t *testing.T) {
	t.Run("DD/MM returns year 0", func(t *testing.T) {
		day, month, year, err := parseDate("05/01")
		require.NoError(t, err)
		assert.Equal(t, 5, day)
		assert.Equal(t, 1, month)
		assert.Equal(t, 0, year)
	})

	t.Run("DD/MM/YYYY returns year", func(t *testing.T) {
		day, month, year, err := parseDate("05/01/2026")
		require.NoError(t, err)
		assert.Equal(t, 5, day)
		assert.Equal(t, 1, month)
		assert.Equal(t, 2026, year)
	})

	t.Run("DD/MM/YY (year < 1000) returns error", func(t *testing.T) {
		_, _, _, err := parseDate("05/01/99")
		assert.Error(t, err)
	})

	t.Run("DD/MM/YYYY invalid month returns error", func(t *testing.T) {
		_, _, _, err := parseDate("05/13/2026")
		assert.Error(t, err)
	})

	t.Run("DD/MM/YYYY invalid day returns error", func(t *testing.T) {
		_, _, _, err := parseDate("32/01/2026")
		assert.Error(t, err)
	})
}
