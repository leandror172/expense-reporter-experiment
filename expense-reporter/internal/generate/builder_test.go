package generate

import (
	"fmt"
	"testing"

	"expense-reporter/internal/taxonomy"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xuri/excelize/v2"
)

// entriesInOneMonth builds a [12][]taxonomy.Entry with n identical entries placed in January.
func entriesInOneMonth(n int) [12][]taxonomy.Entry {
	var months [12][]taxonomy.Entry
	for i := 0; i < n; i++ {
		months[0] = append(months[0], taxonomy.Entry{Item: "X", Day: 5, Value: 100})
	}
	return months
}

func TestSubcatMaxEntries(t *testing.T) {
	// Busiest month sets the count: 2 in Jan, 3 in Feb -> 3.
	sub := taxonomy.Subcat{Name: "S", Months: [12][]taxonomy.Entry{
		0: {{Item: "a"}, {Item: "b"}},
		1: {{Item: "c"}, {Item: "d"}, {Item: "e"}},
	}}
	assert.Equal(t, 3, sub.MaxEntries())
	assert.Equal(t, 0, taxonomy.Subcat{Name: "empty"}.MaxEntries())
}

func TestRevenueBlockMaxEntries(t *testing.T) {
	blk := taxonomy.RevenueBlock{Block: "Salário", Label: "Salário", Months: [12][]taxonomy.Entry{
		0: {{Item: "a"}},
		3: {{Item: "b"}, {Item: "c"}},
	}}
	assert.Equal(t, 2, blk.MaxEntries())
}

func TestCalculateSubcatBlockRows(t *testing.T) {
	require.Equal(t, 0, headroomRows, "these expectations assume headroom 0")
	cases := []struct {
		name                         string
		entries                      int
		wantFirst, wantLast, wantTot int
	}{
		{"empty -> 1 row", 0, 3, 3, 4},
		{"one entry -> 1 row", 1, 3, 3, 4},
		{"two entries -> 2 rows", 2, 3, 4, 5},
		{"three entries -> 3 rows", 3, 3, 5, 6},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			sub := taxonomy.Subcat{Name: "S", Months: entriesInOneMonth(tc.entries)}
			first, last, total := calculateBlockRows(3, sub.MaxEntries())
			assert.Equal(t, tc.wantFirst, first)
			assert.Equal(t, tc.wantLast, last)
			assert.Equal(t, tc.wantTot, total)
		})
	}
}

func TestCalculateRevenueBlockRows(t *testing.T) {
	require.Equal(t, 0, headroomRows)
	// 2 entries -> 2 data rows starting at row 6: 6,7 data + 8 total.
	blk := taxonomy.RevenueBlock{Block: "Salário", Label: "Salário", Months: entriesInOneMonth(2)}
	first, last, total := calculateBlockRows(6, blk.MaxEntries())
	assert.Equal(t, 6, first)
	assert.Equal(t, 7, last)
	assert.Equal(t, 8, total)
	// zero-entry block still gets one data row.
	first, last, total = calculateBlockRows(6, taxonomy.RevenueBlock{Block: "none", Label: "none"}.MaxEntries())
	assert.Equal(t, 6, first)
	assert.Equal(t, 6, last)
	assert.Equal(t, 7, total)
}

func TestPlannedGrandTotalRow(t *testing.T) {
	require.True(t, perGroupPctRows, "expectation assumes per-group percent rows on")
	// Two categorias with 2 and 1 subcats. Each consumes len(Subs)+1 group total+2 pct rows.
	cats := []catTotals{
		{Category: "A", Subs: make([]subcatTotal, 2)}, // 2+1+2 = 5
		{Category: "B", Subs: make([]subcatTotal, 1)}, // 1+1+2 = 4
	}
	assert.Equal(t, 6+5+4, plannedGrandTotalRow(6, cats))
}

