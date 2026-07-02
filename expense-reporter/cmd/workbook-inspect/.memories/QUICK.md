# cmd/workbook-inspect — quick notes

## Purpose
Thin CLI wrapper over **`internal/inspect`** — dumps the full structural map of an Excel
workbook to JSON. The library also backs `test/verify.WorkbookStructureMatches`
(generate-workbook acceptance) — **the JSON schema is a test contract**.

## Usage
```
go build -o /tmp/workbook-inspect ./cmd/workbook-inspect/
/tmp/workbook-inspect <workbook.xlsx> <output-dir>
# → output-dir/manifest.json + one <Sheet>.json per sheet
```
Takes the workbook path as an arg — does NOT read config.json.

## Key facts
- Per-sheet JSON: `sheet`, `dimensions`, `columnWidths`, `rowHeights`, `mergedCells[]`,
  `crossSheetRefs[]`, `rows[]` (row → `rowType`, optional `rowFill`, `cells[]` with
  value/formula/style).
- Cell emitted if it has a value, formula, OR non-default style — but only up to the last
  valued cell per row (NOT a complete used-range style map).
- `rowType` classifier degrades on off-palette sheets (text/formula signals first).
- Dump output is gitignored (real expense values).
- Design details + provenance → KNOWLEDGE.md.
