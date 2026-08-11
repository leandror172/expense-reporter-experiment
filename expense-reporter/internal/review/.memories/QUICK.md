# review — QUICK

**What:** builds the offline HTML review page. `ReadQueue` (classified CSV, 8 fields —
col 8 = type) + `BuildTaxonomy` (tree from workbook mappings) + `Render` (inject JSON
into `template/review.html` via go:embed).

**Input contract — `ReadQueue` must match batch-auto's writers exactly (T-54, s68):**
exactly 8 semicolon fields, and `auto_inserted` spelled **`true`/`false`** — the inverse of
`fmt %v` on a bool. `1`/`0` is REJECTED on purpose: it was accepted for months while NO
producer emitted it, so `review` failed on every real `classified.csv` and no test noticed
(the reader's only inputs were fixtures authored to satisfy the reader). Do not "soften"
this back to `strconv.ParseBool` — one producer, one spelling. Guard:
`cmd.TestClassifiedCSV_ReviewReadsWhatBatchAutoWrote`.

**The id is derived, not carried.** `ReadQueue` hashes the CSV's date column into
`QueueEntry.ID`. Since T-41 slice 3 that column is canonical `DD/MM/YYYY`, so the id equals
the one both JSONL logs hold — that is what lets `reviewed.json` join them. Guard:
`cmd.TestClassifiedCSV_QueueIDJoinsTheExpenseLogID` (mutation-verified: revert
`dateCell()` to a year-less date and ONLY that test goes red).

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
