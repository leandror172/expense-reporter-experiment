// workbook-inspect: dumps the structural map of an expense workbook to markdown.
// Usage: go run ./cmd/workbook-inspect <workbook.xlsx> [output.md]
// If output path is omitted, prints to stdout.
package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/xuri/excelize/v2"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: workbook-inspect <workbook.xlsx> [output.md]")
		os.Exit(1)
	}
	workbookPath := os.Args[1]

	out := os.Stdout
	if len(os.Args) >= 3 {
		f, err := os.Create(os.Args[2])
		if err != nil {
			fmt.Fprintf(os.Stderr, "cannot create output file: %v\n", err)
			os.Exit(1)
		}
		defer f.Close()
		out = f
	}

	wb, err := excelize.OpenFile(workbookPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "cannot open workbook: %v\n", err)
		os.Exit(1)
	}
	defer wb.Close()

	sheets := wb.GetSheetList()
	fmt.Fprintf(out, "# Workbook Structure Map\n\n")
	fmt.Fprintf(out, "Source: `%s`\n\n", workbookPath)
	fmt.Fprintf(out, "## Sheet Inventory\n\n")
	for i, s := range sheets {
		fmt.Fprintf(out, "%d. `%s`\n", i+1, s)
	}
	fmt.Fprintln(out)

	for _, sheet := range sheets {
		if sheet == "Referência de Categorias" {
			inspectReferenceSheet(out, wb, sheet)
		} else {
			inspectDataSheet(out, wb, sheet)
		}
	}
}

// ── Data sheets (Adicionais, Fixas, …) ─────────────────────────────────────

