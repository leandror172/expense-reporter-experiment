package main

// Subcat is one subcategory block. v2: composed sub-items are a single col-B string.
type Subcat struct {
	Name string
}

// Categoria groups subcategories under one bold category label.
type Categoria struct {
	Name string
	Subs []Subcat
}

// ExpenseSheet is one of Fixas/Variáveis/Extras/Adicionais.
type ExpenseSheet struct {
	Name string
	Cats []Categoria
}

// ReceitasBlock is one income block (Salário, 13°...).
type ReceitasBlock struct {
	Category string // income category, e.g. "Receita"
	Label    string // block label
}

const headroomRows = 3 // §2 default

// perGroupPctRows toggles the per-group "% sobre Despesas/Receitas" rows (§4.2 ⚠).
// Golden master omits them → default false for Phase A convergence.
const perGroupPctRows = false

// Fixed partial taxonomy for the Phase A template (from task brief).
func buildTaxonomy() ([]ExpenseSheet, []ReceitasBlock) {
	fixas := ExpenseSheet{Name: "Fixas", Cats: []Categoria{
		{Name: "Habitação", Subs: []Subcat{{Name: "Diarista"}, {Name: "Aluguel"}}},
		{Name: "Lazer", Subs: []Subcat{{Name: "Netflix"}, {Name: "Spotify"}}},
	}}
	variaveis := ExpenseSheet{Name: "Variáveis", Cats: []Categoria{
		{Name: "Alimentação / Limpeza", Subs: []Subcat{{Name: "Supermercado"}, {Name: "Padaria"}}},
		{Name: "Pets", Subs: []Subcat{{Name: "Orion - Consultas"}, {Name: "Orion - Ração"}}},
		{Name: "Transporte", Subs: []Subcat{{Name: "Metrô"}}},
	}}
	extras := ExpenseSheet{Name: "Extras", Cats: []Categoria{
		{Name: "Saúde", Subs: []Subcat{{Name: "Médico"}, {Name: "Dentista"}}},
		{Name: "Manutenção / prevenção", Subs: []Subcat{{Name: "Carro"}, {Name: "Casa"}}},
	}}
	adicionais := ExpenseSheet{Name: "Adicionais", Cats: []Categoria{
		{Name: "Lazer", Subs: []Subcat{{Name: "Viagens"}, {Name: "Jogos"}}},
		{Name: "Outros", Subs: []Subcat{{Name: "Presentes"}, {Name: "Papelaria"}}},
	}}
	receitas := []ReceitasBlock{
		{Category: "Receita", Label: "Salário"},
		{Category: "Receita", Label: "13°"},
	}
	return []ExpenseSheet{fixas, variaveis, extras, adicionais}, receitas
}
