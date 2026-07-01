# apply — KNOWLEDGE

*Accumulated decisions. Read on demand.*

## Log-Append Pivot (WS-B slice 4, session 44, consolidated from QUICK.md 2026-07-01)
`apply` no longer writes the workbook; new rows append to `expenses_log.jsonl` via
`appender.ExpandAndAppend` (count=1). Deleted: `insertNewRows` + excel-allocation
pipeline, `--workbook`/`--backup` flags, the `uninsertable` concept, dead insert-era
predicates `IsInsertable`/`IsAlreadyHandled`.
**Why:** JSONL logs are the single source of truth; `generate-workbook` is the only
workbook writer (retire-insertion pivot).
Plan: `.claude/plans/ws-b-slice4-apply-log-append.md`.

## Write Order & Failure Honesty
`appendNewRows` per new row: nil-guard (`Reviewed==nil` → `failed`), parse date,
**expense log FIRST**, **feedback SECOND** (best-effort, warns on error). A log-append
failure downgrades the row → `failed` + non-zero exit.
**Why log-first:** the log is the durable record; feedback is audit/training data.
`dryRun` is threaded through and gates the found+corrected `feedback.Append` — the old
leak where dry-run still wrote feedback for found entries.

## Non-Destructive Pre-Flight
Non-dry-run only: classifications path probed BEFORE `processEntries` (it is read
there); expense-log path probed only when `newRows>0` and non-destructively
(`ensureLogWritable(path, allowCreate=false)` — NO `O_CREATE`, else an empty log file
breaks the found-only `ExpenseLogNotCreated` assertion). Both errors carry `Hint:`.

## Known Limitations
- **T-20 — idempotency is best-effort:** cross-file dedup via `classifications.jsonl`
  can duplicate on re-run if a feedback write fails after a successful log append.
  Pre-flighting both paths makes the common case unreachable.
- **T-21 — installments under-recorded:** apply records count=1; the installment count
  is discarded upstream (`review.ReadQueue`) and absent from `reviewed.json`, so a
  reviewed installment lands as a single row. Deferred fix.
