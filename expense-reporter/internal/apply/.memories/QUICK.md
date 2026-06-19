# apply — QUICK

**What:** ingests the review UI's `reviewed.json`, inserts new rows into the workbook,
and writes feedback + expense logs. Driven by `cmd/.../apply.go`.

- Types (types.go): `ReviewedFile{reviewedAt,source,entries}` → `ReviewedEntry`
  {id,item,date,value,confidence,`Predicted`,`action`,`Reviewed *ReviewedLocation`}.
- `ReviewedLocation` (types.go:31) = `{Sheet,Category,Subcategory}` — the full path the
  user chose. `Sheet` is the expense **type** (to be renamed `Type` + JSON key migrated
  `sheet`→`type` with legacy fallback, Plan A F3).
- `reader.go` `ReadReviewed` decodes + validates.
- `action` ∈ confirmed / corrected / skipped (`apply.Action*` consts).

**Key fact (session 32):** apply.go **already uses** `entry.Reviewed.Sheet` for excel
insertion (`SheetName` at apply.go:194/212/222/248/257) — full path is live on the
workbook path. The ONLY drop is at log-write: `NewExpenseEntry`/`NewCorrectedEntry`
(apply.go:282/136/298/305) don't pass the type. Plan A fixes exactly those sites.

**Fixtures:** `test/fixtures/apply-basic/` has `reviewed.json`, `seed-classifications.jsonl`,
`expected-feedback.jsonl` (no `expected-expenses_log.jsonl`).
