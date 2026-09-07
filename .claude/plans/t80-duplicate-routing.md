# T-80 — a duplicate row must reach the reviewer, not the log

*Plan drafted session 77 (2026-09-07). Status: **D1–D5 LOCKED** (user, s77) — nothing implemented.*
*Prerequisite for the 2026-03 → 2026-08 backfill. Blocking, per the s77 advisor review.*

## The defect, as measured

s75, first close made from a Telegram export. The run printed

```
⚠  duplicate: "<item>" already in expense log (id 4f1d167f9e55)
```

and wrote the row anyway; the id then appeared **twice** in `expenses_log.jsonl` with
identical date and value (1 of 18 appended rows). Two halves compound: the warning goes
only to stderr scrollback, **and** the row classified at 95% so it was AUTO-appended before
review — so the one place a human would catch it is the one place it does not appear.

Both logs are append-only, so the only removal path is `close-cycle.sh restore`, i.e.
rolling back the whole close. **The decision therefore has to be taken before the expensive
human review, not after it.**

## Why it is blocking for the backfill, specifically

The `D1` window in `internal/capture/classify.go` resolves a bare `DD/MM` against the
**message's own `sentAt`**, over `[-180, +7]` days. A message typed in March reporting a
January purchase resolves to January — correctly — and lands in a month that is **already
closed**. The live log already carries generated installment-rollover rows into those
months (2026-03: 10, 2026-04: 5, 2026-05: 2, 2026-06: 1), so the collision surface is real
and grows with every month closed.

## What the research turned up

### F1 — the current behavior is deliberately pinned by a passing test

`test/batch_auto_resume_test.go:TestBatchAutoDuplicateWarning_FiresWithoutResume`, fixture
`test/fixtures/batch-auto-dup-warning/`. Its own words: *"a row whose id already exists in
the log appends AGAIN (append-only default preserved) and emits exactly one stderr
warning."* The fixture's `expected-expenses_log.jsonl` carries the row **twice**, and
`input.csv` documents the intent in a comment.

So T-80 is a **deliberate behavior change to a documented contract**, not a bug fix over
unspecified ground. The rationale being overturned conflates two things: the *log* is
append-only (true, structural, unchanged by this work) with *batch-auto should append a
duplicate* (a choice). Routing to review mutates nothing; the log stays append-only.

### F2 — `apply` will silently swallow a reviewer-confirmed duplicate

This is the finding that changes the shape of the fix.

`cmd/apply.go:handleActiveEntry` looks each confirmed/corrected entry up in
**`classifications.jsonl`** via `feedback.FindLatestEntry(classifPath, entry.ID)`:

- `!found` → new row → appended.
- `found` + `confirmed` → **no-op.** No append, no warning, no summary line.
- `found` + `corrected` → correction feedback only, still no expense-log append (T-65).

`printSummary` reports Appended / Failed / Skipped / Pending / corrections. A
`confirmed + found` entry is counted in **none of them** — `total` includes it, so the
buckets do not sum to `total`, and nothing names the gap.

A genuine same-day duplicate purchase hashes to the same id by construction
(`GenerateID` = `item|date|value`; the live log carries **8** such legitimate pairs). So
under the naive fix the sequence is:

1. batch-auto routes the duplicate to review instead of appending it. ✅
2. The reviewer inspects it and confirms — *yes, this really is a second purchase.* ✅
3. apply finds the prior classification entry and **silently drops it.** ❌

**T-80 alone converts a silent over-count into a silent under-count.** The reviewer's
verdict is unable to be honored in the one direction the fix exists to enable. Any plan
that stops at step 1 makes the budget wrong in a new way and removes the warning that
currently at least hints at it.

### F3 — the ledger is consumable, and consumed in two different phases

`feedback.LoadExpenseIDCounts` → `map[id]int`, a **budget**, not a set.

| Site | When | Consumes |
|---|---|---|
| `evaluateResumeSkip` (`batch_auto_resume.go`) | classify phase, **only under `--resume`** | all ids, on a full match |
| `warnIfDuplicate` ← `appendOneRow` | append phase, **always on** | one per warned id |

Consequences that constrain the fix:

- Moving detection to the classify phase means the append-phase consumption must go, or
  ids get double-consumed.
- `--resume` + full match already **skips** safely (never classified, never appended,
  marked `(already logged)` in classified.csv, excluded from review.csv). **The T-80
  defect is specifically the no-`--resume` path.**
- Intra-batch duplicates are invisible today and deliberately so: `DuplicateWarningCount`'s
  doc says the warning *"must fire for a pre-existing log duplicate but NOT for a second
  in-CSV legitimate duplicate."* The ledger models *what is in the log*, not *what this run
  wrote*. Two identical rows in one input file, with an empty log, both append unwarned.

### F4 — the advisory-column path is already paved

A1's `keyword_hint` (s72) is the exact precedent for getting a signal to the reviewer
without crossing the unguarded `exportReviewed()` JS hop (T-61). Four files:

