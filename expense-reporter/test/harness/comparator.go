//go:build acceptance

package harness

import (
	"bufio"
	"encoding/csv"
	"os"
	"strings"
	"testing"
)

// ReadCSVFile reads a semicolon-delimited CSV, skipping lines starting with '#'.
// Returns [][]string (rows of fields).
func ReadCSVFile(t *testing.T, path string) [][]string {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("ReadCSVFile: open %q: %v", path, err)
	}
	defer f.Close()

	// Pre-filter comment lines into a strings.Builder before CSV parsing.
	var sb strings.Builder
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(strings.TrimSpace(line), "#") {
			continue
		}
		sb.WriteString(line)
		sb.WriteByte('\n')
	}
	if err := scanner.Err(); err != nil {
		t.Fatalf("ReadCSVFile: scan %q: %v", path, err)
	}

	r := csv.NewReader(strings.NewReader(sb.String()))
	r.Comma = ';'
	r.TrimLeadingSpace = true

	records, err := r.ReadAll()
	if err != nil {
		t.Fatalf("ReadCSVFile: parse %q: %v", path, err)
	}
	return records
}

// CompareCSVExact compares actual vs expected CSV files cell-by-cell.
func CompareCSVExact(t *testing.T, actualPath, expectedPath string) {
	t.Helper()
	actual := ReadCSVFile(t, actualPath)
	expected := ReadCSVFile(t, expectedPath)

	if len(actual) != len(expected) {
		t.Fatalf("CompareCSVExact: row count actual=%d expected=%d", len(actual), len(expected))
	}
	for i, row := range actual {
		if len(row) != len(expected[i]) {
			t.Fatalf("CompareCSVExact: row %d field count actual=%d expected=%d", i, len(row), len(expected[i]))
		}
		for j, cell := range row {
			if cell != expected[i][j] {
				t.Errorf("CompareCSVExact: row %d col %d: got %q want %q", i, j, cell, expected[i][j])
			}
		}
	}
}

// CompareCSVFuzzy checks row count and header fields of an actual CSV.
func CompareCSVFuzzy(t *testing.T, actualPath string, expectedRowCount int, expectedHeaders []string) {
	t.Helper()
	rows := ReadCSVFile(t, actualPath)
	if len(rows) != expectedRowCount {
		t.Fatalf("CompareCSVFuzzy: row count got=%d want=%d", len(rows), expectedRowCount)
	}
	if len(expectedHeaders) == 0 || len(rows) == 0 {
		return
	}
	header := rows[0]
	for i, h := range expectedHeaders {
		if i >= len(header) {
			t.Errorf("CompareCSVFuzzy: header col %d missing (want %q)", i, h)
			continue
		}
		if header[i] != h {
			t.Errorf("CompareCSVFuzzy: header col %d got %q want %q", i, header[i], h)
		}
	}
}

// ConfidenceInRange returns true if v is in [0.0, 1.0].
func ConfidenceInRange(v float64) bool {
	return v >= 0.0 && v <= 1.0
}
