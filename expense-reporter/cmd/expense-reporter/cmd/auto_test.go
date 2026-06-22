package cmd

import (
	"strings"
	"testing"

	"expense-reporter/internal/taxonomy"

	"github.com/stretchr/testify/assert"
)

func TestResolveExpenseType(t *testing.T) {
	fixas := taxonomy.ExpenseType{
		Name: "Fixas",
		Cats: []taxonomy.Category{
			{Name: "Moradia", Subs: []taxonomy.Subcat{{Name: "Aluguel"}, {Name: "Condomínio"}}},
		},
	}
	variaveis := taxonomy.ExpenseType{
		Name: "Variáveis",
		Cats: []taxonomy.Category{
			{Name: "Mercado", Subs: []taxonomy.Subcat{{Name: "Feira"}}},
			{Name: "Saúde", Subs: []taxonomy.Subcat{{Name: "Dentista"}}},
		},
	}
	extras := taxonomy.ExpenseType{
		Name: "Extras",
		Cats: []taxonomy.Category{
			{Name: "Saúde", Subs: []taxonomy.Subcat{{Name: "Dentista"}}},
		},
	}

	tests := map[string]struct {
		idx         taxonomy.TypeIndex
		category    string
		subcategory string
		want        string
	}{
		"unique match": {
			idx:         taxonomy.BuildTypeIndex([]taxonomy.ExpenseType{fixas, variaveis}),
			category:    "Moradia",
			subcategory: "Aluguel",
			want:        "Fixas",
		},
		"ambiguous pair": {
			idx:         taxonomy.BuildTypeIndex([]taxonomy.ExpenseType{variaveis, extras}),
			category:    "Saúde",
			subcategory: "Dentista",
			want:        "",
		},
		"no match": {
			idx:         taxonomy.BuildTypeIndex([]taxonomy.ExpenseType{fixas, variaveis}),
			category:    "NãoExiste",
			subcategory: "Nada",
			want:        "",
		},
		"empty index": {
			idx:         taxonomy.TypeIndex{},
			category:    "Moradia",
			subcategory: "Aluguel",
			want:        "",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got := resolveExpenseType(tc.idx, tc.category, tc.subcategory)
			assert.Equal(t, tc.want, got)
		})
	}
}

// --- confirmInsert (interactive prompt with reader) ---

func TestConfirmInsert_Yes(t *testing.T) {
	for _, input := range []string{"y\n", "Y\n", "yes\n", "YES\n"} {
		r := strings.NewReader(input)
		if !confirmInsert(r) {
			t.Errorf("confirmInsert(%q) = false, want true", input)
		}
	}
}

func TestConfirmInsert_No(t *testing.T) {
	for _, input := range []string{"n\n", "N\n", "no\n", "\n", "anything\n"} {
		r := strings.NewReader(input)
		if confirmInsert(r) {
			t.Errorf("confirmInsert(%q) = true, want false", input)
		}
	}
}
