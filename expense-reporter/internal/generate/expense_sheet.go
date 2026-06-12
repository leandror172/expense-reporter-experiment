package generate

import (
	"fmt"
	"time"

	"github.com/xuri/excelize/v2"
)

const lastExpenseCol = "AL" // col 38 (index 37)

// buildExpenseSheet writes one expense sheet (Fixas/Variáveis/Extras/Adicionais) and
// records its subcategory total-row positions into reg.
func buildExpenseSheet(f *excelize.File, st *styleSet, lbl Labels, sh ExpenseSheet, reg *layoutRegistry) error {
	name := sh.Name
	if _, err := f.NewSheet(name); err != nil {
		return err
	}
	setDataSheetWidths(f, name)
	writeMonthHeader(f, st, lbl, name)
	freezeC3(f, name)

	layout := &sheetLayout{Sheet: name}
	row := 3
	for ci, cat := range sh.Cats {
		ct := catTotals{Category: cat.Name}
		catFirst := row
		for _, sub := range cat.Subs {
			firstData, lastData, totalRow := calculateSubcatBlockRows(row, sub)
			writeSubcatBlock(f, st, lbl, name, sub, firstData, lastData, totalRow)
			ct.Subs = append(ct.Subs, subcatTotal{
				Sheet: name, Category: cat.Name, Subcat: sub.Name, TotalRow: totalRow,
			})
			row = totalRow + 1
		}
		catLast := row - 1
		mergeCategoryLabel(f, st, name, cat.Name, catFirst, catLast)
		layout.Cats = append(layout.Cats, ct)
		if ci < len(sh.Cats)-1 {
			row++ // blank
			writeSeparator(f, st, name, row, lastExpenseCol)
			row++ // separator consumed
			row++ // blank
		}
	}
	reg.expense[name] = layout
	reg.sheetOrder = append(reg.sheetOrder, name)
	return nil
}

func setDataSheetWidths(f *excelize.File, name string) {
	f.SetColWidth(name, "A", "A", 20.57)
	f.SetColWidth(name, "B", "B", 15.71)
	for k := 0; k < 12; k++ {
		item, data, valor := expenseMonthCols(k)
		f.SetColWidth(name, item, item, 14.29)
		f.SetColWidth(name, data, data, 8.0)
		f.SetColWidth(name, valor, valor, 12.14)
	}
}

func freezeC3(f *excelize.File, name string) {
	f.SetPanes(name, &excelize.Panes{
		Freeze: true, XSplit: 2, YSplit: 2, TopLeftCell: "C3", ActivePane: "bottomRight",
	})
}

// writeMonthHeader writes row 1 (Mês corner + month banners) and row 2 (Item/Data/Valor).
func writeMonthHeader(f *excelize.File, st *styleSet, lbl Labels, name string) {
	f.SetRowHeight(name, 1, 18)
	f.SetRowHeight(name, 2, 15)
	f.MergeCell(name, "A1", "B2")
	f.SetCellValue(name, "A1", lbl.Month)
	f.SetCellStyle(name, "A1", "A1", st.MesCorner)
	for k := 0; k < 12; k++ {
		item, data, valor := expenseMonthCols(k)
		f.MergeCell(name, item+"1", valor+"1")
		f.SetCellValue(name, item+"1", lbl.MonthNames[k])
		f.SetCellStyle(name, item+"1", item+"1", st.MonthBanner)
		f.SetCellValue(name, item+"2", lbl.Item)
		f.SetCellValue(name, data+"2", lbl.Date)
		f.SetCellValue(name, valor+"2", lbl.Amount)
		f.SetCellStyle(name, item+"2", valor+"2", st.HeaderCol)
	}
}

// writeSubcatBlock writes one subcategory: its data rows (styled, numfmt-ready), its merged
// col-B label across data+total rows, and its total row.
func writeSubcatBlock(f *excelize.File, st *styleSet, lbl Labels, name string, sub Subcat, firstData, lastData, totalRow int) {
	writeSubcatDataRows(f, st, name, sub, firstData, lastData)
	writeTotalRow(f, st, lbl, name, firstData, lastData, totalRow)
	// col B merged across data rows + total row (incl. total per spec §2).
	f.MergeCell(name, cell("B", firstData), cell("B", totalRow))
	f.SetCellValue(name, cell("B", firstData), sub.Name)
	f.SetCellStyle(name, cell("B", firstData), cell("B", totalRow), st.SubcatLabel)
}

