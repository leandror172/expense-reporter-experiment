# feedback — QUICK

**What:** JSONL logging of classification + insertion events. Two separate files,
two separate structs.

| File on disk | Struct | Written by |
|---|---|---|
| `classifications.jsonl` | `Entry` (feedback.go:27) | add / auto / batch-auto / apply / correct |
| `expenses_log.jsonl` | `ExpenseEntry` (expense_log.go:11) | auto / batch-auto / apply (slim insert log) |

- Both share `GenerateID(item,date,value)` (sha256 prefix, 12 hex) — the **join key**
  across all three files (incl. `reviewed.json`).
- Constructors: `NewConfirmedEntry` / `NewManualEntry` / `NewCorrectedEntry` (feedback.go),
  `NewExpenseEntry` (expense_log.go). `Append` / `AppendExpense` write one JSON line.
- Status values: `confirmed` (auto accepted), `manual` (add, no model), `corrected`
  (user overrode model).

**Type field (Plan A / T-05, IMPLEMENTED):** both `Entry` and `ExpenseEntry` now carry
`Type string json:"type,omitempty"` (Fixas/Variáveis/Extras/Adicionais). Set
**post-construction on the apply path only** (constructors unchanged) — apply.go assigns
`.Type = entry.Reviewed.Type` after building each entry. `omitempty` keeps type-less
entries byte-identical on disk (so non-apply fixtures didn't churn).

**Type emission — 5.R4 DONE (was "apply path only"; now updated).** `auto`
(auto.go:200 `entry.Type=typ`), `batch-auto` (batch_auto.go:215 `resolveExpenseType`),
and `apply` all populate `ExpenseEntry.Type` → **`expenses_log.jsonl` is fully typed
going forward.** Remaining type-less producers write ONLY to `classifications.jsonl`
(training/audit log, NOT the generator input): `add` (NewManualEntry, no type) and
`correct`. **Income entries are type-less by nature** (no expense type) and still route
via the [[taxonomy]] bare-name fallback. So the generator's bare-name tier is retirable
for EXPENSES; income needs a dedicated `incomePath` route first (T-09).

**expenses_log is year-implicit** (`DD/MM`; `taxonomy.parseDate` requires exactly 2
parts). Multi-year history (2022–2025 historical extraction, 5.R4 —
[[project_workbook_extraction_5r4]]) is therefore stored as per-year files
`expenses_log-{year}.jsonl` (2025 merged into the base file), fed to generate via
`--year`. **FUTURE: "year adaptation"** — let `parseDate` accept `DD/MM/YYYY` + generate
use a per-entry year, so one multi-year log suffices and the per-year split retires.
