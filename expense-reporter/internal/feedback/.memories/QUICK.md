# feedback — QUICK

**What:** JSONL logging of classification + append events. Two files, two structs.

| File on disk | Struct | Written by |
|---|---|---|
| `classifications.jsonl` | `Entry` (feedback.go) | add / auto / batch-auto / apply / correct |
| `expenses_log.jsonl` | `ExpenseEntry` (expense_log.go) | auto / batch-auto / apply — all via `appender.ExpandAndAppend` (WS-B) |

- Both share `GenerateID(item,date,value)` (sha256 prefix, 12 hex) — the **join key**
  across all three files (incl. `reviewed.json`).
- Constructors: `NewConfirmedEntry`/`NewManualEntry`/`NewCorrectedEntry`,
  `NewExpenseEntry`. `Append`/`AppendExpense` write one JSON line.
- Status values: `confirmed` (auto accepted), `manual` (add), `corrected` (user overrode).
- **`Type` field:** both structs carry `type,omitempty`; all expenses_log producers set it
  → the log is fully typed going forward. `add`/`correct` remain type-less but write only
  to classifications.jsonl. Income is structurally type-less. Rationale + history →
  KNOWLEDGE.md.
- Multi-year history is stored per-year (`expenses_log-{year}.jsonl`); since WS-A the
  loader also accepts `DD/MM/YYYY`, so one merged multi-year log works — promotion to
  canonical pending (user's call). See [[project_workbook_extraction_5r4]].
