# expense-reporter/ — Quick Memory

*Working memory for the Go application. Injected into agents. Keep under 30 lines.*

## Status
Full CLI: add, batch, classify, auto, batch-auto, correct, review, apply,
generate-workbook. 220+ unit tests; JSON output mode; few-shot + MCP feedback done.
**JSONL logs are the source of truth; `generate-workbook` is the ONLY workbook writer.**
WS-B complete: batch-auto and apply append to `expenses_log.jsonl` via
`appender.ExpandAndAppend`. Classifier predicts the full path (T-13); default model
`my-classifier-q3`. Next: WS-D (retire bare-name fallback) → WS-E (delete dead insert
code). History → KNOWLEDGE.md "Milestone Log".

## Structure
```
cmd/expense-reporter/cmd/  # Cobra subcommands (one file each)
cmd/workbook-inspect/      # Thin wrapper over internal/inspect
internal/                  # batch classifier cli config excel feedback generate
                           # taxonomy inspect logger models parser resolver review
                           # apply appender workflow
pkg/utils/  config/config.json
```

## Key Rules
- **Cobra pattern** — one `.go` file per subcommand
- **Table-driven tests with testify** — `assert`/`require`
- **Brazilian format everywhere** — DD/MM/YYYY, comma decimal, BRL
- **Error wrapping** — `fmt.Errorf("context: %w", err)`
- **Installments** — "99,90/3" = 3 monthly payments, expanded at APPEND time;
  plain `batch` still expands at insert

## Deeper Memory → KNOWLEDGE.md
Log-append path (WS-B) · workbook generator design · milestone log (sessions 26–44)
