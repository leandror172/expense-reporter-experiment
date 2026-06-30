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
**Plan A (T-05) + Plan B (T-04) IMPLEMENTED (2026-06-19, PRs #29 + #30):** expense **type**
now persisted end-to-end (feedback.Entry/ExpenseEntry carry `Type`, set on the apply path);
`ExpenseSheet`→`ExpenseType` rename + JSON key `sheets`→`types`/`sheet`→`type` (legacy
read-compat in apply); generator routes typed entries by full path (two-tier: `byPath` for
typed, retained bare-name `byName` + ambiguous-skip for type-less — a **transitional**
fallback), NFC-normalized keys. `backfill-type.py` recovers type into existing logs.
Next: classifier full-path label (5.R4/RUI-4 — closes the type-less producer gap); year
rollover; TF-IDF (5.R1). Real-data proof of A→B chain pending Bf3 (export+backfill check).
**Session 42 (2026-06-29) — T-13 PR #36 code-review follow-ups (3 commits on `feat/t13-classifier-full-path`):**
(1) **classifier default reverted qcoder→`my-classifier-q3`** for all commands — enum validity is
GRAMMAR-enforced (Ollama `format`→GBNF), not model-dependent, so qcoder (20.7 GB, doesn't fit 12 GB GPU)
bought nothing; q3 fits + validated end-to-end. q3 accurate but slow (~12 s/call). (2) **T-15 Slice 1**:
predicted `type` now surfaced in `CandidateOutput`/`AddOutput` JSON (`type,omitempty`) — was dropped at the
MCP boundary. (3) **Acceptance regressions repaired** (T-17 + T-18, 13 tests): running `-tags=acceptance`
exposed that T-13 made `taxonomy_path` mandatory but many `Given`s never set it (build-tag hid it from
`go test ./...`); plus a pre-existing `correct` seed-id bug (T-11 date normalization). Full suite needs
`-timeout 30m` (q3 slow). Open: T-14 (model accuracy+speed benchmark), T-16 doc cleanup, T-19 (no
"none-of-these" escape hatch). See `.claude/session42-postmortem.md`.
**Session 43 (2026-06-30) — WS-B slice 3: `batch-auto` → log-append (branch `feat/ws-b-slice3-batch-auto-log-append`, 4 commits).**
`batch-auto` no longer writes the workbook — auto-rows append to `expenses_log.jsonl` (only durable writer is now
`generate-workbook`). New `appendClassified`/`appendOneRow` route through `appender.ExpandAndAppend` (installments
expand at append time; cross-year installments carry their real next-year date — **`rollover.csv` retired**).
**Failure honesty:** an append error downgrades the row (AutoInserted=false) → honest summary + non-zero exit + row
lands in review.csv (CSVs written AFTER append). **Pre-flight** `preflightLogPath` fails fast on an unwritable log
before classifying. `--dry-run` = classify + CSVs, no append. `workflow.InsertBatchExpensesFromClassified`
**deprecated** (`// Deprecated:`), kept for WS-E delete; plain `batch` keeps `InsertBatchExpenses` live. `logExpense`
deleted; `auto` insert→append vocabulary renamed (`✓ Appended`). Plan: `.claude/plans/ws-b-slice3-batch-auto-log-append.md`.
**WS-B slice 4 (session 44) — `apply` → log-append DONE (branch `feat/ws-b-slice4-apply-log-append`).** `apply`
no longer writes the workbook — new confirmed/corrected rows append to `expenses_log.jsonl` via
`appender.ExpandAndAppend` (count=1; T-21 = reviewed-installment under-recording deferred). Write-order flipped
(log-first/feedback-second/downgrade-on-log-failure); `dryRun` gates the found+corrected feedback write (leak
fix); **non-destructive both-path pre-flight** (classifications before `processEntries`; expense-log only when
newRows>0, NO `O_CREATE`). Deleted `insertNewRows`+excel-allocation pipeline, `--workbook`/`--backup`, the
`uninsertable` concept, dead `IsInsertable`/`IsAlreadyHandled`. Shared excel helpers stay live under plain
`batch`. Plan: `.claude/plans/ws-b-slice4-apply-log-append.md`. **WS-B complete** → next WS-D (retire bare-name
fallback, T-09) → WS-E (delete dead insert code).

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
- **Installment notation** — "99,90/3" means 3 monthly payments. `add`/`auto`/`batch-auto` expand at **append
  time** via `appender.ExpandAndAppend` (cross-year → real next-year date, no rollover); plain `batch`'s workbook
  path still expands at insert time

## Deeper Memory → KNOWLEDGE.md
- **Batch pipeline** — single-open optimization, installment expansion, rollover handling
- **Command design** — classify (read-only) → auto (single → log append) → batch-auto (CSV batch → log append)
- **JSON output mode** — structured output for MCP integration
