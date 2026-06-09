# Workbook Generator Spec

**Goal:** spec for a Go command that generates the expense workbook from scratch, given
(a) a taxonomy (Tipo Principal → Categoria → Sub-categoria) and (b) expense/income entries
per subcategory. Synthesized from the Layer 1 JSON dump (`.claude/workbook-dump/`), the
Layer 2 visual notes, and 7 per-sheet digests (`.claude/workbook-dump/digests/`).

**Design stance — DERIVED LAYOUT:** row positions are *computed* from the taxonomy and entry
counts, never copied from the source workbook. The source dump is a validation reference
only. Where the source contains hand-maintenance drift or bugs, this spec deliberately
deviates (see §8).

**Sheets generated (6):** Listas de itens, Receitas, Fixas, Variáveis, Extras, Adicionais.
**Referência de Categorias is OMITTED** — its machinery (row mappings, audit formulas)
exists to support manual insertion, which the generated workbook does not serve for now.
If the generated workbook must later support the existing `add`/`batch` CLI resolver, add a
slim sheet (title + header + the A/B/C taxonomy columns) — see the Referência digest §4 for
that slim spec.

---

## 1. Inputs (contract)

```
Taxonomy: ordered list of (tipoPrincipal, categoria, subcategoria [, subItem])
  - tipoPrincipal ∈ {Fixas, Variáveis, Extras, Adicionais} → selects the expense sheet
  - subItem applies only on 3-label sheets (Variáveis; Adicionais reserved)
Income taxonomy: ordered list of (incomeCategoria, blockLabel) for Receitas
Entries: per subcategory, per month: list of (item, date, value)
Config: year, headroomRows (default 3), canonical formats (§2)
```

Ordering in the taxonomy input is the ordering in the workbook (categories contiguous,
subcategories in listed order). The current taxonomy (113 subcategories: Fixas 39,
Variáveis 34, Extras 13, Adicionais 27) is recorded in the Referência digest §2.

## 2. Workbook-wide canonical conventions

These normalize the source's inconsistencies (mixed `R$ 200,00` / bare `209,80`; mixed
`17/1` / `21/03` / `1/1/2025` dates; text-typed numbers):

| Concern | Canonical rule |
|---|---|
| Currency cells | true numeric values + number format `R$ #,##0.00` (renders `R$ 1.234,56` in pt-BR locale) |
| Date cells | true date values + format `DD/MM` (year implied by workbook year) |
| Fonts | Header rows (header-month, header-col): Open Sans 14pt. Everything else: Arial 10pt |
| Total-row Data col placeholder | `–` (en dash) everywhere (source mixes `""` and `–`) |
| SUM formulas | always range form `SUM(X<first>:X<last>)`, always over the **Valor** column (fixes source June bug, single-cell collapse) |
| Column widths | one width per column role (see per-sheet tables); source per-column drift is NOT reproduced |
| Row heights | header-month 18pt; all body rows 12.75pt (expense sheets), 15pt (Receitas, Listas) |
| Merges | expense sheets: NONE (labels fill down / values repeat). Receitas + Listas: merges per §5/§6 |

## 3. Expense-sheet family (Fixas, Extras = 2-label; Variáveis, Adicionais = 3-label)

### 3.1 Column model (39 cols, A–AM)

| Col | Role | Width |
|---|---|---|
| A | Categoria label (filled down every row of block, **bold**) | 20.6 |
| B | Subcategoria label (filled down, not bold) | 15.75 |
| C | 2-label sheets: unused spacer (width 1.52). 3-label sheets: sub-item label (width 8.9) |
| D + 3k | month k Item | 14.25 |
| E + 3k | month k Data | 8.0 |
| F + 3k | month k Valor | 12.13 |

Month → column map (k = 0..11): Janeiro D/E/F, Fevereiro G/H/I, Março J/K/L, Abril M/N/O,
Maio P/Q/R, Junho S/T/U, Julho V/W/X, Agosto Y/Z/AA, Setembro AB/AC/AD, Outubro AE/AF/AG,
Novembro AH/AI/AJ, Dezembro AK/AL/AM. **Valor cols: F I L O R U X AA AD AG AJ AM.**

**Frozen panes:** rows 1–2; cols A–B (2-label sheets) or A–C (3-label sheets).

### 3.2 Row model (parametric)

