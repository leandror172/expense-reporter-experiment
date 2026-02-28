#!/bin/bash
# Session-start context summary — replaces reading multiple tracking files.
# Outputs ~30 lines covering everything needed to begin a session.
#
# Usage: .claude/tools/resume.sh

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
SESSION_LOG="$PROJECT_ROOT/.claude/session-log.md"

echo "═══════════════════════════════════════════════════"
echo "  PROJECT RESUME — $(date +%Y-%m-%d)"
echo "═══════════════════════════════════════════════════"
echo ""

# 1. Current status (layer, next tasks, branch)
"$SCRIPT_DIR/ref-lookup.sh" current-status 2>/dev/null \
  | grep -v "^<!-- " | grep -v "^$" | head -20
echo ""

# 2. "Next" pointer from most recent session entry in session-log.md
echo "── Last session ────────────────────────────────────"
# Extract the most recent session heading + its "Next" section
NEXT_SECTION=$(awk '
  /^## 20/ && !found { found=1; print; next }
  found && /^### Next/ { in_next=1; print; next }
  found && in_next && /^---/ { exit }
  found && in_next { print }
' "$SESSION_LOG")

if [ -n "$NEXT_SECTION" ]; then
  echo "$NEXT_SECTION"
else
  echo "(no Next pointer found in session-log.md)"
fi
echo ""

# 3. Recent git commits
echo "── Recent commits ──────────────────────────────────"
git -C "$PROJECT_ROOT" log --oneline -5 2>/dev/null || echo "(not a git repo)"
echo ""

# 4. Git working tree status
STATUS=$(git -C "$PROJECT_ROOT" status -s 2>/dev/null)
if [ -n "$STATUS" ]; then
  echo "── Uncommitted changes ─────────────────────────────"
  echo "$STATUS"
  echo ""
fi

echo "═══════════════════════════════════════════════════"
echo "  Use: .claude/tools/ref-lookup.sh <KEY>"
KEYS=$("$SCRIPT_DIR/ref-lookup.sh" list 2>/dev/null | tr '\n' ' ')
if [ -n "$KEYS" ]; then
  echo "  Keys: $KEYS"
else
  echo "  (no ref:KEY blocks found)"
fi
echo "═══════════════════════════════════════════════════"
