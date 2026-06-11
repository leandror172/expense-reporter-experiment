package main

import "github.com/xuri/excelize/v2"

const lastReceitasCol = "AL"

// buildReceitas writes the Receitas sheet and records block total rows into reg.
// v2: Receitas shares the data-sheet column model; income category in col A merged across
// its blocks, block label in col B merged across the block incl. its total row.
// Separators appear only between income categories (none within a category).
func buildReceitas(f *excelize.File, st *styleSet, lbl Labels, blocks []ReceitasBlock, reg *layoutRegistry) error {
	const name = "Receitas"
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
			firstData := row
			lastData := row + headroomRows - 1
			totalRow := lastData + 1
			writeReceitasBlock(f, st, lbl, name, b.Label, firstData, lastData, totalRow)
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

func writeReceitasBlock(f *excelize.File, st *styleSet, lbl Labels, name, label string, firstData, lastData, totalRow int) {
	for r := firstData; r <= lastData; r++ {
		f.SetRowHeight(name, r, 15)
		style := st.DataCellArial
		if r == firstData {
			style = st.DataCellTop
		}
		// Receitas data rows keep General numFmt (golden master) — no date/currency override.
		f.SetCellStyle(name, cell("C", r), cell(lastReceitasCol, r), style)
	}
	writeTotalRowOpt(f, st, lbl, name, firstData, lastData, totalRow, false)
	f.SetRowHeight(name, totalRow, 15.75)
	f.MergeCell(name, cell("B", firstData), cell("B", totalRow))
	f.SetCellValue(name, cell("B", firstData), label)
	f.SetCellStyle(name, cell("B", firstData), cell("B", totalRow), st.SubcatLabel)
}
