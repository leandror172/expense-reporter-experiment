// workbook-inspect: dumps the full structural map of an expense workbook to JSON.
// Usage: workbook-inspect <workbook.xlsx> <output-dir>
// Produces <output-dir>/manifest.json plus one <SheetName>.json per sheet.
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/xuri/excelize/v2"
)

func main() {
	if len(os.Args) != 3 {
		fmt.Fprintln(os.Stderr, "usage: workbook-inspect <workbook.xlsx> <output-dir>")
		os.Exit(1)
	}

	workbookPath := os.Args[1]
	outputDir := os.Args[2]

	if _, err := os.Stat(workbookPath); os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "workbook does not exist: %s\n", workbookPath)
		os.Exit(1)
	}

	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		fmt.Fprintf(os.Stderr, "cannot create output directory: %v\n", err)
		os.Exit(1)
	}

	wb, err := excelize.OpenFile(workbookPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "cannot open workbook: %v\n", err)
		os.Exit(1)
	}
	defer wb.Close()

	if err := dumpWorkbook(wb, workbookPath, outputDir); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}

func dumpWorkbook(wb *excelize.File, workbookPath, outputDir string) error {
	sheets := wb.GetSheetList()
	manifest := Manifest{Source: workbookPath, Sheets: make([]SheetInfo, 0, len(sheets))}

	for _, sheetName := range sheets {
		dump, err := buildSheetDump(wb, sheetName)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error building dump for sheet %s: %v\n", sheetName, err)
			continue
		}

		filename := sanitizeFilename(sheetName) + ".json"
		if err := writeJSON(filepath.Join(outputDir, filename), dump); err != nil {
			fmt.Fprintf(os.Stderr, "error writing file %s: %v\n", filename, err)
			continue
		}

		manifest.Sheets = append(manifest.Sheets, SheetInfo{
			Name: sheetName,
			File: filename,
			Rows: dump.Dimensions.Rows,
			Cols: dump.Dimensions.Cols,
		})
	}

	if err := writeJSON(filepath.Join(outputDir, "manifest.json"), manifest); err != nil {
		return fmt.Errorf("write manifest: %w", err)
	}
	return nil
}

func buildSheetDump(wb *excelize.File, sheetName string) (*SheetDump, error) {
	rows, err := wb.GetRows(sheetName)
	if err != nil {
		return nil, fmt.Errorf("get rows: %w", err)
	}

	dim := sheetDimensions(rows)
	dump := &SheetDump{
		Sheet:          sheetName,
		Dimensions:     dim,
		ColumnWidths:   collectColumnWidths(wb, sheetName, dim.Cols),
		RowHeights:     collectRowHeights(wb, sheetName, dim.Rows),
		MergedCells:    collectMergedCells(wb, sheetName),
		CrossSheetRefs: []string{},
		Rows:           []RowDump{},
	}

	dump.Rows = collectRowDumps(wb, sheetName, rows)
	dump.CrossSheetRefs = collectCrossSheetRefs(dump.Rows, sheetName)
	classifyRowTypes(dump.Rows)
	return dump, nil
}

func sheetDimensions(rows [][]string) Dim {
	cols := 0
	for _, row := range rows {
		if len(row) > cols {
			cols = len(row)
		}
	}
	return Dim{Rows: len(rows), Cols: cols}
}

func collectColumnWidths(wb *excelize.File, sheetName string, cols int) map[string]float64 {
	widths := make(map[string]float64)
	for ci := 0; ci < cols; ci++ {
		colName, _ := excelize.ColumnNumberToName(ci + 1)
		if w, err := wb.GetColWidth(sheetName, colName); err == nil {
			widths[colName] = w
		}
	}
	return widths
}

func collectRowHeights(wb *excelize.File, sheetName string, rowCount int) map[string]float64 {
	heights := make(map[string]float64)
	for r := 1; r <= rowCount; r++ {
		if h, err := wb.GetRowHeight(sheetName, r); err == nil {
			heights[strconv.Itoa(r)] = h
		}
	}
	return heights
}

