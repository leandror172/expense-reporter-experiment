// workbook-inspect: dumps the full structural map of an expense workbook to JSON.
// Usage: workbook-inspect <workbook.xlsx> <output-dir>
// Produces <output-dir>/manifest.json plus one <SheetName>.json per sheet.
// Thin wrapper over internal/inspect.
package main

import (
	"fmt"
	"os"

	"expense-reporter/internal/inspect"
)

func main() {
	if len(os.Args) != 3 {
		fmt.Fprintln(os.Stderr, "usage: workbook-inspect <workbook.xlsx> <output-dir>")
		os.Exit(1)
	}

	workbookPath := os.Args[1]
	outputDir := os.Args[2]

	if _, err := os.Stat(workbookPath); os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "workbook does not exist: %s\n", workbookPath)
		os.Exit(1)
	}

	if err := inspect.DumpWorkbook(workbookPath, outputDir); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}
