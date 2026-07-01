# mcp-server/ — Quick Memory

*Working memory for the MCP server. Injected into agents. Keep under 30 lines.*

## Status
Complete and operational. Two tools: `classify_expense` (read-only, returns
recommendation + `classification_id`) and `add_expense` (accepts 5 optional prediction
params for feedback linkage).

## Structure
```
src/expense_mcp/
  server.py        # FastMCP server — classify_expense + add_expense tools
  binary.py        # Subprocess wrapper — find_binary(), run_binary()
  __main__.py      # Entry point (python -m expense_mcp)
tests/test_binary.py
pyproject.toml     # uv-managed, mcp[cli] dependency; run-server.sh launcher
```

## Key Rules
- **No classification logic here** — all logic lives in the Go binary
- **Subprocess bridge** — calls `expense-reporter` binary with `--json` flag
- **Binary resolved at startup** — `EXPENSE_REPORTER_BIN` env or `go build` fallback
- **`--data-dir` always passed** — taxonomy lookup works regardless of cwd

## Deeper Memory → KNOWLEDGE.md
Tool design (two not three) · binary resolution · data-directory fix
