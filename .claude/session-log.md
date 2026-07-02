# Session Log — Expense Reporter

**Current Session:** 2026-06-30 — Session 44: WS-B slice 4 — apply → log-append
**Current Layer:** Log-append pivot (WS-B COMPLETE, slices 1–4) → next WS-D
Most recent entry first. Run `.claude/tools/rotate-session-log.sh` when this grows beyond ~3 sessions.

---
## 2026-06-30 - Session 44: WS-B slice 4 — apply → log-append

### Context

Started by reviewing PR #37 (WS-B slice 3, since merged); then planned and shipped WS-B slice 4 (`apply` → log-append) end-to-end via an acceptance-first Sonnet subagent for the tests and a local-model core, mostly unattended with commits between tasks.

### What Was Done

- Wrote + advisor-reviewed the slice-4 plan (`.claude/plans/ws-b-slice4-apply-log-append.md`) across TWO advisor passes: initial design, then a pre-implementation coreB pass that caught the non-destructive-pre-flight trap.
- Created `.claude/workflows/sonnet-max-subagent.js` — a single-agent Workflow harness exposing the per-agent `effort` knob the `Agent` tool lacks (max/xhigh Sonnet subagent).
- Subagent A (Sonnet, acceptance-first) wrote 5 slice-4 acceptance tests RED-first (test/ tree only); independently tree-verified (git status + rerun, confirmed RED for real behavioural reasons).
- Core B (Ollama `my-go-qcoder`, verdict 2): `appendNewRows` (log-first/feedback-second/downgrade-on-append-failure + nil-guard), `dryRun` gate on the found+corrected feedback write, non-destructive both-path pre-flight; deleted `insertNewRows`+excel-allocation pipeline, `--workbook`/`--backup`, `uninsertable`, dead `IsInsertable`/`IsAlreadyHandled`; 2 unit tests.
- Full `-tags=acceptance` suite green (`ok 1020s`, 0 fail) + full unit suite green. Opened **PR #38**.
- Updated memory (apply/feedback/test QUICK, expense-reporter QUICK+KNOWLEDGE) + README.

### Decisions Made

- **Non-destructive expense-log pre-flight** — `MkdirAll(dir)` + open only if the file already exists (NO `O_CREATE`). An `O_CREATE` probe would leave an empty `expenses_log.jsonl` and flip the found-only `TestApply_IdempotencyAndFeedback` (`ExpenseLogNotCreated`) red. apply is the first caller that pre-flights a log it may never write (batch-auto always appends).
- **Pre-flight timing falls out of the data dependency** — classifications probed BEFORE `processEntries` (its dedup index, read there); expense-log probed AFTER, only when `newRows>0`.
- **count=1 installments (T-21)** — the installment count is discarded upstream at `review.ReadQueue` and absent from `reviewed.json`, so apply cannot expand; count=1 is the faithful port. Logged the under-recording as T-21.
- **apply's cross-file idempotency is best-effort** — a feedback-write failure after a successful log append can dup on re-run (T-20); pre-flighting both paths makes the common case unreachable.
- **`Agent` tool has no effort knob** → a single-agent Workflow (`sonnet-max-subagent`) is the only way to run a max/xhigh-effort subagent.

### Next

- Review + merge **PR #38** (slice-4 branch, `feat/ws-b-slice4-apply-log-append`).
- Then **WS-D** (retire bare-name routing fallback, T-09) — resolve the **T-19** escape-hatch gap first (coupled). Then **WS-E** (delete dead insert code, narrow scope).
- Still open: **T-14** (model accuracy+speed benchmark), **T-20** (expense-log dedup), **T-21** (reviewed-installment under-recording).

### Gotchas

- The `O_CREATE` pre-flight trap above would have flipped a green test red at integration with a misleading "file exists" failure pointing away from the cause — caught by the coreB advisor pass, fixed before implementing.
- `Agent` tool exposes `model` but not `effort`; only `Workflow.agent({effort})` sets it. The new `sonnet-max-subagent` harness wraps that.
