# internal/excel/ — Knowledge (Semantic Memory)

*Excel package accumulated decisions + workbook structural map. Read on demand by agents.*

## Workbook Structural Map (2026-06-08, session 26)
The full structure of the target workbook (`Planilha_Normalized_Final.xlsx`) was mapped to
machine-readable JSON by `cmd/workbook-inspect` (Layer 1 of the workbook-mapping plan). Output:
`.claude/workbook-dump/*.json` (gitignored — real expense values). The facts below are the
distilled, non-sensitive structure; they drive the future "generate workbook from database"
command. Visual cross-check notes: `.claude/workbook-visual-notes.md`.
**Rationale:** The insert-into-existing-workbook approach is a bottleneck; long-term we generate
the workbook from `expenses_log.jsonl` as source of truth. A complete structural spec is the
prerequisite.
**Implication:** Anyone touching read/write column logic should consult this map first — it
documents layout rules the old dynamic-detection code only partially encodes.

## Two Sheet Families (2026-06-08)
The 7 sheets split into two families with DIFFERENT rules:
- **Expense-entry sheets** (Fixas, Variáveis, Extras, Adicionais, Receitas): per-month
  Item/Data/Valor columns, subcategory blocks, total rows.
- **Aggregation/reference sheets** (Listas de itens, Referência de Categorias): rollup + taxonomy.
**Rationale:** Each family has its own palette, layout, and formula style.
**Implication:** A generator must branch per family — there is no single workbook-wide template.

## Palette & Fill — Per Family, Not Workbook-Wide (2026-06-08)
- Expense sheets share ONE palette: `C0C0C0` month banner / `D8D8D8` Item·Data·Valor header /
  `F2F2F2` total rows. `B7B7B7` on col C of Variáveis total rows (darker, the sub-item column).
- Listas de itens: `333399` (indigo-violet, white text), `CCCCFF`, `333333`.
- Referência: `D9E1F2` (light blue), `366092` (navy), `E7E6E6` (warm gray); uses **Cambria 14pt**
  (expense sheets use Open Sans headers / Arial data).
**Rationale:** The "Normalized" workbook is hand-styled per sheet, not from a shared theme.
**Implication:** Generator needs a per-sheet style palette + per-sheet font. ⚠ Open issue:
Referência `D9E1F2` renders warm orange-yellow (not blue) on col A — likely theme remap or
conditional formatting; resolve before trusting that hex.

## Fill-Down vs Merge (2026-06-08)
- Expense sheets have NO merged cells — category (col A, bold) and subcategory (col B) are
  filled DOWN on every row of a block. (Verified: Adicionais Lazer block A3:A17 = "Lazer"
  repeated, not merged.) Variáveis & Adicionais add a 3rd label column C (sub-item); Fixas &
  Extras are A+B only.
- Receitas, Listas de itens, Referência DO use merges (month banners, section labels).
**Rationale:** Normalization flattened merged category labels into repeated values on the
expense sheets — which is exactly why the single column-B scan in the write path works.
**Implication:** Generator fills down on expense sheets, merges on the other three.

## Block Structure & Total Formulas (2026-06-08)
Within an expense sheet, each (category, subcategory) block is a vertical run delimited by a
total row (`rowType:"total-row"`, fill `F2F2F2`, col D = "Total"). Each total row carries **12
monthly `SUM` formulas** (one per month column), e.g. `SUM(F26:F36)`. A category also has a
leading run of col-A-only rows (no subcategory) = a category-level open-allocation area summing
into the category total.
**Rationale:** Months are laid out as repeating 3-col groups (Item/Data/Valor); the total row
sums each month's Valor column over the block's data rows.
**Implication:** A data sheet has ~15 total rows × 12 = ~180 SUM formulas. Generator emits them
parametrically: `SUM(<valorCol><firstDataRow>:<valorCol><lastDataRow>)` per month.

## Black Separator Rows (2026-06-08, Layer 2 finding → Layer 1 fix)
Expense sheets have fully-empty rows at category boundaries that render as SOLID BLACK via
ROW-LEVEL fill. `GetRows` yields them as empty slices and a cell-level dump drops them.
`cmd/workbook-inspect` was enhanced (`906a99d`) to probe row-level fill (GetCellStyle on an
empty cell falls back to the row style) and emit them as `rowType:"separator"` with `rowFill`.
Ground-truth black (`000000`) rows: Extras 39/67/83; Adicionais 130/147; Variáveis
63/86/151/183/223/241; Fixas 140; Receitas 114/126.
**Rationale:** Row-level fills are invisible to cell-level extraction — a real blind spot.
**Implication:** Generator must emit these separator rows explicitly; read positions from the JSON.

## Cross-Sheet Wiring (2026-06-08)
- **Listas de itens is a rollup that PULLS, not SUMs** locally: each cell references the source
  sheet's total cell directly (`Fixas!F19`, `Receitas!E3`, …). `crossSheetRefs` = the 5 sources.
- **Referência de Categorias is the row-mapping source of truth**: col D = subcategory's row in
  its data sheet, col E = its row in Listas, col F = total row in Listas. Its formulas use
  `CONCATENATE("'Listas de itens'.F",$E5)` — ODS dot-notation building reference STRINGS inside
  string literals, NOT real `!` refs (so they don't appear in crossSheetRefs).
**Rationale:** The workbook is internally wired so each sheet's totals propagate up to Listas,
and Referência audits that the mapping is correct.
**Implication:** Generator must reproduce Referência's row map and wire Listas to pull from it.

## Source-of-truth caveat: which workbook (2026-06-08)
Mapping was done on `Planilha_Normalized_Final.xlsx` (the target shape). The runtime
`config.json` `workbook_path` points at the live BMeF orçamento workbook. They share structure
but the Normalized one is the canonical generator target.
**Implication:** When implementing the generator, target the Normalized layout; don't assume the
configured workbook is identical in every styled detail.
