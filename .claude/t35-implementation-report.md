# T-35 Implementation Report — join-id date divergence (session 61, 2026-07-18)

**Branch:** `feat/t35-join-id-normalization` · **PR:** #50 · **Status:** complete, suite green

Companion doc: `.claude/t35-date-year-semantics.md` — the full date/year survey (six parsers,
where years enter, what must not be "fixed"). That file is the reference; this file is the
work record. Both were written this session; neither supersedes the other.

---

## 1. The bug

`GenerateID` (`internal/feedback/feedback.go:44`):

```
sha256(lower(trim(item)) + "|" + date + "|" + fmt.Sprintf("%.2f", value))[:12]
```

The date goes into the hash **as raw bytes** — not a parsed date, not an instant. So `15/04`
and `15/04/2026` are two different expenses as far as the key is concerned. That id is the
**only** thing joining `classifications.jsonl` to `expenses_log.jsonl`.

Three commands wrote both logs and fed them different forms of the same date:

| Command | → `classifications.jsonl` | → `expenses_log.jsonl` | Result |
|---------|---------------------------|------------------------|--------|
| `auto` | raw CLI arg `15/04` | `FormatDate(parsedDate)` → `15/04/2026` | two ids |
| `batch-auto` | raw CSV field | normalized in `appendOneRow` | two ids |
| `apply` | raw `reviewed.json` field | normalized via `ParseDateWithYear` | two ids |
| `add` | normalized | normalized | ✅ |
| `correct` | normalized | n/a | ✅ |

Nothing errored. The rows silently stopped corresponding.

**Second-order breakage:** `correct` parses its input through `parseExpenseForFeedback`, which
normalizes, so it looked up the *normalized*-date id — and therefore missed **every** row
`auto` had logged under a raw-date id. A user correcting an auto-classified expense got
"no prior classification found."

## 2. Scope correction vs the filed ticket

T-35 was filed against `auto`, with a note to "check `add`/`batch-auto` for the same pattern."
Both halves of that framing were wrong:

- `add` was **already correct** — `parseExpenseForFeedback` (`add.go:168`) returns an
  already-normalized `dateStr`. Same for `correct`, which shares it.
- `apply` was **buggy and not in the ticket**. It writes both logs and has *three* sites
  reading `entry.Date` (`appendNewRows`, `buildFeedbackEntry`, `handleActiveEntry`).

**How `apply` was found:** a grep for fixtures pinning raw `DD/MM` dates returned exactly one
hit — `apply-basic/expected-feedback.jsonl`. The bug was **committed as expected output**,
which is precisely why it never failed. I ran that grep during orientation, saw the hit, and
did not follow it; the advisor pass caught the omission.

**Deliberately out of scope:** `review.ReadQueue` (`queue.go:85`) also builds an id from a raw
date, but its own comment marks it "stable within a review-to-apply cycle only" — a
session-local handle, not the cross-file join key.

## 3. Design decision — normalize at the boundary

A constraint ruled out the obvious fix: **`utils.ParseDateWithYear` accepts only the 2-part
`DD/MM` form and rejects `DD/MM/YYYY`** (`date.go:53`). So "normalize early and let everything
downstream see it" would have broken `appendNewRows`, which called it. The two logs were fed
by two functions with *incompatible* date-format expectations — plausibly why the divergence
took this shape at all.

Options weighed (user chose A):

- **A — normalize once at the boundary (chosen).** `entriesWithCanonicalDates` rewrites every
  `Date` right after `ReadReviewed`, before any consumer. `appendNewRows` switches to
  `ParseDateFlexible`, safe *because* dates now always carry a year, so it never falls back to
  `time.Now()`. One normalization point; drift becomes structurally impossible.
- **B — thread a normalized date to each writer.** Smaller blast radius, but leaves two
  normalization points that can drift apart again — which is the exact failure mode being fixed.

`auto`/`batch-auto` got the same shape at their own boundaries (one re-derivation immediately
after the parse).

**Year rule, pinned by test:** an explicit year in the string always wins; a supplied year is a
fallback for its *absence*, never an override. Test passes a `year` argument differing from the
date's own year so an override would be visible.

## 4. Why no existing test caught it

Every fixture used full `DD/MM/YYYY` dates, following the standing test-memory rule that
non-dry-run fixtures need explicit-year inputs (which exists so pinned dates don't rot at year
rollover). **With a full date, raw and normalized are byte-identical and the bug is invisible.**

The new tests deliberately use short `DD/MM` and assert **id equality only**, never a literal
date — so nothing is pinned to a year and the rollover concern doesn't apply.

⚠ **This exception is a trap for future sessions** and is recorded in
`expense-reporter/test/.memories/QUICK.md`: "fixing" these fixtures to a full year would
**silently disable** the tests rather than break them.

## 5. Tests added

| Test | Kind | Runtime |
|---|---|---|
| `TestApply_JoinIDMatchesAcrossLogs` | acceptance, **deterministic** (no Ollama) | 0.01s |
| `TestAuto_JoinIDMatchesAcrossLogs` | acceptance, Ollama-gated | ~13s |
| `TestCanonicalDate` (6 cases) | unit, table-driven | <0.01s |
| `TestEntriesWithCanonicalDates_*` (3) | unit | <0.01s |

