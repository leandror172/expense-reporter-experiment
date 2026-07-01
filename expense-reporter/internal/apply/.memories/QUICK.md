# apply — QUICK

**What:** ingests the review UI's `reviewed.json` and **appends** new confirmed/corrected
entries to `expenses_log.jsonl` (via the single writer `appender.ExpandAndAppend`),
recording feedback in `classifications.jsonl`. **No workbook writes** — `generate-workbook`
is the sole workbook writer (WS-B slice 4). Driven by `cmd/.../apply.go`.

- Types (types.go): `ReviewedFile` → `ReviewedEntry{id,item,date,value,confidence,
  Predicted,action,Reviewed *ReviewedLocation}`; `ReviewedLocation{Type,Category,
  Subcategory}` (full path; legacy `"sheet"` key read-compat). `action` ∈
  confirmed/corrected/skipped/pending. `reader.go ReadReviewed` decodes + validates.
- **Flow:** each confirmed/corrected entry looked up in `classifications.jsonl`
  (`FindLatestEntry` = dedup index). Found → corrected-feedback-only, no append; not
  found → new row: expense log FIRST, feedback SECOND (best-effort); a log-append
  failure downgrades the row → `failed` + non-zero exit. `dryRun` gates ALL writes.
- **Pre-flight (non-dry-run):** both paths probed non-destructively before work
  (expense-log only when newRows>0, NO `O_CREATE`); errors carry `Hint:`.

**Open:** T-20 (best-effort idempotency), T-21 (reviewed installments recorded count=1).
Details + rationale → KNOWLEDGE.md.

**Tests:** acceptance `test/apply_test.go` + `type_routing_cycle_test.go` step 3;
unit `cmd/.../apply_test.go`. Fixtures: `test/fixtures/apply-basic/`.