| File | Change |
|---|---|
| `cmd/batch_auto.go` | field on `classifiedRow`; 10th column + header in both CSV writers |
| `internal/review/queue.go` | field count `9` → `10`; read `record[9]` |
| `internal/review/types.go` | `QueueEntry` field, `json:"...,omitempty"` |
| `internal/review/template/review.html` | badge beside the model's answer (~line 1122) |

It must **not** round-trip through `exportReviewed()` — it is input *to* the human, not
part of their decision, exactly as the keyword hint is. Empty means "not a duplicate";
never a placeholder.

## Decisions to lock before implementing

- **D1 — where does the check run: before or after classification? → LOCKED: AFTER.**
  *Before* is cheap and deterministic (no model → the acceptance test runs in the `-short`
  group), but the reviewer then adjudicates a row with no predicted category.
  *After* costs ~12 s on a row that is probably a mistake, but hands the reviewer the
  model's suggestion alongside the duplicate badge.
  Mirrors the existing `partial` mechanism in `classifyLines`,
  which already classifies and *then* forces review for a partially-logged series. The
  reviewer needs the suggestion to adjudicate. Accepts an Ollama-gated acceptance test,
  which the reading guide notes is unavoidable for CSV-shape assertions anyway.

- **D2 — does a review-routed row consume its ledger budget? → LOCKED: YES, ON A FULL MATCH
  ONLY.** Exactly today's semantics, unchanged: `evaluateResumeSkip` consumes inside its
  `allPresent` branch, so a *partial* hit consumes nothing and a second partial row flags
  again. "YES" is NOT unconditional — do not read it as consume-on-every-hit.
  Consuming affects only a *later row in the same batch* with the same id; that row is an
  intra-batch duplicate, which is D5's out-of-scope case.

- **D3 — does `--resume`'s skip behavior change? → LOCKED: NO.** Its purpose is
  re-running an interrupted batch, where skipping is correct. Worth noting the inverse risk
  — `--resume` silently drops a genuine second purchase — but that is a separate defect,
  not this one.

- **D4 — is `apply`'s silent no-op (F2) in scope? → LOCKED: YES, MINIMALLY.**
  Add an `already applied` bucket to `printSummary` so
  the buckets sum to `total`. ~10 lines, no behavior change, converts silent → visible.
  A per-row stderr warning is the wrong shape: `confirmed + found` also fires on every
  legitimate re-run of apply (T-20 idempotency), so it would be noise.
  Making a confirmed duplicate actually *append* needs a new reviewer action crossing the
  unguarded `exportReviewed()` hop — file it, do not build it here.

- **D5 — intra-batch duplicates (F3)? → LOCKED: OUT OF SCOPE**, filed separately.

- **D6 — WHICH ledger predicate? → LOCKED: ANY predicted id already in the log**, i.e. both
  `resumeSkipFull` AND `resumePartial` route to review. **A full-match-only predicate would
  NARROW coverage while removing the warning that covered it.** Worked case: an installment
  row `99,90/3` where 1 of 3 predicted ids is already logged, run WITHOUT `--resume`. Under
  all-present it is not a duplicate → the gate passes → `ExpandAndAppend` writes 3 rows, one
  re-duplicating the logged one. Today's per-id append-phase warning DOES catch that. Verified
  against `evaluateResumeSkip`: 3 ids, 1 in the ledger ⇒ `allPresent=false`, `anyPresent=true`
  ⇒ `resumePartial`. The existing predicate is already correct; it is merely locked behind
  `if resume`.

- **D7 — keep the append-phase duplicate warning? → LOCKED: NO — replace it with an ERROR.**
  With D2 consuming in the classify phase, `ledger[id]` is 0 for every row that reaches
  `appendOneRow`, so `warnDuplicateEntries` there could only ever print nothing — and by this
  repo's own doctrine a check that can only print is not a guard. A hit at that point means the
  classify-phase check MISSED, which is an invariant violation, so it returns an error: the row
  downgrades to `AutoInserted=false` + `Error`, lands in review.csv, and the command exits
  non-zero — the same shape `appendClassified` already applies to an append failure, and the
  fail-fast shape of `preflightLogPath`.

- **D8 — `--dry-run` interaction. NOTE: a standing memory item is STALE.** The s75 gotcha list
  says *"`--dry-run` on `batch-auto` skips the WORKBOOK, not the log append — do not reach for
  it as a safe probe mid-close."* **That is wrong.** `runBatchAuto` gates the append:
  `if !batchAutoDryRun { appendErr = appendClassified(...) }` (`batch_auto.go:206`), and the
  existing green `TestBatchAutoResume_DryRunShowsSkipsWritesNoLog` asserts BOTH
  `logUnchangedByResumeSkip` and `noClassificationsLogWritten`. The note most plausibly derives
  from the stale `--help` text T-58 already files. **Correct it in the handoff.**
  Consequence for this work: moving the check into the classify phase means `--dry-run` now
  REPORTS duplicates where it previously reported none (the old warning lived in the skipped
  append phase). That is an improvement — dry-run becomes a genuine duplicate probe — and it
  must be stated, not discovered.