func TestNeedsQuoteAndSheetRef(t *testing.T) {
	assert.False(t, needsQuote("Fixas"))
	assert.True(t, needsQuote("Variáveis"))       // accented
	assert.True(t, needsQuote("Listas de itens")) // spaces
	assert.Equal(t, "Fixas!E5", sheetRef("Fixas", "E", 5))
	assert.Equal(t, "'Listas de itens'!E5", sheetRef("Listas de itens", "E", 5))
}

func TestNewPtBRLabelsNormalized(t *testing.T) {
	lbl := newPtBRLabels()
	// Normalization: lowercase per-group rows, fixed "Porcentagem" typo.
	assert.Equal(t, "% sobre despesas", lbl.PctOfExpenses)
	assert.Equal(t, "% sobre receita", lbl.PctOfRevenue)
	assert.Equal(t, "Porcentagem da despesa", lbl.ExpenseShareHeader)
	assert.Equal(t, "Porcentagem da renda", lbl.IncomeShareHeader)
	assert.Equal(t, "Total %s", lbl.TotalCategoryFmt)
	assert.Equal(t, "Janeiro", lbl.MonthNames[0])
}

// --- integration: build sheets in-memory and assert formulas/values ---

func newTestFile(t *testing.T) (*excelize.File, *styleSet, Labels) {
	t.Helper()
	f := excelize.NewFile()
	t.Cleanup(func() { _ = f.Close() })
	st, err := newStyles(f)
	require.NoError(t, err)
	return f, st, newPtBRLabels()
}

// TestBuildRevenueSumRange is the regression guard for the inverted-SUM bug:
// the income total row must SUM over the block's data rows, not an empty/backwards range.
func TestBuildRevenueSumRange(t *testing.T) {
	f, st, lbl := newTestFile(t)
	reg := newLayoutRegistry()
	blocks := []taxonomy.RevenueBlock{
		{Category: "Receitas", Block: "Salário", Label: "Bruto", Months: [12][]taxonomy.Entry{
			0: {{Item: "Salário", Day: 5, Value: 5000}},
		}},
	}
	require.NoError(t, buildRevenueSheet(f, st, lbl, blocks, reg))

	// One entry -> data row 3, total row 4. Jan valor column is E.
	formula, err := f.GetCellFormula("Receitas", "E4")
	require.NoError(t, err)
	assert.Equal(t, "SUM(E3:E3)", formula, "total must span the single data row, not invert")

	// col A carries the Block name merged across data+total rows.
	blockLabel, err := f.GetCellValue("Receitas", "A3")
	require.NoError(t, err)
	assert.Equal(t, "Salário", blockLabel, "col A must carry the Block name")

	// col B carries the subline Label.
	subLabel, err := f.GetCellValue("Receitas", "B3")
	require.NoError(t, err)
	assert.Equal(t, "Bruto", subLabel, "col B must carry the subline Label")

	item, err := f.GetCellValue("Receitas", "C3")
	require.NoError(t, err)
	assert.Equal(t, "Salário", item)

	valor, err := f.GetCellValue("Receitas", "E3")
	require.NoError(t, err)
	assert.NotEmpty(t, valor, "amount must be written into the data row")

	require.Len(t, reg.revenue.Blocks, 1)
	assert.Equal(t, "Salário", reg.revenue.Blocks[0].Block)
	assert.Equal(t, "Bruto", reg.revenue.Blocks[0].Label)
	assert.Equal(t, 4, reg.revenue.Blocks[0].TotalRow)
}

func TestBuildRevenueSumRangeMultiEntry(t *testing.T) {
	f, st, lbl := newTestFile(t)
	reg := newLayoutRegistry()
	blocks := []taxonomy.RevenueBlock{
		{Category: "Receitas", Block: "Variável", Label: "Comissão", Months: [12][]taxonomy.Entry{
			0: {{Item: "c1", Day: 3, Value: 100}, {Item: "c2", Day: 9, Value: 200}},
		}},
	}
	require.NoError(t, buildRevenueSheet(f, st, lbl, blocks, reg))
	// Two entries -> data rows 3,4; total row 5.
	formula, err := f.GetCellFormula("Receitas", "E5")
	require.NoError(t, err)
	assert.Equal(t, "SUM(E3:E4)", formula)
}

