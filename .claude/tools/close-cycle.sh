#!/usr/bin/env bash
#
# close-cycle.sh — run a real monthly close, with an undo and two reconciliations.
#
# WHY THIS IS TWO PHASES, AND WHY IT MUST STAY THAT WAY
# ----------------------------------------------------
# The chain is  batch-auto -> review -> apply -> generate-workbook.  The `review`
# step renders an HTML page that a HUMAN categorises in a browser; the page exports
# reviewed.json via a Blob download (internal/review/template/review.html,
# exportReviewed()).  There is no headless path.
#
# The tempting "improvement" is to synthesise reviewed.json from the model's own
# predictions so the whole thing runs as one command.  DO NOT.  That makes this
# script a hand-authored stand-in for a producer it does not implement — precisely
# the T-54 shape (a reader whose only inputs were fixtures written to satisfy it,
# which passed for months while every real file failed).  T-61 records the same rule
# for this exact file.  A real run through the browser is also the ONLY thing in the
# repo that executes exportReviewed(), which is the one hop no Go test reaches.
#
# WHY THERE IS NO SANDBOX INSTALL ROOT
# ------------------------------------
# Session 68's scout ran in a scratch install root because the chain had never run
# and its failure modes were unknown.  They are known now.  Both logs are append-only
# JSONL, so a snapshot taken before the run is a COMPLETE undo — equivalent insurance,
# without a second full review pass by the human.  The snapshot is why the run
# directory is kept: the stored run IS the undo.
#
# Note apply can leave a PARTIAL installment series on a mid-append I/O failure and
# has no ledger to warn (T-60).  `restore` is currently the only recovery for that.
#
# Usage:
#   close-cycle.sh prepare <input.csv> <year>
#   close-cycle.sh finish  <run-dir> <reviewed.json>
#   close-cycle.sh restore <run-dir>
#
set -euo pipefail

readonly REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
# The INSTALL ROOT, not just the module dir.  config.json, taxonomy.json and both
# JSONL logs all resolve against filepath.Dir(os.Executable()) — so the binary must
# sit here, beside config/, or it reads a different configuration entirely.
readonly INSTALL_ROOT="$REPO_ROOT/expense-reporter"
readonly BIN="$INSTALL_ROOT/expense-reporter"
# --data-dir is the TRAINING data dir and is used RAW (batch_auto.go -> LoadKeywordIndex),
# so it resolves against the CWD, not the binary.  Always pass it absolute: that
# asymmetry between "resolved against the exe" and "resolved against the CWD" inside
# one command is the single easiest thing here to get subtly wrong.
readonly DATA_DIR="$REPO_ROOT/data/classification"
readonly CLASSIF_LOG="$INSTALL_ROOT/classifications.jsonl"
readonly EXPENSE_LOG="$INSTALL_ROOT/expenses_log.jsonl"

die() { printf 'ERROR: %s\n' "$*" >&2; exit 1; }
note() { printf '%s\n' "$*" >&2; }

# ---------------------------------------------------------------------------
# Preflight
# ---------------------------------------------------------------------------

require_tools() {
  command -v jq >/dev/null || die "jq is required"
  command -v go >/dev/null || die "go is required"
  command -v sha256sum >/dev/null || die "sha256sum is required"
}

# A foreign resident model turns a ~110s classify run into a multi-thousand-second
# one (recorded in the pre-session reading guide), so this reports rather than fails.
require_ollama() {
  curl -sf localhost:11434/api/version >/dev/null \
    || die "Ollama is not reachable at localhost:11434 — batch-auto cannot classify"
  local resident
  resident="$(curl -sf localhost:11434/api/ps | jq -r '.models[]?.name' | tr '\n' ' ')"
  [ -n "${resident// /}" ] && note "NOTE: models already resident in VRAM: $resident"
  return 0
}

# Build INTO the install root so the fresh binary is the one that reads config/.
build_binary() {
  note "Building $BIN ..."
  ( cd "$INSTALL_ROOT" && go build -o expense-reporter ./cmd/expense-reporter )
}

# ---------------------------------------------------------------------------
# The snapshot — this is the undo
# ---------------------------------------------------------------------------

# wc -l counts NEWLINES, so it undercounts a file whose last line has none — which an
# O_APPEND log can legitimately be.  awk counts records and is exact.
count_lines() { [ -f "$1" ] && awk 'END{print NR}' "$1" || echo 0; }

