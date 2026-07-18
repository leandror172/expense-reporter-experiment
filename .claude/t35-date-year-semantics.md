# Date & Year Semantics Across the System (T-35 survey, session 61)

Written while fixing T-35. The join-key bug was a *symptom*: this repo has **six different
date-parsing functions** with different year rules, and two of them are reachable from the
same command. This is the map.

<!-- ref:date-year-semantics -->

## 1. The six parsers, and what each does about a missing year

| # | Function | Accepts | Year when omitted | Notes |
|---|----------|---------|-------------------|-------|
| 1 | `utils.ParseDate` (`pkg/utils/date.go:12`) | `DD/MM` only | **hardcoded 2025** | See §5 — a live time bomb |
| 2 | `utils.ParseDateWithYear(s, year)` (`:53`) | `DD/MM` only — **rejects** `DD/MM/YYYY` | the `year` argument | Strict-2-part; this rejection shaped the T-35 fix |
| 3 | `utils.ParseDateFlexible` (`:93`) | `DD/MM` **and** `DD/MM/YYYY` | `time.Now().Year()` | The nondeterminism source |
| 4 | `taxonomy.parseDate` (`internal/taxonomy/loader.go:435`) | both | returns `year = 0` (a sentinel, not a guess) | The only one that refuses to invent a year |
| 5 | `cmd.canonicalDate` (`apply.go`, **new in T-35**) | both | the `--year` flag | Composes #2 and #3 to avoid #3's `time.Now()` |
| 6 | `cmd.parseExpenseForFeedback` (`add.go:168`) | both (delegates to #3) | via #3 → `time.Now().Year()` | Returns an **already-normalized** `dateStr` — why `add`/`correct` were never buggy |

**Canonical output form** is `DD/MM/YYYY` via `utils.FormatDate` (`:130`), zero-padded.

## 2. Where a year enters the system

| Source | Applies to | Default |
|--------|-----------|---------|
| The date string itself (`DD/MM/YYYY`) | everything | — wins over every fallback below |
| `--year` flag on `apply` | `apply` entry dates | `time.Now().Year()` |
| `--year` flag on `generate-workbook` | which entries land in the sheet | `time.Now().Year()` |
| `time.Now().Year()` inside `ParseDateFlexible` | `auto`, `batch-auto`, `add`, `correct` | implicit, invisible |
| Hardcoded `2025` inside `ParseDate` | `models.NewExpense` → plain `batch`/`add` parsing | not configurable |
| `config.json` `date_year` | **nothing** | see §5 |

**Rule that now holds everywhere:** an explicit year in the string always wins; a supplied
year is a *fallback for its absence*, never an override. `canonicalDate` pins this with a
test that passes a year argument differing from the date's own year.

## 3. What T-35 actually was

`GenerateID` (`feedback.go:44`) = `sha256(lower(trim(item)) + "|" + date + "|" + %.2f(value))[:12]`.

Note what is in that hash: **the date as a raw string**. Not a parsed date, not a timestamp —
the literal bytes. So `15/04` and `15/04/2026` are two different expenses as far as the hash
is concerned, and this id is the *only* key joining `classifications.jsonl` to
`expenses_log.jsonl`.

Three commands wrote both logs and fed them different forms of the same date:

| Command | → `classifications.jsonl` | → `expenses_log.jsonl` | Result |
|---------|---------------------------|------------------------|--------|
| `auto` | raw CLI arg (`15/04`) | `FormatDate(parsedDate)` (`15/04/2026`) | two ids |
| `batch-auto` | raw CSV field | normalized in `appendOneRow` | two ids |
| `apply` | raw `reviewed.json` field | normalized via `ParseDateWithYear` | two ids |
| `add` | normalized (#6 above) | normalized | ✅ agreed |
| `correct` | normalized (#6) | n/a | ✅ |

Nothing errored. The rows simply stopped corresponding — a silent failure, which is why it
survived from T-11 (when date normalization landed) until now.

**Why the tests never caught it:** every fixture used full `DD/MM/YYYY` dates, following the
existing "non-dry-run fixtures need explicit-year inputs" rule (which exists so pinned dates
don't rot at year rollover). With a full date, raw and normalized are byte-identical and the
bug is invisible. The new join-id tests deliberately use short `DD/MM` — safe only because
they assert id *equality* and never a literal date, so nothing is pinned to a year.

**The fix:** canonicalize to `DD/MM/YYYY` once, before any writer sees the value —
`entriesWithCanonicalDates` at the `apply` boundary, and a single re-derivation right after
parsing in `auto`/`batch-auto`. Fixing it per-writer would have left the same drift latent.

## 4. Year behavior that is correct and deliberate (do not "fix")

- **Installments cross the year boundary honestly.** `appender.addMonths` shifts each
  installment a real month forward, so `99,90/3` dated `15/11/2026` yields entries in Nov,
  Dec and **Jan 2027**. `rollover.csv` was retired in WS-B — the next-year date is now just
  a date.
- **Installment ids legitimately differ between the two logs.** Expansion rewrites the item
  to `X (i/N)` and shifts dates, so `classifications.jsonl` (one row, base item) and
  `expenses_log.jsonl` (N rows, suffixed) *cannot* share an id. The join is meaningful only
  at `count == 1` — which is why the join-id tests use a single expense.
- **`review.ReadQueue` builds a deliberately year-less id** (`queue.go:85`, comment: "stable
  within a review-to-apply cycle only"). This is a session-local handle, not the cross-file
  join key, and is correctly out of T-35's scope.
- **`taxonomy.parseDate` returning `year = 0`** is the right call for the generator: a
  year-less entry matches whatever `--year` asks for, rather than being guessed into the
  wrong sheet (T-11).

## 5. Open problems this survey surfaced (NOT fixed in T-35)

1. **`config.json` `date_year: 2025` is dead.** `internal/config` declares the field
   (`config.go:13`) and **nothing reads it**. The repo KNOWLEDGE.md documents it as "applied
   to DD/MM dates during parsing" and says year rollover "requires updating `date_year`" —
   that instruction has no effect. Either wire it as the fallback year (replacing
   `time.Now().Year()` in the commands) or delete it. → **T-37**
2. **`utils.ParseDate` hardcodes 2025** (`date.go:41`), reached via `models.NewExpense` ←
   `internal/parser` ← the plain `batch`/`add` parse path. Every DD/MM date entered there is
   silently stamped 2025 regardless of the actual year. → **T-38**
3. **`time.Now().Year()` still reachable** in `auto`/`batch-auto` for short dates. Harmless
   for the join (both logs now agree), but a December batch entered in January lands in the
   wrong year. The `batch-auto --resume` help already warns to use `DD/MM/YYYY` for December
   batches (T-20). A configured fallback year (item 1) would close this properly.
4. **`GenerateID` hashes a formatted string, not a normalized instant.** The whole class of
   bug exists because the key is format-sensitive. Hashing a canonical instant (e.g. the
   RFC-3339 date) would make it format-proof — but it rewrites every existing id, so it is a
   migration, not a fix. Noted, not proposed.

<!-- /ref:date-year-semantics -->
