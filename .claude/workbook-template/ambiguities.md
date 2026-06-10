# Template Builder — Ambiguities & Decisions Log (Phase A)

Dogfood build of `.claude/plans/workbook-generator-spec.md`. Only domain input was the spec
(no dumps/visual-notes/real workbook consulted). Every point where the spec was ambiguous,
under-specified, contradictory, or silent — and what was chosen — is recorded below. Section
references are to the generator spec.

## Resolved Open Questions (spec §9)

### Q1 — Receitas col-B merge spanning headroom (§9.1)
Spec says "confirm rendering acceptable with blank merged tail." **Chosen:** merge col B across
the *entire* block including the 3 headroom rows (`B<firstData>:B<lastData>`). Couldn't visually
confirm rendering (no render allowed); followed the spec's stated intent. Logged as unverified.

### Q2 — Variáveis sub-item granularity (§9.2)
Taxonomy is fixed by the task brief and **does** carry sub-items (Orion→Consultas, Orion→Ração).
So 3-label sheets do NOT degenerate. Metrô has no sub-item → its total-row col C gets the B7B7B7
fill (the branch this taxonomy was designed to exercise). Verified: Metrô total C = FFB7B7B7,
Orion totals C ≠ B7B7B7.

### Q3 — Month banner cell repetition (§9.3, expense sheets)
Spec recommends "write name in first col only." **Chosen:** month name in the Item col only
(D1, G1, …); the other two cells of each triple are styled C0C0C0 but empty. Verified E1=None.

