package main

import (
	"fmt"
	"log"

	"github.com/xuri/excelize/v2"
)

func main() {
	f, err := excelize.OpenFile("Z:\\Meu Drive\\controle\\code\\Planilha_Normalized_Final.xlsx")
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	fmt.Println("DETAILED REFERÊNCIA VALIDATION COLUMNS ANALYSIS")
	fmt.Println(strings.Repeat("=", 80))

	rows, _ := f.GetRows("Referência de Categorias")

	// Header row (should be row 4 based on earlier analysis)
	headerRow := 3 // 0-indexed, so row 4

	if len(rows) <= headerRow {
		fmt.Println("Not enough rows")
		return
	}

	fmt.Println("\nALL COLUMNS A-BB:")
	for i := 0; i < 54; i++ { // A-BB = 54 columns
		colName, _ := excelize.ColumnNumberToName(i + 1)
		header := ""
		if i < len(rows[headerRow]) {
			header = rows[headerRow][i]
		}

		// Show even if empty
		if i < 26 || header != "" {
			fmt.Printf("%3s: '%s'\n", colName, header)
		}
	}

	// Sample data from first actual subcategory
	fmt.Println("\nSAMPLE DATA (First subcategory - Aluguel probably):")
	if len(rows) > 4 {
		dataRow := 4
		for i := 0; i < 30; i++ {
			colName, _ := excelize.ColumnNumberToName(i + 1)
			val := ""
			if i < len(rows[dataRow]) {
				val = rows[dataRow][i]
			}
			if val != "" {
				fmt.Printf("%3s: %s\n", colName, val)
			}
		}
	}

	// Check if columns have formulas
	fmt.Println("\nFORMULA CHECK (G-R, first data row):")
	if len(rows) > 4 {
		dataRow := 4
		for col := 7; col <= 18; col++ { // G-R (columns 7-18)
			colName, _ := excelize.ColumnNumberToName(col)
			cell := fmt.Sprintf("%s%d", colName, dataRow+1) // Excel 1-indexed
			formula, _ := f.GetCellFormula("Referência de Categorias", cell)
			value, _ := f.GetCellValue("Referência de Categorias", cell)

			if formula != "" || value != "" {
				fmt.Printf("%3s: formula='%s' value='%s'\n", colName, formula, value)
			}
		}
	}
}
