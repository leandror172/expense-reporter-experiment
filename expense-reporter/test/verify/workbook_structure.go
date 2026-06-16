//go:build acceptance

package verify

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"

	"expense-reporter/internal/inspect"
	"expense-reporter/test/harness"
)

// sheetNames extracts the ordered sheet-name list from a manifest.
func sheetNames(m *inspect.Manifest) []string {
	names := make([]string, len(m.Sheets))
	for i, s := range m.Sheets {
		names[i] = s.Name
	}
	return names
}

// WorkbookStructureMatches asserts that the generated workbook matches the expected structure.
func WorkbookStructureMatches(expectedDumpDir string) func(*harness.Context) {
	return func(ctx *harness.Context) {
		ctx.T.Helper()

		workbookPath := ctx.Artifacts["generated-workbook"]
		if !assert.NotEmpty(ctx.T, workbookPath, "generated-workbook artifact must be present") {
			return
		}

		actualDumpDir := filepath.Join(ctx.WorkDir, "actual-dump")
		if err := inspect.DumpWorkbook(workbookPath, actualDumpDir); !assert.NoError(ctx.T, err, "dumping workbook should succeed") {
			return
		}

		expectedManifest := readManifest(ctx.T, expectedDumpDir)
		actualManifest := readManifest(ctx.T, actualDumpDir)

		assert.Equal(ctx.T, sheetNames(expectedManifest), sheetNames(actualManifest),
			"sheet list must match (ignoring Source and dimensions)")

		for _, expectedSheetInfo := range expectedManifest.Sheets {
			expectedSheetDump := loadSheetDump(ctx.T, expectedDumpDir, expectedSheetInfo.Name)
			actualSheetDump := loadSheetDump(ctx.T, actualDumpDir, expectedSheetInfo.Name)

			compareDimensions(ctx.T, expectedSheetDump.Dimensions, actualSheetDump.Dimensions, expectedSheetInfo.Name)

			expectedCells := buildCellMap(expectedSheetDump.Rows)
			actualCells := buildCellMap(actualSheetDump.Rows)

			compareCells(ctx.T, expectedCells, actualCells, expectedSheetInfo.Name)

			compareMerges(ctx.T, expectedSheetDump.MergedCells, actualSheetDump.MergedCells, expectedSheetInfo.Name)

			for i, expectedRow := range expectedSheetDump.Rows {
				if i >= len(actualSheetDump.Rows) {
					assert.Fail(ctx.T, "missing row", "row %d missing from actual dump in sheet %s", expectedRow.Row, expectedSheetInfo.Name)
					continue
				}
				actualRow := actualSheetDump.Rows[i]
				assert.Equal(ctx.T, expectedRow.RowType, actualRow.RowType, "row type mismatch at row %d in sheet %s", expectedRow.Row, expectedSheetInfo.Name)
				assert.Equal(ctx.T, expectedRow.RowFill, actualRow.RowFill, "row fill mismatch at row %d in sheet %s", expectedRow.Row, expectedSheetInfo.Name)
			}
		}
	}
}

func readManifest(t *testing.T, dumpDir string) *inspect.Manifest {
	t.Helper()

	path := filepath.Join(dumpDir, "manifest.json")
	data, err := os.ReadFile(path)
	if !assert.NoError(t, err, "reading manifest %s", path) {
		return nil
	}

	var manifest inspect.Manifest
	if err := json.Unmarshal(data, &manifest); !assert.NoError(t, err, "unmarshaling manifest %s", path) {
		return nil
	}
	return &manifest
}

func loadSheetDump(t *testing.T, dumpDir, sheetName string) *inspect.SheetDump {
	t.Helper()

	path := filepath.Join(dumpDir, sheetName+".json")
	data, err := os.ReadFile(path)
	if !assert.NoError(t, err, "reading sheet dump %s", path) {
		return nil
	}

	var sheetDump inspect.SheetDump
	if err := json.Unmarshal(data, &sheetDump); !assert.NoError(t, err, "unmarshaling sheet dump %s", path) {
		return nil
	}
	return &sheetDump
}

