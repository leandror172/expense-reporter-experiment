# Session Log — Expense Reporter

**Previous logs:** `.claude/archive/session-log-2026-02-27-to-2026-02-27.md`, `.claude/archive/session-log-2026-05-15-to-2026-05-18.md`, `.claude/archive/session-log-2026-05-18-to-2026-05-18.md`, `.claude/archive/session-log-2026-05-29-to-2026-05-29.md`, `.claude/archive/session-log-2026-06-08-to-2026-06-08.md`, `.claude/archive/session-log-2026-06-08-to-2026-06-08.md`, `.claude/archive/session-log-2026-06-10-to-2026-06-10.md`
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
**Current Session:** 2026-06-15 — Session 30: generate package internal refactor — styles vocabulary, cross-file extraction, memory merge
**Current Layer:** Workbook Generator — COMPLETE (PR #27 pending merge; internal refactor done)
Most recent entry first. Run `.claude/tools/rotate-session-log.sh` when this grows beyond ~3 sessions.

---

## 2026-06-15 - Session 30: generate package internal refactor — styles vocabulary, cross-file extraction, memory merge

### Context
Continued from a compacted session that had left an uncommitted styles.go refactor in the tree.
Picked up that work, committed it, then ran the cross-file domain-extraction pass discussed
pre-compaction, then the memory handoff. All work in `expense-reporter/internal/generate`,
behavior-preserving (the oracle-frozen acceptance dumps stayed byte-identical throughout).

### What Was Done
- **Committed the styles.go refactor** (carried over uncommitted from the pre-compaction session):
  styles.go split into a style *vocabulary* (named constructors `dataCell`/`grayBanner`/
  `columnHeader`/`totalRowCell`/`navyBand`/… + named palette/numfmt constants) over a
  `styleRegistrar` (first-error capture; `family()` mints General/currency/percent trios);
  Portuguese `styleSet` fields anglicized (MonthCorner; TotalText/TotalTextLeft/TotalValue/
  TotalValueRight); dead styles removed. Plus loader/revenue/`summary.balanceBlock` step-extraction.
- **Cross-file domain-extraction pass (A+B+C+D), via a sonnet subagent (codegen routed to Ollama,
  qcoder verdict 1):** moved pure ref/formula helpers (`cell`, `sheetRef`, `needsQuote`) into
  `util.go`; created new `data_sheet.go` holding the data-sheet writing vocabulary shared by the
  expense sheets and Receitas (`writeMonthHeader`, `writeTotalRow(Opt)`, `writeSeparator`,
  `mergeCategoryLabel`, `freezeC3`); unified two near-duplicate pairs into `calculateBlockRows(row,
  maxEntries)` and `writeDataBand(..., rowHeight, lastCol)` (the only behavioral difference between
  the two sheet kinds is row height — 12.75 expense / 15 revenue). Net −144 lines.
- **Verified green and committed:** 480 unit tests / 19 packages, `go vet` clean, 3/3 generate
  acceptance tests with dumps byte-identical. 3 commits on `feat/workbook-generator` (2 refactor +
  1 chore-memory). Independently verified the subagent's tree before trusting its report.
- **Memory handoff:** merged the session-30 pre-compaction scratch into durable memory (generate
  `.memories/QUICK.md` code-organization section; repo `KNOWLEDGE.md` generator-refactor entry;
  root `QUICK.md` status). Wrote two Claude-side auto-memories (`feedback_style_vocabulary`,
  `feedback_subagent_verify_tree`) + reinforced `feedback_method_extraction`. Deleted both scratch
  files.
- **PR #27 description checked → STALE** (covers only G1–G4; predates the PR-review fixes and the
  session-30 refactor; test counts read 220+/17, now 480/19). Update pending the user's go-ahead.

### Decisions Made
- **Package stays FLAT** — no styles/sheets subpackages (Go idiom; styleSet/layoutRegistry/Labels/
  domain types too coupled to split). Style/config definitions are expressed through identifying
  names (the vocabulary pattern) — name WHAT a thing is, not how it's assembled; generalizes
  beyond styles.go.
- **`internal/taxonomy` split deferred to T-02** (the real-taxonomy export), now with a PREREQUISITE
  discovered this session: `taxonomy.go` mixes the domain types (used by every builder) with mutable
  RENDER config (`dataYear`/`headroomRows`/`perGroupPctRows`, set by `Generate()`, read by builders)
  — those vars must relocate into `generate` first; then decide domain-type placement (cycle risk).
  Recorded as the T-02 addendum in tasks.md.

### Next
- **Update the PR #27 description** (stale — pending user approval to push), then merge PR #27 (T-01).
- Real-taxonomy export (T-02) — now carries the `internal/taxonomy` split + its prerequisite.
- Year-rollover (T-03); then TF-IDF (5.R1).

## 2026-06-11 - Session 29: Workbook generator complete — Phase B blessed, Phase G shipped, PR review addressed

