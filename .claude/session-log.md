# Session Log — Expense Reporter

**Current Session:** 2026-06-30 — Session 43: WS-B slice 3 — batch-auto → log-append (+ T-16/T-19 breadcrumbs)
**Current Layer:** "WS-B: retire workbook insertion (commands → log-append)"
Most recent entry first. Run `.claude/tools/rotate-session-log.sh` when this grows beyond ~3 sessions.

---
## 2026-06-30 - Session 43: WS-B slice 3 — batch-auto → log-append (+ T-16/T-19 breadcrumbs)

### Context

Continued from the merged PR #36 (T-13). Opened by discussing next steps, then the T-19 escape-hatch documentation gap and T-16 doc cleanup, then implemented WS-B slice 3 (`batch-auto` → log-append), the standing "next".

### What Was Done

- **T-16 + T-19 breadcrumbs (session start):** corrected CLAUDE.md's classification tier list (q3 is primary; qcoder NOT recommended — enum validity is GRAMMAR-enforced, model-independent; dropped the debunked "qcoder required for 5.7+" claim). Added `TODO(T-19)` at the off-enum drop (`classifier.go`) + the auto-insert threshold (`decision.go`), and a T-09↔T-19 cross-link so the escape-hatch gap surfaces when WS-D leans on the model path. Committed `462d222`-era.
- **WS-B slice 3 (`batch-auto` → log-append)** — advisor-reviewed plan + 4 code/test commits + 1 memory commit on branch `feat/ws-b-slice3-batch-auto-log-append`:
- `batch-auto` appends auto-rows to `expenses_log.jsonl` via `appender.ExpandAndAppend` (no workbook write). `rollover.csv` retired (cross-year installments carry their real next-year date).
- **Failure honesty:** an append error downgrades the row (`AutoInserted=false` + `Error`) → honest summary count, row → `review.csv`, non-zero exit; CSVs written AFTER the append. `preflightLogPath` fails fast on an unwritable log before classifying.
- **Acceptance rewired to the log:** assert via `verify.ExpenseLogMatches`; installments fixture asserts the N expanded dated lines; rollover fixture INVERTED (`NoRolloverFileCreated` + next-year dates); the two workbook-failure tests repurposed (pre-flight fail-fast deterministic test + the failure-downgrade moved to a unit test).
- **Cleanup:** deleted dead `logExpense`; renamed `auto`'s insert→append vocabulary (`✓ Appended`, separate commit for a clean boundary).
- **Deprecated** `workflow.InsertBatchExpensesFromClassified` (`// Deprecated:`); kept + unit-tested for WS-E.
- Updated 4 QUICK/KNOWLEDGE memories (expense-reporter + test).
- Verified green: 554 unit tests; full `-tags=acceptance` batch-auto + type-routing group **15/15** (864 s on q3) — all 5 dry-run survivors, the 3 rewrites, the inverted rollover test, the deterministic pre-flight & downgrade tests, and `TypeRoutingCycle_1..4`.
- Wrote the work report (`.claude/ws-b-slice3-implementation-report.md`) and **opened PR #37** (`feat/ws-b-slice3-batch-auto-log-append` → master).

### Decisions Made

- **`--dry-run` = classify + write CSVs, no log append** — there is no workbook to skip anymore; dry-run also skips the pre-flight.
- **Failure-downgrade verified by a UNIT test, not acceptance** — the pre-flight makes an acceptance-level append failure unreachable (a log that passes pre-flight won't fail at append), so a unit test pointing the log at a missing parent dir is both deterministic and a stronger guard.
- **Deprecate-not-delete scope is NARROW** — only the classified-batch variant orphans; plain `batch` keeps `InsertBatchExpenses` + the rollover/excel machinery LIVE, so WS-E must not delete them with the classified variant.
- **Explicit-year fixture inputs + clean-dividing installment values** — the append path reformats dates to `DD/MM/YYYY` (`ParseDateFlexible` fills bare `DD/MM` with `time.Now().Year()`, a year time-bomb).

### Next

- **Review + merge PR #37** (`feat/ws-b-slice3-batch-auto-log-append` → master, 7 commits, suite 15/15 green).
- **WS-B slice 4 (`apply` → log-append):** keep the log-writing half, delete the workbook-write half — mirror slice 3's failure-honesty + pre-flight pattern.
- Then WS-D (retire bare-name fallback, T-09) → WS-E (delete dead insert code). PR #36 follow-ups still open: T-14 (model accuracy+speed benchmark), T-19 (enum escape hatch).

### Gotchas

- q3 is slow (~12 s/classify): the full `-tags=acceptance` batch-auto + type-routing group took **864 s** (15/15 green). Run target tests in groups with `-timeout 30m`, not the 600 s default.
- **`FeedbackLoggedForInsertedRows` was NOT a clean survivor** — non-dry-run, no taxonomy config, pre-T-13 expected log — and `RequireWorkbook` was SKIPPING it (test workbook absent), which hid the breakage through the entire session-42 sweep. Migrated + renamed `…ForAppendedRows`. Lesson: a "green" batch-auto test may be a SKIP or a `--dry-run` that never exercises the append path — check the fixture's `extra_args` + `RequireWorkbook` before trusting coverage.
- `feedback.AppendExpense` has **no dedup** on the hash ID → a re-run after a partial failure double-appends (new task T-20).
