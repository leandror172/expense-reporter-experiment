// Package taxonomy holds the domain types and loader for the expense taxonomy.
// It is a pure-input package: it imports nothing from the generate package,
// preventing import cycles.
package taxonomy

// Entry is one expense/income line within a subcategory's month.
// Day is the day-of-month; the builder pairs it with the column's month + the
// config year to form the cell date. Value is the BRL amount.
type Entry struct {
	Item  string
	Day   int
	Value float64
}

// Subcat is one subcategory block. v2: composed sub-items are a single col-B string.
// Months holds entries per month (index 0 = Janeiro). A month with no entries is nil.
type Subcat struct {
	Name   string
	Months [12][]Entry
}

// MaxEntries returns the largest entry count across all months — the busiest month
// sets the block's data-row count (spec §3.2: data rows = max-entries-per-month + headroom).
func (s Subcat) MaxEntries() int {
	max := 0
	for _, m := range s.Months {
		if len(m) > max {
			max = len(m)
		}
	}
	return max
}

// Category groups subcategories under one bold category label.
type Category struct {
	Name string
	Subs []Subcat
}

// ExpenseType is one of Fixas/Variáveis/Extras/Adicionais.
type ExpenseType struct {
	Name string
	Cats []Category
}

// RevenueBlock is one income leaf — the intersection of a taxonomy category, a
// mid-level block grouping, and a subline label. One RevenueBlock == one data row
// on the Receitas sheet (mirrors how one Subcat == one expense data row).
//
// Shape (3-level, WS-C):
//
//	Category  "Receitas"       — top-level income category name from taxonomy
//	Block     "Salário"        — mid-level block grouping
//	Label     "INSS"           — leaf subline (the actual data row)
//
// The flat format (legacy taxonomy: blocks:["Salário"]) produces Block==Label for each
// string entry; the nested format (blocks:[{block,sublines:[...]}]) populates all three.
type RevenueBlock struct {
	Category string // e.g. "Receitas"
	Block    string // e.g. "Salário" — mid-level grouping (Step 3: used for sheet band headers)
	Label    string // e.g. "INSS"   — leaf subline label
	Months   [12][]Entry
}

// MaxEntries mirrors Subcat.MaxEntries for income blocks.
func (b RevenueBlock) MaxEntries() int {
	max := 0
	for _, m := range b.Months {
		if len(m) > max {
			max = len(m)
		}
	}
	return max
}
