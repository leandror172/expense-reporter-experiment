# Implementation Plan — `apply` Command (RUI-3)

> **Status:** ready to execute. Await user "go" before each phase.
> **Prereqs:** `.claude/plans/review-command.md` (data contracts), `docs/plans/review-ui-fixtures/reviewed.sample.json` (output shape), `internal/review/template/review.html:1486` (export logic).
> **Workflow:** acceptance test first, then implementation, then unit tests. Pause between phases. Try local model (`my-go-qcoder`) before escalating.

## 1. Goal

Add `expense-reporter apply reviewed.json` — ingests the UI's `reviewed.json` back into
the workbook and feedback logs.

```
expense-reporter apply reviewed.json [--year 2026] [--workbook path]
```

## 2. Input contract (`reviewed.json`)

Exported by `exportReviewed()` in the review HTML (line 1486). Always contains ALL queue
rows — both `auto_inserted=1` and `auto_inserted=0` — each tagged with one of four actions:

| action | meaning |
|--------|---------|
| `confirmed` | user confirmed the prediction (or bulk-accepted) |
| `corrected` | user changed sheet/category/subcategory |
| `skipped` | user explicitly skipped |
| `pending` | user never touched the row |

`reviewed` field is non-null only for `confirmed`/`corrected`; null for `skipped`/`pending`.
`reviewed.sheet` carries the user's explicit sheet choice — must be honored, not re-resolved.

## 3. Decision table

| action | prior entry in classifications.jsonl? | workbook | feedback |
|--------|--------------------------------------|----------|----------|
| `pending` | — | skip | skip |
| `skipped` | — | skip | skip |
| `confirmed` | yes (auto-inserted) | skip | skip (already logged as confirmed) |
| `confirmed` | no (needs-review row) | insert | NewConfirmedEntry |
| `corrected` | yes (auto-inserted) | skip | NewCorrectedEntry + add to "workbook not updated" list |
| `corrected` | no (needs-review row) | insert with reviewed values | NewCorrectedEntry |

Idempotency guard: `feedback.FindLatestEntry(classifPath, id)` — if found, row is
already in workbook. Running `apply` twice is safe.

## 4. Year resolution

`ParseDate` in `pkg/utils/date.go` hardcodes 2025. Fix: add `ParseDateWithYear(dateStr string, year int)` alongside the existing function (do not break existing callers). `apply` uses `--year` flag (default `time.Now().Year()`).

## 5. Sheet honoring

`InsertBatchExpensesFromClassified` re-resolves sheet from subcategory via `PathIndex`.
This loses `reviewed.sheet` for ambiguous subcategories. Fix: add `SheetName string` to
`models.ClassifiedExpense`; modify `resolveAllSubcategories` to use it as an override
when non-empty (one-line guard before the PathIndex lookup). No other callers set
`SheetName` so existing behavior is unchanged.

## 6. Output summary

```
Applied reviewed.json (source: classified.csv, 349 rows)

Inserted:  18 rows (12 confirmed, 6 corrected)
Skipped:   3 rows
Pending:   319 rows (not reviewed)

⚠  2 already-inserted rows were corrected — workbook not updated:
   Unimed vencimento (02/05, R$731.97) Saúde/Consultas → Saúde/Plano de saúde  [logged]
   Delivery brod's   (01/05, R$53.98)  Alimentação/Delivery → confirmed         [logged]
```

No output for confirmed auto-inserted rows (common case, silent). Error rows printed inline.

## 7. Package & file layout

```
internal/apply/
  types.go     ← ReviewedFile, ReviewedEntry (JSON deserialization)
  reader.go    ← ReadReviewed(path) (ReviewedFile, error)
cmd/expense-reporter/cmd/
  apply.go     ← cobra command wiring
pkg/utils/
  date.go      ← add ParseDateWithYear (alongside existing ParseDate)
internal/models/
  expense.go   ← add SheetName string to ClassifiedExpense
internal/workflow/
  pipeline.go  ← modify resolveAllSubcategories to honor ClassifiedExpense.SheetName
```

No new internal package is strictly needed — `apply.go` could inline the logic — but
`internal/apply/` keeps the cobra command thin and makes the reader unit-testable.

## 8. Model sentinel for `confirmed` new rows

For needs-review rows confirmed in the UI, there is no prior `classifications.jsonl` entry
to pull the model name from. Use `"review"` as the model name sentinel in the feedback
entry. This is consistent with the intent: the human reviewer, not a model, confirmed it.

For `corrected` new rows: same — `"review"` as model name, predicted values from the
`reviewed.json` `predicted` field.

For `corrected` already-inserted rows: call `FindLatestEntry(id)` to get the original
model name and predicted values for `NewCorrectedEntry`.

## 9. Phasing (pause for approval between each)

**Phase 0 — acceptance test (red).** Fixture: a small `reviewed.json` covering all
four action types (confirmed-new, corrected-new, confirmed-already-inserted, pending).
Test workbook: reuse existing synthetic fixture from `review` acceptance test.
Assert: correct rows inserted, corrected feedback entries written, no double-insertion.

**Phase 1 — plumbing changes.**
- `ParseDateWithYear` in `pkg/utils/date.go`
- `SheetName` field on `models.ClassifiedExpense`
- `resolveAllSubcategories` override in `internal/workflow/`
- Unit tests for all three

**Phase 2 — `internal/apply/` package.**
- `types.go`: `ReviewedFile`, `ReviewedEntry`
- `reader.go`: `ReadReviewed`
- Unit tests: good parse, unknown action, null reviewed field, invalid JSON

**Phase 3 — `cmd/apply.go`.**
- Cobra wiring: `apply <reviewed.json>`, `--year`, `--workbook`, `--dry-run`
- Core loop: read → for each entry → guard → insert/feedback/collect
- Summary output

**Phase 4 — build/vet/test/smoke.**
- `go build/vet/test ./...`
- Acceptance suite green
- Smoke: run against real `reviewed.json` from last review session if available

## 10. Out of scope
- Modifying `review.html` — no UI changes needed for this plan
- Workbook correction for already-inserted corrected rows — feedback-only by design (D2-A)
- `--dry-run` output format beyond basic echo (v1: print what would be inserted)