func collectMergedCells(wb *excelize.File, sheetName string) []Merge {
	merges, err := wb.GetMergeCells(sheetName)
	if err != nil {
		return []Merge{}
	}
	result := make([]Merge, 0, len(merges))
	for _, m := range merges {
		result = append(result, Merge{
			Range: m.GetStartAxis() + ":" + m.GetEndAxis(),
			Value: m.GetCellValue(),
		})
	}
	return result
}

func collectRowDumps(wb *excelize.File, sheetName string, rows [][]string) []RowDump {
	dumps := []RowDump{}
	for ri, row := range rows {
		rd := buildRowDump(wb, sheetName, ri, row)
		if len(rd.Cells) > 0 {
			dumps = append(dumps, rd)
			continue
		}
		// Resurrect otherwise-empty rows that carry a row-level fill — these are
		// the black separator bands GetRows yields as empty slices and the
		// cell-level dump would drop. GetCellStyle on an empty cell falls back to
		// the row's style, so an empty probe cell reveals the row fill.
		if fill := probeRowFill(wb, sheetName, ri+1); fill != "" {
			dumps = append(dumps, RowDump{Row: ri + 1, RowType: "separator", RowFill: fill})
		}
	}
	return dumps
}

// probeRowFill returns the row-level fill color for a row by reading the style
// of an empty cell (col A), which resolves to the row style when the cell has
// none of its own. Empty string means no row-level fill.
func probeRowFill(wb *excelize.File, sheetName string, rowNum int) string {
	ref, err := excelize.CoordinatesToCellName(1, rowNum)
	if err != nil {
		return ""
	}
	return extractCellStyle(wb, sheetName, ref).BgColor
}

func buildRowDump(wb *excelize.File, sheetName string, ri int, row []string) RowDump {
	rd := RowDump{Row: ri + 1, Cells: []Cell{}}
	for ci, value := range row {
		ref, err := excelize.CoordinatesToCellName(ci+1, ri+1)
		if err != nil {
			continue
		}
		if cell, include := buildCell(wb, sheetName, ref, ci, value); include {
			rd.Cells = append(rd.Cells, cell)
		}
	}
	return rd
}

func buildCell(wb *excelize.File, sheetName, ref string, ci int, value string) (Cell, bool) {
	formula, _ := wb.GetCellFormula(sheetName, ref)
	style := extractCellStyle(wb, sheetName, ref)
	if value == "" && formula == "" && isDefaultStyle(style) {
		return Cell{}, false
	}
	colName, _ := excelize.ColumnNumberToName(ci + 1)
	return Cell{
		Col:     colName,
		Value:   strings.TrimSpace(value),
		Formula: formula,
		Style:   style,
	}, true
}

func extractCellStyle(wb *excelize.File, sheetName, ref string) Style {
	styleIdx, err := wb.GetCellStyle(sheetName, ref)
	if err != nil || styleIdx == 0 {
		return Style{}
	}
	st, err := wb.GetStyle(styleIdx)
	if err != nil {
		return Style{}
	}
	return styleFromExcelize(st)
}

func styleFromExcelize(st *excelize.Style) Style {
	style := Style{Bold: st.Font != nil && st.Font.Bold}
	if st.Fill.Pattern > 0 && len(st.Fill.Color) > 0 {
		style.BgColor = st.Fill.Color[0]
	}
	for _, b := range st.Border {
		if b.Style <= 0 {
			continue
		}
		switch b.Type {
		case "top":
			style.BorderTop = true
		case "bottom":
			style.BorderBottom = true
		case "left":
			style.BorderLeft = true
		case "right":
			style.BorderRight = true
		}
	}
	return style
}

func isDefaultStyle(s Style) bool {
	return s.BgColor == "" && !s.Bold &&
		!s.BorderTop && !s.BorderBottom && !s.BorderLeft && !s.BorderRight
}

