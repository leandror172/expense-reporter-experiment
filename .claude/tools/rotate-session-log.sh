#!/bin/bash
# Rotate session-log.md: archive old session entries, keep the N most recent.
# Called automatically by the session-handoff skill at the end of each session.
#
# Usage: .claude/tools/rotate-session-log.sh [--keep N] [--dry-run]
#   --keep N     Keep N most recent session entries (default: 3)
#   --dry-run    Show what would happen without writing any files

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
SESSION_LOG="$PROJECT_ROOT/.claude/session-log.md"
ARCHIVE_DIR="$PROJECT_ROOT/.claude/archive"

KEEP=3
DRY_RUN=false

while [[ $# -gt 0 ]]; do
  case "$1" in
    --keep)    KEEP="$2"; shift 2 ;;
    --dry-run) DRY_RUN=true; shift ;;
    *) echo "Unknown argument: $1"; exit 1 ;;
  esac
done

# Find line numbers of all session headings (## YYYY-MM-DD ...)
mapfile -t ENTRY_LINES < <(grep -n "^## 20" "$SESSION_LOG" | cut -d: -f1)
ENTRY_COUNT=${#ENTRY_LINES[@]}

if [ "$ENTRY_COUNT" -le "$KEEP" ]; then
  echo "session-log.md has $ENTRY_COUNT session entries (threshold: $KEEP) — no rotation needed."
  exit 0
fi

ARCHIVE_COUNT=$((ENTRY_COUNT - KEEP))

# Line where archived content starts: (KEEP+1)th entry (0-indexed array, so index $KEEP)
ARCHIVE_FROM_LINE="${ENTRY_LINES[$KEEP]}"

# Date range of archived sessions (for the archive filename)
OLDEST_LINE="${ENTRY_LINES[$((ENTRY_COUNT - 1))]}"
NEWEST_ARCHIVED_LINE="${ENTRY_LINES[$KEEP]}"
OLDEST_DATE=$(sed -n "${OLDEST_LINE}p" "$SESSION_LOG" | grep -o '[0-9]\{4\}-[0-9]\{2\}-[0-9]\{2\}' | head -1)
NEWEST_ARCHIVED_DATE=$(sed -n "${NEWEST_ARCHIVED_LINE}p" "$SESSION_LOG" | grep -o '[0-9]\{4\}-[0-9]\{2\}-[0-9]\{2\}' | head -1)

ARCHIVE_FILE="$ARCHIVE_DIR/session-log-${OLDEST_DATE}-to-${NEWEST_ARCHIVED_DATE}.md"

echo "Found $ENTRY_COUNT session entries in session-log.md."
echo "Keeping $KEEP most recent; archiving $ARCHIVE_COUNT older entries."
echo "Archived range: $OLDEST_DATE → $NEWEST_ARCHIVED_DATE"
echo "Archive file: $ARCHIVE_FILE"

if $DRY_RUN; then
  echo "[dry-run] No changes made."
  exit 0
fi

# Write archived entries verbatim into archive file
{
  echo "# Session Log Archive — $OLDEST_DATE to $NEWEST_ARCHIVED_DATE"
  echo ""
  echo "> Archived from session-log.md on $(date +%Y-%m-%d). Kept $KEEP most recent entries."
  echo ""
  echo "---"
  echo ""
  tail -n +"$ARCHIVE_FROM_LINE" "$SESSION_LOG"
} > "$ARCHIVE_FILE"

# Truncate session-log.md: keep header + KEEP most recent entries
head -n $((ARCHIVE_FROM_LINE - 1)) "$SESSION_LOG" > /tmp/session-log-rotated.md

# Update (or append) the "Previous logs" pointer in the header to include new archive
ARCHIVE_BASENAME="$(basename "$ARCHIVE_FILE")"
if grep -q "^\*\*Previous logs:\*\*" /tmp/session-log-rotated.md; then
  # Append new archive to the existing pointer line
  sed -i "s|^\(\*\*Previous logs:\*\*.*\)|\1, \`.claude/archive/$ARCHIVE_BASENAME\`|" \
    /tmp/session-log-rotated.md
else
  # Insert after the header line
  sed -i "2a **Previous logs:** \`.claude/archive/$ARCHIVE_BASENAME\`" \
    /tmp/session-log-rotated.md
fi

mv /tmp/session-log-rotated.md "$SESSION_LOG"

echo "Done. $ARCHIVE_COUNT entries archived to $ARCHIVE_FILE"
echo "session-log.md now has $KEEP session entries."
