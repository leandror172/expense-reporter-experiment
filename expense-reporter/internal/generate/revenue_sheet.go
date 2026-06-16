package generate

import (
	"github.com/xuri/excelize/v2"
)

const lastRevenueCol = "AL"

// buildRevenueSheet writes the Receitas sheet and records block total rows into reg.
// v2: Receitas shares the data-sheet column model; income category in col A merged across
// its blocks, block label in col B merged across the block incl. its total row.
// Separators appear only between income categories (none within a category).
func buildRevenueSheet(f *excelize.File, st *styleSet, lbl Labels, blocks []RevenueBlock, reg *layoutRegistry) error {
	name := lbl.RevenueSheet
	if _, err := f.NewSheet(name); err != nil {
		return err
	}
	setRevenueSheetWidths(f, name)
	writeMonthHeader(f, st, lbl, name)
	freezeC3(f, name)

	// Group consecutive blocks by income category.
	row := 3
	i := 0
	for i < len(blocks) {
		cat := blocks[i].Category
		catFirst := row
		for i < len(blocks) && blocks[i].Category == cat {
			row = writeRevenueBlockRow(blocks, i, row, f, st, lbl, name, reg)
			i++
		}
		catLast := row - 1
		mergeCategoryLabel(f, st, name, cat, catFirst, catLast)
		// separator only between income categories
		writeCategorySeparator(i, blocks, row, f, st, name)
	}
	return nil
}

func writeCategorySeparator(i int, blocks []RevenueBlock, row int, f *excelize.File, st *styleSet, name string) {
	if i < len(blocks) {
		row++ // blank
		writeSeparator(f, st, name, row, lastRevenueCol)
		row++ // separator consumed
		row++ // blank
	}
}

func writeRevenueBlockRow(blocks []RevenueBlock, i int, row int, f *excelize.File, st *styleSet, lbl Labels, name string, reg *layoutRegistry) int {
	b := blocks[i]
	firstData, lastData, totalRow := calculateBlockRows(row, b.MaxEntries())
	writeRevenueBlock(f, st, lbl, name, b, firstData, lastData, totalRow)
	reg.revenue.Blocks = append(reg.revenue.Blocks, revenueBlockTotal{
		Category: b.Category, Label: b.Label, TotalRow: totalRow,
	})
	row = totalRow + 1
	return row
}

func setRevenueSheetWidths(f *excelize.File, name string) {
	f.SetColWidth(name, "A", "A", 12.29)
	f.SetColWidth(name, "B", "B", 13.86)
	f.SetColWidth(name, "C", lastRevenueCol, 12.57)
}

// writeRevenueBlock writes one income block: its data rows (styled with date/currency formats),
// its merged col-B label across data+total rows, and its total row.
func writeRevenueBlock(f *excelize.File, st *styleSet, lbl Labels, name string, b RevenueBlock, firstData, lastData, totalRow int) {
	writeDataBand(f, st, name, b.Months, firstData, lastData, 15, lastRevenueCol)
	writeTotalRowOpt(f, st, lbl, name, firstData, lastData, totalRow, false)
	f.SetRowHeight(name, totalRow, 15.75)
	f.MergeCell(name, cell("B", firstData), cell("B", totalRow))
	f.SetCellValue(name, cell("B", firstData), b.Label)
	f.SetCellStyle(name, cell("B", firstData), cell("B", totalRow), st.SubcatLabel)
}