### Context
Resumed from session 28 handoff: re-review the regenerated data-bearing template, settle 2 open
questions, then Phase G. Branch `feat/workbook-generator`. Sonnet pre-approved for subagents.

### What Was Done
- **Phase B closed:** open questions resolved (spec §7.6/§7.7: Receitas numFmt = DD/MM + R$
  uniform; Listas section headers navy 333399 — spec wins over the Phase A black accommodation).
  Sonnet re-review of the regenerated `template.xlsx`: **PASS, 0 blocking defects**
  (`.claude/workbook-template/phaseB-rereview.md` + 8-item double-check list). Its one
  actionable observation (section `% sobre receita` guarded on numerator, not the D9 income
  denominator) fixed. **User blessed the data-bearing golden master.** `RevenueSheet` Labels
  field added (sheet name + Listas pulls + section header + sheet ordering).
- **G1:** `internal/inspect` — dump core lifted verbatim from `cmd/workbook-inspect` (now a
  thin wrapper); output verified byte-identical; unit tests (qcoder, verdict 1).
- **G3 (acceptance-first, advisor-reviewed):** Opus advisor review
  (`.claude/advisor-G3-acceptance-design.md`) reshaped the design — input contract pinned in
  spec §1.1 (taxonomy JSON schema; entries DD/MM no-year; unknown subcat → warn+skip exit 0;
  taxonomy authority on category mismatch); **oracle bootstrap**: scratch builder taught to read
  the fixture (`LoadTaxonomy` loader + flags), its dumps frozen as `expected-dump-*`;
  `verify.WorkbookStructureMatches` compares a NORMALIZED SUBSET (ignores widths/heights/
  manifest source). 3 scenarios born RED on `unknown command`. Fixture
  `test/fixtures/generate-basic/` documented in PATTERNS.md (new sub-format).
- **G2:** `internal/generate` (builder port) + `generate-workbook -o --taxonomy [--entries
  --year --headroom]` cobra command (qcoder, verdict 2) — **all 3 acceptance scenarios green
  on first run**. Loader + builder math tests ported (Phase B fake dataset is now
  `taxonomy_fixture_test.go`).
- **G4:** README command docs; scratch builder marked SUPERSEDED; **PR #27** title/body updated
  to full scope (via REST API — `gh pr edit` hits a projects-classic GraphQL bug on this repo).
- **PR review comments addressed** (2 drafts in the user's still-pending review): identifiers
  → English (Categoria→Category, Receitas*→Revenue*, listas*→summary*, despesa*→expense*,
  saldo*→balance*; files renamed summary_sheet.go/revenue_sheet.go; pt-BR strings only in
  `Labels`); revenueSection/expenseSection extracted into single-responsibility `write*` steps.
- **Latent bug found by the refactor:** summary sections + balance block hardcoded the 4-sheet
  order → invalid `D0`/`E0` refs for smaller taxonomies, PRESENT IN THE FROZEN DUMPS (oracle
  shared the bug). Fix: registry `sheetOrder`; dumps re-frozen with a manually reviewed delta
  (exactly 6 phantom rows + 4 bogus refs removed; other sheets byte-identical).
- Memories updated (root/expense-reporter/test/inspect QUICK+KNOWLEDGE; new
  `internal/generate/.memories/QUICK.md`; auto-memory rewritten). 10 commits, all pushed.

### Decisions Made
- **Taxonomy source = (b) dedicated JSON file** + entries from `expenses_log.jsonl`; option (a)
  Referência read demoted to a possible one-time export tool. JSON over CSV (nested structure).
- **Oracle-frozen expectations** (advisor): freeze the trusted scratch builder's dumps BEFORE
  the port → G2 = converge-to-green. Recorded limit: oracle and port can share a bug —
  acceptance can't see it; on contract changes, re-freeze + manually review the dump delta.
- **`incomeCategories[].name` is block grouping**, not the sheet label (that's
  `Labels.RevenueSheet`, which appears inside formulas → schema identifier, not cosmetic).
- Verbatim code moves go to sed/python, NOT the local model (3 warm timeouts on a 530-line
  transcription; zero design value, pure drift risk). Models kept for synthesis: loader (1),
  loader tests (1 — hallucinated 8 fixture literals despite being given them), verifier (1),
  acceptance tests (2), cobra cmd (2).

### Next
- **Merge PR #27** (user: also submit/discard the pending review — drafts are invisible until then).
- One-time export: real 113-subcategory taxonomy (Referência) → `taxonomy.json`.
- Year-rollover workflow; fate of `apply`/`add` vs generated workbooks. Then TF-IDF (5.R1).

### Gotchas
- `gh pr edit`/`pr view --comments` fail on this repo (projects-classic GraphQL deprecation);
  use `gh api repos/.../pulls/27` REST instead.
- The expenses log stores `date` as `DD/MM` — NO year component; `--year` supplies it.
- excelize `f.NewSheet` returns (int, error); style via NewStyle+SetCellStyle (no SetCellFont);
  `MergeCell` not `MergeCells` — recurring local-model API confusions, all caught by compile.

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

