package excel

import (
	"expense-reporter/internal/resolver"
	"fmt"
	"strings"

	"github.com/xuri/excelize/v2"
)

// SubcategoryLookupRequest represents a request to find a subcategory row
type SubcategoryLookupRequest struct {
	SheetName   string
	Subcategory string
}

// EmptyRowRequest represents a request to find the next empty row
type EmptyRowRequest struct {
	SheetName      string
	ColumnLetter   string
	StartRow       int
	SubcategoryName string
}

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

		// Get total row from column F if available
		totalRow := 0
		if len(row) > 5 && row[5] != "" {
			fmt.Sscanf(row[5], "%d", &totalRow)
		}

		mapping := resolver.SubcategoryMapping{
			Subcategory: subcategory,
			SheetName:   mainType,
			Category:    category,
			RowNumber:   rowNum,
			TotalRow:    totalRow,
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
// subcategoryName is used to detect when we've crossed into a different subcategory section
func FindNextEmptyRow(workbookPath, sheetName, columnLetter string, startRow int, subcategoryName string) (int, error) {
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
			// Check if we've crossed into a DIFFERENT subcategory (column B has different value)
			// Note: Column B may have merged cells, so all rows in the subcategory section
			// will return the same subcategory name
			nextSubcatRef := fmt.Sprintf("B%d", row)
			nextSubcat, _ := f.GetCellValue(sheetName, nextSubcatRef)

			// If column B has a value AND it's different from our current subcategory,
			// we've crossed into another subcategory section
			if nextSubcat != "" && nextSubcat != subcategoryName {
				return 0, fmt.Errorf("no empty cells available in this subcategory section")
			}

			return row, nil
		}
	}

	return 0, fmt.Errorf("no empty row found within %d rows", maxScan)
}

// FindSubcategoryRowBatch finds multiple subcategory rows in a single file open
// Returns map[sheetName]map[subcategory]rowNumber for fast lookup
// This eliminates N file opens for N subcategory lookups
func FindSubcategoryRowBatch(workbookPath string, requests []SubcategoryLookupRequest) (map[string]map[string]int, error) {
	if len(requests) == 0 {
		return map[string]map[string]int{}, nil
	}

	// Open workbook ONCE
	f, err := excelize.OpenFile(workbookPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open workbook: %w", err)
	}
	defer f.Close()

	// Group requests by sheet name
	requestsBySheet := make(map[string][]string)
	for _, req := range requests {
		requestsBySheet[req.SheetName] = append(requestsBySheet[req.SheetName], req.Subcategory)
	}

	// Process each sheet once
	results := make(map[string]map[string]int)

	for sheetName, subcategories := range requestsBySheet {
		sheetResults := make(map[string]int)

		// Read sheet rows once
		rows, err := f.GetRows(sheetName)
		if err != nil {
			return nil, fmt.Errorf("failed to read sheet %s: %w", sheetName, err)
		}

		// Create lookup set
		needed := make(map[string]bool)
		for _, subcat := range subcategories {
			needed[subcat] = true
		}

		// Scan once, find all
		for i, row := range rows {
			if len(row) > 1 {
				cellValue := strings.TrimSpace(row[1]) // Column B
				if needed[cellValue] {
					sheetResults[cellValue] = i + 1 // Excel rows are 1-indexed
				}
			}
		}

		// Verify all found
		for _, subcat := range subcategories {
			if _, found := sheetResults[subcat]; !found {
				return nil, fmt.Errorf("subcategory '%s' not found in sheet '%s'", subcat, sheetName)
			}
		}

		results[sheetName] = sheetResults
	}

	return results, nil
}

