# Phase B Re-review — template.xlsx
**Date:** 2026-06-11  
**Reviewer:** Sonnet subagent (full re-review, not relying on prior conclusions)  
**File:** `.claude/workbook-template/template.xlsx`  
**Spec authority:** `.claude/plans/workbook-generator-spec.md` v2

---

## Summary

**Overall verdict: PASS with 2 minor observations (no blocking defects)**

All structural, formula, and style requirements from spec §2, §4, §4.2, §5, §7 are met. The
two observations (§6 below) are formula-guard inconsistencies that have no practical impact but
are worth noting for the implementation port.

---

## Item-by-item verdict

### 1. Receitas: block sizing, column placement, typed data, SUM formulas, col-B merges

**PASS**

**Block sizing:**
- Salário block: taxonomy MaxEntries=1 (one entry per month), headroom=0 → 1 data row (row 3) + total row (row 4) = correct.
- 13° block: MaxEntries=1 → 1 data row (row 5) + total row (row 6) = correct.

**Column placement (month k: Item=3+3k, Data=4+3k, Valor=5+3k, 1-indexed cols, k=0..11):**
- Janeiro (k=0): Item=C, Data=D, Valor=E ✓
- Fevereiro (k=1): Item=F, Data=G, Valor=H ✓
- Pattern holds through Dezembro (k=11): Item=AJ, Data=AK, Valor=AL ✓
- Evidence: D3=`datetime(2026,1,5)`, G3=`datetime(2026,2,5)`, E3=5000, H3=5000 (openpyxl confirmed)

**Typed dates and floats:**
- D3=`datetime.datetime(2026,1,5)` (typed date, not string)
- G3=`datetime.datetime(2026,2,5)` (typed date)
- E3=5000 (int/float), H3=5000
- D5=`datetime.datetime(2026,1,20)`, G5=`datetime.datetime(2026,2,20)`
- E5=2500, H5=2500
- Months 3–12 data cells: value=None (correct; taxonomy only has Jan+Feb entries)

**SUM formulas (correct orientation, not inverted):**
- Row 4 (Salário total): `SUM(E3:E3)` for Jan, `SUM(H3:H3)` for Feb ... `SUM(AL3:AL3)` for Dec
  — each Valor total-row covers exactly its 1 data row, all 12 months ✓
- Row 6 (13° total): `SUM(E5:E5)` for Jan ... `SUM(AL5:AL5)` for Dec ✓
- SUM ranges go DATA→TOTAL direction (low row to same row, i.e. single-row blocks). Correct.

**Col-B merges include total rows:**
- B3:B4 = 'Salário' (data row 3 + total row 4) ✓
- B5:B6 = '13°' (data row 5 + total row 6) ✓
- A3:A6 = 'Receita' (covers both blocks + both total rows) ✓

---

### 2. Receitas numFmt: DD/MM on Data cells, R$ #,##0.00 on Valor cells, all 12 months

**PASS**

openpyxl confirmed:
- All Data cells in both blocks, all 12 months: `numFmt='DD/MM'` (even months 3–12 where value=None, the format is pre-applied to the column)
- All Valor cells in both blocks, all 12 months: `numFmt='R$ #,##0.00'`
- Total-row Valor cells: `numFmt='R$ #,##0.00'` ✓
- Evidence: D3 `numFmt=DD/MM`, E3 `numFmt=R$ #,##0.00`, D5 `numFmt=DD/MM`, E5 `numFmt=R$ #,##0.00`

---

### 3. Listas: per-group % rows, denominators, pull formulas

**PASS**

**Per-group % rows present:**
Both "% sobre despesas" and "% sobre receita" rows appear for every subcategory group. Example:
- Row 20: `Total Habitação` (CCCCFF)
- Row 21: `% sobre despesas` D=`IF(D28>0,D20/D28,0)` (CCCCFF, numFmt=0.00%) ✓
- Row 22: `% sobre receita` D=`IF(D9>0,D20/D9,0)` (CCCCFF, numFmt=0.00%) ✓
Pattern repeats for all groups in all four expense sections.

**% sobre receita denominators reference Receitas total correctly:**
- All per-group `% sobre receita` rows use `IF(D9>0, groupTotal/D9, 0)` where D9 is the Receitas
  total row in Listas (`SUM(D6:D7)` = sum of Salário + 13° pull-through cells) ✓
- D9 chain: `SUM(D6:D7)` → D6=`Receitas!E4` (Salário total), D7=`Receitas!E6` (13° total) ✓

**Pull formulas reference correct source-sheet total cells:**
Verified against actual total-row row numbers in each expense sheet:

| Listas row | Formula | Source sheet total row | Source formula | Verdict |
|---|---|---|---|---|
| 18 | `Fixas!E6` | Fixas row 6 | `SUM(E3:E5)` (Diarista 3 rows) | ✓ |
| 19 | `Fixas!E9` | Fixas row 9 | `SUM(E7:E8)` (Aluguel 2 rows) | ✓ |
| 23 | `Fixas!E14` | Fixas row 14 | `SUM(E13:E13)` (Netflix) | ✓ |
| 24 | `Fixas!E16` | Fixas row 16 | `SUM(E15:E15)` (Spotify) | ✓ |
| 32 | `Variáveis!E4` | Variáveis row 4 | `SUM(E3:E3)` (Supermercado) | ✓ |
| 33 | `Variáveis!E6` | Variáveis row 6 | `SUM(E5:E5)` (Padaria) | ✓ |
| 37 | `Variáveis!E11` | Variáveis row 11 | `SUM(E10:E10)` (Orion Consultas) | ✓ |
| 38 | `Variáveis!E13` | Variáveis row 13 | `SUM(E12:E12)` (Orion Ração) | ✓ |
| 42 | `Variáveis!E18` | Variáveis row 18 | `SUM(E17:E17)` (Metrô) | ✓ |
| 50 | `Extras!E4` | Extras row 4 | `SUM(E3:E3)` (Médico) | ✓ |
| 51 | `Extras!E6` | Extras row 6 | `SUM(E5:E5)` (Dentista) | ✓ |
| 55 | `Extras!E11` | Extras row 11 | `SUM(E10:E10)` (Carro) | ✓ |
| 56 | `Extras!E13` | Extras row 13 | `SUM(E12:E12)` (Casa) | ✓ |
| 64 | `Adicionais!E4` | Adicionais row 4 | `SUM(E3:E3)` (Viagens) | ✓ |
| 65 | `Adicionais!E6` | Adicionais row 6 | `SUM(E5:E5)` (Jogos) | ✓ |
| 69 | `Adicionais!E11` | Adicionais row 11 | `SUM(E10:E10)` (Presentes) | ✓ |
| 70 | `Adicionais!E13` | Adicionais row 13 | `SUM(E12:E12)` (Papelaria) | ✓ |

All 17 pull formula cross-checks pass.

---

### 4. Saldo chain on Listas

**PASS**

Summary block (rows 79–97):
- Row 79 `Receita` D: `D9` (=Receitas total from pull-through section) ✓
- Row 80 `Investimentos` D: `D13` (Investimentos total placeholder) ✓
- Row 81 `Total renda` D: `SUM(D79:D80)` (receita + investimentos) ✓, numFmt=`R$ #,##0.00` ✓
- Row 82 `Despesas fixas` D: `D28` ✓
- Row 83 `Despesas variáveis` D: `D46` ✓
- Row 84 `Despesas extras` D: `D60` ✓
- Row 85 `Despesas adicionais` D: `D74` ✓
- Row 86 `Total despesas` D: `SUM(D82,D83,D84,D85)` ✓, numFmt=`R$ #,##0.00` ✓
- Row 97 `Saldo` D: `D81-D86` (total renda minus total despesas) ✓, numFmt=`R$ #,##0.00` ✓

Percentage breakdown blocks (rows 87–96):
- Rows 88–91 (% sobre despesas): `IF(D86>0, section/D86, 0)` numFmt=`0.00%` ✓
- Rows 93–96 (% sobre receita): `IF(D81>0, section/D81, 0)` numFmt=`0.00%` ✓

---

### 5. Listas section-header fill: navy 333399, white font

**PASS**

openpyxl confirmed all merged col-A section labels:
- A6 'Receitas': bg=`FF333399`, font_color=`FFFFFFFF`, bold=True ✓
- A18 'Fixas': bg=`FF333399`, font_color=`FFFFFFFF`, bold=True ✓
- A32 'Variáveis': bg=`FF333399`, font_color=`FFFFFFFF`, bold=True ✓
- A50 'Extras': bg=`FF333399`, font_color=`FFFFFFFF`, bold=True ✓
- A64 'Adicionais': bg=`FF333399`, font_color=`FFFFFFFF`, bold=True ✓

Col-B group labels (Habitação, Lazer, etc.) also carry `FF333399` fill ✓

Not black (000000). Spec §7.7 satisfied.

---

### 6. Expense sheets (Fixas/Variáveis/Extras/Adicionais): block sizing, SUM ranges, EN DASH

**PASS**

**Block sizing:**
- Fixas Diarista: 3 data rows (rows 3–5) + total row 6. `SUM(E3:E5)` covers all 3 rows across all 12 months ✓
- Fixas Aluguel: 2 data rows (rows 7–8) + total row 9. `SUM(E7:E8)` covers 2 rows ✓
  - Feb has only 1 entry (row 7=Aluguel, row 8=empty) — merge/headroom handled correctly ✓
- All single-entry subcats (Netflix, Spotify, etc.): 1 data row + total row, `SUM(Estart:Estart)` ✓
- Variáveis single-entry subcats: all 1 data row + total, verified rows 4, 6, 11, 13, 18 ✓
- Extras and Adicionais: all 1-row blocks, verified ✓

