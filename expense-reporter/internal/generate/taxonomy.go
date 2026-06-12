package generate

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

// ExpenseSheet is one of Fixas/Variáveis/Extras/Adicionais.
type ExpenseSheet struct {
	Name string
	Cats []Category
}

// RevenueBlock is one income block (Salário, 13°...).
type RevenueBlock struct {
	Category string // income category, e.g. "Receita"
	Label    string // block label
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

// headroomRows, perGroupPctRows and dataYear are package state set by Generate()
// from its Options before building (ported from the scratch builder's consts;
// the CLI is single-shot, so mutable package state is acceptable here).
var headroomRows = 0 // §3.2: regenerate-don't-insert → no spare rows by default

// perGroupPctRows toggles the per-group "% sobre despesas/receita" rows (§4.2).
var perGroupPctRows = true

// dataYear is the config year applied to entry dates.
var dataYear = 2026