// TestBuildRevenueMultiBlock is the 3-level layout guard: two Block groups, each with
// two subline leaves. Asserts Block name in col A merged across its leaves; subline Label
// in col B per-leaf; separator gap between blocks; per-leaf registry entries.
func TestBuildRevenueMultiBlock(t *testing.T) {
	f, st, lbl := newTestFile(t)
	reg := newLayoutRegistry()
	blocks := []taxonomy.RevenueBlock{
		// Block "Salário" — two leaves (Bruto, INSS), 1 entry each.
		{Category: "Receitas", Block: "Salário", Label: "Bruto", Months: entriesInOneMonth(1)},
		{Category: "Receitas", Block: "Salário", Label: "INSS", Months: entriesInOneMonth(1)},
		// Block "Variável" — one leaf (Comissão), 1 entry.
		{Category: "Receitas", Block: "Variável", Label: "Comissão", Months: entriesInOneMonth(1)},
	}
	require.NoError(t, buildRevenueSheet(f, st, lbl, blocks, reg))

	// Salário block: Bruto leaf — rows 3(data)+4(total); INSS leaf — rows 5(data)+6(total).
	// Col A must carry "Salário" at A3 (merged A3:A6).
	aVal, err := f.GetCellValue("Receitas", "A3")
	require.NoError(t, err)
	assert.Equal(t, "Salário", aVal, "col A row 3 = Block name for first block group")

	// Col B per-leaf: B3 = "Bruto", B5 = "INSS".
	bBruto, err := f.GetCellValue("Receitas", "B3")
	require.NoError(t, err)
	assert.Equal(t, "Bruto", bBruto)

	bINSS, err := f.GetCellValue("Receitas", "B5")
	require.NoError(t, err)
	assert.Equal(t, "INSS", bINSS)

	// Total rows: Bruto=4, INSS=6. SUM formulas span their own data rows.
	fBruto, err := f.GetCellFormula("Receitas", "E4")
	require.NoError(t, err)
	assert.Equal(t, "SUM(E3:E3)", fBruto)

	fINSS, err := f.GetCellFormula("Receitas", "E6")
	require.NoError(t, err)
	assert.Equal(t, "SUM(E5:E5)", fINSS)

	// Separator gap: row 7=blank, row 8=separator, row 9=blank.
	// Variável block: Comissão leaf starts at row 10.
	aVar, err := f.GetCellValue("Receitas", "A10")
	require.NoError(t, err)
	assert.Equal(t, "Variável", aVar, "Variável block must start at row 10 after separator gap")

	bComissao, err := f.GetCellValue("Receitas", "B10")
	require.NoError(t, err)
	assert.Equal(t, "Comissão", bComissao)

	fComissao, err := f.GetCellFormula("Receitas", "E11")
	require.NoError(t, err)
	assert.Equal(t, "SUM(E10:E10)", fComissao)

	// Registry: 3 entries, one per leaf subline, in order.
	require.Len(t, reg.revenue.Blocks, 3)
	assert.Equal(t, revenueBlockTotal{Category: "Receitas", Block: "Salário", Label: "Bruto", TotalRow: 4}, reg.revenue.Blocks[0])
	assert.Equal(t, revenueBlockTotal{Category: "Receitas", Block: "Salário", Label: "INSS", TotalRow: 6}, reg.revenue.Blocks[1])
	assert.Equal(t, revenueBlockTotal{Category: "Receitas", Block: "Variável", Label: "Comissão", TotalRow: 11}, reg.revenue.Blocks[2])
}

