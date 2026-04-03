# mcp-server/ — Quick Memory

*Working memory for the MCP server. Injected into agents. Keep under 30 lines.*

## Status
Complete and operational (5.8b, 2026-03-27). Two tools registered and smoke-tested.
No planned changes — this layer is intentionally thin.

## Structure
```
src/expense_mcp/
  server.py        # FastMCP server — classify_expense + add_expense tools
  binary.py        # Subprocess wrapper — find_binary(), run_binary()
  __main__.py      # Entry point (python -m expense_mcp)
tests/
  test_binary.py   # Unit tests for binary location/invocation
pyproject.toml     # uv-managed, mcp[cli] dependency
run-server.sh      # Convenience launcher
```

## Key Rules
- **No classification logic here** — all logic lives in the Go binary
- **Subprocess bridge** — calls `expense-reporter` binary with `--json` flag
- **Binary resolved at startup** — via `EXPENSE_REPORTER_BIN` env or `go build` fallback
- **`--data-dir` always passed** — ensures taxonomy lookup works regardless of cwd
- **`classify_expense` is read-only** — never inserts; returns recommendation for caller to act on

## Deeper Memory → KNOWLEDGE.md
- **Tool design** — why two tools, not three
- **Binary resolution** — env var vs auto-build strategy
- **Data directory fix** — why `--data-dir` flag was needed
