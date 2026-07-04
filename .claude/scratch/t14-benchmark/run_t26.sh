#!/usr/bin/env bash
# T-26 benchmark runner — think-on confirmation of the T-22 type descriptions.
# DO NOT EXECUTE without explicit instruction.
#
# Supersedes run_t22.sh's STALE taxonomy-swap mechanism. Post-adoption (session 47)
# the English type descriptions are the DEFAULT, merged in at classify time from the
# tracked sidecar config/type-descriptions.json. So the two A/B conditions are toggled
# by the SIDECAR's presence, NOT by swapping taxonomy.json:
#     descen = sidecar present  (config/type-descriptions.json in place — adopted default)
#     nodesc = sidecar absent   (sidecar temporarily removed for the run)
# taxonomy.json is never touched here. Portuguese (descpt) was dropped after the
# no-think A/B; this run only needs nodesc + descen.
#
# For each condition it runs the 300-item accuracy pass and the 20-item OOD pass
# against my-classifier-q3 and writes:
#     results-<mode>-<cond>-my-classifier-q3.jsonl
#     ood-results-<mode>-<cond>-my-classifier-q3.jsonl
# The <mode> segment keeps these DISTINCT from the existing no-think A/B files
# (results-<cond>-my-classifier-q3.jsonl), so run_benchmark.py's resume-by-id does
# NOT mistake the old no-think results for this run's work. score.py globs
# results-*.jsonl and derives the label by stripping results-/.jsonl, so it will
# report every condition (old + new) side by side.
#
# Usage:
#   ./run_t26.sh          # default: think (q3 think-on, ~12-14s/item — the T-26 target)
#   ./run_t26.sh think
#   ./run_t26.sh nothink  # --think=false (q3 no-think, ~1.5s/item)
#
# RESUMABLE: each run_benchmark.py pass appends+flushes per item and skips ids already
# present in its results file, so this script may be killed and re-run at any time; it
# resumes condition-by-condition. The EXIT trap restores the sidecar to its adopted
# (present) state on every exit path, so an interrupt never corrupts the tracked config.

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../../.." && pwd)"
APP_DIR="$REPO_ROOT/expense-reporter"
CONFIG_DIR="$APP_DIR/config"
SIDECAR_LIVE="$CONFIG_DIR/type-descriptions.json"
SIDECAR_BACKUP="$SCRIPT_DIR/type-descriptions.json.t26.bak"   # *.bak is gitignored
MODEL="my-classifier-q3"

MODE="${1:-think}"
if [[ "$MODE" != "think" && "$MODE" != "nothink" ]]; then
    echo "Usage: $0 [think|nothink]" >&2
    exit 1
fi

EXTRA_ARGS=()
if [[ "$MODE" == "nothink" ]]; then
    EXTRA_ARGS=(--extra-args="--think=false")
fi

# The adopted sidecar must be present at start — it is the source of the descen
# condition's content and the state we restore to on exit.
if [[ ! -f "$SIDECAR_LIVE" ]]; then
    echo "ERROR: adopted sidecar $SIDECAR_LIVE not found — cannot produce the descen condition" >&2
    exit 1
fi

# Build the binary so the run never drives stale code. `go build ./...` does NOT emit
# the cmd binary; it must be built explicitly to expense-reporter/expense-reporter,
# which is the path run_benchmark.py invokes and whose dir the config paths resolve against.
echo "[run_t26] building expense-reporter binary"
( cd "$APP_DIR" && go build -o expense-reporter ./cmd/expense-reporter )

# Back up the adopted sidecar once; restore it on ANY exit path (success, error, Ctrl-C).
cp "$SIDECAR_LIVE" "$SIDECAR_BACKUP"
trap 'echo "[run_t26] restoring adopted sidecar type-descriptions.json"; cp "$SIDECAR_BACKUP" "$SIDECAR_LIVE"' EXIT

for cond in nodesc descen; do
    echo "=================================================================="
    echo "[run_t26] condition=$cond mode=$MODE model=$MODEL"
    echo "=================================================================="

    if [[ "$cond" == "descen" ]]; then
        echo "[run_t26] sidecar PRESENT (English descriptions active)"
        cp "$SIDECAR_BACKUP" "$SIDECAR_LIVE"
    else
        echo "[run_t26] sidecar ABSENT (no descriptions)"
        rm -f "$SIDECAR_LIVE"
    fi

    echo "[run_t26] running 300-item accuracy pass (prefix=results-$MODE-$cond)"
    python3 "$SCRIPT_DIR/run_benchmark.py" \
        --model "$MODEL" \
        --prefix "results-$MODE-$cond" \
        "${EXTRA_ARGS[@]}"

    echo "[run_t26] running 20-item OOD pass (prefix=ood-results-$MODE-$cond)"
    python3 "$SCRIPT_DIR/run_benchmark.py" \
        --model "$MODEL" \
        --sample "$SCRIPT_DIR/ood.jsonl" \
        --prefix "ood-results-$MODE-$cond" \
        "${EXTRA_ARGS[@]}"

    echo "[run_t26] condition=$cond mode=$MODE done"
done

echo "[run_t26] all conditions complete for mode=$MODE"
echo "[run_t26] score with: python3 $SCRIPT_DIR/score.py --dir $SCRIPT_DIR"