// TestBuildExpenseTypeSizingAndSum checks max-entries sizing + the total SUM spans
// exactly the data rows, and a typed entry lands in the right cell.
func TestBuildExpenseTypeSizingAndSum(t *testing.T) {
	f, st, lbl := newTestFile(t)
	reg := newLayoutRegistry()
	sh := taxonomy.ExpenseType{Name: "Fixas", Cats: []taxonomy.Category{
		{Name: "Habitação", Subs: []taxonomy.Subcat{
			{Name: "Diarista", Months: [12][]taxonomy.Entry{
				0: {{Item: "Diarista", Day: 3, Value: 150}, {Item: "Diarista", Day: 10, Value: 160}, {Item: "Diarista", Day: 17, Value: 155.5}},
			}},
		}},
	}}
	require.NoError(t, buildExpenseType(f, st, lbl, sh, reg))

	// 3 entries -> data rows 3..5, total row 6. Jan valor column E.
	formula, err := f.GetCellFormula("Fixas", "E6")
	require.NoError(t, err)
	assert.Equal(t, "SUM(E3:E5)", formula)

	item, err := f.GetCellValue("Fixas", "C3")
	require.NoError(t, err)
	assert.Equal(t, "Diarista", item)

	// Total row carries the localized label + en-dash in the date column.
	total, err := f.GetCellValue("Fixas", "C6")
	require.NoError(t, err)
	assert.Equal(t, lbl.Total, total)
	dash, err := f.GetCellValue("Fixas", "D6")
	require.NoError(t, err)
	assert.Equal(t, lbl.TotalDash, dash)

	layout := reg.expense["Fixas"]
	require.NotNil(t, layout)
	assert.Equal(t, 6, layout.Cats[0].Subs[0].TotalRow)
}

// TestBuildSummaryRevenuePerBlock verifies the per-Block grouping in the Receitas summary
// section: pull rows, col-B Block label, "Total <Block>" group-total rows, and the grand
// total that sums only the per-Block totals (not the individual pulls).
func TestBuildSummaryRevenuePerBlock(t *testing.T) {
	f, st, lbl := newTestFile(t)
	reg := newLayoutRegistry()
	// Two Block groups: "Salário" has two leaves; "Variável" has one.
	blocks := []taxonomy.RevenueBlock{
		{Category: "Receitas", Block: "Salário", Label: "Bruto", Months: [12][]taxonomy.Entry{0: {{Item: "x", Day: 1, Value: 1}}}},
		{Category: "Receitas", Block: "Salário", Label: "INSS", Months: [12][]taxonomy.Entry{0: {{Item: "x", Day: 1, Value: 1}}}},
		{Category: "Receitas", Block: "Variável", Label: "Comissão", Months: [12][]taxonomy.Entry{0: {{Item: "x", Day: 1, Value: 1}}}},
	}
	require.NoError(t, buildRevenueSheet(f, st, lbl, blocks, reg))
	require.NoError(t, buildSummarySheet(f, st, lbl, reg))

	// Revenue section starts at row 6.
	// Row 6: pull — Bruto (col C)
	brutoLabel, err := f.GetCellValue(summarySheetName, "C6")
	require.NoError(t, err)
	assert.Equal(t, "Bruto", brutoLabel, "row 6 col C should be first Salário leaf")

	// Row 7: pull — INSS (col C)
	inssLabel, err := f.GetCellValue(summarySheetName, "C7")
	require.NoError(t, err)
	assert.Equal(t, "INSS", inssLabel, "row 7 col C should be second Salário leaf")

	// Row 6 col B should have the Block label "Salário" (merged over rows 6-7).
	blockLabel, err := f.GetCellValue(summarySheetName, "B6")
	require.NoError(t, err)
	assert.Equal(t, "Salário", blockLabel, "col B should carry the Block group label")

	// Row 8: "Total Salário" group total — sums D6:D7 (contiguous pulls).
	totalSalLabel, err := f.GetCellValue(summarySheetName, "B8")
	require.NoError(t, err)
	assert.Equal(t, fmt.Sprintf(lbl.TotalCategoryFmt, "Salário"), totalSalLabel)
	totalSalFormula, err := f.GetCellFormula(summarySheetName, "D8")
	require.NoError(t, err)
	assert.Equal(t, "SUM(D6:D7)", totalSalFormula, "group-total should sum its pull range")

	// Row 9: pull — Comissão (Variável, single-leaf block).
	comissaoLabel, err := f.GetCellValue(summarySheetName, "C9")
	require.NoError(t, err)
	assert.Equal(t, "Comissão", comissaoLabel)

	// Row 10: "Total Variável" — single-pull group; formula sums D9 (collapsed range).
	totalVarLabel, err := f.GetCellValue(summarySheetName, "B10")
	require.NoError(t, err)
	assert.Equal(t, fmt.Sprintf(lbl.TotalCategoryFmt, "Variável"), totalVarLabel)
	totalVarFormula, err := f.GetCellFormula(summarySheetName, "D10")
	require.NoError(t, err)
	assert.Equal(t, "SUM(D9)", totalVarFormula, "single-leaf group-total collapses to SUM(D9)")

	// Row 11: internal bandRow separator.
	// Row 12: Receitas grand total — sumList of D8 and D10 (block totals, non-contiguous).
	grandFormula, err := f.GetCellFormula(summarySheetName, "D12")
	require.NoError(t, err)
	assert.Equal(t, "SUM(D8,D10)", grandFormula, "grand total should sum block-total rows only")

	// Confirm the Investimentos % formula references the grand-total row 12 (revenueTotalRow).
	// Layout: grand total row 12, b.row+=3 → row 15 (Investimentos shell), 16 (total), 17 (pct).
	pctFormula, err := f.GetCellFormula(summarySheetName, "D17")
	require.NoError(t, err)
	assert.Contains(t, pctFormula, "D12", "Investimentos % should reference the grand-total row 12")
}

