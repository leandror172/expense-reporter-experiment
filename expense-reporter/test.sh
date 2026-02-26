#!/bin/bash
# test.sh – run all tests, working around Kaspersky blocking resolver.
# Usage:  bash test.sh          (from the expense-reporter directory)
#         bash test.sh -v       (verbose: show every test name)
#
# See Makefile comments for the full explanation of *why* resolver is special.

set -e
cd "$(dirname "$0")"   # run from script's own directory regardless of cwd

FLAGS=""
if [ "$1" = "-v" ]; then
    FLAGS="-v"
fi

echo "=== Running all tests except resolver ==="
go test $FLAGS \
    ./cmd/... \
    ./internal/batch/... \
    ./internal/cli/... \
    ./internal/excel/... \
    ./internal/models/... \
    ./internal/parser/... \
    ./internal/workflow/... \
    ./pkg/...

echo ""
echo "=== Compiling resolver tests (Kaspersky workaround) ==="
go test -c -o .resolver.test.exe ./internal/resolver

echo "=== Running resolver tests ==="
./.resolver.test.exe -test.v

echo ""
echo "=== Cleaning up ==="
rm -f .resolver.test.exe

echo ""
echo "=== All tests passed ==="
