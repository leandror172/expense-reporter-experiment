package generate

import (
	"time"

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
			b := blocks[i]
			firstData, lastData, totalRow := calculateRevenueBlockRows(row, b)
			writeRevenueBlock(f, st, lbl, name, b, firstData, lastData, totalRow)
			reg.revenue.Blocks = append(reg.revenue.Blocks, revenueBlockTotal{
				Category: b.Category, Label: b.Label, TotalRow: totalRow,
			})
			row = totalRow + 1
			i++
		}
		catLast := row - 1
		mergeCategoryLabel(f, st, name, cat, catFirst, catLast)
		// separator only between income categories
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

// calculateRevenueBlockRows calculates the row range for an income block.
func calculateRevenueBlockRows(row int, b RevenueBlock) (firstData, lastData, totalRow int) {
	dataRows := b.MaxEntries()
	if dataRows == 0 {
		dataRows = 1
	}
	dataRows += headroomRows
	firstData = row
	lastData = firstData + dataRows - 1
	totalRow = lastData + 1
	return firstData, lastData, totalRow
}

// writeRevenueBlock writes one income block: its data rows (styled with date/currency formats),
// its merged col-B label across data+total rows, and its total row.
func writeRevenueBlock(f *excelize.File, st *styleSet, lbl Labels, name string, b RevenueBlock, firstData, lastData, totalRow int) {
	styleRevenueDataBand(f, st, name, firstData, lastData)
	fillRevenueEntries(f, name, b, firstData)
	writeTotalRowOpt(f, st, lbl, name, firstData, lastData, totalRow, false)
	f.SetRowHeight(name, totalRow, 15.75)
	f.MergeCell(name, cell("B", firstData), cell("B", totalRow))
	f.SetCellValue(name, cell("B", firstData), b.Label)
	f.SetCellStyle(name, cell("B", firstData), cell("B", totalRow), st.SubcatLabel)
}

// styleRevenueDataBand sets row heights and cell styles for the data row band.
func styleRevenueDataBand(f *excelize.File, st *styleSet, name string, firstData, lastData int) {
	for r := firstData; r <= lastData; r++ {
		f.SetRowHeight(name, r, 15)
		style := st.DataCellArial
		if r == firstData {
			style = st.DataCellTop
		}
		// Apply date and currency formats to data rows (not General as before).
		f.SetCellStyle(name, cell("C", r), cell(lastRevenueCol, r), style)
		for k := 0; k < 12; k++ {
			_, dataCol, valorCol := expenseMonthCols(k)
			f.SetCellStyle(name, cell(dataCol, r), cell(dataCol, r), st.DateCell)
			f.SetCellStyle(name, cell(valorCol, r), cell(valorCol, r), st.Currency)
		}
	}
}

// fillRevenueEntries writes item, date, and value cells for each typed entry.
func fillRevenueEntries(f *excelize.File, name string, b RevenueBlock, firstData int) {
	for k := 0; k < 12; k++ {
		itemCol, dataCol, valorCol := expenseMonthCols(k)
		for i, entry := range b.Months[k] {
			row := firstData + i
			f.SetCellValue(name, cell(itemCol, row), entry.Item)
			dateValue := time.Date(dataYear, time.Month(k+1), entry.Day, 0, 0, 0, 0, time.UTC)
			f.SetCellValue(name, cell(dataCol, row), dateValue)
			f.SetCellValue(name, cell(valorCol, row), entry.Value)
		}
	}
}
