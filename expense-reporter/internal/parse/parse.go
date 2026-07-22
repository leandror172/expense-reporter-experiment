package parse

import (
	"errors"
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

// Fields parses item, date, and value strings into a ParsedExpense.
// It trims all inputs first. Empty item or empty dateStr → error.
// Value parsing uses utils.ParseCurrencyWithInstallments.
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

// ExpenseString parses a semicolon-separated string into a ParsedExpense and subcategory.
// The format is "item;DD/MM[/YYYY];value[/N];subcategory".
func ExpenseString(s string, opts Options) (ParsedExpense, string, error) {
	parts := strings.SplitN(s, ";", 4)
	if len(parts) != 4 {
		return ParsedExpense{}, "", errors.New("expense string must have exactly 4 semicolon-separated parts")
	}

	item := strings.TrimSpace(parts[0])
	dateStr := strings.TrimSpace(parts[1])
	valueStr := strings.TrimSpace(parts[2])
	subcategory := strings.TrimSpace(parts[3])

	if item == "" {
		return ParsedExpense{}, "", errors.New("item cannot be empty")
	}
	if subcategory == "" {
		return ParsedExpense{}, "", errors.New("subcategory cannot be empty")
	}

	expense, err := Fields(item, dateStr, valueStr, opts)
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
	return mostRecentNonFuture(dateStr, opts.Now)
}

// mostRecentNonFuture resolves a bare DD/MM date to the most recent year in
// which it is not in the future: now's year, or the previous year when the
// candidate lands strictly after now. The grace window is 0 by decision (T-41
// §4); same-day is never future because the candidate is midnight UTC. This is
// a deliberate behavior change from the old blind time.Now().Year().
func mostRecentNonFuture(dateStr string, now time.Time) (time.Time, error) {
	if now.IsZero() {
		now = time.Now()
	}
	candidate, err := utils.ParseDateWithYear(dateStr, now.Year())
	if err != nil {
		return time.Time{}, err
	}
	if candidate.After(now) {
		return utils.ParseDateWithYear(dateStr, now.Year()-1)
	}
	return candidate, nil
}
