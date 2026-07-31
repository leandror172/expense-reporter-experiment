# review — QUICK

**What:** builds the offline HTML review page. `ReadQueue` (classified CSV, 8 fields —
col 8 = type) + `BuildTaxonomy` (picker tree from **`config/taxonomy.json`**) + `Render`
(inject JSON into `template/review.html` via go:embed).

**The picker's taxonomy comes from `config/taxonomy.json` (S2, s68) — NOT the workbook.**
It used to come from the workbook's "Referência de Categorias" sheet while every other
command routed with `taxonomy.json`, so the close cycle ran on TWO vocabularies and they
drifted: **7 leaves** (`IRFF` vs `IRRF`, `Apoia-se 4i20` vs `Apoia-se`, `alguma coisa
sindicato` vs `extra sindicato`, `Produtos de casa`). A reviewer picking a workbook-only
leaf produced a row `generate-workbook` warn-skipped — it vanished with no visible error —
and two such picks were logged as CORRECTIONS, the highest-priority few-shot source, so
the drift was actively teaching the classifier the unroutable spelling.
`review` now reads NO workbook: `--workbook` is gone and `excel.LoadReferenceSheet` has
only WS-E dead callers left. **`BuildTaxonomy` must never sort** — the JSON array carries
the author's order; the old sorting existed only because map iteration is random.

**`ReadQueue` returns THREE values (S1, s68): `(entries, unreviewable, error)`.** A row
batch-auto could not parse is written with its raw text in the item column and EMPTY
date/value cells; this reader used to hard-error on that shape, so 4 bad rows in 69 killed
the whole review step on real data. Those rows now come back as `unreviewable` and
`cmd/review.go` names each on stderr — skipping without reporting would swap a loud failure
for a lost expense. **The tolerance is deliberately narrow:** empty date/value is a
documented producer output; a bad number, bad confidence or wrong field count is corruption
and still hard-errors. The warning lives in `cmd`, not here — same reason `parse` reports
`YearSource` instead of printing.

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
