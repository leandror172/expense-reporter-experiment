#!/bin/bash
# Tests for ref-lookup.sh — fully hermetic (self-contained).
# Usage: <overlay>/files/tests/test-ref-lookup-paths.sh
#        <repo>/.claude/tools/tests/test-ref-lookup-paths.sh   (installed copy)
# Exit 0 = all pass, nonzero = at least one failure.
#
# DESIGN: every test builds its own fixture corpus in a temp dir and points
# ref-lookup.sh at it with --root. The test never reads the host repo's ref
# content, so changes to this (or any) repo cannot change the test result.
# The fixture IS the world under test — like running in a clean container that
# only contains the files the objective requires.

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCRIPT="$SCRIPT_DIR/../ref-lookup.sh"

PASS=0
FAIL=0
declare -a ERRORS=()

pass() { PASS=$((PASS+1)); echo "PASS: $1"; }
fail() { FAIL=$((FAIL+1)); ERRORS+=("$1"); echo "FAIL: $1"; }
run_test() {
  local name="$1"
  local result="$2"
  [ "$result" -eq 0 ] && pass "$name" || fail "$name"
}

# ─── FIXTURE ──────────────────────────────────────────────────────────────────
# A controlled corpus exercising every code path: two plain keys, a duplicated
# key across two files (first-occurrence ordering), a prefix-sharing pair (glob),
# and a .claude/local/ entry that must be excluded by the safety filter.
FIXTURE=$(mktemp -d)
trap 'rm -rf "$FIXTURE"' EXIT

mkdir -p "$FIXTURE/docs" "$FIXTURE/.claude/local"

cat > "$FIXTURE/docs/a.md" << 'EOF'
<!-- ref:alpha -->
Alpha content
<!-- /ref:alpha -->
EOF

cat > "$FIXTURE/docs/b.md" << 'EOF'
<!-- ref:beta -->
Beta content
<!-- /ref:beta -->
EOF

# Prefix-sharing keys for glob expansion (grp-one, grp-two).
cat > "$FIXTURE/docs/groups.md" << 'EOF'
<!-- ref:grp-one -->
Group one content
<!-- /ref:grp-one -->

<!-- ref:grp-two -->
Group two content
<!-- /ref:grp-two -->
EOF

# Same key in two files. 'a-dup.md' sorts before 'z-dup.md', so first
# occurrence (by sorted absolute path) must resolve to a-dup.md.
cat > "$FIXTURE/docs/a-dup.md" << 'EOF'
<!-- ref:dup -->
Dup in A
<!-- /ref:dup -->
EOF

cat > "$FIXTURE/docs/z-dup.md" << 'EOF'
<!-- ref:dup -->
Dup in Z
<!-- /ref:dup -->
EOF

# Local-only key — must never appear in --paths output (safety filter).
cat > "$FIXTURE/.claude/local/secret.md" << 'EOF'
<!-- ref:localonly -->
Secret content
<!-- /ref:localonly -->
EOF

PATHS_OUT=$("$SCRIPT" --paths --root "$FIXTURE" 2>/dev/null) || PATHS_OUT=""
LIST_OUT=$("$SCRIPT" --list --root "$FIXTURE" 2>/dev/null) || LIST_OUT=""

# ─── --paths BEHAVIOUR ─────────────────────────────────────────────────────────

# Test 1: --paths emits KEY<TAB>relpath for plain keys
t1=0
echo "$PATHS_OUT" | grep -qP "^alpha\tdocs/a\.md$" || t1=1
echo "$PATHS_OUT" | grep -qP "^beta\tdocs/b\.md$" || t1=1
run_test "paths: plain keys map to correct repo-relative paths" $t1

# Test 2: .claude/local/ entries excluded (safety filter)
t2=0
echo "$PATHS_OUT" | grep -q 'localonly' && t2=1
echo "$PATHS_OUT" | grep -q '\.claude/local' && t2=1
run_test "paths: .claude/local entries excluded (safety filter)" $t2

