# Proposed per-folder memories (session 32)

Each section below is the full content of a `.memories/QUICK.md` or `.memories/KNOWLEDGE.md`
to be created at the named path. Split into the real files later. These capture what
cost exploration this session (the expense-type / full-path-routing investigation) so
it won't need re-deriving.

Convention reminder: `QUICK.md` = status + pointers + gotchas (read before touching the
folder); `KNOWLEDGE.md` = architecture/why. Anchors use `file:line` at time of writing
(2026-06-17) — verify before trusting.

After creating the files, add one row each to `.claude/index.md` → "Per-Folder Memories".

================================================================================
## FILE: expense-reporter/internal/feedback/.memories/QUICK.md
================================================================================

# feedback — QUICK

**What:** JSONL logging of classification + insertion events. Two separate files,
two separate structs.

| File on disk | Struct | Written by |
|---|---|---|
| `classifications.jsonl` | `Entry` (feedback.go:27) | add / auto / batch-auto / apply / correct |
| `expenses_log.jsonl` | `ExpenseEntry` (expense_log.go:11) | auto / batch-auto / apply (slim insert log) |

- Both share `GenerateID(item,date,value)` (sha256 prefix, 12 hex) — the **join key**
  across all three files (incl. `reviewed.json`).
- Constructors: `NewConfirmedEntry` / `NewManualEntry` / `NewCorrectedEntry` (feedback.go),
  `NewExpenseEntry` (expense_log.go). `Append` / `AppendExpense` write one JSON line.
- Status values: `confirmed` (auto accepted), `manual` (add, no model), `corrected`
  (user overrode model).

**Gotcha / known gap (session 32):** neither struct carries the expense **type**
(Fixas/Variáveis/Extras/Adicionais). The review UI + apply path *know* the type and
insert into the right worksheet, but it is **dropped at log-write time** → retraining
data loses it. Fix = Plan A (`.claude/plans/persist-expense-type.md`): add
`Type string json:"type,omitempty"`, set post-construction on the apply path only.

**Producers without a type:** auto/batch-auto/add/correct build entries from the
classifier, which doesn't emit type. Only the review→apply path has it.

================================================================================
## FILE: expense-reporter/internal/feedback/.memories/KNOWLEDGE.md
================================================================================

# feedback — KNOWLEDGE

**Two-file split rationale.** `classifications.jsonl` is the rich audit/training log
(predicted vs actual, confidence, model, status). `expenses_log.jsonl` is a slim
"what got inserted" log that the **workbook generator reads** as its entry source
(internal/taxonomy `scanEntries`). So `ExpenseEntry`'s shape is also the generator's
input contract — changing it ripples into taxonomy routing + generate fixtures.

**ID is the contract glue.** `GenerateID` normalizes item (lowercase+trim) then hashes
`item|date|value` (`%.2f`). The same ID appears in `classifications.jsonl`,
`expenses_log.jsonl`, and the UI's `reviewed.json`, which is what lets a backfill match
across them (Plan A Phase B-fill).

**Entry field semantics.** `Entry` keeps both predicted (`PredictedSubcategory/Category`)
and actual (`ActualSubcategory/Category`); confirmed sets predicted==actual, manual
leaves predicted empty, corrected differs. `Now` is a `var` so tests inject a fixed
timestamp.

**Time:** timestamps are RFC3339 UTC.

================================================================================
## FILE: expense-reporter/internal/taxonomy/.memories/QUICK.md
================================================================================

# taxonomy — QUICK

**What:** pure-input package (no excelize) — domain types + loader for the expense
taxonomy + JSONL entry routing. Feeds `internal/generate`.

- Types (types.go): `ExpenseSheet` (→ to be renamed `ExpenseType`, Plan A) → `Category`
  → `Subcat`; `RevenueBlock` for income; `Entry{Item,Day,Value}` is the per-month leaf.
- Loader (loader.go): `LoadTaxonomy(taxonomyPath, entriesPath)`; reads
  `config/taxonomy.json` (keys `sheets`→cats→subcategories; `incomeCategories`→blocks).
- `entriesPath == ""` = skeleton mode (validation only, routing map discarded).

**Identity = full path** (`[ref:taxonomy-identity-key]`,
`.claude/plans/taxonomy-identity-key.md`): a subcategory is `sheet/category/sub`
(income: `group/label`). Only an exact repeated full path errors. Bare leaf names
repeat legitimately (`Orion`×3, `Gás`, `Aluguel` expense+income).

**Routing (loader.go):**
- `buildSubcategoryMap` → bare-name `result` map + sticky `ambiguous` set (a bare name
  on >1 path is dropped + marked ambiguous; `registerTarget` enforces stickiness — see
  the 3× re-add trap).
- `scanEntries` reads `{item,date,value,subcategory}` only → routes by **bare name** →
  `warnUnroutable` (ambiguous vs unknown), skip + exit 0.
