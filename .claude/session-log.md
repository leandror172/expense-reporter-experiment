# Session Log — Expense Reporter

**Previous logs:** `.claude/archive/session-log-2026-02-27-to-2026-02-27.md`, `.claude/archive/session-log-2026-03-02-to-2026-03-02.md`, `.claude/archive/session-log-2026-03-13-to-2026-03-02.md`, `.claude/archive/session-log-2026-03-03-to-2026-03-03.md`, `.claude/archive/session-log-2026-03-11-to-2026-03-11.md`, `.claude/archive/session-log-2026-03-13-to-2026-03-13.md`, `.claude/archive/session-log-2026-03-13-to-2026-03-13.md`, `.claude/archive/session-log-2026-03-14-to-2026-03-14.md`, `.claude/archive/session-log-2026-03-18-to-2026-03-18.md`, `.claude/archive/session-log-2026-03-18-to-2026-03-18.md`, `.claude/archive/session-log-2026-03-23-to-2026-03-23.md`, `.claude/archive/session-log-2026-03-27-to-2026-03-27.md`, `.claude/archive/session-log-2026-04-20-to-2026-04-20.md`
Most recent entry first. Run `.claude/tools/rotate-session-log.sh` when this grows beyond ~3 sessions.

---

## 2026-04-25 — Session 18: Batch-Auto CSV Preservation Fix

### Context
Resumed from session 17's handoff. User: "We'll work on batch-auto-preserve-csvs-on-insert-failure.md to solve BUG_REPORT.md" with strict advisor rules (conflict resolution, response format). Started by reading bug report, plan, current code to ground understanding before calling advisor on test strategy.

### What Was Done
- **Analyzed bug root cause:** `runBatchAuto` orders work (classify → insert → CSV-write). Early return on insert failure discards CSV writes, but classification is already in memory. Plan is two-layer fix: Layer 1 (fail-fast), Layer 2 (reorder).
- **Acceptance-test design forced option B (both layers):** Advisor reconciliation call resolved premise conflict — Layer 2 is *not* dead code if we strengthen `insertClassified` to return error on unopenable workbook, making reorder meaningful for corrupt files.
- **Implementation (TDD: red → green):**
  - `ValidateWorkbook(path) error` in `excel/` — opens + closes xlsx, returns parse error if corrupt
  - `insertClassified` — calls `ValidateWorkbook` after `os.Stat`, returns wrapped error
  - `runBatchAuto` rewrote — captures `insertErr` instead of early-returning, always runs CSV writes
  - Layer 1 UX: workbook path validation before `classifyLines` with actionable hint
- **Acceptance tests (both green, no Ollama):**
  - `TestBatchAuto_MissingWorkbook_FailsFastBeforeClassification` — Layer 1, exits in 0.01s
  - `TestBatchAuto_InsertFailure_PreservesCSVs` — Layer 2, corrupt xlsx, CSVs written, exit 1
- **Testing:** All 13 unit tests green; new acceptance tests green; full acceptance suite hits 600s infrastructure timeout (not regression)
- **Memory saved:** Ollama context_files base path (Go module root, not repo root)
- **1 commit:** `8f838a6` on `fix/batch-auto-preserve-csvs-on-insert-failure`

### Decisions Made
- **Acceptance-first workflow validated again** — test scenarios revealed implementation correctness (Layer 1 fast-fail only observable via elapsed time, Layer 2 CSV+error both visible in acceptance context)
- **Conflict resolution pattern successful:** Advisor's reconciliation call on Layer 2 premise prevented unnecessary code
- **ValidateWorkbook in excel package** — workbook-validation concern belongs with workbook code; orchestrators delegate file ops

### Next
- Create PR for batch-auto fix (base: master)
- BUG_REPORT_DEFAULT_WORKBOOK_PATH.md deferred — latent bug, out-of-scope. Option 1 recommended: remove executable-relative default.
- Two open PRs from session 15: #16, #17 — post-merge order TBD

### Gotchas
- **Acceptance suite timeout:** Full 600s is tight (Basic 286s + MixedConfidence 299s = 585s remaining). New tests fast (<5s) but suite times out mid-flight. Infrastructure constraint for future sessions.
- **Ollama context_files paths:** Must be absolute from module root, not repo root or symlinks. Early attempts failed silently.

---

## 2026-04-24 — Session 17: MCP-Layer Corrections Shipped

### Context
Resumed from session 16's plan (`docs/plans/mcp-layer-corrections.md`). Resolved two open micro-decisions before starting: `chosen == predicted` → write `confirmed` (training signal, consistency with auto); ID miss → warn-and-continue (insert is primary, feedback is best-effort).

