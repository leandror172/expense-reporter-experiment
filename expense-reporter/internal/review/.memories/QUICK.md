# review — QUICK

**What:** builds the offline HTML review page. `ReadQueue` (classified CSV, 10 fields —
col 8 = type, col 9 = keyword_hint, col 10 = already_logged) + `BuildTaxonomy` (picker tree from **`config/taxonomy.json`**) + `Render`
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
exactly 10 semicolon fields since T-80/s77, and `auto_inserted` spelled **`true`/`false`** — the inverse of
`fmt %v` on a bool. `1`/`0` is REJECTED on purpose: it was accepted for months while NO
producer emitted it, so `review` failed on every real `classified.csv` and no test noticed
(the reader's only inputs were fixtures authored to satisfy the reader). Do not "soften"
this back to `strconv.ParseBool` — one producer, one spelling. Guard:
`cmd.TestClassifiedCSV_ReviewReadsWhatBatchAutoWrote`.
⚠️ **There are TWO producers of that cell, and only the filter keeps them equal (noted s72).**
`writeClassifiedCSV` renders `fmt.Sprintf("%v", r.AutoInserted)`; `writeReviewCSV` hardcodes
the literal `"false"`. They agree only because review.csv skips auto-inserted rows, so the
hardcode is never wrong *today*. No test distinguishes them either — the cross-writer tests
feed rows that are all `AutoInserted: false`, so the two spellings agree by construction.
Relax that filter and the literal becomes a lie with nothing to catch it.

**The value column is parsed through `parse.Value`, NOT `utils` directly (T-72, s73).**
`ReadQueue` used to call `utils.ParseCurrencyWithInstallments` itself, which meant it never
applied the BR thousands normalization `internal/parse` performs — so `writeClassifiedCSV`
could emit a `1.234,56` this reader HARD-ERRORED on, killing the whole review step over one
row, with both sides' own tests green. Same shape as T-54: one function, two callers, and a
step only one of them applied. Routing through the boundary is also how the T-64 multiplier
notation (`405,25 x4`, which MULTIPLIES where `total/N` divides) reached this side without
being taught twice — and a count that died here would be the T-21 bug again, with a 4×
error instead of a 3× one. The error is deliberately NOT re-labelled: `parse.Value` wraps
with `ErrInvalidValue` whose text is "invalid value", so the human-facing message is
byte-identical and callers gain `errors.Is` on the field sentinel.
⚠️ **Only the THOUSANDS test guards this routing.** Mutation-verified: deleting the
normalization inside `parse.Value` turns the thousands tests red while both multiplier
tests stay green, because utils knows the notation either way. Do not delete the thousands
case as redundant with the multiplier one — it is the only thing that can tell whether this
reader still goes through the boundary.

**`QueueEntry` carries the installment COUNT, not just the raw token (T-21, s69).**
`ReadQueue` used to discard it (`perInstallment, _, err`) while keeping `RawValue` for
display, so the count died here and `apply` — the only consumer that needs it — recorded
every reviewed `99,90/3` as a single row. It now rides `QueueEntry.Installments` → the
page's embedded DATA → `exportReviewed()` → `reviewed.json`. The page shows ONE row with
a ×N badge (`installmentBadge`), never N rows: expanding would make the reviewer take the
same categorisation decision N times, and expansion is `apply`'s job.
⚠️ **The `exportReviewed()` hop is UNGUARDED** — deleting the field from the export leaves
the whole Go suite green (measured, not assumed). See `test/.memories/KNOWLEDGE.md`.

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

**`QueueEntry.KeywordHint` is the keyword layer's COMPETING suggestion (A1, s72).**
Ninth CSV column, filled by `classifier.KeywordHint` — the advisory sibling of the
auto-insert gate: the gate fires on model⊕keyword AGREEMENT, the hint on DISAGREEMENT,
both at unambiguous specificity 1.00. Measured on the first close: fires on 14 of 65 rows,
right 71.4% of the time, holds the reviewer's answer on 10 of 25 corrections. **Advisory
ONLY** — at that precision, auto-applying would inject errors on ~3 of every 14 rows, so
the page renders an amber `.kw-hint` badge beside the model's answer and nothing selects
it. **It deliberately does NOT round-trip through `exportReviewed()`** — it is input TO the
human, not part of their decision, and a value crossing that unguarded JS hop would be a
second T-61. Empty is the signal for "no second opinion"; never a placeholder.

**`QueueEntry.AlreadyLogged` says why the row is already in the expense log (T-80, s77).**
Tenth CSV column; `logged` = every id this row would write is in the log, `partial` = some
are, empty = none. `batch-auto` holds such a row back from auto-insert WHATEVER the gate
decided, so the reviewer adjudicates it with the model's suggestion in hand. The page draws
two DISTINCT amber badges — "already in log" (solid rule) and "partly logged" — because the
two ask different questions: a full duplicate needs a ruling on whether this is a genuinely
separate purchase (same-day duplicates hash alike, 8 legitimate pairs live in the log),
while a partial series is the one of the two that can legitimately be completed.
Advisory, and like `KeywordHint` it deliberately does NOT round-trip through
`exportReviewed()`. Empty is the signal; never a placeholder.
⚠️ **`TestReadQueue` cannot discriminate columns 8 and 9** — mutation-verified s77: reading
`record[8]` into `AlreadyLogged` leaves the whole reader suite green. The seam tests and
`TestReview_ProducesHTMLWithQueueAndTaxonomy` are what catch it.

**Gotcha:** edit only `internal/review/template/review.html` (60KB). The rendered
`review*.html` files at repo root are large — don't read them into context; use a Haiku
subagent with targeted questions if you must inspect one.
