# Session Log — Expense Reporter

**Previous logs:** `.claude/archive/session-log-2026-02-27-to-2026-02-27.md`, `.claude/archive/session-log-2026-03-02-to-2026-03-02.md`
Most recent entry first. Run `.claude/tools/rotate-session-log.sh` when this grows beyond ~3 sessions.

---

## 2026-03-13 — Session 6: PR #6 merge + 5.5 design (classification feedback logging)

### Context
Session opened with PR #6 (`feature/acceptance-harness-batch-auto`) still open for review.
User reviewed the PR and left a comment about stale data in `test/README.md` ref:acceptance-verify.

### What Was Done
- **Merged PR #6:** `feature/acceptance-harness-batch-auto` merged to master; local master updated
- **Fixed stale ref:acceptance-verify** in `test/README.md`:
  - All 10 verifier function names updated to match session 5 renames
    (e.g. `ExitCodeZero` → `CommandSucceeded`, `SoftAccuracy` → `ClassificationAccuracyAtLeast`, etc.)
  - Dropped `col`/`subcatCol` params that were removed from the actual implementations
- **Fixed testify contradiction in session-log.md:**
  - Session 4 said "decided against testify" but session 5 adopted it in `test/verify/`
  - Added inline reversal note and split the "No testscript, no testify" decision into two bullets
  - Commit: `9ca5149` docs: fix stale ref:acceptance-verify and testify decision log
- **Reviewed auto-category source data** at `/home/leandror/workspaces/expenses/auto-category/`:
  - Confirmed `data/classification/` is an exact copy (minus zips)
  - Understand `classifications.jsonl` is the living successor to the static `feature_dictionary_enhanced.json`
- **Designed 5.5 — Classification Feedback Logging** (full plan at `.claude/plans/polished-knitting-simon.md`):
  - Single file: `classifications.jsonl` (no pending file — data is in memory at insert time)
  - Schema: `{id, item, date, value, predicted_*, actual_*, confidence, model, status, timestamp}`
  - Status values: `confirmed` / `corrected` / `manual`
  - ID: `sha256(lower(trim(item)) + "|" + date + "|" + value)[:12]`
  - Package: `internal/feedback/`
  - Category (parent) included in both predicted and actual fields
  - Commands: `auto` → confirmed; `batch-auto` (inserted rows) → confirmed; `add` → manual
  - `classify` does NOT write (exploratory only)
  - Acceptance tests: 4 cases in new `test/feedback_test.go`
  - New branch: `feature/feedback-logging-5.5`
- **Saved memory:** method-extraction-pattern (SOLID, inline comments as extraction guides)

### Decisions Made
- **Single `classifications.jsonl`** (not pending + confirmed split): pending state is unnecessary because
  classification data is already in memory at insert time; pending file would add complexity for no gain
- **`predicted_*` fields kept in all entries** even in `manual` (zero-valued): free metadata, enables
  accuracy tracking later without cross-referencing ollama-bridge logs
- **`classify` command does NOT write**: it's exploratory; only insert-producing commands write
- **`add` resolves category via taxonomy** (not workflow): avoids changing workflow signature; graceful
  degradation to empty category if taxonomy unavailable
- **`correct` command deferred**: 5.5 is infrastructure only; correction flow is future work
- **TDD for Phase 1** (feedback package): write tests red-first before implementation
- **Ollama for codegen**: feedback.go + test scaffolding are good Ollama candidates

### Next
- [ ] **Implement 5.5** on branch `feature/feedback-logging-5.5` following plan at
  `.claude/plans/polished-knitting-simon.md`:
  1. Phase 1: Create `internal/feedback/feedback.go` + tests (TDD, Ollama for codegen)
  2. Phase 2: Extend config (`ClassificationsPath` field + `ClassificationsFilePath()` method)
  3. Phase 3: Hook `auto.go`, `batch_auto.go`, `add.go`
  4. Phase 4: Acceptance tests in `test/feedback_test.go`

---

## 2026-03-11 — Session 5: Acceptance harness implementation + batch-auto + pipeline refactors