### What Was Done
- **Step 1:** Grepped `internal/classifier/` — zero references to `expenses_log.jsonl`, confirming no double-count risk.
- **Step 2:** Wrote 3 acceptance tests in `feedback_test.go` (prediction match → confirmed, mismatch → corrected, no flags → manual/backwards-compat). Extended `RunAdd` with variadic `extraFlags` and `--data-dir` forwarding. Fixed Given naming per PATTERNS.md: `expenseClassifiedByModel` (past-tense action, not state).
- **Step 3:** TDD inner loop — `TestLogPredictedFeedback` (4 cases: confirmed, corrected, ID-miss-warn, no-path no-op) red first; then implemented `logPredictedFeedback` + 5 new cobra flags in `add.go`. All 190+ unit tests green.
- **Step 4:** `AutoOutput` gains `classification_id` (sha256[:12] of item|date|value); `auto --json` populates it. `add_expense` MCP tool in `server.py` extended with 5 optional prediction params forwarded as CLI flags. 7 MCP tests green.
- **Step 5:** `docs/FEEDBACK_SYSTEM.md` updated — new `add` prediction-flags source documented, "future work" bullet removed. `session-context.md` current-status + Telegram-flow line updated.
- **2 commits:** `15a8082` (feat: add flags + feedback branching) + `6ef3e5b` (feat: MCP layer).

### Decisions Made
- **`chosen == predicted` → write `confirmed`:** Training signal; consistent with `auto`'s existing confirmed-writes; no double-count risk since `add` is the only writer in the MCP path.
- **ID miss → warn-and-continue:** Insert must not be blocked by a log concern. Feedback is best-effort; all predicted context is already in the flags.
- **MCP Python changes done by Claude directly:** No Python persona in tier list; change was purely mechanical pattern-repetition — no benefit to delegating to Ollama.
- **Ollama prompt style correction (saved to memory):** Prompts must describe behavior, not spell out implementation code line-by-line. Prior sessions were passing if-else logic as literal code.
- **Parallel model calls reinforced as bad:** Calling two different-sized models simultaneously causes VRAM contention worse than same-model parallel. Always serial, always tier 1 first.
- **RunAdd extended with variadic extraFlags:** Cleaner than creating multiple named actions; backwards-compatible; --data-dir forwarding was the missing piece for taxonomy resolution in acceptance tests.

### Next
- **Open PRs still unmerged:** #16 (docs/feedback-system-csv-reconstruction) and #17 (correct command) — consider creating PR for the MCP-layer corrections on this branch (`feature/correct-command`)
- **Uncommitted:** `CLAUDE.md`, `.claude/session-context.md`, `docs/FEEDBACK_SYSTEM.md` — commit docs as session close
- **Next feature investment:** 5.R1 TF-IDF retrieval (better few-shot example selection) — documented in `internal/classifier/.memories/QUICK.md`

---

## 2026-04-23 — Session 16: Planning MCP-Layer Corrections

### Context
Short planning session. User asked about the "Telegram-flow corrections at MCP layer" item from session 15's "Next" pointer. Read `docs/expense-classifier-vision.md` mid-discussion to ground the plan in the documented Phase 3/4 seam.

### What Was Done
- Discussed scope, signal shape, and semantics of closing the Telegram correction loop
- Confirmed MCP wrapper lives in **this repo** at `mcp-server/` (not in the llm repo — user corrected an earlier assumption)
- Discovered during scoping grep: feedback schema already supports `status: confirmed | corrected | manual`, `predicted_*` / `actual_*` fields, and `GenerateID`; `auto.go` and `batch_auto.go` already write `NewConfirmedEntry`; `correct.go` writes `NewCorrectedEntry`; only `add.go` is the gap (writes `NewManualEntry` with no prediction context)
- Produced a plan written to `docs/plans/mcp-layer-corrections.md` for execution next session

### Decisions Made
- **Signal shape (c):** both `--classification-id` AND `--predicted-subcategory` on `add` — ID for linkage per vision's lifecycle model, predicted-subcategory as authoritative signal that survives log rotation
- **Confirmation logging: deferred** (user choice 2.C) — but `auto`'s existing confirmed-writes keep working. Open question logged for execution: when `add --predicted-subcategory X` has `chosen == X`, write `confirmed` (lean: yes, consistency with `auto`) or skip.
- **Double-logging is fine:** `expenses_log.jsonl` (insert event) and `classifications.jsonl` (correction event) legitimately record different events, joined by shared ID; no double-count risk since few-shot reads only `classifications.jsonl` (verify at execution start)
- **Out of scope:** status lifecycle, SQLite migration, Telegram bot implementation, 5.R1 retrieval work