```
Row 1  header-month: fill C0C0C0 entire used width.
       A1 = "Mês", B1 = "Mês" (repeated value, NO merge; bold). C1 empty (3-label: filled C0C0C0).
       Month name in the Item col of each triple (D1, G1, J1, …); value NOT repeated across
       the triple's other two cols (cells styled C0C0C0 but empty).  [see Open Q3]
Row 2  header-col: fill D8D8D8 on D..AM; "Item"/"Data"/"Valor" repeated per month triple.
       A2/B2/C2: no fill, empty. Full thin borders on each filled cell.
Row 3+ per category, in taxonomy order:
   per subcategory:
      data rows   = entries-derived rows + headroomRows blank rows (styled as data rows)
      total row   = 1 row (see 3.3)
   after last subcategory of the category: 1 blank row, 1 SEPARATOR row, 1 blank row
   (separator: no cells, row-level fill 000000, the blank+black+blank band the source
   renders; not emitted after the final category)
```

Data-row content: A=categoria (bold), B=subcategoria, C=subItem (3-label sheets, may be
empty), then per month with entries: Item/Data/Valor in that month's triple. Entries for
different months sit on the same rows top-aligned (the grid is per-month independent —
row n of a block holds the n-th entry *of each month*).

Block size = max over months of entry count, + headroomRows. SUM ranges cover the full
block including headroom rows, so manual additions stay counted.

### 3.3 Total row

