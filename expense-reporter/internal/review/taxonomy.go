package review

import (
	"slices"
	"sort"

	"expense-reporter/internal/resolver"
)

var sheetOrder = []string{"Fixas", "Variáveis", "Extras", "Adicionais"}

func BuildTaxonomy(mappings map[string][]resolver.SubcategoryMapping) Taxonomy {
	tree := buildTree(mappings)
	names := sortedSheetNames(tree)

	sheets := make([]Sheet, 0, len(names))
	for _, name := range names {
		sheets = append(sheets, buildSheet(name, tree[name]))
	}
	return Taxonomy{Sheets: sheets}
}

// buildTree converts flat SubcategoryMappings into a 3-level nested set:
// sheetName → categoryName → set of subcategories.
func buildTree(mappings map[string][]resolver.SubcategoryMapping) map[string]map[string]map[string]struct{} {
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
	return tree
}

// sortedSheetNames returns sheet names ordered by sheetOrder, then alphabetically.
func sortedSheetNames(tree map[string]map[string]map[string]struct{}) []string {
	names := make([]string, 0, len(tree))
	for name := range tree {
		names = append(names, name)
	}
	sort.Slice(names, func(i, j int) bool {
		ri, rj := sheetRank(names[i]), sheetRank(names[j])
		if ri != rj {
			return ri < rj
		}
		return names[i] < names[j]
	})
	return names
}

// buildSheet converts one sheet's category map into a Sheet with sorted categories and subcategories.
func buildSheet(name string, catMap map[string]map[string]struct{}) Sheet {
	catNames := make([]string, 0, len(catMap))
	for n := range catMap {
		catNames = append(catNames, n)
	}
	sort.Strings(catNames)

	categories := make([]Category, 0, len(catNames))
	for _, catName := range catNames {
		subs := make([]string, 0, len(catMap[catName]))
		for sub := range catMap[catName] {
			subs = append(subs, sub)
		}
		sort.Strings(subs)
		categories = append(categories, Category{Name: catName, Subcategories: subs})
	}

	return Sheet{Name: name, Categories: categories}
}

func sheetRank(name string) int {
	idx := slices.Index(sheetOrder, name)
	if idx >= 0 {
		return idx
	}
	return len(sheetOrder) // unknown sheets sort after known ones, then alphabetically handled by caller
}
