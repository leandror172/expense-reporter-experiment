# Workbook Generator — Phase B Execution Plan

**Branch:** `feat/workbook-generator` · **Scope:** scratch builder at `.claude/scratch/template-builder/`
**Status:** IN PROGRESS · **Authority:** spec v2 (`.claude/plans/workbook-generator-spec.md` §3.2, §4.2, §4.4)
**Created:** 2026-06-10

---

## Goal

Make the scratch builder emit a **data-bearing** workbook (real typed entries, correct block
sizing, per-group percent rows, centralized labels) that the user reviews and blesses as the
new golden master. Phase A converged the *layout/styling*; Phase B adds *data + formulas + i18n*.

## Method (changed from Phase A)

No longer diff-to-zero against a fixed-row template. The corrected max-entries sizing rule
**intentionally shifts row positions**, so an address-based diff is not viable. Instead:

> **generate → user reviews → bless as new golden master** → then diff-lock for regression.

## Settled decisions (do not relitigate)

| # | Decision |
|---|----------|
| D1 | Block data rows = `max(sub.MaxEntries(), 1)`; `headroomRows = 0` (regenerate-from-DB, never insert). |
| D2 | i18n: all generic strings in a `Labels` struct. **Field names English, values pt-BR.** Config loader deferred but constructor-ready (`newPtBRLabels()`). |
| D3 | Labels normalized: sentence-case, singular/plural unified, sheet-name suffix dropped from per-group rows; `Porcentagem` typo fixed. |
| D4 | Per-group `% sobre despesas` (grp/sheetGrand fwd-ref) + `% sobre receita` (grp/receitasTotal) rows ON (`perGroupPctRows = true`). |
| D5 | Taxonomy source = YAML file; reading-from-file deferred (constructor hardcodes for now). |
| D6 | R6 (Orion em-dash) / R7 (Extras dup B/C labels) are justified hand-edit residuals — builder does NOT replicate them; name in diff allowlist. |
| D7 | Workbook read/review at generate step routed to a **sonnet subagent** (keep raw cell dumps out of main context). |

## Gotchas

- `generate_code` `output_file` resolves **relative paths from the LLM repo root** — always pass an **absolute** path for this repo.
- Ollama timed out last session on the 3-context-file `expense_sheet.go` task → `TIMEOUT_RETRY`, not a 0 verdict. Mitigation: warm model, split the change, raise `timeout`.

---

## Current state (verified 2026-06-10)

- `go build ./...` / `go vet ./...` both EXIT=0 — compiles, but logic is **stale** (green ≠ correct).
- **DONE:** `labels.go` (struct + `newPtBRLabels()`, verdict 2, currently unreferenced); `taxonomy.go` (Entry/Months/MaxEntries, consts, small Jan/Feb dataset).
- **STALE / TODO:** `expense_sheet.go` (zero-row blocks via `headroomRows-1`, writes no entry values, hardcoded `"Total"`/`"–"`); `listas_sheet.go` (per-group rows are a `b.row++` placeholder; `saldoBlock()` hardcodes `b.row = 61`; literal labels); `main.go` (never constructs/threads `Labels`).

---

## Steps

### Step 1 — `expense_sheet.go`: max-entries sizing + typed entries  [task #3]  ← START HERE
**Route:** local model (`my-go-qcoder`), absolute `output_file`, raised timeout. Split if needed.
- **1a Sizing:** each subcat block data rows = `max(sub.MaxEntries(), 1)`; `lastData = firstData + n - 1`; `totalRow = lastData + 1`. Thread `sub` (not just `sub.Name`) into `writeSubcatBlock`.
- **1b Typed entries:** for each month `k`, for each `Entry` in `sub.Months[k]`, write Item (item col), Date = `time.Date(dataYear, time.Month(k+1), entry.Day, 0,0,0,0, time.UTC)` (date col, `DD/MM`), Value (valor col, `R$`). Lighter months leave a **merged headroom tail** above the total (col B merge already spans data+total).
- **Verify:** `go build`; a 3-entry month → 3 data rows; SUM range spans all data rows.

### Step 2 — `listas_sheet.go`: per-group percent rows + dynamic saldo  [task #4]
- Replace `if perGroupPctRows { b.row++ }` (line 230-232) with two real rows: `% sobre despesas` = `IF(sheetGrand>0, grpTot/sheetGrand, 0)` (forward ref to grand row) and `% sobre receita` = `IF(recTot>0, grpTot/recTot, 0)`. Style `GroupTotalPct` (CCCCFF 0.00%); B/C label style.
- Make `saldoBlock()` start from current `b.row` (drop hardcoded `61`) — per-group rows shift everything down. Verify the row-3.75 thin separator (`setHeights` r==60) still lands sensibly or make it dynamic too.