### Next
- Execute `docs/plans/mcp-layer-corrections.md` — start with Step 1 (verify no double-count risk) then acceptance tests
- Still open from session 15: PR #17 merge status vs. PR #16; gitignore cleanup for `expense-reporter` binary + `expenses_failed_*.csv`

---

## 2026-04-22 — Session 15: `correct` Command — Closing the Feedback Loop (Layer 5.9)

### Context
Resumed via `.claude/tools/resume.sh` from session 14's "Next" pointer. Discussed Option B (correction workflow) over Option A (TF-IDF retrieval) because the feedback loop being broken is the bigger value gap. Adopted "acceptance tests first" as a durable workflow rule.

### What Was Done
- **Acceptance-first workflow established** (saved to memory) — discuss scenarios, write acceptance tests, then drop into TDD inner loop for unit tests
- **Naming conventions documented in `expense-reporter/test/PATTERNS.md`**:
  - **Given:** Event Modeling style — past-tense events that happened (e.g., `expenseAutoConfirmed`, `expenseConfirmedThenCorrected`); state-only exception for empty event streams (e.g., `noClassificationsRecorded`)
  - **Then:** composable `[]func(*Context)` slices joined via `slices.Concat`
- **Test scaffolding additions:**
  - `verify.CommandFailed` (inverse of CommandSucceeded)
  - `harness.SeedFileFromFixture` (copy one named fixture file into WorkDir)
  - `actions.RunCorrect` (no `--workbook` flag; passes `--data-dir`)
- **Production code** (TDD: red acceptance → impl → green):
  - `feedback.NewCorrectedEntry` — predicted from prior, actual from user, status=corrected
  - `feedback.FindLatestEntry(path, id)` — scans JSONL, returns last matching entry
  - `cmd/correct.go` cobra command — parses 4-field input, looks up prior, writes corrected entry; fails clearly when no prior entry exists (hint to use `add`); does NOT touch the workbook
- **2 fixtures + 3 acceptance tests** — overrides-confirmed, fails-when-no-prior, latest-wins-on-duplicate-IDs (all 3 green in <1s, no Ollama needed)
- **Tooling:** `.claude/tools/lookup-category.py` — subcategory→category lookup against `feature_dictionary_enhanced.json`
- **Persona/tier-list update:** added `my-go-g3-12b` (gemma3:12b) at #2 in CLAUDE.md Go codegen tier list
- **Memory updates:**
  - New: `feedback_acceptance_first.md`
  - Sharpened `feedback_ollama_timeouts.md` — 3 parallel Ollama codegen calls is too much for non-trivial prompts (VRAM ceiling)
  - Updated `feedback_system_findings.md` — flipped "missing corrections" section to documented closed gap; added correct command to entry points
- **Doc sweep:** `expense-reporter/README.md`, root `README.md`, `expense-reporter/.memories/QUICK.md`, `test/.memories/QUICK.md`, `docs/FEEDBACK_SYSTEM.md` (renamed ref block `feedback-missing-feature` → `feedback-correction-workflow`); `.claude/index.md` ref pointer updated
- **Branch & PRs:**
  - Created `feature/correct-command` branch (off `docs/feedback-system-csv-reconstruction`)
  - 2 commits: `3ddd46f` (feat), `4b97c12` (docs)
  - **PR #17** opened to master: https://github.com/leandror172/expense-reporter-experiment/pull/17 — includes 2 commits from PR #16 (still open) as ancestors

### Decisions Made
- **`correct` is feedback-only** — no `--workbook` flag, does not move cells. Workbook cell relocation is bundled responsibility; keep `correct` single-purpose. User fixes the workbook manually for now.
- **`correct` requires a prior entry** — fails with hint to use `add` if none exists. Matches the design: corrections always override a model prediction. `review.csv` items (never auto-inserted) belong to `add`.
- **Telegram-flow corrections deferred to MCP/bot layer** — writing `corrected` at insert time when user picks a non-top candidate needs an MCP-layer extension (e.g., `--predicted-subcategory` flag on `add`); not part of CLI scope.
- **Gemma persona used as-is** (`my-go-g3-12b`, role differs from qcoder) rather than copied from qcoder — avoid spec-overwrite when an existing active persona works.
- **Ollama parallelization ceiling**: 3 parallel codegen calls only safe for tiny near-identical prompts; default to serial otherwise.

### Next
- Verify PR #17 in CI; merge order with PR #16 (rebase if #17 lands first)
- After merge: 5.R1 TF-IDF retrieval (better few-shot example selection) OR Telegram-flow `corrected` extension at MCP layer — user pick
- Consider gitignore cleanup for `expense-reporter/expense-reporter` binary and `expenses_failed_*.csv` artifacts (separate small commit)

---

