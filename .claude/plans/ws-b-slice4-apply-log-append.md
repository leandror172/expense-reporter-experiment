# WS-B Slice 4 — `apply` → log-append

**Status:** PLANNED (decisions locked + advisor-reviewed, session 44) · **Not yet implemented**
**Advisor review:** `scratchpad/advisor-ws-b-slice4-2026-06-30.md` — 4 findings folded in: #1 dry-run leak
in the unconditional found+corrected `feedback.Append` (BLOCKING, now gated + `TestApply_DryRunWritesNothing`);
#2 the write-order flip *moves* the idempotency hole (pre-flight BOTH log paths; §3 overclaim corrected to
best-effort); #3 `DD/MM`→`DD/MM/YYYY` is a real on-disk change (intended, generator accepts it); #4 nil-deref
guard on `Reviewed==nil` → route to `failed`.
**Pre-B advisor review:** `scratchpad/advisor-ws-b-slice4-coreB-2026-06-30.md` — the both-path pre-flight must be
**non-destructive** on the expense log (`MkdirAll` + open-only-if-exists, NO `O_CREATE`); an `O_CREATE` probe
creates an empty `expenses_log.jsonl` and flips found-only `TestApply_IdempotencyAndFeedback`
(`ExpenseLogNotCreated`) red. Also: the `"no expense-log change"` summary string is a task-4 green-gate → land
it in B (task 3), not task-5 polish; dry-run skips the pre-flight first.
**Parent pivot:** `.claude/plans/retire-insertion-keep-generation.md` (retire workbook insertion,
keep only generation; JSONL logs become the single source of truth, `generate-workbook` the only writer)
**Predecessors:** Slice 1 (`add`) DONE · Slice 2 (`auto`) DONE · Slice 3 (`batch-auto`) DONE (PR #37 merged) · T-13 DONE
**Successors:** WS-D (retire bare-name fallback, T-09) → WS-E (delete dead insert code)
**Sibling reference:** `.claude/plans/ws-b-slice3-batch-auto-log-append.md` — slice 4 mirrors its
failure-honesty + pre-flight pattern, but is **smaller** (no rollover, no installment-string parsing).

---

## 1. Goal

Convert `apply` from "ingest `reviewed.json` → **insert new rows into the workbook** + write feedback +
expense logs" to "ingest `reviewed.json` → **append new rows to `expenses_log.jsonl`** + write feedback".
No workbook writes. `generate-workbook` becomes the sole workbook writer across every classifier-fed command.

The bulk of this slice is **deletion**. `apply` already writes both logs today; slice 4 deletes the
workbook-insert half, routes new rows through the single writer `appender.ExpandAndAppend`, and fixes one
ordering bug that the log-as-source-of-truth model exposes (§2).

---

## 2. The one piece of real logic — write-order flip + failure honesty

`apply` already appends to `expenses_log.jsonl` (`writeFeedbackForNewRows`, apply.go), but in the **wrong
order** for the new world:

```go
// TODAY (apply.go writeFeedbackForNewRows):
feedback.Append(classifPath, fbEntry)        // 1. feedback (training/audit) FIRST
feedback.AppendExpense(expensesLogPath, ...)  // 2. expense log (now the durable artifact) SECOND
```

Once the expense log is the single source of truth, this order is a latent data-loss bug: if step 1
succeeds and step 2 fails, the **feedback log records the row as done** — so a re-run finds it via
`FindLatestEntry` (apply's idempotency key, see §3) and **skips it**, and the expense-log line is lost
forever even though the user confirmed it.

**Slice 4 flips the order and adds failure honesty, mirroring slice-3's `appendClassified`:**

1. **Expense log FIRST** (`appender.ExpandAndAppend`, count=1 — see §2a) — the durable write.
2. **Feedback SECOND** (`feedback.Append`, the existing `buildFeedbackEntry`) — secondary; a failure here
   is non-fatal (warn), exactly as batch-auto's `logConfirmedFeedbackForRow` treats it.
3. **On expense-log append failure: downgrade the row** — do **not** write feedback for it, count it in a
   `failed` bucket, and **exit non-zero**. Because feedback was never written, a re-run re-attempts the row
   (still "new").

This is the load-bearing change. Everything else is mechanical deletion.

> **The flip MOVES the idempotency hole; it does not close it (advisor #2).** apply uses
> *classifications.jsonl* as the dedup index for *expenses_log.jsonl*. Trace the *other* partial failure:
> log append succeeds, then `feedback.Append` **fails** (classifications dir unwritable). Feedback failure is
> secondary/warn and `failed` counts only *log* failures → **exit 0**. A re-run then finds no feedback entry
> → treats the row as new → **appends the log line a second time** (T-20 no-dedup catches nothing). So the
> direction is right (never silently lose a durable line > avoid a visible dup), but "correct idempotency" is
> too strong. Two mitigations, both in §5: **pre-flight BOTH log paths** (makes the common
> classifications-unwritable case unreachable), and state the residual honestly (a mid-run feedback failure —
> disk full between rows — can still dup; true idempotency awaits T-20 dedup-on-ID in the expense log).

### 2a. Installments — count=1 is the only faithful port (traced, session 44)

`apply` **cannot** expand installments and never could: the installment *count* is discarded three layers
upstream and is absent from the `reviewed.json` contract.

```
batch-auto ──"99,90/3"──► review.csv (RawValue preserves notation) ✓
review.ReadQueue (queue.go:63): perInstallment, _, _ := ParseCurrencyWithInstallments(...)
                                                  ▲ count discarded into `_`
review.html export: does NOT serialize rawValue (grep empty)
reviewed.json:  "value": 33.30   (single float, per-installment, no count)
apply ReviewedEntry.Value float64  ← no count field exists
```

The old workbook path also inserted **one row** at the per-installment value. So
`appender.ExpandAndAppend(..., installmentCount: 1, ...)` reproduces existing behaviour exactly, costs
nothing (short-circuits to `buildEntry`+`AppendExpense`), and keeps `apply` on the single writer so it
benefits for free if the count is ever threaded through.

> **Pre-existing bug surfaced, NOT fixed here → new deferred task T-21.** An installment expense routed
> through *manual review* is under-recorded: batch-auto's AUTO path expands `99,90/3` into 3 dated rows
> (33,30 each); the review→apply path records a **single 33,30 row** — two-thirds of the expense vanishes.
> Orthogonal to the log-append pivot; the fix is a 4-layer change (`review.ReadQueue` → HTML export JS →
> `reviewed.json` contract → `ReviewedEntry`) with its own UX design question (show 3 rows or 1 with a ×3
> badge?). Record as **T-21**, decide later.

---

## 3. What stays (the cross-file dedup structure — preserve, but it's best-effort)

`apply` has a cross-file dedup that batch-auto lacks; slice 4 **preserves** it, but it is **best-effort, not
a guarantee** (see the §2 advisor note — a feedback-write failure makes the index unreliable):

- `processEntries` / `handleActiveEntry` keep their structure. Each confirmed/corrected entry is looked up
  in `classifications.jsonl` via `feedback.FindLatestEntry(classifPath, entry.ID)`.
  - **Not found → new row:** append to expense log + write confirmed/corrected feedback (the §2 path).
  - **Found → already applied:** corrected-feedback-only (no expense-log append); confirmed+found is a
    no-op. The feedback entry written on the *first* apply is what makes a *second* apply see the row as
    found → no re-append **on the happy path**. This is apply's built-in dedup — better than batch-auto's
    (which has none, T-20), but it relies on the feedback line having landed, which §2 shows is not
    guaranteed under partial failure. True idempotency awaits T-20 (dedup-on-ID in the expense log itself).
- `pending` / `skipped` routing unchanged.

> **Flag (not slice-4-blocking):** the join key apply's idempotency rests on is documented *inconsistently* —
> `feedback/KNOWLEDGE.md` says `GenerateID` hashes `item|date|value` in one place and
> `sha256(item+date+value+subcategory)` in another. Slice 4 **preserves the existing `FindLatestEntry(...,
> entry.ID)` lookup** (reviewed.json's `id` was minted by the producer, so the match holds regardless), so
> this doesn't block us — but verify the real `GenerateID` body before any future dedup work (T-20), since a
> subcategory-inclusive key changes what "the same expense" means.

> **Out of scope (flag only):** a found+**corrected** row writes corrected feedback but does not amend the
> existing expense-log line — an append-only log can't express an update/delete of an already-logged
> expense. So `generate-workbook` would still route the original category for a post-hoc correction. This
> is the broader "mutable corrections in an append-only log" design question (couples to T-20); slice 4
> preserves current behaviour and does not attempt it.

---

## 4. Locked decisions

1. **Remove the workbook CLI surface (confirmed by user, session 44):** delete the `--workbook` and
   `--backup` flags and the `uninsertable` concept outright (not deprecate) — they are CLI surface, not
   exported API. A user-visible flag change, accepted.
2. **`--dry-run` semantics (mirror slice 3):** keep the flag, redefine it. No workbook to skip, so
   `--dry-run` = parse + print what **would** append, write nothing to either log. Skips the pre-flight.
   **⚠ Existing leak to fix (advisor #1):** today `dryRun` is NOT threaded into `processEntries` /
   `handleActiveEntry`, and the found+corrected branch calls `feedback.Append(classifPath, corrEntry)`
   **unconditionally** — *before* the dry-run-gated insert block. So `apply --dry-run` on a reviewed.json with
   an already-found corrected entry (e.g. `apply-basic` entry 2, Diarista) **writes a corrected feedback line
   today**. Slice 4 must thread `dryRun` through `processEntries`→`handleActiveEntry` and gate that write.
   Guarded by a new acceptance test (§6, `TestApply_DryRunWritesNothing`) — without it the leak survives.
3. **`--year` stays** — `apply` keeps explicit `--year` for `DD/MM` date parsing
   (`utils.ParseDateWithYear`), which is better than batch-auto's `ParseDateFlexible`/`Now().Year()`.
   Keep it.
4. **Installments → count=1** (§2a). New deferred task **T-21** for the review-path under-recording bug.
5. **Deletion only — nothing to deprecate.** The excel batch-insert helpers `apply` calls
   (`FindSubcategoryRowBatch`, `AllocateEmptyRows`, `WriteBatchExpenses`, `ExpenseWithLocation`) are
   **shared with `internal/workflow`** (live under plain `batch`) — grep-verified. Slice 4 removes only
   `apply`'s *private* wrappers + the call sites; the shared helpers stay live and untouched. (No slice-3
   style `// Deprecated:` needed — `apply` owns nothing exported here.)

---

## 5. Code changes (`cmd/expense-reporter/cmd/apply.go`)

### Delete (workbook-write half — all apply-private)
- `insertNewRows`, `buildSubcatRequests`, `buildEmptyRowRequests`, `buildExpenseBatch` — the whole
  excel-allocation pipeline.
- In `runApply`: the `workbookPath` resolution, `excel.ValidateWorkbook`, the `applyBackup` /
  `batch.NewBackupManager().CreateBackup` block, and the `len(newRows) > 0` workbook branch.
- `--workbook` (`applyWorkbook`) + `--backup` (`applyBackup`) flags and their `init()` registrations.
- The `uninsertable` bucket and all its `printSummary` plumbing ("subcategory not found or no empty slot"
  is meaningless without a workbook lookup; the user already chose a valid taxonomy path in review).
- Unused imports after the cut: `excel`, `batch`, `models`, `filepath` (verify with `goimports`).

### Add / change
- New `appendNewRows(newRows []apply.ReviewedEntry, classifPath, expensesLogPath string, year int, dryRun bool) (appendedConfirmed, appendedCorrected int, failed []apply.ReviewedEntry, err error)`
  replacing `insertNewRows` + `writeFeedbackForNewRows`. Per new row:
  - **nil guard (advisor #4):** if `entry.Reviewed == nil` (malformed reviewed.json), route to `failed` —
    do **not** dereference `entry.Reviewed.Type`. Pre-existing exposure (old `buildSubcatRequests` panicked
    too); the new `failed` bucket is its clean home. One defensive guard, free with the rewrite.
  - `utils.ParseDateWithYear(entry.Date, year)` → `time.Time` (downgrade row on parse error).
  - **expense log first:** `appender.ExpandAndAppend(expensesLogPath, entry.Item, date, entry.Value, 1, entry.Reviewed.Type, entry.Reviewed.Category, entry.Reviewed.Subcategory)`.
  - on append error → append to `failed`, **skip feedback**, continue (§2 failure honesty).
  - **feedback second:** reuse `buildFeedbackEntry(entry)` → `feedback.Append` (a failure here is
    non-fatal/warn, secondary — mirror `logConfirmedFeedbackForRow`).
  - `dryRun` → count what would append, write nothing.
  - return non-nil `err` (wrapped) when `len(failed) > 0` so `runApply` exits non-zero.
- **Date format is a real on-disk change, not just a fixture concern (advisor #3):** the old path wrote
  `entry.Date` raw (`DD/MM`); `ExpandAndAppend` reformats to `DD/MM/YYYY`. **Intended** — the generator
  accepts it (T-11 made `taxonomy.parseDate` take `DD/MM/YYYY`), so no downstream break. (Fixture stability
  consequence is in §6.)
- `runApply`: drop the workbook branch; call `appendNewRows`; thread its `failed` into the summary; return
  its error.
- **`handleActiveEntry` is NOT unchanged (advisor #1):** thread `dryRun` through
  `processEntries`→`handleActiveEntry` and **gate the found+corrected `feedback.Append`** behind `!dryRun`.
  Today that write is unconditional and runs before any dry-run check — the existing leak §4.2 names. Keep
  its `expensesLogPath` blindness (corrected feedback only, no expense-log append, §3); the only behavioural
  change is the dry-run gate.

### Pre-flight (non-dry-run only — BOTH log paths, NON-DESTRUCTIVE — advisor #2 + coreB-advisor)
- **Dry-run skips the pre-flight, as the very first thing in `runApply`** — so the only behaviour
  `TestApply_DryRunWritesNothing` exercises is the found+corrected `feedback.Append` gate, not a probe.
- Before `processEntries`, resolve **both** `cfg.ExpensesLogFilePath()` **and** `cfg.ClassificationsFilePath()`
  and fail fast (with a `Hint:`) if either is **not appendable**.
- **⚠ DO NOT reuse slice-3's `verifyAppendable` for the expense log — it `O_CREATE`s, and that breaks a
  green test (coreB-advisor).** `TestApply_IdempotencyAndFeedback` is found-only (zero new rows → the
  expense log is never written) and asserts `verify.ExpenseLogNotCreated()`. An `O_CREATE` probe would
  create an empty `WorkDir/expenses_log.jsonl` before any row is processed → file exists → test flips red at
  integration, with a misleading "file exists" failure that points away from the pre-flight. apply is the
  **first caller that pre-flights a log it then never writes** (batch-auto always appends, so it never hit
  this).
  - **Expense-log probe = non-destructive:** `MkdirAll(dir)` **only**, plus — *only if the file already
    exists* — open `O_APPEND|O_WRONLY` (**no `O_CREATE`**) to catch a read-only existing file. Never
    create-on-probe.
  - This still satisfies **both** unwritable tests: their blocker is a *parent that is a regular file*, so
    `MkdirAll(dir)` errors `"not a directory"` before any open — they don't depend on `O_CREATE`. Confirm the
    wrapped error carries `"Hint:"` on **both** paths.
  - **Classifications probe** can keep `O_CREATE` harmlessly: in `IdempotencyAndFeedback` that file is
    seeded/exists and append-open never truncates, so `ClassificationsMatch` still holds.
  - **"Unconfigured" handling:** only hard-require the **expense-log** path when there are **new rows to
    append** (an all-found reviewed.json legitimately needs no expense log). The type-routing-cycle comment
    says `withFeedbackConfig` sets both paths, so the empty-file variant is the live risk — but design so
    neither an unconfigured nor an empty-file path bites `commandSucceeded()`/`ExpenseLogNotCreated()`.
  - **Why both (unlike batch-auto's log-only pre-flight):** apply depends on `classifications.jsonl` as its
    cross-file dedup index (§3). Pre-flighting it makes the common "classifications dir unwritable" case —
    the §2 duplicate-on-re-run trigger — unreachable before any write. Residual: a *mid-run* feedback failure
    (disk full between rows), rare; name it, don't chase it (T-20 closes it).
  - **Extraction:** the non-destructive expense-log probe now *differs* from `batch_auto.go`'s `verifyAppendable`,
    so don't blindly share that function. A small apply-local `preflightLogPaths` is cleaner than retrofitting
    the slice-3 helper with a create/no-create flag — decide at implementation time.

### `printSummary` — the corrected-section string is CONTRACT, land it in B (coreB-advisor)
- Drop the `uninsertable` and "workbook not updated" lines. Rename `Inserted:` → `Appended:`. Add a
  `Failed:` line (downgraded rows).
- **The corrected-entries section MUST contain the substring `"no expense-log change"`** — A's
  `TestApply_IdempotencyAndFeedback` asserts it (A's chosen full line: `"⚠  %d already-applied rows were
  corrected — feedback logged, no expense-log change:\n"`). This is a green-gate for task 4, **not** task-5
  polish: emit it in B (task 3). Other wording can drift; that substring cannot.

---

## 6. Acceptance + unit test plan

### `test/apply_test.go` — `TestApply_IdempotencyAndFeedback` (migrate)
- Today: `Given expensesAutoInsertedBeforeReview` seeds `classifications.jsonl` so all entries are
  *found*; asserts `noNewExpensesInserted` (`verify.ExpenseLogNotCreated`), corrections logged, and
  `OutputContains("workbook not updated")`.
- Change: the `"workbook not updated"` string is gone. Update `summaryMentionsCorrections` to assert the
  new corrected-only message. `ExpenseLogNotCreated` + `ClassificationsMatch` **still hold** (found rows
  take the corrected-feedback-only branch — no expense-log append). Net: a small string update; the test's
  idempotency proof is unchanged.

### `test/type_routing_cycle_test.go` — `TestTypeRoutingCycle_3_ApplyBackfillsType` (simplify)
- Today: `expenseTypedDuringBrowserReview` calls `buildSkeletonWorkbook` (runs `generate-workbook`) because
  "apply's new-row insert needs a workbook."
- Change: **delete the `buildSkeletonWorkbook` call** — `apply` needs no workbook. The `Given` drops to
  feedback config (`withFeedbackConfig` + the expense-log path). `expenseLogMatchesExpected` stays and gets
  **more deterministic** (no generate-workbook skeleton dependency in the Given). This is the canonical
  "new confirmed row → typed log line" append proof for `apply`.

### New unit — `TestAppendNewRows_DowngradesRowOnAppendFailure` (`cmd/.../apply_test.go` or sibling)
- Mirror slice-3's `TestAppendClassified_DowngradesRowOnAppendFailure`: force an `ExpandAndAppend` failure
  (unwritable log path) and assert the row is downgraded into `failed`, no feedback written for it, and
  `appendNewRows` returns an error → non-zero exit. Deterministic, no Ollama. The pre-flight makes an
  acceptance-level append failure unreachable, so this guards §2 at the unit level (the slice-3 lesson).

### New acceptance — `TestApply_DryRunWritesNothing` (advisor #1, the leak guard)
- Slice-3 analog: `TestBatchAuto_DryRunNoFeedbackLogged`. `Given` seeds `classifications.jsonl` with a
  prior entry so `apply-basic` entry 2 (Diarista, corrected) hits the found+corrected branch; run
  `apply --dry-run`; assert **both** logs are byte-unchanged — `verify.ClassificationsMatch(<the seed>)`
  (the corrected line must NOT be appended) and `verify.ExpenseLogNotCreated()`. **RED before the fix**
  (the unconditional `feedback.Append` writes the corrected line today), green after the dry-run gate.
  Without this test the leak silently survives the rewrite.

### New acceptance — `TestApply_UnwritableLogPath_FailsFast` (`test/apply_test.go`)
- Mirror `TestBatchAuto_UnwritableLogPath_FailsFastBeforeClassification`: an unconfigured/unwritable
  expense-log path fails fast before processing. Deterministic, no Ollama. Assert `verify.OutputContains("Hint:")`.
  **Add a sibling case for the classifications path** (advisor #2 pre-flight-both): an unwritable
  `classifications_path` also fails fast — proves the dedup-index pre-flight, not just the log pre-flight.

### New unit — `TestAppendNewRows_MalformedReviewedRoutedToFailed` (advisor #4, nil guard)
- A confirmed/corrected entry with `Reviewed == nil` is routed to `failed` (not a panic) and yields a
  non-zero exit. Deterministic, no Ollama. Cheap regression pin for the §5 nil guard.

> **`-tags=acceptance` after the contract change** — `apply`'s tests are build-tag-hidden from
> `go test ./...` (the session-42 lesson, [[feedback_rename_json_tag_acceptance]]). Run `./run-acceptance.sh`.

### Test conventions to honour (from `test/KNOWLEDGE.md`)
- **Verifiers already exist — no new happy-path verifier needed** (unlike slice 3): `verify.ExpenseLogMatches`
  (skips `id`/`timestamp`; for apply, dates are deterministic so pin `{item,date,value,type,subcategory,category}`)
  and `verify.ClassificationsMatch`. Only the unwritable-path acceptance test may need an existing
  `verify.OutputContains("Hint:")`.
- **Composable-`Then` rule:** `Then:` blocks contain only named `then*`/result-named helpers, never raw
  `verify.*`. Wrap the two new tests' assertions in named helpers (e.g. `appendFailureDowngradesRow…`,
  `unwritableLogFailsFastWithHint`). Use the shared `commandSucceeded()` base via `slices.Concat`.
- **PR #35 naming sweep (do it while we're in these files):** `test/KNOWLEDGE.md` lists `apply_test.go` and
  `type_routing_cycle_test.go` as **pending** — they still call the mechanism-named `expenseLogMatchesExpected(`
  / `classificationsMatchExpected(`. Rename the **scenario-varying** call sites to outcome names
  (e.g. `confirmedReviewRowRecordedAsTypedLogLine(fixDir)`), leave generic invariants (`commandSucceeded`)
  alone. Folds the sweep into work already touching these files instead of a separate pass.

### Fixture date-stability gotcha (the slice-3 year time-bomb, applied to apply)
- `appender.ExpandAndAppend` writes `date` as `DD/MM/YYYY`. `apply`'s `--year` defaults to
  `time.Now().Year()`, so a bare-`DD/MM` reviewed entry resolves to the *current* year → a moving
  `expected-expenses_log.jsonl`. **`apply-basic/reviewed.json` currently uses bare `"15/04"`/`"10/05"`.**
  For any test that asserts the appended log, either pass an explicit `--year` via the action/`extra_args`
  or use `DD/MM/YYYY` dates in `reviewed.json`, and pin the matching year in the expected log. (The
  type-routing-cycle reviewed.json dates need the same check before asserting `expenseLogMatchesExpected`.)

---

## 7. Implementation order (commit per green step — TDD cadence)

1. **Convert the append path + delete the workbook half.** New `appendNewRows` (with the §2 write-order
   flip + failure downgrade + **nil guard #4**); thread `dryRun` through `processEntries`→`handleActiveEntry`
   and **gate the found+corrected feedback write (#1)**; add the non-dry-run **BOTH-path pre-flight (#2)**;
   delete `insertNewRows` & friends, the `--workbook`/`--backup` flags, the `uninsertable` plumbing; fix
   imports. Add the **downgrade unit test** + **nil-guard unit test** (RED→green). `go build ./... &&
   go vet ./... && go test ./...` green. Commit.
2. **Rewire acceptance tests** (`-tags=acceptance`): migrate `TestApply_IdempotencyAndFeedback` (string),
   simplify `TestTypeRoutingCycle_3` (drop `buildSkeletonWorkbook`), add `TestApply_DryRunWritesNothing`
   (#1 leak guard — should go RED first if written before step 1's gate) + `TestApply_UnwritableLogPath`
   (both-path variant, #2). `./run-acceptance.sh` green (raise `-timeout`; the type-routing-cycle group
   pulls Ollama elsewhere — apply's own step is deterministic). Commit.
3. **Summary polish + cleanup:** `printSummary` wording (`Appended`/`Failed`, drop workbook lines),
   any leftover "insert" vocabulary. Build/vet/test green. Commit.

---

## 8. Process notes (project rules)

- **Local-model-first:** route `appendNewRows` codegen through Ollama (`my-go-qcoder` → `my-go-q25c14`
  fallback); record a 2/1/0 verdict. Strong in-repo template: slice-3 `appendClassified` + slice-2
  `auto.go`'s `appender.ExpandAndAppend` call.
- **advisor() before implementation** (CLAUDE.md rule 5): contract-changing, multi-test-rippling — ask
  first ("I'd like to call advisor()"), then call.
- **Out of scope / do NOT touch:** plain `batch` + the shared excel/insert machinery it uses; WS-D / WS-E;
  T-19 escape-hatch; T-20 dedup; **T-21** review-path installment under-recording (new, this session);
  the append-only-corrections gap (§3 note).

## 9. Tasks to record at handoff (tasks.md is handoff-only — [[feedback_tasks_md_handoff_only]])

- **T-21 (new):** Installment expansion lost on the review→apply path — manual-review installment expenses
  under-recorded as a single per-installment row (§2a). Orthogonal to the pivot; UX design question on fix.
- Slice 4 itself: mark WS-B slice 4 DONE; WS-B fully complete → next is WS-D (T-09) then WS-E.