| Cell | Content | Style |
|---|---|---|
| A, B (, C) | categoria / subcategoria filled down | no fill; C gets **B7B7B7** fill iff 3-label sheet AND sub-item empty (Variáveis rule; never on Adicionais today since its col C is unpopulated) |
| each month Item col | literal `Total` | F2F2F2 |
| each month Data col | literal `–` | F2F2F2 |
| each month Valor col | `=SUM(<ValorCol><firstDataRow>:<ValorCol><lastDataRow>)` | F2F2F2 |
| all D..AM cells | fill F2F2F2, thin borders top+bottom (left on each triple's first col) |

### 3.4 Style fingerprints (expense sheets)

| rowType | A–C fill | D+ fill | bold | borders |
|---|---|---|---|---|
| header-month | C0C0C0 | C0C0C0 | A/B yes | box on A; box on each triple-start col |
| header-col | none | D8D8D8 | no | full thin borders per cell |
| data-row | none | none | A only | top border on first row of block (D+) |
| total-row | none (C: B7B7B7 conditional) | F2F2F2 | no | top+bottom on D+ |
| separator | row-level fill 000000, no cells | | | |

### 3.5 What is NOT reproduced from the source

- `category-label` spacer rows (Extras 26–36, Fixas after most totals, Variáveis 104–107 /
  235–238) — replaced by the uniform headroom + separator policy.
- Category-level total rows (Extras rows 37, 81) — dropped; category totals live in Listas.
- Trailing appended blocks far past the body (Extras 306+, Adicionais 398+) and 200-row
  gaps — derived layout keeps categories contiguous in taxonomy order.
- The 15.75pt row-height patches, manual column-width drift, June `SUM(T..)` bug,
  single-cell `SUM(F16)` forms, fill-down gaps (Extras Mudança).

## 4. Receitas (income; same family look, shifted grid)

Differences from expense sheets — everything else (palette, fonts, total-row pattern,
canonical formats) is identical:

| Aspect | Receitas rule |
|---|---|
| Grid | 38 cols (A–AL). Col A = income category (width 6.75), col B = item/block label (19.13), C–AL data (12.63) |
| Month triples | start at **col C**: Janeiro C/D/E, Fevereiro F/G/H, … Dezembro AJ/AK/AL. **Valor cols: E H K N Q T W Z AC AF AI AL** |
| Row 1 | A1:B1 **merged** = "Mês" (bold, C0C0C0); month banners **merged** 3-wide: C1:E1="Janeiro" … AJ1:AL1="Dezembro" |
| Row 2 | Item/Data/Valor from col C; D8D8D8 |
| Block labels | col B **merged** vertically across the block's rows (B<start>:B<total>) with the block label; col A merged or filled with the income category |
| Total row | `Total` in each month's Item col (C, F, …), `–` in Data col, `=SUM(E<start>:E<end>)` etc. in Valor cols (all 12 months — verified) |
| Separators | same blank + 000000 + blank band between categories |
| Frozen panes | rows 1–2, cols A–B |
| Row heights | row 1: 17.25; body 15; total rows 15.75 |

NOT reproduced: the 10 empty pre-provisioned blocks (rows 37–111) and the anomalous
total-less band at 116–124 — derived layout provisions via headroom instead. The source's
mixed merge usage (blocks 3–32 unmerged, 33+ merged) is normalized to: always merge col B
across the block.

## 5. Listas de itens (rollup — PULLS, never SUMs source ranges)

### 5.1 Geometry

17 cols (A–Q). Widths: A 0.38 (near-hidden), B 12.88, C 9.88, D 13.38, E 10.13 (spacer),
F–Q 16.38. **Month value cols F..Q = Janeiro..Dezembro (1 col per month).**
Frozen panes: rows 1–3, cols A–E. Separators on THIS sheet use fill **333399** (indigo),
not black.

### 5.2 Band structure (top to bottom, positions computed)

```
Rows 1–2  blank
Row 3     header-month: A3:D3 merged "Mês"; F3..Q3 = month names; C0C0C0
Row 4     blank
Row 5     "Valor" repeated in F..Q (no fill)
— Receitas section —
   pull rows: one per Receitas block; A:C merged vertically = "Receitas" (fill 333399,
     white text); D = block label; F..Q = ='Receitas'!<ValorCol><blockTotalRow>
   separator (333399)
   Total row (C0C0C0): D="Total"; F..Q pull each Receitas grand total — or SUM of the pull
     rows above (see Open Q4)
— Investimentos section —
   category-label row (A="Investimentos", no fill), then a manually-maintained block:
   A:C merged 333399; rows have NO formulas (user types monthly amounts); ends with a
   total (C0C0C0) + "% sobre Receita" row
— Despesas block — category-label row (A="Despesas"), sub-header row (B="Categoria", D="Despesa")
   for each source sheet S in (Fixas, Variáveis, Extras, Adicionais):
      for each categoria in S (col C merged vertically with categoria name; A–B 333399):
         pull rows: one per subcategory; D = subcategoria;
            F..Q = ='<S>'!<ValorCol_k><subcatTotalRow>   (ValorCol per §3.1/§4 maps)
         subcategory-group total row (fill CCCCFF): F..Q = SUM(F<firstPull>:F<lastPull>)
         "% sobre Despesas <S>" row (CCCCFF):  =IF(F<tot> >0, F<tot>/F<sheetGrandTotal>, 0)
         "% sobre Receitas" row (C0C0C0):      =F<pctRow> * F<receitasPctRow>
         separator (333399)
      sheet grand-total row (C0C0C0): C = "Total despesas <s>";
         F..Q = SUM(<each CCCCFF group-total cell for S>)
      separator; "% sobre Receita" row: =IF(F<grand> >0, F<grand>/F<receitasTotal>, 0)
— Saldo block (bottom) —
   micro-gap row (height 3.75)
   A:C merged "Saldo" (333399) spanning the block
   C0C0C0 sub-label rows + 333333 (near-black) aggregate rows:
      Receita            = <receitasTotalCell>
      Investimentos      = <investimentosTotalCell>
      Total Renda        = SUM(the two above)            [333333]
      Despesas fixas/variáveis/extras/adicionais = the 4 grand-total cells   [C0C0C0]
      Total Despesas     = SUM(the four above)           [333333]
      Porcentagen da Despesa  (label-only row)           [333333]
      4 rows: each sheet total / Total Despesas          [C0C0C0]
      Porcentagen da Renda = Total Despesas / Total Renda [333333]
      4 rows: each sheet total / Total Renda             [C0C0C0]
      Saldo = Total Renda − Total Despesas               [333333]
   "Dólar" row: A:C merged "Dólar" (333399); manual value
```

All A–C cells on body and saldo rows carry 333399 (white text). The generator computes
every referenced row number during layout (it owns all positions on all sheets — this
replaces Referência's role).

Source anomalies normalized: row 277 mislabel ("Total despesas extras" in the Adicionais
block) → correct label; row 300's missing A–C cells → emit them.

## 6. Cross-sheet wiring summary

Only Listas references other sheets. Pull formula template:

```
Listas!<F+k> = ='<SourceSheet>'!<ValorCol(S,k)><totalRow>     k = 0..11 (month index)
ValorCol(expense sheet, k) = col F + 3k   (F, I, L, … AM)
ValorCol(Receitas, k)      = col E + 3k   (E, H, K, … AL)
```

`totalRow` = the subcategory's (or Receitas block's) total row, computed during generation.
Quote sheet names containing spaces/accents (`'Listas de itens'!`, `Variáveis!` — excelize
handles quoting; always emit quoted form for safety). Set `FullCalcOnLoad` +
`UpdateLinkedValue()` before save (established fix — LibreOffice/Sheets stale-value issue).

## 7. Validation plan (summary — detail in the phase plan)

Hand-built template workbook (partial taxonomy, then partial data) is the golden master.
Compare via `workbook-inspect` structural dumps of template vs generated output
(normalized diff). Phase A: structure only, no entries. Phase B: with entries (totals,
pulls live). Candidate acceptance tests: deterministic, fixture-safe (fake data).

## 8. Deliberate deviations from the source workbook

So template diffs aren't mistaken for generator errors:

1. June Valor totals: source sums col T (Data) — **fixed** to col U (systematic on all 28
   Adicionais totals; Fixas row 12; Extras row 47).
2. Single-cell `SUM(F16)` totals (all Adicionais months; Extras Jan/Feb/Nov/Dec) — **range
   form** everywhere. NOTE: Adicionais source single-cell SUMs reference only the LAST data
   row, so its rendered totals are wrong in the source.
3. Inconsistent fill-down (Extras Mudança col A blank) — **always fill down**.
4. `""` vs `–` in total-row Data cols — **always `–`**.
5. Category-label spacer rows + category-level total rows — **dropped** (headroom +
   Listas totals replace them).
6. Trailing appended blocks / 200-row gaps / category regressions — **contiguous taxonomy
   order**.
7. Text-typed numbers & dates, mixed display formats — **typed values + one numFmt** (§2).
8. Column width / row height drift — **canonical per role**.
9. Receitas pre-provisioned empty blocks & inconsistent col-B merging — **headroom policy +
   always-merge**.
10. Listas row-277 mislabel, row-300 missing cells — **corrected**.
11. Referência sheet — **omitted entirely**.

## 9. Open questions

1. **Headroom interaction with merges (Receitas):** col B merge must span headroom rows
   too — confirm rendering acceptable with blank merged tail.
2. **Variáveis sub-item granularity:** in the source, sub-items split a subcategory into
   multiple blocks (Orion→Consultas/Ração). Does the entries datastore carry sub-item?
   If not, 3-label sheets degenerate to 2-label + B7B7B7 on every total row col C.
3. **Month banner cell repetition:** dump shows month name only in the triple's first col
   on expense sheets (other two cells styled-but-empty); Fixas digest noted "value repeated
   across all 3 cols". Decide: write name in first col only (recommended) — verify against
   template render.
4. **Listas Receitas-section total:** source pulls `Receitas!E21` (a block total) for the
   section Total row — verify whether a Receitas *grand* total exists or whether the Listas
   row should SUM its own pull rows (recommended: SUM of pull rows, self-contained).
5. **Investimentos & Dólar:** manual sections with no backing data — generate as empty
   styled shells? (Recommended: yes, structure only.)
6. **White font on 333399/333333 cells:** visual-notes fact; dump doesn't carry font color.
   Generator must set font color explicitly — confirm exact white vs off-white in template.
7. **Italic subtitle / any other un-dumped styles:** the dump omits italics and font color;
   the template build is the catch-all for these (diff by eye once).
8. **Headroom default:** 3 rows assumed — confirm, and whether per-sheet overrides needed
   (e.g. Variáveis Supermercado/Luz blocks are much larger than typical).
9. **D9E1F2 render discrepancy:** moot (Referência omitted) — recorded here for history;
   resolve only if the slim Referência sheet is ever added with alternating fills.

## 10. Source references

- Digests (per sheet detail, exemplar rows): `.claude/workbook-dump/digests/*.md`
- Raw dumps (gitignored): `.claude/workbook-dump/*.json`
- Visual notes (gitignored): `.claude/workbook-visual-notes.md`
- Taxonomy (113 triples): Referência digest §2
- Layer 3 brief: `.claude/plans/workbook-layer3-instructions.md`
