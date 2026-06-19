#!/usr/bin/env bash
# Whitelist-safe entrypoint for the session-handoff pipeline.
# Pipeline modules live at ~/.claude/tools/handoff/ (user-level install).
# Guard: no-ops silently in repos without a handoff registry (safe at user level).
set -euo pipefail
_root=$(git rev-parse --show-toplevel 2>/dev/null) || _root="$PWD"
[ -f "$_root/.claude/handoff/registry.yaml" ] || exit 0
exec python3 "$HOME/.claude/tools/handoff/handoff.py" "$@"
