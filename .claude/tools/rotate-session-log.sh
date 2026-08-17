#!/bin/bash
# Rotate session-log.md: archive each spilled session entry into its own file,
# keeping the N most recent entries in session-log.md.
# Called automatically by the session-handoff skill at the end of each session.
#
# Usage: .claude/tools/rotate-session-log.sh [--keep N] [--dry-run]
#   --keep N     Keep N most recent session entries (default: 1)
#   --dry-run    Show what would happen without writing any files
#
# Archive filename format: session-log-<date>-s<N>-<slug>.md
# where slug is a lowercased, hyphenated, alnum-only title (truncated to 40 chars).
# Fallback when heading is unparseable: session-log-<date>-s<N>.md

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
SESSION_LOG="$PROJECT_ROOT/.claude/session-log.md"
ARCHIVE_DIR="$PROJECT_ROOT/.claude/archive"

KEEP=1
DRY_RUN=false

while [[ $# -gt 0 ]]; do
  case "$1" in
    --keep)    KEEP="$2"; shift 2 ;;
    --dry-run) DRY_RUN=true; shift ;;
    *) echo "Unknown argument: $1"; exit 1 ;;
  esac
done

mkdir -p "$ARCHIVE_DIR"

# Find line numbers of all session headings (## YYYY-MM-DD ...)
mapfile -t ENTRY_LINES < <(grep -n "^## 20" "$SESSION_LOG" | cut -d: -f1)
ENTRY_COUNT=${#ENTRY_LINES[@]}

if (( ENTRY_COUNT <= KEEP )); then
  echo "session-log.md has $ENTRY_COUNT session entries (threshold: $KEEP) — no rotation needed."
  exit 0
fi

ARCHIVE_COUNT=$((ENTRY_COUNT - KEEP))
TOTAL_LINES=$(wc -l < "$SESSION_LOG")

# Archive each spilled entry (index KEEP .. ENTRY_COUNT-1).
# Entries are newest-first: index 0 = newest (kept), index KEEP+ = spilled.
for (( i = KEEP; i < ENTRY_COUNT; i++ )); do
  heading=$(sed -n "${ENTRY_LINES[$i]}p" "$SESSION_LOG")

  # Parse heading: ## YYYY-MM-DD - Session N: Title text
  entry_date=$(echo "$heading" | grep -oP '\d{4}-\d{2}-\d{2}' | head -1 || true)
  entry_date="${entry_date:-unknown-date}"

  entry_n=$(echo "$heading" | grep -oP 'Session \K\d+' | head -1 || true)
  entry_n="${entry_n:-0}"

  # Title = everything after "Session N: " (the colon-space delimiter)
  raw_title=$(echo "$heading" | sed 's/^.*Session [0-9]*: //' || true)
  # If sed didn't match (no "Session N: " found), title equals full heading — discard
  if [[ "$raw_title" == "$heading" ]]; then
    raw_title=""
  fi

  # Build slug: lowercase, alnum+hyphens only, collapse runs, strip edges, truncate 40
  slug=$(echo "$raw_title" \
    | tr '[:upper:]' '[:lower:]' \
    | sed 's/[^a-z0-9]/-/g' \
    | sed 's/-\+/-/g' \
    | sed 's/^-//; s/-$//' \
    | cut -c1-40 \
    || true)

  if [[ -n "$slug" ]]; then
    archive_filename="session-log-${entry_date}-s${entry_n}-${slug}.md"
  else
    archive_filename="session-log-${entry_date}-s${entry_n}.md"
  fi

  if $DRY_RUN; then
    echo "[dry-run] Would archive: $heading → $archive_filename"
  else
    echo "Archiving: $heading → $archive_filename"
    # Entry i spans ENTRY_LINES[i] .. ENTRY_LINES[i+1]-1, or EOF for the last entry
    if (( i + 1 < ENTRY_COUNT )); then
      end_line=$(( ENTRY_LINES[i+1] - 1 ))
    else
      end_line=$TOTAL_LINES
    fi
    sed -n "${ENTRY_LINES[$i]},${end_line}p" "$SESSION_LOG" > "$ARCHIVE_DIR/$archive_filename"
  fi
done

if $DRY_RUN; then
  echo "[dry-run] No changes made."
  exit 0
fi

# Truncate session-log.md: keep header + KEEP most recent entries
TMPFILE=$(mktemp)
head -n $(( ENTRY_LINES[KEEP] - 1 )) "$SESSION_LOG" > "$TMPFILE" && mv "$TMPFILE" "$SESSION_LOG"

echo "Done. Archived $ARCHIVE_COUNT entries. session-log.md now has $KEEP entries."
