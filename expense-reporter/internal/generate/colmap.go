package generate

import "strings"

var monthNames = []string{
	"Janeiro", "Fevereiro", "Março", "Abril", "Maio", "Junho",
	"Julho", "Agosto", "Setembro", "Outubro", "Novembro", "Dezembro",
}

// colName converts a 0-based column index to an Excel column letter.
func colName(idx int) string {
	if idx < 0 {
		return ""
	}

	var result strings.Builder
	for idx >= 0 {
		result.WriteByte(byte('A' + (idx % 26)))
		idx = (idx / 26) - 1
	}

	// Reverse the string
	name := result.String()
	runes := []rune(name)
	for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
		runes[i], runes[j] = runes[j], runes[i]
	}

	return string(runes)
}

// colIndex converts an Excel column letter to a 0-based index.
func colIndex(name string) int {
	if name == "" {
		return -1
	}

	idx := 0
	for _, char := range name {
		if char < 'A' || char > 'Z' {
			return -1
		}
		idx = idx*26 + int(char-'A'+1)
	}

	return idx - 1
}

// v2: every data sheet (incl. Receitas) starts month triples at col C (index 2).
const dataMonthBase = 2 // Column C

// expenseMonthCols returns the three column letters for a month triple in a data sheet.
func expenseMonthCols(k int) (item, data, valor string) {
	item = colName(dataMonthBase + 3*k)
	data = colName(dataMonthBase + 3*k + 1)
	valor = colName(dataMonthBase + 3*k + 2)
	return item, data, valor
}

// expenseValorCol returns the valor column letter for a month in a data sheet.
func expenseValorCol(k int) string {
	return colName(dataMonthBase + 3*k + 2)
}

// receitasMonthCols / receitasValorCol — v2 unifies Receitas with the data-sheet column model.
func receitasMonthCols(k int) (item, data, valor string) { return expenseMonthCols(k) }
func receitasValorCol(k int) string                      { return expenseValorCol(k) }

// listasMonthCol returns the column letter for a month in a listas sheet (v2: months D..O).
func listasMonthCol(k int) string {
	if k < 0 || k > 11 {
		return ""
	}
	base := 3 // Column D is index 3
	return colName(base + k)
}
