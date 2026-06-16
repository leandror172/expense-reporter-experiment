# Phase B Data Extract — template-data.xlsx

Generated: 2026-06-10

---

## 1. Column Model Confirmation

Row 1: month banners (A1='Mês'; C1='Janeiro', F1='Fevereiro', I1='Março', …)
Row 2: Item/Data/Valor headers per month triple
Row 3+: data blocks

Month k (0-based) → Item col = 3+3k, Data col = 4+3k, Valor col = 5+3k
- k=0 (Janeiro):   C/D/E
- k=1 (Fevereiro): F/G/H
- k=2 (Março):     I/J/K
- ...
- k=11 (Dezembro): AJ/AK/AL

Total date cell (col D of each month triple on the Total row) contains the
string '–' (U+2013 EN DASH), not empty and not '-' (U+002D HYPHEN-MINUS).

---

## 2. Sheet Structures and Block Definitions

### Sheet: Fixas  (max_row=21, max_col=38)

| Categoria  | Subcategoria | Block rows | Data rows | Total row |
|-----------|--------------|-----------|-----------|-----------|
| Habitação  | Diarista     | 3–5       | 3         | 6         |
| Habitação  | Aluguel      | 7–9       | 3         | 10        |
| Lazer      | Netflix      | 14–16     | 3 (1 filled, 2 empty) | 17 |
| Lazer      | Spotify      | 18–20     | 3 (1 filled, 2 empty) | 21 |

Gap rows: 11–13 (empty, separating Habitação from Lazer)

### Sheet: Variáveis  (max_row=32, max_col=38)

| Categoria              | Subcategoria       | Block rows | Data rows | Total row |
|-----------------------|-------------------|-----------|-----------|-----------|
| Alimentação / Limpeza  | Supermercado       | 3–5       | 3         | 6         |
| Alimentação / Limpeza  | Padaria            | 7–9       | 3         | 10        |
| Pets                   | Orion - Consultas  | 14–16     | 3 (1 filled, 2 empty) | 17 |
| Pets                   | Orion - Ração      | 18–20     | 3 (1 filled, 2 empty) | 21 |
| Transporte             | Metrô              | 25–27     | 3 (2 filled, 1 empty) | 28 |

Gap rows: 11–13 (Alim→Pets), 22–24 (Pets→Transporte), 29–32 (after Metrô)

### Sheet: Extras  (max_row=21, max_col=38)

| Categoria               | Subcategoria | Block rows | Data rows | Total row |
|------------------------|--------------|-----------|-----------|-----------|
| Saúde                   | Médico       | 3–5       | 3 (1 filled, 2 empty) | 6 |
| Saúde                   | Dentista     | 7–9       | 3         | 10        |
| Manutenção / prevenção  | Carro        | 14–16     | 3 (1 filled, 2 empty) | 17 |
| Manutenção / prevenção  | Casa         | 18–20     | 3         | 21        |

Gap rows: 11–13 (Saúde→Manutenção)

### Sheet: Adicionais  (max_row=21, max_col=38)

| Categoria | Subcategoria | Block rows | Data rows | Total row |
|----------|--------------|-----------|-----------|-----------|
| Lazer     | Viagens      | 3–5       | 3 (2 filled, 1 empty) | 6 |
| Lazer     | Jogos        | 7–9       | 3         | 10        |
| Outros    | Presentes    | 14–16     | 3 (2 filled, 1 empty) | 17 |
| Outros    | Papelaria    | 18–20     | 3         | 21        |

Gap rows: 11–13 (Lazer→Outros)

### Sheet: Receitas  (max_row=10, max_col=38)

| Categoria | Subcategoria | Block rows | Data rows | Total row |
|----------|--------------|-----------|-----------|-----------|
| Receita   | Salário      | 3–5       | 3         | 6         |
| Receita   | 13°          | 7–9       | 3         | 10        |

---

## 3. Exact Subcategoria Label Strings

