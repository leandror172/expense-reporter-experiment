package taxonomy

// e and mo are terse constructors for the Phase-B fake dataset below.
func e(item string, day int, value float64) Entry { return Entry{Item: item, Day: day, Value: value} }
func mo(entries ...Entry) []Entry                 { return entries }

// buildTaxonomy returns the taxonomy plus a SMALL purpose-built Phase-B dataset
// (Janeiro + Fevereiro only) whose fill counts deliberately vary to exercise:
//   - a 3-row block (Diarista: 3 entries/month),
//   - a 2-row block with a lighter month leaving a merged headroom tail above the
//     total (Aluguel: 2 in Jan, 1 in Feb),
//   - 1-row blocks (everything else),
//   - income blocks feeding the Listas percent denominators.
func buildTaxonomy() ([]ExpenseSheet, []RevenueBlock) {
	fixas := ExpenseSheet{Name: "Fixas", Cats: []Category{
		{Name: "Habitação", Subs: []Subcat{
			{Name: "Diarista", Months: [12][]Entry{
				mo(e("Diarista", 3, 150.00), e("Diarista", 10, 160.00), e("Diarista", 17, 155.50)),
				mo(e("Diarista", 7, 150.00), e("Diarista", 14, 165.00), e("Diarista", 21, 158.00)),
			}},
			{Name: "Aluguel", Months: [12][]Entry{
				mo(e("Aluguel", 5, 1200.00), e("Condomínio", 5, 380.75)),
				mo(e("Aluguel", 5, 1200.00)),
			}},
		}},
		{Name: "Lazer", Subs: []Subcat{
			{Name: "Netflix", Months: [12][]Entry{
				mo(e("Netflix", 2, 39.90)), mo(e("Netflix", 2, 39.90)),
			}},
			{Name: "Spotify", Months: [12][]Entry{
				mo(e("Spotify", 8, 19.90)), mo(e("Spotify", 8, 21.90)),
			}},
		}},
	}}
	variaveis := ExpenseSheet{Name: "Variáveis", Cats: []Category{
		{Name: "Alimentação / Limpeza", Subs: []Subcat{
			{Name: "Supermercado", Months: [12][]Entry{
				mo(e("Mercado", 6, 543.21)), mo(e("Mercado", 9, 612.00)),
			}},
			{Name: "Padaria", Months: [12][]Entry{
				mo(e("Padaria", 4, 45.00)), mo(e("Padaria", 12, 52.30)),
			}},
		}},
		{Name: "Pets", Subs: []Subcat{
			{Name: "Orion - Consultas", Months: [12][]Entry{
				mo(e("Consulta", 15, 200.00)), mo(e("Consulta", 18, 210.00)),
			}},
			{Name: "Orion - Ração", Months: [12][]Entry{
				mo(e("Ração", 1, 120.00)), mo(e("Ração", 3, 120.00)),
			}},
		}},
		{Name: "Transporte", Subs: []Subcat{
			{Name: "Metrô", Months: [12][]Entry{
				mo(e("Bilhete", 1, 100.00)), mo(e("Bilhete", 1, 100.00)),
			}},
		}},
	}}
	extras := ExpenseSheet{Name: "Extras", Cats: []Category{
		{Name: "Saúde", Subs: []Subcat{
			{Name: "Médico", Months: [12][]Entry{
				mo(e("Consulta", 10, 300.00)), mo(e("Consulta", 20, 320.00)),
			}},
			{Name: "Dentista", Months: [12][]Entry{
				mo(e("Limpeza", 11, 250.00)), mo(e("Limpeza", 11, 250.00)),
			}},
		}},
		{Name: "Manutenção / prevenção", Subs: []Subcat{
			{Name: "Carro", Months: [12][]Entry{
				mo(e("Revisão", 9, 480.00)), mo(e("Pneu", 9, 600.00)),
			}},
			{Name: "Casa", Months: [12][]Entry{
				mo(e("Pintura", 14, 350.00)), mo(e("Pintura", 14, 350.00)),
			}},
		}},
	}}
	adicionais := ExpenseSheet{Name: "Adicionais", Cats: []Category{
		{Name: "Lazer", Subs: []Subcat{
			{Name: "Viagens", Months: [12][]Entry{
				mo(e("Hotel", 3, 800.00)), mo(e("Passagem", 4, 950.00)),
			}},
			{Name: "Jogos", Months: [12][]Entry{
				mo(e("Steam", 7, 99.90)), mo(e("Steam", 7, 99.90)),
			}},
		}},
		{Name: "Outros", Subs: []Subcat{
			{Name: "Presentes", Months: [12][]Entry{
				mo(e("Presente", 5, 150.00)), mo(e("Presente", 5, 150.00)),
			}},
			{Name: "Papelaria", Months: [12][]Entry{
				mo(e("Caderno", 18, 45.00)), mo(e("Caneta", 18, 25.00)),
			}},
		}},
	}}
	revenue := []RevenueBlock{
		{Category: "Receita", Label: "Salário", Months: [12][]Entry{
			mo(e("Salário", 5, 5000.00)), mo(e("Salário", 5, 5000.00)),
		}},
		{Category: "Receita", Label: "13°", Months: [12][]Entry{
			mo(e("13° parcela", 20, 2500.00)), mo(e("13° parcela", 20, 2500.00)),
		}},
	}
	return []ExpenseSheet{fixas, variaveis, extras, adicionais}, revenue
}
