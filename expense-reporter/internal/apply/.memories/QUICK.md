# apply — QUICK

**What:** ingests the review UI's `reviewed.json` and **appends** new confirmed/corrected
entries to `expenses_log.jsonl` (via the single writer `appender.ExpandAndAppend`), recording
feedback in `classifications.jsonl`. **No workbook writes** — `generate-workbook` is the sole
workbook writer (WS-B slice 4, log-append pivot, session 44). Driven by `cmd/.../apply.go`.

- Types (types.go): `ReviewedFile{reviewedAt,source,entries}` → `ReviewedEntry`
  {id,item,date,value,confidence,`Predicted`,`action`,`Reviewed *ReviewedLocation`}.
  `ReviewedLocation` = `{Type,Category,Subcategory}` (full path). Custom `UnmarshalJSON` keeps
  backward compat: legacy `"sheet"` key → `Type`. `reader.go` `ReadReviewed` decodes + validates.
  `action` ∈ confirmed / corrected / skipped / pending. (Dead insert-era predicates
  `IsInsertable`/`IsAlreadyHandled` were removed in slice 4.)

**Flow (cmd/.../apply.go):**
- `processEntries`→`handleActiveEntry`: each confirmed/corrected entry is looked up in
  `classifications.jsonl` (`FindLatestEntry`, the dedup index). **Found** → corrected-feedback-only,
  no append; **not found** → new row. `dryRun` is threaded through and gates the found+corrected
  `feedback.Append` (the old leak).
- `appendNewRows`: per new row — nil-guard (`Reviewed==nil` → `failed`), parse date, **expense log
  FIRST** (`appender.ExpandAndAppend`, count=1), **feedback SECOND** (best-effort, warns on error). A
  log-append failure downgrades the row → `failed` + non-zero exit (failure honesty).
- **Pre-flight (non-dry-run):** classifications path probed BEFORE `processEntries` (read there);
  expense-log path probed only when `newRows>0` and **non-destructively** (`ensureLogWritable(path,
  allowCreate=false)` — NO `O_CREATE`, else an empty log breaks found-only `ExpenseLogNotCreated`).
  Both errors carry `Hint:`.

**Key facts:**
- **Idempotency is best-effort**, not guaranteed: cross-file dedup via `classifications.jsonl` can dup
  on re-run if a feedback write fails after a successful log append (T-20). Pre-flighting both paths
  makes the common case unreachable.
- **Installments (T-21):** apply records count=1 — the installment count is discarded upstream
  (`review.ReadQueue`) and absent from `reviewed.json`, so a reviewed installment is under-recorded
  (a single per-installment row). Deferred fix.

**Tests:** acceptance `test/apply_test.go` (idempotency, dry-run-writes-nothing, both unwritable-path
pre-flights) + `type_routing_cycle_test.go` step 3 (typed append, no workbook). Unit
`cmd/.../apply_test.go` (downgrade-on-append-failure, malformed→failed).
**Fixtures:** `test/fixtures/apply-basic/` (reviewed.json, seed-classifications.jsonl, expected-feedback.jsonl).