func inspectDataSheet(out *os.File, wb *excelize.File, sheet string) {
	rows, err := wb.GetRows(sheet)
	if err != nil {
		fmt.Fprintf(out, "## Sheet: `%s`\n\nERROR reading rows: %v\n\n", sheet, err)
		return
	}
	if len(rows) == 0 {
		fmt.Fprintf(out, "## Sheet: `%s`\n\n(empty)\n\n", sheet)
		return
	}

	fmt.Fprintf(out, "## Sheet: `%s`\n\n", sheet)
	fmt.Fprintf(out, "Rows: %d\n\n", len(rows))

	// Header rows: dump first 5 rows verbatim (trimmed)
	fmt.Fprintf(out, "### Header rows (first 5)\n\n")
	fmt.Fprintf(out, "| Row | A | B | C | D | E | F | G | H |\n")
	fmt.Fprintf(out, "|-----|---|---|---|---|---|---|---|---|\n")
	for i := 0; i < min(5, len(rows)); i++ {
		row := rows[i]
		fmt.Fprintf(out, "| %d | %s | %s | %s | %s | %s | %s | %s | %s |\n",
			i+1,
			cell(row, 0), cell(row, 1), cell(row, 2), cell(row, 3),
			cell(row, 4), cell(row, 5), cell(row, 6), cell(row, 7),
		)
	}
	fmt.Fprintln(out)

	// Month header row: scan rows 1-8 for a row containing month names
	fmt.Fprintf(out, "### Month column detection\n\n")
	months := []string{"jan", "fev", "mar", "abr", "mai", "jun", "jul", "ago", "set", "out", "nov", "dez"}
	for i := 0; i < min(8, len(rows)); i++ {
		row := rows[i]
		rowText := strings.ToLower(strings.Join(row, " "))
		hits := 0
		for _, m := range months {
			if strings.Contains(rowText, m) {
				hits++
			}
		}
		if hits >= 3 {
			fmt.Fprintf(out, "Month header at row %d (%d month names found):\n\n", i+1, hits)
			fmt.Fprintf(out, "```\n")
			for ci, val := range row {
				if val != "" {
					colName, _ := excelize.ColumnNumberToName(ci + 1)
					fmt.Fprintf(out, "  col %-4s = %s\n", colName, val)
				}
			}
			fmt.Fprintf(out, "```\n\n")
		}
	}

	// Subcategory blocks: scan column B for non-empty values
	fmt.Fprintf(out, "### Subcategory blocks (column B)\n\n")
	fmt.Fprintf(out, "| Row | Name | Total row | Formula sample |\n")
	fmt.Fprintf(out, "|-----|------|-----------|----------------|\n")

	type block struct {
		name     string
		headerRow int
		totalRow  int
		formula  string
	}
	var blocks []block

	for i, row := range rows {
		bVal := strings.TrimSpace(cell(row, 1))
		if bVal == "" {
			continue
		}
		// Skip rows that look like sheet-level headers (row <= 5 or all-caps short label)
		if i < 5 {
			continue
		}
		b := block{name: bVal, headerRow: i + 1}
		blocks = append(blocks, b)
	}

	// For each block, find its TOTAL row (scan forward for "total" in any cell)
	for bi := range blocks {
		start := blocks[bi].headerRow
		end := len(rows)
		if bi+1 < len(blocks) {
			end = blocks[bi+1].headerRow - 1
		}
		for r := start; r < min(end+5, len(rows)); r++ {
			row := rows[r]
			for _, val := range row {
				if strings.Contains(strings.ToLower(val), "total") {
					blocks[bi].totalRow = r + 1
					// Grab a formula from this row (first non-empty formula cell)
					for ci := range row {
						colName, _ := excelize.ColumnNumberToName(ci + 1)
						ref := fmt.Sprintf("%s%d", colName, r+1)
						formula, _ := wb.GetCellFormula(sheet, ref)
						if formula != "" {
							blocks[bi].formula = fmt.Sprintf("`%s`: `=%s`", ref, formula)
							break
						}
					}
					break
				}
			}
			if blocks[bi].totalRow > 0 {
				break
			}
		}
		fmt.Fprintf(out, "| %d | %s | %d | %s |\n",
			blocks[bi].headerRow, blocks[bi].name, blocks[bi].totalRow, blocks[bi].formula)
	}
	fmt.Fprintln(out)

	// Merged cells
	merges, err := wb.GetMergeCells(sheet)
	if err == nil && len(merges) > 0 {
		fmt.Fprintf(out, "### Merged cells (%d regions)\n\n", len(merges))
		fmt.Fprintf(out, "| Range | Value |\n|-------|-------|\n")
		for _, m := range merges {
			fmt.Fprintf(out, "| %s | %s |\n", m.GetStartAxis()+":"+m.GetEndAxis(), truncate(m.GetCellValue(), 40))
		}
		fmt.Fprintln(out)
	}

	// Formula inventory: sample one formula per column in a TOTAL row
	if len(blocks) > 0 && blocks[0].totalRow > 0 {
		r := blocks[0].totalRow - 1
		if r < len(rows) {
			fmt.Fprintf(out, "### Formula samples (from first TOTAL row, row %d)\n\n", r+1)
			fmt.Fprintf(out, "```\n")
			for ci := range rows[r] {
				colName, _ := excelize.ColumnNumberToName(ci + 1)
				ref := fmt.Sprintf("%s%d", colName, r+1)
				formula, _ := wb.GetCellFormula(sheet, ref)
				if formula != "" {
					fmt.Fprintf(out, "  %s = %s\n", ref, formula)
				}
			}
			fmt.Fprintf(out, "```\n\n")
		}
	}
}

// ── Reference sheet ─────────────────────────────────────────────────────────

func inspectReferenceSheet(out *os.File, wb *excelize.File, sheet string) {
	rows, err := wb.GetRows(sheet)
	if err != nil {
		fmt.Fprintf(out, "## Sheet: `%s`\n\nERROR: %v\n\n", sheet, err)
		return
	}
	fmt.Fprintf(out, "## Sheet: `%s`\n\n", sheet)
	fmt.Fprintf(out, "Rows: %d\n\n", len(rows))
	fmt.Fprintf(out, "### Full content\n\n")
	fmt.Fprintf(out, "| Row | A (Sheet) | B (Category) | C (Subcategory) | D (Row#) | E | F (Total row) |\n")
	fmt.Fprintf(out, "|-----|-----------|--------------|-----------------|----------|---|---------------|\n")
	for i, row := range rows {
		a := cell(row, 0)
		b := cell(row, 1)
		c := cell(row, 2)
		d := cell(row, 3)
		e := cell(row, 4)
		f := cell(row, 5)
		if a == "" && b == "" && c == "" {
			continue
		}
		fmt.Fprintf(out, "| %d | %s | %s | %s | %s | %s | %s |\n", i+1, a, b, c, d, e, f)
	}
	fmt.Fprintln(out)
}

// ── helpers ──────────────────────────────────────────────────────────────────

func cell(row []string, idx int) string {
	if idx >= len(row) {
		return ""
	}
	return strings.TrimSpace(row[idx])
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
