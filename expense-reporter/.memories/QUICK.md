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
year is what dated the entry. `config.json` `date_year` → 2026. **T-41 slice 2 (s66): `auto` migrated** — one
`parse.Fields` call, `--year` flag, `date_year` live, stale-year warning;
`appendExpense` collapsed to take a `ParsedExpense` (the five-scalar signature was
what produced T-35); field sentinels added to `parse`. **T-41 slice 3 (s67):
`batch-auto` migrated** — `inputRow` deleted (`ParsedExpense` already held every
field plus the installment count it discarded, which was the ROOT of the 4× re-parse);
`resumeParseErr` deleted as unreachable; `--year`; ONE counted stale-year warning
(T-49); join-id pinned at construction (T-40). `classified.csv`/`review.csv` now carry
the CANONICAL date, which repairs a live silent divergence — `review.ReadQueue` hashed
the raw column into `reviewed.json`'s id while both logs hashed the canonical form, so
`apply` looked up an id it never writes and an id miss there appends silently.
Next: slice 4 (`apply`, incl. recomputing `entry.ID` alongside the date) → T-21 → T-42 close.
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
  date/currency parsers from command code. Migrated: add/correct/auto; pending:
  batch-auto/apply. Command-layer helpers live in `cmd/parse_boundary.go`
  (`parseOptions`, `warnIfStaleConfiguredYear`, `describeParseFailure`) — `parse`
  must not import `config`. `internal/parser` (no "e") is the DYING pre-pivot one

## Deeper Memory → KNOWLEDGE.md
Log-append path (WS-B) · workbook generator design · milestone log (sessions 26–44)
