package classifier

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMatchStrength(t *testing.T) {
	keywords := KeywordIndex{
		// Unambiguous, maximal specificity.
		"diarista": {DominantSubcategory: "Diarista", Specificity: 1.0, Subcategories: []string{"Diarista"}},
		"faxina":   {DominantSubcategory: "Faxina", Specificity: 1.0, Subcategories: []string{"Faxina"}},
		// Unambiguous, sub-maximal specificity.
		"uber": {DominantSubcategory: "Uber", Specificity: 0.95, Subcategories: []string{"Uber"}},
		// One keyword spanning two subcategories at equal specificity → top-score tie.
		"mercado": {DominantSubcategory: "Supermercado", Specificity: 0.60, Subcategories: []string{"Supermercado", "Farmácia"}},
	}

	tests := []struct {
		name string
		item string
		want MatchSignal
	}{
		{
			name: "unambiguous max specificity",
			item: "Diarista Letícia",
			want: MatchSignal{Matched: true, TopScore: 1.0, TopSubcategory: "Diarista", Ambiguous: false},
		},
		{
			name: "unambiguous sub-maximal specificity",
			item: "Uber Centro",
			want: MatchSignal{Matched: true, TopScore: 0.95, TopSubcategory: "Uber", Ambiguous: false},
		},
		{
			name: "single keyword spanning two subcategories ties at the top",
			item: "compras no mercado",
			want: MatchSignal{Matched: true, TopScore: 0.60, TopSubcategory: "Farmácia", Ambiguous: true},
		},
		{
			name: "two distinct max-specificity keywords tie",
			item: "diarista e faxina",
			want: MatchSignal{Matched: true, TopScore: 1.0, TopSubcategory: "Diarista", Ambiguous: true},
		},
		{
			name: "no keyword matches",
			item: "xpto qualquer coisa",
			want: MatchSignal{},
		},
		{
			name: "empty item yields no tokens",
			item: "",
			want: MatchSignal{},
		},
		{
			name: "only single-rune tokens are filtered out",
			item: "a b c",
			want: MatchSignal{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := MatchStrength(tt.item, keywords)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestMatchStrength_NilIndex(t *testing.T) {
	// A nil index matches nothing and must not panic.
	assert.Equal(t, MatchSignal{}, MatchStrength("Uber Centro", nil))
}
