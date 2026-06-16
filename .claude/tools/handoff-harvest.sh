#!/bin/bash
# handoff-harvest.sh
#
# Emits git commit subjects made since the last session-handoff commit.
# Used by the session-handoff skill to determine what was done during a session.
#
# Usage: .claude/tools/handoff-harvest.sh
#
# Behavior:
# 1. Derive PROJECT_ROOT from SCRIPT_DIR using two-level navigation.
# 2. cd into PROJECT_ROOT so all git commands operate on the correct repo.
# 3. Find the newest commit with subject starting with 'chore(session-handoff):'
# 4. If found, run `git log <sha>..HEAD --format='%s'` and print to stdout.
#    - If <sha>..HEAD is empty (HEAD equals handoff), stdout is empty — no error or exit code change.
# 5. If not found:
#    - Print "handoff-harvest: no chore(session-handoff): commit found; falling back to last 20 commits" to stderr
#    - Fall back to `git log -n 20 --format='%s'` and print to stdout.
# 6. Always exit 0.

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"

cd "$PROJECT_ROOT" || { echo "Failed to cd into PROJECT_ROOT: $PROJECT_ROOT"; exit 1; }

# Find the newest commit with subject starting with 'chore(session-handoff):'
HANDOFF_SHA=$(git log -n1 --format=%H --grep='^chore(session-handoff):' || true)

if [[ -z "$HANDOFF_SHA" ]]; then
  # No handoff commit found — fall back to last 20 commits
  echo "handoff-harvest: no chore(session-handoff): commit found; falling back to last 20 commits" >&2
  git log -n 20 --format='%s'
else
  # Handoff commit exists — show commits after it
  git log "$HANDOFF_SHA"..HEAD --format='%s'
fi

exit 0
