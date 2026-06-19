# Option A — Full-cycle integration test (sandboxed), session 34

**Goal:** prove the type-persistence → routing chain through the REAL commands
(batch-auto → review → apply → generate-workbook), not just the Bf runbook.
Synthetic data must NOT touch the main database (real logs/workbook).

## Isolation
- Sandbox dir `/tmp/exp-sandbox/` with throwaway binary + `config/config.json`
  using ABSOLUTE paths → all writes land in sandbox.
- Copy the REAL workbook into the sandbox (review needs the "Referência de
  Categorias" reference sheet; apply inserts rows). Point config at the copy.
- Classifier still reads real `data/classification/` (read-only training input).

## Known quirk (pre-existing): CSV format gap
`writeClassifiedCSV` emits `auto_inserted` as `true/false`; `review.ReadQueue`
expects `1/0`. So `review` cannot read batch-auto's classified.csv directly.
Bridge inline for this test (transform true→1/false→0). Confirms RUI-4's value.

## Cycle
1. Synthesize ~6-row CSV incl. a known ambiguous leaf (Orion/Gás-type).
2. batch-auto → sandbox classified.csv + logs (BASELINE: entries type-less).
3. Bridge auto_inserted true/false → 1/0.
4. review → review.html → browser edit → export reviewed.json.
5. apply → inserts into sandbox workbook + backfills type into sandbox logs.
6. generate-workbook on sandbox log → confirm routing; compare fallback count.

## Status
Approach approved by user (session 34). Advisor consulted before execution.
Runs in parallel with the two Option B subagents (5.R4, RUI-4) in worktrees.
