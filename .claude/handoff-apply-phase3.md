# Handoff — `apply` Command Phase 3+

**Date:** 2026-05-29
**Branch:** `feat/apply-command`
**Last commit:** `21090d8 feat(apply): scaffold apply command — Phase 0-2`

---

## What was done this session

### Status report
All Layer 5 tasks (5.1–5.8) confirmed complete via codebase check. `review` command (RUI-1) merged. Decided to work on RUI-3 (`apply` command).

### Plan
Full plan written at `.claude/plans/apply-command.md`. Read it. Key decisions:

- **D1 — Idempotency guard:** Use `feedback.FindLatestEntry(classifPath, entry.ID)`. If found → row already in workbook → skip insertion. Uses `classifications.jsonl` as authoritative state. Handles double-insertion AND apply-twice-is-safe.
- **D2 — Corrected already-inserted rows:** Feedback-only (`NewCorrectedEntry` + `Append`). No workbook re-insertion. Printed in `⚠` summary section so user knows to fix manually.
- **D3 — Year:** `--year` flag on `apply`, default `time.Now().Year()`. `ParseDateWithYear` added to `pkg/utils/date.go`.
- **Sheet resolution:** `apply` uses the excel layer directly (`FindSubcategoryRowBatch` + `AllocateEmptyRows` + `WriteBatchExpenses`) with explicit `reviewed.sheet` from reviewed.json. Does NOT use `workflow.InsertBatchExpensesFromClassified` or the resolver pipeline. Rationale: the user already resolved ambiguity in the UI; the resolver is for when sheet is unknown.
- **reviewed.json exports ALL rows** (including pending/already-inserted) with `action: confirmed|corrected|skipped|pending`. `apply` skips `pending` and `skipped`.
- **Model sentinel:** Use `"review"` as model name for entries without a prior classifications.jsonl entry.

### Completed phases
- **Phase 0** — Acceptance test RED: `test/apply_test.go`, `test/fixtures/apply-basic/{reviewed.json, seed-classifications.jsonl, expected-feedback.jsonl}`, `RunApply` in `test/actions/commands.go`. Test fails with `unknown command "apply"` — correct RED state.
- **Phase 1a** — `utils.ParseDateWithYear(dateStr string, year int) (time.Time, error)` in `pkg/utils/date.go` + 7 tests.
- **Phase 1b** — `SheetName string` field added to `models.ClassifiedExpense` (zero value = use resolver as before; no existing callers affected).
- **Phase 1c** — No code change needed. `apply` bypasses the resolver entirely.
- **Phase 2** — `internal/apply/` package: `types.go` (ReviewedFile, ReviewedEntry, ReviewedLocation, action constants, IsInsertable(), IsAlreadyHandled()), `reader.go` (ReadReviewed + validateEntries), `reader_test.go` (4 passing tests).

### NOT YET DONE
- **Phase 3** — `cmd/apply.go` cobra command. Ollama was called but timed out (cold start), session ended before retry. **Start here.**
- **Phase 4** — Build/vet/test/smoke.

---

## Phase 3: what to implement (`cmd/apply.go`)

### File location
`expense-reporter/cmd/expense-reporter/cmd/apply.go`

### Cobra pattern reference
Read `cmd/expense-reporter/cmd/review.go` — same package, same wiring pattern.

### Package-level vars and init()
```
applyWorkbook string  // --workbook flag
applyYear     int     // --year flag, default time.Now().Year()
applyDryRun   bool    // --dry-run flag
```

`init()` calls `rootCmd.AddCommand(applyCmd)` and registers all three flags.

### Key imports needed
```go
"expense-reporter/internal/apply"
"expense-reporter/internal/excel"
"expense-reporter/internal/feedback"
"expense-reporter/internal/models"
"expense-reporter/pkg/utils"
"expense-reporter/internal/config"
```

### Core logic (describe to Ollama — do not spell out code)

`runApply(cmd, args)`:
1. Load config (`config.Load()`), resolve workbook path (use `applyWorkbook` flag or `GetWorkbookPath()`)
2. Validate workbook exists (`excel.ValidateWorkbook`)
3. `apply.ReadReviewed(args[0])` → ReviewedFile
4. Get classifPath from `appCfg.ClassificationsFilePath()`, expensesLogPath from `appCfg.ExpensesLogFilePath()`
5. Call `processEntries(...)` → counts + alreadyInsertedCorrections list
6. Call `printSummary(...)`

`processEntries(entries, classifPath, expensesLogPath, workbookPath string, year int, dryRun bool)`:
- For each entry:
  - Skip `pending`/`skipped` → increment pending/skipped count
  - For `confirmed`/`corrected`: call `feedback.FindLatestEntry(classifPath, entry.ID)`
    - Found + confirmed → no-op
    - Found + corrected → `feedback.NewCorrectedEntry(...)` + `feedback.Append(...)`, collect in alreadyInsertedCorrections
    - Not found → collect in newRows slice
- Call `insertNewRows(newRows, workbookPath, classifPath, expensesLogPath, year, dryRun)` for new rows
- Return counts + alreadyInsertedCorrections

