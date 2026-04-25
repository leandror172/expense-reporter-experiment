# Plan: Fix `batch-auto` losing classification output on workbook failure

**Bug:** `BUG_REPORT.md` (2026-04-20)
**Related latent bug:** `BUG_REPORT_DEFAULT_WORKBOOK_PATH.md` — do NOT address here
**Branch to start from:** current working branch (`feature/correct-command`) or a new branch off it; implementer's call

---

## Root cause

`cmd/expense-reporter/cmd/batch_auto.go:74-130` (`runBatchAuto`) orders work as:

1. `classifyLines(...)` — expensive, N LLM calls (1601 in the reported repro)
2. `insertClassified(...)` — fails → **`return err` exits early**
3. `writeClassifiedCSV` / `writeReviewCSV` / `writeRolloverCSV` — **never reached**

The early return on step 2 discards all classification output, even though it is already
materialized in the in-memory `results` slice.

---

## Fix strategy — two layers

Apply **both**. They are complementary, not redundant.

### Layer 1 — fail-fast workbook validation (UX)

Validate workbook existence **before** `classifyLines`, only when `!batchAutoDryRun`.
Prevents wasting ~1600 LLM calls when the workbook path is misconfigured.

Error message must be actionable:
> `workbook not found: <path>\n  Hint: use --workbook <path>, set EXPENSE_WORKBOOK_PATH, or use --dry-run`

### Layer 2 — defensive ordering (durability)

Restructure `runBatchAuto` so CSV writes always run, even if `insertClassified` returns
an error. The per-row `AutoInserted=false` annotations set inside `insertClassified`
(batch_auto.go:240-246) must still appear in `classified.csv` — so CSV writes happen
*after* the insertion call, not before.

Proposed shape:

```go
results := classifyLines(...)

classifiedPath := filepath.Join(outputDir, "classified.csv")
reviewPath := filepath.Join(outputDir, "review.csv")

var rollovers []workflow.RolloverExpense
var insertErr error
if !batchAutoDryRun {
    rollovers, insertErr = insertClassified(results, appCfg, batchAutoModel)
    // NOTE: do NOT return here — fall through to CSV writes
}

if err := writeClassifiedCSV(classifiedPath, results); err != nil {
    return fmt.Errorf("writing classified.csv: %w", err)
}
if err := writeReviewCSV(reviewPath, results); err != nil {
    return fmt.Errorf("writing review.csv: %w", err)
}

rolloverPath := ""
if len(rollovers) > 0 {
    rolloverPath = filepath.Join(outputDir, "rollover.csv")
    if err := writeRolloverCSV(rolloverPath, rollovers); err != nil {
        return fmt.Errorf("writing rollover.csv: %w", err)
    }
}

printBatchSummary(results, rollovers, batchAutoDryRun, classifiedPath, reviewPath, rolloverPath)

if insertErr != nil {
    return fmt.Errorf("workbook insertion failed (classification CSVs preserved at %s): %w",
        outputDir, insertErr)
}
return nil
```

Key invariants:
- Classification results are persisted to disk before any error is returned to the caller
- Summary is printed so the user sees what got classified
- Final error still surfaces as non-zero exit (Cobra convention); wrapping message points
  to where the preserved CSVs live

---

## Step 1 — Red unit test for Layer 1 (workbook validation)

File: `cmd/expense-reporter/cmd/batch_auto_test.go`

New test: `TestRunBatchAuto_MissingWorkbookFailsFast`

- Create a tempdir
- Write a minimal fixture CSV (1 row) to tempdir
- Set `workbookPath` to a path under tempdir that does *not* exist
- Call `runBatchAuto` via the cobra command (use `batchAutoCmd.Execute()` with args) or
  by invoking `runBatchAuto(batchAutoCmd, []string{csvPath})` directly
- Assert:
  - Returned `error` is non-nil and contains `"workbook not found"`
  - `outputDir/classified.csv` does NOT exist (proves we bailed before classification;
    no wasted LLM calls, and — importantly — we did not touch Ollama)

Use `require`/`assert` from testify (project convention; see
`memory/feedback_testify.md`).

**Caveat for the implementer:** `runBatchAuto` currently loads taxonomy and config
*before* it would validate the workbook. That's fine — those are local file reads, not
LLM calls. But make sure the workbook validation happens **before** `classifyLines`,
which is the expensive call. Place the check between `loadBatchAutoDeps` and
`classifyLines`. Use `GetWorkbookPath()` + `os.Stat` in the same pattern as the existing
check inside `insertClassified` (batch_auto.go:206-212). You can keep the existing check
in `insertClassified` as defense-in-depth, or remove it once Layer 1 always runs first —
implementer's call; no strong preference.

Run: `cd expense-reporter && go test ./cmd/expense-reporter/cmd/ -run TestRunBatchAuto_MissingWorkbookFailsFast`

Must fail red before the Layer 1 code is added.

## Step 2 — Green Layer 1

Add the workbook validation block in `runBatchAuto` between `loadBatchAutoDeps` and
`classifyLines`. Test goes green.

