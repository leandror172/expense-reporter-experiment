# WS-B Slice 3 — `batch-auto` → log-append

**Status:** PLANNED (decisions locked + advisor-reviewed, session 43) · **Not yet implemented**
**Advisor review:** `scratchpad/advisor-ws-b-slice3-2026-06-30.md` — 3 findings folded in (source-verified):
inventory was missing `type_routing_cycle_test.go`; append-failure must downgrade the row (don't delete the
failure tests — repurpose); the CSV survivors run `--dry-run` so they don't cover the append path.
**Parent pivot:** `.claude/plans/retire-insertion-keep-generation.md` (retire workbook insertion,
keep only generation; JSONL logs become the single source of truth, `generate-workbook` the only writer)
**Predecessors:** Slice 1 (`add`) DONE · Slice 2 (`auto`) DONE · T-13 (full-path classifier) DONE
**Successors:** Slice 4 (`apply` → log-append) → WS-D (retire bare-name fallback) → WS-E (delete dead insert code)

---

## 1. Goal

Convert `batch-auto` from "classify → insert auto-rows into the workbook (+ produce CSVs)" to
"classify → **append auto-rows to `expenses_log.jsonl`** (+ produce CSVs)". No workbook writes.

T-13 already did the hard part: `classifyLines` (`batch_auto.go:214`) captures `top.Type` from the
predicted full path, so every appended entry carries the typed `type/category/subcategory` path and
routes via `byPath`. Slice 3 is mostly **deletion + one behaviour shift**, mirroring the slice-2 `auto`
conversion (`appender.ExpandAndAppend`).

### The behaviour shift (the one piece of real logic)

Today the responsibility is split asymmetrically:
- **workbook** gets *expanded* installments (via `workflow.InsertBatchExpensesFromClassified`)
- **`expenses_log.jsonl`** gets *one un-expanded line per row* (`logExpense` at `batch_auto.go:300`
  writes raw `r.Item` + per-installment value, no `(1/3)` expansion)

When the log becomes the source of truth, that flips: the **log** must now carry the expanded lines.
`appender.ExpandAndAppend` already produces them (N dated `ExpenseEntry` lines; cross-year installments
carry their real next-year date — no rollover concept).

---

## 2. Locked decisions

1. **`--dry-run` semantics (decision 1 = b):** keep the flag, redefine it. Under log-append there is no
   workbook to skip, so `--dry-run` = **classify + write `classified.csv`/`review.csv`, but do NOT append
   to `expenses_log.jsonl` and do NOT write `classifications.jsonl` feedback.** A preview-without-committing
   mode, valuable now that the log is authoritative and append-only.
   *Already validated by `TestBatchAuto_DryRunNoFeedbackLogged` (feedback_test.go:51) — stays meaningful.*

2. **Acceptance tests (decision 2, revised by advisor):** the log is now the *only* persistence, so the
   two workbook-failure tests are **repurposed to log-append failure, not deleted** — their *value*
   (downgrade the row + preserve CSVs + non-zero exit when the durable write fails; fail fast before
   spending ~12 s/row on Ollama) migrates to the log world. Full bucketing of **13** tests (the inventory
   missed `type_routing_cycle_test.go`) in §4.

3. **Cleanup (decision 3):** delete orphaned `logExpense`; rename the `inserted` misnomers to `appended`/`logged`.

4. **Deprecate, don't delete (the workbook-insert code):** Go's equivalent of Java `@Deprecated` is the
   `// Deprecated:` doc-comment convention (recognised by gopls / `go doc` / staticcheck SA1019 — call
   sites get struck through + linted). **Scope is narrow:** the grep shows only
   `workflow.InsertBatchExpensesFromClassified` is orphaned by this slice. `workflow.InsertBatchExpenses`,
   `internal/batch/rollover_writer.go`, and the `excel` writers are **still live under the plain `batch`
   command** (`processor.go:156`), which is NOT in the WS-B pivot. So:
   - **Slice 3:** add `// Deprecated:` to `InsertBatchExpensesFromClassified` only, pointing at
     `appender.ExpandAndAppend` + WS-E. Keep it + `workflow_test.go` compiling/green.
   - **WS-E (already tracked):** hard-delete it once generation-only is validated end-to-end on real data.
   - Do **not** deprecate the still-live shared machinery — that would mislead.