`expect.JoinIDMatchesAcrossLogs()` is the new domain assertion (`test/expect/feedback.go`) —
domain, not upstream `verify`, per the two-package rule.

**`apply` was the better vehicle than `correct`** (the advisor's suggestion) for the
deterministic slot: it is one of the three actually-buggy commands, needs no Ollama, and takes
its year from an explicit `--year`.

## 6. Verification — RED proven, not assumed

Both acceptance tests were confirmed failing **before** the fix, with the divergent id pair
printed. A green test never seen red proves nothing.

| Test | RED evidence |
|---|---|
| `apply` | `f0c3bf1293f3` vs `8c434a0b64c0` |
| `auto` | `88d8db338c3b` vs `98d8e31dbaf9` |

The `apply` pair is an **independent corroboration**: those are exactly the two hashes T-18
documented for `Uber Centro` at `15/04` vs `15/04/2026`, derived in a different session for a
different bug.

For `auto` the fix line was temporarily commented out, the test re-run (RED), then restored and
rebuilt — verified by grep that the line came back.

**Final state:**
- Full acceptance suite **green, 840.583s**
- **56** acceptance test functions (54 session-60 baseline + 2 new); both new tests confirmed
  present via `-list` — checked because a suite that silently stops running tests looks
  identical to one that passes
- Unit suite green; `go vet -tags=acceptance ./...` clean
- `go vet` caught a unit-test call site (`apply_test.go:45,77`) that plain acceptance runs
  would have missed after the `appendNewRows` signature change

**Fixture flipped:** `apply-basic/expected-feedback.jsonl` line 3 `10/05` → `10/05/2026`.
Lines 1–2 are *seeded* pre-existing content and stay raw — which is itself meaningful: it
proves `apply` does not rewrite existing log lines, only what it writes.

## 7. Local model usage

Four `generate_code` calls to `my-go-qcoder`, per the local-model conventions.

| # | Task | Verdict | Note |
|---|---|---|---|
| 1 | `expect.JoinIDMatchesAcrossLogs` | **2** | used as-is; compiled against existing imports |
| 2 | apply acceptance test | **1** | doc comment cited invented dates |
| 3 | `canonicalDate` helper | **1** | double-parsed (called `ParseDateFlexible`, discarded it, re-parsed); restructured to branch first |
| 4 | `canonicalDate` unit tests | **1** | hallucinated `Reviewed` fields into `want` (would fail); one subtest asserted on the *input*, testing nothing |
| 5 | auto acceptance test | **1** | violated the explicit no-raw-`expect.*`-in-`Then` rule and invented a 2-arg signature for a no-arg function |

Pattern worth carrying forward: the model is reliable at *structure* (harness shape, gates,
table-driven skeletons) and unreliable at *API specifics* — it invents signatures and struct
fields. Passing the file under test as `context_files` fixed shape but not invention.

## 8. Corrections to earlier claims in this session

Recorded because both were stated confidently and were wrong:

1. **"You said one small PR"** — I attributed that framing to the user; it was my own wording,
   repeated back after the advisor echoed it. The user only ever selected "T-35 + T-36".
2. **T-36's `gofmt` claim** — filed as "5 pre-existing `internal/batch/` files, CRLF-era". It is
   **15 files across seven packages**, and not CRLF: `file` reports plain ASCII/UTF-8 and the
   diffs are struct-comment alignment. `.gitattributes` is not the lever.
3. **`auto-basic/input.csv`** — T-36 called the fixture unused. The *directory* is used
   (`auto_test.go`, `feedback_test.go`); only `input.csv` is the ghost.

## 9. Found, not fixed → T-37 / T-38

Both verified by exhaustive grep, not inferred. Detail in `t35-date-year-semantics.md` §5.

- **T-37 — `config.json` `date_year` is dead.** `DateYear` appears **exactly once** in the
  whole codebase: the struct field declaration (`internal/config/config.go:13`). Nothing reads
  it. Yet `KNOWLEDGE.md` states year rollover "requires updating `date_year` in config" — an
  instruction that has never had any effect. Either wire it as the fallback year (replacing
  `time.Now().Year()`) or delete it and correct the doc.
- **T-38 — `utils.ParseDate` hardcodes year 2025** (`date.go:41`) and is **live**:
  `internal/batch/processor.go` → `parser.ParseExpenseString` → `models.NewExpense` →
  `ParseDate`. Every `DD/MM` date entered through the plain `batch` command is stamped 2025
  regardless of the actual year. Touches workbook data, so deliberately not bundled here.

Neither is in `tasks.md` yet — that is a handoff-time edit per the project convention.

## 10. Known coverage gap

The `batch-auto` feedback write has **no dedicated test**. Its fixtures all use full
`DD/MM/YYYY` dates, which by construction cannot expose this bug class. The fix is the same
one-line pattern the `auto` and `apply` tests do cover. Closing it needs either a short-date
batch-auto fixture (~2 min of Ollama runtime, against a suite already near T-08's timeout
ceiling) or a unit-level seam around `logConfirmedFeedbackForRow`.
