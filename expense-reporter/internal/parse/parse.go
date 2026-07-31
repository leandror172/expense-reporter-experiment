package parse

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"expense-reporter/pkg/utils"
)

// YearSource names the rung of the year-precedence ladder that supplied a
// parsed expense's year. Callers use it to tell a year the user stated from one
// the boundary inferred — the two are indistinguishable in the resulting date.
type YearSource int

const (
	// YearSourceUnknown means "not produced by a successful parse". Every error
	// path returns ParsedExpense{}, so if the zero value named a real rung a
	// failed parse would report a plausible source to code that reads this field
	// to make a decision. Same reasoning as the zero Date formatting as
	// 01/01/0001 — wrong loudly rather than wrong quietly.
	YearSourceUnknown YearSource = iota

	// YearFromDateString: the year was written into the date string itself.
	YearFromDateString

	// YearFromFlag: the year came from Options.Year (the --year flag).
	YearFromFlag

	// YearFromConfig: the year came from Options.ConfigYear (config date_year).
	YearFromConfig

	// YearFromClock: no year was supplied, so the most-recent-non-future rule
	// resolved it against the clock.
	YearFromClock
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
	YearSource   YearSource
}

// DateString returns the canonical DD/MM/YYYY representation of the expense date.
func (pe ParsedExpense) DateString() string {
	return utils.FormatDate(pe.Date)
}

// Sentinel errors naming WHICH field Fields rejected. They exist so a caller can
// branch on the failure without reading the message, which matters for two
// consumers this package must not know about: a command re-attaching its own
// per-field hint (auto knows which argument held the bad token; parse does not),
// and — per the vision's Phase 1 and 4 — an error log that records the failing
// phase, plus a chat layer that has to explain the failure in Portuguese. An
// English message is one presentation of a parse failure; it must not be the only
// representation of it.
//
// Every wrap keeps the underlying cause in the chain, so the specific reason
// ("...year 12345 is not renderable...") still reaches whoever prints it. The
// generic sentinel text is a prefix, never a replacement.
//
// There is deliberately no sentinel for an empty item: no caller distinguishes
// that case, and an unused one would be API nobody reads.
var (
	ErrInvalidDate  = errors.New("invalid date")
	ErrInvalidValue = errors.New("invalid value")
)

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
	date, yearSource, err := Date(dateStr, opts)
	if err != nil {
		return ParsedExpense{}, err
	}

	value, installments, err := utils.ParseCurrencyWithInstallments(normalizeThousands(valueStr))
	if err != nil {
		return ParsedExpense{}, fmt.Errorf("%w: %w", ErrInvalidValue, err)
	}

	return ParsedExpense{
		Item:         item,
		Date:         date,
		Value:        value,
		Installments: installments,
		RawValue:     valueStr,
		YearSource:   yearSource,
	}, nil
}

// Date resolves a date string under the year-precedence ladder and validates the
// resolved year. It is the boundary's date-only entry point.
//
// It exists because not every consumer arrives holding three strings. `apply` reads
// `reviewed.json`, whose item and value are already typed by the JSON decoder and
// whose date alone is still text — so Fields would have forced it to re-serialize a
// float purely to have the boundary parse it back. A caller with one unresolved field
// should not have to fake the other two.
//
// Fields delegates here rather than duplicating the sequence, so there is exactly one
// ladder and one pair of year validations no matter which door a caller comes through.
// That matters more than the shared lines: the two validations are what stand between a
// mistyped year and a row that hashes cleanly into both logs and then vanishes from the
// workbook.
//
// Every failure returns YearSourceUnknown, for the same reason ParsedExpense{} does —
// a zero that named a real rung would let a failed resolution report a plausible source
// to code that reads it to decide.
func Date(dateStr string, opts Options) (time.Time, YearSource, error) {
	dateStr = strings.TrimSpace(dateStr)
	if dateStr == "" {
		return time.Time{}, YearSourceUnknown, fmt.Errorf("%w: date string cannot be empty", ErrInvalidDate)
	}

	date, yearSource, err := resolveDate(dateStr, opts)
	if err != nil {
		return time.Time{}, YearSourceUnknown, fmt.Errorf("%w: %w", ErrInvalidDate, err)
	}
	if err := validateRenderableYear(date.Year()); err != nil {
		return time.Time{}, YearSourceUnknown, fmt.Errorf("%w: %w", ErrInvalidDate, err)
	}
	if err := validateYearNotBeyondCurrent(date, opts); err != nil {
		return time.Time{}, YearSourceUnknown, fmt.Errorf("%w: %w", ErrInvalidDate, err)
	}

	return date, yearSource, nil
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
// Options.ConfigYear, then the most-recent-non-future rule. It reports which
// rung fired alongside the date, because this is the only place that knows —
// the resolved date carries no trace of where its year came from, and a caller
// re-deriving it would be a second copy of the ladder, free to drift from this one.
func resolveDate(dateStr string, opts Options) (time.Time, YearSource, error) {
	if strings.Count(dateStr, "/") == 2 {
		date, err := utils.ParseDateFlexible(dateStr)
		return date, YearFromDateString, err
	}
	if opts.Year != 0 {
		date, err := utils.ParseDateWithYear(dateStr, opts.Year)
		return date, YearFromFlag, err
	}
	if opts.ConfigYear != 0 {
		date, err := utils.ParseDateWithYear(dateStr, opts.ConfigYear)
		return date, YearFromConfig, err
	}
	date, err := mostRecentNonFuture(dateStr, opts)
	return date, YearFromClock, err
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
