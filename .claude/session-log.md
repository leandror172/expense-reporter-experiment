# Session Log — Expense Reporter

**Previous logs:** `.claude/archive/session-log-2026-02-27-to-2026-02-27.md`, `.claude/archive/session-log-2026-03-02-to-2026-03-02.md`, `.claude/archive/session-log-2026-03-13-to-2026-03-02.md`, `.claude/archive/session-log-2026-03-03-to-2026-03-03.md`, `.claude/archive/session-log-2026-03-11-to-2026-03-11.md`, `.claude/archive/session-log-2026-03-13-to-2026-03-13.md`, `.claude/archive/session-log-2026-03-13-to-2026-03-13.md`, `.claude/archive/session-log-2026-03-14-to-2026-03-14.md`, `.claude/archive/session-log-2026-03-18-to-2026-03-18.md`, `.claude/archive/session-log-2026-03-18-to-2026-03-18.md`, `.claude/archive/session-log-2026-03-23-to-2026-03-23.md`, `.claude/archive/session-log-2026-03-27-to-2026-03-27.md`, `.claude/archive/session-log-2026-04-20-to-2026-04-20.md`, `.claude/archive/session-log-2026-04-22-to-2026-04-22.md`, `.claude/archive/session-log-2026-04-23-to-2026-04-23.md`, `.claude/archive/session-log-2026-04-24-to-2026-04-24.md`
Most recent entry first. Run `.claude/tools/rotate-session-log.sh` when this grows beyond ~3 sessions.

---

## 2026-05-15 — Session 21: Review UI Design Brief + `review` Command Plan

### Context
User wants a web-ish UI to review classified expenses with a full-tree category
picker (sheet → category → subcategory), instead of eyeballing a CSV. They had
tested Lovable (`docs/plans/lovable-suggestion-plan.md`) but disliked its
cloud-first architecture (CLI HMAC-pushing to Lovable Postgres).

### What Was Done
- **Rejected the Lovable cloud architecture** in favour of local-first: the review
  UI is a single self-contained HTML file the CLI bakes data into. No cloud, no
  HMAC, no CLI-to-cloud push.
- **Wrote the design brief** `docs/plans/review-ui-design-brief.md` — for
  claude.ai/design (builds the HTML in a separate session). Specifies the
  `__REVIEW_DATA__` injection contract, three JSON data contracts as TS types, the
  3-level cascading picker, the pre-fill rule (1/>1/0 taxonomy matches), export
  format, offline/single-file hard constraints.
- **Wrote fixtures** `docs/plans/review-ui-fixtures/{review-data,reviewed}.sample.json`
  — input + output samples; taxonomy deliberately includes a category under two
  sheets to exercise the ambiguity case.
- **Wrote the implementation plan** `.claude/plans/review-command.md` — detailed,
  phased plan for `expense-reporter review` (CSV + workbook taxonomy → review.html),
  acceptance-test-first, with open questions O1–O3 flagged. To be executed in a new
  session.
- Indexed all new docs in `.claude/index.md`.

### Decisions Made
- **Local, CLI-served, not cloud.** Single user, desk-only review — confirmed no
  multi-device need. Lovable plan is superseded.
- **CLI is the producer of a self-contained file, not a server.** `review` bakes
  queue + taxonomy into an HTML template; the browser's file-download is the only
  "output" channel. No `review` HTTP server, no endpoints.
- **Workbook write is out of scope for `review`.** The UI emits `reviewed.json`; a
  separate future `apply` command ingests it into the workbook + feedback logs.
- **Taxonomy source = workbook's "Referência de Categorias" sheet** via
  `excel.LoadReferenceSheet` → grouped into the 3-level tree at runtime.
- **`Predicted.sheet` is optional** in the contract — forward-compat for a planned
  CSV change that emits the full 3-level path.

### Next
- Execute `.claude/plans/review-command.md` in a new session (Phase 0 → 5).
- Resolve open questions O1 (installment value notation), O2 (test workbook
  fixture), O3 (`id` hash without year) before/while coding.
- Hand the brief + fixtures to claude.ai/design to build `review.html`.

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
- **Advisor post-review fix:** `batch_auto.go:104` was wrapping `GetWorkbookPath` error with `"workbook path not configured: %w"`, producing a duplicate prefix. Fixed to bare `%w` (inner error is now self-sufficient). `batch_auto.go:220` updated to `"failed to get workbook path: %w"` for consistency with all other callers (`add`, `auto`, `batch`). Smoke test confirmed clean message; 417 tests green.

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

