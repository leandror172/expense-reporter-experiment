# taxonomy — QUICK

**What:** pure-input package (no excelize) — domain types + loader for the expense
taxonomy + JSONL entry routing. Feeds `internal/generate`; `internal/classifier`
depends on it (path enum). Mechanics + rationale → KNOWLEDGE.md.

- Types: `ExpenseType` → `Category` → `Subcat`; `RevenueBlock` (3-level income);
  `Entry{Item,Day,Value}` per-month leaf.
- Loader: `LoadTaxonomy(taxonomyPath, entriesPath, incomeEntriesPath, targetYear)`;
  `entriesPath==""` = skeleton mode (validation only).
- **Identity = full path** (`[ref:taxonomy-identity-key]`): `type/category/sub`. Only an
  exact repeated full path errors; bare leaves repeat legally.
- **Routing is TWO-TIER:** typed → `byPath` (byte-match or warn+skip, never silent
  misroute); type-less → transitional `byName` fallback (ambiguous-skip, stderr count).
  Income routes via a separate block+label index, values SIGNED, dateless = loud skip.
- **path.go (T-13):** `BuildPathMap`→`Enum()` (112 enum strings), `Split(path)`
  reverse-map lookup — **NEVER parse on `/`** (names contain `/`); `PathFor`,
  `ResolveLeaf(sub,typeHint)`, `TypesForLeaf`.
- **Multi-year (WS-A):** `parseDate` accepts `DD/MM` and `DD/MM/YYYY`; `scanEntries`
  filters by `targetYear` — one merged log can feed `generate --year N`.

**Gotchas:** index into backing slices (range copies lose appends); paths joined with
`\x00` + kind prefix; year filter runs AFTER `routeEntry` (out-of-year type-less
entries still bump the fallback count — matters to the T-09 gate).
