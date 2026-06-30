package taxonomy

import (
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fixtureTypes is a deliberately small synthetic taxonomy that exercises every
// path-handling hazard the real (gitignored) config/taxonomy.json contains:
//
//   - a subcategory whose name contains '/' (Uber/Taxi) — proves SplitPath must
//     never parse on '/'.
//   - a CATEGORY whose name contains '/' (Alimentação / Limpeza) — same hazard,
//     one level up.
//   - a leaf name that repeats across types with the SAME category (Dentista in
//     Variáveis/Saúde and Extras/Saúde) — the ambiguous-resolution case.
//   - unique leaves that resolve from the bare name alone.
func fixtureTypes() []ExpenseType {
	return []ExpenseType{
		{Name: "Fixas", Cats: []Category{
			{Name: "Assinaturas", Subs: []Subcat{{Name: "Spotify"}}},
			{Name: "Saúde", Subs: []Subcat{{Name: "Plano"}}},
		}},
		{Name: "Variáveis", Cats: []Category{
			{Name: "Transporte", Subs: []Subcat{{Name: "Uber/Taxi"}}},
			{Name: "Saúde", Subs: []Subcat{{Name: "Dentista"}}},
			{Name: "Alimentação / Limpeza", Subs: []Subcat{{Name: "Mercado"}}},
		}},
		{Name: "Extras", Cats: []Category{
			{Name: "Saúde", Subs: []Subcat{{Name: "Dentista"}}},
		}},
	}
}

func TestBuildPathMap_EnumListsEveryLeafAsDisplayString(t *testing.T) {
	m, err := BuildPathMap(fixtureTypes())
	require.NoError(t, err)

	got := m.Enum()
	want := []string{
		"Extras/Saúde/Dentista",
		"Fixas/Assinaturas/Spotify",
		"Fixas/Saúde/Plano",
		"Variáveis/Alimentação / Limpeza/Mercado",
		"Variáveis/Saúde/Dentista",
		"Variáveis/Transporte/Uber/Taxi",
	}
	sort.Strings(got)
	assert.Equal(t, want, got, "enum should list all 6 leaves as Type/Category/Subcategory display strings")
}

func TestPathMapSplit_RoundTripsSlashyNamesViaReverseLookup(t *testing.T) {
	m, err := BuildPathMap(fixtureTypes())
	require.NoError(t, err)

	cases := []struct {
		path          string
		typ, cat, sub string
	}{
		// Subcategory with a slash: naive SplitN on '/' would yield the wrong parts.
		{"Variáveis/Transporte/Uber/Taxi", "Variáveis", "Transporte", "Uber/Taxi"},
		// Category with a slash.
		{"Variáveis/Alimentação / Limpeza/Mercado", "Variáveis", "Alimentação / Limpeza", "Mercado"},
		// Plain leaf.
		{"Fixas/Assinaturas/Spotify", "Fixas", "Assinaturas", "Spotify"},
		// Ambiguous leaf disambiguated by the type segment of the path itself.
		{"Extras/Saúde/Dentista", "Extras", "Saúde", "Dentista"},
	}
	for _, c := range cases {
		typ, cat, sub, ok := m.Split(c.path)
		assert.True(t, ok, "path %q should be in the map", c.path)
		assert.Equal(t, c.typ, typ, "type for %q", c.path)
		assert.Equal(t, c.cat, cat, "category for %q", c.path)
		assert.Equal(t, c.sub, sub, "subcategory for %q", c.path)
	}
}

func TestPathMapSplit_MissReturnsNotOK(t *testing.T) {
	m, err := BuildPathMap(fixtureTypes())
	require.NoError(t, err)

	_, _, _, ok := m.Split("Fixas/Nope/DoesNotExist")
	assert.False(t, ok, "a path not in the taxonomy must report not-ok, never a guessed split")
}

func TestPathEnum_ConvenienceMatchesMapEnum(t *testing.T) {
	sheets := fixtureTypes()
	m, err := BuildPathMap(sheets)
	require.NoError(t, err)

	enum, err := PathEnum(sheets)
	require.NoError(t, err)
	assert.ElementsMatch(t, m.Enum(), enum)
}

func TestPathFor_RoundTripsResolveLeafResult(t *testing.T) {
	sheets := fixtureTypes()
	m, err := BuildPathMap(sheets)
	require.NoError(t, err)

	// The intended pipeline: ResolveLeaf gives (type,cat); PathFor turns that triple
	// back into the canonical enum member — both through the map, no string surgery.
	typ, cat, err := ResolveLeaf(sheets, "Uber/Taxi", "")
	require.NoError(t, err)
	path, ok := m.PathFor(typ, cat, "Uber/Taxi")
	assert.True(t, ok)
	assert.Equal(t, "Variáveis/Transporte/Uber/Taxi", path)
}

func TestPathFor_UnknownTripleReturnsNotOK(t *testing.T) {
	m, err := BuildPathMap(fixtureTypes())
	require.NoError(t, err)

	_, ok := m.PathFor("Fixas", "Transporte", "Uber/Taxi") // wrong type for this leaf
	assert.False(t, ok)
}

func TestResolveLeaf_UniqueLeafResolvesFromBareName(t *testing.T) {
	sheets := fixtureTypes()

	typ, cat, err := ResolveLeaf(sheets, "Spotify", "")
	require.NoError(t, err)
	assert.Equal(t, "Fixas", typ)
	assert.Equal(t, "Assinaturas", cat)
}

func TestResolveLeaf_UniqueLeafIgnoresIrrelevantHint(t *testing.T) {
	sheets := fixtureTypes()

	// Hint is irrelevant for an unambiguous leaf: it resolves uniquely regardless.
	typ, cat, err := ResolveLeaf(sheets, "Mercado", "Extras")
	require.NoError(t, err)
	assert.Equal(t, "Variáveis", typ)
	assert.Equal(t, "Alimentação / Limpeza", cat)
}

func TestResolveLeaf_AmbiguousLeafWithoutHintErrors(t *testing.T) {
	sheets := fixtureTypes()

	_, _, err := ResolveLeaf(sheets, "Dentista", "")
	assert.ErrorIs(t, err, ErrLeafAmbiguous)
}

func TestResolveLeaf_AmbiguousLeafResolvedByTypeHint(t *testing.T) {
	sheets := fixtureTypes()

	typ, cat, err := ResolveLeaf(sheets, "Dentista", "Extras")
	require.NoError(t, err)
	assert.Equal(t, "Extras", typ)
	assert.Equal(t, "Saúde", cat)
}

func TestResolveLeaf_AmbiguousLeafWithNonMatchingHintErrors(t *testing.T) {
	sheets := fixtureTypes()

	// Hint names a type the leaf does not appear under → still ambiguous/unresolved.
	_, _, err := ResolveLeaf(sheets, "Dentista", "Fixas")
	assert.ErrorIs(t, err, ErrLeafAmbiguous)
}

func TestResolveLeaf_AbsentLeafErrors(t *testing.T) {
	sheets := fixtureTypes()

	_, _, err := ResolveLeaf(sheets, "Nonexistent", "")
	assert.ErrorIs(t, err, ErrLeafNotFound)
}

func TestCategoryForLeaf_ConsistentAcrossTypes(t *testing.T) {
	sheets := fixtureTypes()

	// Dentista repeats across Variáveis and Extras but is always under Saúde.
	cat, ok := CategoryForLeaf(sheets, "Dentista")
	assert.True(t, ok)
	assert.Equal(t, "Saúde", cat)

	cat, ok = CategoryForLeaf(sheets, "Spotify")
	assert.True(t, ok)
	assert.Equal(t, "Assinaturas", cat)

	_, ok = CategoryForLeaf(sheets, "Nonexistent")
	assert.False(t, ok)
}

func TestTypesForLeaf_ListsEveryOwningType(t *testing.T) {
	sheets := fixtureTypes()

	assert.Equal(t, []string{"Variáveis", "Extras"}, TypesForLeaf(sheets, "Dentista"))
	assert.Equal(t, []string{"Fixas"}, TypesForLeaf(sheets, "Spotify"))
	assert.Empty(t, TypesForLeaf(sheets, "Nonexistent"))
}
