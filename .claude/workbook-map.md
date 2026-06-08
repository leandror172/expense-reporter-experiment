# Workbook Structure Map

Source: `/home/leandror/workspaces/expenses/code/Planilha_Normalized_Final.xlsx`

## Sheet Inventory

1. `Listas de itens`
2. `Receitas`
3. `Fixas`
4. `Variáveis`
5. `Extras`
6. `Adicionais`
7. `Referência de Categorias`

## Sheet: `Listas de itens`

Rows: 303

### Header rows (first 5)

| Row | A | B | C | D | E | F | G | H |
|-----|---|---|---|---|---|---|---|---|
| 1 |  |  |  |  |  |  |  |  |
| 2 |  |  |  |  |  |  |  |  |
| 3 | Mês |  |  |  |  | Janeiro | Fevereiro | Março |
| 4 |  |  |  |  |  |  |  |  |
| 5 |  |  |  |  |  | Valor | Valor | Valor |

### Month column detection

Month header at row 3 (12 month names found):

```
  col A    = Mês
  col F    = Janeiro
  col G    = Fevereiro
  col H    = Março
  col I    = Abril
  col J    = Maio
  col K    = Junho
  col L    = Julho
  col M    = Agosto
  col N    = Setembro
  col O    = Outubro
  col P    = Novembro
  col Q    = Dezembro
```

### Subcategory blocks (column B)

| Row | Name | Total row | Formula sample |
|-----|------|-----------|----------------|
| 30 | Categoria | 0 |  |
| 31 | Fixas | 0 |  |
| 32 | Aquelas que têm | 0 |  |
| 33 | o mesmo montante | 0 |  |
| 34 | mensalmente | 46 | `F46`: `=SUM(F31:F45)` |
| 116 | Variáveis | 121 | `F121`: `=SUM(F116:F120)` |
| 117 | Aquelas que aconte- | 121 | `F121`: `=SUM(F116:F120)` |
| 118 | cem todos os meses, | 121 | `F121`: `=SUM(F116:F120)` |
| 119 | mas podemos tentar | 121 | `F121`: `=SUM(F116:F120)` |
| 120 | reduzir | 121 | `F121`: `=SUM(F116:F120)` |
| 197 | Extras | 200 | `F200`: `=SUM(F197:F199)` |
| 198 | São as despesas extra- | 200 | `F200`: `=SUM(F197:F199)` |
| 199 | ordinárias, para as | 200 | `F200`: `=SUM(F197:F199)` |
| 200 | quais precisamos estar | 0 |  |
| 201 | preparados quando | 0 |  |
| 202 | acontecerem | 207 | `F207`: `=SUM(F204:F206)` |
| 240 | Adicionais | 0 |  |
| 248 | Aquelas que não | 0 |  |
| 251 | precisam acontecer | 256 | `F256`: `=SUM(F240:F253)` |
| 253 | todos os meses | 256 | `F256`: `=SUM(F240:F253)` |

### Merged cells (39 regions)

| Range | Value |
|-------|-------|
| F1:F2 |  |
| A3:D3 | Mês |
| A6:C15 | Receitas |
| A19:C27 | Investimentos
Insira aqui o montante men… |
| B30:C30 | Categoria |
| C31:C48 | Habitação |
| C50:C59 | Lazer |
| C62:C67 | Transporte |
| C69:C77 | Saúde |
| C79:C83 | Educação |
| C86:C93 | Pet |
| D86:D87 | Orion |
| D88:D89 | Lilly |
| C95:C102 | Impostos |
| C104:C110 | Outros |
| C116:C123 | Habitação |
| C125:C131 | Impostos |
| C133:C141 | Transporte |
| C143:C150 | Alimentação / Limpeza |
| C152:C164 | Pet |
| D152:D154 | Orion |
| D155:D158 | Lilly |
| D159:D161 | Ambos |
| C166:C174 | Saúde |
| C176:C183 | Anita |
| C185:C191 | Cuidados pessoais |
| C197:C202 | Saúde |
| C204:C209 | Manutenção/ prevenção |
| C211:C220 | Pets |
| D211:D212 | Orion |
| D213:D215 | Lilly |
| D216:D217 | Ambos |
| C222:C227 | Advogado  |
| C229:C234 | Educação |
| C240:C258 | Lazer |
| C260:C265 | Vestuário |
| C267:C275 | Outros |
| A281:C299 | Saldo |
| A301:C301 | Dólar |

## Sheet: `Receitas`

Rows: 128

### Header rows (first 5)

| Row | A | B | C | D | E | F | G | H |
|-----|---|---|---|---|---|---|---|---|
| 1 | Mês |  | Janeiro |  |  | Fevereiro |  |  |
| 2 |  |  | Item | Data | Valor | Item | Data | Valor |
| 3 | Receita | Salário |  |  |  |  |  |  |
| 4 |  | Arredondamento |  |  |  |  |  |  |
| 5 |  | Ajuda de custo |  |  |  |  |  |  |

### Month column detection

Month header at row 1 (12 month names found):

```
  col A    = Mês
  col C    = Janeiro
  col F    = Fevereiro
  col I    = Março
  col L    = Abril
  col O    = Maio
  col R    = Junho
  col U    = Julho
  col X    = Agosto
  col AA   = Setembro
  col AD   = Outubro
  col AG   = Novembro
  col AJ   = Dezembro
```

### Subcategory blocks (column B)

| Row | Name | Total row | Formula sample |
|-----|------|-----------|----------------|
| 6 | Arredondamento mês anterior | 0 |  |
| 7 | Desconto adiantamento | 0 |  |
| 8 | INSS | 0 |  |
| 9 | IRRF | 0 |  |
| 10 | Cont Assistencial | 0 |  |
| 11 | Assist Médica | 0 |  |
| 12 | Assist Odontológica | 0 |  |
| 13 | Cont Assistencial Parcelamento | 0 |  |
| 14 | Adiantamento próx mês | 0 |  |
| 15 | Arredondamento do mês | 0 |  |
| 16 | Pagamento de Dissídio | 21 | `E21`: `=SUM(E3:E20)` |
| 17 | IRRF Dissídio | 21 | `E21`: `=SUM(E3:E20)` |
| 22 | 13° Integral | 27 | `E27`: `=SUM(E22:E26)` |
| 23 | Média HE 13° | 27 | `E27`: `=SUM(E22:E26)` |
| 24 | Adiantamento 13° | 27 | `E27`: `=SUM(E22:E26)` |
| 25 | INSS | 27 | `E27`: `=SUM(E22:E26)` |
| 26 | IRRF | 27 | `E27`: `=SUM(E22:E26)` |
| 28 | Férias Normais | 32 | `E32`: `=SUM(E28:E31)` |
| 29 | Adicional 1/3 S/ Férias | 32 | `E32`: `=SUM(E28:E31)` |
| 30 | Liquido Ferias Normais | 32 | `E32`: `=SUM(E28:E31)` |
| 31 | IRRF Férias | 32 | `E32`: `=SUM(E28:E31)` |
| 33 | Presente | 36 | `E36`: `=SUM(E33:E35)` |

### Merged cells (34 regions)

| Range | Value |
|-------|-------|
| A1:B1 | Mês |
| C1:E1 | Janeiro |
| F1:H1 | Fevereiro |
| I1:K1 | Março |
| L1:N1 | Abril |
| O1:Q1 | Maio |
| R1:T1 | Junho |
| U1:W1 | Julho |
| X1:Z1 | Agosto |
| AA1:AC1 | Setembro |
| AD1:AF1 | Outubro |
| AG1:AI1 | Novembro |
| AJ1:AL1 | Dezembro |
| B33:B36 | Presente |
| B37:B40 |  |
| B41:B44 |  |
| B45:B50 |  |
| B51:B57 |  |
| B58:B61 |  |
| B62:B65 |  |
| B66:B84 |  |
| B85:B93 |  |
| B94:B102 |  |
| B103:B111 |  |
| A116:A124 |  |
| B116:B118 |  |
| B119:B121 |  |
| B122:B124 |  |
| A128:A154 | Outros |
| B128:B132 |  |
| B133:B139 |  |
| B140:B144 |  |
| B145:B149 |  |
| B150:B154 |  |

## Sheet: `Fixas`

Rows: 237

### Header rows (first 5)

| Row | A | B | C | D | E | F | G | H |
|-----|---|---|---|---|---|---|---|---|
| 1 |  |  |  | Janeiro | Janeiro | Janeiro | Fevereiro | Fevereiro |
| 2 |  |  |  | Item | Data | Valor | Item | Data |
| 3 | Habitação | Diarista |  | Diarista Letícia | 17/1 | R$ 200.00 | Diarista Letícia | 21/2 |
| 4 | Habitação | Diarista |  |  |  |  |  |  |
| 5 | Habitação | Diarista |  |  |  |  |  |  |

### Month column detection

Month header at row 1 (12 month names found):

```
  col D    = Janeiro
  col E    = Janeiro
  col F    = Janeiro
  col G    = Fevereiro
  col H    = Fevereiro
  col I    = Fevereiro
  col J    = Março
  col K    = Março
  col L    = Março
  col M    = Abril
  col N    = Abril
  col O    = Abril
  col P    = Maio
  col S    = Junho
  col V    = Julho
  col Y    = Agosto
  col AB   = Setembro
  col AE   = Outubro
  col AH   = Novembro
  col AK   = Dezembro
```

