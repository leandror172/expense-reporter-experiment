# WS-B Slice 3 Implementation Report — `batch-auto` → log-append

**Session:** 43 (2026-06-30) · **Branch:** `feat/ws-b-slice3-batch-auto-log-append` · **Base:** `master`
**Plan:** `.claude/plans/ws-b-slice3-batch-auto-log-append.md` (advisor-reviewed)

## Outcome

WS-B slice 3 is **implemented and fully green**. `batch-auto` no longer inserts into the Excel
workbook — auto-classified rows are appended to `expenses_log.jsonl` (now the single durable
persistence; `generate-workbook` is the only workbook writer). This continues the session-36 pivot
after `add` (slice 1) and `auto` (slice 2).

`go build ./...`, `go vet ./...`, `go test ./...` (554 unit tests) all pass. The full
`-tags=acceptance` batch-auto + type-routing group is **15/15 green** (864 s on q3).

## Commits (6, + this report)

1. `a263d7b` docs(plan): WS-B slice 3 — advisor-reviewed plan
2. `749ccd2` feat(batch-auto): append to expense log instead of inserting into workbook
3. `bea15da` test(batch-auto): rewire acceptance to the log-append world
4. `d5999b6` refactor(auto): remove dead logExpense helper
5. `10c6ed5` refactor(auto): rename insert vocabulary to append
6. (memory) docs(memory): record WS-B slice 3

## What changed

### `cmd/.../batch_auto.go`
- `insertClassified` + `logBatchFeedback` replaced by `appendClassified` → `appendOneRow` +
  `logConfirmedFeedbackForRow`, routing through `appender.ExpandAndAppend`. Installments expand at
  append time; cross-year installments carry their real next-year date — **`rollover.csv` retired**
  (`writeRolloverCSV` + the `RolloverExpense` plumbing deleted).
- **Failure honesty (the one piece of real logic):** because the log is the only durable
  persistence, a per-row append failure downgrades the row in place (`AutoInserted=false` + `Error`),
  so the summary count stays honest, the row falls into `review.csv`, and the command exits non-zero.
  CSVs are written **after** the append so they reflect downgrades.
- `preflightLogPath`/`verifyAppendable`: open the log in `O_APPEND` mode before classifying → fail
  fast on an unwritable log (saves ~12 s/row). Summary renamed `Auto-appended` / `Would append`.

### `internal/workflow/workflow.go`
- `InsertBatchExpensesFromClassified` marked `// Deprecated:` (superseded by `appender.ExpandAndAppend`;
  scheduled for WS-E removal). Kept + unit-tested. **The plain `batch` command keeps
  `InsertBatchExpenses` + the rollover/excel machinery LIVE** — only the classified variant orphaned.

### `cmd/.../auto.go` (separate commit, clean boundary)
- Dead `logExpense` deleted (its only caller, batch_auto, now uses the appender).
- `insertExpense` → `appendExpense`; `✓ Inserted` → `✓ Appended`; `⚠ Not inserted` → `⚠ Not appended`.

### Tests
- New unit `cmd.TestAppendClassified_DowngradesRowOnAppendFailure` — deterministic guard for the
  failure-downgrade (points the log at a missing parent dir).
- Acceptance retargeted from workbook to the log: `verify.ExpenseLogMatches`; installments fixture
  asserts the N expanded lines; rollover fixture inverted (`NoRolloverFileCreated` + next-year dates);
  the two workbook-failure tests repurposed (pre-flight fail-fast; downgrade → the unit test above).
  `TypeEmittedInExpenseLog`/`FeedbackLoggedForAppendedRows` lost their workbook gate.

## Decisions & deviations from plan

- **`--dry-run` = classify + CSVs, no append** (no workbook to skip).
- **Failure-downgrade is a UNIT test, not acceptance (deviation).** The pre-flight makes an
  acceptance-level append failure unreachable (a log that passes pre-flight won't fail at append), so
  a deterministic unit test is both possible and a stronger guard. The acceptance side keeps the
  pre-flight fail-fast test.
- **Deprecate-not-delete, NARROW scope** — only `…FromClassified` orphans; plain `batch` keeps the
  machinery live, so WS-E must not delete it with the classified variant.
- **Explicit-year fixture inputs + clean-dividing installment values** — the append path reformats
  dates to `DD/MM/YYYY` (`ParseDateFlexible` fills bare `DD/MM` with `time.Now().Year()`, a time-bomb).

## Advisor review (3 findings, all source-verified + folded in)

1. **Inventory missed `type_routing_cycle_test.go`** (`grep 'TestBatchAuto'` can't match `_1_BatchAuto`).
   Added with a position: its batch-auto step is `--dry-run` → survives unchanged. Confirmed green (4/4).
2. **Append-failure must downgrade the row** — folded into `appendClassified`; the failure tests were
   repurposed (not deleted) to the log-append world.
3. **Dry-run survivors don't cover the append path** — verified all three suspect fixtures carry
   `"extra_args": ["--dry-run"]`. Designated `TypeEmittedInExpenseLog` the canonical non-dry-run anchor.
   This caught a real latent bug: `FeedbackLoggedForInsertedRows` was non-dry-run, lacked taxonomy
   config, had a pre-T-13 expected log, and was SKIPPING via `RequireWorkbook` — hidden through the
   whole session-42 sweep. Fixed + renamed `…ForAppendedRows`.

## Local-model usage (per session policy)

- `my-go-qcoder` (primary) **unavailable** — `warm_model` returned HTTP 500 twice (VRAM contention,
  same pattern as the T-13 session). Fell back to `my-go-q25c14` per the tier list.
- `appendClassified` + pre-flight (§3): generated by `my-go-q25c14`, **verdict 1** — logic correct and
  matched the spec; I refactored the 30-line function into `appendOneRow`/`logConfirmedFeedbackForRow`
  per the single-responsibility constraint and strengthened the pre-flight to probe actual
  `O_APPEND` openability (catches a read-only file, not just a missing dir). ~1500 est. tokens saved.

## Test results

- Unit: **554 pass** (21 packages).
- Acceptance (`-tags=acceptance`, q3): **15/15** — `TestBatchAuto_*` (incl. all 5 dry-run survivors,
  the 3 rewrites, the invert, the repurposed pre-flight test, and `TypeEmitted`/`FeedbackLogged`) +
  `TestTypeRoutingCycle_1..4`. 864 s total (q3 is ~12 s/classify).

## Known follow-ups (not in scope here)

- **T-20** (new): `feedback.AppendExpense` has no dedup on the hash ID → a re-run after a partial
  failure double-appends. batch-auto is the first *batch* user of the append path, so this is now
  reachable. Add dedup-on-ID or `--resume` before real-data volume.
- **WS-B slice 4** (`apply` → log-append): delete the workbook-write half; mirror this slice's pattern.
- **WS-D** (T-09, retire bare-name fallback) → **WS-E** (delete the deprecated `…FromClassified`).
- Slice-2 loose ends (T-12): `test/auto_test.go`'s `RequireWorkbook`-gated LOW-path cases still skip;
  `internal/appender/.memories` still to add.
