# expense-reporter/ — Quick Memory

*Working memory for the Go application. Injected into agents. Keep under 30 lines.*

## Status
Full CLI operational: add, batch, classify, auto, batch-auto, correct, version commands.
190+ unit tests passing. JSON output mode (`--json`) on classify/auto/add.
Few-shot injection (5.7) complete — keyword-based example selection active.
MCP prediction feedback (5.9+) complete — `add` accepts `--predicted-subcategory` and related
flags; writes `confirmed`/`corrected` instead of `manual` when prediction context is present.
`auto --json` emits `classification_id` for the caller to pass back to `add`.
Workbook-mapping done (sessions 26-27): `cmd/workbook-inspect` JSON dump -> generator spec v2
(`.claude/plans/workbook-generator-spec.md`) + scratch builder converged to golden master.
Phase B (data validation) IN PROGRESS — scratch builder gains entries, typed values, per-group
percent rows, and a centralized English-identifier `Labels` struct (pt-BR values; i18n config
loader deferred); converges against `template-data.xlsx`. Spec §4.4.
Next: `generate-workbook` command - port `.claude/scratch/template-builder/` into
`internal/generate` + `cmd/` (carry the `Labels` struct over), lift inspect core into
`internal/inspect`, acceptance tests first;
brief: `.claude/plans/workbook-generator-implementation-plan.md`. Then TF-IDF (5.R1).

## Structure
```
cmd/expense-reporter/
  main.go              # Entry point
  cmd/                 # Cobra subcommands: add, auto, batch, batch-auto, classify, correct, version
cmd/workbook-inspect/  # Standalone tool — JSON structural dump of a workbook (workbook mapping L1)
internal/
  batch/               # CSV reading, installment expansion, progress bars, report generation
  classifier/          # LLM classification — Ollama client, few-shot, decision logic
  cli/                 # CLI formatting helpers (confidence bars)
  config/              # config.json loader
  excel/               # Excelize wrapper — read/write workbook, column mapping
  feedback/            # JSONL persistence (classifications + expense log)
  logger/              # Debug logging
  models/              # Domain types: Expense, BatchError, ClassifiedExpense
  parser/              # Semicolon-delimited expense string parser
  resolver/            # Fuzzy subcategory matching against reference sheet
  workflow/            # Orchestration: parse → resolve → insert pipeline
pkg/utils/             # Currency parsing, date formatting, string building
config/config.json     # Runtime config (workbook path, exclusion list, log paths)
```

## Key Rules
- **Cobra pattern** — one `.go` file per subcommand in `cmd/`
- **Table-driven tests with testify** — `assert`/`require`, not stdlib-only
- **Brazilian format everywhere** — DD/MM/YYYY, comma decimal, BRL
- **Error wrapping** — `fmt.Errorf("context: %w", err)`, never bare returns
- **Installment notation** — "99,90/3" means 3 monthly payments, expanded at insert time

## Deeper Memory → KNOWLEDGE.md
- **Batch pipeline** — single-open optimization, installment expansion, rollover handling
- **Command design** — classify (read-only) → auto (single insert) → batch-auto (CSV batch)
- **JSON output mode** — structured output for MCP integration