func writeSubcatDataRows(f *excelize.File, st *styleSet, name string, sub Subcat, firstData, lastData int) {
	for r := firstData; r <= lastData; r++ {
		f.SetRowHeight(name, r, 12.75)
		style := st.DataCellArial
		if r == firstData {
			style = st.DataCellTop
		}
		f.SetCellStyle(name, cell("C", r), cell(lastExpenseCol, r), style)
		for k := 0; k < 12; k++ {
			_, dataCol, valorCol := expenseMonthCols(k)
			f.SetCellStyle(name, cell(dataCol, r), cell(dataCol, r), st.DateCell)
			f.SetCellStyle(name, cell(valorCol, r), cell(valorCol, r), st.Currency)
		}
	}

	for k := 0; k < 12; k++ {
		itemCol, dataCol, valorCol := expenseMonthCols(k)
		for i, entry := range sub.Months[k] {
			row := firstData + i
			f.SetCellValue(name, cell(itemCol, row), entry.Item)
			dateValue := time.Date(dataYear, time.Month(k+1), entry.Day, 0, 0, 0, 0, time.UTC)
			f.SetCellValue(name, cell(dataCol, row), dateValue)
			f.SetCellValue(name, cell(valorCol, row), entry.Value)
		}
	}
}

func calculateSubcatBlockRows(row int, sub Subcat) (firstData, lastData, totalRow int) {
	maxEntries := sub.MaxEntries()
	if maxEntries == 0 {
		maxEntries = 1
	}
	firstData = row
	lastData = firstData + maxEntries + headroomRows - 1
	totalRow = lastData + 1
	return firstData, lastData, totalRow
}

func mergeCategoryLabel(f *excelize.File, st *styleSet, name, label string, first, last int) {
	f.MergeCell(name, cell("A", first), cell("A", last))
	f.SetCellValue(name, cell("A", first), label)
	f.SetCellStyle(name, cell("A", first), cell("A", last), st.CategoryBold)
}

// writeTotalRow writes the total row: Total/TotalDash/SUM per month triple.
// groupSeps adds the every-other-month vertical separator borders (data sheets, not Receitas).
func writeTotalRow(f *excelize.File, st *styleSet, lbl Labels, name string, firstData, lastData, totalRow int) {
	writeTotalRowOpt(f, st, lbl, name, firstData, lastData, totalRow, true)
}

func writeTotalRowOpt(f *excelize.File, st *styleSet, lbl Labels, name string, firstData, lastData, totalRow int, groupSeps bool) {
	f.SetRowHeight(name, totalRow, 12.75)
	for k := 0; k < 12; k++ {
		itemCol, dataCol, valorCol := expenseMonthCols(k)
		f.SetCellValue(name, cell(itemCol, totalRow), lbl.Total)
		f.SetCellValue(name, cell(dataCol, totalRow), lbl.TotalDash)
		formula := fmt.Sprintf("SUM(%s:%s)", cell(valorCol, firstData), cell(valorCol, lastData))
		f.SetCellFormula(name, cell(valorCol, totalRow), formula)
		itemStyle, valorStyle := st.TotalData, st.TotalValor
		if groupSeps && k%2 == 1 { // vertical separator every 2 month-groups (golden pattern)
			itemStyle, valorStyle = st.TotalDataL, st.TotalValorR
		}
		f.SetCellStyle(name, cell(itemCol, totalRow), cell(itemCol, totalRow), itemStyle)
		f.SetCellStyle(name, cell(dataCol, totalRow), cell(dataCol, totalRow), st.TotalData)
		f.SetCellStyle(name, cell(valorCol, totalRow), cell(valorCol, totalRow), valorStyle)
	}
}

func writeSeparator(f *excelize.File, st *styleSet, name string, row int, lastCol string) {
	f.SetCellStyle(name, cell("A", row), cell(lastCol, row), st.Separator)
}

func cell(col string, row int) string { return fmt.Sprintf("%s%d", col, row) }
