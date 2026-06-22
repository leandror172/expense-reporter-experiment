# review — QUICK

**What:** builds the offline HTML review page. `ReadQueue` (classified CSV) +
`BuildTaxonomy` (tree from workbook mappings) + `Render` (inject JSON into
`template/review.html` via go:embed).

- Types (types.go): `QueueEntry` with `Predicted{Type,Category,Subcategory}`
  (`Type` json:"type,omitempty"). `Taxonomy{Types[] json:"types"}` → `Type{Name,Categories}`
  → `Category{Name,Subcategories}`. ⚠ The `sheets`→`types` / `sheet`→`type` JSON-tag rename
  was completed **session 34 (commit 321dd2f)** — the template JS had migrated during Plan A/B
  but the Go structs still serialized `sheets`/`sheet`, crashing the page (`TAX.types not
  iterable`). The cross-language key contract is now guarded by `render_test.go` (asserts the
  embedded JSON carries `"types"`/`"type"`, never `"sheets"`/`"sheet"`).
- Classified CSV input (`queue.go`) is **8 fields**: item,date,value,subcategory,
  category,confidence,auto_inserted,**type** — RUI-4 added the type column (session 33,
  committed 41afa56); `ReadQueue` reads `record[7]` into `Predicted.Type`. The page can also
  re-derive candidate types from the taxonomy (`typeByName`/`SHEETS_FOR`).
- `taxonomy.go`: `BuildTaxonomy`→`Taxonomy{Types}` via `buildType`; translation happens at the
  output construction — source-side helpers (`sheetOrder=[Fixas,Variáveis,Extras,Adicionais]`,
  `sheetRank`) keep "sheet" because they read workbook-facing `resolver.SheetName`.

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
