package generate

import (
	"fmt"
	"time"

	"expense-reporter/internal/taxonomy"
	"github.com/xuri/excelize/v2"
)

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
	f.SetCellStyle(name, "A1", "A1", st.MonthCorner)
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
		itemStyle, valueStyle := st.TotalText, st.TotalValue
		if groupSeps && k%2 == 1 { // vertical separator every 2 month-groups (golden pattern)
			itemStyle, valueStyle = st.TotalTextLeft, st.TotalValueRight
		}
		f.SetCellStyle(name, cell(itemCol, totalRow), cell(itemCol, totalRow), itemStyle)
		f.SetCellStyle(name, cell(dataCol, totalRow), cell(dataCol, totalRow), st.TotalText)
		f.SetCellStyle(name, cell(valorCol, totalRow), cell(valorCol, totalRow), valueStyle)
	}
}

func writeSeparator(f *excelize.File, st *styleSet, name string, row int, lastCol string) {
	f.SetCellStyle(name, cell("A", row), cell(lastCol, row), st.Separator)
}

func mergeCategoryLabel(f *excelize.File, st *styleSet, name, label string, first, last int) {
	f.MergeCell(name, cell("A", first), cell("A", last))
	f.SetCellValue(name, cell("A", first), label)
	f.SetCellStyle(name, cell("A", first), cell("A", last), st.CategoryBold)
}

// calculateBlockRows calculates data row range for a subcategory or revenue block.
// maxEntries is the busiest month's entry count; 0 is treated as 1 (at least one data row).
func calculateBlockRows(row, maxEntries int) (firstData, lastData, totalRow int) {
	e := maxEntries
	if e == 0 {
		e = 1
	}
	firstData = row
	lastData = firstData + e + headroomRows - 1
	totalRow = lastData + 1
	return firstData, lastData, totalRow
}

// writeDataBand sets row heights and cell styles for a data-row band, then fills in
// the typed entries. rowHeight is 12.75 for expense sheets and 15 for revenue sheets.
func writeDataBand(f *excelize.File, st *styleSet, name string, months [12][]taxonomy.Entry,
	firstData, lastData int, rowHeight float64, lastCol string) {
	for r := firstData; r <= lastData; r++ {
		f.SetRowHeight(name, r, rowHeight)
		style := st.DataCellArial
		if r == firstData {
			style = st.DataCellTop
		}
		f.SetCellStyle(name, cell("C", r), cell(lastCol, r), style)
		for k := 0; k < 12; k++ {
			_, dataCol, valorCol := expenseMonthCols(k)
			f.SetCellStyle(name, cell(dataCol, r), cell(dataCol, r), st.DateCell)
			f.SetCellStyle(name, cell(valorCol, r), cell(valorCol, r), st.Currency)
		}
	}

	for k := 0; k < 12; k++ {
		itemCol, dataCol, valorCol := expenseMonthCols(k)
		for i, entry := range months[k] {
			row := firstData + i
			f.SetCellValue(name, cell(itemCol, row), entry.Item)
			dateValue := time.Date(dataYear, time.Month(k+1), entry.Day, 0, 0, 0, 0, time.UTC)
			f.SetCellValue(name, cell(dataCol, row), dateValue)
			f.SetCellValue(name, cell(valorCol, row), entry.Value)
		}
	}
}