### Step 3 — Wire `Labels` through the builder  [task #2]
- `main.go`: construct `lbl := newPtBRLabels()`, thread into `buildExpenseSheet`, `buildReceitas`, `buildListas` (and helpers).
- Replace literals: `"Mês"/"Item"/"Data"/"Valor"`, `"Total"/"–"`, `"Total "+cat`, `"Total despesas "+lower(sheet)`, `"Investimentos"`, `"% sobre Receita"`→`PctOfRevenue`, `"Porcentagen da Despesa"`→`ExpenseShareHeader`, `"Porcentagen da Renda"`→`IncomeShareHeader`, `"Despesas fixas/…"`, `"Receita"`, `"Total Renda"`→`TotalIncome`, `"Total Despesas"`→`TotalExpenses`, `"Saldo"`, `monthNames[k]`→`lbl.MonthNames[k]`.
- Note D3 normalization fixes the `Porcentagen`→`Porcentagem` and singular/plural along the way.

### Step 4 — Generate + review  [task #5]
- `go build` + run builder → produce workbook to a gitignored path.
- Hand to **sonnet subagent** to dump/diff (SUMs, pulls, percents, saldo chain, numFmts, merged tails, block sizes) — return findings only, not raw cells.
- On bless: save as data-bearing golden master; commit per green step. Keep R6/R7 in allowlist.

---

## Verdict log (local model calls)
- 2026-06-10 · `my-go-qcoder` · **Step 1** expense_sheet.go sizing+typed-entries · **verdict 1** (~1900 est. tokens saved) · clean decomposition (`calculateSubcatBlockRows`, `writeSubcatDataRows`, threaded `sub`); one mechanical slip `f.SetCellDate`→`f.SetCellValue` (excelize v2 takes time.Time directly), fixed inline. Build+vet green.
- 2026-06-10 · `my-go-qcoder` · **Step 2** listas per-group pct + dynamic saldo · **TIMEOUT_RETRY ×1** (300s, large full-file rewrite; model stayed loaded → latency not eviction) → **escalated to inline** per prompt-cost tiebreaker (change fully pre-specified; 2nd 300s wait not worth it). Added `plannedGrandTotalRow`, `emitGroupPctRows`, `groupPctRow`; `saldoSepRow` field; `saldoBlock`/`setHeights` made dynamic. Build+vet green. **Invariant:** loop advances b.row by `len(Subs)+1+2` per cat = exactly `plannedGrandTotalRow`'s sum — keep in lockstep if per-cat row shape changes.
- 2026-06-10 · `my-go-qcoder` (via **sonnet subagent**) · **Step 3** wire Labels through builder · **18× `patch_file` calls, all verdict 2**, 0 timeouts. Only `listas_sheet.go` needed changes (main.go/receitas_sheet.go/expense_sheet.go pre-wired by user). All generic literals → Labels fields; build+vet green; no live-call-site literals remain. **Gap:** Listas section header `"Receitas"` (and despesa section names) have no Labels field yet — left as-is (structural); candidate for a future `SectionReceitas`/section-label addition.
- Committed: `b72326a` (steps 1-2), `09a4651` (step 3).
- 2026-06-10 · **Step 4 generate + sonnet review** → review caught **BLOCKING** Receitas bug (pre-Phase-B sizing left → zero income rows, inverted `SUM(E3:E2)`, cascaded to all % sobre receita + Saldo). Expense sheets / Listas formulas / sizing / per-group % / labels all PASS. Minor: Listas section-header fill is `000000` (styles.go SectionLabel) vs review's expected `333399` — **OPEN question** (Phase A converged to black; confirm with user). Dólar row absent = deferred per spec §7.5.
- 2026-06-10 · `my-go-qcoder` · **Receitas fix** (mirror expense pattern: `calculateReceitasBlockRows` + `writeReceitasDataRows`, DD/MM + R$) · **verdict 1** (~1500 est. saved) · one mechanical unused-var, fixed inline. **Decision:** Receitas data cells now use DD/MM + R$ (was General when rows were empty) — flag for user confirm at bless.
- 2026-06-10 · **Unit tests added** (`builder_test.go`, testify; testify added to scratch go.mod) — 12 tests incl. `TestBuildReceitasSumRange` regression guard. build+vet+test green. Workbook regenerated. Committed `ddc0e13`.
