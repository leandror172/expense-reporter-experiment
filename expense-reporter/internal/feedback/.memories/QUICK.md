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

**Producers without a type (still type-less by design):** auto/batch-auto/add/correct
build entries from the classifier, which doesn't emit type yet (TODO(type) marks in
auto.go/batch_auto.go). Only the review→apply path emits a type. Closing this gap =
classifier full-path label, 5.R4/RUI-4. The generator's two-tier routing
([[taxonomy]] loader) handles type-less entries via a bare-name fallback.
