# expense-reporter/ — Quick Memory

*Working memory for the Go application. Injected into agents. Keep under 30 lines.*

## Status
Full CLI: add, batch, classify, auto, batch-auto, correct, review, apply,
generate-workbook. 263 unit test funcs (647 w/ subtests); JSON output mode; few-shot + MCP feedback done.
**JSONL logs are the source of truth; `generate-workbook` is the ONLY workbook writer.**
WS-B complete: batch-auto and apply append to `expenses_log.jsonl` via
`appender.ExpandAndAppend`. Classifier predicts the full path (T-13); default model
`my-classifier-q3`. `batch-auto --resume` (T-20): predicts each row's log-entry ids
up front, skips fully-logged rows BEFORE the LLM; partially-logged series → review
(never auto-completed); always-on stderr duplicate-append warning. Ledger = one
`map[id]int` (`feedback.LoadExpenseIDCounts`), consumed classify-skips-first;
`appender.PredictEntryIDs` shares `expandEntries` with `ExpandAndAppend` (drift
guard). **T-41 parse boundary slice 1 (s63): `add`/`correct` parse via
`internal/parse`; `--year` on both; `date_year` LIVE there (rung 3); slices
2–4 (auto → batch-auto → apply) pending.** **s65 date/year hardening (PRs #56/#57):**
the boundary now validates the RESOLVED year twice (renderable 1..9999; entry beyond
the CURRENT year refused — provisional, year-scale), reports WHICH rung resolved it
(`YearSource`), and `add`/`correct` warn on stderr when a `date_year` below the current
year is what dated the entry. `config.json` `date_year` → 2026. Next after T-41: T-21 → T-42 close.
History → KNOWLEDGE.md "Milestone Log".

## Structure
```
cmd/expense-reporter/cmd/  # Cobra subcommands (one file each)
cmd/workbook-inspect/      # Thin wrapper over internal/inspect
internal/                  # batch classifier cli config excel feedback generate
                           # taxonomy inspect logger models parse parser resolver
                           # review apply appender workflow
pkg/utils/  config/config.json
```

## Key Rules
- **Cobra pattern** — one `.go` file per subcommand
- **Table-driven tests with testify** — `assert`/`require`
- **Brazilian format everywhere** — DD/MM/YYYY, comma decimal, BRL
- **Error wrapping** — `fmt.Errorf("context: %w", err)`
- **Method extraction** — multi-step function bodies read as named delegated steps
  (≤~15 lines); step-comments promote to helper doc comments (KNOWLEDGE.md)
- **Installments** — "99,90/3" = 3 monthly payments, expanded at APPEND time;
  plain `batch` still expands at insert
- **Parse boundary (T-41)** — commands parse input ONLY via `internal/parse`
  (`ParsedExpense`; `DateString()` = the identity bytes); never call utils
  date/currency parsers from command code. Migrated: add/correct; pending:
  auto/batch-auto/apply. `internal/parser` (no "e") is the DYING pre-pivot one

## Deeper Memory → KNOWLEDGE.md
Log-append path (WS-B) · workbook generator design · milestone log (sessions 26–44)