| Sheet      | Label (col B)          | Dash char     | Unicode  |
|-----------|------------------------|---------------|----------|
| Fixas     | Diarista               | —             | —        |
| Fixas     | Aluguel                | —             | —        |
| Fixas     | Netflix                | —             | —        |
| Fixas     | Spotify                | —             | —        |
| Variáveis | Supermercado           | —             | —        |
| Variáveis | Padaria                | —             | —        |
| Variáveis | Orion - Consultas      | hyphen-minus  | U+002D   |
| Variáveis | Orion - Ração          | hyphen-minus  | U+002D   |
| Variáveis | Metrô                  | —             | —        |
| Extras    | Médico                 | —             | —        |
| Extras    | Dentista               | —             | —        |
| Extras    | Carro                  | —             | —        |
| Extras    | Casa                   | —             | —        |
| Adicionais| Viagens                | —             | —        |
| Adicionais| Jogos                  | —             | —        |
| Adicionais| Presentes              | —             | —        |
| Adicionais| Papelaria              | —             | —        |
| Receitas  | Salário                | —             | —        |
| Receitas  | 13°                   | —             | —        |

**NOTE: Mismatch between data sheets and Listas de itens**
- Variáveis col B: `Orion - Consultas` and `Orion - Ração` use **hyphen-minus** (U+002D)
- Listas de itens col C rows 31–32: `Orion — Consultas` and `Orion — Ração` use **em dash** (U+2014)
- These are different characters. The Go builder must use the data-sheet form (hyphen-minus)
  for col B labels, and the Listas form (em dash) for the Listas subcategoria column.

---

## 4. Entry Data Per Sheet / Block / Month

Date pattern: dates are stored as Excel datetime values (datetime.datetime in openpyxl),
all dates are in January 2026 regardless of the month column — the day-of-month increments
with each row and with each month column. This is fake/seed data.

All Valor cells are type `int`. No text-typed numbers found.

### Date generation formulas (observed pattern):

For blocks with multiple rows per month, dates shift: row i in month k has
day = (base_day + i + k), e.g.:
- Diarista month 0: days 1, 2, 3  (Jan 01, 02, 03)
- Diarista month 1: days 2, 3, 4  (Jan 02, 03, 04)
- Diarista month 11: days 12, 13, 14

For blocks with 1 filled row per month (Netflix, Spotify, etc.):
- date day = k+1 (month 0 → day 1, month 11 → day 12)

---

### Sheet: Fixas

#### Habitação / Diarista (3 data rows, all filled every month)

| Month      | Row 1 (item, date, valor) | Row 2          | Row 3          |
|-----------|--------------------------|----------------|----------------|
| Janeiro   | (Diarista, 01/01/2026, 3)| (Diarista, 02/01/2026, 4) | (Diarista, 03/01/2026, 5) |
| Fevereiro | (Diarista, 02/01/2026, 4)| (Diarista, 03/01/2026, 5) | (Diarista, 04/01/2026, 6) |
| Março     | (Diarista, 03/01/2026, 5)| (Diarista, 04/01/2026, 6) | (Diarista, 05/01/2026, 7) |
| Abril     | (Diarista, 04/01/2026, 6)| (Diarista, 05/01/2026, 7) | (Diarista, 06/01/2026, 8) |
| Maio      | (Diarista, 05/01/2026, 7)| (Diarista, 06/01/2026, 8) | (Diarista, 07/01/2026, 9) |
| Junho     | (Diarista, 06/01/2026, 8)| (Diarista, 07/01/2026, 9) | (Diarista, 08/01/2026, 10)|
| Julho     | (Diarista, 07/01/2026, 9)| (Diarista, 08/01/2026, 10)| (Diarista, 09/01/2026, 11)|
| Agosto    | (Diarista, 08/01/2026, 10)|(Diarista, 09/01/2026, 11)| (Diarista, 10/01/2026, 12)|
| Setembro  | (Diarista, 09/01/2026, 11)|(Diarista, 10/01/2026, 12)| (Diarista, 11/01/2026, 13)|
| Outubro   | (Diarista, 10/01/2026, 12)|(Diarista, 11/01/2026, 13)| (Diarista, 12/01/2026, 14)|
| Novembro  | (Diarista, 11/01/2026, 13)|(Diarista, 12/01/2026, 14)| (Diarista, 13/01/2026, 15)|
| Dezembro  | (Diarista, 12/01/2026, 14)|(Diarista, 13/01/2026, 15)| (Diarista, 14/01/2026, 16)|

Pattern: valor[row_i][month_k] = 3 + i + k  (i=0..2, k=0..11)
Date day: day_of_month = 1 + i + k

