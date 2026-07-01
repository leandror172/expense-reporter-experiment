# review — QUICK

**What:** builds the offline HTML review page. `ReadQueue` (classified CSV, 8 fields —
col 8 = type) + `BuildTaxonomy` (tree from workbook mappings) + `Render` (inject JSON
into `template/review.html` via go:embed).

- Types (types.go): `QueueEntry` with `Predicted{Type,Category,Subcategory}`;
  `Taxonomy{Types[]}` → `Type{Name,Categories}` → `Category{Name,Subcategories}`.
  JSON keys are `types`/`type` (never `sheets`/`sheet`) — the cross-language contract is
  guarded by `render_test.go`. History of that rename → KNOWLEDGE.md.
- The page **is fully type-aware**: keys 1–4 set the type, ambiguous (cat,sub) flagged
  for the user; `exportReviewed()` emits `reviewed:{type,category,subcategory}` — full
  path. apply's `UnmarshalJSON` still reads legacy `sheet`.
- In-progress state persists in browser `localStorage`
  (`expense-review:v1:rows:<source>:<generatedAt>`) → recoverable by reopening the same
  HTML and re-exporting.

**Gotcha:** edit only `internal/review/template/review.html` (60KB). The rendered
`review*.html` files at repo root are large — don't read them into context; use a Haiku
subagent with targeted questions if you must inspect one.
