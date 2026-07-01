# internal/generate — Knowledge (Semantic Memory)

*Accumulated design decisions for the workbook generator. Read on demand by agents.*

<!-- ref:generate-build-order -->
## Build Order and the layoutRegistry (generate.go / layout.go)

`buildWorkbook` builds sheets in a fixed order: Receitas → expense sheets → Listas (summary).
This is not arbitrary — it is a solution to a chicken-and-egg problem.

Listas (`summary_sheet.go`) does not compute any cell positions itself. Instead it reads
`layoutRegistry`, which holds the total-row position of every subcategory block and every
Receitas block. Those positions are only known *after* the source sheets are rendered.

The flow:
1. `buildRevenueSheet` and `buildExpenseType` each call `calculateBlockRows` to determine
   where each block's total row lands, then record it into `reg.revenue.Blocks` /
   `reg.expense[name]`.
2. Only after all source sheets are done does `buildSummarySheet` run — it reads the
   registry to emit cross-sheet `=Fixas!AL17` pull-formulas in Listas.

**Implication:** any future sheet type that Listas needs to reference must be built
before `buildSummarySheet` is called, and must record its total rows in `reg`.

**Ref:** `[ref:generate-build-order]`
<!-- /ref:generate-build-order -->

<!-- ref:generate-row-counter -->
## The Accumulating Row Counter in summaryBuilder (summary_sheet.go)

`summaryBuilder` has a single `b.row int` field, starting at 6, that never resets.
Every method that emits rows increments it. The full write sequence is:

```
b.header()              → rows 3,5 (month banner + "Valor" header)
b.revenueSection()      → starts at row 6; emits per-Block groups + Investimentos shell + pct
b.expenseSections()     → one band per sheet (Fixas/Variáveis/…); each calls b.expenseSection()
b.balanceBlock()        → grand totals + per-sheet % breakdown + final balance row
```

There are no hardcoded row numbers anywhere in `summary_sheet.go`. Every absolute row address
is derived from `b.row` at the moment it is written. Cross-sheet formula references
(`=Fixas!AL17`) use `TotalRow` values read from `layoutRegistry` — those are hardcoded at
registration time in `expense_sheet.go` / `revenue_sheet.go`, but the Listas *pull-row*
numbers are all derived from `b.row`.

**Why this matters for tests:** if `revenueSection()` emits N rows, the first expense
section starts at row `6 + N`. A change that adds or removes even one row in
`revenueSection()` shifts every pull-row, grand-total row, and balance-block row that
follows — making all downstream Listas cell addresses wrong. See
`[ref:generate-oracle-coupling]` for what this means for acceptance tests.

**Ref:** `[ref:generate-row-counter]`
<!-- /ref:generate-row-counter -->

<!-- ref:generate-oracle-coupling -->
## Cross-Fixture Oracle Coupling (summary_sheet.go / test fixtures)

The acceptance suite has two generate fixtures:
- `test/fixtures/generate-basic/` — skeleton taxonomy, flat income (Block==Label), no real income entries
- `test/fixtures/generate-income/` — nested `incomeCategories` taxonomy + `income-entries.jsonl`

Both fixtures freeze oracle dumps for the Listas sheet (`expected-dump-data/Listas de itens.json`).
Because `summaryBuilder` uses a single accumulating `b.row` counter (`[ref:generate-row-counter]`),
any change to `revenueSection()` row count cascades:

- `generate-income` oracle: the income block has real multi-Block groups, so its Listas is
  larger; its oracle captures that exact row layout.
- `generate-basic` oracle: the income block is minimal (flat, one block per label), but
  `revenueSection()` still runs and emits its minimum row count. A change to how many rows
  the Investimentos shell, band, or grand total rows occupy shifts all expense sections
  and the balance block even in the basic fixture.

**Rule: when touching `revenueSection()`, `expenseSections()`, or `balanceBlock()` in
`summary_sheet.go`, ALWAYS re-freeze BOTH `generate-basic` AND `generate-income` Listas
oracles, even if the change is income-only.**

The re-freeze procedure: rebuild with `generate-workbook`, run `workbook-inspect` on the
output, copy the new dump files into `expected-dump-data/`, then MANUALLY REVIEW the diff
— acceptance cannot distinguish "both fixed" from "both broken in the same way".

**Expense sheet oracles are NOT affected** by summary changes. `Fixas.json`,
`Variáveis.json` etc. are produced by `expense_sheet.go` / `revenue_sheet.go` which have
their own independent row counters. A Listas layout change does not shift any row in those sheets.

**Ref:** `[ref:generate-oracle-coupling]`
<!-- /ref:generate-oracle-coupling -->

<!-- ref:generate-colmap -->
## Column Layout — Data Sheets and Listas (colmap.go)

