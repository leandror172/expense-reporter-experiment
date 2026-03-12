# Session Log — Expense Reporter

**Previous logs:** `.claude/archive/session-log-2026-02-27-to-2026-02-27.md`
Most recent entry first. Run `.claude/tools/rotate-session-log.sh` when this grows beyond ~3 sessions.

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
- **Created comprehensive implementation plan:** `.claude/plans/acceptance-harness-batch-auto.md`
- **Design decisions for shared code extraction:**
  - `isAutoInsertable` → `internal/classifier/decision.go` (shared by auto + batch-auto)
  - `buildInsertString`/`formatBRValue` → `pkg/utils/format.go` (alongside existing currency/date utils)

### Decisions Made
- **No testscript, no testify** — custom harness on stdlib, consistent with existing 190+ tests
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

## 2026-03-02 — Session 2: Layer 5 tasks 5.1–5.3

### Context
First active feature session. Resumed from session 1 (scaffolding). Started with recontextualization
(memory + resume.sh + ref_lookup), then proceeded with 5.1 → 5.2 → 5.3 in order.

### What Was Done
- **5.1 (docs):** Added three ref blocks to `.claude/index.md`:
  `ref:training-data-schema`, `ref:confidence-thresholds`, `ref:classification-overview`.
  All three were already referenced from `ref:classification` but had no content.
- **5.2 (classify command):** `internal/classifier/` package with `Classify()`, `LoadTaxonomy()`,
  `buildSystemPrompt()`; calls Ollama `/api/chat` with structured JSON format param.
  `cmd/classify.go` — 3 positional args (item, value, DD/MM), `--model`, `--top`, `--data-dir` flags.
  11 tests covering LoadTaxonomy, buildSystemPrompt, and Classify via httptest mock server.
- **5.3 (auto command):** `cmd/auto.go` — classify + auto-insert if confidence ≥ 0.85;
  `--confirm` flag prompts before inserting; `⚠ Not inserted` signal on low confidence; exit 0 always.
  Tests for `formatBRValue`, `buildInsertString`, `confirmInsert` (y/Y/yes + n/N/no/empty).
- **classify fix:** Swapped `strconv.ParseFloat` → `utils.ParseCurrency` so both `35.50` and `35,50` accepted.
- **LLM repo notes:** Added session-37 entry to `session-log.md` and deferred task for `ref_lookup`
  cross-repo support to `tasks.md` in `/mnt/i/workspaces/llm/`.
- **Branch:** `feature/layer5-classifier` (3 commits: b4e4c61, d623cd7, bd8aebe)

### Decisions Made
- **classify input format:** Positional args (`classify "item" value DD/MM`), not semicolon string.
  Chosen for CLI idiom and standard float; `utils.ParseCurrency` added to accept both `.` and `,`.
- **auto exit code:** Always exit 0 on successful run; non-zero only on actual errors. Signal via stdout `⚠`.
- **auto --confirm:** Prompts even on HIGH confidence when flag is set; default no-insert on empty/non-y input.
- **Feature dictionary in 5.2:** Skipped — 5.2 is pure LLM path; pre-filter optimization deferred to 5.7.
- **Local model use:** Cobra command scaffold generated with `my-go-q25c14` (verdict: IMPROVED —
  dropped spurious date parsing and context arg; structure and flag registration were correct).
- **TDD note:** Tests were written after implementation for 5.2 (not red-first); corrected for 5.3.

### Next
- [ ] 5.4 — `batch-auto` command: classify a CSV, write `classified.csv` (HIGH) + `review.csv` (LOW)
- [ ] 5.5 — Correction logging: `corrections.jsonl`
- [ ] Update tasks.md in this repo to mark 5.1–5.3 complete

---