func buildCellMap(rows []inspect.RowDump) map[string]inspect.Cell {
	cellMap := make(map[string]inspect.Cell)
	for _, row := range rows {
		for _, cell := range row.Cells {
			key := fmt.Sprintf("%s%d", cell.Col, row.Row)
			cellMap[key] = cell
		}
	}
	return cellMap
}

func compareDimensions(t *testing.T, expected, actual inspect.Dim, sheetName string) {
	t.Helper()
	assert.Equal(t, expected, actual, "dimensions mismatch in sheet %s", sheetName)
}

func compareCells(t *testing.T, expectedCells, actualCells map[string]inspect.Cell, sheetName string) {
	t.Helper()

	allKeys := make(map[string]bool)
	for k := range expectedCells {
		allKeys[k] = true
	}
	for k := range actualCells {
		allKeys[k] = true
	}

	for key := range allKeys {
		expectedCell, expOk := expectedCells[key]
		actualCell, actOk := actualCells[key]

		if !expOk {
			assert.Fail(t, "extra cell", "cell %s present in actual but not expected in sheet %s", key, sheetName)
			continue
		}
		if !actOk {
			assert.Fail(t, "missing cell", "cell %s present in expected but not actual in sheet %s", key, sheetName)
			continue
		}

		assert.Equal(t, expectedCell.Value, actualCell.Value, "value mismatch for cell %s in sheet %s", key, sheetName)
		assert.Equal(t, expectedCell.Formula, actualCell.Formula, "formula mismatch for cell %s in sheet %s", key, sheetName)

		expectedStyle := expectedCell.Style
		actualStyle := actualCell.Style

		assert.Equal(t, expectedStyle.BgColor, actualStyle.BgColor, "bgColor mismatch for cell %s in sheet %s", key, sheetName)
		assert.Equal(t, expectedStyle.Bold, actualStyle.Bold, "bold mismatch for cell %s in sheet %s", key, sheetName)
		assert.Equal(t, expectedStyle.BorderTop, actualStyle.BorderTop, "borderTop mismatch for cell %s in sheet %s", key, sheetName)
		assert.Equal(t, expectedStyle.BorderBottom, actualStyle.BorderBottom, "borderBottom mismatch for cell %s in sheet %s", key, sheetName)
		assert.Equal(t, expectedStyle.BorderLeft, actualStyle.BorderLeft, "borderLeft mismatch for cell %s in sheet %s", key, sheetName)
		assert.Equal(t, expectedStyle.BorderRight, actualStyle.BorderRight, "borderRight mismatch for cell %s in sheet %s", key, sheetName)
	}
}

func compareMerges(t *testing.T, expectedMerges, actualMerges []inspect.Merge, sheetName string) {
	t.Helper()

	expectedMap := make(map[string]string)
	for _, m := range expectedMerges {
		expectedMap[m.Range] = m.Value
	}

	actualMap := make(map[string]string)
	for _, m := range actualMerges {
		actualMap[m.Range] = m.Value
	}

	allRanges := make(map[string]bool)
	for r := range expectedMap {
		allRanges[r] = true
	}
	for r := range actualMap {
		allRanges[r] = true
	}

	for rangeName := range allRanges {
		expectedValue, expOk := expectedMap[rangeName]
		actualValue, actOk := actualMap[rangeName]

		if !expOk {
			assert.Fail(t, "extra merge", "merge %s present in actual but not expected in sheet %s", rangeName, sheetName)
			continue
		}
		if !actOk {
			assert.Fail(t, "missing merge", "merge %s present in expected but not actual in sheet %s", rangeName, sheetName)
			continue
		}

		assert.Equal(t, expectedValue, actualValue, "value mismatch for merge %s in sheet %s", rangeName, sheetName)
	}
}
