# Plan B — Full-path entry routing (T-04)

**Goal:** route logged entries into the generated workbook by their **full path**
(type / category / subcategory) instead of the bare subcategory leaf name, so that
entries whose leaf name is ambiguous (e.g. `Orion`, `Gás`, `Dentista`) populate the
correct block. Removes the interim ambiguity guard shipped in T-02.

**Decision authority:** `.claude/plans/taxonomy-identity-key.md` §6 / `[ref:taxonomy-identity-key]`.
Identity = full path; this plan makes routing use the full key the entry now carries.

**Hard dependency:** Plan A Phase F must be merged first — entries can only be routed
by type once `ExpenseEntry` carries a `type` field. Until then, `scanEntries` has no
type to route on and the ambiguity guard must stay.

**Out of scope (separate, later):** making the **classifier** emit the type for
*auto* entries (option 1, full-path label). Plan B handles routing of entries that
already carry a type (the apply/review path + backfilled logs).

> **CRITICAL — type-less entries are the majority, not an edge case.**
> `auto.go:172` and `batch_auto.go:296` write **type-less** lines to
> `expenses_log.jsonl` (the file the generator reads), and all ~355 existing log
> lines are type-less. These are the high-confidence auto-inserted expenses whose
> whole point is to bypass review. **Plan B must NOT drop them.** Therefore the
> bare-name routing map + `ambiguous` set are **retained as a fallback** for
> type-less entries — they are not deleted. Full-path routing is *added* for entries
> that carry a type; bare-name routing (with its ambiguous-skip) is *kept* for those
> that don't. The end state is a strict superset of today's behavior: unambiguous
> type-less entries keep routing, and typed ambiguous entries now route correctly.

**Advisor review:** required before implementing — this changes the entry contract and
deletes a safety guard.

---

## Current behavior (the thing being changed)

`internal/taxonomy/loader.go`:
- `buildSubcategoryMap` builds `map[bareName]subcatTarget` + an `ambiguous` set; a
  bare name resolving to >1 full path is dropped from the map and marked ambiguous
  (sticky, via `registerTarget`).
- `scanEntries` parses each JSONL line into `{item,date,value,subcategory}` and looks
  up `subcatMap[entry.Subcategory]`. Miss → `warnUnroutable` (distinguishes ambiguous
  vs unknown), skip, exit 0.

The entry struct in `scanEntries` (lines 128–133) reads **only** `subcategory` — no
type, no category. That is why ambiguous names cannot be routed.

---

## Target behavior

**Two-tier routing — keep both maps:**
1. Entry carries a `type` → route on the **full-path key**
   (`expense\x00type\x00category\x00subcategory`, the existing `expensePath` format).
   Resolves to exactly one block; ambiguity does not apply.
2. Entry has **no** `type` (legacy/auto/batch-auto lines) → fall back to the existing
   **bare-name** map, which still skips genuinely-ambiguous names. This is byte-for-byte
   today's behavior for those entries — no regression on the auto path.

The `ambiguous` set is **still required** — the fallback path needs it to know which
bare names are unsafe to route. Do **not** delete it.

Income entries: route on `incomePath(category, label)`. (If the entry contract does
not yet carry income identity, keep income on its current path and note it.)

---

## Steps

### B1. Build BOTH routing maps (keep bare-name, add full-path)
In `internal/taxonomy/loader.go`, `buildSubcategoryMap` returns **both** maps plus the
ambiguous set:
- Keep the existing bare-name `result` map + `ambiguous` set + `registerTarget`
  exactly as today (the type-less fallback depends on them — see "Target behavior").
- Add a second map `byPath := map[string]subcatTarget` keyed by
  `expensePath(type,cat,sub)` / `incomePath(cat,label)` — the same strings already
  computed for `seen`. (You can populate `byPath[path] = target` right where `seen[path]`
  is set.)
- Keep the **exact-duplicate** check (a repeated full path is still a validation error).
- New signature:
  `buildSubcategoryMap(...) (byPath, byName map[string]subcatTarget, ambiguous map[string]bool, error)`.
- Do **NOT** remove `registerTarget` or the `ambiguous` set.

### B2. Update `LoadTaxonomy` + `loadEntries` plumbing
- `LoadTaxonomy` (lines 22, 31): receive all three (`byPath`, `byName`, `ambiguous`);
  pass all three down.
- `loadEntries` (line 109) + `scanEntries` (line 121): accept `byPath`, `byName`,
  `ambiguous`.

### B3. Entry struct + two-tier lookup in `scanEntries`
- Extend the inline entry struct (lines 128–133) to read the full path:
  ```go
  var entry struct {
  	Item        string  `json:"item"`
  	Date        string  `json:"date"`
  	Value       float64 `json:"value"`
  	Type        string  `json:"type"`        // expense type (Plan A field); may be ""
  	Category    string  `json:"category"`
  	Subcategory string  `json:"subcategory"`
  }
  ```
