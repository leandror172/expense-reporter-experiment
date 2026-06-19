# internal/taxonomy — Knowledge (Semantic Memory)

*Accumulated decisions for the workbook generator's input layer. Read on demand.*

## Package boundary (2026-06-16)
Extracted from `internal/generate` (commits 07f395a + 21c6d4e) as a **pure input
layer** — domain types (`Entry`/`Subcat`/`Category`/`ExpenseSheet`/`RevenueBlock`,
each with `MaxEntries()`) + the taxonomy/entries loader. Imports **nothing** from
`generate` (`go list -deps ./internal/taxonomy` is cycle-free).
**Prerequisite that made the split possible:** `generate`'s mutable RENDER config
(`dataYear`/`headroomRows`/`perGroupPctRows`, set by `Generate()`, read by builders)
was relocated INTO `generate` first — it had been mixed with the domain types in the
old `taxonomy.go`. Generate builders reference `taxonomy.X` by full qualification
(no alias shim).
**Rationale:** input parsing/validation is a separate concern from rendering; a pure,
excelize-free package is independently testable and reusable (e.g. future DB ingestion).
**Implication:** the rest of the generator package stays FLAT — this is the one split.

## Full-path identity key (2026-06-16/17) — authority: `.claude/plans/taxonomy-identity-key.md`
A subcategory's identity is its **full path** (sheet/category/sub; income:
group/label), NOT its bare leaf name. Only an exact repeated full path is a validation
error; cross-path repeats are legal — real data repeats leaves (`Orion` ×3 across Pet
blocks; `Aluguel` as both a Fixas expense and a Receitas income block).
**Routing safety (interim):** routing is still keyed by bare name, but a bare name
shared by >1 full path is *ambiguous* and dropped from the routing map; an entry naming
it gets a distinct warn+skip (exit 0), never a silent misroute. `registerTarget` keeps
the ambiguous set **sticky** so a name appearing 3+ times can't re-add itself; full
paths join on a **null byte** because real names contain `/` (`Uber/Taxi`,
`Óleo/flor cannabis`).
**Rationale:** real-data identity is hierarchical; bare leaf names are not unique. The
old global bare-name guard rejected the real taxonomy outright.
**Implication:** routing logged entries *by* full path is **DEFERRED (T-04)** — it
changes the entry contract (`expenses_log.jsonl` carries only a bare `subcategory`),
the classifier, `scanEntries`, and entry-fed fixtures. Until then the ambiguity guard
is the safety net.

## Validation vs routing share one builder
`buildSubcategoryMap` is dual-purpose: it validates (full-path duplicate detection)
AND returns the routing map. The skeleton path (`LoadTaxonomy(taxonomyPath, "")`) calls
it only for validation and **discards the map** — so ambiguity/routing behavior is NOT
exercised by skeleton generation. Real coverage is an entry-fed unit test
(`TestLoadTaxonomy_AmbiguousEntrySkipped`, a 3× repeat + a matching entry).

## Data
Real `config/taxonomy.json` (112 subcategories, personal categories) is **gitignored**;
the committed `test/fixtures/generate-basic/taxonomy.json` fixture (unique names) is the
test input. Fidelity of a hand-authored taxonomy is verified by a CSV↔JSON
symmetric-difference (count + no-dup checks are necessary but not sufficient).
