# Session Log — Expense Reporter

**Previous logs:** `.claude/archive/session-log-2026-02-27-to-2026-02-27.md`, `.claude/archive/session-log-2026-05-15-to-2026-05-18.md`, `.claude/archive/session-log-2026-05-18-to-2026-05-18.md`, `.claude/archive/session-log-2026-05-29-to-2026-05-29.md`, `.claude/archive/session-log-2026-06-08-to-2026-06-08.md`
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
**Current Session:** 2026-06-11 — Session 28: Workbook generator Phase B — typed entries, max-entries sizing, per-group %, i18n labels + unit tests
**Current Layer:** Workbook Generator — Phase B (data + formula validation), branch `feat/workbook-generator`
Most recent entry first. Run `.claude/tools/rotate-session-log.sh` when this grows beyond ~3 sessions.

---

## 2026-06-11 - Session 28: Workbook generator Phase B — data-bearing builder + unit tests

### Context
Resumed Phase B of the workbook generator from the session-27 handoff, working in the
scratch builder `.claude/scratch/template-builder/`. Tracked plan authored at
`.claude/plans/workbook-generator-phaseB-plan.md` (decisions D1–D7, gotchas, verdict log).

### What Was Done
- **Step 1 (expense_sheet.go):** block data rows = `max(MaxEntries-across-months, 1)` +
  headroom (0); write typed entries — Item, real `time.Date(dataYear, month, day)` dates
  (DD/MM), BRL amounts. Helpers `calculateSubcatBlockRows` / `writeSubcatDataRows`. (qcoder v1.)
- **Step 2 (listas_sheet.go):** per-group `% sobre despesas` (group/sheet-grand, forward
  reference via new `plannedGrandTotalRow`) + `% sobre receita` rows; `saldoBlock` and
  `setHeights` made dynamic (dropped hardcoded rows 61/60/79). Inline after a qcoder timeout.
- **Step 3 (Labels i18n):** wired `newPtBRLabels()` through the builder via a **sonnet
  subagent → my-go-qcoder `patch_file`** (18 calls, all verdict 2). Generic literals →
  `Labels` fields; normalization (lowercase per-group rows, `Porcentagen`→`Porcentagem`).
  main.go/receitas/expense were pre-wired by the user. English field names, pt-BR values.
- **Step 4 (generate + sonnet review):** review caught a **BLOCKING Receitas bug** —
  income blocks still used pre-Phase-B sizing → zero data rows + inverted `SUM(E3:E2)`,
  cascading to every `% sobre receita` denominator and the Saldo. Fixed Receitas to mirror
  the expense pattern (`calculateReceitasBlockRows` / `writeReceitasDataRows`, DD/MM + R$).
  Regenerated the workbook.
- **Unit tests:** added `builder_test.go` (testify, 12 tests) — sizing, MaxEntries,
  `plannedGrandTotalRow`, sheet-ref quoting, label normalization, and in-memory
  build-and-assert integration guards incl. `TestBuildReceitasSumRange` (locks the SUM bug).
  testify added to the scratch `go.mod`.
- Backed up workbook artifacts to `.claude/workbook-template/backup-2026-06-10-pre-phaseB-gen/`.
- Commits: `b72326a` (steps 1–2), `09a4651` (step 3), `ddc0e13` (Receitas fix + tests).

### Decisions Made
- **Block sizing = max-entries-per-month, headroom 0** (workbook is regenerated from the DB,
  never inserted into; zero-entry subcat → 1 row).
- **Phase B validation = generate → review → bless** (not diff-to-zero; max-entries sizing
  shifts row positions, so address-based diffing is non-viable).
- **i18n:** all generic strings in a `Labels` struct, English identifiers + pt-BR values;
  config-file loader deferred but constructor-ready.
- **Receitas income cells now use DD/MM + R$** (the old "General numFmt" note was from when
  rows were empty; a real date in a General cell renders as a serial number) — **NEEDS USER
  CONFIRMATION** at bless.
- **Subagents doing codegen must also route to the local model (Ollama)** — saved to memory.
- **TDD means test-FIRST (red→green)**, not test-after; this session was test-after — a miss,
  corrected in memory. Apply red-first next session, including in the scratch module.

### Next
- **RE-REVIEW the regenerated `.claude/workbook-template/template.xlsx`** with a sonnet review
  subagent — the prior PASS validated the *buggy* file; confirm income rows, `% sobre receita`
  denominators, and the Saldo chain now resolve.
