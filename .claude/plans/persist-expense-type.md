# Plan A — Persist the Expense *Type* (end-to-end) + rename + JSON migration

**Goal:** stop discarding the user-chosen expense type (Fixas / Variáveis / Extras /
Adicionais) when corrections are written to the training/feedback logs, rename the
domain concept from "sheet" to "type", migrate the JSON keys accordingly (with a
backward-compatible reader for already-saved `reviewed.json`), and backfill the type
into existing log lines from saved corrections.

**Why:** the type is *modeled* in `review` + `apply` and *used* for workbook
insertion, but it is **never written** to `classifications.jsonl`
(`feedback.Entry`) or `expenses_log.jsonl` (`feedback.ExpenseEntry`). Those are the
files that feed retraining (tasks 5.R4 / RUI-4). Every ambiguous-leaf correction the
user has already made was saved into the workbook but dropped from the logs.

**Prerequisite:** merge `refactor/internal-taxonomy` first (clean baseline).

**Hard prerequisite for Plan B (full-path routing):** Plan A Phase F must land first —
Plan B routes on a `type` field that Phase F adds to `ExpenseEntry`.

---

## Terminology decision (read before editing)

There are TWO legitimate meanings of the word "sheet" in this codebase — only one
gets renamed:

| Meaning | Stays or renames | Why |
|---|---|---|
| **Expense *type*** — the bucket a subcategory belongs to (Fixas/…): taxonomy domain type, the classification field the user chooses, the value persisted to logs | **RENAME → "type"** | It is a domain concept, not a presentation artifact ([[feedback_style_vocabulary]] — name by WHAT it is) |
| **Excel worksheet** — a literal tab in the `.xlsx`: `models.SheetLocation.SheetName`, `internal/inspect` `SheetInfo`/`"sheet"`, excelize calls | **KEEP "sheet"** | It genuinely addresses a spreadsheet tab |

The type's *value* ("Fixas") happens to equal the worksheet's name — that's fine. We
rename the **concept**, not the worksheet addressing.

---

## Phase F — Functional: add & persist the `Type` field (ESSENTIAL, do first)

Use `omitempty` everywhere so only entries that actually carry a type change on disk
(limits acceptance-fixture churn to the apply path).

### F1. `internal/feedback/feedback.go` — `Entry` struct (line 27)
Add after `ActualCategory` (line 36):
```go
	Type                 string  `json:"type,omitempty"` // expense type: Fixas/Variáveis/Extras/Adicionais
```
Do **not** change the four constructor signatures. Callers that have a type set it
post-construction (F4); all others leave it "" (omitted by `omitempty`).

### F2. `internal/feedback/expense_log.go` — `ExpenseEntry` struct (line 11)
Add after `Category` (line 17):
```go
	Type        string  `json:"type,omitempty"`
```
Leave `NewExpenseEntry` signature unchanged (set post-construction in F4).

### F3. `internal/apply/types.go` — `ReviewedLocation` (line 31)
Rename the Go field + JSON key, keeping read-compat for old exports:
- Line 32 `Sheet string \`json:"sheet,omitempty"\`` → `Type string \`json:"type,omitempty"\``
- Add a custom unmarshal so existing saved `reviewed.json` files (key `"sheet"`)
  still load. In `internal/apply/types.go`:
```go
func (l *ReviewedLocation) UnmarshalJSON(b []byte) error {
	type alias ReviewedLocation // avoid recursion
	var withType struct {
		alias
		LegacySheet string `json:"sheet"`
	}
	if err := json.Unmarshal(b, &withType); err != nil {
		return err
	}
	*l = ReviewedLocation(withType.alias)
	if l.Type == "" {
		l.Type = withType.LegacySheet // fall back to pre-migration key
	}
	return nil
}
```
(add `encoding/json` import if missing).

### F4. Update apply-path call sites to thread the type
All in `cmd/expense-reporter/cmd/apply.go`. After F3, `entry.Reviewed.Type` holds the
value (formerly `entry.Reviewed.Sheet`). Replace every `entry.Reviewed.Sheet`
reference (lines 194, 212, 222, 248, 257, 336) with `entry.Reviewed.Type` — these
feed `SheetName` for excel insertion; the **value** is identical, only the field
renamed.

Then set the persisted type after each entry construction:
- Line 282 (`expEntry`):
  ```go
  expEntry := feedback.NewExpenseEntry(entry.Item, entry.Date, entry.Value, entry.Reviewed.Subcategory, entry.Reviewed.Category)
  expEntry.Type = entry.Reviewed.Type
  ```
