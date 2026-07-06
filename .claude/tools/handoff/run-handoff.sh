#!/usr/bin/env bash
# Whitelist-safe entrypoint for the session-handoff pipeline.
#
# Engine resolution: prefer a handoff.py co-located with THIS shim (the overlay
# source tree / dev home repo, where changes are tested against source); else the
# shared user-level install at ~/.claude/tools/handoff/. Target-repo installs ship
# only the shim, so they resolve to the user-level engine.
#
# Registry guard: no-ops silently in repos that have no handoff registry, so the
# user-level hook is safe in uninstalled repos. An explicit --registry (the
# home-repo / overlay-source invocation) bypasses the guard.
set -euo pipefail
_here="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
_root="$(git rev-parse --show-toplevel 2>/dev/null)" || _root="$PWD"

# An explicit --registry ("--registry path" or "--registry=path") bypasses the
# per-repo registry-file guard.
_explicit_registry=0
for _arg in "$@"; do
  case "$_arg" in
    --registry|--registry=*) _explicit_registry=1; break ;;
  esac
done
if [ "$_explicit_registry" -eq 0 ]; then
  [ -f "$_root/.claude/handoff/registry.yaml" ] || exit 0
fi

# Prefer a co-located engine (source tree); fall back to the user-level install.
if [ -f "$_here/handoff.py" ]; then
  _engine="$_here/handoff.py"
else
  _engine="$HOME/.claude/tools/handoff/handoff.py"
fi
exec python3 "$_engine" "$@"