- Present the user a report (work done + decisions) and a **double-check list** for the
  generated template (see reading guide), then bless as the data-bearing golden master.
- Answer 2 open questions: (a) Receitas DD/MM+R$ vs General; (b) Listas section-header fill
  black `000000` vs navy `333399` (Phase A converged to black).
- Gap: Listas section header `"Receitas"` and despesa section names have no `Labels` field yet.
- Then **Phase G:** port the builder → `internal/generate` + `generate-workbook` command,
  acceptance-first + TDD.

## 2026-06-10 - Session 27: Workbook generator — Layer 3 spec + Phase A golden-master convergence

### Context
Started from session 26 handoff: execute workbook-mapping Layer 3 locally per
`.claude/plans/workbook-layer3-instructions.md`. Grew into Phase A of the generator itself.
Branch `feat/workbook-generator`.

### What Was Done
- **Layer 3 executed locally** (claude.ai 2× usage not needed): 7 parallel Sonnet subagents
  produced per-sheet structural digests (`.claude/workbook-dump/digests/`, gitignored);
  main session synthesized `.claude/plans/workbook-generator-spec.md`. Digests surfaced
  source bugs the reconciled findings missed (systematic June SUM-over-Data-column bug;
  Adicionais single-cell SUMs that only count the LAST data row — its totals are wrong).
- **Dogfood template build (Opus subagent, spec-only context):** standalone builder at
  `.claude/scratch/template-builder/` → `.claude/workbook-template/template.xlsx` +
  `ambiguities.md` (14 decisions, 2 spec contradictions found by building).
- **User hand-reviewed the template** → `template-reviewed.xlsx` (golden master). Sonnet
  subagent diffed it → `review-diff.md` (14 correction patterns).
- **Spec v2:** folds in the hand-review. Redesign stances: vertical MERGES replace
  fill-down; months start col C; sub-item level eliminated ("Orion - Consultas" composed
  strings); Mês A1:B2; freeze C3/D4; Listas 3-col label area, months D–O; Referência
  sheet OMITTED (not an insertion target); B7B7B7 moot. Spec wins over original workbook.
- **Convergence (Opus subagent):** builder rewritten to v2; output matches the golden
  master — 41 residuals, all justified golden-master hand-edit artifacts
  (`convergence-report.md`; diff harness `diff.py`).
- **Implementation plan written:** `.claude/plans/workbook-generator-implementation-plan.md`
  — prep reading list, Phase B, Phase G (generate-workbook command), open questions. Sized
  for a Sonnet executor.
- Memories updated (repo + expense-reporter QUICK, expense-reporter KNOWLEDGE generator
  entry, auto-memory workbook-mapping rewritten); index.md registered all artifacts.
- 4 commits on `feat/workbook-generator`.

### Decisions Made
- **Derived layout** — positions computed from taxonomy + entries; dump is validation
  reference only. Source drift/bugs deliberately normalized (spec §6, 14 deviations).
- **Golden-master validation** — convergence judged ONLY by workbook-inspect dump diff +
  openpyxl pass (`diff.py`), never eyeballing.
- **Per-group percent rows required** but absent from golden master — enter in Phase B
  (builder `perGroupPctRows` switch off until then).
- **Subagent fan-out pattern endorsed** — digests/reports to FILES, main session
  synthesizes; Sonnet for extraction/diffing, Opus for build/convergence.
- excelize gotchas: `SetCellFormula` takes no leading `=`; stale-formula fix =
  `UpdateLinkedValue()` + `SetCalcProps(FullCalcOnLoad)`.

### Next
- Follow `.claude/plans/workbook-generator-implementation-plan.md` §0 prep list.
- Phase B first step BLOCKS ON USER: copy golden master → `template-data.xlsx`, hand-fill
  fake entries, add per-group percent rows to Listas.
- Then Phase G: lift inspect core into `internal/inspect`; port builder into
  `internal/generate` + `cmd/generate.go`; acceptance tests FIRST.

### Gotchas
- Templates/builder are tracked (fake data only); dump-* dirs gitignored via
  `.claude/workbook-template/.gitignore`. Source dumps/digests/visual notes stay gitignored.
- Golden master has 6 hand-edit inconsistencies (convergence-report.md) — builder follows
  the spec rule; do NOT "fix" the builder toward them.
- No `jq` on this machine — use python3 (openpyxl available).
- Long-running subagents (>5 min) expire the main session's prompt cache — one full
  re-read per agent return; acceptable, not a bug.

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

