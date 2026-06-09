# excel package — quick notes

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