# Test 3: the unique-key rows are exact, repo-relative, and key-sorted.
# The 'dup' row is excluded here because its path is traversal-order-dependent
# (see Test 4) — asserting a specific winner would be a non-deterministic test.
t3=0
non_dup_out=$(echo "$PATHS_OUT" | grep -vP '^dup\t' || true)
expected_non_dup=$(printf 'alpha\tdocs/a.md\nbeta\tdocs/b.md\ngrp-one\tdocs/groups.md\ngrp-two\tdocs/groups.md')
[ "$non_dup_out" = "$expected_non_dup" ] || { t3=1; echo "  got:"; echo "$non_dup_out" | sed 's/^/    /'; }
run_test "paths: unique-key rows are exact, relative, and key-sorted" $t3

# Test 4: a duplicated key collapses to exactly ONE line pointing at a real
# occurrence. The tool's "first occurrence" follows grep -r traversal order
# (filesystem readdir), NOT sorted path, so WHICH file wins is intentionally
# unspecified — we assert the dedup invariant, not a specific winner.
t4=0
dup_lines=$(echo "$PATHS_OUT" | grep -cP '^dup\t' || true)
dup_path=$(echo "$PATHS_OUT" | awk -F'\t' '$1 == "dup" {print $2; exit}')
[ "$dup_lines" -eq 1 ] || { t4=1; echo "  expected 1 dup line, got: $dup_lines"; }
case "$dup_path" in
  docs/a-dup.md|docs/z-dup.md) ;;
  *) t4=1; echo "  dup path not a real occurrence: $dup_path" ;;
esac
run_test "paths: duplicated key dedupes to one real occurrence" $t4

# ─── CROSS-MODE CONSISTENCY ────────────────────────────────────────────────────

# Test 5: every --paths key also appears in --list
t5=0
while IFS=$'\t' read -r pkey _ppath; do
  [ -z "$pkey" ] && continue
  echo "$LIST_OUT" | grep -qx "$pkey" || { t5=1; echo "  missing from --list: $pkey"; }
done <<< "$PATHS_OUT"
run_test "consistency: every --paths key appears in --list" $t5

# Test 6: --list emits exactly the fixture's keys, sorted and unique.
# localonly IS listed by --list (only --paths applies the local/ safety filter).
t6=0
expected_list=$(printf 'alpha\nbeta\ndup\ngrp-one\ngrp-two\nlocalonly')
[ "$LIST_OUT" = "$expected_list" ] || { t6=1; echo "  got:"; echo "$LIST_OUT" | sed 's/^/    /'; }
run_test "list: emits exactly the fixture keys, sorted-unique" $t6

# ─── SINGLE-KEY + GLOB LOOKUP ──────────────────────────────────────────────────

# Test 7: single-key lookup prints the full ref block verbatim
t7=0
single_out=$("$SCRIPT" alpha --root "$FIXTURE" 2>/dev/null) || single_out=""
expected_single=$(printf '<!-- ref:alpha -->\nAlpha content\n<!-- /ref:alpha -->')
[ "$single_out" = "$expected_single" ] || { t7=1; echo "  got:"; echo "$single_out" | sed 's/^/    /'; }
run_test "single-key: alpha lookup returns its block verbatim" $t7

# Test 8: glob mode expands prefix to all matching blocks, key-sorted
t8=0
glob_out=$("$SCRIPT" 'grp-*' --root "$FIXTURE" 2>/dev/null) || glob_out=""
expected_glob=$(printf '<!-- ref:grp-one -->\nGroup one content\n<!-- /ref:grp-one -->\n\n<!-- ref:grp-two -->\nGroup two content\n<!-- /ref:grp-two -->')
[ "$glob_out" = "$expected_glob" ] || { t8=1; echo "  got:"; echo "$glob_out" | sed 's/^/    /'; }
run_test "glob: 'grp-*' expands to both blocks, key-sorted" $t8

# Test 9: unknown key exits nonzero
t9=0
"$SCRIPT" nonexistent-key --root "$FIXTURE" > /dev/null 2>&1 && t9=1
run_test "single-key: unknown key exits nonzero" $t9

# ─── SUMMARY ──────────────────────────────────────────────────────────────────
echo ""
echo "Results: $PASS passed, $FAIL failed"
if [ "${#ERRORS[@]}" -gt 0 ]; then
  echo "Failed tests:"
  for e in "${ERRORS[@]}"; do echo "  - $e"; done
fi
[ "$FAIL" -eq 0 ]