### Context
Implemented the full acceptance harness + batch-auto plan from session 4, then spent significant
time on PR review cycles, harness stabilisation, and pipeline quality improvements.

### What Was Done
- **batch-auto command (5.4):** `cmd/batch_auto.go` — read 3-field CSV → classify → insert → write
  `classified.csv` + `review.csv`; flags: `--model`, `--data-dir`, `--ollama-url`, `--threshold`,
  `--top`, `--dry-run`, `--output-dir`, `--workbook`
- **Acceptance test harness:** `test/harness/`, `test/actions/`, `test/verify/` with
  `//go:build acceptance`; Given/When/Then pattern; `run-acceptance.sh` with Ollama pre-flight,
  workbook auto-detection, test filter arg, `-keep-on-failure`/`-keep-artifacts` flags
- **9 acceptance tests, all passing:** Basic, MixedConfidence, ExcludedCategoriesGoToReview,
  ClassificationAccuracy, OutputDirFlag, SameYearInstallmentsExpanded, RolloverInstallmentsWrittenToFile,
  KnownExpenseIsClassifiedWithConfidence, AmbiguousExpenseKeptForManualReview
- **Fixtures:** `batch-auto-basic` (10 rows dry-run), `batch-auto-exclusions`, `batch-auto-installments`
  (midyear + lateyear input files, threshold 0.0), `batch-auto-rollover` (threshold 0.0)
- **Pipeline refactors:**
  - `InsertBatchExpenses` extracted into 5 focused helpers (parseExpenseStrings, expandAllInstallments,
    resolveAllSubcategories, buildEmptyRowRequests, buildExpensesWithLocations)
  - `InsertBatchExpensesFromClassified`: new entry point for pre-classified expenses, no string round-trip
  - `InsertExpense` unified — now delegates to `InsertBatchExpenses` (single pipeline)
  - `models.ClassifiedExpense` struct with `RawValue string` (preserves installment notation)
  - `models.NewExpense` constructor extracted from `ParseExpenseString`
  - `parser.ParseExpense(ClassifiedExpense)` for classified path
  - `NewAmbiguousError` now takes `[]string` sheet names — message includes actual sheet names
- **auto command fix:** Resolution/ambiguous errors now fall back to review (exit 0 + ⚠ message)
  instead of propagating as hard failures (exit 1); IO/capacity errors remain hard
- **Verify package migrated to testify** with business-descriptive names:
  `ExitCodeZero` → `CommandSucceeded`, `FileExists` → `OutputFileExists`,
  `AllConfidencesInRange` → `AllClassificationScoresValid`, `AllInReview` → `NoneWereAutoInserted`,
  `SoftAccuracy` → `ClassificationAccuracyAtLeast`, etc.
- **Harness improvements:** Removed `t.Run()` for real-time `t.Log()` output; added
  `CopyWorkbookToWorkDir` for per-test workbook isolation; `RunBatchAutoWithInput` for
  multi-input-file fixtures; `EXPENSE_WORKBOOK_PATH` auto-detected from workbook relative to script
- **Ollama IMPROVED/ACCEPTED policy documented** in CLAUDE.md and LLM repo deferred tasks

### Decisions Made
- **testify in verify package** — only there; unit tests remain stdlib
- **`t.Run()` removed from `harness.Run()`** — real-time log flushing outweighs losing subtest nesting
- **Threshold 0.0 in mechanics-testing fixtures** (installments, rollover) — decouples from
  classifier non-determinism; confidence correctness tested by other fixtures
- **Uber Centro as canonical reliable fixture item** — classifier consistently returns Uber/Taxi
- **`EXPENSE_WORKBOOK_PATH` is the global config** for workbook-dependent tests; script auto-detects
- **Exclusions acceptance test scoped to structural validation** — LLM non-determinism makes
  routing assertions fragile; exclusion logic is deterministic and covered by unit tests
- **Deferred:** Rollover/installment tests could be reduced to structural scope (tasks.md)
- **Deferred to LLM repo:** IMPROVED verdict workflow — stubs-then-Ollama pattern for missing test cases

