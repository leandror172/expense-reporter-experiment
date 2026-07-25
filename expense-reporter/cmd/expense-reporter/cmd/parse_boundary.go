package cmd

// Command-layer helpers sitting on top of the internal/parse boundary. The
// boundary owns parsing and knows nothing about config or terminal output; this
// file is where those command concerns meet a ParsedExpense.

import (
	"fmt"
	"os"
	"time"

	"expense-reporter/internal/config"
	"expense-reporter/internal/parse"
)

// warnIfStaleConfiguredYear tells the user when a leftover config date_year is
// what dated this entry. A wrong year is the quietest failure in the system: the
// row hashes and writes cleanly to both logs, and then `generate-workbook --year N`
// simply does not route it, so it vanishes from the workbook with no error anywhere.
//
// Both conditions are required, and each one alone would make the warning useless:
//   - Warning whenever date_year is merely old would fire on every command
//     throughout a legitimate backfill of an earlier year, training the reader to
//     ignore it.
//   - Warning whenever the config rung fires would fire when date_year names the
//     year actually being closed, which is the supported way to use the setting.
//
// A configured year that no input consulted is not a hazard at all, which is why
// this asks the parse boundary which rung ran rather than re-deriving the ladder.
//
// Writes to stderr, never stdout: --json mode puts its payload on stdout, so a
// warning on the wrong stream would corrupt machine-readable output rather than
// merely annoy. It returns nothing — a warning must not change control flow.
func warnIfStaleConfiguredYear(pe parse.ParsedExpense, appCfg *config.Config) {
	if pe.YearSource != parse.YearFromConfig {
		return
	}
	currentYear := time.Now().Year()
	if appCfg.DateYear >= currentYear {
		return
	}

	fmt.Fprintf(os.Stderr,
		"⚠  config date_year=%d is before the current year (%d); it dated this entry %s — pass --year to override, or update config.json\n",
		appCfg.DateYear, currentYear, pe.DateString())
}