`insertNewRows(...)`:
For each new row:
1. `utils.ParseDateWithYear(entry.Date, year)` → time.Time
2. Build `models.Expense{Item, Date, Value, Subcategory: entry.Reviewed.Subcategory}`
3. `excel.GetMonthColumns(date.Month())` → itemCol, dateCol, valueCol
4. `excel.FindSubcategoryRowBatch(workbookPath, []SubcategoryLookupRequest{{SheetName: entry.Reviewed.Sheet, Subcategory: entry.Reviewed.Subcategory}})` → subcatRow
5. `excel.AllocateEmptyRows(workbookPath, []EmptyRowRequest{{SheetName, ColumnLetter: itemCol, StartRow: subcatRow+1, SubcategoryName: entry.Reviewed.Subcategory, ExpenseIndex: 0}})` → targetRow
6. Build `models.SheetLocation{SheetName: entry.Reviewed.Sheet, Category: entry.Reviewed.Category, SubcatRow: subcatRow, TargetRow: targetRow, MonthColumn: itemCol}`
7. If !dryRun: `excel.WriteBatchExpenses(workbookPath, []excel.ExpenseWithLocation{{Expense: &expense, Location: &location}})`
8. Write feedback entry: if `corrected` → `NewCorrectedEntry(model="review")`; if `confirmed` → `NewConfirmedEntry(model="review")`
9. `feedback.AppendExpense(expensesLogPath, ...)`

`printSummary(source string, inserted, insertedConfirmed, insertedCorrected, skipped, pending int, alreadyInsertedCorrections []apply.ReviewedEntry)`:
```
Applied <source> (N entries)
  Inserted:  N rows (X confirmed, Y corrected)
  Skipped:   N rows
  Pending:   N rows

⚠  N already-inserted rows were corrected — workbook not updated:
   <item> (<date>, R$<value>) <predicted_subcategory> → <actual_subcategory>  [logged]
```
(omit the ⚠ section entirely if alreadyInsertedCorrections is empty)

### For NewCorrectedEntry when already-inserted
`feedback.FindLatestEntry` returns the prior entry. Use `prior.Model` as the model name, `prior.PredictedSubcategory`/`prior.PredictedCategory` for the predicted values.

### For NewCorrectedEntry when new row (not previously inserted)
Use `"review"` as model name. Use `entry.Predicted.Subcategory` and `entry.Predicted.Category` for predicted values.

### For NewConfirmedEntry when new row
Use `"review"` as model name. `classifier.Result{Subcategory: entry.Reviewed.Subcategory, Category: entry.Reviewed.Category, Confidence: entry.Confidence}`.

---

## Acceptance test to turn GREEN

`TestApply_IdempotencyAndFeedback` in `expense-reporter/test/apply_test.go`

The fixture (`test/fixtures/apply-basic/reviewed.json`) has:
- Uber Centro (id: `f0c3bf1293f3`): action=confirmed, already in seed-classifications.jsonl → no-op
- Diarista Letícia (id: `733224d39e01`): action=corrected, already in seed → write corrected feedback
- Academia Smart Fit (id: `24c75fff9223`): action=pending → no-op

Expected after apply runs:
- `classifications.jsonl` = 3 lines: 2 seed entries + 1 new corrected entry for Diarista Letícia
- `expenses_log.jsonl` NOT created (no new rows inserted — all were already-inserted or pending)
- stdout contains `"workbook not updated"`
- exit code 0

Run with: `go test -tags=acceptance -run TestApply_IdempotencyAndFeedback ./test/...`

---

## Key files to read at session start

1. `.claude/plans/apply-command.md` — full design plan
2. `expense-reporter/internal/apply/types.go` — ReviewedEntry types, IsInsertable(), action constants
3. `expense-reporter/internal/apply/reader.go` — ReadReviewed
4. `expense-reporter/cmd/expense-reporter/cmd/review.go` — cobra pattern to follow
5. `expense-reporter/internal/feedback/feedback.go` — FindLatestEntry, NewConfirmedEntry, NewCorrectedEntry, Append
6. `expense-reporter/internal/feedback/expense_log.go` — AppendExpense, ExpenseEntry
7. `expense-reporter/internal/excel/reader.go` lines 13–50 — SubcategoryLookupRequest, EmptyRowRequest, FindSubcategoryRowBatch
8. `expense-reporter/internal/excel/columns.go` — GetMonthColumns
9. `expense-reporter/internal/excel/writer.go` lines 40–55 — ExpenseWithLocation, WriteBatchExpenses
10. `expense-reporter/test/apply_test.go` — the acceptance test to turn green

---

## Local model usage for Phase 3

Use `my-go-qcoder` (qwen3-coder:30b). Provide context files (see above — items 2–9). The command is complex enough to warrant the stubs-then-Ollama pattern if the first attempt has structural issues (3+ sites wrong). See `.claude/overlays/local-model-conventions.md` for the full protocol.

Try `mcp__ollama-bridge__warm_model` with `my-go-qcoder` before the first generate_code call to avoid cold-start timeout.

---

## After Phase 3: Phase 4 checklist

```bash
cd expense-reporter
go build ./...
go vet ./...
go test ./...
go test -tags=acceptance -run TestApply_IdempotencyAndFeedback ./test/...
```

If all green: commit, then do `/session-handoff` normally.

---

## Tasks tracker state

| # | Task | Status |
|---|------|--------|
| 1 | Phase 0 — Acceptance test (red) | ✅ completed |
| 2 | Phase 1a — ParseDateWithYear | ✅ completed |
| 3 | Phase 1b — SheetName on ClassifiedExpense | ✅ completed |
| 4 | Phase 1c — Resolver strategy (revised: no change needed) | ✅ completed |
| 5 | Phase 2 — internal/apply package | ✅ completed |
| 6 | Phase 3 — cmd/apply.go cobra command | 🔄 in progress — start here |
| 7 | Phase 4 — Build/vet/test/smoke | ⏳ pending |
