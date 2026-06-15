package generate

import (
	"fmt"

	"github.com/xuri/excelize/v2"
)

const (
	summarySheetName = "Listas de itens"
	lastSummaryCol   = "O"
)

// quoteSheet wraps a sheet name in single quotes for safe cross-sheet references.
// Plain (unquoted) names are emitted when the name has no spaces, matching the golden master.
func sheetRef(name, col string, row int) string {
	if needsQuote(name) {
		return fmt.Sprintf("'%s'!%s%d", name, col, row)
	}
	return fmt.Sprintf("%s!%s%d", name, col, row)
}

// needsQuote: Excel quotes a sheet name in a reference unless it is purely ASCII
// alphanumeric (letters/digits). Names with spaces, accents, or punctuation get quoted.
func needsQuote(s string) bool {
	for _, r := range s {
		isASCIIAlnum := (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9')
		if !isASCIIAlnum {
			return true
		}
	}
	return false
}

type summaryBuilder struct {
	f   *excelize.File
	st  *styleSet
	lbl Labels
	reg *layoutRegistry
	row int

	revenueTotalRow int            // Listas row of the Receitas-section grand total
	investTotalRow  int            // Listas row of Investimentos total
	sheetGrandRow   map[string]int // expense sheet -> Listas row of its grand-total
	balanceSepRow   int            // row of the thin 3.75 separator before the saldo block
}

func buildSummarySheet(f *excelize.File, st *styleSet, lbl Labels, reg *layoutRegistry) error {
	if _, err := f.NewSheet(summarySheetName); err != nil {
		return err
	}
	b := &summaryBuilder{f: f, st: st, lbl: lbl, reg: reg, sheetGrandRow: map[string]int{}}
	b.setWidths()
	f.SetPanes(summarySheetName, &excelize.Panes{Freeze: true, XSplit: 3, YSplit: 3, TopLeftCell: "D4", ActivePane: "bottomRight"})

	b.header()
	b.revenueSection()
	b.expenseSections()
	b.balanceBlock()
	b.setHeights()
	return nil
}

func (b *summaryBuilder) setWidths() {
	b.f.SetColWidth(summarySheetName, "A", "A", 12.86)
	b.f.SetColWidth(summarySheetName, "B", "B", 14)
	b.f.SetColWidth(summarySheetName, "C", "C", 18.14)
	b.f.SetColWidth(summarySheetName, "D", lastSummaryCol, 16.43)
}

func (b *summaryBuilder) setHeights() {
	for r := 1; r <= b.row; r++ {
		switch {
		case r == b.balanceSepRow:
			b.f.SetRowHeight(summarySheetName, r, 3.75)
		case r <= 20:
			b.f.SetRowHeight(summarySheetName, r, 15)
		default:
			b.f.SetRowHeight(summarySheetName, r, 15.75)
		}
	}
}

// header writes rows 3 (month banner) and 5 ("Valor").
func (b *summaryBuilder) header() {
	f, st := b.f, b.st
	f.MergeCell(summarySheetName, "A3", "C3")
	f.SetCellStyle(summarySheetName, "A3", "C3", st.SummaryMonth)
	for k := 0; k < 12; k++ {
		c := summaryMonthCol(k)
		f.SetCellValue(summarySheetName, c+"3", b.lbl.MonthNames[k])
		f.SetCellStyle(summarySheetName, c+"3", c+"3", st.SummaryMonth)
	}
	for k := 0; k < 12; k++ {
		f.SetCellValue(summarySheetName, summaryMonthCol(k)+"5", b.lbl.Amount)
	}
}

// monthFormulas writes D..O on a row, each a formula built by fn(k), styled valStyle.
func (b *summaryBuilder) monthFormulas(row, valStyle int, fn func(k int) string) {
	for k := 0; k < 12; k++ {
		c := summaryMonthCol(k)
		b.f.SetCellFormula(summarySheetName, cell(c, row), fn(k))
		b.f.SetCellStyle(summarySheetName, cell(c, row), cell(c, row), valStyle)
	}
}

// bandRow styles B..O of a row with the 333399 internal-separator band.
func (b *summaryBuilder) bandRow(row int) {
	b.f.SetCellStyle(summarySheetName, cell("B", row), cell(lastSummaryCol, row), b.st.IndigoBand)
}

// revenueSection: rows 6..14 — Receitas pulls + total, then Investimentos shell + total + %.
// The section label "Receitas" (col A) merges across the whole band (incl. Investimentos).
func (b *summaryBuilder) revenueSection() {
	b.row = 6
	sectionFirst := b.row

	firstPull, lastPull := b.writeRevenuePullRows()
	b.bandRow(b.row) // internal separator
	b.row++
	b.writeRevenueTotalRow(firstPull, lastPull)
	investRow := b.writeInvestmentsShellRow()
	b.writeInvestmentsTotalRow(investRow)
	b.writeInvestmentsPctRow()
	sectionLast := b.row

	mergeSection(b.f, b.st, b.lbl.RevenueSheet, sectionFirst, sectionLast)
	b.row += 4 // % row consumed at sectionLast; skip blanks to row 18
}

// writeRevenuePullRows emits one pull row per Receitas block and returns the
// pull band's first and last rows.
func (b *summaryBuilder) writeRevenuePullRows() (firstPull, lastPull int) {
	firstPull = b.row
	for _, blk := range b.reg.revenue.Blocks {
		b.f.SetCellStyle(summarySheetName, cell("B", b.row), cell("B", b.row), b.st.IndigoBand)
		b.f.SetCellValue(summarySheetName, cell("C", b.row), blk.Label)
		tr := blk.TotalRow
		b.monthFormulas(b.row, b.st.PullCur, func(k int) string {
			return sheetRef(b.lbl.RevenueSheet, expenseValorCol(k), tr)
		})
		b.row++
	}
	return firstPull, b.row - 1
}

// writeRevenueTotalRow emits the C0C0C0 total row (C=lbl.Total; D..O = SUM of
// pulls; B/C labels General, D..O currency) and records revenueTotalRow.
func (b *summaryBuilder) writeRevenueTotalRow(firstPull, lastPull int) {
	b.f.SetCellStyle(summarySheetName, cell("B", b.row), cell("C", b.row), b.st.SummaryTotalLbl)
	b.f.SetCellValue(summarySheetName, cell("C", b.row), b.lbl.Total)
	b.monthFormulas(b.row, b.st.SummaryTotalCur, func(k int) string {
		c := summaryMonthCol(k)
		return fmt.Sprintf("SUM(%s:%s)", cell(c, firstPull), cell(c, lastPull))
	})
	b.revenueTotalRow = b.row
	b.row += 3 // total, then two blank rows (10, 11)
}

// writeInvestmentsShellRow emits the manual-entry Investimentos row (no formulas)
// and returns its row.
func (b *summaryBuilder) writeInvestmentsShellRow() int {
	b.f.SetCellStyle(summarySheetName, cell("B", b.row), cell("B", b.row), b.st.IndigoBand)
	b.f.SetCellValue(summarySheetName, cell("C", b.row), b.lbl.Investments)
	for k := 0; k < 12; k++ {
		c := summaryMonthCol(k)
		b.f.SetCellStyle(summarySheetName, cell(c, b.row), cell(c, b.row), b.st.PullCur)
	}
	investRow := b.row
	b.row++
	return investRow
}

// writeInvestmentsTotalRow emits the Investimentos total (a direct pull of the
// shell row) and records investTotalRow.
func (b *summaryBuilder) writeInvestmentsTotalRow(investRow int) {
	b.f.SetCellStyle(summarySheetName, cell("B", b.row), cell("C", b.row), b.st.SummaryTotalLbl)
	b.f.SetCellValue(summarySheetName, cell("C", b.row), b.lbl.Total)
	b.monthFormulas(b.row, b.st.SummaryTotalCur, func(k int) string {
		return cell(summaryMonthCol(k), investRow)
	})
	b.investTotalRow = b.row
	b.row++
}

// writeInvestmentsPctRow emits the "% sobre receita" row for Investimentos
// (guarded against a zero revenue denominator).
func (b *summaryBuilder) writeInvestmentsPctRow() {
	b.f.SetCellStyle(summarySheetName, cell("B", b.row), cell("C", b.row), b.st.SummaryTotalLbl)
	b.f.SetCellValue(summarySheetName, cell("C", b.row), b.lbl.PctOfRevenue)
	invTot, recTot := b.investTotalRow, b.revenueTotalRow
	b.monthFormulas(b.row, b.st.GroupTotalPct, func(k int) string {
		c := summaryMonthCol(k)
		return fmt.Sprintf("IF(%s>0,%s/%s,0)", cell(c, recTot), cell(c, invTot), cell(c, recTot))
	})
}

func mergeSection(f *excelize.File, st *styleSet, label string, first, last int) {
	f.MergeCell(summarySheetName, cell("A", first), cell("A", last))
	f.SetCellValue(summarySheetName, cell("A", first), label)
	f.SetCellStyle(summarySheetName, cell("A", first), cell("A", last), st.SectionLabel)
}

// expenseSections: one band per source sheet (Fixas, Variáveis, Extras, Adicionais).
func (b *summaryBuilder) expenseSections() {
	for _, sName := range b.reg.sheetOrder {
		b.expenseSection(sName)
	}
}

func (b *summaryBuilder) expenseSection(sName string) {
	layout := b.reg.expense[sName]
	if layout == nil {
		return
	}
	sectionFirst := b.row
	plannedGrand := plannedGrandTotalRow(sectionFirst, layout.Cats)

	var groupTotalRows []int
	for _, ct := range layout.Cats {
		groupTotalRows = append(groupTotalRows, b.writeCategoryGroup(sName, ct, plannedGrand))
	}
	grandRow := b.writeSheetGrandTotalRow(sName, groupTotalRows)
	b.bandRow(b.row) // internal separator
	b.row++
	b.writeSectionPctOfRevenueRow(grandRow)
	sectionLast := b.row

	mergeSection(b.f, b.st, sName, sectionFirst, sectionLast)
	b.row += 2 // % row consumed; skip blank to next section
}

// writeCategoryGroup emits one categoria band: its pull rows, the merged col-B
// label, the group total, and (when enabled) the per-group percent rows.
// Returns the group-total row.
func (b *summaryBuilder) writeCategoryGroup(sName string, ct catTotals, plannedGrand int) int {
	firstPull, lastPull := b.writeCategoryPullRows(sName, ct)
	b.mergeCategoryBand(ct, firstPull, lastPull)
	groupTotalRow := b.writeGroupTotalRow(ct, firstPull, lastPull)
	if perGroupPctRows {
		b.emitGroupPctRows(groupTotalRow, plannedGrand)
	}
	return groupTotalRow
}

// writeCategoryPullRows emits one pull row per subcategory of the categoria and
// returns the pull band's first and last rows.
func (b *summaryBuilder) writeCategoryPullRows(sName string, ct catTotals) (firstPull, lastPull int) {
	firstPull = b.row
	for i, sub := range ct.Subs {
		if i == 0 {
			b.f.SetCellStyle(summarySheetName, cell("B", b.row), cell("B", b.row), b.st.IndigoBand)
		}
		b.f.SetCellValue(summarySheetName, cell("C", b.row), sub.Subcat)
		tr := sub.TotalRow
		b.monthFormulas(b.row, b.st.PullCur, func(k int) string {
			return sheetRef(sName, expenseValorCol(k), tr)
		})
		b.row++
	}
	return firstPull, b.row - 1
}

// mergeCategoryBand merges col B across the categoria's pull rows (skipping the
// degenerate single-row merge) and writes the categoria label.
func (b *summaryBuilder) mergeCategoryBand(ct catTotals, firstPull, lastPull int) {
	if lastPull > firstPull {
		b.f.MergeCell(summarySheetName, cell("B", firstPull), cell("B", lastPull))
	}
	b.f.SetCellValue(summarySheetName, cell("B", firstPull), ct.Category)
	b.f.SetCellStyle(summarySheetName, cell("B", firstPull), cell("B", lastPull), b.st.IndigoLabel)
}

// writeGroupTotalRow emits the CCCCFF group total (B = "Total <cat>"; D..O = SUM
// of pulls; B/C General, D..O currency) and returns its row.
func (b *summaryBuilder) writeGroupTotalRow(ct catTotals, firstPull, lastPull int) int {
	b.f.SetCellStyle(summarySheetName, cell("B", b.row), cell("C", b.row), b.st.GroupTotalLbl)
	b.f.SetCellValue(summarySheetName, cell("B", b.row), fmt.Sprintf(b.lbl.TotalCategoryFmt, ct.Category))
	b.monthFormulas(b.row, b.st.GroupTotalCur, func(k int) string {
		c := summaryMonthCol(k)
		return sumRange(cell(c, firstPull), cell(c, lastPull))
	})
	groupTotalRow := b.row
	b.row++
	return groupTotalRow
}

// writeSheetGrandTotalRow emits the C0C0C0 sheet grand-total (B = "Total
// despesas <sheet-lower>"; D..O = sum of the group totals), records it in
// sheetGrandRow, and returns its row.
func (b *summaryBuilder) writeSheetGrandTotalRow(sName string, groupTotalRows []int) int {
	b.f.SetCellStyle(summarySheetName, cell("B", b.row), cell("C", b.row), b.st.SummaryTotalLbl)
	b.f.SetCellValue(summarySheetName, cell("B", b.row), fmt.Sprintf(b.lbl.TotalSheetExpensesFmt, lower(sName)))
	grandRow := b.row
	b.monthFormulas(b.row, b.st.SummaryTotalCur, func(k int) string {
		c := summaryMonthCol(k)
		terms := make([]string, len(groupTotalRows))
		for i, gr := range groupTotalRows {
			terms[i] = cell(c, gr)
		}
		return sumList(terms)
	})
	b.sheetGrandRow[sName] = grandRow
	b.row++
	return grandRow
}

// writeSectionPctOfRevenueRow emits the section-level "% sobre receita" row
// (CCCCFF pct on D..O; B..C C0C0C0 label; guarded on the revenue denominator).
func (b *summaryBuilder) writeSectionPctOfRevenueRow(grandRow int) {
	b.f.SetCellStyle(summarySheetName, cell("B", b.row), cell("C", b.row), b.st.SummaryTotalLbl)
	b.f.SetCellValue(summarySheetName, cell("B", b.row), b.lbl.PctOfRevenue)
	recTot := b.revenueTotalRow
	b.monthFormulas(b.row, b.st.GroupTotalPct, func(k int) string {
		c := summaryMonthCol(k)
		return fmt.Sprintf("IF(%s>0,%s/%s,0)", cell(c, recTot), cell(c, grandRow), cell(c, recTot))
	})
}

// plannedGrandTotalRow returns the row where a despesa section's grand total will
// land, given the section's first row. Each categoria consumes its pull rows + 1
// group-total row + (when enabled) 2 per-group percent rows. The per-group
// "% sobre despesas" formula references the grand total, which is written after the
// loop — so its row must be known in advance.
func plannedGrandTotalRow(sectionFirst int, cats []catTotals) int {
	row := sectionFirst
	for _, ct := range cats {
		row += len(ct.Subs) + 1
		if perGroupPctRows {
			row += 2
		}
	}
	return row
}

// emitGroupPctRows writes a categoria's two percent rows beneath its group total:
// its share of total expenses (group / sheet grand total) and of revenue
// (group / revenue total).
func (b *summaryBuilder) emitGroupPctRows(grpRow, grandRow int) {
	b.groupPctRow(b.lbl.PctOfExpenses, grpRow, grandRow)
	b.groupPctRow(b.lbl.PctOfRevenue, grpRow, b.revenueTotalRow)
}

// groupPctRow writes one CCCCFF percent row: label in col B (B:C label style),
// D..O = IF(denom>0, group/denom, 0) per month.
func (b *summaryBuilder) groupPctRow(label string, grpRow, denomRow int) {
	f, st := b.f, b.st
	f.SetCellStyle(summarySheetName, cell("B", b.row), cell("C", b.row), st.GroupTotalLbl)
	f.SetCellValue(summarySheetName, cell("B", b.row), label)
	b.monthFormulas(b.row, st.GroupTotalPct, func(k int) string {
		c := summaryMonthCol(k)
		return fmt.Sprintf("IF(%s>0,%s/%s,0)", cell(c, denomRow), cell(c, grpRow), cell(c, denomRow))
	})
	b.row++
}

// balanceBlock: bottom aggregate block, labels in col A. Starts at the current row
// (dynamic — per-group percent rows push it down) behind a thin separator.
func (b *summaryBuilder) balanceBlock() {
	b.balanceSepRow = b.row
	b.f.SetRowHeight(summarySheetName, b.balanceSepRow, 3.75)
	b.row++
	order := b.reg.sheetOrder

	_, _, totalRendaRow := b.writeIncomeRows()
	despRows, totalDespRow := b.writeExpenseRows(order)
	b.writeExpenseShareBlock(order, despRows, totalDespRow)
	b.writeIncomeShareBlock(order, despRows, totalRendaRow)
	b.writeFinalBalanceRow(totalRendaRow, totalDespRow)
}

// writeIncomeRows writes the Revenue pull, Investments pull, and TotalIncome sum rows.
// Returns all three row numbers for use in later formula references.
func (b *summaryBuilder) writeIncomeRows() (receitaRow, investRow, totalRendaRow int) {
	receitaRow = b.balanceRow(b.lbl.Revenue, b.st.SummaryTotalLbl, b.st.SummaryTotalCur, func(k int) string {
		return cell(summaryMonthCol(k), b.revenueTotalRow)
	})
	investRow = b.balanceRow(b.lbl.Investments, b.st.SummaryTotalLbl, b.st.SummaryTotalCur, func(k int) string {
		return cell(summaryMonthCol(k), b.investTotalRow)
	})
	totalRendaRow = b.balanceRow(b.lbl.TotalIncome, b.st.NearBlack, b.st.NearBlackCur, func(k int) string {
		c := summaryMonthCol(k)
		return fmt.Sprintf("SUM(%s:%s)", cell(c, receitaRow), cell(c, investRow))
	})
	return
}

// writeExpenseRows writes one balance row per sheet (using sheetGrandRow) and the
// TotalExpenses sum row. Returns the per-sheet row map and the total row number.
func (b *summaryBuilder) writeExpenseRows(order []string) (despRows map[string]int, totalDespRow int) {
	despRows = map[string]int{}
	for _, s := range order {
		gr := b.sheetGrandRow[s]
		despRows[s] = b.balanceRow(fmt.Sprintf(b.lbl.SheetExpensesFmt, lower(s)), b.st.SummaryTotalLbl, b.st.SummaryTotalCur, func(k int) string {
			return cell(summaryMonthCol(k), gr)
		})
	}
	totalDespRow = b.balanceRow(b.lbl.TotalExpenses, b.st.NearBlack, b.st.NearBlackCur, func(k int) string {
		c := summaryMonthCol(k)
		terms := make([]string, len(order))
		for i, s := range order {
			terms[i] = cell(c, despRows[s])
		}
		return sumList(terms)
	})
	return
}

// writeExpenseShareBlock writes the ExpenseShareHeader label then one percent row per sheet
// showing each sheet's share of total expenses.
func (b *summaryBuilder) writeExpenseShareBlock(order []string, despRows map[string]int, totalDespRow int) {
	b.balanceLabelRow(b.lbl.ExpenseShareHeader)
	for _, s := range order {
		dr := despRows[s]
		b.balanceRowPct(s, func(k int) string {
			c := summaryMonthCol(k)
			return fmt.Sprintf("IF(%s>0,%s/%s,0)", cell(c, totalDespRow), cell(c, dr), cell(c, totalDespRow))
		})
	}
}

// writeIncomeShareBlock writes the IncomeShareHeader label then one percent row per sheet
// showing each sheet's share of total income.
func (b *summaryBuilder) writeIncomeShareBlock(order []string, despRows map[string]int, totalRendaRow int) {
	b.balanceLabelRow(b.lbl.IncomeShareHeader)
	for _, s := range order {
		dr := despRows[s]
		b.balanceRowPct(s, func(k int) string {
			c := summaryMonthCol(k)
			return fmt.Sprintf("IF(%s>0,%s/%s,0)", cell(c, totalRendaRow), cell(c, dr), cell(c, totalRendaRow))
		})
	}
}

// writeFinalBalanceRow writes the Balance row (total income minus total expenses).
func (b *summaryBuilder) writeFinalBalanceRow(totalRendaRow, totalDespRow int) {
	b.balanceRow(b.lbl.Balance, b.st.NearBlack, b.st.NearBlackCur, func(k int) string {
		c := summaryMonthCol(k)
		return fmt.Sprintf("%s-%s", cell(c, totalRendaRow), cell(c, totalDespRow))
	})
}

// balanceRow writes a label (col A) + month formulas. A..C carry the fill-only label style
// (General numFmt); D..O carry valStyle. Returns the row number.
func (b *summaryBuilder) balanceRow(label string, lblStyle, valStyle int, fn func(k int) string) int {
	r := b.row
	b.f.SetCellValue(summarySheetName, cell("A", r), label)
	b.f.SetCellStyle(summarySheetName, cell("A", r), cell("C", r), lblStyle)
	b.monthFormulas(r, valStyle, fn)
	b.row++
	return r
}

// balanceRowPct: label A (C0C0C0), D..O percent (CCCCFF).
func (b *summaryBuilder) balanceRowPct(label string, fn func(k int) string) int {
	r := b.row
	b.f.SetCellValue(summarySheetName, cell("A", r), label)
	b.f.SetCellStyle(summarySheetName, cell("A", r), cell("C", r), b.st.SummaryTotalLbl)
	b.monthFormulas(r, b.st.GroupTotalPct, fn)
	b.row++
	return r
}

// balanceLabelRow: a full-width 333333 label row (col A label, rest near-black).
func (b *summaryBuilder) balanceLabelRow(label string) {
	r := b.row
	b.f.SetCellValue(summarySheetName, cell("A", r), label)
	b.f.SetCellStyle(summarySheetName, cell("A", r), cell(lastSummaryCol, r), b.st.NearBlack)
	b.row++
}
