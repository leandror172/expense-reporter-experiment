#!/usr/bin/env bash
# Bash wrapper for expense-reporter MCP server.
# Whitelistable entry point for Claude Code.
set -euo pipefail
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
exec uv run --project "$SCRIPT_DIR" python -m expense_mcp