# Two snapshots are taken, and they serve DIFFERENT purposes — do not collapse them.
#   logs-before/      taken before batch-auto. The restore point: the undo for the whole close.
#   logs-after-batch/ taken after batch-auto, before apply. The BASELINE for finish's
#                     reconciliation, because apply's dedup set is the classifications log
#                     as it stands once batch-auto has written its auto-inserted rows.
# Measuring finish against logs-before instead mixes two points in time: the row delta would
# include batch-auto's appends while the dedup set would predate them. That cancels out only
# when the reviewer confirms every auto-inserted row, and fires spuriously the moment one is
# skipped — reporting a T-21 regression that never happened.
snapshot_logs() {
  local dest="$1"
  mkdir -p "$dest"
  cp "$CLASSIF_LOG" "$dest/classifications.jsonl"
  cp "$EXPENSE_LOG" "$dest/expenses_log.jsonl"
  ( cd "$dest" && sha256sum ./*.jsonl > SHA256SUMS )
  note "Snapshot -> $(basename "$dest"): $(count_lines "$CLASSIF_LOG") classifications, $(count_lines "$EXPENSE_LOG") expense rows"
}

restore_logs() {
  local run_dir="$1"
  [ -d "$run_dir/logs-before" ] || die "no snapshot in $run_dir"
  ( cd "$run_dir/logs-before" && sha256sum -c SHA256SUMS --quiet ) \
    || die "snapshot in $run_dir is itself corrupt — refusing to restore from it"
  cp "$run_dir/logs-before/classifications.jsonl" "$CLASSIF_LOG"
  cp "$run_dir/logs-before/expenses_log.jsonl" "$EXPENSE_LOG"
  note "Restored both logs from $run_dir/logs-before"
}

# ---------------------------------------------------------------------------
# Assertion: no row is silently lost between the input CSV and the CSVs review reads
# ---------------------------------------------------------------------------

# classified.csv has a header and one row per input line; review.csv is the subset with
# auto_inserted == false (which includes unparsed rows — they are not auto-inserted).
#
# rows_in must count what batch.CSVReader.Read actually yields, NOT the raw line count:
# that reader drops blank lines and lines starting with '#'. Counting raw lines would make
# a statement with one trailing blank line abort this check — after batch-auto had already
# spent minutes classifying and appended to the real log.
count_input_rows() { grep -cvE '^[[:space:]]*($|#)' "$1"; }

assert_row_accounting() {
  local input_csv="$1" run_dir="$2"
  local rows_in classified review auto
  rows_in="$(count_input_rows "$input_csv")"
  classified=$(( $(count_lines "$run_dir/classified.csv") - 1 ))
  review=$(( $(count_lines "$run_dir/review.csv") - 1 ))
  auto=$(( classified - review ))

  printf 'Row accounting: %d in -> %d classified (%d auto, %d for review)\n' \
    "$rows_in" "$classified" "$auto" "$review"
  [ "$classified" -eq "$rows_in" ] \
    || die "row loss: $rows_in input rows produced $classified classified rows"
}

# ---------------------------------------------------------------------------
# Commands
# ---------------------------------------------------------------------------

# An unparsed row is written to classified.csv with EVERY field after the raw line empty
# (cmd.TestWriteClassifiedCSV_UnparsedRowKeepsItsRawLine pins that shape), so this counts
# rejects from the CSV rather than from batch-auto's own "Errors: N" line. Deriving it from
# the summary would make the check a restatement of what the command just claimed.
count_rejected_rows() { grep -c ';;;;;0\.0000;false;$' "$1" || true; }

# failed.csv must exist EXACTLY when something was rejected: its presence is the signal that
# this run needs a human, so an empty file would report a problem that did not happen and a
# missing one would hide rows that reach no other durable artifact.
assert_failed_file_matches_rejects() {
  local run_dir="$1"
  local rejected failed_lines
  rejected="$(count_rejected_rows "$run_dir/classified.csv")"

  if [ "$rejected" -eq 0 ]; then
    [ -f "$run_dir/failed.csv" ] \
      && die "no row was rejected, but failed.csv exists — its presence means 'rows need a human'"
    printf 'Rejected rows: 0 (no failed.csv, as expected)\n'
    return 0
  fi

  [ -f "$run_dir/failed.csv" ] \
    || die "$rejected row(s) were rejected but failed.csv was not written — they reach no other durable artifact"
  failed_lines="$(grep -cvE '^[[:space:]]*($|#)' "$run_dir/failed.csv")"
  printf 'Rejected rows: %d, listed in failed.csv: %d\n' "$rejected" "$failed_lines"
  [ "$failed_lines" -eq "$rejected" ] \
    || die "failed.csv lists $failed_lines row(s) but $rejected were rejected"
}

cmd_prepare() {
  local input_csv="$1" year="$2"
  [ -f "$input_csv" ] || die "no such CSV: $input_csv"
  [[ "$year" =~ ^[0-9]{4}$ ]] || die "year must be 4 digits, got: $year"

  require_tools
  require_ollama
  build_binary

  local run_dir="$REPO_ROOT/.claude/scratch/close-$(date +%Y%m%d-%H%M%S)"
  mkdir -p "$run_dir"
  printf '%s\n' "$year" > "$run_dir/YEAR"
  printf '%s\n' "$(cd "$(dirname "$input_csv")" && pwd)/$(basename "$input_csv")" > "$run_dir/INPUT"

  snapshot_logs "$run_dir/logs-before"

  # From here on the run can write to the real logs, so any abort must name the undo.
  # batch-auto's failure-honesty design appends some rows and THEN exits non-zero, and
  # T-60 puts partial installment series on exactly this path — the failure case is
  # precisely when the restore line is most needed and least likely to be remembered.
  trap 'printf "\nAborted. To undo anything already written:\n  %s restore %s\n" "$0" "$run_dir" >&2' ERR

  # --output-dir is mandatory here.  Its default is "same directory as the input CSV",
  # and the real input lives beside classified.csv / review.csv left over from earlier
  # sessions — files outside the repo, which git would not bring back.
  note "Running batch-auto (this classifies every row; expect minutes) ..."
  "$BIN" batch-auto "$input_csv" \
    --year "$year" \
    --data-dir "$DATA_DIR" \
    --output-dir "$run_dir"

  # Baseline for finish's reconciliation: taken AFTER batch-auto's appends, BEFORE apply.
  snapshot_logs "$run_dir/logs-after-batch"

  # Both assertions run BEFORE the review page, because both read only batch-auto's outputs
  # and `review` exits non-zero when EVERY row was rejected ("no rows to review", T-69).
  # Ordering them after it meant the reject assertion was skipped in exactly the case it
  # most needs to check -- found by smoke-testing this script against an all-malformed batch.
  assert_row_accounting "$input_csv" "$run_dir"
  assert_failed_file_matches_rejects "$run_dir"

  note "Building the review page ..."
  "$BIN" review "$run_dir/classified.csv" --output "$run_dir/review.html" --force

  cat >&2 <<EOF

--- prepare complete -------------------------------------------------------
Run dir: $run_dir

NEXT: open this in a browser, categorise, and click Export:
  $run_dir/review.html

The page downloads reviewed.json through the browser. Pass the file it actually
saved -- do NOT assume ~/Downloads: the review runs in a Windows browser, whose
download folder has been observed on more than one drive (E:\\...\\Downloads =
/mnt/e/.../Downloads). Then:
  $0 finish $run_dir /path/to/the/reviewed.json your browser saved

To undo everything batch-auto just appended:
  $0 restore $run_dir
----------------------------------------------------------------------------
EOF
}

# ---------------------------------------------------------------------------
# Assertion: a reviewed installment purchase lands as N rows, not one (T-21)
# ---------------------------------------------------------------------------

# EXPECTED is derived independently of apply's own accounting: sum `installments`
# over the entries apply would treat as NEW (confirmed/corrected, and whose id is not
# already in the pre-run classifications log — that lookup is apply's dedup, see
# handleActiveEntry).  ACTUAL is the row delta on the expense log.  Deriving expected
# from the reviewed file rather than from apply's summary is what makes this a check
# and not a restatement of what apply just claimed.
sum_installments_for_new_entries() {
  local reviewed_json="$1" seen_ids_file="$2"
  jq --rawfile seen "$seen_ids_file" '
    ($seen | split("\n") | map(select(length > 0))) as $seen_ids
    | [ .entries[]
        | select(.action == "confirmed" or .action == "corrected")
        | . as $e
        | select(($seen_ids | index($e.id)) == null)
        | ($e.installments
           // error("entry \($e.id): no installments count in reviewed.json — the browser export is broken (T-61)"))
        | if (type != "number") or (. < 1) or (. != floor)
          then error("entry \($e.id): invalid installments count \(.)")
          else . end
      ]
    | add // 0
  ' "$reviewed_json"
}

assert_installment_rows_reconcile() {
  local reviewed_json="$1" run_dir="$2"
  local base="$run_dir/logs-after-batch"   # NOT logs-before — see snapshot_logs
  [ -d "$base" ] || die "missing $base — this run dir predates the two-snapshot fix"
  local seen_ids="$run_dir/seen-ids.txt"
  jq -r '.id // empty' "$base/classifications.jsonl" > "$seen_ids"

  local expected actual
  expected="$(sum_installments_for_new_entries "$reviewed_json" "$seen_ids")"
  actual=$(( $(count_lines "$EXPENSE_LOG") - $(count_lines "$base/expenses_log.jsonl") ))

  printf 'Installment reconciliation: expected %d rows, log grew by %d\n' "$expected" "$actual"
  [ "$expected" -eq "$actual" ] || die \
    "installment under/over-recording: reviewed.json implies $expected rows, the log grew by $actual (T-21 regression)"
}

# A reviewed.json exported from a page left open from an EARLIER run applies cleanly,
# because apply recomputes ids from the date rather than trusting them — so a stale
# export is silent.  Its export timestamp predating this run's queue is the discriminator.
assert_export_is_from_this_run() {
  local reviewed_json="$1" run_dir="$2"
  local reviewed_at queue_built
  # Both sides are truncated to whole seconds before comparing. The browser emits
  # milliseconds ("...:00.000Z") and `date` does not ("...:00Z"), and '.' sorts BELOW 'Z' —
  # so an untruncated comparison calls a same-second export stale.
  reviewed_at="$(jq -r '.reviewedAt // empty' "$reviewed_json" | sed 's/\.[0-9]*Z$/Z/')"
  [ -n "$reviewed_at" ] || die "reviewed.json has no reviewedAt timestamp"
  queue_built="$(date -u -r "$run_dir/classified.csv" +%Y-%m-%dT%H:%M:%SZ)"
  [[ "$reviewed_at" > "$queue_built" ]] || die \
    "reviewed.json was exported at $reviewed_at, before this run's queue was built at $queue_built — that is a stale page from an earlier run"
}

cmd_finish() {
  local run_dir="$1" reviewed_json="$2"
  [ -d "$run_dir" ] || die "no such run dir: $run_dir"
  [ -f "$reviewed_json" ] || die "no such reviewed file: $reviewed_json"
  require_tools

  local year; year="$(cat "$run_dir/YEAR")"
  assert_export_is_from_this_run "$reviewed_json" "$run_dir"
  # The run dir is the obvious place to drop the export, and cp refuses to copy a file onto
  # itself with exit 1 -- which set -e turns into an abort before apply ever runs. Compare
  # resolved paths rather than the strings, so ./x and $PWD/x are recognised as the same file.
  local stored="$run_dir/reviewed.json"
  if [ "$(cd "$(dirname "$reviewed_json")" && pwd)/$(basename "$reviewed_json")" \
       != "$(cd "$(dirname "$stored")" && pwd)/$(basename "$stored")" ]; then
    cp "$reviewed_json" "$stored"
  fi

  note "Applying reviewed entries ..."
  "$BIN" apply "$run_dir/reviewed.json" --year "$year"

  assert_installment_rows_reconcile "$run_dir/reviewed.json" "$run_dir"

  # --taxonomy is deliberately OMITTED: config supplies taxonomy_path since S5, and
  # omitting it is what proves that on the live path.  --entries is deliberately
  # PASSED: omitting it is how you ask for an empty year skeleton.
  # generate-workbook warn-skips an entry whose subcategory is not in the taxonomy and
  # still exits 0 — a row can vanish from the workbook with no failure signal (this is
  # how the S2 taxonomy drift stayed invisible). The picker now shares taxonomy.json so a
  # fresh pick cannot drop, but the workbook is built from the WHOLE log, so a historical
  # row carrying a retired leaf still can. Report it; do not fail on it.
  note "Generating the workbook ..."
  "$BIN" generate-workbook \
    --entries "$EXPENSE_LOG" \
    --output "$run_dir/workbook-$year.xlsx" \
    --year "$year" 2> "$run_dir/generate-stderr.txt" || { cat "$run_dir/generate-stderr.txt" >&2; return 1; }
  # Captured to a file rather than piped through `tee` in a process substitution: bash does
  # not wait on process substitutions, so the grep below could read a half-written file.
  cat "$run_dir/generate-stderr.txt" >&2

  local skipped
  skipped="$(grep -ci 'skip' "$run_dir/generate-stderr.txt" || true)"
  [ "$skipped" -gt 0 ] && note "WARNING: generate-workbook skipped $skipped entr(ies) — see $run_dir/generate-stderr.txt"

  cat >&2 <<EOF

--- close complete ---------------------------------------------------------
Workbook: $run_dir/workbook-$year.xlsx
Logs are now LIVE. To undo the whole close:
  $0 restore $run_dir
----------------------------------------------------------------------------
EOF
}

main() {
  local sub="${1:-}"
  shift || true
  case "$sub" in
    prepare) [ $# -eq 2 ] || die "usage: $0 prepare <input.csv> <year>"; cmd_prepare "$@" ;;
    finish)  [ $# -eq 2 ] || die "usage: $0 finish <run-dir> <reviewed.json>"; cmd_finish "$@" ;;
    restore) [ $# -eq 1 ] || die "usage: $0 restore <run-dir>"; restore_logs "$1" ;;
    *) die "usage: $0 {prepare|finish|restore} ..." ;;
  esac
}

# Only dispatch when EXECUTED, never when SOURCED. Sourcing is how the assertions above
# get exercised in isolation against synthetic logs — without this guard, sourcing runs
# main, hits the usage error and exits, so every such test reports failure identically
# and a genuinely broken assertion is indistinguishable from a working one.
if [ "${BASH_SOURCE[0]}" = "$0" ]; then
  main "$@"
fi