- Line 136 (`corrEntry := feedback.NewCorrectedEntry(...)`): after the call,
  `corrEntry.Type = entry.Reviewed.Type`.
- Lines 298 & 305 (the confirmed/corrected helper returns): set `.Type =
  entry.Reviewed.Type` on the built entry before returning (these helpers receive
  `entry`, which has `Reviewed`; if `Reviewed` may be nil for confirmed, guard with
  `if entry.Reviewed != nil`).

### F5. Other callers — leave type empty (documented)
`auto.go:183/196`, `add.go:185/216/218`, `batch_auto.go:292`, `correct.go:69` build
entries from the **classifier**, which has no type today (see Plan B / classifier
work). No change needed — `omitempty` omits the key. Add a one-line `// TODO(type):
classifier does not yet emit expense type` comment at `auto.go:196` and
`batch_auto.go:292` so the gap is discoverable.

> **Until the classifier emits type, the review UI is the ONLY type producer.**
> `auto`, `batch-auto`, `add`, and `correct` entries stay type-less by design. Do not
> expect those log lines to carry a type. Plan B's routing accounts for this with a
> bare-name fallback for type-less entries — read Plan B's "CRITICAL" note before
> assuming type-less entries can be dropped.

### F6. Tests & fixtures
- `internal/feedback/*_test.go`: add a case asserting `Type` round-trips through
  `Append`/`AppendExpense` + JSON marshal (testify).
- `internal/apply/reader_test.go`: add a case proving a `reviewed.json` with the
  **legacy `"sheet"` key** loads into `ReviewedLocation.Type` (guards F3 fallback),
  and one with the new `"type"` key.
- Acceptance fixture `test/fixtures/apply-basic/expected-feedback.jsonl`: add
  `"type":"<value>"` to the corrected/confirmed lines that now carry it. Run the
  acceptance suite and update **only** the apply-path expected files that change;
  auto/add/batch fixtures must remain byte-identical (proves `omitempty` scoping).

### F7. Verify Phase F
```
cd expense-reporter && gofmt -l . && go vet ./... && go build ./... && go test ./...
./run-acceptance.sh
```
Expect: only apply-path fixtures changed; all green.

---

## Phase R — Rename `ExpenseSheet` → `ExpenseType` + migrate taxonomy/review JSON keys (RECOMMENDED, do after F is green)

Coordinated cosmetic rename of the **domain** concept. Build after every step.

### R1. `internal/taxonomy/types.go`
- Line 41: `type ExpenseSheet struct` → `type ExpenseType struct`; update the doc
  comment (line 40).

### R2. `internal/taxonomy/loader.go`
- Replace all `ExpenseSheet` → `ExpenseType` (return types, `rawSheetsToExpenseSheets`
  → `rawTypesToExpenseTypes`, locals). Keep `expensePath` semantics (still
  `expense\x00sheet\x00cat\x00sub`; the segment is the type name — value unchanged).
- `rawSheet` struct + tag (lines 39, 55): rename to `rawType` and migrate the JSON
  key `json:"sheets"` → `json:"types"`.

### R3. `internal/generate/` (`expense_sheet.go`, `generate.go`, `layout.go`,
`summary_sheet.go`)
- Replace `ExpenseSheet` → `ExpenseType`. **`sheetOrder` stays named `sheetOrder`** —
  it orders the *worksheets* and excelize sheet creation (presentation). Verify the
  generated workbook's sheet names are unchanged.

### R4. `internal/review/` (`types.go`, `taxonomy.go`)
- `types.go`: `Predicted.Sheet` `json:"sheet,omitempty"` → `Type` `json:"type,omitempty"`;
  `Taxonomy.Sheets []Sheet \`json:"sheets"\`` → `Types []ExpenseType \`json:"types"\``;
  rename the `Sheet` struct → `ExpenseType` (its `Name`/`Categories` unchanged).
  `taxonomy.go`: `sheetOrder`/`SheetName`/`buildSheet` — the *mappings* carry a sheet
  name from the workbook; rename the review-facing identifiers to `type` but keep the
  underlying mapping field that reads the workbook's sheet name.

### R5. JSON data files (migrate `"sheets"` → `"types"`)
- `config/taxonomy.json` (gitignored): rename top-level `"sheets"` → `"types"`.
- `test/fixtures/generate-basic/taxonomy.json`: same.
- `internal/review` sample (`review-data.sample.json` if present) + `docs/plans/review-ui-fixtures/*`: same.
- These are generated/fixture inputs (no saved user artifacts) → **no fallback
  reader needed**; a hard rename is fine.

