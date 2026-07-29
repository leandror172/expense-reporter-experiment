# internal/parse — Quick Memory

*Working memory for the T-41 parse boundary. Injected into agents. Keep under 30 lines.*

## Status
Slice 1 (s63): package founded from `parseExpenseForFeedback` (add.go, deleted).
Slice 2 (s66): `auto` migrated — consumers now `add`/`correct`/`auto`. Slices
pending: batch-auto → apply. Design authority: `.claude/plans/parse-boundary.md`
(FINAL, session 63).

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
- `YearSource` (s65) — WHICH rung resolved the year. Zero is
  `YearSourceUnknown`, never a real rung: every error path returns
  `ParsedExpense{}`, so a zero naming a rung would let a failed parse report a
  plausible source to code that reads it to decide. Produced by `resolveDate`,
  where the branch is made — re-deriving it in a caller would be a second copy
  of the ladder, free to drift.

## Key Rules
- **Year ladder (Options):** explicit year in string > `Year` (--year) >
  `ConfigYear` (config date_year) > most-recent-non-future. Grace window 0:
  a bare date strictly after Now resolves to LAST year; same-day is not future
  (candidate is midnight UTC). `Options.Now` zero value = time.Now(); tests
  inject the clock.
- **Two year validations on the RESOLVED year (s65), so one check covers every
  rung:** `validateRenderableYear` (permanent — outside 1..9999 `DateString()`
  emits malformed identity bytes) and `validateYearNotBeyondCurrent`
  (PROVISIONAL policy, T-47 — year-scale, so later-this-year is fine). Kept
  separate so the policy can be relaxed without touching the contract. Past side
  is deliberately unguarded (T-48). Installment expansion never re-enters here,
  so a 24× purchase still writes rows years ahead.
- **BR thousands:** token with BOTH '.' and ',' → dots stripped ("1.234,56" OK);
  dot-only tokens keep legacy decimal-dot meaning. Boundary owns ALL input
  normalization (design Q3).
- **Field sentinels (s66):** `ErrInvalidDate` / `ErrInvalidValue` name WHICH field
  `Fields` rejected, so a caller branches with `errors.Is` instead of matching
  English message text — required because the eventual consumers are an error log
  keyed by phase and a PT-BR chat layer (vision Phases 1 and 4), neither of which
  can read an English string. **Wrapped with TWO `%w` verbs** (sentinel + cause):
  the field must be identifiable AND the specific reason must survive. Dropping
  the cause is a live hazard, not a theoretical one — slice 2's first command-layer
  helper answered every date rejection with a format hint, telling a user who typed
  `15/04/2027` their format was wrong when the objection was the future year.
  No item sentinel: nothing branches on it.
- **Tests build values via the real parser**, never struct literals.
