# taxonomy — KNOWLEDGE

**Why full path is the identity.** Real taxonomy repeats leaf names across sheets/cats
(7 bare-name collisions). Category-qualification is insufficient: `Pets` is a category
on both Variáveis and Extras, so `Pets/Orion` still collides — the **sheet dimension is
load-bearing**. Full three-segment path is the only safe identity.

**Two collision classes:** same-sheet collisions (`Gás`, `Produtos` in Variáveis) were
genuine data bugs fixed at source while authoring `config/taxonomy.json`; cross-sheet /
income-vs-expense collisions (`Orion`, `Dentista`, `Aluguel`) are legal and only broke
the old bare-name validation.

**Two-tier routing landed (Plan B / T-04).** Typed entries route by full path
(`byPath`), which resolves ambiguous leaves to exactly one block. Type-less entries
fall back to the retained bare-name map (`byName`) with its ambiguous-skip. Plan A
(persist type) landed first so the entry contract could carry type. The bare-name
fallback is **transitional, not permanent**: `scanEntries` logs a one-line count of
type-less fallbacks so the remaining surface is measurable. Routing is value-equality on
the full path — a typed entry whose type/category/sub don't byte-match the taxonomy
warn+skips (never silent misroute), so emitters MUST produce taxonomy-exact strings
(accents/case/whitespace count; guarded by `TestLoadTaxonomy_TypedEntryWrongPathSkipped`).

**Routing mechanics (loader.go, consolidated from QUICK.md 2026-07-01).**
`buildSubcategoryMap` returns `(byPath, byName, ambiguous, err)`: `byPath` keys are
`expensePath`/`incomePath` (segments joined with null byte `\x00` + kind prefix, because
names contain `/` — `Uber/Taxi`, `Alimentação / Limpeza`); `byPath` is also the
exact-duplicate detector. `byName` is the bare-name map with a sticky `ambiguous` set
(`registerTarget`, 3× re-add trap). `scanEntries` reads
`{item,date,value,type,category,subcategory}` → `routeEntry`: `type != ""` →
`byPath[expensePath(...)]` (miss → warn+skip, exit 0); `type == ""` → `byName[sub]`
(ambiguous-skip preserved); single stderr summary "N entries routed via the type-less
bare-name fallback". Skeleton mode builds the maps only to validate, then discards them.

**Full-path helpers (path.go, T-13 session 41).** `BuildPathMap(sheets)`→`PathMap` with
`Enum()` (112 `Type/Category/Subcategory` enum strings for the classifier's structured
output), `Split(path)` — **reverse-map lookup, never split on `/`** — and
`PathFor(typ,cat,sub)` (forward, map-validated). `PathEnum(sheets)` is the convenience.
`ResolveLeaf(sheets,sub,typeHint)`→`(typ,cat,err)`: unique→from name, ambiguous→needs
hint else `ErrLeafAmbiguous`, absent→`ErrLeafNotFound`. `TypesForLeaf`/`CategoryForLeaf`
back `add`/`correct` messages. The classifier depends on this package (Option 2B) — no
import cycle (taxonomy is a leaf). Tested via a committed synthetic fixture.

**5.R4 LANDED — fallback retirement (T-09) reassessed.** The classifier now emits type
(auto/batch-auto/apply populate `ExpenseEntry.Type`) and the historical logs were
backfilled with type ([[project_workbook_extraction_5r4]]) — so the type-less EXPENSE
surface is ~0. Retiring the bare-name tier is now gated NOT on the classifier but on:
(1) **income is structurally type-less** and routes via the SAME `byName` map — it needs
its own `incomePath` route before `byName` can be deleted; (2) `add`/`correct` still
emit type-less (but only to classifications.jsonl, not generator input); (3) confirm the
stderr fallback count is ~0 on real typed data.

**Year adaptation LANDED (WS-A/T-11, session 37 — supersedes the old "pending" note).**
`parseDate` accepts BOTH `DD/MM` and `DD/MM/YYYY`, returning `(day, month, year, err)`
with `year==0` sentinel for the no-year form. `scanEntries` filters by `targetYear`:
keep iff `entryYear==0 || targetYear==0 || entryYear==targetYear` (year-0 legacy always
kept). One merged multi-year log can feed `generate --year N` directly; the 5.R4
per-year split is retire-capable (merge script `.claude/scratch/merge_year_logs.py`,
byte-identical gate passed). Promotion of the merged log to canonical is the user's call.

**Income routing (WS-C, session 38 — DONE).** 3-level model:
`RevenueBlock{Category,Block,Label,Months}` — one block == one subline leaf; `incomePath`
is 3-segment. Loader parses BOTH legacy flat `blocks:["Salário"]` (Block==Label) and
nested `blocks:[{block,sublines:[…]}]` via a custom `rawIncomeBlock.UnmarshalJSON`.
A SEPARATE income scan (`loadIncomeEntries`/`scanIncomeEntries`) reads the extractor's
`income_log.jsonl` schema (`income_category`=block, `income_label`=leaf, `item_note`=Item),
routes via a `buildIncomeIndex` block+label index (category implicit), keeps values
SIGNED. Wired through `LoadTaxonomy(..., incomeEntriesPath, ...)` + `generate-workbook
--income-entries`. **Dateless income is skipped with a LOUD stderr count** (not fatal) —
a silent skip would leave a near-empty Receitas reading as success
(`TestIncomeMissingDateSkipped`). Real WS-0b income is month-stamped (blank day →
`01/MM/YYYY`), so the count is ~0 on real data.

**The two-taxonomy-sources debt is RESOLVED (T-13, session 41).** Previously:
- `config/taxonomy.json` — sheet→cat→sub tree, generator side.
- `feature_dictionary_enhanced.json` `category_mapping` — flat sub→cat, classifier side.
The classifier never saw the sheet dimension, so it couldn't emit type. **T-13 makes the
classifier depend on THIS package** and predict the full path against `config/taxonomy.json`
(via `PathEnum`/`Split`). `classifier.LoadTaxonomy` (the flat feature-dict map) is deleted;
the feature dict is now keyword-only (few-shot selection), no longer a category authority.
NOTE: `BuildTypeIndex`/`LookupType` (the old `(cat,sub)→type` reverse index) is unused by
production code — kept + tested; a future WS-E cleanup may remove it.

**Real file is gitignored** (`config/taxonomy.json` reveals personal categories); the
committed test input is `test/fixtures/generate-basic/taxonomy.json`. Fidelity of a
hand-authored taxonomy is checked by CSV↔JSON symmetric-difference, not "112 subs + no
dups" (necessary but not sufficient).