### Subcategory blocks (column B)

| Row | Name | Total row | Formula sample |
|-----|------|-----------|----------------|
| 6 | Diarista | 0 |  |
| 7 | Diarista | 12 | `F12`: `=SUM(F3:F11)` |
| 8 | Diarista | 12 | `F12`: `=SUM(F3:F11)` |
| 9 | Diarista | 12 | `F12`: `=SUM(F3:F11)` |
| 10 | Diarista | 12 | `F12`: `=SUM(F3:F11)` |
| 11 | Diarista | 12 | `F12`: `=SUM(F3:F11)` |
| 12 | Diarista | 0 |  |
| 14 | Aluguel | 19 | `F19`: `=SUM(F14:F18)` |
| 15 | Aluguel | 19 | `F19`: `=SUM(F14:F18)` |
| 16 | Aluguel | 19 | `F19`: `=SUM(F14:F18)` |
| 17 | Aluguel | 19 | `F19`: `=SUM(F14:F18)` |
| 18 | Aluguel | 19 | `F19`: `=SUM(F14:F18)` |
| 19 | Aluguel | 0 |  |
| 21 | Condomínio | 26 | `F26`: `=SUM(F21:F25)` |
| 22 | Condomínio | 26 | `F26`: `=SUM(F21:F25)` |
| 23 | Condomínio | 26 | `F26`: `=SUM(F21:F25)` |
| 24 | Condomínio | 26 | `F26`: `=SUM(F21:F25)` |
| 25 | Condomínio | 26 | `F26`: `=SUM(F21:F25)` |
| 26 | Condomínio | 0 |  |
| 28 | Prestação da casa | 33 | `F33`: `=SUM(F28:F32)` |
| 29 | Prestação da casa | 33 | `F33`: `=SUM(F28:F32)` |
| 30 | Prestação da casa | 33 | `F33`: `=SUM(F28:F32)` |
| 31 | Prestação da casa | 33 | `F33`: `=SUM(F28:F32)` |
| 32 | Prestação da casa | 33 | `F33`: `=SUM(F28:F32)` |
| 33 | Prestação da casa | 0 |  |
| 35 | Seguro da casa | 40 | `F40`: `=SUM(F35:F39)` |
| 36 | Seguro da casa | 40 | `F40`: `=SUM(F35:F39)` |
| 37 | Seguro da casa | 40 | `F40`: `=SUM(F35:F39)` |
| 38 | Seguro da casa | 40 | `F40`: `=SUM(F35:F39)` |
| 39 | Seguro da casa | 40 | `F40`: `=SUM(F35:F39)` |
| 40 | Seguro da casa | 0 |  |
| 42 | Celular Leandro | 47 | `F47`: `=SUM(F42:F46)` |
| 43 | Celular Leandro | 47 | `F47`: `=SUM(F42:F46)` |
| 44 | Celular Leandro | 47 | `F47`: `=SUM(F42:F46)` |
| 45 | Celular Leandro | 47 | `F47`: `=SUM(F42:F46)` |
| 46 | Celular Leandro | 47 | `F47`: `=SUM(F42:F46)` |
| 47 | Celular Leandro | 0 |  |
| 49 | Celular Ana | 54 | `F54`: `=SUM(F49:F53)` |
| 50 | Celular Ana | 54 | `F54`: `=SUM(F49:F53)` |
| 51 | Celular Ana | 54 | `F54`: `=SUM(F49:F53)` |
| 52 | Celular Ana | 54 | `F54`: `=SUM(F49:F53)` |
| 53 | Celular Ana | 54 | `F54`: `=SUM(F49:F53)` |
| 54 | Celular Ana | 0 |  |
| 56 | Google storage | 61 | `F61`: `=SUM(F56:F60)` |
| 57 | Google storage | 61 | `F61`: `=SUM(F56:F60)` |
| 58 | Google storage | 61 | `F61`: `=SUM(F56:F60)` |
| 59 | Google storage | 61 | `F61`: `=SUM(F56:F60)` |
| 60 | Google storage | 61 | `F61`: `=SUM(F56:F60)` |
| 61 | Google storage | 0 |  |
| 63 | Internet | 68 | `F68`: `=SUM(F63:F67)` |
| 64 | Internet | 68 | `F68`: `=SUM(F63:F67)` |
| 65 | Internet | 68 | `F68`: `=SUM(F63:F67)` |
| 66 | Internet | 68 | `F68`: `=SUM(F63:F67)` |
| 67 | Internet | 68 | `F68`: `=SUM(F63:F67)` |
| 68 | Internet | 72 | `F72`: `=SUM(F70:F71)` |
| 70 | Netflix | 72 | `F72`: `=SUM(F70:F71)` |
| 71 | Netflix | 72 | `F72`: `=SUM(F70:F71)` |
| 72 | Netflix | 76 | `F76`: `=SUM(F74:F75)` |
| 74 | Amazon | 76 | `F76`: `=SUM(F74:F75)` |
| 75 | Amazon | 76 | `F76`: `=SUM(F74:F75)` |
| 76 | Amazon | 80 | `F80`: `=SUM(F78:F79)` |
| 78 | HBO | 80 | `F80`: `=SUM(F78:F79)` |
| 79 | HBO | 80 | `F80`: `=SUM(F78:F79)` |
| 80 | HBO | 84 | `F84`: `=SUM(F82:F83)` |
| 82 | Spotify | 84 | `F84`: `=SUM(F82:F83)` |
| 83 | Spotify | 84 | `F84`: `=SUM(F82:F83)` |
| 84 | Spotify | 88 | `F88`: `=SUM(F86:F87)` |
| 86 | Globoplay | 88 | `F88`: `=SUM(F86:F87)` |
| 87 | Globoplay | 88 | `F88`: `=SUM(F86:F87)` |
| 88 | Globoplay | 92 | `F92`: `=SUM(F90:F91)` |
| 90 | Clube do Malte | 92 | `F92`: `=SUM(F90:F91)` |
| 91 | Clube do Malte | 92 | `F92`: `=SUM(F90:F91)` |
| 92 | Clube do Malte | 96 | `F96`: `=SUM(F94:F95)` |
| 94 | Gamepass | 96 | `F96`: `=SUM(F94:F95)` |
| 95 | Gamepass | 96 | `F96`: `=SUM(F94:F95)` |
| 96 | Gamepass | 101 | `F101`: `=SUM(F98:F100)` |
| 98 | Prestação do carro | 101 | `F101`: `=SUM(F98:F100)` |
| 99 | Prestação do carro | 101 | `F101`: `=SUM(F98:F100)` |
| 100 | Prestação do carro | 101 | `F101`: `=SUM(F98:F100)` |
| 101 | Prestação do carro | 105 | `F105`: `=SUM(F103:F104)` |
| 103 | Seguro do carro | 105 | `F105`: `=SUM(F103:F104)` |
| 104 | Seguro do carro | 105 | `F105`: `=SUM(F103:F104)` |
| 105 | Seguro do carro | 109 | `F109`: `=SUM(F107:F108)` |
| 107 | Estacionamento | 109 | `F109`: `=SUM(F107:F108)` |
| 108 | Estacionamento | 109 | `F109`: `=SUM(F107:F108)` |
| 109 | Estacionamento | 114 | `F114`: `=SUM(F111:F113)` |
| 111 | Seguro saúde | 114 | `F114`: `=SUM(F111:F113)` |
| 112 | Seguro saúde | 114 | `F114`: `=SUM(F111:F113)` |
| 113 | Seguro saúde | 114 | `F114`: `=SUM(F111:F113)` |
| 114 | Seguro saúde | 118 | `F118`: `=SUM(F116:F117)` |
| 116 | Plano odontológico | 118 | `F118`: `=SUM(F116:F117)` |
| 117 | Plano odontológico | 118 | `F118`: `=SUM(F116:F117)` |
| 118 | Plano odontológico | 122 | `F122`: `=SUM(F120:F121)` |
| 120 | Pilates | 122 | `F122`: `=SUM(F120:F121)` |
| 121 | Pilates | 122 | `F122`: `=SUM(F120:F121)` |
| 122 | Pilates | 126 | `F126`: `=SUM(F124:F125)` |
| 124 | Laya | 126 | `F126`: `=SUM(F124:F125)` |
| 125 | Laya | 126 | `F126`: `=SUM(F124:F125)` |
| 126 | Laya | 0 |  |
| 128 | Santa Cannabis | 133 | `F133`: `=SUM(F128:F132)` |
| 129 | Santa Cannabis | 133 | `F133`: `=SUM(F128:F132)` |
| 130 | Santa Cannabis | 133 | `F133`: `=SUM(F128:F132)` |
| 131 | Santa Cannabis | 133 | `F133`: `=SUM(F128:F132)` |
| 132 | Santa Cannabis | 133 | `F133`: `=SUM(F128:F132)` |
| 133 | Santa Cannabis | 137 | `F137`: `=SUM(F135:F136)` |
| 135 | Plano de saúde | 137 | `F137`: `=SUM(F135:F136)` |
| 136 | Plano de saúde | 137 | `F137`: `=SUM(F135:F136)` |
| 137 | Plano de saúde | 0 |  |
| 141 | Faculdade | 146 | `F146`: `=SUM(F141:F145)` |
| 142 | Faculdade | 146 | `F146`: `=SUM(F141:F145)` |
| 143 | Faculdade | 146 | `F146`: `=SUM(F141:F145)` |
| 144 | Faculdade | 146 | `F146`: `=SUM(F141:F145)` |
| 145 | Faculdade | 146 | `F146`: `=SUM(F141:F145)` |
| 146 | Faculdade | 0 |  |
| 148 | Bateria | 153 | `F153`: `=SUM(F148:F152)` |
| 149 | Bateria | 153 | `F153`: `=SUM(F148:F152)` |
| 150 | Bateria | 153 | `F153`: `=SUM(F148:F152)` |
| 151 | Bateria | 153 | `F153`: `=SUM(F148:F152)` |
| 152 | Bateria | 153 | `F153`: `=SUM(F148:F152)` |
| 153 | Bateria | 0 |  |
| 155 | Orion | 160 | `F160`: `=SUM(F155:F159)` |
| 156 | Orion | 160 | `F160`: `=SUM(F155:F159)` |
| 157 | Orion | 160 | `F160`: `=SUM(F155:F159)` |
| 158 | Orion | 160 | `F160`: `=SUM(F155:F159)` |
| 159 | Orion | 160 | `F160`: `=SUM(F155:F159)` |
| 160 | Orion | 0 |  |
| 162 | Lilly | 167 | `F167`: `=SUM(F162:F166)` |
| 163 | Lilly | 167 | `F167`: `=SUM(F162:F166)` |
| 164 | Lilly | 167 | `F167`: `=SUM(F162:F166)` |
| 165 | Lilly | 167 | `F167`: `=SUM(F162:F166)` |
| 166 | Lilly | 167 | `F167`: `=SUM(F162:F166)` |
| 167 | Lilly | 0 |  |
| 169 | Ambos | 174 | `F174`: `=SUM(F169:F173)` |
| 170 | Ambos | 174 | `F174`: `=SUM(F169:F173)` |
| 171 | Ambos | 174 | `F174`: `=SUM(F169:F173)` |
| 172 | Ambos | 174 | `F174`: `=SUM(F169:F173)` |
| 173 | Ambos | 174 | `F174`: `=SUM(F169:F173)` |
| 174 | Ambos | 0 |  |
| 176 | IPTU | 181 | `F181`: `=SUM(F176:F180)` |
| 177 | IPTU | 181 | `F181`: `=SUM(F176:F180)` |
| 178 | IPTU | 181 | `F181`: `=SUM(F176:F180)` |
| 179 | IPTU | 181 | `F181`: `=SUM(F176:F180)` |
| 180 | IPTU | 181 | `F181`: `=SUM(F176:F180)` |
| 181 | IPTU | 0 |  |
| 183 | Licenciamento | 188 | `F188`: `=SUM(F183:F187)` |
| 184 | Licenciamento | 188 | `F188`: `=SUM(F183:F187)` |
| 185 | Licenciamento | 188 | `F188`: `=SUM(F183:F187)` |
| 186 | Licenciamento | 188 | `F188`: `=SUM(F183:F187)` |
| 187 | Licenciamento | 188 | `F188`: `=SUM(F183:F187)` |
| 188 | Licenciamento | 0 |  |
| 190 | INSS | 195 | `F195`: `=SUM(F190:F194)` |
| 191 | INSS | 195 | `F195`: `=SUM(F190:F194)` |
| 192 | INSS | 195 | `F195`: `=SUM(F190:F194)` |
| 193 | INSS | 195 | `F195`: `=SUM(F190:F194)` |
| 194 | INSS | 195 | `F195`: `=SUM(F190:F194)` |
| 195 | INSS | 0 |  |
| 197 | IRFF | 202 | `F202`: `=SUM(F197:F201)` |
| 198 | IRFF | 202 | `F202`: `=SUM(F197:F201)` |
| 199 | IRFF | 202 | `F202`: `=SUM(F197:F201)` |
| 200 | IRFF | 202 | `F202`: `=SUM(F197:F201)` |
| 201 | IRFF | 202 | `F202`: `=SUM(F197:F201)` |
| 202 | IRFF | 0 |  |
| 204 | IPVA | 209 | `F209`: `=SUM(F204:F208)` |
| 205 | IPVA | 209 | `F209`: `=SUM(F204:F208)` |
| 206 | IPVA | 209 | `F209`: `=SUM(F204:F208)` |
| 207 | IPVA | 209 | `F209`: `=SUM(F204:F208)` |
| 208 | IPVA | 209 | `F209`: `=SUM(F204:F208)` |
| 209 | IPVA | 0 |  |
| 211 | alguma coisa sindicato | 216 | `F216`: `=SUM(F211:F215)` |
| 212 | alguma coisa sindicato | 216 | `F216`: `=SUM(F211:F215)` |
| 213 | alguma coisa sindicato | 216 | `F216`: `=SUM(F211:F215)` |
| 214 | alguma coisa sindicato | 216 | `F216`: `=SUM(F211:F215)` |
| 215 | alguma coisa sindicato | 216 | `F216`: `=SUM(F211:F215)` |
| 216 | alguma coisa sindicato | 0 |  |
| 218 | Contribuição sindicato | 223 | `F223`: `=SUM(F218:F222)` |
| 219 | Contribuição sindicato | 223 | `F223`: `=SUM(F218:F222)` |
| 220 | Contribuição sindicato | 223 | `F223`: `=SUM(F218:F222)` |
| 221 | Contribuição sindicato | 223 | `F223`: `=SUM(F218:F222)` |
| 222 | Contribuição sindicato | 223 | `F223`: `=SUM(F218:F222)` |
| 223 | Contribuição sindicato | 0 |  |
| 225 | Apoia-se 4i20 | 230 | `F230`: `=SUM(F225:F229)` |
| 226 | Apoia-se 4i20 | 230 | `F230`: `=SUM(F225:F229)` |
| 227 | Apoia-se 4i20 | 230 | `F230`: `=SUM(F225:F229)` |
| 228 | Apoia-se 4i20 | 230 | `F230`: `=SUM(F225:F229)` |
| 229 | Apoia-se 4i20 | 230 | `F230`: `=SUM(F225:F229)` |
| 230 | Apoia-se 4i20 | 0 |  |
| 232 | Seguro de vida | 237 | `F237`: `=SUM(F232:F236)` |
| 233 | Seguro de vida | 237 | `F237`: `=SUM(F232:F236)` |
| 234 | Seguro de vida | 237 | `F237`: `=SUM(F232:F236)` |
| 235 | Seguro de vida | 237 | `F237`: `=SUM(F232:F236)` |
| 236 | Seguro de vida | 237 | `F237`: `=SUM(F232:F236)` |
| 237 | Seguro de vida | 0 |  |

