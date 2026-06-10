# Workbook Generator — Phase A Convergence Report

Builder: `.claude/scratch/template-builder/` (excelize v2.10.1) → generates
`.claude/workbook-template/template.xlsx`. Target: `template-reviewed.xlsx` (golden master).

Convergence method: `workbook-inspect` JSON dump diff (values, formulas, fills, bold,
borders, merges, widths, heights) + an openpyxl pass (font name/size/color/italic, numFmt,
alignment, freeze panes). Diff harness: `.claude/workbook-template/diff.py`.

## 1. Convergence status

- **inspect diff: 41 residuals**, all justified golden-master artifacts (below).
- **openpyxl diff: 3950 residuals**, of which:
  - 3753 = `fontcolor None != 000000` (every styled cell), 160 = `font.size None != 11.0`
    (empty cells). Both are **excelize-serialization noise**: excelize writes an explicit
    ARGB black / default font on styled or touched cells, where Excel/LibreOffice leaves the
    theme default (`None`). Visually identical; not a content difference. (The anticipated
    "openpyxl default-style noise.")
  - 37 = the golden-master artifacts already counted in the inspect residuals (Receitas Dec
    numFmt, Variáveis trailing-row formatting, Extras `Saúde` wrap).
- **freeze panes**: all 6 sheets match (C3 data sheets, D4 Listas).
- All structural geometry, every cell value, every formula, all merges, widths, the
  derived row/column layout, and all canonical styling converge exactly.

## 2. Residual diffs (each justified)

| # | Where | Diff | Justification |
|---|---|---|---|
| R1 | Receitas `B7:B9` vs `B7:B10` | 13° col-B merge excludes its total row | **Documented spec §5 self-inconsistency.** Builder implements the spec rule (merge includes total row → B7:B10); golden omits row 10. Expected residual. |
| R2 | Data sheets, banner anchors `U1/AA1/AG1` (k=6,8,10) | golden lacks the left border these cells have | Golden manual-edit gap: the user's border drag skipped months 6/8/10 on the month-banner anchors. Irregular, non-derivable; builder applies the border uniformly. |
| R3 | Adicionais total rows, `I/K/O/Q` (k=2,4) | golden lacks the group-separator borders the builder emits | Golden's Adicionais total-row vertical-separator pattern is irregular (`011111010101`) vs the clean odd-k pattern on Fixas/Variáveis/Extras. Builder uses the consistent odd-k rule (matches 3 of 4 sheets exactly). |
| R4 | Variáveis rows 22–24 height; A29–32 font/align/wrap/bold | golden = 15.75 / Arial-14-bold-centered-wrapped on empty rows | Golden has stray manual formatting on blank separator/trailing rows below the data. Empty-cell cosmetic noise. |
| R5 | Receitas Dec column `AK*/AL*` numFmt | golden = `DD/MM` / `R$ #,##0.00`, builder = General | Golden retained date/currency numFmt only on the December data column; all other Receitas data cells are General (which the builder matches uniformly). |
| R6 | Listas `C31/C32` em-dash | `Orion — Consultas` vs `Orion - Consultas` | Golden uses an em-dash in the Listas pull label but a hyphen in the Variáveis source label (`B14:B17`). Builder uses the spec §1 hyphen everywhere (consistent). |
| R7 | Listas Extras section `C42/C45/C48` | golden duplicates the col-B label into col C | Golden duplicates `Total Saúde` / `Total Manutenção / prevenção` / `% sobre Receita` into col C **only in the Extras section** (not Fixas/Variáveis/Adicionais). One-off manual artifact; builder keeps these in col B only. |
| R8 | All styled cells: `fontcolor 000000`, empty cells `font.size 11` | openpyxl reads explicit black / default size | excelize serialization vs Excel theme defaults. Visually identical. |

## 3. Golden-master statements that contradict spec v2

1. **Receitas 13° merge** excludes its total row (`B7:B9`) — spec §2/§5 says include. (R1)
2. **Month-banner left borders** are not uniform — gaps at k=6,8,10 (R2); spec §3.4 implies
   a uniform banner style.
3. **Adicionais total-row group separators** are irregular (R3); the other three data sheets
   use a clean every-other-month pattern, which the spec's "left on each triple's first col"
   only loosely describes.
4. **Receitas data-cell numFmt** is General except the December column (R5); spec §2 implies
   uniform per-month numFmts on all data cells.
5. **Listas sub-item dash** is an em-dash in the rollup but a hyphen in the source sheet (R6);
   spec §1 specifies one dash-separated col-B string.
6. **Listas Extras col-C label duplication** (R7) — not in the spec's Listas geometry (§4),
   which defines col C as the item column only.

## 4. Key builder changes (v1 → v2)

- **Column model** (`colmap.go`): all data sheets (incl. Receitas) start month triples at
  col C (index 2); eliminated the sub-item/spacer column. Listas months moved to D–O.
- **Labels merged, not filled-down** (`expense_sheet.go`, `receitas_sheet.go`): col A merged
  across each category section (incl. total rows), col B merged across each subcategory block
  (incl. its total row). Category labels = Arial 14 bold, centered + wrapped.
- **Headers**: `Mês` merged `A1:B2`; month banners merged 3-wide with the style on the anchor
  only (top+left border, matching golden); header-col row left un-centered (golden).
- **Freeze panes**: C3 on data sheets, D4 on Listas.
- **Receitas unified** into the data-sheet family but with its own widths/heights, no
  inter-month total-row separators, and General-numFmt data cells (golden).
- **Total rows**: every-other-month vertical group-separator borders (odd k) on data sheets.
- **Composed subcategory**: `Orion - Consultas` as a single col-B string (`taxonomy.go`).
- **Listas** (`listas_sheet.go`): full v2 rebuild — 3-col label area, black Arial-18 section
  labels in col A, 333399 categoria bands in col B (centered/wrapped merged labels), CCCCFF
  group totals + percent rows, C0C0C0 sheet totals, 333333 saldo aggregates; label columns
  carry General numFmt while only D–O carry currency/percent; single-row blocks emit
  `SUM(D34)` (not `SUM(D34:D34)`) and skip degenerate merges; cross-sheet refs quote sheet
  names with non-ASCII-alnum chars (`'Variáveis'!E6`, `Fixas!E6`). `perGroupPctRows` const
  added (default false per §4.2 ⚠).

## 5. Notes

- Cached formula values (`0`, `0.00%`) appear in the golden dump but not the generated file;
  populated on open via `UpdateLinkedValue()` + `SetCalcProps(FullCalcOnLoad)`. The diff
  harness suppresses these cached-only value diffs.