- Resolve in two tiers:
  - **`entry.Type != ""`** → `key := expensePath(entry.Type, entry.Category, entry.Subcategory)`;
    `subcat, exists := byPath[key]`. On miss → `warnUnroutable(... not in taxonomy ...)`.
  - **`entry.Type == ""`** (legacy/auto/batch-auto) → `subcat, exists := byName[entry.Subcategory]`
    (today's path). On miss → `warnUnroutable(item, subcategory, ambiguous[entry.Subcategory])`
    (preserves the ambiguous-vs-unknown distinction and the warn+skip+exit-0 safety).

### B4. `warnUnroutable` — keep both branches
No change to its signature/branches: the type-less fallback still needs the ambiguous
branch. (Optionally add a distinct message for the typed "full path not in taxonomy"
miss.)

### B5. Tests — `internal/taxonomy/loader_test.go`
- `TestLoadTaxonomy_SamePathDuplicate` — keep (exact full-path repeat still errors).
- `TestLoadTaxonomy_CrossPathDuplicateAllowed` — keep (same bare name, two paths,
  loads fine).
- `TestLoadTaxonomy_AmbiguousEntrySkipped` — **keep** (a 3× repeated bare name + a
  **type-less** entry → still skipped via the bare-name fallback; proves the fallback
  ambiguous-skip survives).
- Add `TestLoadTaxonomy_AmbiguousEntryRoutedByFullPath` — a 3× repeated bare name +
  three entries each carrying a distinct `type`, asserting each routes to the
  **correct** one of the three blocks (the new capability).
- Add `TestLoadTaxonomy_TypelessUnambiguousEntryRoutes` — a unique bare name + a
  type-less entry → routes (guards the no-regression-on-auto-path promise).
- `TestLoadTaxonomy_WithEntries` / `_UnmappedSubcategory`: keep their existing
  type-less entries working via the fallback; add at least one typed entry covering
  the `byPath` tier.

> **Do not** make every fixture entry typed — keep a mix of typed and type-less
> entries, or the suite will pass while the type-less fallback is silently broken
> (the exact gap this revision fixes).

### B6. Entry-fed fixtures
- Keep most `test/fixtures/generate-basic/entries.jsonl` lines **as-is (type-less)** so
  the fallback tier stays exercised end-to-end; add `type`+`category` to only a few
  lines to cover the `byPath` tier. (Do not type every line — that would mask a broken
  fallback, the same trap as B5.)
- `entries-with-unmapped.jsonl`: leave the unmapped case type-less.
- Re-freeze the `generate-basic` oracle dump **only if** routing changes which cells
  populate. Type-less unambiguous entries route identically to today, so output should
  be byte-identical — verify first; only re-freeze if a real, reviewed difference
  appears.

### B7. Documentation
- Update `.claude/plans/taxonomy-identity-key.md` §5/§6: full-path routing landed for
  typed entries; the interim bare-name guard is **retained as the fallback** for
  type-less entries (not removed); move task #5/T-04 to done.
- Update `[ref:taxonomy-identity-key]` block to reflect two-tier routing (guard still
  active for type-less entries).
- `internal/taxonomy/.memories/` QUICK/KNOWLEDGE: note routing is two-tier
  (full-path when typed, bare-name fallback otherwise).

### B8. Verify
```
cd expense-reporter && gofmt -l . && go vet ./... && go build ./... && go test ./...
./run-acceptance.sh
```
- All green.
- `generate-basic` oracle dumps byte-identical unless an intentional routing change is
  reviewed and re-frozen.
- Manually: a `config/taxonomy.json` + an `expenses_log.jsonl` containing an `Orion`
  entry with `type:"Variáveis"` populates the Variáveis/Pets block (and **not** Fixas
  or Extras).

---

## Done-criteria
- Two-tier routing: typed entries route by full path; type-less entries route by the
  retained bare-name map (with ambiguous-skip preserved).
- Ambiguous leaf names route correctly when the entry carries a type.
- Type-less unambiguous entries (auto/batch-auto/legacy) still route — **no regression
  on the auto path** (verified by a dedicated test + a type-less fixture line).
- Type-less ambiguous entries still warn+skip (exit 0) — no silent misroute.
- Exact full-path duplicate still a validation error.
- Docs + memories updated; tests added; fixtures keep a typed/type-less mix.

## Files touched (checklist)
- `internal/taxonomy/loader.go` (buildSubcategoryMap returns both maps + ambiguous,
  LoadTaxonomy, loadEntries, scanEntries two-tier lookup; **keep** registerTarget +
  ambiguous set)
- `internal/taxonomy/loader_test.go`
- `test/fixtures/generate-basic/entries.jsonl`, `entries-with-unmapped.jsonl`
- `.claude/plans/taxonomy-identity-key.md`, `.claude/index.md` ref block
- `internal/taxonomy/.memories/{QUICK,KNOWLEDGE}.md`

## Follow-on (not this plan)
Classifier emits type for auto entries (option 1 full-path label) — uses the
type-labeled training data (607/694 examples already carry it in `source`) + the
backfilled gold corrections from Plan A. Tracked with 5.R4 / RUI-4.