## Sheet: `Variáveis`

Rows: 491

### Header rows (first 5)

| Row | A | B | C | D | E | F | G | H |
|-----|---|---|---|---|---|---|---|---|
| 1 | Mês | Mês |  | Janeiro |  |  | Fevereiro |  |
| 2 |  |  |  | Item | Data | Valor | Item | Data |
| 3 | Alimentação / Limpeza | Gás | Gás |  |  |  | Gás | 3/2 |
| 4 | Alimentação / Limpeza | Gás | Gás |  |  |  |  |  |
| 5 | Alimentação / Limpeza | Gás | Gás |  |  |  |  |  |

### Month column detection

Month header at row 1 (12 month names found):

```
  col A    = Mês
  col B    = Mês
  col D    = Janeiro
  col G    = Fevereiro
  col J    = Março
  col M    = Abril
  col P    = Maio
  col S    = Junho
  col V    = Julho
  col Y    = Agosto
  col AB   = Setembro
  col AE   = Outubro
  col AH   = Novembro
  col AK   = Dezembro
```

### Subcategory blocks (column B)

| Row | Name | Total row | Formula sample |
|-----|------|-----------|----------------|
| 6 | Gás | 8 | `F8`: `=SUM(F3:F7)` |
| 7 | Gás | 8 | `F8`: `=SUM(F3:F7)` |
| 8 | Gás | 0 |  |
| 9 | Supermercado VA | 0 |  |
| 10 | Supermercado VA | 0 |  |
| 11 | Supermercado VA | 16 | `F16`: `=SUM(F9:F15)` |
| 12 | Supermercado VA | 16 | `F16`: `=SUM(F9:F15)` |
| 13 | Supermercado VA | 16 | `F16`: `=SUM(F9:F15)` |
| 14 | Supermercado VA | 16 | `F16`: `=SUM(F9:F15)` |
| 15 | Supermercado VA | 16 | `F16`: `=SUM(F9:F15)` |
| 16 | Supermercado VA | 0 |  |
| 17 | Supermercado | 0 |  |
| 18 | Supermercado | 0 |  |
| 19 | Supermercado | 0 |  |
| 20 | Supermercado | 0 |  |
| 21 | Supermercado | 0 |  |
| 22 | Supermercado | 0 |  |
| 23 | Supermercado | 0 |  |
| 24 | Supermercado | 0 |  |
| 25 | Supermercado | 0 |  |
| 26 | Supermercado | 0 |  |
| 27 | Supermercado | 0 |  |
| 28 | Supermercado | 33 | `F33`: `=SUM(F17:F32)` |
| 29 | Supermercado | 33 | `F33`: `=SUM(F17:F32)` |
| 30 | Supermercado | 33 | `F33`: `=SUM(F17:F32)` |
| 31 | Supermercado | 33 | `F33`: `=SUM(F17:F32)` |
| 32 | Supermercado | 33 | `F33`: `=SUM(F17:F32)` |
| 33 | Supermercado | 0 |  |
| 34 | Feira | 0 |  |
| 35 | Feira | 40 | `F40`: `=SUM(F34:F39)` |
| 36 | Feira | 40 | `F40`: `=SUM(F34:F39)` |
| 37 | Feira | 40 | `F40`: `=SUM(F34:F39)` |
| 38 | Feira | 40 | `F40`: `=SUM(F34:F39)` |
| 39 | Feira | 40 | `F40`: `=SUM(F34:F39)` |
| 40 | Feira | 44 | `F44`: `=SUM(F41:F43)` |
| 41 | Açougue | 44 | `F44`: `=SUM(F41:F43)` |
| 42 | Açougue | 44 | `F44`: `=SUM(F41:F43)` |
| 43 | Açougue | 44 | `F44`: `=SUM(F41:F43)` |
| 44 | Açougue | 0 |  |
| 45 | Padaria | 0 |  |
| 46 | Padaria | 0 |  |
| 47 | Padaria | 0 |  |
| 48 | Padaria | 0 |  |
| 49 | Padaria | 0 |  |
| 50 | Padaria | 0 |  |
| 51 | Padaria | 0 |  |
| 52 | Padaria | 0 |  |
| 53 | Padaria | 0 |  |
| 54 | Padaria | 0 |  |
| 55 | Padaria | 0 |  |
| 56 | Padaria | 61 | `F61`: `=SUM(F45:F60)` |
| 57 | Padaria | 61 | `F61`: `=SUM(F45:F60)` |
| 58 | Padaria | 61 | `F61`: `=SUM(F45:F60)` |
| 59 | Padaria | 61 | `F61`: `=SUM(F45:F60)` |
| 60 | Padaria | 61 | `F61`: `=SUM(F45:F60)` |
| 61 | Padaria | 67 | `F67`: `=SUM(F62:F66)` |
| 65 | Orion | 67 | `F67`: `=SUM(F62:F66)` |
| 66 | Orion | 67 | `F67`: `=SUM(F62:F66)` |
| 67 | Orion | 70 | `F70`: `=SUM(F68:F69)` |
| 68 | Orion | 70 | `F70`: `=SUM(F68:F69)` |
| 69 | Orion | 70 | `F70`: `=SUM(F68:F69)` |
| 70 | Orion | 73 | `F73`: `=SUM(F71:F72)` |
| 71 | Lilly | 73 | `F73`: `=SUM(F71:F72)` |
| 72 | Lilly | 73 | `F73`: `=SUM(F71:F72)` |
| 73 | Lilly | 76 | `F76`: `=SUM(F74:F75)` |
| 74 | Lilly | 76 | `F76`: `=SUM(F74:F75)` |
| 75 | Lilly | 76 | `F76`: `=SUM(F74:F75)` |
| 76 | Lilly | 79 | `F79`: `=SUM(F77:F78)` |
| 77 | Ambos | 79 | `F79`: `=SUM(F77:F78)` |
| 78 | Ambos | 79 | `F79`: `=SUM(F77:F78)` |
| 79 | Ambos | 84 | `F84`: `=SUM(F80:F83)` |
| 80 | Ambos | 84 | `F84`: `=SUM(F80:F83)` |
| 81 | Ambos | 84 | `F84`: `=SUM(F80:F83)` |
| 82 | Ambos | 84 | `F84`: `=SUM(F80:F83)` |
| 83 | Ambos | 84 | `F84`: `=SUM(F80:F83)` |
| 84 | Ambos | 90 | `F90`: `=SUM(F88:F89)` |
| 88 | Metrô | 90 | `F90`: `=SUM(F88:F89)` |
| 89 | Metrô | 90 | `F90`: `=SUM(F88:F89)` |
| 90 | Metrô | 0 |  |
| 91 | Ônibus | 96 | `F96`: `=SUM(F91:F95)` |
| 92 | Ônibus | 96 | `F96`: `=SUM(F91:F95)` |
| 93 | Ônibus | 96 | `F96`: `=SUM(F91:F95)` |
| 94 | Ônibus | 96 | `F96`: `=SUM(F91:F95)` |
| 95 | Ônibus | 96 | `F96`: `=SUM(F91:F95)` |
| 96 | Ônibus | 0 |  |
| 97 | Uber/Taxi | 0 |  |
| 98 | Uber/Taxi | 103 | `F103`: `=SUM(F97:F102)` |
| 99 | Uber/Taxi | 103 | `F103`: `=SUM(F97:F102)` |
| 100 | Uber/Taxi | 103 | `F103`: `=SUM(F97:F102)` |
| 101 | Uber/Taxi | 103 | `F103`: `=SUM(F97:F102)` |
| 102 | Uber/Taxi | 103 | `F103`: `=SUM(F97:F102)` |
| 103 | Uber/Taxi | 108 | `F108`: `=SUM(F104:F107)` |
| 109 | Combustível | 0 |  |
| 110 | Combustível | 115 | `F115`: `=SUM(F109:F114)` |
| 111 | Combustível | 115 | `F115`: `=SUM(F109:F114)` |
| 112 | Combustível | 115 | `F115`: `=SUM(F109:F114)` |
| 113 | Combustível | 115 | `F115`: `=SUM(F109:F114)` |
| 114 | Combustível | 115 | `F115`: `=SUM(F109:F114)` |
| 115 | Combustível | 0 |  |
| 116 | Pedágio | 0 |  |
| 117 | Pedágio | 0 |  |
| 118 | Pedágio | 0 |  |
| 119 | Pedágio | 0 |  |
| 120 | Pedágio | 0 |  |
| 121 | Pedágio | 0 |  |
| 122 | Pedágio | 0 |  |
| 123 | Pedágio | 0 |  |
| 124 | Pedágio | 0 |  |
| 125 | Pedágio | 0 |  |
| 126 | Pedágio | 0 |  |
| 127 | Pedágio | 0 |  |
| 128 | Pedágio | 0 |  |
| 129 | Pedágio | 134 | `F134`: `=SUM(F116:F133)` |
| 130 | Pedágio | 134 | `F134`: `=SUM(F116:F133)` |
| 131 | Pedágio | 134 | `F134`: `=SUM(F116:F133)` |
| 132 | Pedágio | 134 | `F134`: `=SUM(F116:F133)` |
| 133 | Pedágio | 134 | `F134`: `=SUM(F116:F133)` |
| 134 | Pedágio | 0 |  |
| 135 | Estacionamento | 0 |  |
| 136 | Estacionamento | 0 |  |
| 137 | Estacionamento | 0 |  |
| 138 | Estacionamento | 0 |  |
| 139 | Estacionamento | 0 |  |
| 140 | Estacionamento | 0 |  |
| 141 | Estacionamento | 0 |  |
| 142 | Estacionamento | 0 |  |
| 143 | Estacionamento | 148 | `F148`: `=SUM(F135:F147)` |
| 144 | Estacionamento | 148 | `F148`: `=SUM(F135:F147)` |
| 145 | Estacionamento | 148 | `F148`: `=SUM(F135:F147)` |
| 146 | Estacionamento | 148 | `F148`: `=SUM(F135:F147)` |
| 147 | Estacionamento | 148 | `F148`: `=SUM(F135:F147)` |
| 148 | Estacionamento | 157 | `F157`: `=SUM(F153:F156)` |
| 153 | Exames | 157 | `F157`: `=SUM(F153:F156)` |
| 154 | Exames | 157 | `F157`: `=SUM(F153:F156)` |
| 155 | Exames | 157 | `F157`: `=SUM(F153:F156)` |
| 156 | Exames | 157 | `F157`: `=SUM(F153:F156)` |
| 157 | Exames | 162 | `F162`: `=SUM(F158:F161)` |
| 158 | Consultas | 162 | `F162`: `=SUM(F158:F161)` |
| 159 | Consultas | 162 | `F162`: `=SUM(F158:F161)` |
| 160 | Consultas | 162 | `F162`: `=SUM(F158:F161)` |
| 161 | Consultas | 162 | `F162`: `=SUM(F158:F161)` |
| 162 | Consultas | 167 | `F167`: `=SUM(F163:F166)` |
| 163 | Óleo/flor cannabis | 167 | `F167`: `=SUM(F163:F166)` |
| 164 | Óleo/flor cannabis | 167 | `F167`: `=SUM(F163:F166)` |
| 165 | Óleo/flor cannabis | 167 | `F167`: `=SUM(F163:F166)` |
| 166 | Óleo/flor cannabis | 167 | `F167`: `=SUM(F163:F166)` |
| 167 | Óleo/flor cannabis | 172 | `F172`: `=SUM(F168:F171)` |
| 168 | Dentista | 172 | `F172`: `=SUM(F168:F171)` |
| 169 | Dentista | 172 | `F172`: `=SUM(F168:F171)` |
| 170 | Dentista | 172 | `F172`: `=SUM(F168:F171)` |
| 171 | Dentista | 172 | `F172`: `=SUM(F168:F171)` |
| 172 | Dentista | 0 |  |
| 173 | Farmácia | 0 |  |
| 174 | Farmácia | 0 |  |
| 175 | Farmácia | 0 |  |
| 176 | Farmácia | 0 |  |
| 177 | Farmácia | 182 | `F182`: `=SUM(F173:F181)` |
| 178 | Farmácia | 182 | `F182`: `=SUM(F173:F181)` |
| 179 | Farmácia | 182 | `F182`: `=SUM(F173:F181)` |
| 180 | Farmácia | 182 | `F182`: `=SUM(F173:F181)` |
| 181 | Farmácia | 182 | `F182`: `=SUM(F173:F181)` |
| 182 | Farmácia | 187 | `F187`: `=SUM(F185:F186)` |
| 185 | Terpenos | 187 | `F187`: `=SUM(F185:F186)` |
| 186 | Terpenos | 187 | `F187`: `=SUM(F185:F186)` |
| 187 | Terpenos | 0 |  |
| 188 | Ingredientes | 0 |  |
| 189 | Ingredientes | 0 |  |
| 190 | Ingredientes | 0 |  |
| 191 | Ingredientes | 0 |  |
| 192 | Ingredientes | 0 |  |
| 193 | Ingredientes | 198 | `F198`: `=SUM(F188:F197)` |
| 194 | Ingredientes | 198 | `F198`: `=SUM(F188:F197)` |
| 195 | Ingredientes | 198 | `F198`: `=SUM(F188:F197)` |
| 196 | Ingredientes | 198 | `F198`: `=SUM(F188:F197)` |
| 197 | Ingredientes | 198 | `F198`: `=SUM(F188:F197)` |
| 198 | Ingredientes | 203 | `F203`: `=SUM(F199:F202)` |
| 199 | Ingrediente chocolate | 203 | `F203`: `=SUM(F199:F202)` |
| 200 | Ingrediente chocolate | 203 | `F203`: `=SUM(F199:F202)` |
| 201 | Ingrediente chocolate | 203 | `F203`: `=SUM(F199:F202)` |
| 202 | Ingrediente chocolate | 203 | `F203`: `=SUM(F199:F202)` |
| 203 | Ingrediente chocolate | 0 |  |
| 204 | Empréstimo | 0 |  |
| 205 | Empréstimo | 0 |  |
| 206 | Empréstimo | 0 |  |
| 207 | Empréstimo | 0 |  |
| 208 | Empréstimo | 0 |  |
| 209 | Empréstimo | 0 |  |
| 210 | Empréstimo | 0 |  |
| 211 | Empréstimo | 0 |  |
| 212 | Empréstimo | 0 |  |
| 213 | Empréstimo | 0 |  |
| 214 | Empréstimo | 0 |  |
| 215 | Empréstimo | 0 |  |
| 216 | Empréstimo | 221 | `F221`: `=SUM(F204:F220)` |
| 217 | Empréstimo | 221 | `F221`: `=SUM(F204:F220)` |
| 218 | Empréstimo | 221 | `F221`: `=SUM(F204:F220)` |
| 219 | Empréstimo | 221 | `F221`: `=SUM(F204:F220)` |
| 220 | Empréstimo | 221 | `F221`: `=SUM(F204:F220)` |
| 221 | Empréstimo | 229 | `F229`: `=SUM(F225:F228)` |
| 225 | Cabelereiro | 229 | `F229`: `=SUM(F225:F228)` |
| 226 | Cabelereiro | 229 | `F229`: `=SUM(F225:F228)` |
| 227 | Cabelereiro | 229 | `F229`: `=SUM(F225:F228)` |
| 228 | Cabelereiro | 229 | `F229`: `=SUM(F225:F228)` |
| 229 | Cabelereiro | 234 | `F234`: `=SUM(F230:F233)` |
| 230 | Produtos | 234 | `F234`: `=SUM(F230:F233)` |
| 231 | Produtos | 234 | `F234`: `=SUM(F230:F233)` |
| 232 | Produtos | 234 | `F234`: `=SUM(F230:F233)` |
| 233 | Produtos | 234 | `F234`: `=SUM(F230:F233)` |
| 234 | Produtos | 239 | `F239`: `=SUM(F235:F238)` |
| 243 | Produtos | 248 | `F248`: `=SUM(F243:F247)` |
| 244 | Produtos | 248 | `F248`: `=SUM(F243:F247)` |
| 245 | Produtos | 248 | `F248`: `=SUM(F243:F247)` |
| 246 | Produtos | 248 | `F248`: `=SUM(F243:F247)` |
| 247 | Produtos | 248 | `F248`: `=SUM(F243:F247)` |
| 248 | Produtos | 0 |  |
| 451 | Luz | 456 | `F456`: `=SUM(F451:F455)` |
| 456 | Luz | 0 |  |
| 458 | Água | 463 | `F463`: `=SUM(F458:F462)` |
| 463 | Água | 0 |  |
| 465 | IOF | 470 | `F470`: `=SUM(F465:F469)` |
| 470 | IOF | 0 |  |
| 472 | Acupuntura / Massagem | 477 | `F477`: `=SUM(F472:F476)` |
| 477 | Acupuntura / Massagem | 0 |  |
| 479 | Manicure | 484 | `F484`: `=SUM(F479:F483)` |
| 484 | Manicure | 0 |  |
| 486 | Academia | 491 | `F491`: `=SUM(F486:F490)` |
| 491 | Academia | 0 |  |

