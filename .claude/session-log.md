# Session Log — Expense Reporter

**Previous logs:** `.claude/archive/session-log-2026-02-27-to-2026-02-27.md`, `.claude/archive/session-log-2026-03-02-to-2026-03-02.md`, `.claude/archive/session-log-2026-03-13-to-2026-03-02.md`, `.claude/archive/session-log-2026-03-03-to-2026-03-03.md`, `.claude/archive/session-log-2026-03-11-to-2026-03-11.md`
Most recent entry first. Run `.claude/tools/rotate-session-log.sh` when this grows beyond ~3 sessions.

---

## 2026-03-18 — Session 10: 5.7 planning — few-shot injection

### Context
First session after 5.6 merge. Recontextualized via `resume.sh`. Entire session was design/planning for 5.7 — no implementation code written.

### What Was Done
- **Retrieval strategy analysis** — explored current classifier flow, token budget, data sources, and designed a 3-layer cascade pipeline (keywords → TF-IDF → embeddings)
- **3 reference documents created** (committed to master):
  - `data/classification/retrieval-strategy.md` — high-level pipeline, token budget (~462 baseline, ~20/example), data source merge strategy
  - `data/classification/tfidf-retrieval.md` — TF-IDF findings: existing IDF weights in feature dict, 229-dim vectors, Go implementation approach
  - `data/classification/embedding-retrieval.md` — Ollama `/api/embeddings` API, vector store sizing, multilingual considerations, decision criteria
- **5 deferred tasks** added to `tasks.md` (5.R1–5.R5): TF-IDF layer, embedding layer, value-range plausibility, historical workbook extraction, correction-weighted selection
- **8 BDD acceptance test scenarios** designed for `fewshot_test.go`
- **Implementation plan** written at `.claude/plans/5.7-few-shot-injection.md` — 6 phases: example selection engine, data loading, prompt construction, acceptance tests, existing test verification, doc updates
- **Convention change:** unit tests now use testify (assert/require) — saved to memory

### Decisions Made
- **Layered retrieval cascade** — keywords first (5.7), TF-IDF later (5.R1), embeddings last (5.R2); complementary layers, not replacements
- **Both data sources** for examples: `training_data_complete.json` (694 static entries) + `classifications.jsonl` (runtime feedback, filtered `status != manual`)
- **Few-shot as conversation turns** — user/assistant message pairs before the real query (not appended to system prompt)
- **`--verbose` flag** for observability — existing flag, use `logger.Debug` for few-shot injection details; acceptance tests assert on this output
- **No file-based prompt template** (for now) — keep `strings.Builder` pattern; consider `embed.FS` if iteration velocity increases
- **Correction prioritization** — simple sort (corrected > training > confirmed), not weighted scoring; tested via acceptance (verbose output) + unit tests (selection algorithm)
- **Testify for unit tests** — convention change from stdlib-only; don't retroactively convert existing tests
- **Graceful degradation** — missing training data returns empty examples (not error); classifier falls back to taxonomy-only prompt

### Next
- [ ] **Execute 5.7 plan** on branch `feature/5.7-few-shot-injection` following `.claude/plans/5.7-few-shot-injection.md` — implementation intended for Sonnet model
- [ ] Plan includes recontextualization instructions (resume.sh + read session-context.md)

---

## 2026-03-14 — Session 9: 5.6 expense persistence log

### Context
Resumed after 5.5 merge. Recontextualized; discovered 5.6 was in tasks.md but skipped in last session's "next" pointer. Copied vision docs from LLM repo to `docs/`. Discussed separation of `expenses_log.jsonl` from `classifications.jsonl` and decided the separation was worth doing now.

