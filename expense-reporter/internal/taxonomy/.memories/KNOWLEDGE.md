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
(legacy/auto/batch-auto) fall back to the retained bare-name map (`byName`) with its
ambiguous-skip — behavior unchanged for them. Plan A (persist type) landed first so the
entry contract could carry type. The bare-name fallback is **transitional, not
permanent**: it's a bridge to be retired once the classifier emits a type for every
entry (5.R4/RUI-4); `scanEntries` logs a one-line count of type-less fallbacks so the
remaining surface is measurable. Routing is value-equality on the full path — a typed
entry whose type/category/sub don't byte-match the taxonomy warn+skips (never silent
misroute), so the classifier MUST emit taxonomy-exact strings when 5.R4 lands.

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
