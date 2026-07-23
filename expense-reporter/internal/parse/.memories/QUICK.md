# internal/parse — Quick Memory

*Working memory for the T-41 parse boundary. Injected into agents. Keep under 30 lines.*

## Status
Slice 1 (s63): package founded from `parseExpenseForFeedback` (add.go, deleted);
consumers = `add`/`correct`. Slices pending: auto → batch-auto → apply.
Design authority: `.claude/plans/parse-boundary.md` (FINAL, session 63).

## Contract
- `ParsedExpense` — SINGLE stored `Date time.Time`; `DateString()` derives the
  canonical `DD/MM/YYYY` and is the SOLE identity-bytes formatting site
  (`GenerateID` + both JSONL logs consume its output). An inconsistent date pair
  is unrepresentable by construction — do not add a stored string date field.
- `Fields(item, dateStr, valueStr, Options)` — field-wise core; all commands use it.
- `ExpenseString(s, Options)` — 4-field semicolon wrapper; returns subcategory
  ALONGSIDE (classification is not parsing — never add it to the struct).
- `Value` is PER-INSTALLMENT (total ÷ N for "total/N"); `RawValue` keeps the
  original token for display/audit.

## Key Rules
- **Year ladder (Options):** explicit year in string > `Year` (--year) >
  `ConfigYear` (config date_year) > most-recent-non-future. Grace window 0:
  a bare date strictly after Now resolves to LAST year; same-day is not future
  (candidate is midnight UTC). `Options.Now` zero value = time.Now(); tests
  inject the clock.
- **BR thousands:** token with BOTH '.' and ',' → dots stripped ("1.234,56" OK);
  dot-only tokens keep legacy decimal-dot meaning. Boundary owns ALL input
  normalization (design Q3).
- **Tests build values via the real parser**, never struct literals.
