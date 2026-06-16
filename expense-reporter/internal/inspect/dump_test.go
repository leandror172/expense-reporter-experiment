package inspect

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xuri/excelize/v2"
)

func TestDumpWorkbook_Basic(t *testing.T) {
	tmpDir := t.TempDir()
	workbookPath := filepath.Join(tmpDir, "test.xlsx")
	outputDir := filepath.Join(tmpDir, "output")

	f := excelize.NewFile()
	_, err := f.NewSheet("Mini")
	require.NoError(t, err)
	_, err = f.NewSheet("Outra")
	require.NoError(t, err)
	sheet1, sheet2 := "Mini", "Outra"
	require.NoError(t, f.DeleteSheet("Sheet1"))

	// Set up Mini sheet
	f.SetCellValue(sheet1, "A1", "Item")
	f.SetCellValue(sheet1, "B1", "Valor")
	f.SetRowHeight(sheet1, 1, 20)
	f.SetColWidth(sheet1, "A", "A", 10)
	boldStyle, err := f.NewStyle(&excelize.Style{Font: &excelize.Font{Bold: true}})
	require.NoError(t, err)
	require.NoError(t, f.SetCellStyle(sheet1, "A1", "A1", boldStyle))
	f.SetCellValue(sheet1, "A2", "Café")
	f.SetCellValue(sheet1, "B2", 3.5)
	f.SetCellFormula(sheet1, "B3", "SUM(B2:B2)")
	f.MergeCell(sheet1, "A4", "B4")
	f.SetCellValue(sheet1, "A4", "Nota")

	// Set up Outra sheet
	f.SetCellFormula(sheet2, "A1", "Mini!B3")

	err = f.SaveAs(workbookPath)
	require.NoError(t, err)

	err = DumpWorkbook(workbookPath, outputDir)
	require.NoError(t, err)

	// Read manifest.json
	manifestData, err := os.ReadFile(filepath.Join(outputDir, "manifest.json"))
	require.NoError(t, err)

	var manifest Manifest
	err = json.Unmarshal(manifestData, &manifest)
	require.NoError(t, err)

	assert.Len(t, manifest.Sheets, 2)
	sheetNames := make(map[string]bool)
	for _, sheet := range manifest.Sheets {
		sheetNames[sheet.Name] = true
	}
	assert.True(t, sheetNames["Mini"])
	assert.True(t, sheetNames["Outra"])

	// Read Mini.json
	miniData, err := os.ReadFile(filepath.Join(outputDir, "Mini.json"))
	require.NoError(t, err)

	var miniDump SheetDump
	err = json.Unmarshal(miniData, &miniDump)
	require.NoError(t, err)

	assert.GreaterOrEqual(t, miniDump.Dimensions.Rows, 4)
	assert.GreaterOrEqual(t, miniDump.Dimensions.Cols, 2)

	// Find cell A2
	var a2Cell *Cell
	for _, row := range miniDump.Rows {
		for _, cell := range row.Cells {
			if cell.Col == "A" && row.Row == 2 {
				a2Cell = &cell
				break
			}
		}
	}
	require.NotNil(t, a2Cell)
	assert.Equal(t, "Café", a2Cell.Value)

	// Find B3 formula
	var b3Cell *Cell
	for _, row := range miniDump.Rows {
		for _, cell := range row.Cells {
			if cell.Col == "B" && row.Row == 3 {
				b3Cell = &cell
				break
			}
		}
	}
	require.NotNil(t, b3Cell)
	assert.Equal(t, "SUM(B2:B2)", b3Cell.Formula)

	// Check merged cells
	foundMerge := false
	for _, merge := range miniDump.MergedCells {
		if merge.Range == "A4:B4" && merge.Value == "Nota" {
			foundMerge = true
			break
		}
	}
	assert.True(t, foundMerge)

	// Read Outra.json
	outData, err := os.ReadFile(filepath.Join(outputDir, "Outra.json"))
	require.NoError(t, err)

	var outDump SheetDump
	err = json.Unmarshal(outData, &outDump)
	require.NoError(t, err)

	foundRef := false
	for _, ref := range outDump.CrossSheetRefs {
		if ref == "Mini" {
			foundRef = true
			break
		}
	}
	assert.True(t, foundRef)
}

func TestDumpWorkbook_Errors(t *testing.T) {
	tmpDir := t.TempDir()

	tests := []struct {
		name          string
		workbookPath  string
		outputDir     string
		expectError   bool
		errorContains string
	}{
		{
			name:         "nonexistent workbook",
			workbookPath: "/path/does/not/exist.xlsx",
			outputDir:    filepath.Join(tmpDir, "output"),
			expectError:  true,
		},
		{
			name:         "unwritable output dir",
			workbookPath: filepath.Join(tmpDir, "dummy.xlsx"),
			outputDir:    filepath.Join(tmpDir, "file"), // This is a file
			expectError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.workbookPath != "/path/does/not/exist.xlsx" {
				f := excelize.NewFile()
				err := f.SaveAs(tt.workbookPath)
				require.NoError(t, err)

				// Create a file where we want to make directory
				file, err := os.Create(tt.outputDir)
				require.NoError(t, err)
				file.Close()
			}

			err := DumpWorkbook(tt.workbookPath, tt.outputDir)
			if tt.expectError {
				assert.Error(t, err)
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestSanitizeFilename(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "plain name",
			input:    "Sheet1",
			expected: "Sheet1",
		},
		{
			name:     "with slashes",
			input:    "Sheet/Name:Test",
			expected: "Sheet_Name:Test", // only path separators are replaced
		},
		{
			name:     "empty input",
			input:    "",
			expected: "", // Implementation returns empty string
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizeFilename(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}
