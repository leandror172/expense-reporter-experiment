#!/bin/bash
# run-acceptance.sh — pre-flight checks + run acceptance test suite
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

echo "=== Acceptance Test Runner ==="

# Pre-flight: workbook
if [ -z "${EXPENSE_WORKBOOK_PATH:-}" ]; then
  echo "WARNING: EXPENSE_WORKBOOK_PATH is not set."
  echo "         Tests that require workbook insertion will be skipped."
  echo "         Set it to the absolute path of your Excel workbook to run all tests."
fi

# Pre-flight: Ollama
echo "[1/3] Checking Ollama..."
if ! curl -sf http://localhost:11434/api/tags > /dev/null 2>&1; then
  echo "ERROR: Ollama is not reachable at http://localhost:11434"
  echo "       Start Ollama and ensure a classifier model is available."
  exit 1
fi
echo "      OK — Ollama is up"

# Pre-flight: build
echo "[2/3] Building binary..."
if ! go build ./... > /dev/null 2>&1; then
  echo "ERROR: go build failed"
  go build ./...
  exit 1
fi
echo "      OK — build succeeded"

# Run acceptance tests
echo "[3/3] Running acceptance tests..."
go test -tags=acceptance -v -timeout 1800s ./test/...

echo ""
echo "=== Done ==="
