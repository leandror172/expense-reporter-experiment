# Session Log — Expense Reporter

**Current Session:** 2026-08-19 — Session 72: the auto-insert gate measured at 94.1%, and the keyword's second opinion reaches the reviewer (A1, PR #66)
**Current Layer:** Close-cycle follow-up — the backlog is now measured rather than argued. A1 shipped (PR #66); T-65 supersede-by-append is next, DESIGN ONLY by the user's call
Most recent entry first. Run `.claude/tools/rotate-session-log.sh` when this grows beyond ~3 sessions.

---
## 2026-08-19 - Session 72: the auto-insert gate measured at 94.1%, and the keyword's second opinion reaches the reviewer (A1, PR #66)

### Context

Opened as a "PRs merged, let's discuss next steps" session against a backlog of three undecided findings from the first real monthly close (T-64/T-65/T-66). It became a measurement session: two numbers extracted from data session 71 had already written to disk reordered the plan before any code was written, and one of them retired a claim WS-D had been argued from since session 52.

### What Was Done

- **Measured the A1 keyword-hint predicate on real data** (`.claude/a1-keyword-hint-precheck.md`) using the PRODUCTION `MatchStrength` rather than the report's approximate Python tokenizer: fires on **14 of 65** rows, keyword right **71.4%**, recovers **10 of 25** corrections, only 2 noise fires. Verdict GO.
- **Measured ABSOLUTE agreement-gate precision at 94.1%** (`.claude/b1-gate-precision-measurement.md`) — 1 correction in 17 gate-passing rows vs **50.0%** error on the 48 gate-refused rows, an 8.5× discrimination on a full unselected month. This is the number T-32 declared unmeasurable and WS-D has been blocked on since session 52; it was already on disk.
- **Corrected `t42-close-findings.md` in place**, inside the ref block readers resolve: its claim that the 17 auto-inserted rows were appended "without any human seeing them" is false, and contradicted by its own `4x álcool 70` example.
- **Shipped A1 (PR #66, 5 commits):** `classifier.KeywordHint` + a ninth `keyword_hint` CSV column + `QueueEntry.KeywordHint` + an amber `.kw-hint` badge on the review page, with deterministic behavioral acceptance coverage.
- **Closed a model-only blind spot** by extracting `classifiedRowFromPrediction` as a pure function, making the one line that decides what the keyword is compared against unit-testable without Ollama.
- **Swept the memories and docs**: retired the now-false "absolute precision is unmeasured" claim from four live places, and fixed `test/README.md`'s batch-auto column table, which had listed seven columns and been missing `type` since RUI-4 in session 33 — ~39 sessions stale with nothing breaking.
- Recorded the T-64 direction (`x4` MULTIPLIES) in agent memory the moment it was confirmed, since a wrong direction there is a silent 4× error.

### Decisions Made

- **B1 (gate-to-review) was DROPPED.** Its two justifications were eliminating unrepairable rows and obtaining the absolute-precision measurement. Ten minutes of joining existing files delivered the second for free and dissolved the first — the rows were never unseen. T-65 was promoted in its place.
- **The keyword hint is ADVISORY and never applied.** At 71.4% precision, auto-preferring it would inject errors on roughly 3 of every 14 rows; displaying it beside the model's answer cannot.
- **ONE new column, appended, not two.** No `gate_passed` — without B1's flip, `auto_inserted` still means "the gate passed", so a second column would have had no consumer. Appending keeps `auto_inserted` at index 6 for the positional verifiers.
- **The hint deliberately does NOT round-trip through `exportReviewed()`** — it is input TO the human, not part of their decision, and a value crossing that unguarded JS hop would be a second T-61.
- **Strict predicate only** (unambiguous, specificity 1.00). Not because a middle setting is impossible — `MatchStrength` can structurally return unambiguous-at-0.8 — but because `review` loads no keyword index and has no consumer for the extra signal. Measured: every row scoring in [0.70, 1.00) this month was ambiguous, so the 0.70 threshold was inert.
- **T-64 direction CONFIRMED by the user: `405,25 x4` MULTIPLIES** (the written value is per-installment). Implementation still pending.
- **T-65 is design-only for now** at the user's direction: build supersede-by-append, but write no code yet.

### Next

- **T-65 supersede-by-append — DESIGN DOCUMENT ONLY, no code.** Two constraints drive it. (1) The supersede record must key off the **STORED** id: `expenses_log.jsonl`'s ids are deliberately stale (hashes of pre-merge year-less dates), the one place this repo's "recompute, never trust" rule is inverted on purpose, and 347 cross-log joins depend on it — recomputing drops them to zero. Getting this backwards makes superseding match nothing, silently, the same shape as the slice-4 bug. (2) A supersede record needs a date that survives `scanEntries`' year filter; a tombstone with no date or a year-0 date leaks into EVERY year, the mechanism behind s70's 107K workbook of 2025 data labelled 2026.
- Then T-64 implementation (direction now confirmed): `x4` multiplies, in `internal/parse`, with both notations pinned in one test so they can never be conflated.
- Lines 19 and 45 of `expenses-2026-01.csv` remain unprocessed; 45 also needs a date and a value the user can reconcile (299,00 + 49,90 ≠ 646,25).
- PR #66 awaits review.
- Still open and untouched: T-58 (confirmed live — `batch-auto --help` still claims it inserts "into the workbook"), T-69, T-56, T-57, 5.R6.

### Gotchas

- **A mutation that fails to COMPILE proves nothing.** Deleting the hint field from `ReadQueue` made the package unbuildable — red from the compiler, not from a test. Swapping the source column to 7 kept it compiling and then discriminated precisely: only the seam test failed, while `internal/review` stayed green, proving its own tests cannot tell where the hint came from.
- **The `-short` group is structurally blind to CSV format changes.** Widening the contract passed `-short` and the entire unit suite, then failed **six** `-full` scenarios. Every assertion about `classified.csv`'s shape sits behind `RequireOllama`, because producing one means classifying. Green on the fast group means "nothing deterministic broke", never "the contract holds".
- **A join keyed on item silently collapsed duplicate rows.** The first gate-precision join reported 26 corrections against a known total of 25 — a one-row overshoot that was the only visible symptom of `San michel` ×3 and two other duplicate item names. Had they fallen the other way the arithmetic would have reconciled and the wrong number would have shipped looking right. Fixed with a POSITIONAL join verified on item AND date across all 65 pairs.
- **Reverting a mutation restored the guard clauses in the WRONG ORDER**, and no test could have caught it because the four clauses are independent guards all returning the same value. Found only by reading the file back after the revert.
- **`calls.jsonl` DOES exist** at `~/.local/share/ollama-bridge/calls.jsonl` (908 entries) — the session-71 note that it is nowhere on this filesystem is wrong. But a **backgrounded** `generate_code` call silently ignores `output_file` (content returned, nothing written) and injects no verdict template.
- **A local-model draft used a helper that does not exist** (`esc` instead of `escapeHtml`), which would have thrown a ReferenceError on every hinted row. The prompt told it to assume that helper — the miss was the prompt's, which is exactly why the name needed verifying rather than trusting.
- **The shell's cwd persisted across tool calls and short-circuited a `cd`**, making a chained `&&` report a test failure that had not happened. Use absolute paths.
