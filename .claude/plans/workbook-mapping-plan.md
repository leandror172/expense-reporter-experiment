# Plan: Complete Workbook Mapping
## Goal

Produce a complete, multi-layer structural map of `Planilha_Normalized_Final.xlsx` that is
rich enough to spec and implement a "generate workbook from database" command — replacing
the current insert-into-workbook approach.

The three layers are complementary: JSON gives machine-readable fidelity, screenshots give
visual fidelity, claude.ai synthesizes both into an actionable generator spec.

---

## Layer 1 — Complete JSON Dump (enhanced Go script)

**File:** `expense-reporter/cmd/workbook-inspect/main.go` (already exists, needs rewrite)

**What the current script misses:**
- Column A (category labels, merged across rows)
- Column C (third label column in some sheets)
- Cell styles: background fill color, font weight (bold), border presence — these are the
  only way to distinguish header rows / data rows / TOTAL rows / category groupings
- Cross-sheet formula references (especially in `Listas de itens`)
- Full formula inventory (current script only samples one per block)
- Column widths and row heights
- Conditional formatting rules

**Output format:** JSON, not markdown. Markdown is human-readable but lossy for a machine.
JSON is directly loadable into a future session's context.

**Schema to produce per sheet:**

```json
{
  "sheet": "Adicionais",
  "dimensions": { "rows": 417, "cols": 40 },
  "columnWidths": { "A": 12.5, "B": 18.0, ... },
  "rowHeights": { "1": 15.0, "2": 15.0, ... },
  "mergedCells": [
    { "range": "A3:A17", "value": "Lazer" }
  ],
  "rows": [
    {
      "row": 1,
      "cells": [
        {
          "col": "D",
          "value": "Janeiro",
          "formula": "",
          "style": {
            "bgColor": "FF4472C4",
            "bold": true,
            "borderBottom": true
          }
        }
      ]
    }
  ]
}
```

**Implementation steps:**

1. For each sheet, use `f.GetRows()` to iterate all rows and columns
2. For each non-empty cell, call:
   - `f.GetCellValue(sheet, ref)` — display value
   - `f.GetCellFormula(sheet, ref)` — formula if present
   - `f.GetCellStyle(sheet, ref)` → style index → `f.GetStyle(idx)` → extract:
     - `Fill.Pattern.Color` (background)
     - `Font.Bold`
     - `Border` (top/bottom/left/right presence)
3. `f.GetMergeCells(sheet)` — merged regions
4. `f.GetColWidth(sheet, col)` — column widths
5. `f.GetRowHeight(sheet, row)` — row heights
6. Detect cross-sheet formula references: scan all formulas for `SheetName!CellRef` pattern
7. Output one JSON file per sheet + a manifest JSON listing all sheets

**Style classification pass (post-extraction):**
After dumping raw styles, add a classifier that maps style fingerprints to row types:
- "header-month": bold + blue/dark background → month name row
- "header-col": bold + lighter background → Item/Data/Valor label row  
- "subcat-header": bold, specific bg → subcategory name row (currently found via col B scan)
- "data-row": no fill or white fill, no bold
- "total-row": bold + border-top, often contains SUM formula
- "category-label": merged col A, specific color

This classification is what a generator needs to reproduce styling faithfully.

**Run command:**
```bash
cd expense-reporter
go run ./cmd/workbook-inspect/ /path/to/workbook.xlsx ./workbook-dump/
# produces: workbook-dump/manifest.json, workbook-dump/Adicionais.json, etc.
```

---

## Layer 2 — Screenshot Capture via Chrome Automation

**Tool:** `mcp__claude-in-chrome__*` or `mcp__plugin_playwright_playwright__*`

**Goal:** Visual ground truth — captures frozen panes, color themes, merged cell rendering,
and anything excelize's style parsing might miss or misinterpret.

**Strategy:** Open workbook in Google Sheets (already confirmed it opens), then
programmatically scroll and screenshot each sheet section-by-section.