### What Was Done
- **Copied vision docs** — `docs/expense-classifier-vision.md` and `docs/expense-classifier-data-inventory.md` from LLM repo; indexed in `.claude/index.md`
- **5.6 implementation** — `internal/feedback/expense_log.go`: `ExpenseEntry`, `NewExpenseEntry`, `AppendExpense`; `ExpensesLogPath`/`ExpensesLogFilePath()` in config; `expenses_log_path` in `config.json`; `logExpense()` wired into `auto`, `add`, `batch-auto`
- **Unit tests** — `expense_log_test.go` (4 tests incl. cross-ID consistency check); 3 new `config_test.go` tests for `ExpensesLogFilePath`
- **Acceptance tests refactored** — composable Then helpers (`commandSucceeded`, `autoInsertSucceeded`, `classificationsMatchExpected`, `expenseLogMatchesExpected`, `noLogsCreated`) composed via `slices.Concat`; file-specific verifiers (`ClassificationsMatch`, `ExpenseLogMatches`, `ClassificationsNotCreated`, `ExpenseLogNotCreated`) added to `verify/feedback.go`; `expected-expenses_log.jsonl` fixture files added for 3 fixture dirs
- **PATTERNS.md updated** — Then composition pattern, JSONL log verification rules, fixture field selection guidance documented
- **All 13 acceptance tests pass** on master
- **Merged to master** via PR

### Decisions Made
- `expenses_log.jsonl` separated from `classifications.jsonl` — the two files have different concerns (insert identity vs. learning signal); mixing them would require filtering `status: manual` entries in 5.7 and complicates future lifecycle tracking
- `expected-expenses_log.jsonl` omits `subcategory`/`category` for classifier-dependent tests (LLM non-determinism); includes them for `add` tests (deterministic)
- Then helpers use `slices.Concat` over plain slice literal — preserves ability to group multiple assertions per helper while staying composable
- File-specific verifiers embed artifact key — `ClassificationsMatch(path)` not `FeedbackMatchesExpected("classifications.jsonl", path)`
- GoLand `//go:build acceptance` navigation: configure via `Settings → Go → Build Tags`, add `acceptance` — do not remove the tag

### Next
- [ ] **Start 5.7** — few-shot injection: load top-K entries from `classifications.jsonl` (filter `status != manual`), keyword pre-match against training data, inject as few-shot examples into Ollama classifier prompt

---

## 2026-03-13 — Session 8: 5.5 PR review + merge

### Context
Resumed with PR #7 (`feature/feedback-logging-5.5`) open. User reviewed it and left 3 inline comments on `feedback_test.go`.

### What Was Done
- **Ran full acceptance suite** — all 13 tests pass (4 new feedback tests confirmed green end-to-end)
- **Fixed run-acceptance.sh CRLF** — file had Windows line endings; stripped `\r`; restored to LF (was already LF in index, so no commit needed)
- **Addressed 3 PR review comments:**
  1. **Given names** — renamed to business-oriented: `knownExpenseReadyForAutoInsert`, `knownExpenseBatchReadyForInsert`, `mixedExpensesReadyForDryRun`, `singleExpenseReadyForManualAdd`
  2. **Then functions** — extracted named helpers returning `[]func(*harness.Context)`: `autoInsertConfirmedInFeedback`, `batchInsertionsConfirmedInFeedback`, `noFeedbackFileCreated`, `manualEntryLoggedInFeedback`
  3. **Expected output files** — added `FeedbackMatchesExpected` verifier (partial-match, skips `id`/`timestamp`); added `expected-feedback.jsonl` to `auto-basic/`, `batch-auto-feedback/`, and new `add-feedback/` fixtures
- **PR #7 merged** to master; local master updated

### Decisions Made
- **`FeedbackMatchesExpected` uses partial-match semantics**: expected file specifies only the fields to check; `id` and `timestamp` always skipped (implementation details)
- **LLM-dependent fields excluded from auto/batch expected files**: `predicted_subcategory`, `actual_subcategory`, `confidence` omitted since classifier output is non-deterministic; add test has full expected (no LLM path)
- **1st Ollama timeout = retry, not reject** (user clarification): only treat as first rejection if the model actually responds with wrong output; cold-start timeouts get one free retry

### Next
- [ ] **Start 5.7** — few-shot injection: load top-K entries from `classifications.jsonl` and inject into Ollama classifier prompt
- [ ] Run full acceptance suite with `run-acceptance.sh` to confirm master is clean

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