## Step 3 — Red acceptance test for Layer 2 (corrupt workbook)

File: `expense-reporter/test/batch_auto_test.go`

New scenario: `TestBatchAuto_InsertFailure_PreservesCSVs`

Approach — **real failure via fixture**, no mocks, no Go code refactor:

- Create a fixture directory `expense-reporter/test/fixtures/batch-auto-corrupt-workbook/`
  with a small input CSV (2-3 rows; use the same style as `batch-auto-basic`)
- In the Given step, copy the fixture into `ctx.WorkDir`, then **write an invalid file**
  at the workbook path — e.g., a plain-text file named like the workbook
  (`Planilha_BMeFBovespa_Leandro_OrcamentoPessoal-2025.xlsx`) containing
  `"not a real xlsx"`. This:
  - Passes Layer 1's `os.Stat` check (file exists)
  - Fails inside `workflow.InsertBatchExpensesFromClassified` when `excelize` tries to
    open it
- Run `batch-auto` against the fixture with `EXPENSE_WORKBOOK_PATH` pointing at the
  corrupt file (non-dry-run)
- Verify:
  - Command exited **non-zero** (insertion genuinely failed)
  - `classified.csv` exists in output dir
  - `review.csv` exists in output dir
  - `classified.csv` has header + N data rows
  - Stderr/stdout contains something like `"classification CSVs preserved"` — optional
    but nice

**Existing verify helpers to reuse:**
- `verify.OutputFileExists("classified.csv")`
- `verify.OutputFileHasAtLeastRows("classified.csv", 1)`
- `verify.OutputFileHasColumns("classified.csv", 7)`

**New verify helper probably needed:** `verify.CommandFailed()` — mirror of
`verify.CommandSucceeded()` that asserts non-zero exit. Check `expense-reporter/test/verify/`
first; if a similar helper already exists (e.g. `CommandFailedWith`), reuse it.

**Harness caveat:** the acceptance harness currently invokes the binary and asserts
success in several places. Check how `verify.CommandSucceeded()` is wired — if the
harness records exit status into `ctx` regardless and the verify functions inspect it,
a `CommandFailed()` helper is trivial. If the harness `t.Fatal`s on non-zero exit before
verifies run, it needs a small change (an expected-failure flag on the scenario). Do
whichever requires the smaller diff; if the second option is needed, consult the user
before expanding scope.

Run: `cd expense-reporter && ./run-acceptance.sh -run TestBatchAuto_InsertFailure_PreservesCSVs`

Must fail red before Layer 2 reordering is applied.

## Step 4 — Green Layer 2

Apply the `runBatchAuto` restructure shown above. Both tests go green.

## Step 5 — Regression check

- `cd expense-reporter && go test ./...` — all 190+ unit tests green
- `cd expense-reporter && go vet ./...` — clean
- `cd expense-reporter && ./run-acceptance.sh` — full acceptance suite green
  (requires live Ollama; if unavailable, document that and run the unit suite only)

## Step 6 — Commit

Two commits, or one with both layers — implementer's call. Suggested messages:

```
fix(batch-auto): preserve classification CSVs when workbook insertion fails

Previously, `batch-auto` would discard all classification output if
workbook insertion failed after classification completed. This meant
users lost 1000+ LLM calls worth of work if the workbook path was
misconfigured or the file was corrupt.

- Validate workbook exists before running expensive classification
  (saves the LLM calls entirely when path is wrong)
- Reorder CSV writes to always run, even when insertion fails
  (preserves work done on a mid-pipeline insert failure)
- classified.csv / review.csv are always written; error is still
  surfaced to the caller with a path to the preserved CSVs

Fixes BUG_REPORT.md
```

---

## Non-goals / out of scope

- `BUG_REPORT_DEFAULT_WORKBOOK_PATH.md` (the `go run` build-cache path issue) — separate
  fix, separate PR
- Partial-insert recovery (e.g., tracking which rows made it into the workbook before
  an IO failure) — current behavior is atomic-ish via `excelize.SaveAs`; not worth
  touching here
- Adding JSON output mode to `batch-auto` — unrelated

## Style / convention reminders

- Testify `assert`/`require` for unit tests
- `//go:build acceptance` for acceptance tests
- Error wrapping with `fmt.Errorf("context: %w", err)`, never bare returns
- Don't add comments explaining *what* the code does; only *why* when non-obvious
- No backwards-compat shims — a simple reorder is fine; don't add a flag to opt into
  the new behavior

## Estimated effort

- Step 1 (red unit test): 15 min
- Step 2 (Layer 1 impl): 5 min
- Step 3 (red acceptance test + fixture + maybe CommandFailed helper): 30-45 min
- Step 4 (Layer 2 impl): 15 min
- Step 5-6 (regression + commit): 10 min

Total: ~90 min for a careful implementation. Local model should handle Layer 1 easily;
Layer 2 restructure is small enough for local model too. Acceptance fixture is
boilerplate — delegate to Ollama with `test/PATTERNS.md` + existing
`batch-auto-basic` fixture as context.
