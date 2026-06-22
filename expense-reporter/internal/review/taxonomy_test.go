package review

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"expense-reporter/internal/resolver"
)

func TestBuildTaxonomy(t *testing.T) {
	t.Run("groups by sheet and category", func(t *testing.T) {
		mappings := map[string][]resolver.SubcategoryMapping{
			"sheet1": {
				{SheetName: "sheet1", Category: "cat1", Subcategory: "sub1"},
				{SheetName: "sheet1", Category: "cat1", Subcategory: "sub2"},
				{SheetName: "sheet1", Category: "cat2", Subcategory: "sub3"},
			},
		}

		result := BuildTaxonomy(mappings)

		assert.Len(t, result.Types, 1)
		typ := result.Types[0]
		assert.Equal(t, "sheet1", typ.Name)
		assert.Len(t, typ.Categories, 2)

		cat1 := typ.Categories[0]
		assert.Equal(t, "cat1", cat1.Name)
		assert.Equal(t, []string{"sub1", "sub2"}, cat1.Subcategories)

		cat2 := typ.Categories[1]
		assert.Equal(t, "cat2", cat2.Name)
		assert.Equal(t, []string{"sub3"}, cat2.Subcategories)
	})

	t.Run("deduplicates subcategories", func(t *testing.T) {
		mappings := map[string][]resolver.SubcategoryMapping{
			"sheet1": {
				{SheetName: "sheet1", Category: "cat1", Subcategory: "sub1"},
				{SheetName: "sheet1", Category: "cat1", Subcategory: "sub1"},
			},
		}

		result := BuildTaxonomy(mappings)

		assert.Len(t, result.Types, 1)
		cat := result.Types[0].Categories[0]
		assert.Equal(t, []string{"sub1"}, cat.Subcategories)
	})

	t.Run("deterministic sheet ordering", func(t *testing.T) {
		mappings := map[string][]resolver.SubcategoryMapping{
			"a": {{SheetName: "Adicionais", Category: "c", Subcategory: "s"}},
			"b": {{SheetName: "Extras", Category: "c", Subcategory: "s"}},
			"c": {{SheetName: "Variáveis", Category: "c", Subcategory: "s"}},
			"d": {{SheetName: "Fixas", Category: "c", Subcategory: "s"}},
		}

		result := BuildTaxonomy(mappings)

		assert.Len(t, result.Types, 4)
		assert.Equal(t, "Fixas", result.Types[0].Name)
		assert.Equal(t, "Variáveis", result.Types[1].Name)
		assert.Equal(t, "Extras", result.Types[2].Name)
		assert.Equal(t, "Adicionais", result.Types[3].Name)
	})

	t.Run("unknown sheet sorts after known ones", func(t *testing.T) {
		mappings := map[string][]resolver.SubcategoryMapping{
			"a": {{SheetName: "Outros", Category: "c", Subcategory: "s"}},
			"b": {{SheetName: "Fixas", Category: "c", Subcategory: "s"}},
		}

		result := BuildTaxonomy(mappings)

		assert.Len(t, result.Types, 2)
		assert.Equal(t, "Fixas", result.Types[0].Name)
		assert.Equal(t, "Outros", result.Types[1].Name)
	})

	t.Run("subcategories sorted alphabetically within category", func(t *testing.T) {
		mappings := map[string][]resolver.SubcategoryMapping{
			"s": {
				{SheetName: "s", Category: "cat1", Subcategory: "Zebra"},
				{SheetName: "s", Category: "cat1", Subcategory: "Alpha"},
				{SheetName: "s", Category: "cat1", Subcategory: "Mango"},
			},
		}

		result := BuildTaxonomy(mappings)

		assert.Equal(t, []string{"Alpha", "Mango", "Zebra"}, result.Types[0].Categories[0].Subcategories)
	})

	t.Run("multiple unknown sheets sort alphabetically after known ones", func(t *testing.T) {
		mappings := map[string][]resolver.SubcategoryMapping{
			"a": {{SheetName: "Zeta", Category: "c", Subcategory: "s"}},
			"b": {{SheetName: "Fixas", Category: "c", Subcategory: "s"}},
			"c": {{SheetName: "Alpha", Category: "c", Subcategory: "s"}},
		}

		result := BuildTaxonomy(mappings)

		assert.Len(t, result.Types, 3)
		assert.Equal(t, "Fixas", result.Types[0].Name)
		assert.Equal(t, "Alpha", result.Types[1].Name)
		assert.Equal(t, "Zeta", result.Types[2].Name)
	})

	t.Run("categories sorted alphabetically within sheet", func(t *testing.T) {
		mappings := map[string][]resolver.SubcategoryMapping{
			"s": {
				{SheetName: "s", Category: "Zebra", Subcategory: "sub1"},
				{SheetName: "s", Category: "Alpha", Subcategory: "sub1"},
			},
		}

		result := BuildTaxonomy(mappings)

		assert.Len(t, result.Types[0].Categories, 2)
		assert.Equal(t, "Alpha", result.Types[0].Categories[0].Name)
		assert.Equal(t, "Zebra", result.Types[0].Categories[1].Name)
	})
}