#### Habitação / Aluguel (3 data rows, all filled every month)

| Month      | Row 1                         | Row 2                         | Row 3                         |
|-----------|-------------------------------|-------------------------------|-------------------------------|
| Janeiro   | (Aluguel, 02/01/2026, 50)    | (Aluguel, 03/01/2026, 51)    | (Aluguel, 04/01/2026, 52)    |
| Fevereiro | (Aluguel, 03/01/2026, 51)    | (Aluguel, 04/01/2026, 52)    | (Aluguel, 05/01/2026, 53)    |
| Março     | (Aluguel, 04/01/2026, 52)    | (Aluguel, 05/01/2026, 53)    | (Aluguel, 06/01/2026, 54)    |
| Abril     | (Aluguel, 05/01/2026, 53)    | (Aluguel, 06/01/2026, 54)    | (Aluguel, 07/01/2026, 55)    |
| Maio      | (Aluguel, 06/01/2026, 54)    | (Aluguel, 07/01/2026, 55)    | (Aluguel, 08/01/2026, 56)    |
| Junho     | (Aluguel, 07/01/2026, 55)    | (Aluguel, 08/01/2026, 56)    | (Aluguel, 09/01/2026, 57)    |
| Julho     | (Aluguel, 08/01/2026, 56)    | (Aluguel, 09/01/2026, 57)    | (Aluguel, 10/01/2026, 58)    |
| Agosto    | (Aluguel, 09/01/2026, 57)    | (Aluguel, 10/01/2026, 58)    | (Aluguel, 11/01/2026, 59)    |
| Setembro  | (Aluguel, 10/01/2026, 58)    | (Aluguel, 11/01/2026, 59)    | (Aluguel, 12/01/2026, 60)    |
| Outubro   | (Aluguel, 11/01/2026, 59)    | (Aluguel, 12/01/2026, 60)    | (Aluguel, 13/01/2026, 61)    |
| Novembro  | (Aluguel, 12/01/2026, 60)    | (Aluguel, 13/01/2026, 61)    | (Aluguel, 14/01/2026, 62)    |
| Dezembro  | (Aluguel, 13/01/2026, 61)    | (Aluguel, 14/01/2026, 62)    | (Aluguel, 15/01/2026, 63)    |

Pattern: valor[row_i][month_k] = 50 + i + k
Date day: day_of_month = 2 + i + k

#### Lazer / Netflix (3 data rows; only row 1 filled, rows 2–3 empty every month)

| Month      | Row 1 (filled)              | Row 2 | Row 3 |
|-----------|-----------------------------|-------|-------|
| Janeiro   | (Netflix, 01/01/2026, 30)  | —     | —     |
| Fevereiro | (Netflix, 02/01/2026, 31)  | —     | —     |
| Março     | (Netflix, 03/01/2026, 32)  | —     | —     |
| Abril     | (Netflix, 04/01/2026, 33)  | —     | —     |
| Maio      | (Netflix, 05/01/2026, 34)  | —     | —     |
| Junho     | (Netflix, 06/01/2026, 35)  | —     | —     |
| Julho     | (Netflix, 07/01/2026, 36)  | —     | —     |
| Agosto    | (Netflix, 08/01/2026, 37)  | —     | —     |
| Setembro  | (Netflix, 09/01/2026, 38)  | —     | —     |
| Outubro   | (Netflix, 10/01/2026, 39)  | —     | —     |
| Novembro  | (Netflix, 11/01/2026, 40)  | —     | —     |
| Dezembro  | (Netflix, 12/01/2026, 41)  | —     | —     |

Pattern: valor[month_k] = 30 + k; date day = k + 1

#### Lazer / Spotify (3 data rows; only row 1 filled, rows 2–3 empty every month)

| Month      | Row 1 (filled)             |
|-----------|----------------------------|
| Janeiro   | (Spotify, 01/01/2026, 5)  |
| Fevereiro | (Spotify, 02/01/2026, 6)  |
| Março     | (Spotify, 03/01/2026, 7)  |
| Abril     | (Spotify, 04/01/2026, 8)  |
| Maio      | (Spotify, 05/01/2026, 9)  |
| Junho     | (Spotify, 06/01/2026, 10) |
| Julho     | (Spotify, 07/01/2026, 11) |
| Agosto    | (Spotify, 08/01/2026, 12) |
| Setembro  | (Spotify, 09/01/2026, 13) |
| Outubro   | (Spotify, 10/01/2026, 14) |
| Novembro  | (Spotify, 11/01/2026, 15) |
| Dezembro  | (Spotify, 12/01/2026, 16) |

