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

- **Identity is DERIVED, never trusted (T-41 slice 4, s67).** `entriesWithCanonicalDates`
  rewrites each entry's Date to canonical form AND recomputes its `ID` from it. The id
  in `reviewed.json` is advisory: it was hashed from a date apply is about to change, so
  trusting it meant looking up an id apply never writes — and that miss is SILENT
  (unfound → treated as new → appended), so the symptom was a duplicate expense, not an
  error. `canonicalDate` now delegates to `internal/parse` (`parse.Date`), inheriting the
  year ladder and BOTH year validations — including T-47's future-year refusal, which
  here downgrades the row to `failed` rather than aborting the run.
- **Fixture note:** `apply-basic` is self-consistent and therefore CANNOT catch a
  regression of the above — mutation-verified. `apply-stale-id` is the only
  discriminating fixture; its id is deliberately hashed from the pre-canonical date.

- **Installments ride in `reviewed.json` (T-21, s69).** `ReviewedEntry.Installments` is
  REQUIRED — `validateEntries` rejects `< 1`, because an absent key decodes to 0 and a
  default of 1 would hide a producer that stopped emitting it (the T-54 shape). apply
  passes it to `ExpandAndAppend`, so a reviewed `99,90/3` lands as 3 dated rows; it used
  to pass a literal `1` and land as one. The summary counts ROWS via `rowsWrittenFor`,
  not entries — it prints "Appended: N rows" and that must stay literally true.

**Open:** T-20 (best-effort idempotency); **partial installment series** — a mid-append
failure leaves k of N rows with no classification entry, so a re-run re-appends the whole
series (new since T-21; apply has NO ledger and cannot even warn). Details → KNOWLEDGE.md.
Details + rationale → KNOWLEDGE.md.

**Tests:** acceptance `test/apply_test.go` + `type_routing_cycle_test.go` step 3;
unit `cmd/.../apply_test.go`. Fixtures: `test/fixtures/apply-basic/`.
