# Session Log — Expense Reporter

**Previous logs:** `.claude/archive/session-log-2026-02-27-to-2026-02-27.md`, `.claude/archive/session-log-2026-03-02-to-2026-03-02.md`, `.claude/archive/session-log-2026-03-13-to-2026-03-02.md`, `.claude/archive/session-log-2026-03-03-to-2026-03-03.md`, `.claude/archive/session-log-2026-03-11-to-2026-03-11.md`, `.claude/archive/session-log-2026-03-13-to-2026-03-13.md`, `.claude/archive/session-log-2026-03-13-to-2026-03-13.md`, `.claude/archive/session-log-2026-03-14-to-2026-03-14.md`, `.claude/archive/session-log-2026-03-18-to-2026-03-18.md`, `.claude/archive/session-log-2026-03-18-to-2026-03-18.md`, `.claude/archive/session-log-2026-03-23-to-2026-03-23.md`, `.claude/archive/session-log-2026-03-27-to-2026-03-27.md`, `.claude/archive/session-log-2026-04-20-to-2026-04-20.md`, `.claude/archive/session-log-2026-04-22-to-2026-04-22.md`, `.claude/archive/session-log-2026-04-23-to-2026-04-23.md`
Most recent entry first. Run `.claude/tools/rotate-session-log.sh` when this grows beyond ~3 sessions.

---

## 2026-05-12 — Session 20: Workbook Path Resolution Bug Fix

### Context
Resumed from session 19. Discussed next steps; chose to fix the latent `BUG_REPORT_DEFAULT_WORKBOOK_PATH.md` bug. PRs #16 and #17 had already been merged to master.

### What Was Done
- **Investigated the bug:** Confirmed `GetWorkbookPath` in `root.go` used `os.Executable()` to resolve a default path — broken under `go run` (exe in build cache). Also confirmed `config.json` already has `workbook_path` field and `Config` struct has `WorkbookPath`, but neither was being used by `GetWorkbookPath`.
- **Designed fix:** Resolution order: `--workbook` flag → `EXPENSE_WORKBOOK_PATH` env → `config.Load()` + `cfg.WorkbookFilePath()` → clear error. Malformed config surfaces; absent config falls through to error. Multi-workbook-per-year design deferred.
- **Plan written:** `.claude/plans/fix-workbook-path-resolution.md` — advisor-reviewed, covers TDD order, test separation, session-end housekeeping.
- **TDD implementation (red → green):**
  - Added `TestWorkbookFilePath_{Empty,Absolute,Relative}` to `internal/config/config_test.go` (mirrors existing `ClassificationsFilePath` test pattern)
  - Updated `TestGetWorkbookPath/empty_environment_variable` → `wantErr: true`
  - Added `WorkbookFilePath()` to `Config` in `internal/config/config.go` (exact mirror of `ExpensesLogFilePath`)
  - Rewrote `GetWorkbookPath` in `cmd/root.go`: dropped `os.Executable()` relative path block and Windows hardcoded fallback; now calls `config.Load()` + `cfg.WorkbookFilePath()`
- **All 13 unit test packages green.** Call site verification: all 4 callers (`add`, `auto`, `batch`, `batch_auto`) already surface the error.
- **Branch created:** `fix/workbook-path-resolution`

### Decisions Made
- **Config as fallback, not elimination:** `WorkbookFilePath()` on Config used as step 3 in resolution — consistent with how `ClassificationsFilePath` and `ExpensesLogFilePath` work.
- **Multi-workbook-per-year deferred:** Vision is `"workbooks": {"2025": "...", "2026": "..."}` keyed by year from expense date. Tracked in tasks.md for a future session.
- **No acceptance test changes needed:** Acceptance tests pass `--workbook` explicitly via `ctx.WorkbookPath`; fix is transparent to them.
- **`SetupBinaryConfig` is acceptance-only:** Could not be used from `main_test.go`. `WorkbookFilePath` tested directly in `config_test.go` instead.

### Next
- Commit all changes on `fix/workbook-path-resolution`
- Create PR (base: master)
- Future options: `TestBatchAuto_SameYearInstallmentsExpanded` test-debt fix (trivial), 5.R1 TF-IDF retrieval (if classification run data justifies it)

---

## 2026-04-27 — Session 19: Layer 5.8 Doc Reconciliation

### Context
Post-merge of session-18 PRs. User asked to review next-steps and confirm whether 5.8 had remaining work.

### What Was Done
- Verified Layer 5.8 is fully shipped: MCP server in `mcp-server/` (this repo, not LLM repo), 2 tools (`classify_expense`, `add_expense`), `auto_add` deliberately dropped per `mcp-server/.memories/KNOWLEDGE.md` "Two Tools, Not Three" (2026-03-27).
- Reconciled stale docs (commit 3e38fed): `tasks.md` 5.8 → `[x]`, `docs/expense-classifier-vision.md` Phase 3 → SHIPPED, `session-context.md` Domain Boundary collapsed.
- Saved 2 memory entries: MCP server lives in this repo (corrected user feedback); Layer 5.R1 evaluation procedure (trigger definitions + verified instrumentation prerequisite — `feedback.Entry` lacks keyword-hit field, so existing `classifications.jsonl` cannot be segmented hit-vs-miss without code change first).

### Decisions Made
- Don't start R1 on speculation. Wait for user's run review; future session uses saved evaluation procedure.
- Misc planning .txt files moved out of repo root by user (no longer untracked).

### Next
- User reviewing classification run; may justify R1 in a future session — see memory `project_r1_evaluation_procedure.md`.
- Otherwise: BUG_REPORT_DEFAULT_WORKBOOK_PATH (small), `TestBatchAuto_SameYearInstallmentsExpanded` test-debt fix (trivial), or acceptance suite timeout split.

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

