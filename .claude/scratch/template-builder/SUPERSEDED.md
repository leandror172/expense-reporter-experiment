# SUPERSEDED (2026-06-11, Phase G4)

This scratch builder was the Phase A/B reference implementation of spec v2 and the
oracle that froze the `generate-basic` acceptance fixture dumps. It has been ported to
`expense-reporter/internal/generate` (+ the `generate-workbook` command); all its unit
tests (builder math, SUM regression guard, loader) were ported with it, and the G3
acceptance suite is green against the frozen dumps.

Do not extend this module. It is kept only because the Phase A/B convergence reports
(`.claude/workbook-template/*.md`) reference its history. The hardcoded Phase B fake
dataset lives on as `internal/generate/taxonomy_fixture_test.go`.