### Next
- [ ] **PR #6** is open on `feature/acceptance-harness-batch-auto` — ready for merge to master
- [ ] **5.5** — Correction logging (`corrections.jsonl`)
- [ ] **5.6** — Few-shot injection (feature dictionary pre-filter + top-K into prompt)
- [ ] **5.7/5.8** — MCP thin wrapper in LLM repo

---

## 2026-03-03 — Session 4: Debt fix + acceptance test harness & batch-auto design

### Context
Resumed from session 3. User asked to recontextualize, address tech debt, then discuss
design for both an acceptance test harness and the 5.4 batch-auto command.

### What Was Done
- **Fixed `runtime.Caller` debt** (commit `405953e`): replaced `runtime.Caller(0)` with
  `os.Executable()` in `internal/config/config.go` — config path now resolves relative to
  the installed binary, not the compile-time source path.
- **Designed acceptance test harness** — BDD-style Given/When/Then with function injection:
  - Domain-agnostic `test/harness/` package (Context, Scenario, Run, fixture loader, CSV comparator, Ollama check)
  - Domain-specific `test/actions/` (command runners) + `test/verify/` (composable assertions)
  - `//go:build acceptance` tag, `run-acceptance.sh` wrapper, `test/results/` for drift tracking
  - Dir-per-fixture with representative batches, soft vs hard assertions for non-deterministic outputs
- **Designed batch-auto command (5.4)** — 3-field CSV input, classify via Ollama, split to
  classified.csv + review.csv, dry-run flag, continue-on-error, workbook backup before insert
- **Evaluated testscript** (rogpeppe/go-internal) — decided against: script DSL conflicts with
  user's function-injection BDD vision. Custom harness on stdlib is more aligned.
- **Evaluated testify** — decided against: project uses stdlib `testing` throughout, no value added.
  _(Reversed in session 5: testify adopted in `test/verify/` package only for business-descriptive assertions; unit tests remain stdlib)_
- **Created comprehensive implementation plan:** `.claude/plans/acceptance-harness-batch-auto.md`
- **Design decisions for shared code extraction:**
  - `isAutoInsertable` → `internal/classifier/decision.go` (shared by auto + batch-auto)
  - `buildInsertString`/`formatBRValue` → `pkg/utils/format.go` (alongside existing currency/date utils)

### Decisions Made
- **No testscript** — script DSL conflicts with function-injection BDD vision
- **testify in verify package only** — adopted in session 5 for business-descriptive assertion names; unit tests remain stdlib
- **Dir-per-fixture** with representative batches (10-20 rows), not single-row scenarios
- **Continue-on-error (option b)** for Ollama failures mid-batch — failed rows get confidence=0, go to review
- **Dry-run flag** for batch-auto — classify + write CSVs, skip workbook insertion
- **Multiple test files per command** in `package acceptance_test` (setup_test.go + per-command files)
- **Local model candidates** for bounded Go tasks during implementation (decision.go, scenario.go, etc.)

### Next
- [ ] Implement plan from `.claude/plans/acceptance-harness-batch-auto.md`:
  - Phase 1: Extract shared logic (decision.go, format.go)
  - Phase 2: Acceptance test harness skeleton + initial fixtures
  - Phase 3: batch-auto command (5.4)
  - Phase 4: Polish (run-acceptance.sh, drift tracking, docs)

---

## 2026-03-02 — Session 3: Integration testing + Diversos auto-insert fix

### Context
Continuation of session 2 (same day). After the earlier handoff, user asked to run live test
cases against real Ollama using `Planilha_Normalized_Final_copy.xlsx`.

### What Was Done
- **Integration testing (4 cases):**
  - "Diarista Letícia" 160 05/01 → 95% Diarista ✓ (--confirm, user declined)
  - "Uber Centro" 35.50 15/04 → 100% Uber/Taxi ✓ (--confirm, user declined)
  - "VA compras" 85.00 10/03 → 100% Supermercado VA — auto-inserted into workbook copy
    (finding: LLM resolves multi-word ambiguity better than keyword specificity alone; "va"
    has specificity=0.36 in feature dict but "VA compras" was unambiguous to the model)
  - "TechCorp SaaS assinatura" 49.90 01/03 → 95% Diversos — auto-inserted (**bug found**)
