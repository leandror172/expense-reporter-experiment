# Session Log — Expense Reporter

**Previous logs:** `.claude/archive/session-log-2026-02-27-to-2026-02-27.md`, `.claude/archive/session-log-2026-05-15-to-2026-05-18.md`, `.claude/archive/session-log-2026-05-18-to-2026-05-18.md`
, `.claude/archive/session-log-2026-03-02-to-2026-03-02.md`
, `.claude/archive/session-log-2026-03-13-to-2026-03-02.md`
, `.claude/archive/session-log-2026-03-03-to-2026-03-03.md`
, `.claude/archive/session-log-2026-03-11-to-2026-03-11.md`
, `.claude/archive/session-log-2026-03-13-to-2026-03-13.md`
, `.claude/archive/session-log-2026-03-13-to-2026-03-13.md`
, `.claude/archive/session-log-2026-03-14-to-2026-03-14.md`
, `.claude/archive/session-log-2026-03-18-to-2026-03-18.md`
, `.claude/archive/session-log-2026-03-18-to-2026-03-18.md`
, `.claude/archive/session-log-2026-03-23-to-2026-03-23.md`
, `.claude/archive/session-log-2026-03-27-to-2026-03-27.md`
, `.claude/archive/session-log-2026-04-20-to-2026-04-20.md`
, `.claude/archive/session-log-2026-04-22-to-2026-04-22.md`
, `.claude/archive/session-log-2026-04-23-to-2026-04-23.md`
, `.claude/archive/session-log-2026-04-24-to-2026-04-24.md`
, `.claude/archive/session-log-2026-04-25-to-2026-04-25.md`
, `.claude/archive/session-log-2026-04-27-to-2026-04-27.md`
, `.claude/archive/session-log-2026-05-12-to-2026-05-12.md`
**Current Session:** 2026-06-08 — Session 26: Workbook mapping Layers 1+2 — JSON dump, visual notes, memories
**Current Layer:** Workbook Mapping — Layers 1+2 complete; Layer 3 (generator spec) next
Most recent entry first. Run `.claude/tools/rotate-session-log.sh` when this grows beyond ~3 sessions.

---

## 2026-06-08 - Session 26: Workbook mapping Layers 1+2 (JSON dump, visual notes, memories)

### Context
Resumed from session 25 to execute the workbook-mapping plan. Ran as TWO parallel Claude Code
sessions: this one owned Layer 1 (JSON dump) + coordination + memories; a parallel session owned
Layer 2 (visual notes). Entry point: `.claude/plans/workbook-mapping-plan.md`.

