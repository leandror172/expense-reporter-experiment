# apply — QUICK

**What:** ingests the review UI's `reviewed.json`, inserts new rows into the workbook,
and writes feedback + expense logs. Driven by `cmd/.../apply.go`.

- Types (types.go): `ReviewedFile{reviewedAt,source,entries}` → `ReviewedEntry`
  {id,item,date,value,confidence,`Predicted`,`action`,`Reviewed *ReviewedLocation`}.
- `ReviewedLocation` (types.go) = `{Type,Category,Subcategory}` — the full path the user
  chose. `Type` is the expense type (renamed from `Sheet`, Plan A F3). Custom
  `UnmarshalJSON` keeps **backward compat**: legacy `"sheet"` key falls back to `Type`
  when `Type` is empty, so already-saved `reviewed.json` exports still load.
- `reader.go` `ReadReviewed` decodes + validates.
- `action` ∈ confirmed / corrected / skipped (`apply.Action*` consts).

**Key fact (Plan A / T-05 IMPLEMENTED):** apply.go uses `entry.Reviewed.Type` for excel
insertion AND now persists it to the logs — `.Type = entry.Reviewed.Type` is set
post-construction after `NewExpenseEntry`/`NewConfirmedEntry`/`NewCorrectedEntry`. The
log-write drop is fixed. Type-less producers (auto/batch-auto/add/correct) are unchanged.

**Fixtures:** `test/fixtures/apply-basic/` has `reviewed.json`, `seed-classifications.jsonl`,
`expected-feedback.jsonl` (no `expected-expenses_log.jsonl`).
