package review

import "expense-reporter/internal/taxonomy"

// BuildTaxonomy projects the taxonomy loaded from config/taxonomy.json into the picker
// tree the review page embeds.
//
// The SOURCE is the point of this function. Review used to derive the picker from the
// workbook's "Referência de Categorias" sheet while every other command routed with
// config/taxonomy.json — two vocabularies for one close cycle, and they drifted:
// measured at 7 divergent leaves (e.g. the workbook's IRFF vs the taxonomy's IRRF).
// A reviewer picking a workbook-only leaf produced a row that generate-workbook could
// not route, so it was warn-skipped and vanished from the workbook with no error the
// user would ever see. Worse, two such picks were logged as CORRECTIONS — the
// highest-priority few-shot source — teaching the classifier the unroutable spelling.
//
// Order is preserved verbatim at all three levels, and NOTHING here sorts. The previous
// implementation sorted (a hardcoded type order, then alphabetical) purely because it
// folded workbook rows through Go maps, whose iteration order is random, and the
// spreadsheet carried no authored order of its own. A JSON array does carry one. Do not
// "restore" the sorting: it would silently reorder what the taxonomy author chose.
func BuildTaxonomy(types []taxonomy.ExpenseType) Taxonomy {
	result := make([]Type, 0, len(types))
	for _, t := range types {
		categories := buildCategories(t.Cats)
		result = append(result, Type{
			Name:       t.Name,
			Categories: categories,
		})
	}
	return Taxonomy{Types: result}
}

// buildCategories converts a slice of taxonomy.Category into review.Categories,
// preserving the source order and dropping per-month entry data.
func buildCategories(cats []taxonomy.Category) []Category {
	result := make([]Category, 0, len(cats))
	for _, c := range cats {
		subcategories := make([]string, 0, len(c.Subs))
		for _, s := range c.Subs {
			subcategories = append(subcategories, s.Name)
		}
		result = append(result, Category{
			Name:          c.Name,
			Subcategories: subcategories,
		})
	}
	return result
}