All data sheets (Fixas, Variáveis, Receitas) use an identical 3-column-per-month layout
starting at col C (`dataMonthBase = 2`, 0-indexed):

```
A: category label (merged)   B: subcategory label (merged)
C: Jan Item   D: Jan Data   E: Jan Valor
F: Fev Item   G: Fev Data   H: Fev Valor
…             (12 × 3 = 36 columns)         …AL: Dez Valor
```

`expenseMonthCols(k)` → `(item, data, valor)` for month k.
`expenseValorCol(k)` → just the valor column (used for Listas pull-formulas and SUM totals).

Listas uses a different layout: A–C are label columns, D–O are month columns (one per month,
no Item/Data sub-columns):
```
A: section label (merged)   B: block/category label   C: subcategory label
D: Jan   E: Fev   …   O: Dez
```
`summaryMonthCol(k)` → column letter D–O for Listas.

**The two `revenueMonthCols` / `revenueAmountCol` aliases** in `colmap.go` are intentional
no-ops (they delegate to `expenseMonthCols`/`expenseValorCol`). They exist as named
documentation that Receitas shares the data-sheet column model — the v2 spec unified them
(v1 had Receitas with a different layout).

**Ref:** `[ref:generate-colmap]`
<!-- /ref:generate-colmap -->

<!-- ref:generate-block-sizing -->
## Block Sizing — calculateBlockRows (data_sheet.go)

Every subcategory or revenue block occupies a band of rows:

```
firstData … lastData   (data rows — one per entry slot)
totalRow               (SUM formula row, always lastData+1)
```

`calculateBlockRows(row, maxEntries)`:
- `maxEntries` = busiest month's entry count for that block (0 → treated as 1)
- `lastData = firstData + maxEntries + headroomRows - 1`
- `totalRow = lastData + 1`

`headroomRows` (default 0) is the `--headroom` CLI flag, stored as package-level state.
The design intent is regenerate-don't-insert: no spare rows are needed because the workbook
is always regenerated from the log, not hand-edited.

**Implication:** adding an entry to a month that was previously the max pushes `lastData`
down by one, which shifts the `totalRow`, which shifts everything below it in that sheet's
`layoutRegistry` entries, which shifts the cross-sheet formulas in Listas. This is why
re-freezing oracles is required after loading new real data entries.

**Ref:** `[ref:generate-block-sizing]`
<!-- /ref:generate-block-sizing -->

<!-- ref:generate-package-state -->
## Package-Level State — dataYear, headroomRows, perGroupPctRows (generate.go)

Three package-level vars are set by `Generate()` before each build:

| Var | Default | Source |
|---|---|---|
| `dataYear` | 2026 | `opts.Year` |
| `headroomRows` | 0 | `opts.Headroom` |
| `perGroupPctRows` | true | hardcoded (spec §4.2) |

These are mutable package state, not struct fields, inherited from the scratch builder.
The comment in `generate.go` explains the tradeoff: the CLI is single-shot, so package
state is acceptable. They were relocated from `taxonomy.go` into `generate` as a
prerequisite for the `internal/taxonomy` extraction (cycle-free split).