- **Fix: configurable auto-insert exclusion list** (commit `1ac43dd`):
  - `internal/config/config.go`: `Config` struct + `Load()` reads `config/config.json`
    (known debt: uses `runtime.Caller` for path resolution; should use `os.Executable`)
  - `config/config.json`: added `"auto_insert_excluded": ["Diversos"]`
  - `cmd/auto.go`: extracted `isAutoInsertable(result, excluded []string) bool`; distinct ⚠
    messages for threshold vs exclusion rejection
  - `cmd/auto_test.go`: 9 tests for `isAutoInsertable` including empty-exclusion-list case
- **Deferred items logged:**
  - `tasks.md` (this repo): `runtime.Caller → os.Executable` config reader debt
  - LLM `tasks.md`: `ollama-bridge file_path input` (token efficiency gap) + same config debt
- **Local model verdict:** `my-go-q25c14` used for test update — IMPROVED (wrong package name
  `expense_reporter_test` → `cmd`; syntax typo `0,.95` → `0.95`; structure was correct)

### Decisions Made
- **Exclusion list in config.json:** Not hardcoded — configurable so users can add subcategories
  without recompiling. Empty list = no exclusions (Diversos would pass through).
- **Distinct ⚠ messages:** "below threshold" vs "excluded from auto-insert" — different root
  causes deserve different messages.
- **runtime.Caller acknowledged as debt:** Logged and deferred; not blocking for development use.
- **ollama-bridge file_path:** Token efficiency gap identified — file content must pass through
  Claude context twice (read + embed in prompt). Logged in LLM tasks for future bridge enhancement.

### Next
- [ ] 5.4 — `batch-auto` command: classify a CSV, write `classified.csv` (HIGH) + `review.csv` (LOW)
- [ ] Consider: `Transporte` appearing as subcategory at 90% in case 2 — taxonomy oddity, not urgent

---


## 2026-03-13 — Session 7: 5.5 implementation (classification feedback logging)

### What Happened
- Implemented 5.5 in 4 phases on branch `feature/feedback-logging-5.5`
- **Phase 1 (TDD):** `internal/feedback/` package — `Entry`, `GenerateID`, `Append`, `NewConfirmedEntry`, `NewManualEntry`; 6 unit tests green
- **Phase 2 (Config):** `ClassificationsPath` + `ClassificationsFilePath()` in `internal/config/`; 3 unit tests
- **Phase 3 (Hooks):** `auto.go` → `logConfirmedFeedback`; `batch_auto.go` → `logBatchFeedback`; `add.go` → `logManualFeedbackFromAdd` with 3 helpers
- **Phase 4 (Acceptance):** `feedback_test.go` (4 tests), `verify/feedback.go` (5 verifiers), `RunAdd` action, `SetupBinaryConfig` harness helper, `batch-auto-feedback` fixture
- Ollama calls: `feedback.go` ACCEPTED (~800 est. tokens saved); test file timed out x2 (cold start); feedback_test.go REJECTED (wrong harness API)
- Rule change noted: 1st timeout = retry, not reject

### Decisions Made
- **Non-fatal logging:** All feedback writes are best-effort — stderr warning only, never blocks the command
- **add.go data-dir:** Hardcoded `"data/classification"` relative path for taxonomy lookup; silently empty category if not found
- **DryRun test fixture:** Reused existing `batch-auto-basic` (already has `--dry-run` in extra_args) — no new fixture needed
- **batch-auto-feedback fixture:** threshold=0.0 to force all rows into auto-insert regardless of confidence

### Next
- [ ] **Open PR for 5.5** on `feature/feedback-logging-5.5` → merge to master
- [ ] **Run acceptance tests** to verify end-to-end: `./run-acceptance.sh`
- [ ] **5.7** — few-shot injection: consume `classifications.jsonl` as training signal

---
