#!/bin/bash
# Resolve a [ref:KEY] tag to its reference content.
# Searches all *.md files in the project for <!-- ref:KEY --> markers.
#
# Usage: .claude/tools/ref-lookup.sh <KEY>
# Example: .claude/tools/ref-lookup.sh bash-wrappers

set -euo pipefail

# Resolve project root (one level above .claude/tools/)
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"

KEY="${1:-}"

# --list mode: print all available keys and exit 0 (MCP-friendly)
if [ "$KEY" = "--list" ] || [ "$KEY" = "list" ]; then
  grep -roh --include="*.md" '<!-- ref:[a-z-]* -->' "$PROJECT_ROOT" 2>/dev/null \
    | sed 's/<!-- ref://;s/ -->//' | sort -u
  exit 0
fi

if [ -z "$KEY" ]; then
  echo "Usage: $0 <KEY>"
  echo "Available keys:"
  grep -roh --include="*.md" '<!-- ref:[a-z-]* -->' "$PROJECT_ROOT" 2>/dev/null \
    | sed 's/<!-- ref://;s/ -->//' | sort -u
  exit 1
fi

FILE=$(grep -rl --include="*.md" "<!-- ref:$KEY -->" "$PROJECT_ROOT" 2>/dev/null | head -1)
if [ -z "$FILE" ]; then
  echo "ref:$KEY not found in any *.md file under $PROJECT_ROOT"
  exit 1
fi

sed -n "/<!-- ref:$KEY -->/,/<!-- \/ref:$KEY -->/p" "$FILE"
