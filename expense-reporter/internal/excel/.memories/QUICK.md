# excel package — quick notes

## What
Excelize wrapper — workbook read/write, reference-sheet loading, row allocation. Since
the WS-B log-append pivot, only plain `batch` still writes the workbook through here;
`generate-workbook` uses `internal/generate` instead.

## Key facts
- `LoadReferenceSheet` reads Referência de Categorias cols A (sheet), B (category),
  C (subcategory), D (header row), F (TOTAL row) — but **D and F are never read back**;
  the write path resolves rows dynamically via `FindSubcategoryRowBatch`. New reference
  entries only need A, B, C. Details + dead-code notes → KNOWLEDGE.md.
- **Boundary detection trims whitespace:** `AllocateEmptyRows`/`FindNextEmptyRow` must
  `strings.TrimSpace` both sides — workbook cells carry trailing spaces
  (e.g. `"Escritório "`) that otherwise falsely trigger "section full".
- **Two reader.go files:** `internal/excel/reader.go` = workbook I/O;
  `internal/apply/reader.go` = reviewed.json parsing (unrelated).
- Workbook structural map (sheet families, palette, block/total patterns, cross-sheet
  wiring) → KNOWLEDGE.md.
