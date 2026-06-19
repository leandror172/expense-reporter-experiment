package generate

import (
	"expense-reporter/internal/taxonomy"
	"github.com/xuri/excelize/v2"
)

const lastExpenseCol = "AL" // col 38 (index 37)

// buildExpenseType writes one expense sheet (Fixas/Variáveis/Extras/Adicionais) and
// records its subcategory total-row positions into reg.
func buildExpenseType(f *excelize.File, st *styleSet, lbl Labels, sh taxonomy.ExpenseType, reg *layoutRegistry) error {
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
			firstData, lastData, totalRow := calculateBlockRows(row, sub.MaxEntries())
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

// writeSubcatBlock writes one subcategory: its data rows (styled, numfmt-ready), its merged
// col-B label across data+total rows, and its total row.
func writeSubcatBlock(f *excelize.File, st *styleSet, lbl Labels, name string, sub taxonomy.Subcat, firstData, lastData, totalRow int) {
	writeDataBand(f, st, name, sub.Months, firstData, lastData, 12.75, lastExpenseCol)
	writeTotalRow(f, st, lbl, name, firstData, lastData, totalRow)
	// col B merged across data rows + total row (incl. total per spec §2).
	f.MergeCell(name, cell("B", firstData), cell("B", totalRow))
	f.SetCellValue(name, cell("B", firstData), sub.Name)
	f.SetCellStyle(name, cell("B", firstData), cell("B", totalRow), st.SubcatLabel)
}


