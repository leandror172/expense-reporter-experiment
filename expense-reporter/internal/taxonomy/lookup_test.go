package taxonomy

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/text/unicode/norm"
)

// sampleTypes builds a small expense-type tree for reverse-lookup tests.
//
// Layout (type / category / subcategory):
//
//	Fixas     / Moradia  / Aluguel, Condomínio
//	Variáveis / Mercado  / Feira
//	Variáveis / Saúde    / Dentista
//	Extras    / Saúde    / Dentista   <- same (category, sub) as Variáveis → ambiguous
func sampleTypes() []ExpenseType {
	return []ExpenseType{
		{
			Name: "Fixas",
			Cats: []Category{
				{Name: "Moradia", Subs: []Subcat{{Name: "Aluguel"}, {Name: "Condomínio"}}},
			},
		},
		{
			Name: "Variáveis",
			Cats: []Category{
				{Name: "Mercado", Subs: []Subcat{{Name: "Feira"}}},
				{Name: "Saúde", Subs: []Subcat{{Name: "Dentista"}}},
			},
		},
		{
			Name: "Extras",
			Cats: []Category{
				{Name: "Saúde", Subs: []Subcat{{Name: "Dentista"}}},
			},
		},
	}
}

func TestLookupType_UniquePairResolvesToType(t *testing.T) {
	idx := BuildTypeIndex(sampleTypes())

	cases := []struct {
		category, subcategory, wantType string
	}{
		{"Moradia", "Aluguel", "Fixas"},
		{"Moradia", "Condomínio", "Fixas"},
		{"Mercado", "Feira", "Variáveis"},
	}
	for _, c := range cases {
		got, err := idx.LookupType(c.category, c.subcategory)
		require.NoError(t, err, "%s/%s", c.category, c.subcategory)
		assert.Equal(t, c.wantType, got, "%s/%s", c.category, c.subcategory)
	}
}

func TestLookupType_PairAbsentReturnsNotFound(t *testing.T) {
	idx := BuildTypeIndex(sampleTypes())

	// subcategory exists but under a different category
	_, err := idx.LookupType("Mercado", "Dentista")
	assert.ErrorIs(t, err, ErrTypeNotFound)

	// neither category nor subcategory exists
	_, err = idx.LookupType("Lazer", "Cinema")
	assert.ErrorIs(t, err, ErrTypeNotFound)
}

func TestLookupType_SamePairUnderTwoTypesIsAmbiguous(t *testing.T) {
	idx := BuildTypeIndex(sampleTypes())

	// Saúde/Dentista lives under both Variáveis and Extras.
	_, err := idx.LookupType("Saúde", "Dentista")
	assert.ErrorIs(t, err, ErrTypeAmbiguous)
}

// A pair must remain ambiguous regardless of how many times the colliding path
// recurs — a third occurrence must not silently re-resolve it (mirrors the
// bare-name registerTarget 3× re-add trap).
func TestLookupType_AmbiguityIsSticky(t *testing.T) {
	types := sampleTypes()
	types = append(types, ExpenseType{
		Name: "Adicionais",
		Cats: []Category{{Name: "Saúde", Subs: []Subcat{{Name: "Dentista"}}}},
	})
	idx := BuildTypeIndex(types)

	_, err := idx.LookupType("Saúde", "Dentista")
	assert.ErrorIs(t, err, ErrTypeAmbiguous)
}

// Lookups must succeed across accent-encoding skew: an NFD-encoded query key must
// match an NFC-authored taxonomy key (same guarantee normalizeKey gives routing).
func TestLookupType_NFCNormalizedKeys(t *testing.T) {
	idx := BuildTypeIndex(sampleTypes())

	// Force a decomposed (NFD) query key — i + combining acute — so the test proves
	// normalization rather than relying on how the editor saved the literal.
	nfdCategory := norm.NFD.String("Moradia")
	nfdSub := norm.NFD.String("Condomínio")
	require.NotEqual(t, norm.NFC.String("Condomínio"), nfdSub,
		"query key must be decomposed to exercise NFC matching")

	got, err := idx.LookupType(nfdCategory, nfdSub)
	require.NoError(t, err)
	assert.Equal(t, "Fixas", got)
}

func TestLookupType_EmptyIndexReturnsNotFound(t *testing.T) {
	idx := BuildTypeIndex(nil)
	_, err := idx.LookupType("Moradia", "Aluguel")
	assert.ErrorIs(t, err, ErrTypeNotFound)
}

// The sentinel errors must be distinguishable so callers can choose to leave the
// entry type-less (for the bare-name fallback) on either condition.
func TestLookupType_SentinelsAreDistinct(t *testing.T) {
	assert.False(t, errors.Is(ErrTypeNotFound, ErrTypeAmbiguous))
	assert.False(t, errors.Is(ErrTypeAmbiguous, ErrTypeNotFound))
}