**`perGroupPctRows = true`** means each category band in an expense section emits two
extra rows below its group total: `% sobre despesas` and `% sobre receita`. These rows
are counted by `plannedGrandTotalRow` in `summary_sheet.go`, which must know the total
row count of a section *before* writing it (because the per-group pct rows reference the
section's grand total, which comes at the end of the loop).

**Ref:** `[ref:generate-package-state]`
<!-- /ref:generate-package-state -->

<!-- ref:generate-excelize-gotchas -->
## excelize Gotchas (generate.go, data_sheet.go)

- **`SetCellFormula` takes NO leading `=`**: write `"SUM(C3:C5)"`, not `"=SUM(C3:C5)"`.
  Silent wrong output if you include the `=`.
- **Stale formula fix:** after all sheets are built, `Generate` calls
  `f.UpdateLinkedValue()` then `f.SetCalcProps(&excelize.CalcPropsOptions{FullCalcOnLoad: &yes})`.
  Without this, cross-sheet pull formulas in Listas show stale zero values when the file
  is first opened.
- **Sheet name quoting in formulas:** Portuguese sheet names contain spaces and accented
  characters ("Listas de itens", "Variáveis"). `sheetRef()` in `util.go` calls `needsQuote()`
  which requires quoting for any non-ASCII-alphanumeric character. Unquoted refs to these
  sheets cause a parse error in Excel/Sheets.
- **`MoveSheet` order:** `orderSheets` walks the desired order in *reverse* because
  `MoveSheet(source, target)` inserts `source` *before* `target`. Reversing gives the
  correct final order (Listas → Receitas → expense sheets).
- **Date cells:** entries are written as `time.Time` via `SetCellValue`. Combined with the
  `st.DateCell` style (numfmt `DD/MM`), this renders as a Brazilian date without string
  formatting. Changing to a string would break sorting and date arithmetic in the workbook.

**Ref:** `[ref:generate-excelize-gotchas]`
<!-- /ref:generate-excelize-gotchas -->

<!-- ref:generate-income-symmetry -->
## Income — 3-Level Model Symmetric with Expenses (WS-C, session 38)

Income in the workbook has a 3-level hierarchy: Category → Block → Label (leaf subline).
This mirrors the expense hierarchy: Sheet → Category → Subcategory.

`taxonomy.RevenueBlock{Category, Block, Label, Months [12][]Entry}` is the leaf type.
`revenue_sheet.go` groups blocks by `Block` field, mirroring how `expense_sheet.go` groups
subcategories by `Category`:

| Expense sheet | Revenue sheet |
|---|---|
| Col A: Category label (merged across subcats) | Col A: Block label (merged across sublines) |
| Col B: Subcategory label (merged across data+total) | Col B: Label (subline; merged across data+total) |
| Total row per subcategory | Total row per subline |
| Separator between categories | Separator between Block groups |

`summary_sheet.go` `writeRevenueBlockGroups` mirrors `expenseSection`: for each Block group,
it emits one pull row per subline (col C = Label; D–O = `=Receitas!<valorCol><totalRow>`),
merges col B with the Block name, then writes a "Total <Block>" group-total row. The Receitas
grand total is `sumList` of the per-Block totals (non-contiguous cells) recorded in
`revenueTotalRow`.

**Dual-format income taxonomy** (`internal/taxonomy`): the loader accepts both:
- Flat: `blocks: ["Salário", "Freelance"]` → Block==Label for each
- Nested: `blocks: [{block: "Trabalho", sublines: ["Salário", "Freelance"]}]`

`generate-basic` fixture uses the flat form (Block==Label) for dual-format coverage.
`generate-income` fixture uses the nested form with real multi-Block groups.

**Ref:** `[ref:generate-income-symmetry]`; see also `[ref:taxonomy-identity-key]`
<!-- /ref:generate-income-symmetry -->

## Code Organization Conventions (sessions 30–31, consolidated from QUICK.md 2026-07-01)
Behavior-preserving refactor; oracle dumps unchanged.
- **styles.go = vocabulary + registration.** Named constructors say WHAT a cell is
  (`dataCell`/`centeredLabel`/`grayBanner`/`columnHeader`/`totalRowCell`/`navyBand`/
  `fillOnly`) over named palette + numfmt constants; `styleRegistrar.family(fill,font)`
  mints the General/currency/percent trio. Never inline a raw `&excelize.Style{...}` in a
  sheet builder — extend the vocabulary. `styleSet` fields are English (MonthCorner;
  TotalText/TotalTextLeft/TotalValue/TotalValueRight — "Text" = non-currency total-row
  cells, not "Date").
- **File homes = domain, not first caller.** `util.go` = pure string/formula/ref helpers,
  no excelize (`cell`, `sheetRef`, `needsQuote`, `sumList/Range`, `lower`, `atoi`).
  `data_sheet.go` = the data-sheet vocabulary shared by expense sheets AND Receitas
  (`writeMonthHeader`, `writeTotalRow(Opt)`, `writeSeparator`, `mergeCategoryLabel`,
  `freezeC3`, unified `calculateBlockRows` + `writeDataBand`).
- **One sizing/band path for both sheet kinds:** the ONLY behavioral difference is row
  height (12.75 expense / 15 revenue). `Subcat` and `RevenueBlock` both expose
  `Months [12][]Entry` + `MaxEntries()`.
- **Method-extraction convention:** sheet-builder bodies read as named delegated steps
  ≤~15 lines (exemplars: summary_sheet.go `revenueSection`/`balanceBlock`); inline step
  comments promote to doc comments.
- **Package stays flat** (Go idiom; styleSet/layoutRegistry/Labels too coupled to
  subpackage). The one penciled split is DONE (2026-06-16, commits 07f395a + 21c6d4e):
  domain types + loader → `internal/taxonomy`; render config (`dataYear`/`headroomRows`/
  `perGroupPctRows`) relocated into `generate` first (cycle-free).
- **Income symmetry (WS-C, session 38):** `buildRevenueSheet` mirrors `buildExpenseType`
  (Block groups ≙ Categories, col-A merge, per-leaf total rows, separators);
  `writeRevenueBlockGroups` mirrors the expense category rollup; Receitas grand total =
  `sumList` of per-Block totals (non-contiguous) recorded in `revenueTotalRow`;
  `revenueBlockTotal` gained a `Block` field. `generate-basic` deliberately kept FLAT
  (Block==Label) for dual-format coverage.
- **Block sizing** = max-entries-per-month + headroom (default 0); single-row SUMs valid.
- Phase B fake dataset lives in `taxonomy_fixture_test.go`.