Pattern: valor[month_k] = 5 + k; date day = k + 1

---

### Sheet: Variáveis

#### Alimentação / Limpeza / Supermercado (3 data rows, all filled, item='Despesa')

| Month      | Row 1                      | Row 2                      | Row 3                     |
|-----------|----------------------------|----------------------------|---------------------------|
| Janeiro   | (Despesa, 01/01/2026, 45) | (Despesa, 01/01/2026, 72) | (Despesa, 01/01/2026, 20)|
| Fevereiro | (Despesa, 02/01/2026, 46) | (Despesa, 02/01/2026, 73) | (Despesa, 02/01/2026, 21)|
| Março     | (Despesa, 03/01/2026, 47) | (Despesa, 03/01/2026, 74) | (Despesa, 03/01/2026, 22)|
| Abril     | (Despesa, 04/01/2026, 48) | (Despesa, 04/01/2026, 75) | (Despesa, 04/01/2026, 23)|
| Maio      | (Despesa, 05/01/2026, 49) | (Despesa, 05/01/2026, 76) | (Despesa, 05/01/2026, 24)|
| Junho     | (Despesa, 06/01/2026, 50) | (Despesa, 06/01/2026, 77) | (Despesa, 06/01/2026, 25)|
| Julho     | (Despesa, 07/01/2026, 51) | (Despesa, 07/01/2026, 78) | (Despesa, 07/01/2026, 26)|
| Agosto    | (Despesa, 08/01/2026, 52) | (Despesa, 08/01/2026, 79) | (Despesa, 08/01/2026, 27)|
| Setembro  | (Despesa, 09/01/2026, 53) | (Despesa, 09/01/2026, 80) | (Despesa, 09/01/2026, 28)|
| Outubro   | (Despesa, 10/01/2026, 54) | (Despesa, 10/01/2026, 81) | (Despesa, 10/01/2026, 29)|
| Novembro  | (Despesa, 11/01/2026, 55) | (Despesa, 11/01/2026, 82) | (Despesa, 11/01/2026, 30)|
| Dezembro  | (Despesa, 12/01/2026, 56) | (Despesa, 12/01/2026, 83) | (Despesa, 12/01/2026, 31)|

Note: all 3 rows in same month share the same date (day = k+1, month = January 2026).
Valores: row1 = 45+k, row2 = 72+k, row3 = 20+k

#### Alimentação / Limpeza / Padaria (3 data rows, all filled, item='Despesa')

| Month      | Row 1                     | Row 2                      | Row 3                    |
|-----------|---------------------------|----------------------------|--------------------------|
| Janeiro   | (Despesa, 01/01/2026, 8) | (Despesa, 01/01/2026, 10) | (Despesa, 01/01/2026, 1)|
| Fevereiro | (Despesa, 02/01/2026, 9) | (Despesa, 02/01/2026, 11) | (Despesa, 02/01/2026, 2)|
| Março     | (Despesa, 03/01/2026, 10)| (Despesa, 03/01/2026, 12) | (Despesa, 03/01/2026, 3)|
| Abril     | (Despesa, 04/01/2026, 11)| (Despesa, 04/01/2026, 13) | (Despesa, 04/01/2026, 4)|
| Maio      | (Despesa, 05/01/2026, 12)| (Despesa, 05/01/2026, 14) | (Despesa, 05/01/2026, 5)|
| Junho     | (Despesa, 06/01/2026, 13)| (Despesa, 06/01/2026, 15) | (Despesa, 06/01/2026, 6)|
| Julho     | (Despesa, 07/01/2026, 14)| (Despesa, 07/01/2026, 16) | (Despesa, 07/01/2026, 7)|
| Agosto    | (Despesa, 08/01/2026, 15)| (Despesa, 08/01/2026, 17) | (Despesa, 08/01/2026, 8)|
| Setembro  | (Despesa, 09/01/2026, 16)| (Despesa, 09/01/2026, 18) | (Despesa, 09/01/2026, 9)|
| Outubro   | (Despesa, 10/01/2026, 17)| (Despesa, 10/01/2026, 19) | (Despesa, 10/01/2026, 10)|
| Novembro  | (Despesa, 11/01/2026, 18)| (Despesa, 11/01/2026, 20) | (Despesa, 11/01/2026, 11)|
| Dezembro  | (Despesa, 12/01/2026, 19)| (Despesa, 12/01/2026, 21) | (Despesa, 12/01/2026, 12)|

