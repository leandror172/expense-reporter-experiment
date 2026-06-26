package generate

import (
	"expense-reporter/internal/taxonomy"
	"github.com/xuri/excelize/v2"
)

const lastRevenueCol = "AL"

// buildRevenueSheet writes the Receitas sheet and records block total rows into reg.
// v3: Receitas groups by Block field; col A merged across all sublines in a Block group,
//
//	col B merged per-subline. Separators only between Block groups.
func buildRevenueSheet(f *excelize.File, st *styleSet, lbl Labels, blocks []taxonomy.RevenueBlock, reg *layoutRegistry) error {
	name := lbl.RevenueSheet
	if _, err := f.NewSheet(name); err != nil {
		return err
	}
	setRevenueSheetWidths(f, name)
	writeMonthHeader(f, st, lbl, name)
	freezeC3(f, name)

	row := 3
	i := 0
	for i < len(blocks) {
		block := blocks[i].Block
		blockFirst := row
		for i < len(blocks) && blocks[i].Block == block {
			firstData, lastData, totalRow := calculateBlockRows(row, blocks[i].MaxEntries())
			writeRevenueBlock(f, st, lbl, name, blocks[i], firstData, lastData, totalRow)
			reg.revenue.Blocks = append(reg.revenue.Blocks, revenueBlockTotal{
				Category: blocks[i].Category,
				Block:    blocks[i].Block,
				Label:    blocks[i].Label,
				TotalRow: totalRow,
			})
			row = totalRow + 1
			i++
		}
		blockLast := row - 1
		mergeCategoryLabel(f, st, name, block, blockFirst, blockLast)
		if i < len(blocks) {
			row++ // blank
			writeSeparator(f, st, name, row, lastRevenueCol)
			row++ // separator consumed
			row++ // blank
		}
	}
	return nil
}

func setRevenueSheetWidths(f *excelize.File, name string) {
	f.SetColWidth(name, "A", "A", 12.29)
	f.SetColWidth(name, "B", "B", 13.86)
	f.SetColWidth(name, "C", lastRevenueCol, 12.57)
}

// writeRevenueBlock writes one income leaf: its data rows (styled with date/currency formats),
// its merged col-B label across data+total rows, and its total row.
func writeRevenueBlock(f *excelize.File, st *styleSet, lbl Labels, name string, b taxonomy.RevenueBlock, firstData, lastData, totalRow int) {
	writeDataBand(f, st, name, b.Months, firstData, lastData, 15, lastRevenueCol)
	writeTotalRowOpt(f, st, lbl, name, firstData, lastData, totalRow, false)
	f.SetRowHeight(name, totalRow, 15.75)
	f.MergeCell(name, cell("B", firstData), cell("B", totalRow))
	f.SetCellValue(name, cell("B", firstData), b.Label)
	f.SetCellStyle(name, cell("B", firstData), cell("B", totalRow), st.SubcatLabel)
}
