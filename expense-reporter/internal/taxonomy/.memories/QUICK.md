# internal/taxonomy — quick notes

## Purpose
Pure **input layer** for the workbook generator — split from `internal/generate`
(2026-06-16, commits 07f395a + 21c6d4e). Holds the domain types and the taxonomy/
entries loader; imports nothing from `generate` (cycle-free, `go list -deps` clean).

## Contents
- **types.go** — `Entry`, `Subcat`, `Category`, `ExpenseSheet`, `RevenueBlock` +
  `MaxEntries()`. Used pervasively by generate builders via `taxonomy.X` qualification.
- **loader.go** — `LoadTaxonomy(taxonomyPath, entriesPath)`: parse taxonomy JSON
  (spec §1.1), validate, route entries. DD/MM dates (year from generate Options),
  unknown subcategory → warn+skip exit 0, taxonomy authority on category mismatch.
- **loader_test.go / taxonomy_fixture_test.go** — input-layer tests + Phase-B fixture.

## Identity key (the important rule) — `[ref:taxonomy-identity-key]`
A subcategory's identity is its **full path** (sheet/category/sub; income: group/label),
NOT the bare leaf name. Only an exact repeated full path is a validation error;
cross-path repeats are legal (real data repeats names: `Orion` ×3 across Pet blocks,
`Aluguel` as expense + income). A bare name shared by >1 full path is **ambiguous** →
dropped from routing, entry warn+skip (exit 0), never silent misroute.
- `registerTarget` keeps the ambiguous set **sticky** (guards the 3× re-add trap).
- Full paths join on a **null byte** — real names contain `/` (`Uber/Taxi`).
- Full decision record: `.claude/plans/taxonomy-identity-key.md`.

## Deferred (task #5)
Routing entries by full path (not bare name). Changes the entry contract
(`expenses_log.jsonl` carries only a bare `subcategory` today) + the classifier +
`scanEntries` + fixtures. Until then the ambiguity guard keeps entry-fed generation safe.

## Data
Real `config/taxonomy.json` (112 subs, personal categories) is **gitignored**; the
committed fixture `test/fixtures/generate-basic/taxonomy.json` is the test input.
