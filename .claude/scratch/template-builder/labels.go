// Package main provides centralized workbook strings for internationalization.
package main

// Labels holds all user-visible strings for the workbook application.
// The field names are in English to indicate semantic role, while the values
// contain localized text.
type Labels struct {
	// headers group
	Month     string
	Item      string
	Date      string
	Amount    string
	Total     string
	TotalDash string

	// listas percent rows
	PctOfExpenses string
	PctOfRevenue  string

	// formatted totals
	TotalCategoryFmt      string
	TotalSheetExpensesFmt string
	SheetExpensesFmt      string

	// sheet / section names
	RevenueSheet string

	// saldo block
	Revenue            string
	Investments        string
	TotalIncome        string
	TotalExpenses      string
	ExpenseShareHeader string
	IncomeShareHeader  string
	Balance            string
	Dollar             string

	// MonthNames contains the names of months in Portuguese (Brazil).
	MonthNames [12]string
}

// newPtBRLabels returns a Labels struct with all strings localized for Brazilian Portuguese.
func newPtBRLabels() Labels {
	return Labels{
		Month:                 "Mês",
		Item:                  "Item",
		Date:                  "Data",
		Amount:                "Valor",
		Total:                 "Total",
		TotalDash:             "–",
		PctOfExpenses:         "% sobre despesas",
		PctOfRevenue:          "% sobre receita",
		TotalCategoryFmt:      "Total %s",
		TotalSheetExpensesFmt: "Total despesas %s",
		SheetExpensesFmt:      "Despesas %s",
		RevenueSheet:          "Receitas",
		Revenue:               "Receita",
		Investments:           "Investimentos",
		TotalIncome:           "Total renda",
		TotalExpenses:         "Total despesas",
		ExpenseShareHeader:    "Porcentagem da despesa",
		IncomeShareHeader:     "Porcentagem da renda",
		Balance:               "Saldo",
		Dollar:                "Dólar",
		MonthNames: [12]string{
			"Janeiro",
			"Fevereiro",
			"Março",
			"Abril",
			"Maio",
			"Junho",
			"Julho",
			"Agosto",
			"Setembro",
			"Outubro",
			"Novembro",
			"Dezembro",
		},
	}
}
