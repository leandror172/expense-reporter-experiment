# Advisor Review — Session 24 (`apply` command Phase 3)
**Date:** 2026-05-29
**Context:** Post-implementation review of cmd/apply.go and session handoff notes.

---

## The one thing that matters: `insertNewRows` has a real index-aliasing bug

Your own earlier advisor call flagged "insertion path silently ships a bug if wrong; verify by reading after codegen." You acknowledged it, then verified compile/vet/test — none of which touch this path. Reading it now: there's a concrete bug.

**Mechanism.** `excel.AllocateEmptyRows` returns `map[int]int` keyed by **position in the `requests` slice** (`results[idx] = row`, where `idx` ranges over the requests you passed). It does *not* key by `EmptyRowRequest.ExpenseIndex` — that field is caller-side correlation metadata you're meant to map back through.

But `buildExpenseBatch` looks up `targetRows[i]` where `i` is the index into `newRows`:

```go
for i, entry := range newRows {
    targetRow, ok := targetRows[i]   // i = newRows index
```

These two indexings only coincide when `buildEmptyRowRequests` produces one request per newRow with no skips. The moment any row's subcategory isn't found in the workbook, this fires:

```go
subcatRow, ok := subcatRows[entry.Reviewed.Sheet][entry.Reviewed.Subcategory]
if !ok {
    continue   // emptyReqs now shorter than newRows → positions shift
}
```

Now `emptyReqs` is shorter than `newRows`, `AllocateEmptyRows` keys by the *compacted* position, and every subsequent `targetRows[i]` lookup returns the allocation belonging to a **different entry**. `ExpenseIndex` is set but never read.

**Concrete trace** (newRows = [A: subcat not in workbook, B: subcat found]):
- emptyReqs = `[reqForB]`; AllocateEmptyRows returns `{0: rowForB}`
- i=0 (A): `targetRows[0]` = rowForB exists → A gets written at **B's row**, with `subcatRow` = 0 (missing-key zero value)
- i=1 (B): `targetRows[1]` missing → B is **silently dropped**
- `writtenIndices=[0]` → feedback written for A as if inserted

So the unplaceable row gets inserted at the wrong location and the placeable row is silently dropped. On a real workbook this is silent corruption.

**Why your green checks didn't catch it:** the acceptance test never calls `insertNewRows` (all fixture rows are already-inserted/pending), and the happy path *is* correct (when every subcategory resolves, positions align). A Phase 4 smoke against clean data would likely **pass and give false confidence**. The discriminating input is a `reviewed.json` with one new row whose `reviewed.subcategory` doesn't exactly match a workbook reference subcategory.

**Verify, then fix:** confirm `AllocateEmptyRows`'s key by re-reading `results[idx] = row` in `internal/excel/reader.go`. The fix is to make the lookup key match what AllocateEmptyRows produces — iterate `emptyReqs` by position, use `req.ExpenseIndex` to fetch the original entry and its `parsedDates[ExpenseIndex]`, rather than ranging over `newRows` and indexing the result map with the newRows index.

**This blocks RUI-3a being meaningful and blocks merge of the insertion feature.** The feedback-only path (which the test does cover) is genuinely correct — that part ships fine.

---

## Second, lower severity: `--dry-run` still writes both logs

```go
if !dryRun && len(batch) > 0 {
    excel.WriteBatchExpenses(...)   // correctly skipped
}
return writeFeedbackForNewRows(...)  // runs unconditionally
```

`writeFeedbackForNewRows` calls `feedback.Append` and `feedback.AppendExpense` regardless of `dryRun`, and increments the "inserted" counts. The plan's intent for `--dry-run` was "print what would be inserted." Right now dry-run skips only the workbook write but still mutates `classifications.jsonl` and `expenses_log.jsonl`. Gate `writeFeedbackForNewRows` (and the counts) behind `!dryRun` too.

---

## Handoff notes: accurate, with one addition

The handoff is solid — the blind-spot framing (RUI-3a, `insertNewRows` untested) is exactly right, and the Ollama-context lesson is worth keeping. Two adjustments:

- **Fold the specific bug into RUI-3a** so next session fixes it *before* smoke, not after discovering corrupted output. As written, RUI-3a says "exercise the path" — it should say "fix the index-mapping bug in `buildExpenseBatch`/`AllocateEmptyRows` usage, then smoke."
- **PR #23 body** asserts "452 tests passing" next to the insertion table without flagging that the table's insertion rows are unverified *and* currently buggy. Worth a one-line caveat on the PR so a reviewer doesn't read the decision table as validated behavior.

---

## Net

- **Feedback-only path:** correct, tested, ships.
- **Insertion path:** one high-severity correctness bug (index aliasing) + one medium (dry-run side effects). Both are in the code committed this session and both are invisible to the checks you ran.
- **Don't run Phase 4 smoke as a pass/fail gate until the index bug is fixed** — clean test data will mask it.
