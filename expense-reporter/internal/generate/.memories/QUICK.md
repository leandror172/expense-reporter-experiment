# internal/generate — quick notes

## Purpose
Workbook generator (spec v2: `.claude/plans/workbook-generator-spec.md` — the design
authority). Builds a complete expense workbook (Listas de itens + Receitas + one sheet per
expense group) from a taxonomy JSON + optional entries JSONL. Regenerate-don't-insert:
the workbook is a projection of the data, never an insertion target.

## Entry points
- `Generate(Options{TaxonomyPath, EntriesPath, OutPath, Year, Headroom})` — CLI:
  `generate-workbook -o out.xlsx --taxonomy t.json [--entries log.jsonl --year N --headroom N]`
- `LoadTaxonomy(taxonomyPath, entriesPath)` — join layer (loader.go): DD/MM dates (year from
  Options), unknown subcategory → warn+skip exit 0, taxonomy authority on category mismatch,
  duplicate subcategory names = error.

## Key facts
- **Identifiers English, strings pt-BR** — all user-visible text in `Labels` (labels.go);
  `Labels.RevenueSheet` ("Receitas") appears inside cross-sheet formulas → schema identifier,
  renaming it breaks existing references.
- **Sheet order = taxonomy order** via `layoutRegistry.sheetOrder` — never hardcode the
  4-sheet list (a hardcoded order once emitted invalid D0/E0 refs for smaller taxonomies).
- **Block sizing** = max-entries-per-month + headroom (default 0); single-row SUMs valid.
- **excelize gotchas:** SetCellFormula takes NO leading `=`; stale-formula fix =
  `UpdateLinkedValue()` + `SetCalcProps(FullCalcOnLoad)`.
- **Acceptance contract:** `test/fixtures/generate-basic/expected-dump-*` (oracle-frozen);
  any deliberate output change requires re-freezing + manually reviewing the dump delta.
- Provenance: port of `.claude/scratch/template-builder/` (SUPERSEDED), converged to the
  user-blessed data-bearing golden master; Phase B fake dataset lives in
  `taxonomy_fixture_test.go`.