---

## 3. Code changes (`cmd/expense-reporter/cmd/batch_auto.go`)

### Delete
- `insertClassified()` (:220–267) — workbook path resolution, `excel.ValidateWorkbook`,
  `batch.NewBackupManager().CreateBackup`, `workflow.InsertBatchExpensesFromClassified`, the
  `models.ClassifiedExpense` build, the per-row insert-error feedback loop.
- The pre-flight workbook checks in `runBatchAuto` (:103–111).
- `rollover.csv`: `writeRolloverCSV` (:424), the `rollovers`/`rolloverPath` plumbing in `runBatchAuto`
  (:115–137), and the rollover line in `printBatchSummary` (:324–326).

### Change
- `runBatchAuto` (:77): drop the `!batchAutoDryRun` workbook branch; after `classifyLines`, when **not**
  dry-run, append auto-rows to the log via a new `appendClassified(results, appCfg, batchAutoModel)`
  helper (replaces `insertClassified` + `logBatchFeedback`). Still always write `classified.csv` /
  `review.csv`.
- New `appendClassified` (folds in `logBatchFeedback`'s surviving half): for each `AutoInserted && Error==nil`
  row — parse `installmentCount` from `RawValue` (`utils.ParseCurrencyWithInstallments`), `ParseDateFlexible(r.Date)`
  → `time.Time`, then `appender.ExpandAndAppend(logPath, r.Item, parsedDate, perInstallment, installmentCount,
  r.Type, r.Category, r.Subcategory)`. **Keep** the `feedback.Append` confirmed-entry write to
  `classifications.jsonl` (`NewConfirmedEntry`). Both skipped under `--dry-run`.
- **CRITICAL — failure honesty (advisor finding 2).** The log is now the only persistence, so an append
  error must NOT be a silent non-fatal warning the way `auto.go`/`logExpense` treat it. `appendClassified`
  **must mirror the old `insertClassified:260` behaviour**: on `ExpandAndAppend` error, set
  `results[idx].AutoInserted = false` and record the error, so (a) `printBatchSummary` doesn't count an
  un-persisted row as "appended", and (b) `runBatchAuto` returns a non-zero exit (preserve the existing
  `"... CSVs preserved at %s"` wrapped-error pattern from :140-142). The confirmed feedback for that row is
  then also skipped (its `AutoInserted` is now false). Without this, a per-row append failure silently
  over-counts and exits 0 — a correctness regression, not a cosmetic one.
- **Log-path pre-flight (advisor finding 2, replaces the workbook pre-flight).** In `runBatchAuto`,
  non-dry-run only, before `classifyLines`: resolve `appCfg.ExpensesLogFilePath()` and fail fast if it is
  unconfigured or its directory is unwritable — so we don't spend ~12 s/row × N on Ollama and *then*
  discover the log can't be written. Keep a `Hint:` in the error (the harness asserts on `"Hint:"`).
- `printBatchSummary`: rename `Auto-inserted` → `Auto-appended` (and drop the rollover line).
- The per-row status line in `classifyLines` (:204) prints `AUTO  ` — keep the label or rename to
  `APPEND`; update the `installmentExpenseAutoInserted` verifier's `OutputContains` string to match
  whatever is chosen. **Decision: rename summary + keep the rest minimal; update the one verifier string.**

### Cleanup (decision 3)
- Delete `logExpense` (`auto.go:181`) — zero callers after this slice (confirmed by grep: only
  `auto.go` defines it and `batch_auto.go:300` calls it). Unexported → compiler won't catch it,
  staticcheck will.
- `auto.go:162` `✓ Inserted:` → `✓ Appended:` (slice-2 loose end; do it here for consistency). Note the
  ambiguous-path test in `test/auto_test.go` keys on `"✓ Inserted"` — update both together.

### Deprecate (decision 4)
- `internal/workflow/workflow.go` `InsertBatchExpensesFromClassified`: prepend
  `// Deprecated: superseded by appender.ExpandAndAppend (WS-B log-append pivot). Scheduled for removal in WS-E.`

---

## 4. Acceptance test plan (13 batch-auto tests)

> **Source-verified dry-run status (advisor finding 3).** A fixture runs `--dry-run` only if its
> `config.json` lists it in `extra_args`. Confirmed: `batch-auto-basic`, `batch-auto-exclusions`,
> `type-routing-cycle` → **dry-run**. `batch-auto-installments` (`extra_args: []`, threshold 0.0) and
> `batch-auto-typed` → **non-dry-run** (real append path). Consequence: post-slice-3 the dry-run survivors
> **pass trivially without ever appending a line**, so they prove nothing about the new behaviour. The
> append path needs an explicit non-dry-run anchor.

### Canonical append anchor (non-dry-run) — the load-bearing coverage
- **`TestBatchAuto_TypeEmittedInExpenseLog`** (typed_log_test.go): runs non-dry-run, provides a workbook,
  and **already asserts `expenseLogMatchesExpected`** — it exercises the log write today (via the insert
  path's `logExpense`). Slice 3: **drop `RequireWorkbook` + `CopyWorkbookToWorkDir`** (no workbook needed);
  it then directly validates `appender.ExpandAndAppend` output and becomes the canonical "HIGH row →
  correct log line" test, mirroring slice-2's HIGH-append test. Check the expected file: no-installment
  input → expected stays valid; with installments → regenerate for the expanded lines.

### Survive ~as-is (dry-run or CSV/feedback-only) — 7
`Basic`, `MixedConfidence`, `ExcludedCategoriesGoToReview`, `ClassificationAccuracy`, `OutputDirFlag`
(batch_auto_test.go); `FeedbackLoggedForInsertedRows`, `DryRunNoFeedbackLogged` (feedback_test.go).
- Most are dry-run → do **not** cover the append path. Verify `FeedbackLoggedForInsertedRows` isn't dry-run
  (if it is, under (b) it would assert *no* feedback — make sure it isn't silently inverted). Cosmetic:
  rename its "Inserted".

### Survive but take an explicit position (advisor finding 1) — 1
- **`TestTypeRoutingCycle_1_BatchAutoEmitsType`** (type_routing_cycle_test.go) — the inventory's original
  `grep 'TestBatchAuto'` structurally **missed** this (`_1_BatchAuto` ≠ `TestBatchAuto`). It's the e2e test
  of the whole pivot (batch-auto→review→apply→generate-workbook). **Position (source-verified):** its
  batch-auto step runs **`--dry-run`** (per `type-routing-cycle/config.json`), asserting only the
  `classified.csv` type column — it does **not** use the insert path, and the cycle's actual log is written
  later by **`apply`** (step 4). ∴ slice 3 **leaves it unchanged.** Caveat to record: despite its name it
  does NOT cover "batch-auto writes a log generate-workbook consumes" — that gap is filled by the canonical
  append anchor, not here.

### Rewrite (1) — repurpose to the new behaviour
- **`TestBatchAuto_SameYearInstallmentsExpanded`** (batch_auto_test.go:82): today `RequireWorkbook`-gated
  (`batch-auto-installments`, non-dry-run, threshold 0.0), only asserts the summary string. Drop the
  workbook gate (`RequireWorkbook` + `CopyWorkbookToWorkDir`), point `Given` at a taxonomy config, and
  assert the **N expanded installment lines in `expenses_log.jsonl`** (the harness can read the log; it
  could not read workbook cells). New verifier if none exists (count log lines matching the installment
  items). Turns the test's old weakness into a real assertion.

### Invert (1) — the behaviour reversed
- **`TestBatchAuto_RolloverInstallmentsWrittenToFile`** (batch_auto_test.go:96): rollover.csv no longer
  exists. Rewrite to assert **no `rollover.csv`** AND next-year installments appear in the log with their
  real next-year dates (mirror slice-1 `NoRolloverFileCreated`). Rename the test + its verifier accordingly.

### Repurpose, do NOT delete (advisor finding 2) — 2
The corrupt/missing-workbook *mechanism* is gone, but the *value* migrates to the log-append world:
- **`TestBatchAuto_InsertFailure_PreservesCSVs`** (:264) → **`...LogAppendFailure_PreservesCSVs`**: force a
  log-append failure (unwritable/read-only `expenses_log.jsonl` — point it at a dir, or 0400 the file) and
  assert: CSVs still written, summary does **not** over-count (failed row downgraded), non-zero exit. This
  is the guard for the §3 failure-honesty change.
- **`TestBatchAuto_MissingWorkbook_FailsFastBeforeClassification`** (:228) →
  **`...UnwritableLogPath_FailsFastBeforeClassification`**: assert an unconfigured/unwritable log path fails
  fast *before* any classification (guards the §3 pre-flight; preserves the "don't burn 12 s/row then fail"
  property). Keep `verify.OutputContains("Hint:")`.
- Helpers: retarget `batchExpensesSubmittedToCorruptWorkbook` / `batchExpensesSubmittedToMissingWorkbook` to
  set up the unwritable log path instead of a bad workbook; keep `commandFailedWithWorkbookHint` (rename →
  `commandFailedWithHint`). `classificationCSVsPreservedDespiteInsertFailure` → retarget to the append case.

---

## 5. Implementation order (commit per green step — TDD cadence)

1. **Convert the append path + delete insert/rollover branch.** New `appendClassified` (with the
   **failure-downgrade** of §3); add the **non-dry-run log-path pre-flight**; delete
   `insertClassified`/`writeRolloverCSV`/workbook checks; deprecate `...FromClassified`. `go build ./...`,
   `go vet ./...`, `go test ./...` green. Commit.
2. **Rewire acceptance tests** (`-tags=acceptance` — hides from `go test ./...`; the session-42 lesson):
   drop the workbook gate on the **canonical append anchor** (`TypeEmittedInExpenseLog`); rewrite
   `SameYearInstallments`; invert `Rollover`; **repurpose** the two failure tests to log-append failure +
   pre-flight (not delete). `./run-acceptance.sh` green (raise `-timeout` — q3 ~12 s/classify; or run the
   batch-auto group alone). Commit.
3. **Cleanup:** delete `logExpense`, rename `inserted`→`appended` misnomers. **Keep the `auto.go:162`
   rename a SEPARATE commit** (advisor: it crosses into shipped slice-2 territory; clean revert boundary)
   — including the `test/auto_test.go` `"✓ Inserted"` string update. Build/vet/test green. Commit.

---

## 6. Process notes (project rules)

- **Local-model-first:** route the `appendClassified` codegen through Ollama (`my-go-qcoder` →
  `my-go-q25c14` fallback); record a 2/1/0 verdict. The conversion mirrors `auto.go:insertExpense`, so the
  prompt has a strong in-repo template.
- **advisor() before implementation** (CLAUDE.md rule 5): contract-changing, multi-test-rippling change —
  the kind advisor is for. Ask first ("I'd like to call advisor()"), then call.
- **`-tags=acceptance` after any contract change** — `//go:build acceptance` hid 13 regressions in PR #36.
- **Out of scope / do NOT touch:** plain `batch` command + the shared insert/rollover/excel machinery it
  still uses (`InsertBatchExpenses`, `rollover_writer.go`); WS-E deletes the deprecated `...FromClassified`
  later; T-19 escape-hatch is independent.

---

## 7. Minors + idempotency (advisor, note — don't block)

- **Dry-run summary truthfulness:** under (b) nothing is appended, so "Auto-appended : N (dry-run)" lies.
  Print **"Would append : N"** in dry-run mode (we're renaming the line anyway).
- **Idempotency / no dedup (CONFIRMED):** `feedback.AppendExpense` (expense_log.go:36) is a plain
  `O_APPEND` write with **no dedup on the hash ID** — verified by reading the body. batch-auto is the first
  *batch* user of the append path (add/auto are single-row), so a re-run after a partial failure
  **double-appends** every previously-succeeded row. This is a **slice-1/2 inheritance to flag, not fix
  here** (it affects `add`/`auto` too) — but it raises the stakes on the §3 failure-honesty work: a partial
  failure that isn't surfaced invites exactly the re-run that duplicates. Recommend a follow-up task
  (dedup-on-ID or a `--resume` semantics) before batch-auto is used on real data at volume.

## 8. Open questions for implementation time

- Exact verifier for "N expanded installment lines in the log" — likely a new `verify.ExpenseLogHasLines`
  / line-count-by-item helper; check `verify/` for an existing JSONL reader before writing one.
- The unwritable-log-path setup for the repurposed failure tests (0400 file vs. path-is-a-dir) — pick
  whichever is portable on the WSL2 filesystem; confirm the binary's error message carries `Hint:`.