- **D9 — set the badge INDEPENDENTLY of the gate verdict.** `if duplicate { autoInsert = false;
  row.Duplicate = true }`. Nesting the badge inside the override (`if autoInsert && duplicate`)
  silently drops it for a duplicate that was ALSO gate-refused — the reviewer then sees an
  ordinary review row and re-confirms it, which is the exact harm this task exists to stop.

- **D10 — keep `"already in expense log"` as a substring of the new REVIEW line.**
  `expect.duplicateWarningMarker` is that exact string. Audited: it has exactly ONE assertion
  site, `DuplicateWarningCount(1)` in the test T-80 changes, and NO `DuplicateWarningCount(0)`
  exists — so there is no vacuous-check risk today. Preserving the substring keeps that
  assertion meaningful across the behavior change instead of retiring a working guard.
  Relevant to the backfill (the same purchase typed twice on one day), but a distinct
  mechanism: it needs the ledger to grow as the run writes, not just be consumed.

## Implementation slices

1. **Decision** — **hoist `classifyResumeDecision` out of the `if resume` block** so the
   ledger is consulted on every run, and let `resume` control only what a FULL match does.
   This REMOVES a branch rather than adding one:

   | ledger outcome | `--resume` | behavior |
   |---|---|---|
   | full match | yes | skip — unchanged |
   | full match | no | **review + badge (T-80 core)** |
   | partial | either | **review + badge** (today: resume-only) |
   | none | either | gate decides — unchanged |

   Then force `autoInsert = false` and print a REVIEW line naming the reason. Remove the
   append-phase `warnDuplicateEntries` call so the ledger is consumed once — see D7.
2. **Signal** — the 10th CSV column through to the review page badge (F4's four files).
3. **Honesty** — `apply`'s `already applied` summary bucket (D4).
4. **Docs** — update the overturned fixture comment and the `DuplicateWarningCount` doc;
   `.claude/index.md` entry for this plan; task-board updates at handoff.

## Tests

Acceptance-first, per the repo convention.

**Changed — `TestBatchAutoDuplicateWarning_FiresWithoutResume`.**
*Previous behavior:* without `--resume`, a row already in the log is classified,
gate-passed, **appended a second time**, and warned about once on stderr; the fixture's
`expected-expenses_log.jsonl` holds the row twice.
*New behavior:* the same row is classified, **routed to review** (`auto_inserted=false`,
present in `review.csv`, duplicate badge set), and the expense log is **unchanged** —
`expected-expenses_log.jsonl` drops to the single seeded row.
*Reason:* the append was unrecoverable without rolling back the whole close, and the row
never reached the one surface a human inspects. Ollama-gated (the row must gate-pass to
prove the duplicate check overrides a PASS, not merely that a non-gating row went to
review).

**New — the seam test.** `classified.csv` written by batch-auto is read back by
`review.ReadQueue`, asserting the duplicate flag survives. The T-54/T-72 shape: feed real
producer bytes to the real consumer, never a hand-authored fixture.

**New — unit, deterministic.** The duplicate decision as a pure function (the
`classifiedRowFromPrediction` precedent): a gate-PASSING result + a ledger hit ⇒ not
auto-insertable. This is the one that survives without Ollama.

**New — `apply`'s summary bucket** (D4): a confirmed entry already present in
`classifications.jsonl` is reported, and the buckets sum to `total`.

## Verification

- **Mutation is mandatory, and it must compile.** Deleting a struct field makes the package
  unbuildable — that is the compiler talking, not the test (s72). Mutate the *decision*:
  flip the forced `autoInsert = false` back to leaving the gate's verdict alone, and confirm
  precisely the T-80 scenarios go red while the rest of the suite stays green.
- **`-full` before calling it done.** Widening 8→9 columns passed `-short` and the whole
  unit suite, then failed **six** `-full` scenarios — every `OutputFileHasColumns`
  assertion sits behind `RequireOllama`. `0 skipped` is the load-bearing number, not the
  scenario count; corroborate with a duration.
- **Append the column, never insert it.** `expect.NoneWereAutoInserted` reads index 6
  positionally and would silently retarget.
- Unload any foreign resident model before `-full` (`POST /api/generate`, `keep_alive:0`).

## Open risk, recorded

`review.csv` and `classified.csv` have **two producers** of the `auto_inserted` cell —
`writeClassifiedCSV` renders `fmt %v`, `writeReviewCSV` hardcodes `"false"` — kept equal
only because review.csv filters auto-inserted rows out (T-70). This work does **not** relax
that filter: a duplicate is routed by setting `AutoInserted = false`, so the hardcode stays
true. Do not let the implementation drift into relaxing it.
