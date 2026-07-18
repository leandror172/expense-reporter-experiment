# T-20 Implementation Report — `batch-auto --resume` + duplicate-append warning

**Date:** 2026-07-17 (session 60)
**Branch/PR:** `feat/t20-batch-auto-resume`
**Design provenance:** user-locked Option B (`--resume` flag + always-on warning; ledger
stays append-only by default) → Opus-xhigh advisor review (`.claude/advisor-t20-resume-dedup.md`,
verdict proceed-with-adjustments, all adjustments adopted) → implemented by an
`impl-opus-xhigh` subagent (TDD, my-go-qcoder codegen: four 2-verdicts, two 1s, zero 0s)
→ orchestrator-verified (independent build/vet/unit reruns + full acceptance suite).

<!-- ref:batch-auto-resume -->
## batch-auto --resume semantics (T-20)

**Problem:** `feedback.AppendExpense` is plain `O_APPEND`; a `batch-auto` re-run after a
partial failure double-appends every previously-succeeded row, and the JSONL log is the
single source of truth for `generate-workbook`. Identical (item,date,value) triples can be
LEGITIMATE distinct expenses (two same-day bus fares), so a global uniqueness constraint
on the log is wrong by design — the ledger never dedups silently.

**Contract:**
- **Ledger:** ONE `map[id]int` from `feedback.LoadExpenseIDCounts(expenses_log.jsonl)`
  (counts, not a set — multiplicity preserves legit duplicates). Loaded once per run
  whenever the log exists. Consumption order: classify-phase full-skips first, then
  append-phase appends/warnings. Partially-matched rows consume NOTHING in the classify
  phase. Missing log → empty ledger; malformed line → hard error (corruption surfaces).
- **Prediction:** `appender.PredictEntryIDs(item, parsedDate, perInstallment, count)`
  builds entries via the SAME `expandEntries` step `ExpandAndAppend` uses — installment
  `(i/N)` suffixes, +i-month dates, per-installment values — so predicted ids cannot
  drift from written ids. Takes parsed inputs only (advisor blocker: raw-string
  prediction would mismatch every DD/MM and installment id).
- **`--resume` (default false):** rows whose predicted ids are ALL available in the
  ledger are consumed + skipped BEFORE the LLM call (`SKIP … (already logged)`); they
  land in classified.csv with the `(already logged)` marker in the subcategory column
  (schema stays 8 columns — `review.ReadQueue` hard-errors otherwise), NOT in
  review.csv, and write no feedback entries. Rows with only SOME ids present route to
  REVIEW (never auto-completed: a divergent re-classification would silently split an
  installment series across categories). Parse failures are error rows, not skips.
- **Always-on warning (no flag):** each append whose id still has ledger count emits one
  stderr warning and consumes — so it fires for pre-existing log duplicates but NOT for
  a second in-CSV legitimate duplicate.
- **Year-boundary caveat:** bare `DD/MM` input infers the current year
  (`ParseDateFlexible`); a resume crossing Dec→Jan re-infers a different year → id
  mismatch → no skip (double-append). Documented in `--resume` help: use `DD/MM/YYYY`
  for December batches.
- **`--dry-run --resume`:** skip decisions shown, zero writes.
<!-- /ref:batch-auto-resume -->

## Files

| File | Change |
|------|--------|
| `internal/feedback/id_counts.go` (+test) | NEW — `LoadExpenseIDCounts` |
| `internal/appender/appender.go` (+test) | `expandEntries` extraction + `PredictEntryIDs`; predict==append property test |
| `cmd/expense-reporter/cmd/batch_auto_resume.go` (+test) | NEW — ledger load, skip evaluation, duplicate warning, resume-decision helpers |
| `cmd/expense-reporter/cmd/batch_auto.go` | `--resume` flag, ledger threading, skip/review routing, summary `Skipped` count, help text |
| `test/batch_auto_resume_test.go` + `test/expect/resume.go` + 5 fixture dirs | NEW acceptance scenarios A1–A5 + `ResumeSkipCount`/`DuplicateWarningCount` counters |

## Acceptance test rundown (all NEW)

- **A1 `TestBatchAutoResume_AllRowsAlreadyLogged`** (deterministic, no Ollama): every input
  row (one plain + one full 3-installment series) pre-seeded via the REAL
  `ExpandAndAppend` path → `--resume` skips all before classification; log unchanged,
  classifications.jsonl never created, SKIP markers present. Exercises the
  highest-risk id path (suffix + addMonths + date format).
- **A2 `TestBatchAutoResume_OneOfTwoDuplicatesSkipped`** (Ollama): two identical legit
  rows, log seeded with ONE copy → exactly one skips, the other proceeds (count
  consumption, the reason the ledger is a multiset).
- **A3 `TestBatchAutoResume_OnlyNewRowClassifies`** (Ollama): one seeded + one new
  gate-passing `Posto Ipiranga` row → only the new row classifies/appends; final log =
  seed + 1.
- **A4 `TestBatchAutoDuplicateWarning_FiresWithoutResume`** (Ollama): NO flag, seeded
  input row → appends AGAIN (append-only default preserved, count 2) + exactly one
  stderr duplicate warning.
- **A5 `TestBatchAutoResume_DryRunShowsSkipsWritesNoLog`** (deterministic): `--dry-run
  --resume` → SKIPs shown, zero writes.

Unit: `TestLoadExpenseIDCounts` (6 cases), `TestPredictEntryIDs_MatchesExpandAndAppend`
(the drift guard), `TestEvaluateResumeSkip` (5 cases incl. one-of-two-dups),
`TestWarnIfDuplicate`; `TestAppendClassified` + `TestBatchAutoCommand_Flags` updated for
the new ledger arg / flag.

## Deviations from the advisor contract (all endorsed in-run)

1. No persisted review-reason column — review.csv is schema-locked at 8 fields; the
   "partially logged — resolve manually" reason prints to stderr at classify time.
2. Up-front date/value parsing scoped to the `--resume` path only, preserving the
   non-resume classify-then-downgrade behavior and existing fixtures.
3. Delegation split: pure helpers + unit tests via my-go-qcoder; harness-idiom-dense
   acceptance file + loop-threading edits hand-written (prompt-cost tiebreaker).

## Verification

- Subagent: red→green TDD on all new tests; targeted acceptance run green.
- Orchestrator (independent): `go build` / `go vet` / `go vet -tags=acceptance` /
  `go test ./...` all green; `gofmt` clean on touched files (the 5 flagged
  `internal/batch/` files are pre-existing, untouched by this work).
- Full acceptance suite: 54/54 green (see PR).