### Formula samples (from first TOTAL row, row 8)

```
  F8 = SUM(F3:F7)
  I8 = SUM(I3:I7)
  L8 = SUM(L3:L7)
  O8 = SUM(O3:O7)
  R8 = SUM(R3:R7)
  U8 = SUM(U3:U7)
  X8 = SUM(X3:X7)
  AA8 = SUM(AA3:AA7)
  AD8 = SUM(AD3:AD7)
  AG8 = SUM(AG3:AG7)
  AJ8 = SUM(AJ3:AJ7)
  AM8 = SUM(AM3:AM7)
```

## Sheet: `Extras`

Rows: 325

### Header rows (first 5)

| Row | A | B | C | D | E | F | G | H |
|-----|---|---|---|---|---|---|---|---|
| 1 | Mês | Mês |  | Janeiro |  |  | Fevereiro |  |
| 2 |  |  |  | Item | Data | Valor | Item | Data |
| 3 | Saúde | Médico |  |  |  |  |  |  |
| 4 | Saúde | Médico |  |  |  |  |  |  |
| 5 | Saúde | Médico |  |  |  |  |  |  |

### Month column detection

Month header at row 1 (12 month names found):

```
  col A    = Mês
  col B    = Mês
  col D    = Janeiro
  col G    = Fevereiro
  col J    = Março
  col M    = Abril
  col P    = Maio
  col S    = Junho
  col V    = Julho
  col Y    = Agosto
  col AB   = Setembro
  col AE   = Outubro
  col AH   = Novembro
  col AK   = Dezembro
```

