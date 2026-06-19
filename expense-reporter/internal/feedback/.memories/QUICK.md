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

**Gotcha / known gap (session 32):** neither struct carries the expense **type**
(Fixas/Variáveis/Extras/Adicionais). The review UI + apply path *know* the type and
insert into the right worksheet, but it is **dropped at log-write time** → retraining
data loses it. Fix = Plan A (`.claude/plans/persist-expense-type.md`): add
`Type string json:"type,omitempty"`, set post-construction on the apply path only.

**Producers without a type:** auto/batch-auto/add/correct build entries from the
classifier, which doesn't emit type. Only the review→apply path has it.