// collectCrossSheetRefs scans formulas captured in the dumped rows for
// references to other sheets (SheetName!Ref, sheet name optionally single-quoted),
// returning the distinct, sorted set of referenced sheet names excluding self.
func collectCrossSheetRefs(rows []RowDump, selfSheet string) []string {
	seen := map[string]bool{}
	for _, rd := range rows {
		for _, c := range rd.Cells {
			if c.Formula == "" {
				continue
			}
			for _, name := range sheetNamesInFormula(c.Formula) {
				if name != "" && name != selfSheet {
					seen[name] = true
				}
			}
		}
	}
	result := make([]string, 0, len(seen))
	for name := range seen {
		result = append(result, name)
	}
	sort.Strings(result)
	return result
}

// sheetNamesInFormula extracts every sheet name appearing before a '!' in a
// formula. Handles both 'Quoted Name'!A1 and BareName!A1 forms.
func sheetNamesInFormula(formula string) []string {
	var names []string
	for i := 0; i < len(formula); i++ {
		if formula[i] != '!' {
			continue
		}
		if name := sheetNameBefore(formula, i); name != "" {
			names = append(names, name)
		}
	}
	return names
}

// sheetNameBefore returns the sheet-name token immediately preceding the '!'
// at index bang. A quoted token ends at the '!' and is delimited by a matching
// single quote; a bare token is a run of letters/digits/underscore (incl. accents).
func sheetNameBefore(formula string, bang int) string {
	if bang == 0 {
		return ""
	}
	if formula[bang-1] == '\'' {
		open := strings.LastIndexByte(formula[:bang-1], '\'')
		if open < 0 {
			return ""
		}
		return formula[open+1 : bang-1]
	}
	start := bang
	for start > 0 && isSheetNameByte(formula[start-1]) {
		start--
	}
	return formula[start:bang]
}

func isSheetNameByte(b byte) bool {
	switch {
	case b >= 'a' && b <= 'z', b >= 'A' && b <= 'Z':
		return true
	case b >= '0' && b <= '9':
		return true
	case b == '_', b == '.', b >= 0x80: // underscore, dot, UTF-8 continuation/accents
		return true
	}
	return false
}

func writeJSON(path string, data interface{}) error {
	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("create file %s: %w", path, err)
	}
	defer file.Close()

	enc := json.NewEncoder(file)
	enc.SetIndent("", "  ")
	if err := enc.Encode(data); err != nil {
		return fmt.Errorf("encode JSON to %s: %w", path, err)
	}
	return nil
}

func sanitizeFilename(name string) string {
	return strings.ReplaceAll(name, "/", "_")
}

// ── Row-type classification ──────────────────────────────────────────────────
//
// The five data-entry sheets (Fixas, Variáveis, Extras, Adicionais, Receitas)
// share one palette: a month banner (C0C0C0), an Item/Data/Valor column-label
// band (D8D8D8), and per-block Total rows (F2F2F2 + a SUM formula). Categories
// (col A, bold) and subcategories (col B) are filled down on every data row;
// blocks are delimited by Total rows rather than separate header rows.
// Listas de itens and Referência de Categorias use other palettes, so the
// classifier leads with text/formula signals and treats fill as corroboration.

const (
	fillMonthBand = "C0C0C0" // month banner row
	fillColHeader = "D8D8D8" // Item / Data / Valor label row
	fillTotalRow  = "F2F2F2" // per-block Total row
)

var monthNames = []string{
	"janeiro", "fevereiro", "março", "abril", "maio", "junho",
	"julho", "agosto", "setembro", "outubro", "novembro", "dezembro",
}

func classifyRowTypes(rows []RowDump) {
	for i := range rows {
		if rows[i].RowType == "" { // keep pre-assigned types (e.g. "separator")
			rows[i].RowType = classifyRow(rows[i])
		}
	}
}

func classifyRow(rd RowDump) string {
	switch {
	case monthNameCount(rd) >= 3 || dominantBg(rd) == fillMonthBand:
		return "header-month"
	case hasColumnLabels(rd) || dominantBg(rd) == fillColHeader:
		return "header-col"
	case isTotalRow(rd):
		return "total-row"
	case isCategoryLabel(rd):
		return "category-label"
	case hasSubcategory(rd) || hasDataValues(rd):
		return "data-row"
	default:
		return "unknown"
	}
}