// TestBuildSummaryPerGroupPctRows checks the new per-group percent rows are emitted
// with IF-percent formulas that reference the sheet grand total (forward reference).
func TestBuildSummaryPerGroupPctRows(t *testing.T) {
	f, st, lbl := newTestFile(t)
	reg := newLayoutRegistry()
	blocks := []taxonomy.RevenueBlock{
		{Category: "Receitas", Block: "Salário", Label: "Bruto", Months: [12][]taxonomy.Entry{0: {{Item: "Salário", Day: 5, Value: 5000}}}},
	}
	require.NoError(t, buildRevenueSheet(f, st, lbl, blocks, reg))
	sh := taxonomy.ExpenseType{Name: "Fixas", Cats: []taxonomy.Category{
		{Name: "Habitação", Subs: []taxonomy.Subcat{
			{Name: "Aluguel", Months: [12][]taxonomy.Entry{0: {{Item: "Aluguel", Day: 5, Value: 1200}}}},
		}},
	}}
	require.NoError(t, buildExpenseType(f, st, lbl, sh, reg))
	require.NoError(t, buildSummarySheet(f, st, lbl, reg))

	// Scan column B of Listas for the per-group percent labels.
	rows, err := f.GetRows(summarySheetName)
	require.NoError(t, err)
	var pctExpRow int
	for r := 1; r <= len(rows); r++ {
		v, _ := f.GetCellValue(summarySheetName, cell("B", r))
		if v == lbl.PctOfExpenses {
			pctExpRow = r
			break
		}
	}
	require.NotZero(t, pctExpRow, "expected a %q row in column B", lbl.PctOfExpenses)

	// The Jan formula on that row (col D) must be an IF percent expression.
	formula, err := f.GetCellFormula(summarySheetName, cell("D", pctExpRow))
	require.NoError(t, err)
	assert.Contains(t, formula, "IF(")
	assert.Contains(t, formula, "/")
}
