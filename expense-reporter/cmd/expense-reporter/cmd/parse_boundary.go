package cmd

// Command-layer helpers sitting on top of the internal/parse boundary. The
// boundary owns parsing and knows nothing about config or terminal output; this
// file is where those command concerns meet a ParsedExpense.

import (
	"errors"
	"fmt"
	"os"
	"time"

	"expense-reporter/internal/config"
	"expense-reporter/internal/parse"
)

// describeParseFailure adds what the command layer knows and the boundary cannot:
// the accepted spellings of a value, which parse's own message never mentions.
//
// It only ever ADDS to the boundary's reason, never replaces it. An earlier version
// substituted a fixed "expected DD/MM or DD/MM/YYYY" hint for every date rejection,
// and so told a user entering 15/04/2027 that their FORMAT was wrong when the real
// objection was the future year (T-47): the field was named correctly and the reason
// was destroyed. That is the same defect the sentinels' double-%w wrapping exists to
// prevent inside parse, reintroduced one layer up — so the rule here is the same,
// the cause always survives.
//
// Hence the apparent asymmetry, which is deliberate: parse's date messages already
// state the accepted format (and, for a rejected year, say so instead), so a date
// failure needs nothing added. Its value messages never mention installment syntax,
// so that hint is genuinely the command's to supply.
//
// The rejected token is not repeated here — every value rejection already quotes it
// — so this appends the accepted forms and nothing else.
func describeParseFailure(err error) error {
	if errors.Is(err, parse.ErrInvalidValue) {
		return fmt.Errorf("%w — accepted: 35.50, 35,50, or 35,50/3 for three installments", err)
	}
	return err
}

// parseOptions assembles the year-precedence ladder's inputs from a command's
// --year flag and the loaded config, so the ladder is wired in ONE place instead
// of at each command's call site. The boundary package cannot do this itself:
// internal/parse must not import config, which is exactly why this helper is
// command-layer.
//
// It takes no seam for per-command divergence because there is none — add,
// correct and auto all want the identical ladder, and a parameter that only one
// caller would ever pass is speculative generality rather than a modelled
// difference.
//
// Options.Now is deliberately left at its zero value, which parse reads as
// time.Now(). Every command wants the real clock; only tests inject one, and
// they build Options directly rather than going through here.
func parseOptions(yearFlag int, cfg *config.Config) parse.Options {
	return parse.Options{Year: yearFlag, ConfigYear: cfg.DateYear}
}

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
