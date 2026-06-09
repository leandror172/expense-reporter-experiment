# excel package — quick notes

## Workbook structural map → KNOWLEDGE.md
Full structure of `Planilha_Normalized_Final.xlsx` mapped session 26 via `cmd/workbook-inspect`
(JSON dump in `.claude/workbook-dump/`, gitignored). See `KNOWLEDGE.md` for the distilled facts:
two sheet families, per-sheet palette/fonts, fill-down vs merge, block/total-formula pattern,
black separator rows, and cross-sheet wiring (Listas pulls; Referência is the row-map). This
feeds the future "generate workbook from database" command.

## Reference sheet columns (Referência de Categorias)

`LoadReferenceSheet` reads 6 columns:
- A: sheet name (mainType)
- B: category
- C: subcategory
- D: subcategory header row number → `SubcategoryMapping.RowNumber`
- F: TOTAL row number → `SubcategoryMapping.TotalRow`

**Columns D and F are never read back after loading** — confirmed across all commands
(apply, batch, auto, batch-auto). The write path always resolves row numbers dynamically
via `FindSubcategoryRowBatch` (scans column B of the data sheet). New reference sheet
entries only need columns A, B, C.

`CapacityInfo` and `GetSubcategoryCapacity` (reader.go) consume `TotalRow` but are dead
code — no callers outside reader.go as of session 25 (2026-06-08).

## Subcategory boundary detection (AllocateEmptyRows / FindNextEmptyRow)

Scans column B of the data sheet. Treats a non-empty column B value that differs from
the current subcategory as a section boundary. **Must use `strings.TrimSpace` on both
sides** — workbook cells often carry trailing spaces (e.g. `"Escritório "`), which
without trimming falsely trigger "section full". Fixed in session 25.

## Two reader.go files

- `internal/excel/reader.go` — workbook I/O: LoadReferenceSheet, FindSubcategoryRowBatch,
  AllocateEmptyRows, CapacityInfo
- `internal/apply/reader.go` — parses reviewed.json; unrelated to excel I/O
