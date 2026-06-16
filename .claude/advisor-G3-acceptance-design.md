# Advisor — G3 acceptance-test design review (2026-06-11)

Context: pre-implementation review of the G3 design (acceptance tests for `generate-workbook`,
written before the generator exists). Taxonomy source decision (b)+JSON already made with user.
Advisor = Opus subagent.

---

Overall: the shape is right — Ollama-free, deterministic, fixture-driven, dump-compared, born RED. Three issues are blocking, mostly because the design silently absorbs decisions that belong in the spec/fixtures, not in the generator's head.

---

**1. BLOCKING — Taxonomy JSON schema drops fields the builder actually consumes.** The proposed schema (sheets/categories/subcategories + income category/blocks) is a *strict subset* of the scratch structs and loses load-bearing data:

- **`income[].category` vs `Labels.RevenueSheet`.** The scratch `ReceitasBlock` carries `Category: "Receita"` AND the sheet is named `Receitas` (RevenueSheet label). Your schema's `income[].category` collides semantically with the Receita/Renda distinction the spec §4.4 went out of its way to encode (Receita→Revenue, Renda→Income). Pin: is `income[].category` a *block grouping* (drives separators between income categories, spec §3.2 "separators only between income CATEGORIES") or the sheet label? It's the former. Name it so — `incomeCategory` or nest blocks under it — or the generator will conflate it with the Labels-supplied sheet name.
- **Entries embedded vs external is a real fork, not a detail.** Your fixtures put entries in a separate `entries.jsonl` (feedback.ExpenseEntry), but the scratch struct embeds `Months [12][]Entry` *inside* Subcat. The generator therefore needs a **join layer** (entries.jsonl → taxonomy structs) that does not exist yet and is not in any G-step. That join is where routing (#4), date parsing (DD/MM/YYYY → Day+year), and value typing live. Make the join an explicit, separately-tested unit — don't let it hide inside `RunGenerateWorkbook`.
- **Investimentos shell block (spec §4.2) is taxonomy-invisible.** The Listas Investimentos label + manual-entry shell row + `% sobre Receita` is hardcoded structure, not taxonomy-derived. Confirm it's a constant in the generator (it is in scratch) and that the structure-only fixture's expected dump includes it — otherwise the skeleton scenario under-specifies Listas.

**2. BLOCKING — entry→sheet routing has no defined behavior for unknown subcategories.** entries.jsonl rows carry subcategory+category but not sheet; the taxonomy is the only subcategory→sheet map. An entry whose subcategory isn't in the taxonomy is *the* most likely real-world failure (classifier drift, taxonomy edits). This must be pinned **now**, in a scenario, because it's a contract decision the generator can't infer:
- Recommend **warn+skip to stderr, non-zero-free exit** for G3 (Diversos fallback is a product decision that needs the taxonomy to actually have a Diversos bucket — it doesn't). Add a third scenario: `expensesRecordedWithUnmappedSubcategory` → `commandSucceeded()` + `thenOutputWarnsSkipped(subcat)` + dump shows the entry absent. Cheap, deterministic, and it freezes the contract before G2 implements it by accident.
- Also pin **category mismatch**: entry.Category vs taxonomy categoria for the same subcategory — trust taxonomy, ignore entry.Category? Say so.

**3. BLOCKING — full structural dump equality overfits. Compare a normalized subset.** The inspect `Style` includes BgColor/Bold/4 borders, plus ColumnWidths/RowHeights as floats and RowFill. diff.py already documented excelize-serialization noise; float widths/heights are the classic flake source. Asserting *full* deep equality means every cosmetic excelize quirk becomes a red test that has nothing to do with correctness. Define the acceptance contract as **exact equality on the load-bearing subset, ignore the rest**:
- **Exact:** cell Value, Formula, Merge ranges+values, RowType, RowFill, and the *categorical* style facets the spec actually mandates (BgColor, Bold, border booleans).
- **Ignore/tolerate:** ColumnWidths & RowHeights (or compare with epsilon), manifest `source` (already noted), and any field excelize round-trips inconsistently.
- Build this as `verify.WorkbookStructureMatches` taking an explicit ignore-set, not a blanket `reflect.DeepEqual` on the JSON. Otherwise you'll spend G2 fighting the verifier instead of the generator. Keep widths/heights coverage as a *separate, looser* assertion if you want them at all.

---

**4. Bootstrap: adopt your own option (c) — it's strictly better.** Born-RED-with-placeholders means the expected dumps are *meaningless* until G2, so the tests assert nothing real in the interim and the "freeze validated output" step has no independent oracle (freezing the generator's own output against itself is circular; the "cross-check against scratch builder" is the only real check and it's buried in a parenthetical). Invert it:
- **Now (G3):** teach the scratch builder to read the fixture `taxonomy.json` + entries (small change — it already has `buildTaxonomy()` returning exactly these structs; swap the hardcoded body for a JSON loader). Run it, dump, **freeze those dumps as the committed expectation.** The scratch builder is already convergence-verified against the golden master, so its output is a *trustworthy oracle*.
- **G2 becomes a true converge-to-green port:** the new generator must reproduce the frozen dumps the scratch builder produced from the same fixture. That's the whole point of keeping scratch around until G3 is green (plan line 103). This also means the tests are RED-because-generator-absent, not RED-because-expectation-fake — a much healthier red.

**5. Harness conventions — mostly clean, three nits.**
- `then*`-only in `Then:` blocks (KNOWLEDGE 2026-04): your `thenWorkbookStructureMatches` wrapper is correct; make sure the raw `verify.WorkbookStructureMatches` is never called directly in the scenario.
- Given names are event-style: `taxonomyDefined`/`taxonomyAndExpensesRecorded` — `taxonomyDefined` is a state predicate, not an event. Prefer `taxonomyAuthored(fixDir)` / `expensesRecordedUnderTaxonomy(fixDir)`. Minor but the harness police will flag it.
- Fixture format contract (KNOWLEDGE): existing fixtures key on `config.json`+`input.csv`. Your generate fixtures use `taxonomy.json`+`entries.jsonl`+`expected-dump/`. That's a **new fixture sub-format** — fine, but document it in PATTERNS.md `ref:acceptance-fixtures` and the fixture table, or the next agent will assume input.csv.

**6. Timing — confirmed safe, one watch-item.** No RequireOllama, pure file I/O + excelize + JSON compare → well under 5s; no suite-timeout risk. Watch: `inspect.DumpWorkbook` writes to disk per sheet — use `t.TempDir()` for the generated-file dump dir so parallel runs don't collide and cleanup is automatic.

**7. Non-blocking — value typing & SUM sizing need a dedicated assertion, not just "lands in right column."** Scenario 2 ("typed values, SUM ranges sized by max-entries-per-month, headroom 0") bundles three distinct correctness claims. The Aluguel case in the scratch dataset (2 entries Jan, 1 Feb → merged headroom tail above the total) is *exactly* the off-by-one trap for SUM-range sizing and merge spans. Make sure the fixture preserves that asymmetric block and that the expected dump's Formula field pins the SUM range endpoints explicitly — that single cell is the highest-signal assertion in the whole suite.

---

Bottom line: the testing *mechanism* is sound. The risk is all in **decisions being absorbed silently** — schema field semantics (#1), unknown-subcategory routing (#2), and what "matches" means (#3). Pin those three in fixtures/spec before writing the helpers, flip the bootstrap to scratch-as-oracle (#4), and G2 becomes a mechanical converge-to-green instead of a debugging session against a self-referential expectation.
