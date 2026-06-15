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

## Code organization (refactored sessions 30–31, behavior-preserving)
- **styles.go = vocabulary + registration.** Named constructors say WHAT a cell is
  (`dataCell`/`centeredLabel`/`grayBanner`/`columnHeader`/`totalRowCell`/`navyBand`/`fillOnly`)
  over named palette + numfmt constants; `styleRegistrar.family(fill,font)` mints the
  General/currency/percent trio. Never inline a raw `&excelize.Style{...}` in a sheet builder —
  extend the vocabulary. `styleSet` fields are English (MonthCorner; TotalText/TotalTextLeft/
  TotalValue/TotalValueRight — "Text" = non-currency total-row cells, not "Date").
- **File homes = domain, not first caller.** `util.go` = pure string/formula/ref helpers, no
  excelize (`cell`, `sheetRef`, `needsQuote`, `sumList/Range`, `lower`, `atoi`). `data_sheet.go`
  = the data-sheet vocabulary shared by the expense sheets AND Receitas (`writeMonthHeader`,
  `writeTotalRow(Opt)`, `writeSeparator`, `mergeCategoryLabel`, `freezeC3`, plus the unified
  `calculateBlockRows` and `writeDataBand`).
- **One sizing/band path for both sheet kinds:** `calculateBlockRows(row, maxEntries)` and
  `writeDataBand(..., rowHeight, lastCol)` serve expense + revenue; the ONLY behavioral
  difference is row height (12.75 expense / 15 revenue). `Subcat` and `RevenueBlock` both expose
  `Months [12][]Entry` + `MaxEntries()`.
- **Method-extraction convention:** sheet-builder bodies read as named delegated steps ≤~15 lines
  (exemplars: summary_sheet.go `revenueSection`/`balanceBlock`); inline step comments promote to
  doc comments.
- **Package stays flat** (Go idiom; styleSet/layoutRegistry/Labels/domain types too coupled to
  subpackage). One split penciled: `internal/taxonomy` (pure input layer) alongside the T-02
  real-taxonomy export — non-trivial: `taxonomy.go` mixes the domain types (used pervasively)
  with mutable RENDER config (`dataYear`/`headroomRows`/`perGroupPctRows`, set by `Generate()`,
  read by builders) that must relocate to `generate` first.
