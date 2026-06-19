# review — QUICK

**What:** builds the offline HTML review page. `ReadQueue` (classified CSV) +
`BuildTaxonomy` (tree from workbook mappings) + `Render` (inject JSON into
`template/review.html` via go:embed).

- Types (types.go): `QueueEntry` with `Predicted{Sheet,Category,Subcategory}`
  (`Sheet` = expense type; `omitempty`). `Taxonomy{Sheets[]}` → `Sheet{Name,Categories}`
  → `Category{Name,Subcategories}`.
- Classified CSV input (`queue.go`) is **7 fields**: item,date,value,subcategory,
  category,confidence,autoInserted — **no sheet/type column** (RUI-4 would add it,
  still open). The page re-derives candidate types from the taxonomy (`SHEETS_FOR`).
- `taxonomy.go`: `sheetOrder = [Fixas, Variáveis, Extras, Adicionais]`.

**Key facts (session 32):**
- The page **is fully type-aware**: keys 1–4 set the type, it flags ambiguous (cat,sub)
  for the user to choose, and `exportReviewed()` (template ~line 1486) **already emits**
  `reviewed:{sheet,category,subcategory}` — full path. No page change needed to capture
  type; the loss is purely on the Go ingest/log side.
- In-progress state persists in browser `localStorage`, key
  `expense-review:v1:rows:<source>:<generatedAt>` → saved corrections are recoverable by
  reopening the same HTML and re-exporting.

**Gotcha:** edit only `internal/review/template/review.html` (60KB). The rendered
`review*.html` files at repo root are large — don't read them into context; use a Haiku
subagent with targeted questions if you must inspect one.
