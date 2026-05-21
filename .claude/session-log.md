# Session Log — Expense Reporter

**Previous logs:** `.claude/archive/session-log-2026-02-27-to-2026-02-27.md`, `.claude/archive/session-log-2026-03-02-to-2026-03-02.md`, `.claude/archive/session-log-2026-03-13-to-2026-03-02.md`, `.claude/archive/session-log-2026-03-03-to-2026-03-03.md`, `.claude/archive/session-log-2026-03-11-to-2026-03-11.md`, `.claude/archive/session-log-2026-03-13-to-2026-03-13.md`, `.claude/archive/session-log-2026-03-13-to-2026-03-13.md`, `.claude/archive/session-log-2026-03-14-to-2026-03-14.md`, `.claude/archive/session-log-2026-03-18-to-2026-03-18.md`, `.claude/archive/session-log-2026-03-18-to-2026-03-18.md`, `.claude/archive/session-log-2026-03-23-to-2026-03-23.md`, `.claude/archive/session-log-2026-03-27-to-2026-03-27.md`, `.claude/archive/session-log-2026-04-20-to-2026-04-20.md`, `.claude/archive/session-log-2026-04-22-to-2026-04-22.md`, `.claude/archive/session-log-2026-04-23-to-2026-04-23.md`, `.claude/archive/session-log-2026-04-24-to-2026-04-24.md`, `.claude/archive/session-log-2026-04-25-to-2026-04-25.md`, `.claude/archive/session-log-2026-04-27-to-2026-04-27.md`
Most recent entry first. Run `.claude/tools/rotate-session-log.sh` when this grows beyond ~3 sessions.

---

## 2026-05-18 — Session 23: `review` Command Implementation (Phase 2–5 complete)

### Context
Resumed directly from session 22 handoff. Phase 2 was partially done (types.go + queue.go committed); taxonomy.go had a prior Ollama 0-verdict. User said "proceed" after task list was created.

### What Was Done
- **Phase 2 complete** — wrote `taxonomy.go` (BuildTaxonomy: 3-level tree, deterministic sort), `render.go` (placeholder guard + JSON injection), `embed.go` (go:embed TemplateHTML export). taxonomy.go + render.go written directly (escalation after prior 0-verdict); embed.go trivial 3-liner.
- **Phase 3** — `cmd/review.go` cobra command via `my-go-qcoder` (verdict 1; fixed: import alias not needed, `0644`→`0o644`).
- **Phase 4** — 17 unit tests via `my-go-qcoder`: `render_test.go` (verdict 2), `taxonomy_test.go` (verdict 2), `queue_test.go` (verdict 1; fixed: `FieldsPerRecord=-1` in queue.go so our explicit field-count error fires). All green.
- **Phase 5** — `go build/vet/test` all clean; acceptance test `TestReview_ProducesHTMLWithQueueAndTaxonomy` passes; smoke: 349 rows, 23 need review.
- **PR #22 opened** — `worktree-feat+review-command` → `master`.
- **Tracking updates** — `index.md` go-structure table updated (internal/review row added); `session-context.md` updated with `my-go-qcoder` benchmark data and preferred model change.
- **Read `local-model-conventions.md`** per user request; noted that taxonomy.go/render.go were escalations (written directly) and should be retried with pre-defined stubs in future.

### Decisions Made
- **`my-go-qcoder` is now the preferred codegen model** — 4 calls, verdicts 2/2/1/1; replaces `my-go-q25c14` as default. Falls back to `my-go-q25c14` if unavailable.
- **`my-go-qcoder` weakness identified** — struggles with Go intermediate map types when types are NOT pre-defined in context; passes cleanly when types are available as context files.
- **`FieldsPerRecord = -1` pattern** — when you want custom field-count error messages in Go CSV parsing, disable the library's built-in check.
- **tasks.md updated at handoff** — Claude Code TaskCreate/TaskUpdate used during session; tasks.md reflects final state only at session-handoff.

### Next
- Merge PR #22 (RUI-1a).
- Decide next feature: RUI-3 (`apply` command), RUI-4 (emit 3-level path into classified CSV), or deferred retrieval work (5.R1 TF-IDF).
- **Worktree:** `.claude/worktrees/feat+review-command` — clean, can be removed after merge.

---

## 2026-05-18 — Session 22: `review` Command Implementation (Phase 0–2 partial)

### Context
Resumed from Session 21 plan. Resolved open questions O1–O3 before coding. New
directives established this session: implementation in worktree (`feat/review-command`),
tasks.md updated only at handoff (internal task list during session), stop at 150k/75%
context for handoff.

### What Was Done
- **Resolved O1** — raw installment notation (`300,00/24`) kept in `QueueEntry.RawValue`;
  parsed per-installment float in `Value` via `ParseCurrencyWithInstallments`.
- **Resolved O2** — `Planilha_Normalized_Final.xlsx` confirmed has "Referência de
  Categorias" sheet; used via `EXPENSE_WORKBOOK_PATH` env var in acceptance tests.
- **Resolved O3** — `GenerateID` stays DD/MM (no year); document constraint in code;
  `apply` will use full date column for correlation, not the hash alone.
- **Phase 0** — `internal/review/template/review.html` placed (claude.ai/design output
  with sample data replaced by `__REVIEW_DATA__` placeholder); root `review.html`
  kept as dev preview.
- **Phase 1** — acceptance test red:
  - `test/fixtures/review-basic/input.csv` (5 rows: 4 auto-inserted, 1 needs review, 1 installment)
  - `test/verify/html.go` — `HTMLFileEmbeddedJSON` + `HTMLFileContainsScript`
  - `test/actions/commands.go` — `RunReview` appended
  - `test/review_test.go` — `TestReview_ProducesHTMLWithQueueAndTaxonomy` (confirmed red)
- **Phase 2 partial** — `internal/review/types.go` + `queue.go` written and committed.
  `taxonomy.go` Ollama output rejected (verdict 0 — type mismatches throughout);
  `render.go` not started. Both pending next session.
- **RUI-2 complete** — template built by claude.ai/design and placed.

### Decisions Made
- **tasks.md = handoff-only artifact.** Use `TaskCreate`/`TaskUpdate` during session;
  update tasks.md only when running session-handoff skill.
- **Stop at 150k/75% context.** Trigger session-handoff before context degrades.
- **Worktree per feature.** Implementation work lives in `feat/review-command` worktree
  (`worktree-feat+review-command` branch).
- **Verdict scale confirmed as 0/1/2** (not ACCEPTED/IMPROVED/REJECTED — stale reference
  in user-prefs corrected this session).
- **O3 design** — `GenerateID` no-year constraint is acceptable; `apply` uses full date.

### Next
- Phase 2 resume: write `taxonomy.go` (last Ollama attempt verdict 0 — type mismatches
  with intermediate map types; retry with better context or write directly). Then `render.go`.
- Phase 3: `cmd/review.go` cobra wiring.
- Phase 4: unit tests (testify, table-driven).
- Phase 5: build/vet/test/smoke + run against real classified.csv.
- **Worktree:** `.claude/worktrees/feat+review-command`, branch `worktree-feat+review-command`.

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