Valores: row1 = 8+k, row2 = 10+k, row3 = 1+k

#### Pets / Orion - Consultas (3 data rows; only row 1 filled, rows 2–3 empty)

| Month      | Row 1 (filled)                       |
|-----------|--------------------------------------|
| Janeiro   | (Despesa, 01/01/2026, 100)          |
| Fevereiro | (Despesa, 02/01/2026, 101)          |
| ...       | valor = 100+k, date day = k+1        |
| Dezembro  | (Despesa, 12/01/2026, 111)          |

#### Pets / Orion - Ração (3 data rows; only row 1 filled, rows 2–3 empty)

Same pattern as Orion - Consultas: valor = 100+k, date day = k+1

#### Transporte / Metrô (3 data rows; rows 1–2 filled, row 3 empty)

| Month      | Row 1                    | Row 2                    | Row 3 |
|-----------|--------------------------|--------------------------|-------|
| Janeiro   | (Despesa, 01/01/2026, 7)| (Despesa, 01/01/2026, 7)| —     |
| Fevereiro | (Despesa, 02/01/2026, 8)| (Despesa, 02/01/2026, 8)| —     |
| ...       | valor = 7+k              | valor = 7+k (same)       | —     |
| Dezembro  | (Despesa, 12/01/2026, 18)|(Despesa, 12/01/2026, 18)| —    |

Both rows have identical date and valor per month. valor = 7+k, date day = k+1

---

### Sheet: Extras

#### Saúde / Médico (3 data rows; only row 1 filled)

valor = 7+k, date day = k+1, item='Despesa'

#### Saúde / Dentista (3 data rows, all filled)

All 3 rows identical per month: valor = 7+k each, date day = k+1, item='Despesa'

#### Manutenção / prevenção / Carro (3 data rows; only row 1 filled)

valor = 3+k, date day = k+1, item='Despesa'

#### Manutenção / prevenção / Casa (3 data rows, all filled)

All 3 rows identical per month: valor = 3+k each, date day = k+1, item='Despesa'

---

### Sheet: Adicionais

#### Lazer / Viagens (3 data rows; rows 1–2 filled, row 3 empty)

| Month      | Row 1                    | Row 2                    |
|-----------|--------------------------|--------------------------|
| Janeiro   | (Despesa, 01/01/2026, 3)| (Despesa, 01/01/2026, 5)|
| Fevereiro | (Despesa, 02/01/2026, 4)| (Despesa, 02/01/2026, 6)|
| ...       | valor1 = 3+k             | valor2 = 5+k             |
| Dezembro  | (Despesa, 12/01/2026, 14)|(Despesa, 12/01/2026, 16)|

#### Lazer / Jogos (3 data rows, all filled)

| Month     | Row 1                    | Row 2                    | Row 3                    |
|----------|--------------------------|--------------------------|--------------------------|
| Janeiro  | (Despesa, 01/01/2026, 3)| (Despesa, 01/01/2026, 5)| (Despesa, 01/01/2026, 5)|
| ...      | valor = 3+k              | valor = 5+k              | valor = 5+k              |

#### Outros / Presentes (3 data rows; rows 1–2 filled, row 3 empty)

Row 1 and Row 2 each have valor = 5+k, date day = k+1, item='Despesa'

#### Outros / Papelaria (3 data rows, all filled)

All 3 rows: valor = 5+k, date day = k+1, item='Despesa'

---

### Sheet: Receitas

#### Receita / Salário (3 data rows, all filled)