### Subcategory blocks (column B)

| Row | Name | Total row | Formula sample |
|-----|------|-----------|----------------|
| 6 | Médico | 8 | `F8`: `=SUM(F7)` |
| 7 | Médico | 8 | `F8`: `=SUM(F7)` |
| 8 | Médico | 0 |  |
| 9 | Dentista | 0 |  |
| 10 | Dentista | 0 |  |
| 11 | Dentista | 0 |  |
| 12 | Dentista | 0 |  |
| 13 | Dentista | 0 |  |
| 14 | Dentista | 0 |  |
| 15 | Dentista | 0 |  |
| 16 | Dentista | 21 | `F21`: `=SUM(F20)` |
| 17 | Dentista | 21 | `F21`: `=SUM(F20)` |
| 18 | Dentista | 21 | `F21`: `=SUM(F20)` |
| 19 | Dentista | 21 | `F21`: `=SUM(F20)` |
| 20 | Dentista | 21 | `F21`: `=SUM(F20)` |
| 21 | Dentista | 25 | `F25`: `=SUM(F24)` |
| 22 | Hospital | 25 | `F25`: `=SUM(F24)` |
| 23 | Hospital | 25 | `F25`: `=SUM(F24)` |
| 24 | Hospital | 25 | `F25`: `=SUM(F24)` |
| 25 | Hospital | 37 | `F37`: `=SUM(F26:F36)` |
| 41 | Carro | 0 |  |
| 42 | Carro | 47 | `F47`: `=SUM(F46)` |
| 43 | Carro | 47 | `F47`: `=SUM(F46)` |
| 44 | Carro | 47 | `F47`: `=SUM(F46)` |
| 45 | Carro | 47 | `F47`: `=SUM(F46)` |
| 46 | Carro | 47 | `F47`: `=SUM(F46)` |
| 47 | Carro | 0 |  |
| 48 | Casa | 53 | `F53`: `=SUM(F52)` |
| 49 | Casa | 53 | `F53`: `=SUM(F52)` |
| 50 | Casa | 53 | `F53`: `=SUM(F52)` |
| 51 | Casa | 53 | `F53`: `=SUM(F52)` |
| 52 | Casa | 53 | `F53`: `=SUM(F52)` |
| 53 | Casa | 0 |  |
| 54 | Mudança | 0 |  |
| 55 | Mudança | 0 |  |
| 56 | Mudança | 0 |  |
| 57 | Mudança | 0 |  |
| 58 | Mudança | 0 |  |
| 59 | Mudança | 0 |  |
| 60 | Mudança | 65 | `F65`: `=SUM(F64)` |
| 61 | Mudança | 65 | `F65`: `=SUM(F64)` |
| 62 | Mudança | 65 | `F65`: `=SUM(F64)` |
| 63 | Mudança | 65 | `F65`: `=SUM(F64)` |
| 64 | Mudança | 65 | `F65`: `=SUM(F64)` |
| 65 | Mudança | 0 |  |
| 69 | Material escolar | 0 |  |
| 70 | Material escolar | 75 | `F75`: `=SUM(F74)` |
| 71 | Material escolar | 75 | `F75`: `=SUM(F74)` |
| 72 | Material escolar | 75 | `F75`: `=SUM(F74)` |
| 73 | Material escolar | 75 | `F75`: `=SUM(F74)` |
| 74 | Material escolar | 75 | `F75`: `=SUM(F74)` |
| 75 | Material escolar | 81 | `F81`: `=SUM(F76:F80)` |
| 85 | Orion | 0 |  |
| 86 | Orion | 91 | `F91`: `=SUM(F90)` |
| 87 | Orion | 91 | `F91`: `=SUM(F90)` |
| 88 | Orion | 91 | `F91`: `=SUM(F90)` |
| 89 | Orion | 91 | `F91`: `=SUM(F90)` |
| 90 | Orion | 91 | `F91`: `=SUM(F90)` |
| 91 | Orion | 0 |  |
| 92 | Lilly | 97 | `F97`: `=SUM(F96)` |
| 93 | Lilly | 97 | `F97`: `=SUM(F96)` |
| 94 | Lilly | 97 | `F97`: `=SUM(F96)` |
| 95 | Lilly | 97 | `F97`: `=SUM(F96)` |
| 96 | Lilly | 97 | `F97`: `=SUM(F96)` |
| 97 | Lilly | 0 |  |
| 98 | Ambos | 103 | `F103`: `=SUM(F102)` |
| 99 | Ambos | 103 | `F103`: `=SUM(F102)` |
| 100 | Ambos | 103 | `F103`: `=SUM(F102)` |
| 101 | Ambos | 103 | `F103`: `=SUM(F102)` |
| 102 | Ambos | 103 | `F103`: `=SUM(F102)` |
| 103 | Ambos | 0 |  |
| 306 | HC grow | 311 | `F311`: `=SUM(F306:F310)` |
| 311 | HC grow | 0 |  |
| 313 | Rematrícula faculdade | 318 | `F318`: `=SUM(F313:F317)` |
| 318 | Rematrícula faculdade | 0 |  |
| 320 | Uniforme | 325 | `F325`: `=SUM(F320:F324)` |
| 325 | Uniforme | 0 |  |

