#!/usr/bin/env bash
# T-22 benchmark runner — DO NOT EXECUTE without explicit instruction.
#
# For each of three taxonomy conditions (nodesc / descen / descpt), temporarily
# swaps expense-reporter/config/taxonomy.json for the condition's variant, then
# runs the 300-item accuracy pass and the 20-item OOD pass against
# my-classifier-q3, writing results-<cond>-my-classifier-q3.jsonl and
# ood-results-<cond>-my-classifier-q3.jsonl (matching score.py's expected glob).
#
# Usage:
#   ./run_t22.sh think     # default Ollama thinking (q3 think-on, ~12s/item)
#   ./run_t22.sh nothink   # --think=false (q3 no-think, ~1.5s/item, T-14 fast lane)
#
# The original config/taxonomy.json is backed up once to config/taxonomy.json.t22bak
# and restored on EXIT via trap, regardless of how the script terminates.

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../../.." && pwd)"
CONFIG_DIR="$REPO_ROOT/expense-reporter/config"
TAXONOMY_LIVE="$CONFIG_DIR/taxonomy.json"
TAXONOMY_BACKUP="$CONFIG_DIR/taxonomy.json.t22bak"
MODEL="my-classifier-q3"

MODE="${1:-}"
if [[ "$MODE" != "think" && "$MODE" != "nothink" ]]; then
    echo "Usage: $0 <think|nothink>" >&2
    exit 1
fi

EXTRA_ARGS=()
if [[ "$MODE" == "nothink" ]]; then
    EXTRA_ARGS=(--extra-args="--think=false")
fi

if [[ ! -f "$TAXONOMY_LIVE" ]]; then
    echo "ERROR: $TAXONOMY_LIVE not found" >&2
    exit 1
fi

# Back up the live taxonomy exactly once, restore it on any exit path.
cp "$TAXONOMY_LIVE" "$TAXONOMY_BACKUP"
trap 'echo "[run_t22] restoring original taxonomy.json"; cp "$TAXONOMY_BACKUP" "$TAXONOMY_LIVE"' EXIT

declare -A CONDITIONS=(
    [nodesc]="$TAXONOMY_BACKUP"
    [descen]="$SCRIPT_DIR/taxonomy-descen.json"
    [descpt]="$SCRIPT_DIR/taxonomy-descpt.json"
)

for cond in nodesc descen descpt; do
    variant_file="${CONDITIONS[$cond]}"
    if [[ ! -f "$variant_file" ]]; then
        echo "ERROR: variant file for condition '$cond' not found: $variant_file" >&2
        exit 1
    fi

    echo "=================================================================="
    echo "[run_t22] condition=$cond mode=$MODE model=$MODEL"
    echo "[run_t22] taxonomy source: $variant_file"
    echo "=================================================================="

    cp "$variant_file" "$TAXONOMY_LIVE"

    echo "[run_t22] running 300-item accuracy pass (prefix=results-$cond)"
    python3 "$SCRIPT_DIR/run_benchmark.py" \
        --model "$MODEL" \
        --prefix "results-$cond" \
        "${EXTRA_ARGS[@]}"

    echo "[run_t22] running 20-item OOD pass (prefix=ood-results-$cond)"
    python3 "$SCRIPT_DIR/run_benchmark.py" \
        --model "$MODEL" \
        --sample "$SCRIPT_DIR/ood.jsonl" \
        --prefix "ood-results-$cond" \
        "${EXTRA_ARGS[@]}"

    echo "[run_t22] condition=$cond mode=$MODE done"
done

echo "[run_t22] all conditions complete for mode=$MODE"
