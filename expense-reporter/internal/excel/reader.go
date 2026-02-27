package excel

import (
	"expense-reporter/internal/logger"
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
	SheetName       string
	ColumnLetter    string
	StartRow        int
	SubcategoryName string
	ExpenseIndex    int // original index in the expenses slice, for result mapping
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
				logger.Debug("FindSubcategoryRow: found", "subcategory", subcategory, "sheet", sheetName, "row", i+1)
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

	logger.Debug("FindNextEmptyRow: scanning", "sheet", sheetName, "col", columnLetter, "startRow", startRow, "subcategory", subcategoryName)

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
			// Found empty cell in the month column.
			// Verify we haven't drifted past this subcategory's section by checking
			// column B: if it holds a *different* subcategory name we've hit the boundary.
			nextSubcatRef := fmt.Sprintf("B%d", row)
			nextSubcat, _ := f.GetCellValue(sheetName, nextSubcatRef)

			logger.Debug("FindNextEmptyRow: empty cell found", "row", row, "colB", nextSubcat)

			if nextSubcat != "" && nextSubcat != subcategoryName {
				logger.Debug("FindNextEmptyRow: boundary hit", "row", row, "expected", subcategoryName, "got", nextSubcat)
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

		// Scan once, find all — stop tracking a subcategory on first match
		// (mirrors FindSubcategoryRow's first-match behaviour)
		for i, row := range rows {
			if len(row) > 1 {
				cellValue := strings.TrimSpace(row[1]) // Column B
				if needed[cellValue] {
					sheetResults[cellValue] = i + 1 // Excel rows are 1-indexed
					delete(needed, cellValue)        // first match wins
					logger.Debug("FindSubcategoryRowBatch: found", "subcategory", cellValue, "sheet", sheetName, "row", i+1)
				}
			}
			if len(needed) == 0 {
				break // all requested subcategories found
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

// AllocateEmptyRows opens the workbook once and assigns a target row to each request
// in order.  Requests sharing the same (sheet, subcategory, month-column) are allocated
// sequentially: once a row is claimed for one expense in this batch, the next request
// for the same section starts scanning from the row *after* that one.  Per-request
// errors (section full) are returned in the errors map rather than aborting the whole batch.
// Returns: map[requestIndex]targetRow, map[requestIndex]error
func AllocateEmptyRows(workbookPath string, requests []EmptyRowRequest) (map[int]int, error) {
	if len(requests) == 0 {
		return map[int]int{}, nil
	}

	f, err := excelize.OpenFile(workbookPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open workbook: %w", err)
	}
	defer f.Close()

	results := make(map[int]int, len(requests))

	// nextScanRow tracks where to resume scanning for each unique section.
	// Key: "sheet\x00subcategory\x00column"
	type sectionKey struct {
		sheet, subcategory, column string
	}
	nextScanRow := make(map[sectionKey]int)

	maxScan := 100

	for idx, req := range requests {
		key := sectionKey{req.SheetName, req.SubcategoryName, req.ColumnLetter}

		startRow := req.StartRow
		if prev, ok := nextScanRow[key]; ok && prev > startRow {
			startRow = prev // resume after last allocation for this section
		}

		logger.Debug("AllocateEmptyRows: scanning", "reqIdx", idx,
			"sheet", req.SheetName, "subcategory", req.SubcategoryName,
			"col", req.ColumnLetter, "startRow", startRow)

		found := false
		for i := 0; i < maxScan; i++ {
			row := startRow + i
			cellRef := fmt.Sprintf("%s%d", req.ColumnLetter, row)
			cellValue, err := f.GetCellValue(req.SheetName, cellRef)
			if err != nil {
				return nil, fmt.Errorf("failed to read cell %s in sheet %s: %w", cellRef, req.SheetName, err)
			}

			if cellValue == "" {
				// Verify we haven't crossed into a different subcategory section
				bRef := fmt.Sprintf("B%d", row)
				bVal, _ := f.GetCellValue(req.SheetName, bRef)

				logger.Debug("AllocateEmptyRows: empty cell", "row", row, "colB", bVal)

				if bVal != "" && bVal != req.SubcategoryName {
					// Hit the boundary — section is full for this month
					logger.Debug("AllocateEmptyRows: section full", "subcategory", req.SubcategoryName,
						"boundary", bVal, "row", row)
					// Don't abort the whole batch; leave this request out of results.
					// The caller will see the missing entry and record the error.
					break
				}

				results[idx] = row
				nextScanRow[key] = row + 1 // next request for same section starts here
				found = true
				logger.Debug("AllocateEmptyRows: allocated", "reqIdx", idx, "row", row)
				break
			}
		}

		if !found {
			logger.Warn("AllocateEmptyRows: no row allocated", "reqIdx", idx,
				"subcategory", req.SubcategoryName, "sheet", req.SheetName)
		}
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

// BuildPathIndex creates hierarchical index from flat mappings
// This enables progressive resolution: "Diarista" → "Habitação,Diarista" → "Fixas,Habitação,Diarista"
func BuildPathIndex(flatMappings map[string][]resolver.SubcategoryMapping) *resolver.PathIndex {
	index := &resolver.PathIndex{
		BySubcategory: flatMappings,
		ByFullPath:    make(map[string]*resolver.SubcategoryMapping),
		By2Level:      make(map[string][]resolver.SubcategoryMapping),
		By1Level:      make(map[string][]resolver.SubcategoryMapping),
	}

	for _, mappings := range flatMappings {
		for i := range mappings {
			m := mappings[i]

			// 1-level: subcategory only
			subKey := resolver.NormalizePath(m.Subcategory)
			index.By1Level[subKey] = append(index.By1Level[subKey], m)

			// 2-level: category,subcategory
			if m.Category != "" {
				twoKey := resolver.NormalizePath(m.Category + "," + m.Subcategory)
				index.By2Level[twoKey] = append(index.By2Level[twoKey], m)
			}

			// 3-level: sheet,category,subcategory
			fullKey := resolver.NormalizePath(m.SheetName + "," + m.Category + "," + m.Subcategory)
			index.ByFullPath[fullKey] = &m
		}
	}

	return index
}