func dominantBg(rd RowDump) string {
	counts := map[string]int{}
	for _, c := range rd.Cells {
		if c.Style.BgColor != "" {
			counts[c.Style.BgColor]++
		}
	}
	best, bestN := "", 0
	for color, n := range counts {
		if n > bestN {
			best, bestN = color, n
		}
	}
	return best
}

func monthNameCount(rd RowDump) int {
	seen := map[string]bool{}
	for _, c := range rd.Cells {
		v := strings.ToLower(c.Value)
		for _, m := range monthNames {
			if v == m {
				seen[m] = true
			}
		}
	}
	return len(seen)
}

func hasColumnLabels(rd RowDump) bool {
	hasItem, hasValor := false, false
	for _, c := range rd.Cells {
		switch strings.ToLower(c.Value) {
		case "item":
			hasItem = true
		case "valor":
			hasValor = true
		}
	}
	return hasItem && hasValor
}

func isTotalRow(rd RowDump) bool {
	hasTotalLabel, hasSum := false, false
	for _, c := range rd.Cells {
		if strings.EqualFold(strings.TrimSpace(c.Value), "total") {
			hasTotalLabel = true
		}
		if strings.HasPrefix(strings.ToUpper(c.Formula), "SUM(") {
			hasSum = true
		}
	}
	return hasTotalLabel || (hasSum && dominantBg(rd) == fillTotalRow)
}

// isCategoryLabel matches a standalone grouping row: col A populated, every
// other cell empty (no subcategory, no data, no formula).
func isCategoryLabel(rd RowDump) bool {
	hasA := false
	for _, c := range rd.Cells {
		if c.Col == "A" {
			if c.Value == "" {
				return false
			}
			hasA = true
			continue
		}
		if c.Value != "" || c.Formula != "" {
			return false
		}
	}
	return hasA
}

func hasSubcategory(rd RowDump) bool {
	for _, c := range rd.Cells {
		if c.Col == "B" && c.Value != "" {
			return true
		}
	}
	return false
}

func hasDataValues(rd RowDump) bool {
	for _, c := range rd.Cells {
		if num, err := excelize.ColumnNameToNumber(c.Col); err == nil && num >= 4 {
			if c.Value != "" || c.Formula != "" {
				return true
			}
		}
	}
	return false
}

// ── JSON schema types ────────────────────────────────────────────────────────

type Manifest struct {
	Source string      `json:"source"`
	Sheets []SheetInfo `json:"sheets"`
}
type SheetInfo struct {
	Name string `json:"name"`
	File string `json:"file"`
	Rows int    `json:"rows"`
	Cols int    `json:"cols"`
}
type SheetDump struct {
	Sheet          string             `json:"sheet"`
	Dimensions     Dim                `json:"dimensions"`
	ColumnWidths   map[string]float64 `json:"columnWidths"`
	RowHeights     map[string]float64 `json:"rowHeights"`
	MergedCells    []Merge            `json:"mergedCells"`
	CrossSheetRefs []string           `json:"crossSheetRefs"`
	Rows           []RowDump          `json:"rows"`
}
type Dim struct {
	Rows int `json:"rows"`
	Cols int `json:"cols"`
}
type Merge struct {
	Range string `json:"range"`
	Value string `json:"value"`
}
type RowDump struct {
	Row     int    `json:"row"`
	RowType string `json:"rowType,omitempty"`
	RowFill string `json:"rowFill,omitempty"` // row-level fill on otherwise-empty rows (e.g. black separator bands)
	Cells   []Cell `json:"cells"`
}
type Cell struct {
	Col     string `json:"col"`
	Value   string `json:"value"`
	Formula string `json:"formula,omitempty"`
	Style   Style  `json:"style"`
}
type Style struct {
	BgColor      string `json:"bgColor,omitempty"`
	Bold         bool   `json:"bold"`
	BorderTop    bool   `json:"borderTop"`
	BorderBottom bool   `json:"borderBottom"`
	BorderLeft   bool   `json:"borderLeft"`
	BorderRight  bool   `json:"borderRight"`
}
