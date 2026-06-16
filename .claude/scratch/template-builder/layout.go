package main

// subcatTotal records the total-row position of one subcategory block on an expense sheet,
// so Listas can wire pull formulas to it.
type subcatTotal struct {
	Sheet     string
	Categoria string
	Subcat    string
	TotalRow  int // 1-based row of the total row
}

// catTotals groups the subcatTotal rows of one categoria (preserving order).
type catTotals struct {
	Categoria string
	Subs      []subcatTotal
}

// sheetLayout is the recorded layout of one expense sheet: its categorias and their
// subcategory total rows, in taxonomy order.
type sheetLayout struct {
	Sheet string
	Cats  []catTotals
}

// receitasLayout records each Receitas block's total row.
type receitasLayout struct {
	Blocks []receitasBlockTotal
}

type receitasBlockTotal struct {
	Category string
	Label    string
	TotalRow int
}

// layoutRegistry accumulates positions across all source sheets for Listas wiring.
type layoutRegistry struct {
	expense  map[string]*sheetLayout // keyed by sheet name
	receitas receitasLayout
}

func newLayoutRegistry() *layoutRegistry {
	return &layoutRegistry{expense: map[string]*sheetLayout{}}
}
