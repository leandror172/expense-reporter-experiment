package classifier

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTokenize(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantNil  bool
		expected []string
	}{
		{
			name:     "basic tokens",
			input:    "Uber Centro",
			expected: []string{"uber", "centro"},
		},
		{
			name:     "mixed case and spaces",
			input:    "VA compras",
			expected: []string{"va", "compras"},
		},
		{
			name:     "accented characters preserved, special chars removed",
			input:    "café & pão",
			expected: []string{"café", "pão"},
		},
		{
			name:    "single rune filtered out",
			input:   "X",
			wantNil: true,
		},
		{
			name:    "empty string",
			input:   "",
			wantNil: true,
		},
		{
			name:     "all caps lowercased",
			input:    "SUPERMERCADO",
			expected: []string{"supermercado"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tokenize(tt.input)
			if tt.wantNil {
				assert.Empty(t, result)
			} else {
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestSelectExamples(t *testing.T) {
	pool := []Example{
		// Uber/Transporte — 2 Training, 1 Corrected
		{Item: "Uber Centro", Value: 25.50, Date: "15/06", Subcategory: "Uber", Category: "Transporte", Source: SourceTraining},
		{Item: "Uber X", Value: 18.75, Date: "14/06", Subcategory: "Uber", Category: "Transporte", Source: SourceTraining},
		{Item: "Uber 24h", Value: 35.00, Date: "13/06", Subcategory: "Uber", Category: "Transporte", Source: SourceCorrected},
		// Supermercado/Alimentação — 2 Training
		{Item: "Supermercado Extra", Value: 89.20, Date: "12/06", Subcategory: "Supermercado", Category: "Alimentação", Source: SourceTraining},
		{Item: "Mercado Pão de Açúcar", Value: 145.30, Date: "11/06", Subcategory: "Supermercado", Category: "Alimentação", Source: SourceTraining},
		// Farmácia/Saúde — 1 Training
		{Item: "Farmácia São Paulo", Value: 22.50, Date: "10/06", Subcategory: "Farmácia", Category: "Saúde", Source: SourceTraining},
	}

	keywords := KeywordIndex{
		"uber": {
			DominantSubcategory: "Uber",
			Specificity:         0.95,
			Subcategories:       []string{"Uber"},
		},
		"supermercado": {
			DominantSubcategory: "Supermercado",
			Specificity:         0.80,
			Subcategories:       []string{"Supermercado"},
		},
		// Low specificity: appears in both Supermercado and Farmácia
		"mercado": {
			DominantSubcategory: "Supermercado",
			Specificity:         0.60,
			Subcategories:       []string{"Supermercado", "Farmácia"},
		},
	}

	tests := []struct {
		name        string
		item        string
		topK        int
		wantNil     bool
		wantLen     int
		wantFirst   ExampleSource // source of result[0], ignored if wantLen==0
		checkFirst  bool
	}{
		{
			name:       "high specificity - all 3 Uber examples returned, Corrected first",
			item:       "Uber Centro",
			topK:       5,
			wantLen:    3,
			wantFirst:  SourceCorrected,
			checkFirst: true,
		},
		{
			name:       "high specificity with topK limit",
			item:       "Uber Centro",
			topK:       2,
			wantLen:    2,
			wantFirst:  SourceCorrected,
			checkFirst: true,
		},
		{
			name:    "ambiguous match - interleaved from Supermercado and Farmácia",
			item:    "produto mercado",
			topK:    5,
			// pool has 2 Supermercado + 1 Farmácia → interleaved: [S0, F0, S1] = 3
			wantLen: 3,
		},
		{
			name:    "no keyword match returns nil",
			item:    "XYZ unknown item",
			topK:    3,
			wantNil: true,
		},
		{
			name:       "high specificity - Supermercado examples",
			item:       "supermercado extra",
			topK:       3,
			wantLen:    2,
			wantFirst:  SourceTraining,
			checkFirst: true,
		},
		{
			name:    "topK zero returns nil",
			item:    "Uber Centro",
			topK:    0,
			wantNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SelectExamples(tt.item, pool, keywords, tt.topK)

			if tt.wantNil {
				assert.Nil(t, result)
				return
			}

			require.NotNil(t, result)
			assert.Len(t, result, tt.wantLen)
			if tt.checkFirst && len(result) > 0 {
				assert.Equal(t, tt.wantFirst, result[0].Source, "unexpected source at position 0")
			}
		})
	}
}

func TestSelectExamplesCorrectedPriority(t *testing.T) {
	// Pool with one Corrected and two Training entries for the same subcategory.
	pool := []Example{
		{Item: "Rappi lanches", Subcategory: "Delivery", Category: "Alimentação", Source: SourceTraining},
		{Item: "Rappi comida", Subcategory: "Delivery", Category: "Alimentação", Source: SourceCorrected},
		{Item: "Rappi pizza", Subcategory: "Delivery", Category: "Alimentação", Source: SourceTraining},
	}
	keywords := KeywordIndex{
		"rappi": {DominantSubcategory: "Delivery", Specificity: 0.90, Subcategories: []string{"Delivery"}},
	}

	result := SelectExamples("rappi comida", pool, keywords, 3)
	require.NotNil(t, result)
	require.Len(t, result, 3)
	assert.Equal(t, SourceCorrected, result[0].Source, "corrected entry must come first")
}