### R6. `internal/review/template/review.html` (the template, NOT the rendered file)
- Input parse: `DATA...predicted.sheet` → `.type`; `TAX.sheets` → `TAX.types`;
  `sheet.name`/`sheetByName`/`SHEETS_FOR` → type equivalents.
- Export (`exportReviewed`, ~line 1486): change emitted key
  `reviewed = { sheet: ... }` → `reviewed = { type: ... }`.
- Keyboard hint text "set sheet" → "set type"; "Predicted sheet" option value.
- **Do not** open the large rendered `review*.html` files at repo root; only edit the
  template. (If you must inspect a rendered file, delegate to a Haiku subagent and
  ask it targeted questions — do not read it into context.)

### R7. Verify Phase R
- `go build ./... && go test ./...` green.
- Acceptance suite green; the `generate-basic` oracle dumps must be **byte-identical**
  (rename is value-preserving — sheet/type names unchanged in the workbook).
- Render `review` for a small fixture and confirm the page still loads and the
  exported `reviewed.json` uses the `"type"` key.

---

## Phase B-fill — Backfill existing logs from saved corrections (RECOVERY)

The user has saved corrections (browser `localStorage`, key
`expense-review:v1:rows:<source>:<generatedAt>`) and/or already-exported
`reviewed.json` files. The export already contains the type (formerly `"sheet"`).
Recover them **without re-running `apply`** (which would double-insert workbook rows).

### Bf1. Re-export
Reopen the saved-state `review.html` in the same browser and click **Export reviewed
file** → `reviewed.json`. (Export already carries the type; no page change required
for recovery — the F3 fallback reader handles the legacy `"sheet"` key.)

### Bf2. Backfill tool — `.claude/tools/backfill-type.py`
Pure stdlib Python. Inputs: one or more `reviewed.json`, plus
`expenses_log.jsonl` and `classifications.jsonl`. For each log line whose `id`
matches a reviewed entry and whose `type` is empty/absent, set
`type = reviewed.reviewed.type` (or legacy `.sheet`). Rewrite the logs in place
(back up first: `*.bak`). Match key = `id` (the shared `GenerateID` hash; present in
all three files). Report: matched / filled / unmatched counts.
- Add it to `.claude/index.md` Tools table.

> **Backfill is partial by design.** It only fills entries that were touched in
> review (those have a `reviewed.json` line). Auto-inserted / batch-auto / `add`
> entries were never reviewed → no match → stay type-less. That is expected: those
> type-less entries are covered by Plan B's bare-name fallback at routing time, not by
> backfill. Do not try to "fix" the unmatched count to zero.

### Bf3. Verify
- `wc -l` unchanged on both logs (no lines added/lost).
- Spot-check that ambiguous-leaf lines (`Orion`, `Gás`, `Dentista`) now carry a
  `type`.
- These backfilled lines become the gold few-shot labels for Plan B + classifier work.

---

## Done-criteria for Plan A
- `feedback.Entry` and `ExpenseEntry` carry `type` (omitempty), written on the apply
  path.
- `reviewed.json` round-trips the type under key `"type"`, **and** legacy `"sheet"`
  exports still load.
- Domain renamed `ExpenseType`; taxonomy/review JSON keys migrated to `"types"`/`"type"`;
  worksheet-addressing "sheet" untouched.
- Existing corrections backfilled into the logs.
- All unit + acceptance tests green; `generate-basic` oracle dumps byte-identical.

## Files touched (checklist)
- `internal/feedback/feedback.go`, `expense_log.go` (+ tests)
- `internal/apply/types.go` (+ `reader_test.go`)
- `cmd/expense-reporter/cmd/apply.go`
- `cmd/expense-reporter/cmd/{auto,batch_auto}.go` (TODO comments only)
- `internal/taxonomy/{types,loader}.go`
- `internal/generate/{expense_sheet,generate,layout,summary_sheet}.go`
- `internal/review/{types,taxonomy}.go`, `internal/review/template/review.html`
- `config/taxonomy.json`, `test/fixtures/generate-basic/taxonomy.json`, review fixtures
- `test/fixtures/apply-basic/expected-feedback.jsonl`
- `.claude/tools/backfill-type.py` (new), `.claude/index.md`
