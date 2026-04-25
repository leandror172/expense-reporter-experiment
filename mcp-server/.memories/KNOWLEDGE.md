# mcp-server/ — Knowledge (Semantic Memory)

*MCP server accumulated decisions. Read on demand by agents.*

## Two Tools, Not Three (2026-03-27)
Originally planned three MCP tools: classify, add, and auto (classify + insert).
Reduced to two: `classify_expense` (read-only) and `add_expense` (insert).
The "auto" tool was dropped because the MCP caller (Claude Code) can compose
classify → decide → add itself, and keeping insertion authority explicit is safer.
**Rationale:** An auto-insert tool behind MCP is a footgun — the caller should make
the insert decision, not the server. Two composable tools are more flexible than
one opinionated one.
**Implication:** The MCP caller must check the `action` field from `classify_expense`
and call `add_expense` separately if it decides to insert.

## Thin Wrapper Pattern (2026-03-26)
The server contains zero classification logic. Every tool call shells out to the
Go binary with `--json` and returns `stdout` verbatim. Error handling converts
non-zero exit codes to JSON error responses.
**Rationale:** Single source of truth. If classification logic existed in both Go and
Python, they could diverge. The Go binary is the canonical implementation.
**Implication:** New output fields in Go (e.g. `classification_id` in `AutoOutput`)
surface to MCP clients automatically — no Python changes needed. But new *input* flags
(e.g. prediction flags on `add`) require explicitly adding them to the Python tool
signature and forwarding them, because they are part of the declared MCP tool contract.

## Binary Resolution Strategy (2026-03-26)
Priority: `EXPENSE_REPORTER_BIN` env var → auto-build from Go module.
The auto-build runs `go build` at server startup if no env var is set.
Path to Go module is resolved relative to `binary.py`'s file location.
**Rationale:** Development convenience — no manual build step needed. Production
deployments can pin a specific binary via the env var.
**Implication:** First startup is slower (includes `go build`). Subsequent starts
reuse the built binary if `EXPENSE_REPORTER_BIN` is set.

## Data Directory Fix (2026-03-27)
The Go binary defaults `--data-dir` to `data/classification` (relative path).
When invoked via MCP, the working directory is the Go module root (`expense-reporter/`),
but `data/classification/` is one level up. The `--data-dir` flag with an absolute
path (resolved from `_DATA_DIR` in `binary.py`) fixes this.
**Rationale:** Relative paths break when the caller's cwd differs from expectations.
The MCP server always knows the repo layout, so it passes the absolute path.
**Implication:** The `--data-dir` flag was added to the Go `add` command specifically
for this MCP use case (it already existed on `auto`).

## Technology Choices (2026-03-26)
- **FastMCP** — high-level MCP framework; handles protocol negotiation, tool registration
- **uv** — Python package manager; manages venv and dependencies
- **pytest-asyncio** — async test support (FastMCP tools are async)
**Rationale:** FastMCP reduces boilerplate to near-zero for a simple tool server.
uv is fast and handles the complete Python toolchain.
