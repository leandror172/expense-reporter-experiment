# taxonomy — QUICK

**What:** pure-input package (no excelize) — domain types + loader for the expense
taxonomy + JSONL entry routing. Feeds `internal/generate`.

- Types (types.go): `ExpenseType` (renamed from `ExpenseSheet`, Plan A) → `Category`
  → `Subcat`; `RevenueBlock` for income; `Entry{Item,Day,Value}` is the per-month leaf.
- Loader (loader.go): `LoadTaxonomy(taxonomyPath, entriesPath, targetYear int)`; reads
  `config/taxonomy.json` (keys `types`→cats→subcategories; `incomeCategories`→blocks).
  (JSON key migrated `sheets`→`types` in Plan A Phase R.)
- `entriesPath == ""` = skeleton mode (validation only, routing maps discarded).
- **Multi-year (WS-A/T-11, session 37):** `parseDate` accepts BOTH `DD/MM` and
  `DD/MM/YYYY`, returns `(day, month, year, err)` with `year==0` sentinel for the
  no-year form. `scanEntries` filters by `targetYear`: keep iff
  `entryYear==0 || targetYear==0 || entryYear==targetYear` (year-0 legacy always kept).
  One merged multi-year log can now feed `generate --year N` directly; the 5.R4 per-year
  split is retired-capable (merge script `.claude/scratch/merge_year_logs.py`, gate passed).
  NOTE: the year filter `continue` is AFTER `routeEntry`, so out-of-year type-less entries
  still bump the stderr fallback count before being skipped (cosmetic; matters to T-09 gate).

**Identity = full path** (`[ref:taxonomy-identity-key]`,
`.claude/plans/taxonomy-identity-key.md`): a subcategory is `sheet/category/sub`
(income: `group/label`). Only an exact repeated full path errors. Bare leaf names
repeat legitimately (`Orion`×3, `Gás`, `Aluguel` expense+income).

**Routing (loader.go) — TWO-TIER (Plan B / T-04, landed):**
- `buildSubcategoryMap` returns `(byPath, byName, ambiguous, err)`:
  - `byPath` — full-path key (`expensePath`/`incomePath`) → target. Routes **typed**
    entries; resolves ambiguous leaves to exactly one block. Also the exact-duplicate
    detector (a repeated full path errors).
  - `byName` — bare-name map + sticky `ambiguous` set (`registerTarget`, 3× re-add trap).
    **Retained as the fallback** for **type-less** entries (legacy/auto/batch-auto).
- `scanEntries` reads `{item,date,value,type,category,subcategory}` → `routeEntry`:
  - `type != ""` → `byPath[expensePath(type,cat,sub)]` (miss → warn+skip, exit 0).
  - `type == ""` → `byName[sub]` (today's path, ambiguous-skip preserved).
  - Emits a single stderr summary: "N entries routed via the type-less bare-name fallback".
- Paths joined with **null byte** `\x00` + `expense`/`income` kind prefix (names contain
  `/`, e.g. `Uber/Taxi`).
- **String-equality contract:** a typed entry routes only if its type+category+sub
  byte-match taxonomy.json (accents/case/whitespace count). Wrong spelling → warn+skip,
  never silent misroute (guarded by `TestLoadTaxonomy_TypedEntryWrongPathSkipped`).

**Fallback is TRANSITIONAL, not permanent:** the bare-name tier is a bridge for
type-less entries, to be retired once the classifier emits a type for every entry
(5.R4/RUI-4). The stderr count measures remaining type-less lines.

**Gotchas:**
- Skeleton path builds the maps only to validate, then **discards** them — routing logic
  is exercised only by entry-fed unit tests, not skeleton generation.
- Index into backing slices (`&sheets[i].Cats[j].Subs[k]`) — range copies lose appends.
- Income routing is SCAFFOLDED but UNREACHED: income blocks register in `byPath` via
  `incomePath`, but `routeEntry` only ever builds `expensePath` → no log line reaches an
  income block today. **WS-C (planned, not started)** wires a separate income scan from a
  new `--income-entries` input (extractor `income_log.jsonl` schema) and lifts the model to
  3-level (`Receitas→block→subline`). See `.claude/plans/retire-insertion-keep-generation.md`
  "WS-C task breakdown".
