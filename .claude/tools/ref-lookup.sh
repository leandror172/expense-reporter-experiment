#!/bin/bash
# Resolve a [ref:KEY] tag to its reference content.
# Searches all *.md files in the project for <!-- ref:KEY --> markers.
#
# Usage: .claude/tools/ref-lookup.sh <KEY> [--root /abs/path/to/repo]
#        .claude/tools/ref-lookup.sh --list [--root /abs/path/to/repo]
#        .claude/tools/ref-lookup.sh --paths [--root /abs/path/to/repo]
#        .claude/tools/ref-lookup.sh 'ltg-plan-*' [--root /abs/path/to/repo]
#
# KEY may contain a trailing wildcard (*) for prefix search.
# All matching blocks are printed in key-sorted order, separated by a blank line.
#
# --paths prints KEY<TAB>repo-relative-path for the first (non-local) occurrence of
# each ref key, sorted by key. Entries under .claude/local/ are excluded (safety filter).
#
# --root overrides the default project root (repo containing this script).
# Use it to look up refs from a different repository.

set -euo pipefail

# Resolve default project root (one level above .claude/tools/)
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"

# Parse arguments: extract optional --root before processing KEY
KEY=""
while [[ $# -gt 0 ]]; do
  case "$1" in
    --root)
      PROJECT_ROOT="$2"
      shift 2
      ;;
    *)
      KEY="$1"
      shift
      ;;
  esac
done

# Validate root directory
if [ ! -d "$PROJECT_ROOT" ]; then
  echo "Error: --root directory not found: $PROJECT_ROOT"
  exit 1
fi

# --list mode: print all available keys and exit 0 (MCP-friendly)
if [ "$KEY" = "--list" ] || [ "$KEY" = "list" ]; then
  grep -roh --include="*.md" '<!-- ref:[a-z0-9-]* -->' "$PROJECT_ROOT" 2>/dev/null \
    | sed 's/<!-- ref://;s/ -->//' | sort -u
  exit 0
fi

# --paths mode: print KEY<TAB>repo-relative-path for first non-local occurrence of each key.
# Pipeline order rationale:
#   (1) grep -rHn collects all marker occurrences with file+line (absolute paths)
#   (2) grep -v local/ BEFORE dedup — prevents a local copy sorting first from winning
#       "first occurrence" and leaking or masking the canonical non-local path
#   (3) grep -v local/ BEFORE prefix-strip — pattern needs the absolute path
#   (4) sed extracts KEY<TAB>absfile from each grep -Hn line
#   (5) awk !seen[$1]++ = first occurrence per key (mirrors single-key grep -rl|head -1)
#   (6) final sed strips PROJECT_ROOT/ prefix → repo-relative (git-grep form)
#   (7) sort for deterministic output
#   || true: robustness when corpus is empty or all refs are local-only
if [ "$KEY" = "--paths" ] || [ "$KEY" = "paths" ]; then
  grep -rHn --include="*.md" '<!-- ref:[a-z0-9-]* -->' "$PROJECT_ROOT" 2>/dev/null \
    | grep -v '/\.claude/local/' \
    | sed -E 's#^(.+):[0-9]+:<!-- ref:([a-z0-9-]+) -->$#\2\t\1#' \
    | grep -P '^[a-z0-9-]+\t/' \
    | awk -F'\t' '!seen[$1]++' \
    | sed "s#^\([a-z0-9-]*\)\t$PROJECT_ROOT/#\1\t#" \
    | sort \
    || true
  exit 0
fi

if [ -z "$KEY" ]; then
  echo "Usage: $0 <KEY> [--root /abs/path/to/repo]"
  echo "Available keys:"
  grep -roh --include="*.md" '<!-- ref:[a-z0-9-]* -->' "$PROJECT_ROOT" 2>/dev/null \
    | sed 's/<!-- ref://;s/ -->//' | sort -u
  exit 1
fi

# Glob mode: KEY contains '*' — expand to all matching keys and emit each block.
if [[ "$KEY" == *"*"* ]]; then
  PATTERN="^${KEY//\*/[a-z0-9-]*}$"
  MATCHES=$(grep -roh --include="*.md" '<!-- ref:[a-z0-9-]* -->' "$PROJECT_ROOT" 2>/dev/null \
    | sed 's/<!-- ref://;s/ -->//' | grep -E "$PATTERN" | sort -u)
  if [ -z "$MATCHES" ]; then
    echo "No ref keys matching pattern: $KEY"
    exit 1
  fi
  FIRST=1
  while IFS= read -r MATCH_KEY; do
    MFILE=$(grep -rl --include="*.md" "<!-- ref:$MATCH_KEY -->" "$PROJECT_ROOT" 2>/dev/null | head -1)
    [ -z "$MFILE" ] && continue
    [ "$FIRST" -eq 0 ] && echo
    FIRST=0
    sed -n "/<!-- ref:$MATCH_KEY -->/,/<!-- \/ref:$MATCH_KEY -->/p" "$MFILE"
  done <<< "$MATCHES"
  exit 0
fi

FILE=$(grep -rl --include="*.md" "<!-- ref:$KEY -->" "$PROJECT_ROOT" 2>/dev/null | head -1)
if [ -z "$FILE" ]; then
  echo "ref:$KEY not found in any *.md file under $PROJECT_ROOT"
  exit 1
fi

sed -n "/<!-- ref:$KEY -->/,/<!-- \/ref:$KEY -->/p" "$FILE"
