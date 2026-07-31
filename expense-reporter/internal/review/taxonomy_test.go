package review

import (
	"testing"

	"expense-reporter/internal/taxonomy"
	"github.com/stretchr/testify/assert"
)

func TestBuildTaxonomy(t *testing.T) {
	t.Run("order is preserved verbatim at all three levels", func(t *testing.T) {
		// The previous implementation sorted because it folded workbook rows through Go maps
		// with random iteration order, and a JSON array carries the author's own order.
		// Re-introducing a sort would silently reorder the picker.
		inputTypes := []taxonomy.ExpenseType{
			{
				Name: "Zebra",
				Cats: []taxonomy.Category{
					{
						Name: "Yak",
						Subs: []taxonomy.Subcat{
							{Name: "Xenon"},
							{Name: "Wombat"},
						},
					},
					{
						Name: "Vulture",
						Subs: []taxonomy.Subcat{
							{Name: "Ursa"},
						},
					},
				},
			},
			{
				Name: "Alpha",
				Cats: []taxonomy.Category{
					{
						Name: "Beta",
						Subs: []taxonomy.Subcat{
							{Name: "Gamma"},
						},
					},
				},
			},
		}

		actual := BuildTaxonomy(inputTypes)

		assert.Len(t, actual.Types, 2, "Should preserve type count")
		assert.Equal(t, "Zebra", actual.Types[0].Name, "Type order should be preserved")
		assert.Equal(t, "Alpha", actual.Types[1].Name, "Type order should be preserved")

		assert.Len(t, actual.Types[0].Categories, 2, "Should preserve category count for first type")
		assert.Equal(t, "Yak", actual.Types[0].Categories[0].Name, "Category order should be preserved")
		assert.Equal(t, "Vulture", actual.Types[0].Categories[1].Name, "Category order should be preserved")

		assert.Len(t, actual.Types[1].Categories, 1, "Should preserve category count for second type")
		assert.Equal(t, "Beta", actual.Types[1].Categories[0].Name, "Category order should be preserved")

		assert.Len(t, actual.Types[0].Categories[0].Subcategories, 2, "Should preserve subcategory count for first category of first type")
		assert.Equal(t, "Xenon", actual.Types[0].Categories[0].Subcategories[0], "Subcategory order should be preserved")
		assert.Equal(t, "Wombat", actual.Types[0].Categories[0].Subcategories[1], "Subcategory order should be preserved")

		assert.Len(t, actual.Types[0].Categories[1].Subcategories, 1, "Should preserve subcategory count for second category of first type")
		assert.Equal(t, "Ursa", actual.Types[0].Categories[1].Subcategories[0], "Subcategory order should be preserved")

		assert.Len(t, actual.Types[1].Categories[0].Subcategories, 1, "Should preserve subcategory count for first category of second type")
		assert.Equal(t, "Gamma", actual.Types[1].Categories[0].Subcategories[0], "Subcategory order should be preserved")
	})

	t.Run("empty input yields an empty, non-nil Types slice", func(t *testing.T) {
		// Non-nil is the point, not pedantry: this struct is serialized into the review
		// page, where a nil slice marshals to `null` and the page's iteration over
		// taxonomy.types throws, while `[]` iterates zero times harmlessly. Do not
		// "simplify" the make(...) in BuildTaxonomy to a var declaration.
		actual := BuildTaxonomy([]taxonomy.ExpenseType{})

		assert.NotNil(t, actual.Types, "Types slice should not be nil")
		assert.Empty(t, actual.Types, "Types slice should be empty when input is empty")
	})

	t.Run("a type with no categories and a category with no subcategories are carried through rather than dropped", func(t *testing.T) {
		// The picker must show exactly what the taxonomy authored.
		inputTypes := []taxonomy.ExpenseType{
			{
				Name: "EmptyType",
				Cats: []taxonomy.Category{}, // No categories
			},
			{
				Name: "PopulatedType",
				Cats: []taxonomy.Category{
					{
						Name: "EmptyCategory",
						Subs: []taxonomy.Subcat{}, // No subcategories
					},
				},
			},
		}

		actual := BuildTaxonomy(inputTypes)

		assert.Len(t, actual.Types, 2, "Should carry through types with no categories")
		assert.Equal(t, "EmptyType", actual.Types[0].Name, "Should preserve name of type with no categories")

		assert.Len(t, actual.Types[1].Categories, 1, "Should carry through category with no subcategories")
		assert.Equal(t, "EmptyCategory", actual.Types[1].Categories[0].Name, "Should preserve name of category with no subcategories")
	})

	t.Run("the per-month entry payload is dropped", func(t *testing.T) {
		// The picker takes only Name; the projection does not depend on, and does not leak,
		// the entry data.
		inputTypes := []taxonomy.ExpenseType{
			{
				Name: "TestType",
				Cats: []taxonomy.Category{
					{
						Name: "TestCategory",
						Subs: []taxonomy.Subcat{
							{
								Name: "TestSubcategory",
								Months: [12][]taxonomy.Entry{
									0: {{Item: "Entry1", Day: 1, Value: 100.0}},
									1: {{Item: "Entry2", Day: 2, Value: 200.0}},
								},
							},
						},
					},
				},
			},
		}

		actual := BuildTaxonomy(inputTypes)

		assert.Len(t, actual.Types[0].Categories[0].Subcategories, 1, "Should carry only subcategory names")
		assert.Equal(t, "TestSubcategory", actual.Types[0].Categories[0].Subcategories[0], "Should preserve name of subcategory even with populated month data")
	})
}
