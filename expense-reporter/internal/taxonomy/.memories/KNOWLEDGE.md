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
warn+skips (never silent misroute), so emitters MUST produce taxonomy-exact strings.

**5.R4 LANDED — fallback retirement (T-09) reassessed.** The classifier now emits type
(auto/batch-auto/apply populate `ExpenseEntry.Type`) and the historical logs were
backfilled with type ([[project_workbook_extraction_5r4]]) — so the type-less EXPENSE
surface is ~0. Retiring the bare-name tier is now gated NOT on the classifier but on:
(1) **income is structurally type-less** and routes via the SAME `byName` map — it needs
its own `incomePath` route before `byName` can be deleted; (2) `add`/`correct` still
emit type-less (but only to classifications.jsonl, not generator input); (3) confirm the
stderr fallback count is ~0 on real typed data.

**Pending "year adaptation":** `parseDate` requires exactly `DD/MM` (2 parts), making
expenses_log year-implicit (year comes from generate `--year`). Multi-year data is split
into per-year logs as a workaround. Future: accept `DD/MM/YYYY` + per-entry year so one
multi-year log routes directly.

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
