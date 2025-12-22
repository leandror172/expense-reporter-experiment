package excel

import (
	"expense-reporter/internal/resolver"
	"fmt"
	"strings"

	"github.com/xuri/excelize/v2"
)

// LoadReferenceSheet loads the "Referência de Categorias" sheet and builds subcategory mappings
func LoadReferenceSheet(workbookPath string) (map[string][]resolver.SubcategoryMapping, error) {
	f, err := excelize.OpenFile(workbookPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open workbook: %w", err)
	}
	defer func() {
		if err := f.Close(); err != nil {
			// Log error but don't fail
		}
	}()

	// Read the reference sheet
	sheetName := "Referência de Categorias"
	rows, err := f.GetRows(sheetName)
	if err != nil {
		return nil, fmt.Errorf("failed to read reference sheet: %w", err)
	}

	mappings := make(map[string][]resolver.SubcategoryMapping)

	// Start from row 5 (skip headers which are rows 1-4)
	for i := 4; i < len(rows); i++ {
		row := rows[i]
		if len(row) < 3 {
			continue // Skip incomplete rows
		}

		mainType := strings.TrimSpace(row[0])     // Column A: Tipo Principal
		category := ""
		if len(row) > 1 {
			category = strings.TrimSpace(row[1])  // Column B: Categoria
		}
		subcategory := strings.TrimSpace(row[2])  // Column C: Sub-categoria

		if mainType == "" || subcategory == "" {
			continue // Skip rows with missing critical data
		}

		// Get row number from column D if available
		rowNum := 0
		if len(row) > 3 && row[3] != "" {
			fmt.Sscanf(row[3], "%d", &rowNum)
		}

		mapping := resolver.SubcategoryMapping{
			Subcategory: subcategory,
			SheetName:   mainType,
			Category:    category,
			RowNumber:   rowNum,
		}

		mappings[subcategory] = append(mappings[subcategory], mapping)
	}

	return mappings, nil
}

// FindSubcategoryRow finds the row number where a subcategory is located in a sheet
// Searches column B for the subcategory name
func FindSubcategoryRow(workbookPath, sheetName, subcategory string) (int, error) {
	f, err := excelize.OpenFile(workbookPath)
	if err != nil {
		return 0, fmt.Errorf("failed to open workbook: %w", err)
	}
	defer func() {
		if err := f.Close(); err != nil {
			// Log error but don't fail
		}
	}()

	rows, err := f.GetRows(sheetName)
	if err != nil {
		return 0, fmt.Errorf("failed to read sheet %s: %w", sheetName, err)
	}

	// Search column B (index 1) for subcategory
	for i, row := range rows {
		if len(row) > 1 {
			cellValue := strings.TrimSpace(row[1])
			if cellValue == subcategory {
				return i + 1, nil // Excel rows are 1-indexed
			}
		}
	}

	return 0, fmt.Errorf("subcategory '%s' not found in sheet '%s'", subcategory, sheetName)
}

// FindNextEmptyRow finds the next empty row in the specified column starting from startRow
func FindNextEmptyRow(workbookPath, sheetName, columnLetter string, startRow int) (int, error) {
	f, err := excelize.OpenFile(workbookPath)
	if err != nil {
		return 0, fmt.Errorf("failed to open workbook: %w", err)
	}
	defer func() {
		if err := f.Close(); err != nil {
			// Log error but don't fail
		}
	}()

	// Start scanning from startRow downward
	maxScan := 100 // Safety limit: don't scan more than 100 rows
	for i := 0; i < maxScan; i++ {
		row := startRow + i
		cellRef := fmt.Sprintf("%s%d", columnLetter, row)
		cellValue, err := f.GetCellValue(sheetName, cellRef)
		if err != nil {
			return 0, fmt.Errorf("failed to read cell %s: %w", cellRef, err)
		}

		if cellValue == "" {
			// Found empty cell
			// Check if we've crossed into next subcategory (column B has value)
			nextSubcatRef := fmt.Sprintf("B%d", row)
			nextSubcat, _ := f.GetCellValue(sheetName, nextSubcatRef)
			if nextSubcat != "" {
				return 0, fmt.Errorf("no empty cells available in this subcategory section")
			}

			return row, nil
		}
	}

	return 0, fmt.Errorf("no empty row found within %d rows", maxScan)
}