**SUM ranges match data rows (all 12 months):**
- Fixas row 6: `SUM(E3:E5)` through `SUM(AL3:AL5)` for all 12 months ✓
- Fixas row 9: `SUM(E7:E8)` through `SUM(AL7:AL8)` ✓
- Single-row blocks: `SUM(Erow:Erow)` pattern consistently applied ✓

**EN DASH (U+2013 '–') in total-row col-D:**
- Fixas rows 6, 9, 14, 16: all confirm `val='–' unicode=['0x2013']` ✓
- Variáveis rows 4, 6, 11, 13, 18: all confirm `unicode=['0x2013']` ✓
- Receitas rows 4, 6: all 12 months confirm `val='–'` ✓

**Col-B merges include total rows:**
- Diarista B3:B6 ✓, Aluguel B7:B9 ✓, Netflix B13:B14 ✓, Spotify B15:B16 ✓
- Col-A category merges: Habitação A3:A9, Lazer A13:A16 ✓

---

### 7. Other issues (formula errors, off-by-one, suspicious patterns)

**OBSERVATION 1 (minor, non-blocking):** Section-level `% sobre receita` formula guard inconsistency

Rows 30, 48, 62, 76 (section `% sobre receita`) use:
```
IF(D{section_total}>0, D{section_total}/D9, 0)
```
Example: `IF(D28>0,D28/D9,0)`

All per-group `% sobre receita` rows use:
```
IF(D9>0, groupTotal/D9, 0)
```

The section-level rows guard on `D{section_total}` rather than `D9`. This means if `D9=0` (no
income) but a section has expenses, the formula would produce `DIV/0`. In practice with real
data this cannot happen (Receitas must be non-zero before expenses make sense), but the guard is
semantically wrong and inconsistent with the per-group rows.

**Recommendation:** Change section-level rows to `IF(D9>0, D28/D9, 0)` etc. in the port.

---

**OBSERVATION 2 (structural note, non-blocking):** Rows 10 and 12 in Fixas are empty (no fill,
no value) — they act as blank spacer rows between the Habitação category and the Lazer category.
This was confirmed by openpyxl (bg=`00000000`). The dump simply skips them. This matches the
spec's category-separator pattern and is correct behavior.

---

**OBSERVATION 3 (non-blocking):** Investimentos block (Listas rows 12–13) has only a placeholder
row with no pull-through formula. Row 13 D: `D12` which references an empty row 12. This is
expected — Investimentos has no entries in the Phase B taxonomy. The chain `D80=D13=D12=0` is
correct for now.

---

## Defect count

**Blocking defects: 0**  
**Minor observations: 3** (all noted above; only Observation 1 warrants a formula fix in the port)

---

## Double-check list (for LibreOffice visual verification)

1. **Receitas sheet** — Open sheet, confirm D3 displays "05/01" (not a raw date number), E3
   displays "R$ 5.000,00". Check D4 shows "–" and E4 shows the SUM result (R$ 5.000,00).
   Repeat for rows 5–6 (13° block): D5="20/01", E5="R$ 2.500,00".

2. **Receitas numFmt months 3–12** — Click J3 (Março Data column, Salário block): cell should
   be empty but format bar should show "DD/MM". Click K3: format bar should show "R$ #,##0.00".

3. **Fixas Diarista block** — Rows 3–6: confirm 3 data rows (Diarista, Diarista, Diarista) then
   a Total row. Jan column E6 should show "=SUM(E3:E5)". Feb (H6) also `=SUM(H3:H5)`.
   Aluguel block rows 7–9: row 8 Feb (col H) should be EMPTY (only 1 Feb entry), but H9
   `=SUM(H7:H8)` still covers the 2-row range.

4. **Listas de itens — section fill** — Scroll col A: all section labels (Receitas, Fixas,
   Variáveis, Extras, Adicionais) should appear in navy blue with white text, not black.
   Col B group labels (Habitação, Lazer, etc.) should also be navy.

5. **Listas de itens — % rows** — Rows 21–22 (Habitação group): confirm CCCCFF (light blue)
   fill and that the cells display "0,00%" format. Row 21 D formula should be
   `=IF(D28>0,D20/D28,0)`.

6. **Listas de itens — Saldo** — Row 97: confirm D97 = `=D81-D86` and displays in R$ format.
   With Jan data: D81 should be 7500 (Salário 5000 + 13° 2500), D86 should sum all expenses.
   D97 = D81 - D86 should show a negative number (expenses > income in this fake dataset).

7. **Listas de itens — pull formulas** — Click D18: formula bar should show `=Fixas!E6`.
   Click D32: should show `=Variáveis!E4`. Confirm both resolve to non-zero Jan values.

8. **EN DASH** — In Fixas row 6 col D, in Receitas row 4 col D: confirm the character is
   "–" (en dash), not "-" (hyphen) or "—" (em dash). Visually it should be a medium-width dash.
