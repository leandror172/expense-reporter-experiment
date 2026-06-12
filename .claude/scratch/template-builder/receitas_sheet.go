package main

import (
	"time"

	"github.com/xuri/excelize/v2"
)

const lastReceitasCol = "AL"

// buildReceitas writes the Receitas sheet and records block total rows into reg.
// v2: Receitas shares the data-sheet column model; income category in col A merged across
// its blocks, block label in col B merged across the block incl. its total row.
// Separators appear only between income categories (none within a category).
func buildReceitas(f *excelize.File, st *styleSet, lbl Labels, blocks []ReceitasBlock, reg *layoutRegistry) error {
	name := lbl.RevenueSheet
	if _, err := f.NewSheet(name); err != nil {
		return err
	}
	setReceitasWidths(f, name)
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
			firstData, lastData, totalRow := calculateReceitasBlockRows(row, b)
			writeReceitasBlock(f, st, lbl, name, b, firstData, lastData, totalRow)
			reg.receitas.Blocks = append(reg.receitas.Blocks, receitasBlockTotal{
				Category: b.Category, Label: b.Label, TotalRow: totalRow,
			})
			row = totalRow + 1
			i++
		}
		catLast := row - 1
		mergeCategoriaLabel(f, st, name, cat, catFirst, catLast)
		// separator only between income categories
		if i < len(blocks) {
			row++ // blank
			writeSeparator(f, st, name, row, lastReceitasCol)
			row++ // separator consumed
			row++ // blank
		}
	}
	return nil
}

func setReceitasWidths(f *excelize.File, name string) {
	f.SetColWidth(name, "A", "A", 12.29)
	f.SetColWidth(name, "B", "B", 13.86)
	f.SetColWidth(name, "C", lastReceitasCol, 12.57)
}

// calculateReceitasBlockRows calculates the row range for an income block.
func calculateReceitasBlockRows(row int, b ReceitasBlock) (firstData, lastData, totalRow int) {
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

// writeReceitasBlock writes one income block: its data rows (styled with date/currency formats),
// its merged col-B label across data+total rows, and its total row.
func writeReceitasBlock(f *excelize.File, st *styleSet, lbl Labels, name string, b ReceitasBlock, firstData, lastData, totalRow int) {
	writeReceitasDataRows(f, st, name, b, firstData, lastData)
	writeTotalRowOpt(f, st, lbl, name, firstData, lastData, totalRow, false)
	f.SetRowHeight(name, totalRow, 15.75)
	f.MergeCell(name, cell("B", firstData), cell("B", totalRow))
	f.SetCellValue(name, cell("B", firstData), b.Label)
	f.SetCellStyle(name, cell("B", firstData), cell("B", totalRow), st.SubcatLabel)
}

// writeReceitasDataRows styles each data row and fills typed entries.
func writeReceitasDataRows(f *excelize.File, st *styleSet, name string, b ReceitasBlock, firstData, lastData int) {
	for r := firstData; r <= lastData; r++ {
		f.SetRowHeight(name, r, 15)
		style := st.DataCellArial
		if r == firstData {
			style = st.DataCellTop
		}
		// Apply date and currency formats to data rows (not General as before).
		f.SetCellStyle(name, cell("C", r), cell(lastReceitasCol, r), style)
		for k := 0; k < 12; k++ {
			_, dataCol, valorCol := expenseMonthCols(k)
			f.SetCellStyle(name, cell(dataCol, r), cell(dataCol, r), st.DateCell)
			f.SetCellStyle(name, cell(valorCol, r), cell(valorCol, r), st.Currency)
		}
	}

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
