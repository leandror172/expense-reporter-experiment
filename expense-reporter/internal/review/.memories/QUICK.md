# review — QUICK

**What:** builds the offline HTML review page. `ReadQueue` (classified CSV) +
`BuildTaxonomy` (tree from workbook mappings) + `Render` (inject JSON into
`template/review.html` via go:embed).

- Types (types.go): `QueueEntry` with `Predicted{Type,Category,Subcategory}`
  (`Type` = expense type, renamed from `Sheet`, Plan A R4; `omitempty`).
  `Taxonomy{Types[]}` → `ExpenseType{Name,Categories}` → `Category{Name,Subcategories}`.
- Classified CSV input (`queue.go`) is **7 fields**: item,date,value,subcategory,
  category,confidence,autoInserted — **no type column** (RUI-4 would add it, still open).
  The page re-derives candidate types from the taxonomy (`typeByName`/`SHEETS_FOR`).
- `taxonomy.go`: `sheetOrder = [Fixas, Variáveis, Extras, Adicionais]` (still named
  `sheetOrder` — it orders worksheets; the review-facing identifiers are `type`).

**Key facts:**
- The page **is fully type-aware**: keys 1–4 set the type, it flags ambiguous (cat,sub)
  for the user to choose, and `exportReviewed()` (template ~line 1504) emits
  `reviewed:{type,category,subcategory}` — full path. JSON export key migrated
  `sheet`→`type` (Plan A R6); apply's `UnmarshalJSON` still reads legacy `sheet`.
- The Go ingest/log loss is **fixed** (Plan A): apply now persists the type. The
  generator routes typed entries by full path ([[taxonomy]] two-tier routing).
- In-progress state persists in browser `localStorage`, key
  `expense-review:v1:rows:<source>:<generatedAt>` → saved corrections are recoverable by
  reopening the same HTML and re-exporting.

**Gotcha:** edit only `internal/review/template/review.html` (60KB). The rendered
`review*.html` files at repo root are large — don't read them into context; use a Haiku
subagent with targeted questions if you must inspect one.
