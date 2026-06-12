package generate

// subcatTotal records the total-row position of one subcategory block on an expense sheet,
// so Listas can wire pull formulas to it.
type subcatTotal struct {
	Sheet    string
	Category string
	Subcat   string
	TotalRow int // 1-based row of the total row
}

// catTotals groups the subcatTotal rows of one categoria (preserving order).
type catTotals struct {
	Category string
	Subs     []subcatTotal
}

// sheetLayout is the recorded layout of one expense sheet: its categorias and their
// subcategory total rows, in taxonomy order.
type sheetLayout struct {
	Sheet string
	Cats  []catTotals
}

// revenueLayout records each Receitas block's total row.
type revenueLayout struct {
	Blocks []revenueBlockTotal
}

type revenueBlockTotal struct {
	Category string
	Label    string
	TotalRow int
}

// layoutRegistry accumulates positions across all source sheets for Listas wiring.
type layoutRegistry struct {
	expense    map[string]*sheetLayout // keyed by sheet name
	sheetOrder []string                // expense sheet names in taxonomy (= build) order
	revenue    revenueLayout
}

func newLayoutRegistry() *layoutRegistry {
	return &layoutRegistry{expense: map[string]*sheetLayout{}}
}