// FindNextEmptyRowBatch finds empty rows for multiple subcategories in one file open
// Returns map[sheetName]map[startRow]nextEmptyRow
func FindNextEmptyRowBatch(workbookPath string, requests []EmptyRowRequest) (map[string]map[int]int, error) {
	if len(requests) == 0 {
		return map[string]map[int]int{}, nil
	}

	// Open workbook ONCE
	f, err := excelize.OpenFile(workbookPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open workbook: %w", err)
	}
	defer f.Close()

	// Group by sheet
	requestsBySheet := make(map[string][]EmptyRowRequest)
	for _, req := range requests {
		requestsBySheet[req.SheetName] = append(requestsBySheet[req.SheetName], req)
	}

	// Process each sheet
	results := make(map[string]map[int]int)

	for sheetName, sheetRequests := range requestsBySheet {
		sheetResults := make(map[int]int)

		for _, req := range sheetRequests {
			maxScan := 100
			found := false

			for i := 0; i < maxScan; i++ {
				row := req.StartRow + i
				cellRef := fmt.Sprintf("%s%d", req.ColumnLetter, row)
				cellValue, err := f.GetCellValue(sheetName, cellRef)
				if err != nil {
					return nil, fmt.Errorf("failed to read cell %s: %w", cellRef, err)
				}

				if cellValue == "" {
					// Check subcategory boundary
					nextSubcatRef := fmt.Sprintf("B%d", row)
					nextSubcat, _ := f.GetCellValue(sheetName, nextSubcatRef)

					if nextSubcat != "" && nextSubcat != req.SubcategoryName {
						return nil, fmt.Errorf("no empty cells available in subcategory section for %s", req.SubcategoryName)
					}

					sheetResults[req.StartRow] = row
					found = true
					break
				}
			}

			if !found {
				return nil, fmt.Errorf("no empty row found within %d rows for %s", maxScan, req.SubcategoryName)
			}
		}

		results[sheetName] = sheetResults
	}

	return results, nil
}

// CapacityInfo holds information about subcategory section capacity
type CapacityInfo struct {
	SubcategoryRow int
	TotalRow       int
	UsedRows       int
	AvailableRows  int
	IsFull         bool
}

// CapacityCheckRequest represents a request to check capacity
type CapacityCheckRequest struct {
	SheetName      string
	SubcategoryRow int
	TotalRow       int
	MonthColumn    string
}

// CheckCapacityBatch checks capacity for multiple subcategories in one file open
// Returns map[sheetName]map[subcategoryRow]*CapacityInfo
func CheckCapacityBatch(
	workbookPath string,
	requests []CapacityCheckRequest,
) (map[string]map[int]*CapacityInfo, error) {
	if len(requests) == 0 {
		return map[string]map[int]*CapacityInfo{}, nil
	}

	f, err := excelize.OpenFile(workbookPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open workbook: %w", err)
	}
	defer f.Close()

	// Group requests by sheet
	requestsBySheet := make(map[string][]CapacityCheckRequest)
	for _, req := range requests {
		requestsBySheet[req.SheetName] = append(requestsBySheet[req.SheetName], req)
	}

	results := make(map[string]map[int]*CapacityInfo)

	for sheetName, sheetRequests := range requestsBySheet {
		sheetResults := make(map[int]*CapacityInfo)

		for _, req := range sheetRequests {
			// Skip capacity check if no total row info
			if req.TotalRow == 0 {
				sheetResults[req.SubcategoryRow] = &CapacityInfo{
					SubcategoryRow: req.SubcategoryRow,
					TotalRow:       0,
					UsedRows:       0,
					AvailableRows:  999, // Assume unlimited
					IsFull:         false,
				}
				continue
			}

			// Validate total row is after subcategory row
			if req.TotalRow <= req.SubcategoryRow {
				sheetResults[req.SubcategoryRow] = &CapacityInfo{
					SubcategoryRow: req.SubcategoryRow,
					TotalRow:       req.TotalRow,
					UsedRows:       0,
					AvailableRows:  999, // Invalid config, skip check
					IsFull:         false,
				}
				continue
			}

			// Count used rows by scanning from subcategoryRow to totalRow
			usedRows := 0
			maxCapacity := req.TotalRow - req.SubcategoryRow - 1 // -1 for TOTAL row itself

			for row := req.SubcategoryRow; row < req.TotalRow; row++ {
				cellRef := fmt.Sprintf("%s%d", req.MonthColumn, row)
				cellValue, err := f.GetCellValue(sheetName, cellRef)
				if err != nil {
					return nil, fmt.Errorf("failed to read cell %s: %w", cellRef, err)
				}

				if cellValue != "" {
					usedRows++
				}
			}

			availableRows := maxCapacity - usedRows
			isFull := availableRows <= 0

			sheetResults[req.SubcategoryRow] = &CapacityInfo{
				SubcategoryRow: req.SubcategoryRow,
				TotalRow:       req.TotalRow,
				UsedRows:       usedRows,
				AvailableRows:  availableRows,
				IsFull:         isFull,
			}
		}

		results[sheetName] = sheetResults
	}

	return results, nil
}
