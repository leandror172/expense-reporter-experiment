# Workbook Generator Spec — v2

**Goal:** spec for a Go command that generates the expense workbook from scratch, given
(a) a taxonomy (Tipo Principal → Categoria → Sub-categoria) and (b) expense/income entries
per subcategory.

**Provenance:** v1 was synthesized from the Layer 1 JSON dump (`.claude/workbook-dump/`),
Layer 2 visual notes, and 7 per-sheet digests (`.claude/workbook-dump/digests/`). v2 folds in
the user's hand-corrections to the Phase-A template (diff catalogue:
`.claude/workbook-template/review-diff.md`; golden master:
`.claude/workbook-template/template-reviewed.xlsx`). **Where v2 and the original workbook
disagree, v2 wins — the generated workbook is a redesign, not a replica.**

**Design stances:**
- **DERIVED LAYOUT:** row positions are computed from taxonomy + entry counts, never copied.
- **MERGES, not fill-down:** v2 reversal — labels are written once and merged vertically.
  (The source used fill-down because hand-inserting rows into merges is painful; a generated
  workbook doesn't care.)
- **2 label columns everywhere:** the sub-item level is eliminated; composed subcategories
  become one string in col B (`Orion - Consultas`, dash-separated).

**Sheets generated (6, tab order):** Listas de itens, Receitas, Fixas, Variáveis, Extras,
Adicionais. **Referência de Categorias is OMITTED** (insertion-support machinery; the
generated workbook is not an insertion target for now). If CLI `add`/`batch` compatibility
is ever needed, add a slim taxonomy sheet — see Referência digest §4.

---

## 1. Inputs (contract)

```
Taxonomy: ordered list of (tipoPrincipal, categoria, subcategoria)
  - tipoPrincipal ∈ {Fixas, Variáveis, Extras, Adicionais} → selects the expense sheet
  - subcategoria may be a composed string ("Orion - Consultas") — no separate sub-item field
Income taxonomy: ordered list of (incomeCategoria, blockLabel) for Receitas
Entries: per subcategory, per month: list of (item, date, value)
Config: year, headroomRows (default 0 — see §3.2 / §7 Q1; the workbook is regenerated from
the database, never inserted into, so spare rows are optional manual-entry convenience)
```

Taxonomy order = workbook order (categories contiguous, subcategories in listed order).
Current full taxonomy (113 subcategories): Referência digest §2 — sub-item splits must be
composed into col-B strings when imported.

## 2. Workbook-wide canonical conventions

| Concern | Rule |
|---|---|
| Currency cells | numeric values + numFmt `R$ #,##0.00` |
| Date cells | date values + numFmt `DD/MM` |
| Fonts | header rows: Open Sans 14 bold. Merged category labels (col A): Arial 14 bold, centered both axes, wrap. Body: Arial 10. |
| Total-row Data col placeholder | `–` (en dash) |
| SUM formulas | always range form over the **Valor** column |
| Labels | vertical merges (category col A across its whole section incl. total rows; subcategory col B across its block **including the total row**) |
| "Mês" corner | merged `A1:B2` (spans both header rows), centered |
| Freeze panes | `C3` on data sheets (both header rows + both label cols); `D4` on Listas |
| Number formats | applied to all headroom cells too, so Phase-B data types correctly |
| Forward references | legal and expected (percent rows reference totals below them) |

## 3. Data-sheet family (Fixas, Variáveis, Extras, Adicionais, **and Receitas** — v2 unifies them)

### 3.1 Column model (38 cols, A–AL)

| Col | Role | Notes |
|---|---|---|
| A | Categoria (expense) / income category (Receitas) | merged vertically per section; Arial 14 bold centered wrap |
| B | Subcategoria / income block label | merged vertically per block incl. total row |
| C + 3k | month k Item | numFmt General |
| D + 3k | month k Data | numFmt `DD/MM` |
| E + 3k | month k Valor | numFmt `R$ #,##0.00` |

Month → cols (k=0..11): Janeiro C/D/E, Fevereiro F/G/H, Março I/J/K, Abril L/M/N,
Maio O/P/Q, Junho R/S/T, Julho U/V/W, Agosto X/Y/Z, Setembro AA/AB/AC, Outubro AD/AE/AF,
Novembro AG/AH/AI, Dezembro AJ/AK/AL. **Valor cols: E H K N Q T W Z AC AF AI AL.**

(v1 started months at D with col C as sub-item/spacer; v2 eliminates that column entirely.)

### 3.2 Row model

```
Row 1  header-month: A1:B2 merged "Mês" (bold, C0C0C0, centered).
       Month banners merged 3-wide: C1:E1="Janeiro" … AJ1:AL1="Dezembro";
       fill C0C0C0 + Open Sans 14 bold on the anchor cell.
Row 2  header-col: "Item / Data / Valor" repeated per month triple, D8D8D8, centered,
       thin borders.
Row 3+ per category (in taxonomy order):
   per subcategory block:
      data rows = max-entries-per-month + headroomRows (default 0); the row count is set by
         the BUSIEST month for that subcategory; zero-entry subcategory → 1 row (edge, settle in G2)
      total row (3.3)
      col B merged <firstData>:<totalRow> with the subcategoria label
   col A merged across the category's entire row span (incl. total rows)
   after the category (not after the last one): blank row, separator row (no cells,
   row-level fill 000000 across A..AL), blank row
Receitas delta: separators only between income CATEGORIES (none between blocks of the
   same category, e.g. Salário | 13°).
```

### 3.3 Total row

| Cell | Content | Style |
|---|---|---|
| A, B | covered by the vertical merges (no own value) | no fill |
| each month Item col | `Total` | F2F2F2 |
| each month Data col | `–` | F2F2F2 |
| each month Valor col | `=SUM(<ValorCol><firstDataRow>:<ValorCol><lastDataRow>)` | F2F2F2, currency numFmt |

Borders: top+bottom thin on C..AL; left on each triple's first col.

### 3.4 Style fingerprints

| rowType | A–B | C+ | notes |
|---|---|---|---|
| header-month | C0C0C0 (merged Mês) | C0C0C0 on month anchors only (cells under a merge carry no own fill/font) | Open Sans 14 bold |
| header-col | (covered by Mês merge) | D8D8D8, centered, thin borders | Open Sans 14 |
| data-row | merge anchors only | none | Arial 10 |
| total-row | (covered by merges) | F2F2F2 | Arial 10 |
| separator | row-level fill 000000, no cells, flanked by one blank row each side |

## 4. Listas de itens (rollup — PULLS source total cells)

### 4.1 Geometry — v2: 15 cols (A–O)

| Col | Role |
|---|---|
| A | sheet-type / section label (Receitas, Fixas, Variáveis, Extras, Adicionais, Saldo, Dólar) — merged vertically per section, 333399 fill, white text |
| B | categoria label — merged vertically per group (e.g. B40:B41 "Saúde") |
| C | item: subcategoria name / block label / saldo row label |
| D–O | months Janeiro–Dezembro, one col each. numFmt `R$ #,##0.00` (percent rows: `0.00%`) |

Freeze panes `D4`. Separators on this sheet: row-level fill **333399**.

### 4.2 Band structure (positions computed)

```
Rows 1–2 blank
Row 3   header-month: "Mês" label area; D3..O3 month names, C0C0C0
Row 4   blank
Row 5   "Valor" repeated D..O (no fill)
— Receitas section — A merged "Receitas" (333399, white)
   pull rows: C = block label; D..O = ='Receitas'!<ValorCol(k)><blockTotalRow>
   separator (333399)
   Total row (C0C0C0): C="Total"; D..O = SUM of the pull rows above
— Investimentos — label row + one manual-entry shell row + Total (=manual row) +
   "% sobre Receita" (=IF(receitasTot>0, invTot/receitasTot, 0)), numFmt 0.00%
— Despesas — label row; sub-header row
   for each source sheet S (Fixas, Variáveis, Extras, Adicionais):
      A merged with S's name across the section (333399, white)
      for each categoria (B merged with categoria name across its group):
         pull rows: C = subcategoria; D..O = ='<S>'!<ValorCol(k)><subcatTotalRow>
         categoria-group total row (CCCCFF): D..O = SUM(D<firstPull>:D<lastPull>)
         "% sobre despesas" row (CCCCFF): =IF(D<grpTot>>0, D<grpTot>/D<sheetGrand>, 0)
         "% sobre receita" row (CCCCFF): =IF(D<recTot>>0, D<grpTot>/D<recTot>, 0)
      sheet grand-total row (C0C0C0): C = "Total despesas <s>"; D..O = SUM(group totals)
      separator; "% sobre receita" row (fill CCCCFF, numFmt 0.00%)
— Saldo block — A merged "Saldo" (333399); labels in col A-adjacent layout per golden master:
   Receita (=receitas total) | Investimentos (=inv total)        [C0C0C0]
   Total renda = SUM of the two                                   [333333]
   4 rows: Despesas fixas/variáveis/extras/adicionais (= grand totals) [C0C0C0]
   Total despesas = SUM of the four                               [333333]
   "Porcentagem da despesa" label row [333333]; 4 percent rows (sheet/TotalDespesas) [CCCCFF, 0.00%]
   "Porcentagem da renda" label row [333333]; 4 percent rows (sheet/TotalRenda) [CCCCFF, 0.00%]
   Saldo = TotalRenda − TotalDespesas                             [333333]
— Dólar — single merged labelled row (333399), manual value
```

White font (FFFFFF) on all 333399 and 333333 cells — set explicitly.

**Per-group percent rows — RESOLVED (2026-06-10, Phase B).** The two rows above (now labelled
`% sobre despesas` and `% sobre receita`) are required and enter in Phase B: `perGroupPctRows`
flips ON in the builder, and a scripted edit adds the matching rows to the convergence target
`template-data.xlsx` (both derive from this spec rule independently, so the diff still catches
a transcription error in either). Forward reference to the sheet grand total below is legal
(§2). Closes §7 Q2.

### 4.4 Label conventions & i18n

All **generic** user-visible strings are normalized and centralized; **taxonomy-supplied**
names (categoria, subcategoria, sheet names) keep the casing of the taxonomy input.

- **Normalization (pt-BR default):** sentence case — first word capitalized, common nouns
  lowercase; `% sobre …` labels stay fully lowercase; singular/plural unified (`receita`).
  Examples applied: `% sobre receita` (was `% sobre Receita`); per-group `% sobre despesas`
  (sheet-name suffix dropped — col A already shows the sheet, keeps the string reusable
  across all four sheets); `Total renda`/`Total despesas`/`Porcentagem da despesa`/
  `Porcentagem da renda` (was Title-cased / `Porcentagen` typo).
- **Centralization:** a `Labels` struct holds every generic string. **Field names are English
  semantic roles** (`PctOfExpenses`, `Investments`, `TotalIncome`); only the VALUES are
  localized. This forces the Receita-vs-Renda distinction into the naming:
  **Receita → Revenue** (Receitas-sheet total), **Renda → Income** (Revenue + Investments).
- **Month names** live in `Labels` too (language-specific).
- **i18n:** values are hardcoded now via a `newPtBRLabels()` constructor; a `loadLabels(path)`
  config reader (per-language YAML, same family as the taxonomy file) is deferred but is a
  drop-in replacement for the constructor — call sites unchanged.

### 4.3 Pull formula templates

```
Listas!<D+k> = ='<SourceSheet>'!<ValorCol(k)><totalRow>      k = 0..11
ValorCol(any data sheet, k) = col E + 3k    (E, H, K, … AL)  — v2: uniform, Receitas included
```

`totalRow` computed during generation. Always emit quoted sheet names. Set
`UpdateLinkedValue()` + `SetCalcProps(FullCalcOnLoad)` before save (excelize ≥2.8;
LibreOffice/Sheets stale-cache fix). excelize `SetCellFormula` takes the formula WITHOUT
the leading `=`.

## 5. Validation

Golden master: `.claude/workbook-template/template-reviewed.xlsx` (user-curated).
Convergence = `workbook-inspect` dump of generated output diffs empty against the golden
master's dump, plus an openpyxl-level pass for what inspect doesn't capture (font
name/size/color, italics, numFmt, alignment, freeze panes). Phase B repeats with data.
Known golden-master self-inconsistencies (normalize toward the spec, flag in diff report):
- Receitas 13° col-B merge excludes its total row (B7:B9, total 10) while Salário/Fixas
  merges include it → spec rule: include.

## 6. Deliberate deviations from the source workbook

1. June Valor totals summed col T (Data) — fixed to the Valor column.
2. Single-cell `SUM(F16)` collapse (Adicionais all months; Extras Jan/Feb/Nov/Dec) — range form.
3. Inconsistent fill-down — superseded: v2 uses merges.
4. `""` vs `–` total-row placeholders — always `–`.
5. Category-label spacer rows + category-level total rows — dropped (headroom + Listas totals).
6. Trailing appended blocks / 200-row gaps / category regressions — contiguous taxonomy order.
7. Text-typed numbers & dates — typed values + single numFmt (§2).
8. Column width / row height drift — canonical per role.
9. Receitas pre-provisioned empty blocks & inconsistent merging — headroom + always-merge.
10. Listas row-277 mislabel, row-300 missing cells — corrected.
11. Referência sheet — omitted.
12. B7B7B7 col-C patch — **moot in v2** (no col C); analysis preserved in §7 Q-B7 for the
    record.
13. Receitas separators — between income categories only.
14. **v2 redesigns (hand-review):** months start col C (no spacer/sub-item col); labels
    merged not filled-down; Mês spans A1:B2; freeze C3; sub-item level composed into col B;
    Listas label area 5→3 cols (months D–O), sheet-section labels in col A; percent-row fill
    CCCCFF + numFmt 0.00%; merged category labels Arial 14 bold centered.
15. **Label normalization + i18n (Phase B):** generic labels sentence-cased, singular/plural
    unified, sheet suffix dropped from per-group percent rows; all generic strings centralized
    in an English-identifier `Labels` struct (pt-BR values), month names included, config
    loader deferred. See §4.4.

## 7. Open questions

1. **Headroom — RESOLVED (2026-06-10):** block rows = **max-entries-per-month** (the busiest
   month sets the count); `headroomRows` default **0**. Rationale: the workbook is regenerated
   from the database on every change ("add expense → DB; generate → file"), so it is never an
   insertion target and spare rows are optional. ⚠ `template-data.xlsx` was hand-filled to
   fixed-3 blocks (a Phase-A template artifact) and therefore does NOT honor this rule for
   sub-3-entry blocks — Phase B regenerates a corrected golden master rather than diffing
   template-data to zero. Zero-entry subcategory handling deferred to G2.
2. **Per-group percent rows** — ✅ RESOLVED (Phase B, 2026-06-10): added with normalized
   labels `% sobre despesas` / `% sobre receita`; see §4.2.
3. **Q-B7 (B7B7B7), for the record only:** medium-gray patch the source put on col C of
   some Variáveis total rows (16/38, Transporte onward only; never Adicionais). Moot in v2
   since col C no longer exists. If a sub-item column ever returns, prefer the "column in
   use" rule (apply only on sheets where ≥1 subcategory has a sub-item).
4. **Merged headroom tails** (blank rows inside B merges) — accepted in golden master;
   re-check render with data in Phase B.
5. **Dólar row semantics** — manual currency-rate cell; confirm placement/format when data
   phase touches the saldo block.

## 8. Source references

- Golden master: `.claude/workbook-template/template-reviewed.xlsx` (untracked, fake data)
- Hand-review diff catalogue: `.claude/workbook-template/review-diff.md`
- Phase-A build ambiguities: `.claude/workbook-template/ambiguities.md`
- Builder: `.claude/scratch/template-builder/` (standalone Go module)
- Per-sheet source digests: `.claude/workbook-dump/digests/*.md` (gitignored)
- Raw source dumps / visual notes: `.claude/workbook-dump/`, `.claude/workbook-visual-notes.md` (gitignored)
- Taxonomy (113 triples): Referência digest §2
