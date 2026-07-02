# internal/generate — quick notes

## Purpose
Workbook generator (spec v2: `.claude/plans/workbook-generator-spec.md` — design
authority). Builds a complete workbook (Listas + Receitas + one sheet per expense type)
from taxonomy JSON + optional entries JSONL. **Regenerate-don't-insert:** the workbook
is a projection of the data, never an insertion target.

## Entry point
`Generate(Options{TaxonomyPath, EntriesPath, OutPath, Year, Headroom})` — CLI
`generate-workbook`. Loading/routing lives in `internal/taxonomy` (see its memories).

## Key facts
- **Identifiers English, strings pt-BR** — user-visible text in `Labels` (labels.go);
  `Labels.RevenueSheet` appears inside cross-sheet formulas → renaming breaks refs.
- **Sheet order = taxonomy order** via `layoutRegistry.sheetOrder` — never hardcode.
- **excelize:** SetCellFormula takes NO leading `=`; stale-formula fix =
  `UpdateLinkedValue()` + `SetCalcProps(FullCalcOnLoad)`.
- **Oracle-frozen acceptance:** output changes re-freeze BOTH generate-basic AND
  generate-income dumps (shared accumulating `b.row` counter couples them).
- **Never inline `&excelize.Style{...}`** — extend the styles.go vocabulary.
  Conventions → KNOWLEDGE.md "Code Organization Conventions".
- Income is 3-level, symmetric with expenses (WS-C) — KNOWLEDGE.md "Income".
