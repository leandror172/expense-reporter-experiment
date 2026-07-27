package parse

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"expense-reporter/pkg/utils"
)

// Options controls how date strings are resolved when they lack year information.
type Options struct {
	// Year is the explicit year from command-line flag --year.
	Year int
	// ConfigYear is the year from configuration file date_year setting.
	ConfigYear int
	// Now is the clock used for resolving bare dates to years.
	// Zero value means time.Now().
	Now time.Time
}

// ParsedExpense represents a parsed expense with all fields resolved and validated.
type ParsedExpense struct {
	Item         string
	Date         time.Time
	Value        float64
	Installments int
	RawValue     string
}

// DateString returns the canonical DD/MM/YYYY representation of the expense date.
func (pe ParsedExpense) DateString() string {
	return utils.FormatDate(pe.Date)
}

// Fields parses the three field-wise inputs into a ParsedExpense. Inputs are
// trimmed; an empty item or date is an error. The value token accepts BR
// formats including thousands separators ("1.234,56") and installment
// notation ("total/N", yielding the per-installment value); the date resolves
// under the year-precedence ladder in Options.
func Fields(item, dateStr, valueStr string, opts Options) (ParsedExpense, error) {
	item = strings.TrimSpace(item)
	dateStr = strings.TrimSpace(dateStr)
	valueStr = strings.TrimSpace(valueStr)

	if item == "" {
		return ParsedExpense{}, errors.New("item cannot be empty")
	}
	if dateStr == "" {
		return ParsedExpense{}, errors.New("date string cannot be empty")
	}

	date, err := resolveDate(dateStr, opts)
	if err != nil {
		return ParsedExpense{}, err
	}
	if err := validateRenderableYear(date.Year()); err != nil {
		return ParsedExpense{}, err
	}
	if err := validateYearNotBeyondCurrent(date, opts); err != nil {
		return ParsedExpense{}, err
	}

	value, installments, err := utils.ParseCurrencyWithInstallments(normalizeThousands(valueStr))
	if err != nil {
		return ParsedExpense{}, err
	}

	return ParsedExpense{
		Item:         item,
		Date:         date,
		Value:        value,
		Installments: installments,
		RawValue:     valueStr,
	}, nil
}

// ExpenseString parses the 4-field semicolon CLI form
// "item;DD/MM[/YYYY];value[/N];subcategory", returning the subcategory
// alongside the parsed expense (classification is not parsing). It owns the
// split and the subcategory; field validation is Fields' job.
func ExpenseString(s string, opts Options) (ParsedExpense, string, error) {
	parts := strings.SplitN(s, ";", 4)
	if len(parts) != 4 {
		return ParsedExpense{}, "", errors.New("expense string must have exactly 4 semicolon-separated parts")
	}

	subcategory := strings.TrimSpace(parts[3])
	if subcategory == "" {
		return ParsedExpense{}, "", errors.New("subcategory cannot be empty")
	}

	expense, err := Fields(parts[0], parts[1], parts[2], opts)
	if err != nil {
		return ParsedExpense{}, "", err
	}

	return expense, subcategory, nil
}

// normalizeThousands strips BR thousands separators from a value token: a token
// carrying both '.' and ',' uses dots as separators ("1.234,56" → "1234,56").
// Dot-only tokens keep their legacy decimal-dot meaning ("35.50" stays 35.50).
func normalizeThousands(valueStr string) string {
	if strings.Contains(valueStr, ".") && strings.Contains(valueStr, ",") {
		return strings.ReplaceAll(valueStr, ".", "")
	}
	return valueStr
}

// resolveDate resolves a date string to a time.Time under the year-precedence
// ladder: an explicit year in the string wins; otherwise Options.Year, then
// Options.ConfigYear, then the most-recent-non-future rule.
func resolveDate(dateStr string, opts Options) (time.Time, error) {
	if strings.Count(dateStr, "/") == 2 {
		return utils.ParseDateFlexible(dateStr)
	}
	if opts.Year != 0 {
		return utils.ParseDateWithYear(dateStr, opts.Year)
	}
	if opts.ConfigYear != 0 {
		return utils.ParseDateWithYear(dateStr, opts.ConfigYear)
	}
	return mostRecentNonFuture(dateStr, opts)
}

// nowFromOptions resolves the clock every year rule reads, so "now" has exactly
// one definition in this package: the injected Options.Now when set, else the
// real clock. Tests inject it to stay independent of the calendar year they run in.
func nowFromOptions(opts Options) time.Time {
	if !opts.Now.IsZero() {
		return opts.Now
	}
	return time.Now()
}

// validateYearNotBeyondCurrent rejects an ENTERED date landing in a later year
// than the current one. The rule is deliberately year-scale, not day-scale: a
// date later in the current year is accepted, because the failure worth blocking
// is a mistyped YEAR. A wrong year is silent and expensive — the row hashes and
// writes cleanly to both logs, then `generate-workbook --year N` simply does not
// route it, so it vanishes from the workbook with no error anywhere.
//
// This guards only what a human types. Installment expansion builds its later
// dates downstream from an already-parsed time.Time and never re-enters this
// boundary, so a 24x purchase still writes rows years ahead — deliberately, since
// those payments are consequences of a purchase that already happened.
//
// Provisional (T-47): kept separate from validateRenderableYear, which is the
// permanent DD/MM/YYYY contract, so this policy can be relaxed on its own.
func validateYearNotBeyondCurrent(date time.Time, opts Options) error {
	currentYear := nowFromOptions(opts).Year()
	if date.Year() > currentYear {
		return fmt.Errorf("date %s is in a future year: entering an expense dated beyond the current year (%d) is not allowed", utils.FormatDate(date), currentYear)
	}
	return nil
}

// validateRenderableYear rejects a year DateString cannot render as DD/MM/YYYY.
// The DD/MM/YYYY form is not cosmetic: DateString is the sole producer of the
// bytes hashed into the join id shared by classifications.jsonl and
// expenses_log.jsonl, so a year outside 1..9999 would put a malformed year
// field ("15/04/-005", "15/04/12345") into that key. Checking the RESOLVED year
// covers every rung of the ladder at once — a year typed into the date string,
// --year, and config date_year are all guarded by this one check, and the
// clock rung cannot violate it. Note 0 is unreachable here: it is the "unset"
// sentinel for Year/ConfigYear, so those rungs fall through rather than
// resolving to year 0.
func validateRenderableYear(year int) error {
	if year < 1 || year > 9999 {
		return fmt.Errorf("year %d cannot be written as DD/MM/YYYY: expected a year between 1 and 9999", year)
	}
	return nil
}

// mostRecentNonFuture resolves a bare DD/MM date to the most recent year in
// which it is not in the future: now's year, or the previous year when the
// candidate lands strictly after now. The grace window is 0 by decision (T-41
// §4); same-day is never future because the candidate is midnight UTC. This is
// a deliberate behavior change from the old blind time.Now().Year().
func mostRecentNonFuture(dateStr string, opts Options) (time.Time, error) {
	now := nowFromOptions(opts)
	candidate, err := utils.ParseDateWithYear(dateStr, now.Year())
	if err != nil {
		return time.Time{}, err
	}
	if candidate.After(now) {
		return utils.ParseDateWithYear(dateStr, now.Year()-1)
	}
	return candidate, nil
}
