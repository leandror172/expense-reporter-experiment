# Oracle Freeze Placeholder — WS-C Step 5

This directory will hold the frozen `internal/inspect` dump for the income-route workbook.

## Freeze is BLOCKED until WS-C implementation is complete

Steps before freezing:
1. WS-C Steps 1–6 must land (model, loader, router, generator, taxonomy data, CLI flag)
2. Run `generate-workbook --income-entries income-entries.jsonl --entries entries.jsonl --year 2026`
   against the `generate-income` fixture taxonomy
3. Dump via `internal/inspect.DumpWorkbook`
4. Review the Receitas sheet manually — confirm 3-level grouping (Receitas → block → subline rows)
5. Verify signed sums per block:
   - Salário Jan: 5000 + (−550) + (−300) = 4150
   - Salário Fev: 5000 + (−550) = 4450
   - Salário block total (across months): 10000 − 1100 − 300 = 8600
   - Férias Jul: 8000 + (−400) = 7600
6. Advisor sign-off on the frozen dump
7. Commit the dump here and remove this file

## DO NOT fabricate dump files here — the frozen oracle must come from the real generator.
