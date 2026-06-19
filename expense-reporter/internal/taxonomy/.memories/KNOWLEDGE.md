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
