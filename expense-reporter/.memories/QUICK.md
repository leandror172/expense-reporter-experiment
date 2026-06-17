# expense-reporter/ — Quick Memory

*Working memory for the Go application. Injected into agents. Keep under 30 lines.*

## Status
Full CLI operational: add, batch, classify, auto, batch-auto, correct, review, apply,
**generate-workbook**, version. 220+ unit tests. JSON output mode on classify/auto/add.
Few-shot injection (5.7) + MCP prediction feedback (5.9+) complete.
**Workbook generator COMPLETE (Phase G, session 29):** `internal/inspect` = structural-dump
core (lifted from cmd/workbook-inspect, now a thin wrapper); `internal/generate` = spec-v2
builder port — `Generate(Options)`, taxonomy JSON (spec §1.1) + entries JSONL loader (join
layer: DD/MM dates, unknown subcat → warn+skip exit 0, taxonomy authority), English
identifiers (Revenue*/summary*/balance*; pt-BR strings only in `Labels`), sheet order derived
from taxonomy via registry `sheetOrder` (hardcoded 4-sheet order bug fixed).
Acceptance: `test/fixtures/generate-basic/` with oracle-frozen expected dumps;
`verify.WorkbookStructureMatches` (normalized subset). Scratch builder SUPERSEDED.
**`internal/taxonomy` extracted (2026-06-16, PR #27 merged):** pure input layer (domain types
+ loader) split from generate; identity = **full path**, not bare leaf name —
`[ref:taxonomy-identity-key]`. Real `config/taxonomy.json` authored (112 subs, **gitignored**).
Next: year rollover; full-path entry routing (DEFERRED, task #5); TF-IDF (5.R1).

## Structure
```
cmd/expense-reporter/
  main.go              # Entry point
  cmd/                 # Cobra subcommands: add, auto, batch, batch-auto, classify, correct, version
cmd/workbook-inspect/  # Thin wrapper over internal/inspect (JSON structural dump)
internal/
  batch/               # CSV reading, installment expansion, progress bars, report generation
  classifier/          # LLM classification — Ollama client, few-shot, decision logic
  cli/                 # CLI formatting helpers (confidence bars)
  config/              # config.json loader
  excel/               # Excelize wrapper — read/write workbook, column mapping
  feedback/            # JSONL persistence (classifications + expense log)
  generate/            # Workbook generator — spec v2: taxonomy+entries → full workbook
  taxonomy/            # Pure input layer — domain types + taxonomy/entries loader (split from generate)
  inspect/             # Structural dump core (values/formulas/styles/merges/rowType)
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
