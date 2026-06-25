## 2026-06-24 - Session 38: WS-C income route — DONE + validated on real data (incl. WS-0b month fix)

### Context

Continued the retire-insertion pivot from session 37. Plan said WS-C (income/revenue route) was next; all prerequisites (WS-0b `income_log.jsonl`, revenue taxonomy proposal) already existed. Subagent-driven, acceptance-first, local-first codegen.

### What Was Done

- **WS-C income route — COMPLETE (commits `50d09f7`…`0b28ee5`), acceptance-green, validated on real data.** Acceptance-first: RED `TestGenerateWorkbook_IncomeRoute` (new `generate-income` fixture) authored before any production code.
- 3-level income model: `RevenueBlock{Category,Block,Label,Months}` (one block == one subline leaf), 3-segment `incomePath`. Dual-format loader parses legacy flat `blocks:["Salário"]` (→ Block==Label) AND nested `blocks:[{block,sublines:[…]}]` via custom `rawIncomeBlock.UnmarshalJSON`.
- Separate income scan (`loadIncomeEntries`/`scanIncomeEntries`) from a new `--income-entries` flag, reading the extractor `income_log.jsonl` schema (`income_category`=block, `income_label`=leaf, `item_note`=Item), routed by a block+label index, values SIGNED. Threaded through `LoadTaxonomy(...incomeEntriesPath...)` + `generate.Options`.
- 3-level Receitas render (`buildRevenueSheet` mirrors `buildExpenseType`: Block col-A merge, subline col-B + per-leaf total) and per-Block Listas rollup (`writeRevenueBlockGroups` mirrors expense category rollup: "Total <Block>" + grand total = `sumList` of block totals). Merged nested `incomeCategories` into real `config/taxonomy.json` (gitignored).
- Froze `generate-income` data-bearing oracle; re-froze `generate-basic` data+skeleton oracles (income summary shifted Listas). `generate-basic` deliberately kept FLAT for dual-format coverage.
- **Real-data proof + WS-0b fix:** generating from real 2024 data exposed 175/179 income rows were dateless — `extract_income.py` `fmt_date` discarded the (always-known) band month when the source day cell was blank. Fixed (blank day → `01/MM/YYYY`) + re-extracted (0 null dates). WS-C hardened to SKIP a dateless income row with a loud stderr count instead of aborting. Real 2024 regen: exit 0, 0 income warnings, 101/101 records placed, Receitas populated.

### Decisions Made

- Income summary rollup = **per-Block symmetric** (subline pulls under "Total <Block>"), matching the locked 3-level-symmetric decision and the expense-category structure (not flat per-subline).
- `generate-basic` kept on the **legacy flat** income format on purpose, to retain dual-format/legacy-path test coverage (user: "keep flat, but I may want to change that later").
- Month-loss fixed at the **WS-0b extractor** (not WS-C) — WS-C correctly can't invent a month; dateless rows skip loudly as a guardrail.

### Next

- **WS-B** — convert commands to log-append: `add`/`auto`/`batch-auto`/`apply` stop touching the workbook, append to `expenses_log.jsonl` via the apply writer; entry contract carries the full typed path.
- Then **WS-D** (retire bare-name fallback, gate on type-less count ~0; income now has its own `incomePath` route) and **WS-E** (delete dead insert code: `internal/workflow`, `internal/excel` write side, batch insert path).
- Deferred: promote merged `expenses_log-allyears.jsonl` to canonical + retire per-year split (user's call).

### Gotchas

- A dump-review subagent returned a false NEEDS-FIX on the income oracle due to row-index confusion (claimed off-by-one bugs that did not exist). Direct inspection of the dump JSON overturned it — workbook dumps need careful row accounting; verify subagent dump claims against the raw JSON before acting.
- `my-go-qcoder` was intermittently returning zero-bytes/hanging while VRAM-loaded (genuine failures, not cold-start). Fallbacks: `my-go-g3-12b` / `my-go-q25c14`, then hand-write after 2nd reject. One step landed clean on qcoder (verdict 2) once warmed.
- Changing the income/summary render re-freezes BOTH `generate-income` AND `generate-basic`'s data+skeleton oracles (income summary shifts Listas rows/dimensions).
