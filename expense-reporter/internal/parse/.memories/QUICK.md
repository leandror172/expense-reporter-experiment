# internal/parse — Quick Memory

*Working memory for the T-41 parse boundary. Injected into agents. Keep under 30 lines.*

## Status
Slice 1 (s63): package founded from `parseExpenseForFeedback` (add.go, deleted).
Slice 2 (s66): `auto`. Slice 3 (s67): `batch-auto`. Slice 4 (s67): `apply` —
**all six parsers now route here**; `cmd.canonicalDate` (the T-35 prototype) is
retired. Design authority: `.claude/plans/parse-boundary.md` (FINAL, session 63;
slice-3 decisions in §11).

## Contract
- `ParsedExpense` — SINGLE stored `Date time.Time`; `DateString()` derives the
  canonical `DD/MM/YYYY` and is the SOLE identity-bytes formatting site
  (`GenerateID` + both JSONL logs consume its output). An inconsistent date pair
  is unrepresentable by construction — do not add a stored string date field.
- `Fields(item, dateStr, valueStr, Options)` — field-wise core; all commands use it.
- `Date(dateStr, Options)` (s67) — date-only entry point, for a caller whose other
  fields are already structured. `apply` reads `reviewed.json`, where the JSON
  decoder already typed item and value and only the date is text; without this it
  would have to re-serialize a float purely to have the boundary parse it back.
  `Fields` DELEGATES to it — one ladder, one pair of year validations, whichever
  door a caller comes through.
- `Value(valueStr)` (s73) — value-only entry point, the mirror of `Date`, for a
  caller whose other fields are already structured. `review.ReadQueue` is that
  caller: its date column has been canonical since slice 3 and its flags are
  typed, so only the value token is still text. **`Fields` DELEGATES to it**, so
  normalization has exactly ONE call site — verify with
  `grep -n "normalizeThousands(" internal/parse/parse.go` (one call + the
  definition, nothing more). Before this, `ReadQueue` called utils directly and
  therefore never stripped BR thousands, so `writeClassifiedCSV` could emit a
  `1.234,56` the reader HARD-ERRORED on, killing the whole review step over one
  row — with both sides' own tests green (T-72). Wraps with the double `%w`, so
  a consumer gets `errors.Is(err, ErrInvalidValue)`; `ReadQueue` therefore
  dropped its own "invalid value: " literal and the message stayed byte-identical.
- `ExpenseString(s, Options)` — 4-field semicolon wrapper; returns subcategory
  ALONGSIDE (classification is not parsing — never add it to the struct).
- **`ParsedExpense.Value` the FIELD** (not to be confused with the `Value`
  function above) is PER-INSTALLMENT under BOTH installment notations;
  `RawValue` keeps the original token for display/audit.
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
  separate so the policy can be relaxed without touching the contract. Installment
  expansion never re-enters here, so a 24× purchase still writes rows years ahead.
- ⚠️ **Past side unguarded (T-48) is NOT just a policy footnote — it is a LIVE
  DEFECT, measured twice on real data (s75/s76).** `Date("29/12/25")` returns
  **29/12/0025 with NO error**: `ParseDateFlexible` takes any integer year and
  only `validateYearNotBeyondCurrent` looks, forward. The row then hashes cleanly
  into both logs and is never routed by `generate-workbook --year`, so the failure
  is invisible — no error, no lost row. **It BLOCKS T-63's documented recovery
  workflow:** the rejects file is re-imported through `batch-auto` (the raw
  boundary, no expansion), T-63 tells the human to fix the file, and the natural
  fix for a wrong year is to correct the digits in the `DD/MM/YY` notation they
  already use. Hit for real in s76. Likewise `Date("21/08/1200")` → year 1200.
  `internal/capture` guards its own door (D1 amendment: the resolved date must sit
  in `[-180,+7]` of the message, explicit year or not) — **that fix is the
  CONVERTER's and does not protect `add`, `batch` or the rejects re-import.**
  Filed for a boundary decision of its own: reject a 2-digit year, or expand it.
- **BR thousands:** token with BOTH '.' and ',' → dots stripped ("1.234,56" OK);
  dot-only tokens keep legacy decimal-dot meaning (a hazard the multiplier made
  reachable in a new position — filed as T-74, NOT introduced by it). Boundary
  owns ALL input normalization (design Q3). **Q3 is what decided T-72's fix:**
  the alternative was pushing `normalizeThousands` down into `pkg/utils`, which
  would have put input normalization outside the boundary.
- **TWO installment notations, and they are INVERSES (T-64, s73).** `405,25/4`
  states the TOTAL and DIVIDES; `405,25 x4` states the PER-INSTALLMENT amount and
  MULTIPLIES. Same digits, 4× apart in the budget. They live together in
  `utils.ParseCurrencyWithInstallments` and are pinned in ONE test table on
  purpose — a reader who conflates them must edit adjacent contradictory lines.
  The mode switch is the PRESENCE of `x`/`X`, never position. The value/count
  split is then resolved by trying three readings (`value x count`,
  `value count x`, `count x value`) and requiring **EXACTLY ONE** to hold:
  two readings means AMBIGUOUS and is an ERROR, never a guess. So `4x5` and
  `646,254x` are rejected — and rejecting is the cheap side, because T-63 returns
  the row in `failed.csv` with its reason inline to be repaired in place, while
  guessing writes a budget row wrong by the installment count and
  indistinguishable from a legitimate one. Reading helpers report SHAPE match
  separately from count plausibility, which is what lets `405,25 x0` say "must be
  positive" instead of degrading to a generic format error.
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
- **Tests build values via the real parser**, never struct literals. This is not
  style: `YearSource` is recorded by `resolveDate`, so a hand-set field tests the
  literal rather than the ladder, and a literal can express a combination the
  boundary rejects — the test would then describe behavior production cannot
  produce. A generated slice-3 test was rejected for the adjacent sin of asserting
  the parser against a second call to itself.