| Month      | Row 1                           | Row 2                       | Row 3                       |
|-----------|----------------------------------|-----------------------------|-----------------------------|
| Janeiro   | (Receita 1, 01/01/2026, 1000)  | (Receita 2, 02/01/2026, 2) | (Receita 3, 03/01/2026, 3) |
| Fevereiro | (Receita 2, 02/01/2026, 2000)  | (Receita 3, 03/01/2026, 3) | (Receita 4, 04/01/2026, 4) |
| Março     | (Receita 3, 03/01/2026, 3000)  | (Receita 4, 04/01/2026, 4) | (Receita 5, 05/01/2026, 5) |
| Abril     | (Receita 4, 04/01/2026, 4000)  | (Receita 5, 05/01/2026, 5) | (Receita 6, 06/01/2026, 6) |
| Maio      | (Receita 5, 05/01/2026, 4000)  | (Receita 6, 06/01/2026, 6) | (Receita 7, 07/01/2026, 7) |
| Junho     | (Receita 6, 06/01/2026, 4000)  | (Receita 7, 07/01/2026, 7) | (Receita 8, 08/01/2026, 8) |
| Julho     | (Receita 7, 07/01/2026, 4000)  | (Receita 8, 08/01/2026, 8) | (Receita 9, 09/01/2026, 9) |
| Agosto    | (Receita 8, 08/01/2026, 4000)  | (Receita 9, 09/01/2026, 9) | (Receita 10, 10/01/2026, 10)|
| Setembro  | (Receita 9, 09/01/2026, 4000)  | (Receita 10, 10/01/2026, 10)|(Receita 11, 11/01/2026, 11)|
| Outubro   | (Receita 10, 10/01/2026, 4000) | (Receita 11, 11/01/2026, 11)|(Receita 12, 12/01/2026, 12)|
| Novembro  | (Receita 11, 11/01/2026, 4000) | (Receita 12, 12/01/2026, 12)|(Receita 13, 13/01/2026, 13)|
| Dezembro  | (Receita 12, 12/01/2026, 4000) | (Receita 13, 13/01/2026, 13)|(Receita 14, 14/01/2026, 14)|

Row 1: item = "Receita k+1", valor = k==0? 1000 : (k<=3? k*1000 : 4000), date day = k+1
Row 2: item = "Receita k+2", valor = k+2, date day = k+2
Row 3: item = "Receita k+3", valor = k+3, date day = k+3

#### Receita / 13° (3 data rows, all filled)

| Month      | Row 1                       | Row 2                       | Row 3                       |
|-----------|-----------------------------|-----------------------------|------------------------------|
| Janeiro   | (Receita 1, 01/01/2026, 1) | (Receita 2, 02/01/2026, 2) | (Receita 3, 03/01/2026, 3) |
| Fevereiro | (Receita 2, 02/01/2026, 2) | (Receita 3, 03/01/2026, 3) | (Receita 4, 04/01/2026, 4) |
| ...       | item="Receita k+1", val=k+1 | item="Receita k+2", val=k+2 | item="Receita k+3", val=k+3 |
| Dezembro  | (Receita 12, 12/01/2026, 12)|(Receita 13, 13/01/2026, 13)|(Receita 14, 14/01/2026, 14)|

Pattern: all rows: item = "Receita (row_i + k + 1)", valor = row_i + k + 1, date day = row_i + k + 1

---

## 5. Listas de itens — Row Map

Sheet: max_row=79, max_col=15

Header rows:
- Row 3: col D='Janeiro', col E='Fevereiro' (month headers)
- Row 5: col D='Valor', col E='Valor' (section headers)

### Receitas section (rows 6–14)

| Row | A         | B    | C               | D formula (Jan)                 |
|-----|-----------|------|-----------------|---------------------------------|
| 6   | Receitas  |      | Salário         | =Receitas!E6                    |
| 7   |           |      | 13°             | =Receitas!E10                   |
| 8   | [empty]   |      |                 |                                 |
| 9   |           |      | Total           | =SUM(D6:D7)                     |
| 10  | [empty]   |      |                 |                                 |
| 11  | [empty]   |      |                 |                                 |
| 12  |           |      | Investimentos   | (no formula, empty value)       |
| 13  |           |      | Total           | =D12                            |
| 14  |           |      | % sobre Receita | =IF(D9>0,D13/D9,0)              |

### Fixas section (rows 18–26)