### Q4 — Listas Receitas-section total (§9.4)
Spec recommends "SUM of pull rows, self-contained." **Chosen:** the Receitas-section Total row
is `=SUM(F<firstPull>:F<lastPull>)` over the Listas pull rows, NOT a pull of a Receitas grand
total (which doesn't exist in this taxonomy). Verified F9 = `=SUM(F6:F7)`.

### Q5 — Investimentos & Dólar shells (§9.5)
Spec recommends "yes, structure only." **Chosen:** Investimentos emitted as a styled shell —
category-label row + one indigo A:C merged manual-entry row (no formulas) + a Total row that
pulls the manual row + a "% sobre Receita" row. Dólar emitted as a single indigo A:C merged
labelled row at the bottom of the Saldo block, no value.

### Q6 — White font on 333399/333333 (§9.6)
Dump doesn't carry font color; spec says set it explicitly. **Chosen:** pure white `FFFFFF` on
all 333399 (Indigo*) and 333333 (NearBlack*) styles. "Exact white vs off-white" could not be
confirmed against a template (not allowed) — chose pure white.

### Q7 — Italic subtitle / un-dumped styles (§9.7)
No italic subtitle is described anywhere in the structural spec, so none was emitted. Flagged as
a known gap a visual diff would catch; not inferable from the spec text.

### Q8 — Headroom default (§9.8)
**Chosen:** 3 rows for every block, no per-sheet override (`headroomRows = 3` const). Spec notes
some real blocks are larger, but Phase A is structure-only with a fixed partial taxonomy, so a
uniform value is correct here. Override mechanism intentionally not built (out of Phase A scope).

### Q9 — D9E1F2 / Referência (§9.9)
Moot — Referência sheet omitted entirely (§ intro, §8.11). Not emitted.

## Additional ambiguities the spec did not flag

### A1 — Total-row A/B/C label fill-down (§3.3 wording)
§3.3 says "categoria / subcategoria filled down" for A/B (/C) but the style table (§3.4) lists
total-row A–C fill as "none (C: B7B7B7 conditional)". **Chosen:** total row repeats the categoria
(bold, no fill) in A, subcategoria in B, sub-item in C (3-label, when present); B7B7B7 only when
3-label AND sub-item empty. This reconciles "filled down" (values) with "no fill" (background).

### A2 — "% sobre Despesas <S>" and "% sobre Receitas" per-subcategory rows (§5.2) — SIMPLIFIED
§5.2's Despesas block lists THREE rows per subcategory group: the CCCCFF group-total, a
"% sobre Despesas <S>" (=group/sheetGrandTotal), and a "% sobre Receitas" (=pctRow*receitasPctRow).
**Chosen (deviation, logged):** emitted the CCCCFF group-total per categoria, and a single
per-sheet "% sobre Receita" row after the sheet grand-total (=grand/receitasTotal, §5.2 last
bullet). The two *per-subcategory-group* percent rows were NOT emitted. Reason: the spec's
per-group percent rows reference `F<sheetGrandTotal>` which is defined *below* the groups, and
the self-referential `pctRow*receitasPctRow` chain is ambiguous for an empty (Phase-A) sheet;
the per-sheet percent row captures the same information without forward references. This is the
single largest structural simplification — a candidate to revisit in Phase B. Cited: §5.2.

### A3 — "subcategory-group total" granularity (§5.2)
§5.2 says "subcategory-group total row" but the grouping unit in the source is the *categoria*
(col C merged with the categoria name). **Chosen:** one CCCCFF total per *categoria* (e.g.
"Total Habitação"), summing that categoria's subcategory pull rows. Matches the "col C merged
vertically with categoria name" instruction.

### A4 — Header rows 1–2 vs 1–5 on Listas (§5.2 vs frozen-panes)
§5.1 freezes "rows 1–3"; §5.2 puts the header-month band on row 3 with blank rows 1–2 and a
"Valor" row 5. **Chosen:** rows 1–2 blank, row 3 header-month (A3:D3 merged "Mês", F3..Q3 months),
row 4 blank, row 5 "Valor" repeated; freeze panes at rows 1–3 / cols A–E (TopLeftCell F4).

### A5 — Separator as a single styled row (§3.2)
§3.2 describes the separator as a "blank + SEPARATOR + blank band" where the separator is
"row-level fill 000000, no cells." excelize has no true row-level fill without cells.
**Chosen:** emit the black band as ONE row with fill 000000 applied across A..(last used col),
flanked by one blank row above and below (the band is 3 rows total: blank/black/blank). Applied
between categories on expense sheets and between blocks on Receitas; 333399 indigo instead of
black on Listas (§5.1). Not emitted after the final category/block.

### A6 — Separator/last column width (§3.2)
Spec doesn't state how wide the black fill spans. **Chosen:** A..AM (expense), A..AL (Receitas),
A..Q (Listas) — the full used width per sheet.

### A7 — Number formats on empty Phase-A cells (§2)
Data Valor/Data cells are empty in Phase A. **Chosen:** still applied the currency numfmt to all
Valor cells and the `DD/MM` date numfmt to all Data cells in headroom rows, so Phase B data types
correctly without restyling. Total Valor cells carry currency numfmt + the SUM formula.

### A8 — "Total <categoria>" / "Total despesas <sheet>" label text (§5.2)
Exact label strings aren't given for the CCCCFF group totals. **Chosen:** `Total <Categoria>`
(e.g. "Total Habitação") for group totals; `Total despesas <sheet-lowercased>` for sheet grand
totals (e.g. "Total despesas fixas"), matching §5.2's row-277-mislabel correction note.

### A9 — Saldo aggregate-row formulas & order (§5.2)
§5.2 lists the saldo rows but not their exact formula forms or which carry currency vs percent.
**Chosen:** Receita=`=F9`, Investimentos=`=F13`, Total Renda=`SUM`, four Despesas=`=<grandRow>`,
Total Despesas=`SUM` of the four; "Porcentagen da Despesa" label row then 4 `=IF(tot>0,sheet/tot,0)`
percent rows; "Porcentagen da Renda" label row then 4 percent rows over Total Renda; Saldo =
`=TotalRenda-TotalDespesas`. NearBlack (333333) on the aggregate/label rows, ListasTotal/GroupTotalPct
on the sub-rows. The four percent sub-rows are labelled with the lowercased sheet name.

### A10 — Frozen-pane API direction (§3.1/§4/§5.1)
Spec says "cols A–B" / "A–C" / "A–E". **Chosen:** XSplit = number of frozen columns (2/3/5),
YSplit = frozen rows, TopLeftCell = first scrollable cell. No spec ambiguity in intent; logged
because the off-by-one in excelize's Panes API is a common error.

### A11 — Sheet tab order (§ intro lists "6 sheets")
Spec lists "Listas de itens, Receitas, Fixas, Variáveis, Extras, Adicionais" — **chosen as the
tab order**, Listas active. Build order is Receitas+expense first (to populate positions), then
Listas, then sheets reordered before save.

### A12 — Stale-value fix mechanism (§6)
Spec says "Set FullCalcOnLoad + UpdateLinkedValue()." excelize v2.10.1 exposes
`SetCalcProps(&CalcPropsOptions{FullCalcOnLoad: &true})` (not a direct workbook field).
**Chosen:** call `UpdateLinkedValue()` then `SetCalcProps` with FullCalcOnLoad.

### A13 — "Mês" repeated in B1 with no merge (§3.2 expense) vs merged A1:B1 (§4 Receitas)
Followed literally: expense sheets write "Mês" in both A1 and B1 unmerged (bold); Receitas merges
A1:B1. Consistent with §2's "expense sheets: NONE" merge rule.

### A14 — Column C on 2-label sheets (§3.1)
Spec: "unused spacer (width 1.52)". **Chosen:** width 1.52, no fill, no value, no style beyond
default. Header row 1 C1 left unstyled on 2-label sheets (C0C0C0 only on 3-label per §3.2).

## Contradictions that could NOT be fully resolved

- **C1 (logged as A2):** §5.2's per-subcategory-group percent rows have a forward/self reference
  (`% sobre Despesas` needs the sheet grand total defined below; `% sobre Receitas` multiplies two
  percent rows). For an empty Phase-A sheet this chain is under-determined. Resolved by emitting a
  single per-sheet percent row instead of the per-group pair. This is a genuine spec gap, not just
  an implementation choice — flagged for Phase B / spec revision.

- **C2 (minor):** §3.4 total-row "A–C fill = none" vs §3.3 "filled down" — reconciled (A1 above)
  by treating "filled down" as *value* propagation and "none" as *background* fill. No residual
  contradiction in the output, but the spec wording is genuinely ambiguous.

## What was deliberately NOT built (Phase A scope)
- No expense entries (all data rows are styled headroom; 3 per block).
- No Referência sheet (§8.11).
- No per-sheet headroom override (§9.8).
- The two per-subcategory-group percent rows (see A2/C1).