### Formula samples (from first TOTAL row, row 8)

```
  F8 = SUM(F7)
  I8 = SUM(I7)
  L8 = SUM(L3:L7)
  O8 = SUM(O3:O7)
  R8 = SUM(R3:R7)
  U8 = SUM(U3:U7)
  X8 = SUM(X3:X7)
  AA8 = SUM(AA3:AA7)
  AD8 = SUM(AD3:AD7)
  AG8 = SUM(AG7)
  AJ8 = SUM(AJ7)
  AM8 = SUM(AM7)
```

## Sheet: `Adicionais`

Rows: 417

### Header rows (first 5)

| Row | A | B | C | D | E | F | G | H |
|-----|---|---|---|---|---|---|---|---|
| 1 |  |  |  | Janeiro |  |  | Fevereiro |  |
| 2 |  |  |  | Item | Data | Valor | Item | Data |
| 3 | Lazer | Viagens |  | Passagem Machado | 2/1 | R$ 47.95 |  |  |
| 4 | Lazer | Viagens |  | Ana Machado | 7/1 | 263,00/2 |  |  |
| 5 | Lazer | Viagens |  | Motoboy | 7/1 | R$ 7.00 |  |  |

### Month column detection

Month header at row 1 (12 month names found):

```
  col D    = Janeiro
  col G    = Fevereiro
  col J    = Março
  col M    = Abril
  col P    = Maio
  col S    = Junho
  col V    = Julho
  col Y    = Agosto
  col AB   = Setembro
  col AE   = Outubro
  col AH   = Novembro
  col AK   = Dezembro
```

### Subcategory blocks (column B)