- Paths joined with **null byte** `\x00` + `expense`/`income` kind prefix (names contain
  `/`, e.g. `Uber/Taxi`).

**Gotchas:**
- Skeleton path builds the map only to validate, then **discards** it — routing/ambiguity
  logic is exercised only by entry-fed unit tests, not skeleton generation.
- Index into backing slices (`&sheets[i].Cats[j].Subs[k]`) — range copies lose appends.

**Pending (Plan B, `.claude/plans/full-path-entry-routing.md`):** add a second
full-path map; route typed entries by full path, keep bare-name fallback (+ambiguous
skip) for type-less auto/legacy entries. Do NOT delete the ambiguous set.

================================================================================
## FILE: expense-reporter/internal/taxonomy/.memories/KNOWLEDGE.md
================================================================================

# taxonomy — KNOWLEDGE

**Why full path is the identity.** Real taxonomy repeats leaf names across sheets/cats
(7 bare-name collisions). Category-qualification is insufficient: `Pets` is a category
on both Variáveis and Extras, so `Pets/Orion` still collides — the **sheet dimension is
load-bearing**. Full three-segment path is the only safe identity.

**Two collision classes:** same-sheet collisions (`Gás`, `Produtos` in Variáveis) were
genuine data bugs fixed at source while authoring `config/taxonomy.json`; cross-sheet /
income-vs-expense collisions (`Orion`, `Dentista`, `Aluguel`) are legal and only broke
the old bare-name validation.

**The interim guard is a stopgap, not the design.** Bare-name routing + warn+skip keeps
behavior *safe* (never silent misroute) but *incomplete* (ambiguous entries don't
populate) until entries carry their type. Full-path routing (Plan B) needs the entry
contract to carry type — which is why Plan A (persist type) must land first.

**Two taxonomy sources exist and don't know about each other** (cross-cutting, also
noted in classifier memory):
- `config/taxonomy.json` — sheet→cat→sub tree, this package, generator side.
- `data/classification/feature_dictionary_enhanced.json` `category_mapping` — flat
  sub→cat, classifier side. **No sheet dimension.**
The classifier never reads the sheet-aware tree; that's the architectural debt behind
"the classifier can't emit type yet."

**Real file is gitignored** (`config/taxonomy.json` reveals personal categories); the
committed test input is `test/fixtures/generate-basic/taxonomy.json`. Fidelity of a
hand-authored taxonomy is checked by CSV↔JSON symmetric-difference, not "112 subs + no
dups" (necessary but not sufficient).

================================================================================
## FILE: expense-reporter/internal/apply/.memories/QUICK.md
================================================================================

# apply — QUICK

**What:** ingests the review UI's `reviewed.json`, inserts new rows into the workbook,
and writes feedback + expense logs. Driven by `cmd/.../apply.go`.

- Types (types.go): `ReviewedFile{reviewedAt,source,entries}` → `ReviewedEntry`
  {id,item,date,value,confidence,`Predicted`,`action`,`Reviewed *ReviewedLocation`}.
- `ReviewedLocation` (types.go:31) = `{Sheet,Category,Subcategory}` — the full path the
  user chose. `Sheet` is the expense **type** (to be renamed `Type` + JSON key migrated
  `sheet`→`type` with legacy fallback, Plan A F3).
- `reader.go` `ReadReviewed` decodes + validates.
- `action` ∈ confirmed / corrected / skipped (`apply.Action*` consts).

**Key fact (session 32):** apply.go **already uses** `entry.Reviewed.Sheet` for excel
insertion (`SheetName` at apply.go:194/212/222/248/257) — full path is live on the
workbook path. The ONLY drop is at log-write: `NewExpenseEntry`/`NewCorrectedEntry`
(apply.go:282/136/298/305) don't pass the type. Plan A fixes exactly those sites.

**Fixtures:** `test/fixtures/apply-basic/` has `reviewed.json`, `seed-classifications.jsonl`,
`expected-feedback.jsonl` (no `expected-expenses_log.jsonl`).

================================================================================
## FILE: expense-reporter/internal/review/.memories/QUICK.md
================================================================================

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

================================================================================
## FILE: expense-reporter/internal/classifier/.memories/KNOWLEDGE.md  (ADDENDUM — append)
================================================================================

# classifier — KNOWLEDGE (addendum, session 32)

**The classifier is sheet/type-unaware, by data design.**
- `Result = {Subcategory, Category, Confidence}` (classifier.go:15) — no sheet/type.
- `Taxonomy = map[string]string` (sub→cat), loaded from
  `feature_dictionary_enhanced.json` `category_mapping` — **flat, no sheet dimension.**
- Output is schema-locked: `responseSchema` forces exactly {subcategory,category,
  confidence}; the model can only emit those.

