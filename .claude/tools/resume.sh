#!/usr/bin/env bash
# Session-start context summary. Thin shim — the sections live in .claude/resume.yaml.
#
# This file used to be six hardcoded bash sections, which is why it needed an
# `overlay-keep:` region for repos that wanted a different summary. It doesn't any more:
# WHAT gets printed, in what order, filtered how, is now per-repo CONFIG. Edit
# .claude/resume.yaml. See docs/plans/resume-config-steps.md (R-D1/R-D2/R-D5).
#
# Engine resolution mirrors run-handoff.sh:
#   1. `st-resume` on PATH        — the installed package. Preferred.
#   2. a sibling src/ tree        — the overlay source checkout (dev home repo).
set -euo pipefail
_here="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

if command -v st-resume >/dev/null 2>&1; then
  exec st-resume "$@"
fi

# Overlay source checkout: this script installs to .claude/tools/, but in the source
# tree it sits at <overlay>/files/, so try both roots for a sibling src/.
for _candidate in "$_here/../src" "$_here/../../src"; do
  if [ -d "$_candidate/sessiontracking" ]; then
    _src="$(cd "$_candidate" && pwd)"
    exec env PYTHONPATH="$_src${PYTHONPATH:+:$PYTHONPATH}" python3 -m sessiontracking.resume.cli "$@"
  fi
done

echo "resume: session-tracking is not installed. Install it with:" >&2
echo "  uv tool install --editable <llm-repo>/overlays/session-tracking" >&2
exit 127