| Row | Name | Total row | Formula sample |
|-----|------|-----------|----------------|
| 6 | Viagens | 0 |  |
| 7 | Viagens | 0 |  |
| 8 | Viagens | 0 |  |
| 9 | Viagens | 0 |  |
| 10 | Viagens | 0 |  |
| 11 | Viagens | 0 |  |
| 12 | Viagens | 17 | `F17`: `=SUM(F16)` |
| 13 | Viagens | 17 | `F17`: `=SUM(F16)` |
| 14 | Viagens | 17 | `F17`: `=SUM(F16)` |
| 15 | Viagens | 17 | `F17`: `=SUM(F16)` |
| 16 | Viagens | 17 | `F17`: `=SUM(F16)` |
| 17 | Viagens | 0 |  |
| 18 | Grow | 0 |  |
| 19 | Grow | 0 |  |
| 20 | Grow | 0 |  |
| 21 | Grow | 26 | `F26`: `=SUM(F25)` |
| 22 | Grow | 26 | `F26`: `=SUM(F25)` |
| 23 | Grow | 26 | `F26`: `=SUM(F25)` |
| 24 | Grow | 26 | `F26`: `=SUM(F25)` |
| 25 | Grow | 26 | `F26`: `=SUM(F25)` |
| 26 | Grow | 30 | `F30`: `=SUM(F29)` |
| 27 | Jogos | 30 | `F30`: `=SUM(F29)` |
| 28 | Jogos | 30 | `F30`: `=SUM(F29)` |
| 29 | Jogos | 30 | `F30`: `=SUM(F29)` |
| 30 | Jogos | 0 |  |
| 31 | Cigarro | 0 |  |
| 32 | Cigarro | 0 |  |
| 33 | Cigarro | 38 | `F38`: `=SUM(F37)` |
| 34 | Cigarro | 38 | `F38`: `=SUM(F37)` |
| 35 | Cigarro | 38 | `F38`: `=SUM(F37)` |
| 36 | Cigarro | 38 | `F38`: `=SUM(F37)` |
| 37 | Cigarro | 38 | `F38`: `=SUM(F37)` |
| 38 | Cigarro | 0 |  |
| 39 | Café | 44 | `F44`: `=SUM(F43)` |
| 40 | Café | 44 | `F44`: `=SUM(F43)` |
| 41 | Café | 44 | `F44`: `=SUM(F43)` |
| 42 | Café | 44 | `F44`: `=SUM(F43)` |
| 43 | Café | 44 | `F44`: `=SUM(F43)` |
| 44 | Café | 0 |  |
| 45 | Cerveja | 0 |  |
| 46 | Cerveja | 0 |  |
| 47 | Cerveja | 0 |  |
| 48 | Cerveja | 53 | `F53`: `=SUM(F52)` |
| 49 | Cerveja | 53 | `F53`: `=SUM(F52)` |
| 50 | Cerveja | 53 | `F53`: `=SUM(F52)` |
| 51 | Cerveja | 53 | `F53`: `=SUM(F52)` |
| 52 | Cerveja | 53 | `F53`: `=SUM(F52)` |
| 53 | Cerveja | 57 | `F57`: `=SUM(F56)` |
| 54 | Discos | 57 | `F57`: `=SUM(F56)` |
| 55 | Discos | 57 | `F57`: `=SUM(F56)` |
| 56 | Discos | 57 | `F57`: `=SUM(F56)` |
| 57 | Discos | 61 | `F61`: `=SUM(F60)` |
| 58 | Livros | 61 | `F61`: `=SUM(F60)` |
| 59 | Livros | 61 | `F61`: `=SUM(F60)` |
| 60 | Livros | 61 | `F61`: `=SUM(F60)` |
| 61 | Livros | 0 |  |
| 62 | Cinema/teatro | 0 |  |
| 63 | Cinema/teatro | 68 | `F68`: `=SUM(F67)` |
| 64 | Cinema/teatro | 68 | `F68`: `=SUM(F67)` |
| 65 | Cinema/teatro | 68 | `F68`: `=SUM(F67)` |
| 66 | Cinema/teatro | 68 | `F68`: `=SUM(F67)` |
| 67 | Cinema/teatro | 68 | `F68`: `=SUM(F67)` |
| 68 | Cinema/teatro | 0 |  |
| 69 | Restaurantes/bares | 0 |  |
| 70 | Restaurantes/bares | 0 |  |
| 71 | Restaurantes/bares | 0 |  |
| 72 | Restaurantes/bares | 0 |  |
| 73 | Restaurantes/bares | 0 |  |
| 74 | Restaurantes/bares | 0 |  |
| 75 | Restaurantes/bares | 0 |  |
| 76 | Restaurantes/bares | 0 |  |
| 77 | Restaurantes/bares | 0 |  |
| 78 | Restaurantes/bares | 0 |  |
| 79 | Restaurantes/bares | 0 |  |
| 80 | Restaurantes/bares | 0 |  |
| 81 | Restaurantes/bares | 0 |  |
| 82 | Restaurantes/bares | 87 | `F87`: `=SUM(F86)` |
| 83 | Restaurantes/bares | 87 | `F87`: `=SUM(F86)` |
| 84 | Restaurantes/bares | 87 | `F87`: `=SUM(F86)` |
| 85 | Restaurantes/bares | 87 | `F87`: `=SUM(F86)` |
| 86 | Restaurantes/bares | 87 | `F87`: `=SUM(F86)` |
| 87 | Restaurantes/bares | 0 |  |
| 88 | Delivery | 0 |  |
| 89 | Delivery | 0 |  |
| 90 | Delivery | 0 |  |
| 91 | Delivery | 96 | `F96`: `=SUM(F95)` |
| 92 | Delivery | 96 | `F96`: `=SUM(F95)` |
| 93 | Delivery | 96 | `F96`: `=SUM(F95)` |
| 94 | Delivery | 96 | `F96`: `=SUM(F95)` |
| 95 | Delivery | 96 | `F96`: `=SUM(F95)` |
| 96 | Delivery | 0 |  |
| 97 | Diamba | 0 |  |
| 98 | Diamba | 0 |  |
| 99 | Diamba | 0 |  |
| 100 | Diamba | 0 |  |
| 101 | Diamba | 0 |  |
| 102 | Diamba | 0 |  |
| 103 | Diamba | 0 |  |
| 104 | Diamba | 0 |  |
| 105 | Diamba | 110 | `F110`: `=SUM(F97:F109)` |
| 106 | Diamba | 110 | `F110`: `=SUM(F97:F109)` |
| 107 | Diamba | 110 | `F110`: `=SUM(F97:F109)` |
| 108 | Diamba | 110 | `F110`: `=SUM(F97:F109)` |
| 109 | Diamba | 110 | `F110`: `=SUM(F97:F109)` |
| 110 | Diamba | 0 |  |
| 111 | Shows | 0 |  |
| 112 | Shows | 0 |  |
| 113 | Shows | 0 |  |
| 114 | Shows | 119 | `F119`: `=SUM(F118)` |
| 115 | Shows | 119 | `F119`: `=SUM(F118)` |
| 116 | Shows | 119 | `F119`: `=SUM(F118)` |
| 117 | Shows | 119 | `F119`: `=SUM(F118)` |
| 118 | Shows | 119 | `F119`: `=SUM(F118)` |
| 119 | Shows | 0 |  |
| 120 | Sx | 0 |  |
| 121 | Sx | 0 |  |
| 122 | Sx | 0 |  |
| 123 | Sx | 128 | `F128`: `=SUM(F127)` |
| 124 | Sx | 128 | `F128`: `=SUM(F127)` |
| 125 | Sx | 128 | `F128`: `=SUM(F127)` |
| 126 | Sx | 128 | `F128`: `=SUM(F127)` |
| 127 | Sx | 128 | `F128`: `=SUM(F127)` |
| 128 | Sx | 0 |  |
| 132 | Roupas | 0 |  |
| 133 | Roupas | 0 |  |
| 134 | Roupas | 139 | `F139`: `=SUM(F138)` |
| 135 | Roupas | 139 | `F139`: `=SUM(F138)` |
| 136 | Roupas | 139 | `F139`: `=SUM(F138)` |
| 137 | Roupas | 139 | `F139`: `=SUM(F138)` |
| 138 | Roupas | 139 | `F139`: `=SUM(F138)` |
| 139 | Roupas | 142 | `F142`: `=SUM(F141)` |
| 140 | Calçados | 142 | `F142`: `=SUM(F141)` |
| 141 | Calçados | 142 | `F142`: `=SUM(F141)` |
| 142 | Calçados | 145 | `F145`: `=SUM(F144)` |
| 143 | Acessórios | 145 | `F145`: `=SUM(F144)` |
| 144 | Acessórios | 145 | `F145`: `=SUM(F144)` |
| 145 | Acessórios | 153 | `F153`: `=SUM(F152)` |
| 149 | Presentes | 153 | `F153`: `=SUM(F152)` |
| 150 | Presentes | 153 | `F153`: `=SUM(F152)` |
| 151 | Presentes | 153 | `F153`: `=SUM(F152)` |
| 152 | Presentes | 153 | `F153`: `=SUM(F152)` |
| 153 | Presentes | 0 |  |
| 154 | Diversos | 0 |  |
| 155 | Diversos | 0 |  |
| 156 | Diversos | 0 |  |
| 157 | Diversos | 0 |  |
| 158 | Diversos | 0 |  |
| 159 | Diversos | 0 |  |
| 160 | Diversos | 165 | `F165`: `=SUM(F164)` |
| 161 | Diversos | 165 | `F165`: `=SUM(F164)` |
| 162 | Diversos | 165 | `F165`: `=SUM(F164)` |
| 163 | Diversos | 165 | `F165`: `=SUM(F164)` |
| 164 | Diversos | 165 | `F165`: `=SUM(F164)` |
| 165 | Diversos | 0 |  |
| 166 | Jardinagem | 171 | `F171`: `=SUM(F170)` |
| 167 | Jardinagem | 171 | `F171`: `=SUM(F170)` |
| 168 | Jardinagem | 171 | `F171`: `=SUM(F170)` |
| 169 | Jardinagem | 171 | `F171`: `=SUM(F170)` |
| 170 | Jardinagem | 171 | `F171`: `=SUM(F170)` |
| 171 | Jardinagem | 0 |  |
| 172 | Ferramentas | 0 |  |
| 173 | Ferramentas | 178 | `F178`: `=SUM(F177)` |
| 174 | Ferramentas | 178 | `F178`: `=SUM(F177)` |
| 175 | Ferramentas | 178 | `F178`: `=SUM(F177)` |
| 176 | Ferramentas | 178 | `F178`: `=SUM(F177)` |
| 177 | Ferramentas | 178 | `F178`: `=SUM(F177)` |
| 178 | Ferramentas | 0 |  |
| 179 | Escritório | 0 |  |
| 180 | Escritório | 0 |  |
| 181 | Escritório | 0 |  |
| 182 | Escritório | 0 |  |
| 183 | Escritório | 0 |  |
| 184 | Escritório | 0 |  |
| 185 | Escritório | 190 | `F190`: `=SUM(F189)` |
| 186 | Escritório | 190 | `F190`: `=SUM(F189)` |
| 187 | Escritório | 190 | `F190`: `=SUM(F189)` |
| 188 | Escritório | 190 | `F190`: `=SUM(F189)` |
| 189 | Escritório | 190 | `F190`: `=SUM(F189)` |
| 190 | Escritório | 195 | `F195`: `=SUM(F194)` |
| 191 | Papelaria | 195 | `F195`: `=SUM(F194)` |
| 192 | Papelaria | 195 | `F195`: `=SUM(F194)` |
| 193 | Papelaria | 195 | `F195`: `=SUM(F194)` |
| 194 | Papelaria | 195 | `F195`: `=SUM(F194)` |
| 195 | Papelaria | 200 | `F200`: `=SUM(F199)` |
| 196 | Cozinha | 200 | `F200`: `=SUM(F199)` |
| 197 | Cozinha | 200 | `F200`: `=SUM(F199)` |
| 198 | Cozinha | 200 | `F200`: `=SUM(F199)` |
| 199 | Cozinha | 200 | `F200`: `=SUM(F199)` |
| 200 | Cozinha | 205 | `F205`: `=SUM(F204)` |
| 201 | Telescópio | 205 | `F205`: `=SUM(F204)` |
| 202 | Telescópio | 205 | `F205`: `=SUM(F204)` |
| 203 | Telescópio | 205 | `F205`: `=SUM(F204)` |
| 204 | Telescópio | 205 | `F205`: `=SUM(F204)` |
| 205 | Telescópio | 0 |  |
| 398 | Computador ($) | 403 | `F403`: `=SUM(F398:F402)` |
| 403 | Computador ($) | 0 |  |
| 405 | Arte | 410 | `F410`: `=SUM(F405:F409)` |
| 410 | Arte | 0 |  |
| 412 | Diamba | 417 | `F417`: `=SUM(F412:F416)` |
| 417 | Diamba | 0 |  |

## Sheet: `Referência de Categorias`

Rows: 117

### Full content