| Row | A      | B                    | C        | D formula (Jan)              |
|-----|--------|----------------------|----------|------------------------------|
| 15  | [empty]|                      |          |                              |
| 16  | [empty]|                      |          |                              |
| 17  | [empty]|                      |          |                              |
| 18  | Fixas  | Habitação            | Diarista | =Fixas!E6                    |
| 19  |        |                      | Aluguel  | =Fixas!E10                   |
| 20  |        | Total Habitação      |          | =SUM(D18:D19)                |
| 21  |        | Lazer                | Netflix  | =Fixas!E17                   |
| 22  |        |                      | Spotify  | =Fixas!E21                   |
| 23  |        | Total Lazer          |          | =SUM(D21:D22)                |
| 24  |        | Total despesas fixas |          | =SUM(D20,D23)                |
| 25  | [empty]|                      |          |                              |
| 26  |        | % sobre Receita      |          | =IF(D24>0,D24/D9,0)          |
| 27  | [empty]|                      |          |                              |

### Variáveis section (rows 28–38)

| Row | A         | B                            | C                  | D formula (Jan)              |
|-----|-----------|------------------------------|--------------------|------------------------------|
| 28  | Variáveis | Alimentação / Limpeza        | Supermercado       | =Variáveis!E6                |
| 29  |           |                              | Padaria            | =Variáveis!E10               |
| 30  |           | Total Alimentação / Limpeza  |                    | =SUM(D28:D29)                |
| 31  |           | Pets                         | Orion — Consultas  | =Variáveis!E17               |
| 32  |           |                              | Orion — Ração      | =Variáveis!E21               |
| 33  |           | Total Pets                   |                    | =SUM(D31:D32)                |
| 34  |           | Transporte                   | Metrô              | =Variáveis!E28               |
| 35  |           | Total Transporte              |                    | =SUM(D34)                    |
| 36  |           | Total despesas variáveis     |                    | =SUM(D30,D33,D35)            |
| 37  | [empty]   |                              |                    |                              |
| 38  |           | % sobre Receita              |                    | =IF(D36>0,D36/D9,0)          |
| 39  | [empty]   |                              |                    |                              |

### Extras section (rows 40–48)

| Row | A      | B                           | C                           | D formula (Jan)            |
|-----|--------|-----------------------------|-----------------------------|----------------------------|
| 40  | Extras | Saúde                       | Médico                      | =Extras!E6                 |
| 41  |        |                             | Dentista                    | =Extras!E10                |
| 42  |        | Total Saúde                 | Total Saúde                 | =SUM(D40:D41)              |
| 43  |        | Manutenção / prevenção      | Carro                       | =Extras!E17                |
| 44  |        |                             | Casa                        | =Extras!E21                |
| 45  |        | Total Manutenção / prevenção| Total Manutenção / prevenção| =SUM(D43:D44)              |
| 46  |        | Total despesas extras       |                             | =SUM(D42,D45)              |
| 47  | [empty]|                             |                             |                            |
| 48  |        | % sobre Receita             | % sobre Receita             | =IF(D46>0,D46/D9,0)        |
| 49  | [empty]|                             |                             |                            |

Note: Rows 42 and 45 have the categoria-group total label duplicated in BOTH col B and col C.
Same for row 48: `% sobre Receita` appears in both col B and col C.

### Adicionais section (rows 50–58)

| Row | A          | B                          | C         | D formula (Jan)              |
|-----|------------|----------------------------|-----------|------------------------------|
| 50  | Adicionais | Lazer                      | Viagens   | =Adicionais!E6               |
| 51  |            |                            | Jogos     | =Adicionais!E10              |
| 52  |            | Total Lazer                |           | =SUM(D50:D51)                |
| 53  |            | Outros                     | Presentes | =Adicionais!E17              |
| 54  |            |                            | Papelaria | =Adicionais!E21              |
| 55  |            | Total Outros               |           | =SUM(D53:D54)                |
| 56  |            | Total despesas adicionais  |           | =SUM(D52,D55)                |
| 57  | [empty]    |                            |           |                              |
| 58  |            | % sobre Receita            |           | =IF(D56>0,D56/D9,0)          |
| 59  | [empty]    |                            |           |                              |
| 60  | [empty]    |                            |           |                              |

### Saldo block (rows 61–79)

