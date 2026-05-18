package review

import (
	"slices"
	"sort"

	"expense-reporter/internal/resolver"
)

var sheetOrder = []string{"Fixas", "Variáveis", "Extras", "Adicionais"}

func BuildTaxonomy(mappings map[string][]resolver.SubcategoryMapping) Taxonomy {
	// Intermediate: sheetName → categoryName → set of subcategories
	tree := map[string]map[string]map[string]struct{}{}

	for _, list := range mappings {
		for _, m := range list {
			if _, ok := tree[m.SheetName]; !ok {
				tree[m.SheetName] = map[string]map[string]struct{}{}
			}
			if _, ok := tree[m.SheetName][m.Category]; !ok {
				tree[m.SheetName][m.Category] = map[string]struct{}{}
			}
			tree[m.SheetName][m.Category][m.Subcategory] = struct{}{}
		}
	}

	// Collect and sort sheet names
	sheetNames := make([]string, 0, len(tree))
	for name := range tree {
		sheetNames = append(sheetNames, name)
	}
	sort.Slice(sheetNames, func(i, j int) bool {
		return sheetRank(sheetNames[i]) < sheetRank(sheetNames[j])
	})

	sheets := make([]Sheet, 0, len(sheetNames))
	for _, sheetName := range sheetNames {
		catMap := tree[sheetName]

		catNames := make([]string, 0, len(catMap))
		for name := range catMap {
			catNames = append(catNames, name)
		}
		sort.Strings(catNames)

		categories := make([]Category, 0, len(catNames))
		for _, catName := range catNames {
			subMap := catMap[catName]

			subs := make([]string, 0, len(subMap))
			for sub := range subMap {
				subs = append(subs, sub)
			}
			sort.Strings(subs)

			categories = append(categories, Category{
				Name:          catName,
				Subcategories: subs,
			})
		}

		sheets = append(sheets, Sheet{
			Name:       sheetName,
			Categories: categories,
		})
	}

	return Taxonomy{Sheets: sheets}
}

func sheetRank(name string) int {
	idx := slices.Index(sheetOrder, name)
	if idx >= 0 {
		return idx
	}
	return len(sheetOrder) // unknown sheets sort after known ones, then alphabetically handled by caller
}
