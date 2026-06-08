#!/usr/bin/env bash
# Whitelist-safe entrypoint for the session-handoff pipeline.
# Thin wrapper over handoff.py — see that file for argument docs (--payload,
# --repo-root, --registry, --dry-run). Forwards all args verbatim.
#
# No `cd`: invoking `python3 "$SCRIPT_DIR/handoff.py"` puts $SCRIPT_DIR on
# sys.path[0], so the package's flat imports resolve without changing the
# working directory — which keeps relative --payload paths and git-root
# detection anchored to the caller's CWD.
set -euo pipefail
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
exec python3 "$SCRIPT_DIR/handoff.py" "$@"