| Row | A (Sheet) | B (Category) | C (Subcategory) | D (Row#) | E | F (Total row) |
|-----|-----------|--------------|-----------------|----------|---|---------------|
| 1 | REFERÊNCIA DE CATEGORIAS E SUB-CATEGORIAS |  |  |  |  |  |
| 2 | Esta planilha lista todas as categorias e sub-categorias encontradas nas planilhas de despesas |  |  |  |  |  |
| 4 | Tipo Principal | Categoria | Sub-categoria | Linha na Planilha | Lista Linha | Total Linha |
| 5 | Fixas | Habitação | Diarista | 3 | 37 | 9 |
| 6 | Fixas | Educação | Faculdade | 17 | 79 | 143 |
| 7 | Fixas | Habitação | Aluguel | 31 | 31 | 16 |
| 8 | Fixas | Habitação | Condomínio | 38 | 32 | 23 |
| 9 | Fixas | Habitação | Prestação da casa | 45 | 33 | 30 |
| 10 | Fixas | Habitação | Seguro da casa | 52 | 34 | 37 |
| 11 | Fixas | Habitação | Celular Leandro | 59 | 35 | 44 |
| 12 | Fixas | Habitação | Celular Ana | 66 | 36 | 51 |
| 13 | Fixas | Habitação | Google storage | 73 | 44 | 58 |
| 14 | Fixas | Habitação | Internet | 80 | 45 | 65 |
| 15 | Fixas | Lazer | Netflix | 87 | 50 | 69 |
| 16 | Fixas | Lazer | Amazon | 94 | 51 | 73 |
| 17 | Fixas | Lazer | HBO | 101 | 52 | 77 |
| 18 | Fixas | Lazer | Spotify | 108 | 53 | 81 |
| 19 | Fixas | Lazer | Globoplay | 115 | 54 | 85 |
| 20 | Fixas | Lazer | Clube do Malte | 122 | 55 | 89 |
| 21 | Fixas | Lazer | Gamepass | 129 | 56 | 93 |
| 22 | Fixas | Transporte | Prestação do carro | 136 | 62 | 98 |
| 23 | Fixas | Transporte | Seguro do carro | 143 | 63 | 102 |
| 24 | Fixas | Transporte | Estacionamento | 150 | 64 | 106 |
| 25 | Fixas | Saúde | Seguro saúde | 157 | 69 | 111 |
| 26 | Fixas | Saúde | Plano odontológico | 164 | 70 | 115 |
| 27 | Fixas | Saúde | Pilates | 171 | 71 | 119 |
| 28 | Fixas | Saúde | Laya | 178 | 72 | 123 |
| 29 | Fixas | Saúde | Santa Cannabis | 185 | 73 | 130 |
| 30 | Fixas | Saúde | Plano de saúde | 192 | 74 | 134 |
| 31 | Fixas | Educação | Bateria | 206 | 80 | 150 |
| 32 | Fixas | Pet | Orion | 213 | 86 | 157 |
| 33 | Fixas | Pet | Lilly | 220 | 88 | 164 |
| 34 | Fixas | Pet | Ambos | 227 | 90 | 171 |
| 35 | Fixas | Impostos | IPTU | 234 | 95 | 178 |
| 36 | Fixas | Impostos | Licenciamento | 241 | 96 | 185 |
| 37 | Fixas | Impostos | INSS | 248 | 97 | 192 |
| 38 | Fixas | Impostos | IRFF | 255 | 98 | 199 |
| 39 | Fixas | Impostos | IPVA | 262 | 99 | 206 |
| 40 | Fixas | Outros | alguma coisa sindicato | 269 | 104 | 213 |
| 41 | Fixas | Outros | Contribuição sindicato | 276 | 105 | 220 |
| 42 | Fixas | Outros | Apoia-se 4i20 | 283 | 106 | 227 |
| 43 | Fixas | Outros | Seguro de vida | 290 | 107 | 234 |
| 44 | Variáveis | Alimentação / Limpeza | Gás | 3 | 120 | 8 |
| 45 | Variáveis | Alimentação / Limpeza | Supermercado VA | 9 | 143 | 16 |
| 46 | Variáveis | Alimentação / Limpeza | Supermercado | 17 | 144 | 33 |
| 47 | Variáveis | Alimentação / Limpeza | Feira | 34 | 145 | 40 |
| 48 | Variáveis | Alimentação / Limpeza | Açougue | 41 | 146 | 44 |
| 49 | Variáveis | Alimentação / Limpeza | Padaria | 45 | 147 | 61 |
| 50 | Variáveis | Pets | Orion | 65 | 152 | 70 |
| 51 | Variáveis | Pets | Lilly | 71 | 155 | 76 |
| 52 | Variáveis | Pets | Ambos | 77 | 159 | 84 |
| 53 | Variáveis | Transporte | Metrô | 88 | 133 | 90 |
| 54 | Variáveis | Transporte | Ônibus | 91 | 134 | 96 |
| 55 | Variáveis | Transporte | Uber/Taxi | 97 | 135 | 103 |
| 56 | Variáveis | Transporte | Combustível | 109 | 136 | 115 |
| 57 | Variáveis | Transporte | Pedágio | 116 | 137 | 134 |
| 58 | Variáveis | Transporte | Estacionamento | 135 | 138 | 148 |
| 59 | Variáveis | Saúde | Exames | 153 | 166 | 157 |
| 60 | Variáveis | Saúde | Consultas | 158 | 167 | 162 |
| 61 | Variáveis | Saúde | Óleo/flor cannabis | 163 | 169 | 167 |
| 62 | Variáveis | Saúde | Dentista | 168 | 170 | 172 |
| 63 | Variáveis | Saúde | Farmácia | 173 | 171 | 182 |
| 64 | Variáveis | Anita | Terpenos | 185 | 176 | 187 |
| 65 | Variáveis | Anita | Ingredientes | 188 | 177 | 198 |
| 66 | Variáveis | Anita | Ingrediente chocolate | 199 | 178 | 203 |
| 67 | Variáveis | Anita | Empréstimo | 204 | 179 | 221 |
| 68 | Variáveis | Cuidados pessoais | Cabelereiro | 225 | 185 | 229 |
| 69 | Variáveis | Cuidados pessoais | Produtos | 230 | 187 | 248 |
| 70 | Variáveis | Cuidados pessoais | Gás | 243 | 120 | 8 |
| 71 | Variáveis | Habitação | Produtos | 249 | 187 | 248 |
| 72 | Variáveis | Habitação | Luz | 457 | 116 | 456 |
| 73 | Variáveis | Habitação | Água | 464 | 117 | 463 |
| 74 | Variáveis | Impostos | IOF | 471 | 125 | 470 |
| 75 | Variáveis | Saúde | Acupuntura / Massagem | 478 | 168 | 477 |
| 76 | Variáveis | Cuidados pessoais | Manicure | 485 | 186 | 484 |
| 77 | Variáveis | Cuidados pessoais | Academia | 492 | 188 | 491 |
| 78 | Extras | Saúde | Médico | 3 | 197 | 8 |
| 79 | Extras | Saúde | Dentista | 9 | 198 | 21 |
| 80 | Extras | Saúde | Hospital | 22 | 199 | 25 |
| 81 | Extras | Manutenção / prevenção | Carro | 41 | 204 | 47 |
| 82 | Extras | Manutenção / prevenção | Casa | 48 | 205 | 53 |
| 83 | Extras | Manutenção / prevenção | Mudança | 54 | 206 | 65 |
| 84 | Extras | Educação | Material escolar | 69 | 229 | 75 |
| 85 | Extras | Pets | Orion | 85 | 211 | 91 |
| 86 | Extras | Pets | Lilly | 92 | 213 | 97 |
| 87 | Extras | Pets | Ambos | 98 | 216 | 103 |
| 88 | Extras | Advogado | HC grow | 306 | 222 | 311 |
| 89 | Extras | Educação | Rematrícula faculdade | 313 | 230 | 318 |
| 90 | Extras | Educação | Uniforme | 320 | 231 | 325 |
| 91 | Adicionais | Lazer | Viagens | 3 | 240 | 17 |
| 92 | Adicionais | Lazer | Grow | 18 | 241 | 26 |
| 93 | Adicionais | Lazer | Jogos | 27 | 242 | 30 |
| 94 | Adicionais | Lazer | Cigarro | 31 | 244 | 38 |
| 95 | Adicionais | Lazer | Café | 39 | 245 | 44 |
| 96 | Adicionais | Lazer | Cerveja | 45 | 246 | 53 |
| 97 | Adicionais | Lazer | Discos | 54 | 250 | 57 |
| 98 | Adicionais | Lazer | Livros | 58 | 247 | 61 |
| 99 | Adicionais | Lazer | Cinema/teatro | 62 | 248 | 65 |
| 100 | Adicionais | Lazer | Restaurantes/bares | 66 | 251 | 84 |
| 101 | Adicionais | Lazer | Delivery | 85 | 252 | 93 |
| 102 | Adicionais | Lazer | Diamba | 94 | 253 | 107 |
| 103 | Adicionais | Lazer | Shows | 108 | 254 | 116 |
| 104 | Adicionais | Lazer | Sx | 117 | 255 | 125 |
| 105 | Adicionais | Vestuário | Roupas | 129 | 260 | 136 |
| 106 | Adicionais | Vestuário | Calçados | 137 | 261 | 139 |
| 107 | Adicionais | Vestuário | Acessórios | 140 | 262 | 142 |
| 108 | Adicionais | Outros | Presentes | 146 | 267 | 150 |
| 109 | Adicionais | Outros | Diversos | 151 | 268 | 162 |
| 110 | Adicionais | Outros | Jardinagem | 163 | 269 | 168 |
| 111 | Adicionais | Outros | Ferramentas | 169 | 270 | 175 |
| 112 | Adicionais | Outros | Escritório | 176 | 271 | 183 |
| 113 | Adicionais | Outros | Papelaria | 184 | 272 | 188 |
| 114 | Adicionais | Lazer | Computador ($) | 391 | 243 | 396 |
| 115 | Adicionais | Lazer | Arte | 398 | 249 | 403 |
| 116 | Adicionais | Outros | Cozinha |  |  |  |
| 117 | Adicionais | Outros | Telescópio |  |  |  |