**Consequence:** the classifier cannot emit the expense **type** (Fixas/…). For
*unambiguous* subcategories the type is recoverable as a lookup (a unique leaf pins one
full path); for the ~6 *ambiguous* leaves (`Orion`,`Gás`,`Dentista`,…) it is a genuine
classification decision the model has never had labeled examples for. This is why
full-path entry routing (Plan B) currently relies on the review UI for type.

**Retraining is cheaper than it looks.** `training_data_complete.json` already carries
the type in its `source` field: Variáveis 260 / Adicionais 187 / Fixas 143 / Extras 17
(607/694), mapping 1:1 onto the four taxonomy types. Plus the review-corrected gold
labels (once Plan A stops discarding them). Tasks 5.R4 (historical extraction) + RUI-4
(emit full path into classified CSV) are the path to a type-emitting classifier.

**Terminology:** "expense type" = the domain concept (the bucket); "sheet" = the Excel
worksheet that renders it (same value, e.g. "Fixas"). Rename the concept to *type*; keep
worksheet-addressing as *sheet* (`models.SheetLocation.SheetName`, `internal/inspect`).

================================================================================
# UPDATES TO EXISTING MEMORIES (apply when the triggering plan lands)
================================================================================

These existing memory files carry facts that Plan A / Plan B make stale. Anchors are
file:line as of 2026-06-17 — re-find the text before editing. Grouped by trigger so
they're applied at the right time (don't pre-edit before the code changes).

--------------------------------------------------------------------------------
## Trigger: Plan A Phase F (add `type` field)
--------------------------------------------------------------------------------

### expense-reporter/.memories/KNOWLEDGE.md  (~line 75)
- The line `expenses_log.jsonl — slim: item, date, value, subcategory, category,
  timestamp.` → add **`type`** (omitempty) to the field list. Note it carries the
  expense type and is set only on the apply path (review-produced entries).

### expense-reporter/test/.memories/KNOWLEDGE.md  (~lines 22, 27–28, 78–79)
- Fixture-format section: note `expected-feedback.jsonl` / `expected-expenses_log.jsonl`
  may now contain an optional `"type"` key; `verify.ExpenseLogMatches` /
  `ClassificationsMatch` compare it.
- Add a gotcha next to "the fixture format is a contract": **when adding `type`, keep a
  typed/type-less MIX across fixture lines** — typing every line masks a broken
  type-less routing fallback (the Plan B regression trap).

--------------------------------------------------------------------------------
## Trigger: Plan A Phase R (rename ExpenseSheet → ExpenseType + JSON key migration)
--------------------------------------------------------------------------------

### expense-reporter/.memories/KNOWLEDGE.md  (~line 173)
- The domain-types list `Entry/Subcat/Category/ExpenseSheet/RevenueBlock` →
  rename `ExpenseSheet` to **`ExpenseType`**.

### expense-reporter/internal/generate/.memories/QUICK.md  (~line 55)
- `taxonomy.Entry`/`taxonomy.ExpenseSheet` → `taxonomy.ExpenseType`.
- (~line 21) `sheetOrder` note is still correct — it orders **worksheets**, not the
  domain type; leave the name `sheetOrder`. Optionally add: "sheet-order = worksheet
  ordering; the domain concept is now `ExpenseType`, value == worksheet name."

### expense-reporter/internal/excel/.memories/ (QUICK or KNOWLEDGE — optional, low priority)
- One line: the worksheet name equals the expense **type** value (Fixas/…); excel
  addressing stays "sheet" (presentation), unaffected by the domain rename.

--------------------------------------------------------------------------------
## Trigger: Plan B (full-path entry routing)
--------------------------------------------------------------------------------

### expense-reporter/.memories/QUICK.md  (~line 20)
- "Next: ... full-path entry routing (DEFERRED, task #5)" → mark **DONE**; replace with
  the two-tier routing summary (typed → full path; type-less → bare-name fallback +
  ambiguous skip).

### expense-reporter/.memories/KNOWLEDGE.md  (~lines 188–196)
- The "ambiguous → dropped from routing / routing logged entries by full path is
  DEFERRED (task #5)" narrative → update to: routing is **two-tier**; full-path routing
  added for typed entries; the bare-name map + sticky `ambiguous` set are **retained as
  the type-less fallback** (NOT removed). The interim guard is no longer "pending" — it
  is the permanent fallback for entries without a type.

### expense-reporter/internal/generate/.memories/QUICK.md  (~lines 14–15)
- The identity/ambiguous "warn+skip" sentence → note routing is two-tier (full path
  when the entry carries a type; bare-name fallback otherwise).

### `[ref:taxonomy-identity-key]` block (in `.claude/plans/taxonomy-identity-key.md`)
- Already listed in Plan B B7 — update the guard language to "retained as fallback,"
  not "removed."

(Also: the two NEW `internal/taxonomy/.memories/*` files above already describe the
post-Plan-B two-tier state in their "Pending (Plan B)" notes — flip those notes to
"done / current" when Plan B lands.)
