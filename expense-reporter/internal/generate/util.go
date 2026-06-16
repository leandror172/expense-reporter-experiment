package generate

import (
	"fmt"
	"strconv"
	"strings"
)

// lower lowercases a sheet name for label use ("Fixas" -> "fixas").
func lower(s string) string { return strings.ToLower(s) }

// atoi parses a base-10 int, returning 0 on error (positions are always valid here).
func atoi(s string) int {
	n, _ := strconv.Atoi(s)
	return n
}

// sumList joins cell refs into an Excel SUM of discrete terms: SUM(a,b,c).
func sumList(terms []string) string {
	return "SUM(" + strings.Join(terms, ",") + ")"
}

// sumRange returns SUM(first:last), collapsing to SUM(first) when first==last (golden style).
func sumRange(first, last string) string {
	if first == last {
		return "SUM(" + first + ")"
	}
	return "SUM(" + first + ":" + last + ")"
}

// cell formats a col+row pair into an Excel cell address (e.g. "C3").
func cell(col string, row int) string { return fmt.Sprintf("%s%d", col, row) }

// sheetRef formats a cross-sheet reference, quoting the sheet name when required.
// Plain (unquoted) names are emitted when the name is purely ASCII alphanumeric.
func sheetRef(name, col string, row int) string {
	if needsQuote(name) {
		return fmt.Sprintf("'%s'!%s%d", name, col, row)
	}
	return fmt.Sprintf("%s!%s%d", name, col, row)
}

// needsQuote returns true when an Excel sheet name must be quoted in a formula reference.
// Names are unquoted only when every character is ASCII alphanumeric.
func needsQuote(s string) bool {
	for _, r := range s {
		isASCIIAlnum := (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9')
		if !isASCIIAlnum {
			return true
		}
	}
	return false
}

func sumCellRange(col string, fromRow, toRow int) string {
	return fmt.Sprintf("SUM(%s:%s)", cell(col, fromRow), cell(col, toRow))
}

func safeRatioFormula(col string, denomRow, numeratorRow int) string {
	return fmt.Sprintf("IF(%s>0,%s/%s,0)", cell(col, denomRow), cell(col, numeratorRow), cell(col, denomRow))
}
