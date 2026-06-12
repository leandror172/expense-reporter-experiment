package main

import (
	"fmt"

	"github.com/xuri/excelize/v2"
)

const (
	listasName    = "Listas de itens"
	lastListasCol = "O"
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

type listasBuilder struct {
	f   *excelize.File
	st  *styleSet
	lbl Labels
	reg *layoutRegistry
	row int

	receitasTotalRow int            // Listas row of the Receitas-section grand total
	investTotalRow   int            // Listas row of Investimentos total
	sheetGrandRow    map[string]int // expense sheet -> Listas row of its grand-total
	saldoSepRow      int            // row of the thin 3.75 separator before the saldo block
}

func buildListas(f *excelize.File, st *styleSet, lbl Labels, reg *layoutRegistry) error {
	if _, err := f.NewSheet(listasName); err != nil {
		return err
	}
	b := &listasBuilder{f: f, st: st, lbl: lbl, reg: reg, sheetGrandRow: map[string]int{}}
	b.setWidths()
	f.SetPanes(listasName, &excelize.Panes{Freeze: true, XSplit: 3, YSplit: 3, TopLeftCell: "D4", ActivePane: "bottomRight"})

	b.header()
	b.receitasSection()
	b.despesasSections()
	b.saldoBlock()
	b.setHeights()
	return nil
}

func (b *listasBuilder) setWidths() {
	b.f.SetColWidth(listasName, "A", "A", 12.86)
	b.f.SetColWidth(listasName, "B", "B", 14)
	b.f.SetColWidth(listasName, "C", "C", 18.14)
	b.f.SetColWidth(listasName, "D", lastListasCol, 16.43)
}

func (b *listasBuilder) setHeights() {
	for r := 1; r <= b.row; r++ {
		switch {
		case r == b.saldoSepRow:
			b.f.SetRowHeight(listasName, r, 3.75)
		case r <= 20:
			b.f.SetRowHeight(listasName, r, 15)
		default:
			b.f.SetRowHeight(listasName, r, 15.75)
		}
	}
}

// header writes rows 3 (month banner) and 5 ("Valor").
func (b *listasBuilder) header() {
	f, st := b.f, b.st
	f.MergeCell(listasName, "A3", "C3")
	f.SetCellStyle(listasName, "A3", "C3", st.ListasMonth)
	for k := 0; k < 12; k++ {
		c := listasMonthCol(k)
		f.SetCellValue(listasName, c+"3", b.lbl.MonthNames[k])
		f.SetCellStyle(listasName, c+"3", c+"3", st.ListasMonth)
	}
	for k := 0; k < 12; k++ {
		f.SetCellValue(listasName, listasMonthCol(k)+"5", b.lbl.Amount)
	}
}

// monthFormulas writes D..O on a row, each a formula built by fn(k), styled valStyle.
func (b *listasBuilder) monthFormulas(row, valStyle int, fn func(k int) string) {
	for k := 0; k < 12; k++ {
		c := listasMonthCol(k)
		b.f.SetCellFormula(listasName, cell(c, row), fn(k))
		b.f.SetCellStyle(listasName, cell(c, row), cell(c, row), valStyle)
	}
}

// bandRow styles B..O of a row with the 333399 internal-separator band.
func (b *listasBuilder) bandRow(row int) {
	b.f.SetCellStyle(listasName, cell("B", row), cell(lastListasCol, row), b.st.IndigoBand)
}

// receitasSection: rows 6..14 — Receitas pulls + total, then Investimentos shell + total + %.
// The section label "Receitas" (col A) merges across the whole band (incl. Investimentos).
func (b *listasBuilder) receitasSection() {
	f, st := b.f, b.st
	b.row = 6
	sectionFirst := b.row

	firstPull := b.row
	for _, blk := range b.reg.receitas.Blocks {
		f.SetCellStyle(listasName, cell("B", b.row), cell("B", b.row), st.IndigoBand)
		f.SetCellValue(listasName, cell("C", b.row), blk.Label)
		tr := blk.TotalRow
		b.monthFormulas(b.row, st.PullCur, func(k int) string {
			return sheetRef("Receitas", expenseValorCol(k), tr)
		})
		b.row++
	}
	lastPull := b.row - 1

	b.bandRow(b.row) // internal separator
	b.row++

	// Total row (C0C0C0): C=lbl.Total; D..O = SUM of pulls. B/C labels General, D..O currency.
	f.SetCellStyle(listasName, cell("B", b.row), cell("C", b.row), st.ListasTotalLbl)
	f.SetCellValue(listasName, cell("C", b.row), b.lbl.Total)
	b.monthFormulas(b.row, st.ListasTotalCur, func(k int) string {
		c := listasMonthCol(k)
		return fmt.Sprintf("SUM(%s:%s)", cell(c, firstPull), cell(c, lastPull))
	})
	b.receitasTotalRow = b.row
	b.row += 3 // total, then two blank rows (10, 11)

	// Investimentos shell row.
	f.SetCellStyle(listasName, cell("B", b.row), cell("B", b.row), st.IndigoBand)
	f.SetCellValue(listasName, cell("C", b.row), b.lbl.Investments)
	for k := 0; k < 12; k++ {
		c := listasMonthCol(k)
		f.SetCellStyle(listasName, cell(c, b.row), cell(c, b.row), st.PullCur)
	}
	investRow := b.row
	b.row++

	// Investimentos total.
	f.SetCellStyle(listasName, cell("B", b.row), cell("C", b.row), st.ListasTotalLbl)
	f.SetCellValue(listasName, cell("C", b.row), b.lbl.Total)
	b.monthFormulas(b.row, st.ListasTotalCur, func(k int) string {
		return cell(listasMonthCol(k), investRow)
	})
	b.investTotalRow = b.row
	b.row++

	// % sobre Receita.
	f.SetCellStyle(listasName, cell("B", b.row), cell("C", b.row), st.ListasTotalLbl)
	f.SetCellValue(listasName, cell("C", b.row), b.lbl.PctOfRevenue)
	invTot, recTot := b.investTotalRow, b.receitasTotalRow
	b.monthFormulas(b.row, st.GroupTotalPct, func(k int) string {
		c := listasMonthCol(k)
		return fmt.Sprintf("IF(%s>0,%s/%s,0)", cell(c, recTot), cell(c, invTot), cell(c, recTot))
	})
	sectionLast := b.row

	mergeSection(f, st, "Receitas", sectionFirst, sectionLast)
	b.row += 4 // % row consumed at sectionLast; skip blanks to row 18
}

func mergeSection(f *excelize.File, st *styleSet, label string, first, last int) {
	f.MergeCell(listasName, cell("A", first), cell("A", last))
	f.SetCellValue(listasName, cell("A", first), label)
	f.SetCellStyle(listasName, cell("A", first), cell("A", last), st.SectionLabel)
}

// despesasSections: one band per source sheet (Fixas, Variáveis, Extras, Adicionais).
func (b *listasBuilder) despesasSections() {
	order := []string{"Fixas", "Variáveis", "Extras", "Adicionais"}
	for _, sName := range order {
		b.despesaSection(sName)
	}
}

func (b *listasBuilder) despesaSection(sName string) {
	f, st := b.f, b.st
	layout := b.reg.expense[sName]
	if layout == nil {
		return
	}
	sectionFirst := b.row
	plannedGrand := plannedGrandTotalRow(sectionFirst, layout.Cats)
	var groupTotalRows []int

	for _, ct := range layout.Cats {
		firstPull := b.row
		for i, sub := range ct.Subs {
			if i == 0 {
				f.SetCellStyle(listasName, cell("B", b.row), cell("B", b.row), st.IndigoBand)
			}
			f.SetCellValue(listasName, cell("C", b.row), sub.Subcat)
			tr := sub.TotalRow
			b.monthFormulas(b.row, st.PullCur, func(k int) string {
				return sheetRef(sName, expenseValorCol(k), tr)
			})
			b.row++
		}
		lastPull := b.row - 1
		// merge col B across the categoria's pull rows (skip degenerate single-row merge).
		if lastPull > firstPull {
			f.MergeCell(listasName, cell("B", firstPull), cell("B", lastPull))
		}
		f.SetCellValue(listasName, cell("B", firstPull), ct.Categoria)
		f.SetCellStyle(listasName, cell("B", firstPull), cell("B", lastPull), st.IndigoLabel)

		// group total (CCCCFF): B = "Total <cat>"; D..O = SUM of pulls. B/C General, D..O cur.
		f.SetCellStyle(listasName, cell("B", b.row), cell("C", b.row), st.GroupTotalLbl)
		f.SetCellValue(listasName, cell("B", b.row), fmt.Sprintf(b.lbl.TotalCategoryFmt, ct.Categoria))
		b.monthFormulas(b.row, st.GroupTotalCur, func(k int) string {
			c := listasMonthCol(k)
			return sumRange(cell(c, firstPull), cell(c, lastPull))
		})
		groupTotalRows = append(groupTotalRows, b.row)
		b.row++

		if perGroupPctRows {
			b.emitGroupPctRows(b.row-1, plannedGrand)
		}
	}

	// sheet grand-total (C0C0C0): B = "Total despesas <sheet-lower>". B/C General, D..O cur.
	f.SetCellStyle(listasName, cell("B", b.row), cell("C", b.row), st.ListasTotalLbl)
	f.SetCellValue(listasName, cell("B", b.row), fmt.Sprintf(b.lbl.TotalSheetExpensesFmt, lower(sName)))
	grandRow := b.row
	b.monthFormulas(b.row, st.ListasTotalCur, func(k int) string {
		c := listasMonthCol(k)
		terms := make([]string, len(groupTotalRows))
		for i, gr := range groupTotalRows {
			terms[i] = cell(c, gr)
		}
		return sumList(terms)
	})
	b.sheetGrandRow[sName] = grandRow
	b.row++

	// internal separator band.
	b.bandRow(b.row)
	b.row++

	// % sobre Receita (CCCCFF pct on D..O; B..C C0C0C0 label).
	f.SetCellStyle(listasName, cell("B", b.row), cell("C", b.row), st.ListasTotalLbl)
	f.SetCellValue(listasName, cell("B", b.row), b.lbl.PctOfRevenue)
	recTot := b.receitasTotalRow
	b.monthFormulas(b.row, st.GroupTotalPct, func(k int) string {
		c := listasMonthCol(k)
		return fmt.Sprintf("IF(%s>0,%s/%s,0)", cell(c, recTot), cell(c, grandRow), cell(c, recTot))
	})
	sectionLast := b.row

	mergeSection(f, st, sName, sectionFirst, sectionLast)
	b.row += 2 // % row consumed; skip blank to next section
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
// (group / receitas total).
func (b *listasBuilder) emitGroupPctRows(grpRow, grandRow int) {
	b.groupPctRow(b.lbl.PctOfExpenses, grpRow, grandRow)
	b.groupPctRow(b.lbl.PctOfRevenue, grpRow, b.receitasTotalRow)
}

// groupPctRow writes one CCCCFF percent row: label in col B (B:C label style),
// D..O = IF(denom>0, group/denom, 0) per month.
func (b *listasBuilder) groupPctRow(label string, grpRow, denomRow int) {
	f, st := b.f, b.st
	f.SetCellStyle(listasName, cell("B", b.row), cell("C", b.row), st.GroupTotalLbl)
	f.SetCellValue(listasName, cell("B", b.row), label)
	b.monthFormulas(b.row, st.GroupTotalPct, func(k int) string {
		c := listasMonthCol(k)
		return fmt.Sprintf("IF(%s>0,%s/%s,0)", cell(c, denomRow), cell(c, grpRow), cell(c, denomRow))
	})
	b.row++
}

// saldoBlock: bottom aggregate block, labels in col A. Starts at the current row
// (dynamic — per-group percent rows push it down) behind a thin separator.
func (b *listasBuilder) saldoBlock() {
	b.saldoSepRow = b.row
	b.f.SetRowHeight(listasName, b.saldoSepRow, 3.75)
	b.row++
	order := []string{"Fixas", "Variáveis", "Extras", "Adicionais"}
	rt, st := b.reg, b.st

	receitaRow := b.saldoRow(b.lbl.Revenue, st.ListasTotalLbl, st.ListasTotalCur, func(k int) string {
		return cell(listasMonthCol(k), b.receitasTotalRow)
	})
	investRow := b.saldoRow(b.lbl.Investments, st.ListasTotalLbl, st.ListasTotalCur, func(k int) string {
		return cell(listasMonthCol(k), b.investTotalRow)
	})
	totalRendaRow := b.saldoRow(b.lbl.TotalIncome, st.NearBlack, st.NearBlackCur, func(k int) string {
		c := listasMonthCol(k)
		return fmt.Sprintf("SUM(%s:%s)", cell(c, receitaRow), cell(c, investRow))
	})
	_ = rt

	despRows := map[string]int{}
	for _, s := range order {
		gr := b.sheetGrandRow[s]
		despRows[s] = b.saldoRow(fmt.Sprintf(b.lbl.SheetExpensesFmt, lower(s)), st.ListasTotalLbl, st.ListasTotalCur, func(k int) string {
			return cell(listasMonthCol(k), gr)
		})
	}
	totalDespRow := b.saldoRow(b.lbl.TotalExpenses, st.NearBlack, st.NearBlackCur, func(k int) string {
		c := listasMonthCol(k)
		terms := make([]string, len(order))
		for i, s := range order {
			terms[i] = cell(c, despRows[s])
		}
		return sumList(terms)
	})

	b.saldoLabelRow(b.lbl.ExpenseShareHeader)
	for _, s := range order {
		dr := despRows[s]
		b.saldoRowPct(s, func(k int) string {
			c := listasMonthCol(k)
			return fmt.Sprintf("IF(%s>0,%s/%s,0)", cell(c, totalDespRow), cell(c, dr), cell(c, totalDespRow))
		})
	}
	b.saldoLabelRow(b.lbl.IncomeShareHeader)
	for _, s := range order {
		dr := despRows[s]
		b.saldoRowPct(s, func(k int) string {
			c := listasMonthCol(k)
			return fmt.Sprintf("IF(%s>0,%s/%s,0)", cell(c, totalRendaRow), cell(c, dr), cell(c, totalRendaRow))
		})
	}
	b.saldoRow(b.lbl.Balance, st.NearBlack, st.NearBlackCur, func(k int) string {
		c := listasMonthCol(k)
		return fmt.Sprintf("%s-%s", cell(c, totalRendaRow), cell(c, totalDespRow))
	})
}

// saldoRow writes a label (col A) + month formulas. A..C carry the fill-only label style
// (General numFmt); D..O carry valStyle. Returns the row number.
func (b *listasBuilder) saldoRow(label string, lblStyle, valStyle int, fn func(k int) string) int {
	r := b.row
	b.f.SetCellValue(listasName, cell("A", r), label)
	b.f.SetCellStyle(listasName, cell("A", r), cell("C", r), lblStyle)
	b.monthFormulas(r, valStyle, fn)
	b.row++
	return r
}

// saldoRowPct: label A (C0C0C0), D..O percent (CCCCFF).
func (b *listasBuilder) saldoRowPct(label string, fn func(k int) string) int {
	r := b.row
	b.f.SetCellValue(listasName, cell("A", r), label)
	b.f.SetCellStyle(listasName, cell("A", r), cell("C", r), b.st.ListasTotalLbl)
	b.monthFormulas(r, b.st.GroupTotalPct, fn)
	b.row++
	return r
}

// saldoLabelRow: a full-width 333333 label row (col A label, rest near-black).
func (b *listasBuilder) saldoLabelRow(label string) {
	r := b.row
	b.f.SetCellValue(listasName, cell("A", r), label)
	b.f.SetCellStyle(listasName, cell("A", r), cell(lastListasCol, r), b.st.NearBlack)
	b.row++
}
