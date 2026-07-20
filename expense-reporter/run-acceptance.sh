#!/bin/bash
# run-acceptance.sh — pre-flight checks + run acceptance test suite
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

echo "=== Acceptance Test Runner ==="

# Modes:
#   default      deterministic group only (go test -short): Ollama-gated tests
#                skip via extern.RequireOllama, no Ollama needed, tight timeout
#   -full        entire suite including Ollama-gated tests (requires Ollama up)
FULL=0
for arg in "$@"; do
  [ "$arg" = "-full" ] && FULL=1
done

export EXPENSE_WORKBOOK_PATH="${EXPENSE_WORKBOOK_PATH:-$(realpath "$SCRIPT_DIR/../Planilha_Normalized_Final_copy.xlsx" 2>/dev/null)}"
# Pre-flight: workbook
if [ -z "${EXPENSE_WORKBOOK_PATH:-}" ]; then
  echo "WARNING: EXPENSE_WORKBOOK_PATH is not set."
  echo "         Tests that require workbook insertion will be skipped."
  echo "         Set it to the absolute path of your Excel workbook to run all tests."
fi

# Pre-flight: Ollama (full mode only — the deterministic group never calls it)
if [ "$FULL" = "1" ]; then
  echo "[1/3] Checking Ollama..."
  if ! curl -sf http://localhost:11434/api/tags > /dev/null 2>&1; then
    echo "ERROR: Ollama is not reachable at http://localhost:11434"
    echo "       Start Ollama and ensure a classifier model is available."
    exit 1
  fi
  echo "      OK — Ollama is up"
else
  echo "[1/3] Deterministic mode (-short): skipping Ollama pre-flight"
fi

# Pre-flight: build
echo "[2/3] Building binary..."
if ! go build ./... > /dev/null 2>&1; then
  echo "ERROR: go build failed"
  go build ./...
  exit 1
fi
echo "      OK — build succeeded"

# Run acceptance tests
# Usage: ./run-acceptance.sh [-full] [TestFilter] [-keep-on-failure] [-keep-artifacts]
echo "[3/3] Running acceptance tests..."
RUN_FILTER=""
EXTRA_FLAGS=""
for arg in "$@"; do
  case "$arg" in
    -full) ;; # handled above
    -keep-on-failure|-keep-artifacts) EXTRA_FLAGS="$EXTRA_FLAGS $arg" ;;
    *) RUN_FILTER="$arg" ;;
  esac
done

if [ "$FULL" = "1" ]; then
  # Full suite measured 1989s (session 61) with 2.4x machine variance from
  # Ollama model-load contention — 3600s keeps headroom over the worst case.
  MODE_FLAGS="-timeout 3600s"
else
  MODE_FLAGS="-short -timeout 600s"
fi

if [ -n "$RUN_FILTER" ]; then
  go test -tags=acceptance -v $MODE_FLAGS -run "$RUN_FILTER" ./test/... $EXTRA_FLAGS
else
  go test -tags=acceptance -v $MODE_FLAGS ./test/... $EXTRA_FLAGS
fi

echo ""
echo "=== Done ==="
