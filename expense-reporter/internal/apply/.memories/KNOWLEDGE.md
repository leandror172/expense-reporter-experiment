# apply — KNOWLEDGE

*Accumulated decisions. Read on demand.*

## Log-Append Pivot (WS-B slice 4, session 44, consolidated from QUICK.md 2026-07-01)
`apply` no longer writes the workbook; new rows append to `expenses_log.jsonl` via
`appender.ExpandAndAppend` (count=1 until T-21, s69 — now the entry's real installment
count). Deleted: `insertNewRows` + excel-allocation
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
- **T-21 — FIXED (session 69).** apply used to record every reviewed installment as ONE
  row because it passed a literal `1`: the count was discarded at `review.ReadQueue`
  (`perInstallment, _, err`) and absent from `reviewed.json`. `ReviewedEntry.Installments`
  now carries it end to end. Measured impact before the fix: 10 of 45 real rows.
  - **`installments` is REQUIRED, not defaulted.** An absent key decodes to 0, so a
    default of 1 would make "producer never sent it" indistinguishable from "producer
    sent zero" — the T-54 shape. `validateEntries` rejects `< 1` and says a
    non-installment row is `1`.
  - **The summary counts ROWS, not entries** (`rowsWrittenFor`). `printSummary` prints
    "Appended: N rows", which was true by accident while entries and rows were 1:1;
    T-21 broke that identity, so a `900,00/3` purchase printed "1 rows" and wrote 3.
  - **NEW limitation this created — partial series.** `ExpandAndAppend` appends
    row-by-row and returns on the first error, while the classification entry is written
    only AFTER success. A mid-series I/O failure therefore leaves k of N rows with no
    classification entry, and a re-run treats the row as new and re-appends the whole
    series, duplicating the k that landed. Impossible before T-21 (apply wrote one row,
    atomic by construction). T-20 designed batch-auto to route partial series to review
    and never auto-complete them; apply is now a second producer of partial series
    WITHOUT that guard, and it has no ledger at all (`LoadExpenseIDCounts` /
    `warnDuplicateEntries` are `batch_auto_resume.go`-only), so it cannot even warn.
