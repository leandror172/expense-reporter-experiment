# expense-reporter/ — Quick Memory

*Working memory for the Go application. Injected into agents. Keep under 30 lines.*

## Status
Full CLI: add, batch, classify, auto, batch-auto, correct, review, apply,
generate-workbook. 263 unit test funcs (647 w/ subtests); JSON output mode; few-shot + MCP feedback done.
**JSONL logs are the source of truth; `generate-workbook` is the ONLY workbook writer.**
WS-B complete: batch-auto and apply append to `expenses_log.jsonl` via
`appender.ExpandAndAppend`. Classifier predicts the full path (T-13); default model
`my-classifier-q3`. `batch-auto --resume` (T-20): predicts each row's log-entry ids
up front, skips fully-logged rows BEFORE the LLM; partially-logged series → review
(never auto-completed); always-on stderr duplicate-append warning. Ledger = one
`map[id]int` (`feedback.LoadExpenseIDCounts`), consumed classify-skips-first;
`appender.PredictEntryIDs` shares `expandEntries` with `ExpandAndAppend` (drift
guard). **T-41 parse boundary slice 1 (s63): `add`/`correct` parse via
`internal/parse`; `--year` on both; `date_year` LIVE there (rung 3); slices
2–4 (auto → batch-auto → apply) pending.** **s65 date/year hardening (PRs #56/#57):**
the boundary now validates the RESOLVED year twice (renderable 1..9999; entry beyond
the CURRENT year refused — provisional, year-scale), reports WHICH rung resolved it
(`YearSource`), and `add`/`correct` warn on stderr when a `date_year` below the current
year is what dated the entry. `config.json` `date_year` → 2026. **T-41 slice 2 (s66): `auto` migrated** — one
`parse.Fields` call, `--year` flag, `date_year` live, stale-year warning;
`appendExpense` collapsed to take a `ParsedExpense` (the five-scalar signature was
what produced T-35); field sentinels added to `parse`. **T-41 slice 3 (s67):
`batch-auto` migrated** — `inputRow` deleted (`ParsedExpense` already held every
field plus the installment count it discarded, which was the ROOT of the 4× re-parse);
`resumeParseErr` deleted as unreachable; `--year`; ONE counted stale-year warning
(T-49); join-id pinned at construction (T-40). `classified.csv`/`review.csv` now carry
the CANONICAL date, which repairs a live silent divergence — `review.ReadQueue` hashed
the raw column into `reviewed.json`'s id while both logs hashed the canonical form, so
`apply` looked up an id it never writes and an id miss there appends silently.
Slice 4 (`apply`, incl. recomputing `entry.ID` alongside the date) DONE s67; **T-21 DONE
s69** (installment count threaded to `apply`; `reviewed.json` gained a REQUIRED
`installments`; apply's summary counts rows, not entries).
Next: T-59 (script the s68 scout) → re-run it clean → the T-42 close.
History → KNOWLEDGE.md "Milestone Log".

**T-63 (s71, PR #65): `batch-auto` writes a THIRD output — `failed.csv`.** Rows the parse
boundary rejected go there, each with its reason as a trailing `#` comment on the SAME line,
so the file is edited in place and re-run as-is (`stripTrailingComment` in `parse3FieldLine`
ignores it, but ONLY when `#` is whitespace-preceded AND the 3 fields are already complete —
so `Mesa #5` stays data and a 4-field `item;date;value;subcategory` still fails loudly).
**No file is written when nothing was rejected: its existence IS the signal.** Writer is
`batch.WriteFailedRows` (new `internal/batch/failed_rows.go`, deliberately independent of the
WS-E-doomed `FailedWriter`). Before this, a rejected row reached NO durable artifact — it was
named on stderr and lost, and `review.html` cannot render a row with no date (T-56).

**T-64 + T-72 (s73, branch `feat/t64-multiplier-notation`): a SECOND installment notation,
and the seam bug found while shipping it.** `405,25 x4` states the PER-INSTALLMENT amount
and MULTIPLIES, the inverse of `405,25/4` which states the TOTAL and divides — same digits,
4× apart. Both orders accepted (`x4`, `4x`, plus retail `4x405,25`), case-insensitive. The
mode switch is the PRESENCE of `x`/`X`; the value/count split is resolved by trying three
readings and requiring **exactly one** to hold, so `4x5` and `646,254x` are REJECTED as
ambiguous rather than guessed. Implemented once, in `utils.ParseCurrencyWithInstallments`.
**T-72, found the same session and folded in:** `review.ReadQueue` called that function
directly and so never applied BR thousands normalization, so a `1.234,56` row hard-errored
and killed the entire review step — the T-54 shape a third time. Fixed by adding
`parse.Value` (the value-only door, mirror of `parse.Date`), having `Fields` delegate to it,
and routing `ReadQueue` through it. Normalization now has ONE call site.
Filed and NOT fixed: T-73 (`classify` still calls `utils.ParseCurrency` directly),
T-74 (a dot-only token like `1.234` resolves 1000× wrong — pre-existing).
⚠️ **`ParseCurrencyWithInstallments` has THREE callers, not two.** The third is
`models.NewExpense`, reached from plain `batch` — which is retired-but-not-deleted and
therefore still REGISTERED and reachable (`batch.go:44`). So `batch` silently gained the
multiplier on a path that writes DIRECTLY to the workbook, unnormalized and untested.
Behavior is probably right (it expands at insert time); it is simply unexercised, and
WS-E must delete that caller. Retired ≠ unreachable — check `AddCommand`, not the label.

**The review template does NOT re-derive the installment count** (checked s73, not
assumed): `installmentBadge(count)` takes the count, the page passes `e.installments`
from the embedded DATA, and `exportReviewed()` emits `s.entry.installments`. No JS parses
`rawValue`, so the multiplier needed no template change — which matters because the
`exportReviewed()` hop is unguarded (T-61) and a JS-side `total/N` re-parse would have
recorded 1 row where 4 belong, with no Go test able to fail.

## Structure
```
cmd/expense-reporter/cmd/  # Cobra subcommands (one file each)
cmd/workbook-inspect/      # Thin wrapper over internal/inspect
internal/                  # batch classifier cli config excel feedback generate
                           # taxonomy inspect logger models parse parser resolver
                           # review apply appender workflow capture telegram
pkg/utils/  config/config.json
```

## Key Rules
- **Cobra pattern** — one `.go` file per subcommand
- **Table-driven tests with testify** — `assert`/`require`
- **Brazilian format everywhere** — DD/MM/YYYY, comma decimal, BRL
- **Error wrapping** — `fmt.Errorf("context: %w", err)`
- **Method extraction** — multi-step function bodies read as named delegated steps
  (≤~15 lines); step-comments promote to helper doc comments (KNOWLEDGE.md)
- **Installments** — "99,90/3" = 3 monthly payments, expanded at APPEND time;
  plain `batch` still expands at insert
- **Parse boundary (T-41)** — commands parse input ONLY via `internal/parse`
  (`ParsedExpense`; `DateString()` = the identity bytes); never call utils
  date/currency parsers from command code. **All six parsers migrated (s67 slice 4);
  `cmd.canonicalDate` is retired.** ⚠️ Its past-year side is UNGUARDED and that is a
  live defect, not a footnote — `Date("29/12/25")` → year 0025, no error; see
  `internal/parse/.memories/QUICK.md`. Command-layer helpers live in `cmd/parse_boundary.go`
  (`parseOptions`, `warnIfStaleConfiguredYear`, `describeParseFailure`) — `parse`
  must not import `config`. `internal/parser` (no "e") is the DYING pre-pivot one

## Deeper Memory → KNOWLEDGE.md
Log-append path (WS-B) · workbook generator design · milestone log (sessions 26–44)
