#!/bin/bash
# Session-start context summary — replaces reading multiple tracking files.
# Outputs ~80-100 lines covering everything needed to begin a session.
#
# Section order (post-advisor revision, session 60):
#   1. Current status (ref:current-status, head -30)
#   2. Last session Next pointer (parsed from session-log.md)
#   3. Key file locations (ref:quick-pointers, full)
#   4. Active decisions (ref:active-decisions, head -12)
#   5. Recent commits + uncommitted changes
#   6. Footer: user-prefs (multiline), deferred hint, ref key count
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
STATUS_OUT=$("$SCRIPT_DIR/ref-lookup.sh" current-status 2>/dev/null \
  | grep -v "^<!-- " | grep -v "^$" || true)
if [ -n "$STATUS_OUT" ]; then
  echo "$STATUS_OUT" | head -30
else
  echo "(no ref:current-status block found)"
fi
echo ""

# 2. "Next" pointer from most recent session entry in session-log.md
echo "── Last session ────────────────────────────────────"
NEXT_SECTION=$(awk '
  /^## 20/ && !found { found=1; print; next }
  found && /^### Next/ { in_next=1; print; next }
  found && in_next && /^---/ { exit }
  found && in_next { print }
' "$SESSION_LOG" 2>/dev/null || true)

if [ -n "$NEXT_SECTION" ]; then
  echo "$NEXT_SECTION"
else
  echo "(no Next pointer found in session-log.md)"
fi
echo ""

# 3. Key file locations
echo "── Key files (ref:quick-pointers) ─────────────────"
PTRS=$("$SCRIPT_DIR/ref-lookup.sh" quick-pointers 2>/dev/null \
  | grep -v "^<!-- " | grep -v "^$" || true)
if [ -n "$PTRS" ]; then
  echo "$PTRS"
else
  echo "(no ref:quick-pointers block found)"
fi
echo ""

# 4. Active decisions (cross-cutting principles, frozen layer pointers)
echo "── Active decisions (ref:active-decisions) ─────────"
DECISIONS=$("$SCRIPT_DIR/ref-lookup.sh" active-decisions 2>/dev/null \
  | grep -v "^<!-- " | grep -v "^$" || true)
if [ -n "$DECISIONS" ]; then
  echo "$DECISIONS" | head -12
else
  echo "(no ref:active-decisions block found)"
fi
echo ""

# 5. Recent git commits
echo "── Recent commits ──────────────────────────────────"
git -C "$PROJECT_ROOT" log --oneline -5 2>/dev/null || echo "(not a git repo)"
echo ""

# 6. Git working tree status (conditional)
DIRTY=$(git -C "$PROJECT_ROOT" status -s 2>/dev/null)
if [ -n "$DIRTY" ]; then
  echo "── Uncommitted changes ─────────────────────────────"
  echo "$DIRTY"
  echo ""
fi

# Footer
echo "═══════════════════════════════════════════════════"
PREFS=$("$SCRIPT_DIR/ref-lookup.sh" user-prefs 2>/dev/null \
  | grep -v "^<!-- " | grep -v "^$" || true)
if [ -n "$PREFS" ]; then
  echo "  User preferences:"
  echo "$PREFS"
else
  echo "  (no ref:user-prefs block found)"
fi

echo "═══════════════════════════════════════════════════"
KEY_COUNT=$("$SCRIPT_DIR/ref-lookup.sh" list 2>/dev/null | wc -l || echo "?")
echo "  (items pending — see ref:deferred-infra)"
echo "  $KEY_COUNT ref keys available — run: .claude/tools/ref-lookup.sh --list"
echo "═══════════════════════════════════════════════════"
