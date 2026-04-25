# Bug: batch-auto loses classification output on workbook insertion failure

**Date:** 2026-04-20  
**Status:** Reported  

## Issue
When `batch-auto` fails to find/access the workbook, it exits without writing the classification output files (classified.csv, review.csv, rollover.csv).

## Reproduction
```bash
go run ./cmd/expense-reporter batch-auto \
  /home/leandror/workspaces/expenses/extracted_entities.csv \
  --data-dir /mnt/i/workspaces/expenses/code/data/classification \
  --output-dir /tmp/expense-output \
  --model my-classifier-q3
```

## Error
```
Error: workbook not found: /home/leandror/.cache/go-build/68/Planilha_BMeFBovespa_Leandro_OrcamentoPessoal-2025.xlsx
exit status 1
```

## Current behavior
1. Classification runs successfully ✓
2. Workbook insertion fails ✗
3. **No output CSV files are written** ✗
4. **All classification work is lost** ✗

## Expected behavior
- Classification output files (classified.csv, review.csv, rollover.csv) should be written to --output-dir even if workbook insertion fails
- OR: Add early validation to check workbook exists before running expensive classification

## Impact
Loss of work when workbook path is misconfigured or unavailable. 1601 rows were classified but output discarded.

## Workaround
Use `--dry-run` first to validate and generate classification CSVs without requiring workbook:
```bash
go run ./cmd/expense-reporter batch-auto \
  /home/leandror/workspaces/expenses/extracted_entities.csv \
  --data-dir /mnt/i/workspaces/expenses/code/data/classification \
  --output-dir /tmp/expense-output \
  --dry-run
```
