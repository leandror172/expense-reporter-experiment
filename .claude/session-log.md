# Session Log — Expense Reporter

**Current Session:** 2026-09-07 — Session 76: T-80 COMPLETE — a duplicate row reaches the reviewer, not the log
**Current Layer:** CAPTURE IS SOLVED AND THE DUPLICATE HOLE IS CLOSED. T-80 shipped (PR #69): a row already in the expense log is classified but never auto-appended — it reaches the review page as a badge for a human to rule on. The 2026-03..08 backfill is unblocked on CODE and waits only on a fresh Telegram export; then T-79/T-48/T-76, the six closes, and 5.R6 afterwards with six months of labels behind its denominator.
Most recent entry first. Run `.claude/tools/rotate-session-log.sh` when this grows beyond ~3 sessions.

---
## 2026-09-07 - Session 76: T-80 COMPLETE — a duplicate row reaches the reviewer, not the log

### Context

Opened as a next-steps discussion, not a coding task. Reading the board against the live log showed 2026-01 and 02 closed and March onward empty but for installment rollovers, and that no Telegram export covers 2026-03..09 — so the backfill is blocked on a human export, not on code. An advisor pass then reordered the plan twice: 5.R6 moved AFTER the closes (its rebuild changes which rows the gate admits, so the 94.1% baseline would not transfer, and six closes are the largest labelling opportunity this project has had), and T-80 — which I had described as a risk and then filed under neither do-nor-defer — was promoted to blocking.

### What Was Done

- Researched and planned T-80 end to end: `.claude/plans/t80-duplicate-routing.md`, decisions D1–D10 locked with the user, indexed in `.claude/index.md`.
- Slice 1 (`c04f2be`) — the decision: ledger consulted on EVERY run via `ledgerOutcomeFor`; `--resume` now governs only what a FULL match does; `applyResumeDecision` collapsed into `recordSkippedRow`; the append-phase warning became `refuseDuplicateAppend`, returning an error.
- Slice 2 (`d68db94`) — the signal: a tenth `already_logged` column APPENDED through both writers → `ReadQueue` → `QueueEntry` → two distinct amber badges on the review page.
- Slice 3 (`626715e`) — the honesty: `apply` records and counts found+confirmed, which had been reported in no bucket at all.
- Slice 4 (`133ec16`) — docs: test/README column table, two package QUICKs, review KNOWLEDGE, README, `expect/resume.go`, and the `batch-auto --help` clause that advertised the removed behavior.
- Opened PR #69 against master. Full acceptance suite 79 pass / 0 skip / 0 fail (108.6s) on every run; units green; `check-ref-integrity.py` unchanged at the standing 7 errors / 5 warnings.

### Decisions Made

- **5.R6 moved AFTER the six closes.** `spec` derives from IDF over the corpus denominator, so a dictionary rebuild changes which rows the gate ADMITS — a precision measured on the old admission population does not transfer. Running the closes first multiplies n against a predicate that already has a baseline; running 5.R6 first spends six months of labels on an unmeasured gate and leaves neither number.
- **The T-80 predicate is ANY predicted id present, not ALL** (D6). A full-match-only test would have narrowed coverage while deleting the per-id warning that covered the partial case.
- **Route to review rather than skip or refuse** (user ruling). Skipping loses a genuine second purchase; refusing into `failed.csv` gives a correct row nothing to edit. Review lets the human adjudicate where they already are.
- **`apply`'s gap is in scope, minimally** (D4): a summary bucket, not a new reviewer action. Honouring a confirmed duplicate would widen the unguarded `exportReviewed()` hop, which is the riskiest seam in the codebase.
- **The marker is a string, not a bool** — `logged` and `partial` ask the reviewer different questions, and a flag could not say which.
- **The already-applied detail block is verbose-gated**, unlike the corrections block: a correction WRITES feedback so the user must be told unasked, while an already-applied confirm changes nothing — and an unconditional block would print a wall of warnings on a legitimate re-run.

### Next

- **The user must produce a fresh Telegram export of "Gastos", widest range available** — `ChatExport_2026-04-20` contains ZERO 2026 messages (393, all 2025-05..12), so the filename date says nothing about coverage. Jan/Feb coming along is useful: import them dry and diff as an oracle check, the method that found T-78.
- Merge PR #69, then the agreed pre-close batch: **T-79** (`*` as a third multiplier spelling — user-ruled s75; must amend `.claude/scratch/t75_expected_buckets.py` in the SAME commit and re-run the two-half gate with `--perturb`), **T-48** (`29/12/25` → year 0025 on the rejects re-import path), **T-76** (guard the live-log id invariant).
- Then the six closes 2026-03..08, then **5.R6** with a far better denominator — and measure first how many of the 649 reviewed rows land on the 13 keyword-less leaves, a join over files already on disk.
- T-78's remaining 11 short series (R$ 3.594,44 of missing tail rows) still gated on the user's knowledge of whether those tails were paid.

### Gotchas

- **The s75 gotcha `--dry-run on batch-auto skips the WORKBOOK, not the log append` is WRONG.** `runBatchAuto` gates the append on `!batchAutoDryRun` (`batch_auto.go:206`), and `TestBatchAutoResume_DryRunShowsSkipsWritesNoLog` has asserted both `logUnchangedByResumeSkip` and `noClassificationsLogWritten` all along. It most plausibly derives from the stale `--help` text T-58 already files — **a rule inherited from documentation inherits that documentation's staleness**, while one inherited from a test that can fail does not.
- **`TestReadQueue` cannot tell column 8 from column 9.** Twelve subtests, all green, while the reader read the wrong column — its fixtures set neither distinctly. Third time this session that a test existed, touched the broken behavior, and stayed green; the others were the `-full`-only fixture breakage and `apply-basic`, which had been exercising the silent no-op since the day it was written. **Not missing coverage — coverage that cannot discriminate**, which is why mutation is load-bearing rather than a nicety here.
- **The local model fails on BREADTH, not difficulty.** Two `0` verdicts, both from asking for five declarations across two files: output truncated at roughly the same length each time, and one call inverted the core conditional and invented a `pe.ID` field that does not exist. Every single-function call succeeded (verdicts 1,1,1,1). Scope to one function and state corrections explicitly.
- **`generate_code` does NOT honour the CLAUDE.md tier list.** Its automatic persona routing picked `my-go-q3` for `language=go` even after `warm_model` had loaded `my-go-qcoder`. Pass `model=` explicitly if the documented priority matters.
- **A mutation must COMPILE or it measures nothing** — hit twice this session. Blanking a struct field gave `declared and not used`; the valid mutations changed the DECISION (`autoInsert = false` removed) and the SOURCE COLUMN (`record[8]` for `record[9]`), both of which stayed runnable and then discriminated precisely.