### What Was Done
- **Layer 1 — `cmd/workbook-inspect` rewritten to JSON** (PR #25, branch
  `feat/workbook-inspect-json-dump`, stacked on #24). Emits `manifest.json` + per-sheet JSON into
  `.claude/workbook-dump/` (gitignored — real expense values). Per cell: value, formula, style
  (bgColor/bold/4 borders). Per sheet: dims, col widths, row heights, merged cells, cross-sheet
  refs (parsed from formulas, handles `'Quoted Name'!Ref`). Per row: `rowType` classifier
  (header-month/header-col/total-row/data-row/category-label/separator). Extraction core via
  `my-go-qcoder` (verdict 1); cross-ref parser, classifier, and fixes written directly.
- **Row-level black separator fills** (commit `906a99d`) — otherwise-empty rows with a row-level
  fill (solid-black category dividers) were a cell-level blind spot; now probed via the
  GetCellStyle→row-style fallback and emitted as `rowType:"separator"` + `rowFill`. Verified
  against catalogued gaps; corrected two visual inferences (Fixas has one black band not seven;
  Receitas has two the visual pass missed).
- **Layer 2 — visual notes** for all 7 sheets in `.claude/workbook-visual-notes.md` (gitignored):
  frozen panes, rendered-vs-hex colors, fonts, number/date formatting.
- **Memories** — new `internal/excel/.memories/KNOWLEDGE.md` (workbook structural map), new
  `cmd/workbook-inspect/.memories/QUICK.md`, refreshed excel/repo/root QUICK pointers,
  `.claude/index.md` registration. Project memory `project_workbook_mapping.md` updated.
- **Layer 3 cowork brief** — `.claude/plans/workbook-layer3-instructions.md`.

### Decisions Made
- **Merges are sheet-specific** (corrected an earlier wrong "no merges anywhere"): expense sheets
  (Fixas/Variáveis/Extras/Adicionais) are merge-free fill-down; Receitas/Listas/Referência merge.
- **Two sheet families, per-sheet palette + fonts** — not workbook-wide. Generator must branch.
- **Listas de itens PULLS, not SUMs** (references source totals directly, e.g. `Fixas!F19`);
  **Referência is the row-mapping source of truth** (ODS dot-notation reference strings inside
  CONCATENATE, not real `!` refs).
- **Layer 3 should run locally in Claude Code, not claude.ai** — feeding the raw dump to the web
  would upload real financial data; local keeps the project's local-first privacy. claude.ai is a
  fallback requiring a sanitized (values-stripped) dump.

### Next
- Workbook mapping Layer 3 — produce `.claude/plans/workbook-generator-spec.md` per
  `.claude/plans/workbook-layer3-instructions.md`.
- Resolve the Referência `D9E1F2` render-vs-hex discrepancy (theme remap or conditional format).

### Gotchas
- `.claude/workbook-dump/`, `.claude/workbook-visual-notes.md`, `.claude/workbook-screenshots/`
  are all gitignored (real expense values) — never commit; Layer 3 inputs stay local.
- The dump iterates `GetRows`, so it is a faithful style map only up to the last valued cell per
  row (trailing empties dropped); row-level fills are captured separately via the separator probe.
- `cmd/workbook-inspect` takes the workbook path as an arg and does NOT read config.json.
- PR #25 is stacked on #24 — retarget to `master` once #24 merges.

## 2026-06-08 - Session 25: apply Phase 4 smoke, workbook fixes, mapping plan

### Context
Started by merging PR #23 (apply command). Ran Phase 4 smoke against a real `reviewed.json`
(349 entries from May–Dec 2025). Iteratively fixed issues surfaced by the smoke test.

### What Was Done
- **Phase 4 smoke completed** — 347 rows inserted live into `Planilha_Normalized_Final.xlsx`
- **PR branch `fix/apply-dry-run-unallocated`** opened with 5 commits:
  1. `fix(apply)`: dry-run now reports correct counts (was always 0/0)
  2. `fix(apply)`: surface uninsertable rows in summary (two silent drop points surfaced)
  3. `fix(excel)`: trim whitespace in subcategory boundary comparisons — trailing spaces
     in workbook cells (e.g. `"Escritório "`) caused false "section full" errors
  4. `feat(apply)`: `--backup` flag added (uses `batch.BackupManager`, same as `batch`)
  5. `feat(apply)`: `--verbose` lists skipped and pending entries with item/date/value
- **Formula recalculation fix** — `setFullCalcOnLoad` now also calls `f.UpdateLinkedValue()`
  before save; `FullCalcOnLoad` was Excel-only, LibreOffice/Google Sheets were showing
  stale cached values
- **`workbook-inspect` tool** created at `expense-reporter/cmd/workbook-inspect/main.go`;
  initial structural map saved to `.claude/workbook-map.md` (1,237 lines, all 7 sheets)
- **3-layer workbook mapping plan** written to `.claude/plans/workbook-mapping-plan.md`
- **`internal/excel/.memories/QUICK.md`** created — documents reference sheet columns,
  boundary trim fix, and two-reader.go disambiguation

### Decisions Made
- **Workbook-generate direction confirmed** — long-term: treat `classifications.jsonl` +
  `expenses_log.jsonl` as source of truth, generate workbook from scratch. Insertions
  kept for now as fallback.
- **3-layer mapping approach:** Layer 1 = full JSON dump with cell styles; Layer 2 =
  Chrome screenshots via Google Sheets; Layer 3 = claude.ai synthesis (2× usage expires
  2026-07-05 — prioritise before then)
- **`Referência de Categorias` columns D and F are dead** — `RowNumber`/`TotalRow` loaded
  but never read back in any command. New entries only need columns A, B, C.

### Next
- Open PR for `fix/apply-dry-run-unallocated` and merge
- Execute workbook mapping: Layer 1 first (JSON dump drives Layer 2 screenshot targeting)
- Use claude.ai 2× usage (expires 2026-07-05) for Layer 3 spec synthesis
- Decide: RUI-4 (3-level path in CSV) or 5.R1 (TF-IDF) after mapping work

### Gotchas
- `go run ./cmd/expense-reporter` fails to load config — binary must be built first
  (`go build -o expense-reporter ./cmd/expense-reporter`) so it sits alongside `config/`
- `backupFlag` is a package-level var in `batch.go`; `apply.go` uses `applyBackup` to avoid collision
- Two `reader.go` files: `internal/excel/reader.go` (workbook I/O) vs `internal/apply/reader.go`
  (reviewed.json parser) — always disambiguate

## 2026-05-29 — Session 24: `apply` Command Phase 3 (implementation complete, PR #23)

### Context
Resumed from `.claude/handoff-apply-phase3.md` (written same day — Phases 0–2 done, Phase 3 cut short by limits). Full orientation read before coding: all key context files + advisor call (output at `.claude/advisor-apply-phase3.md`).

### What Was Done
- **Advisor review** — key findings: lazy workbook validation required (test has no workbook), model-from-prior critical for corrected+found path, no taxonomy loading needed, `insertNewRows` is entirely uncovered by the acceptance test.
- **Ollama codegen for cmd/apply.go:**
  - `my-go-qcoder`: 3× TIMEOUT_COLD_START (30b + 14 context files exceeded 300s window)
  - `my-go-g3-12b`: verdict 0 — wrong package refs throughout (`review.*` instead of `apply.*`), broken decision table, early workbook validation
  - Escalated to Claude (beyond retry budget: 3 timeouts + 1 rejected)
- **cmd/apply.go written directly** — 287 lines, 9 functions; correct decision table, lazy workbook validation, batch excel APIs, `prior.Model` for corrected+found feedback entries, "review" sentinel only for new rows.
- `go build + go vet` — clean; `go test ./...` — 452 tests passing.
- `TestApply_IdempotencyAndFeedback` — PASS (4/4 assertions, 10ms).
- **PR #23 opened** — `feat/apply-command` → `master`.

### Decisions Made
- **Lazy workbook validation** — validate/open workbook only when `len(newRows) > 0`; error clearly if workbookPath empty in that case. Required by acceptance test (no workbook path configured).
- **`insertNewRows` is a blind spot** — zero test coverage on the insertion path; Phase 4 smoke against real `reviewed.json` is the only behavioral check.
- **Ollama context lesson** — 30b model + 14 large context files exceeds 300s. For complex multi-function files: prefer stubs-then-Ollama or accept Claude escalation early. Also: including `review.go` as context caused g3-12b to misidentify the `apply` package as `review` — disambiguate explicitly in the prompt when package names are similar.

### Post-handoff fixes (same session, second advisor review)
- **Bug 1 (high)** — index-aliasing in `buildExpenseBatch`: looked up `targetRows[newRowsIndex]` but `AllocateEmptyRows` keys by emptyReqs position. When any row is skipped in `buildEmptyRowRequests`, indices diverge → wrong row written, valid row silently dropped. Fixed: iterate emptyReqs by position, use `req.ExpenseIndex` to get back to the original entry.
- **Bug 2 (medium)** — `--dry-run` still wrote both JSONL logs; `writeFeedbackForNewRows` ran unconditionally. Fixed: gate behind `!dryRun`.
- PR #23 description updated with insertion-path caveat and bug-fix notes.

### Next
- **Phase 4 smoke**: run `apply` against real `reviewed.json` from a prior review session (index bug fixed; exercises the now-correct insertion path).
- **Review/merge PR #23**.
- **Decide next feature**: RUI-4 (emit 3-level path into classified CSV) or 5.R1 (TF-IDF retrieval layer).

---