| Row | A                    | D formula (Jan)                   |
|-----|----------------------|-----------------------------------|
| 61  | Receita              | =D9                               |
| 62  | Investimentos        | =D13                              |
| 63  | Total Renda          | =SUM(D61:D62)                     |
| 64  | Despesas fixas       | =D24                              |
| 65  | Despesas variáveis   | =D36                              |
| 66  | Despesas extras      | =D46                              |
| 67  | Despesas adicionais  | =D56                              |
| 68  | Total Despesas       | =SUM(D64,D65,D66,D67)             |
| 69  | Porcentagem da Despesa | (header, no value)              |
| 70  | Fixas                | =IF(D68>0,D64/D68,0)              |
| 71  | Variáveis            | =IF(D68>0,D65/D68,0)              |
| 72  | Extras               | =IF(D68>0,D66/D68,0)              |
| 73  | Adicionais           | =IF(D68>0,D67/D68,0)              |
| 74  | Porcentagem da Renda | (header, no value)                |
| 75  | Fixas                | =IF(D63>0,D64/D63,0)              |
| 76  | Variáveis            | =IF(D63>0,D65/D63,0)              |
| 77  | Extras               | =IF(D63>0,D66/D63,0)              |
| 78  | Adicionais           | =IF(D63>0,D67/D63,0)              |
| 79  | Saldo                | =D63-D68                          |

---

## 6. Per-Group % rows — Current Status

**Status: PER-GROUP % SOBRE DESPESAS ROWS DO NOT EXIST.**

The current template has `% sobre Receita` rows AFTER each sheet's grand total
(not after each categoria-group total). There are NO `% sobre despesas` per-group rows anywhere.

Current `% sobre Receita` rows (at sheet level, not group level):
- Row 26: after "Total despesas fixas" (row 24)
- Row 38: after "Total despesas variáveis" (row 36)
- Row 48: after "Total despesas extras" (row 46)
- Row 58: after "Total despesas adicionais" (row 56)

If per-group `% sobre despesas` / `% sobre receita` rows were to be added, they would
be inserted AFTER each categoria-group total row (currently absent). Insertion points:
- After row 20 (Total Habitação) → new row between 20 and 21
- After row 23 (Total Lazer, Fixas) → new row between 23 and 24
- After row 30 (Total Alimentação / Limpeza) → new row between 30 and 31
- After row 33 (Total Pets) → new row between 33 and 34
- After row 35 (Total Transporte) → new row between 35 and 36
- After row 42 (Total Saúde) → new row between 42 and 43
- After row 45 (Total Manutenção / prevenção) → new row between 45 and 46
- After row 52 (Total Lazer, Adicionais) → new row between 52 and 53
- After row 55 (Total Outros) → new row between 55 and 56

---

## 7. Surprises and Anomalies

1. **Dash mismatch**: Variáveis data sheet uses `Orion - Consultas` / `Orion - Ração`
   (hyphen-minus U+002D), but Listas de itens uses `Orion — Consultas` / `Orion — Ração`
   (em dash U+2014). These are inconsistent — the builder must replicate both exactly.

2. **Partial fill pattern**: Each block has a fixed number of allocated data rows (always 3),
   but not all rows are filled. The fill count varies by subcategoria (1, 2, or 3 rows used).
   Empty rows are truly empty (no item/date/valor).

3. **Same-date rows**: In Variáveis, Extras, and Adicionais, multiple rows within the same
   month-column for a block share identical dates (all are day k+1 of the fake January sequence).
   Only Fixas/Receitas blocks have date-per-row variation within a month.

4. **Receitas row 1 valor anomaly**: For Salário, row 1 has valor=1000 (Jan), 2000 (Feb),
   3000 (Mar), then 4000 for all subsequent months. Rows 2–3 have tiny values (2, 3, 4…).
   The 4000 "salary" is only in row 1 col E (item, date, valor triple for each month).

5. **Total date column**: The Data cell in Total rows contains the string '–' (U+2013 EN DASH),
   not null, not hyphen-minus. Builder must write this exact string.

6. **Extras Listas duplicate labels**: Rows 42 and 45 have the group-total label in BOTH
   col B and col C (e.g., 'Total Saúde' appears in B42 AND C42). Row 48 has '% sobre Receita'
   in both B48 and C48. This appears to be intentional formatting in the template.

7. **No % sobre despesas per-group**: Confirmed absent. The task says they SHOULD NOT yet exist —
   confirmed correct. The Listas sheet has only sheet-level % sobre Receita rows (26, 38, 48, 58).
