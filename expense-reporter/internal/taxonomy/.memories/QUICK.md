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
