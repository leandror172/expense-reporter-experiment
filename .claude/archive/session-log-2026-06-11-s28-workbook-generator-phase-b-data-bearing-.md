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