**Steps per sheet:**

1. Navigate to Google Sheets with the file already open (or upload it)
2. Click the sheet tab
3. Scroll to top-left (home position)
4. Take screenshot of the header area (rows 1–5, full width if possible)
5. Scroll right to capture the full month column span (col A → col AK+)
   - The sheets are ~37 columns wide; may need 3–4 horizontal screenshots per section
6. Scroll down to capture each subcategory block (use the block list from Layer 1
   JSON to know where blocks start/end)
7. Name screenshots systematically: `{sheet}_{startRow}-{endRow}_{colRange}.png`

**Screenshot inventory to produce:**
- Per data sheet (Fixas, Variáveis, Extras, Adicionais):
  - Header rows (1–2): full width
  - 2–3 representative subcategory blocks with data: full width
  - One TOTAL row in context
- `Listas de itens`: full-width overview (this is the summary/rollup sheet)
- `Receitas`: header + a few rows
- `Referência de Categorias`: full dump (it's narrow)

**Output:** `workbook-screenshots/{sheet}/` directories with numbered PNGs.

**Note on automation:** The scrolling automation in Google Sheets can be tricky — use
`mcp__claude-in-chrome__javascript_tool` to call `scrollTo` on the sheet container
rather than relying on keyboard/click navigation for precise positioning.

---

## Layer 3 — Spec Synthesis via claude.ai

**Tool:** claude.ai web (2× usage active until 2026-07-05 — use it)

**Input to provide:**
1. The JSON dump from Layer 1 (or a summarized version if too large for context)
2. Selected screenshots from Layer 2 (the header rows + 1–2 blocks per sheet)
3. The existing `workbook-map.md` (from session 25) as context
4. The current `columns.go` (month column mapping) and `Referência de Categorias` dump

**Prompt framing for claude.ai:**

> "I want to write a Go program that generates this Excel workbook from scratch, given a
> list of expense entries per subcategory. The workbook has 7 sheets. Here is the complete
> structural data [JSON] and visual screenshots. Please produce:
> 1. A sheet-by-sheet spec describing the layout rules (not the data, just the structure)
> 2. The formula patterns for each sheet, expressed as templates
> 3. The style palette: what colors/bold/borders are used for each row type
> 4. Any cross-sheet references I need to replicate
> 5. Open questions or ambiguities that need clarification before I can implement"

**Output expected:** A `workbook-generator-spec.md` document that a future Claude Code
session can implement directly, without needing to re-examine the workbook.

---

## Sequencing

1. **Layer 1 first** — the JSON dump informs which sheet sections to screenshot in Layer 2
   (block start/end rows, column extents). Don't run Layer 2 blind.
2. **Layer 2 after Layer 1** — use the block list to drive targeted screenshots.
3. **Layer 3 last** — needs both layers as input. Can be done in a separate session or
   immediately after Layer 2 if context allows.

## Files to produce

| File | Layer | Description |
|------|-------|-------------|
| `expense-reporter/cmd/workbook-inspect/main.go` | 1 | Rewritten to output JSON + style extraction |
| `.claude/workbook-dump/manifest.json` | 1 | Sheet inventory with dimensions |
| `.claude/workbook-dump/{SheetName}.json` | 1 | Per-sheet cell dump with styles |
| `.claude/workbook-screenshots/{sheet}/*.png` | 2 | Section screenshots |
| `.claude/plans/workbook-generator-spec.md` | 3 | Generator spec from claude.ai synthesis |

## Open questions to answer during mapping

- Does `Listas de itens` aggregate the other sheets via cross-sheet SUM formulas, or
  is it manually maintained?
- Are column widths meaningful (i.e. do they differ by column type) or uniform?
- Are row heights uniform across all data sheets?
- Do styles differ between sheets (e.g. Adicionais vs Fixas color scheme) or is the
  palette workbook-wide?
- What drives the varying block sizes (5 rows vs 16 rows)? Is it fixed per subcategory
  or does it grow with data?
