package utils

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// maxInstallments is the largest installment count either notation accepts.
const maxInstallments = 60

// ParseCurrency parses a currency string in ##,## format and returns a float64
// Accepts both comma (,) and period (.) as decimal separator
func ParseCurrency(valueStr string) (float64, error) {
	if valueStr == "" {
		return 0, errors.New("value string cannot be empty")
	}

	// Trim spaces
	valueStr = strings.TrimSpace(valueStr)

	// Replace comma with period for standard float parsing
	// This handles both "50,00" and "50.00"
	normalized := strings.ReplaceAll(valueStr, ",", ".")

	value, err := strconv.ParseFloat(normalized, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid value format: %s", valueStr)
	}

	if value < 0 {
		return 0, errors.New("negative values are not allowed")
	}

	return value, nil
}

// ParseCurrencyWithInstallments parses PT-BR currency with optional installment syntax.
//
// There are TWO installment notations and they are INVERSES of one another:
//
//	"100,00"      → (100.00, 1, nil)   // regular value
//	"300,00/3"    → (100.00, 3, nil)   // the written number is the TOTAL, divided by N
//	"405,25 x4"   → (405.25, 4, nil)   // the written number is PER-INSTALLMENT, multiplied
//
// The same digits therefore mean things N times apart depending on the marker —
// "405,25/4" is a 405,25 purchase, "405,25 x4" is a 1.621,00 one. The divisor form is how
// a card statement prints it; the multiplier form is Brazilian retail usage ("4x de
// R$ 405,25"). Both return the PER-INSTALLMENT value, which is what every consumer
// downstream expects: the appender writes count rows of it.
//
// The token is expected to be already normalized — internal/parse strips BR thousands
// separators before calling here (design Q3: the boundary owns all input normalization).
func ParseCurrencyWithInstallments(s string) (perInstallment float64, count int, err error) {
	s = strings.TrimSpace(s)

	// Presence of the marker IS the mode switch. This is checked before "/" so that a
	// token carrying both markers is rejected as a malformed multiplier rather than
	// silently parsed as a divisor with a strange count.
	if strings.ContainsAny(s, "xX") {
		return parseMultiplierNotation(s)
	}
	if strings.Contains(s, "/") {
		return parseDivisorNotation(s)
	}

	value, err := ParseCurrency(s)
	if err != nil {
		return 0, 0, err
	}
	return value, 1, nil
}

// multiplierReading is one candidate interpretation of a token containing x/X.
type multiplierReading struct {
	value float64
	count int
}

// parseMultiplierNotation resolves a token containing the x/X marker.
//
// It does NOT decide what the token means by a grammar. It asks a different question:
// how many things COULD this mean? Each reading below is tried independently, and the
// token resolves only when exactly one of them holds. Two readings is an ambiguity, and
// an ambiguity is an error — never a guess.
//
// That asymmetry is deliberate and is the whole point of the notation work. Guessing
// wrong writes a budget row off by a factor of the installment count, and a wrong row is
// indistinguishable from a legitimate one once logged. Rejecting costs the user a single
// edit, because batch-auto sends the row back in failed.csv with its reason on the same
// line (T-63) to be repaired in place.
func parseMultiplierNotation(s string) (perInstallment float64, count int, err error) {
	if strings.Count(s, "x")+strings.Count(s, "X") != 1 {
		return 0, 0, fmt.Errorf("invalid installment format: %s", s)
	}

	marker := strings.IndexAny(s, "xX")
	left := strings.TrimSpace(s[:marker])
	right := strings.TrimSpace(s[marker+1:])

	shaped := shapedReadings(left, right)
	return resolveSingleReading(shaped, s)
}

// shapedReadings returns every reading whose SHAPE fits — a currency value paired with an
// integer — regardless of whether that integer is a plausible installment count.
//
// The range check is deliberately excluded here. A reading that fits the shape but carries
// an implausible count is what lets the caller report the specific objection ("must be
// positive") instead of degrading to a generic parse failure; collapsing the two makes
// that message unreachable.
func shapedReadings(left, right string) []multiplierReading {
	var shaped []multiplierReading
	for _, read := range []func(string, string) (multiplierReading, bool){readingA, readingB, readingC} {
		if reading, ok := read(left, right); ok {
			shaped = append(shaped, reading)
		}
	}
	return shaped
}

// resolveSingleReading applies the "exactly one" rule to the readings that fit the shape.
func resolveSingleReading(shaped []multiplierReading, token string) (float64, int, error) {
	var possible []multiplierReading
	for _, reading := range shaped {
		if plausibleCount(reading.count) {
			possible = append(possible, reading)
		}
	}

	switch {
	case len(possible) == 1:
		return possible[0].value, possible[0].count, nil
	case len(possible) > 1:
		return 0, 0, fmt.Errorf("ambiguous installment format: %s", token)
	case len(shaped) > 0:
		// The shape fits, so the objection is the count itself and can be named exactly.
		return 0, 0, countRangeError(shaped[0].count)
	default:
		return 0, 0, fmt.Errorf("invalid installment format: %s", token)
	}
}

// readingA interprets the split as "<value> x <count>", e.g. "405,25 x4".
func readingA(left, right string) (multiplierReading, bool) {
	value, err := ParseCurrency(left)
	if err != nil {
		return multiplierReading{}, false
	}
	count, err := strconv.Atoi(right)
	if err != nil {
		return multiplierReading{}, false
	}
	return multiplierReading{value: value, count: count}, true
}

// readingB interprets the split as "<value> <count> x", e.g. "646,25 4x".
//
// Applies only when the right side is EMPTY and the left side splits on whitespace into
// exactly a value and a count. The whitespace requirement is load bearing: without it
// "646,254x" carries nothing marking where the value ends, so "646,25"×4 and "646,2"×54
// are equally available and a greedier or lazier match would pick a different budget.
func readingB(left, right string) (multiplierReading, bool) {
	if right != "" {
		return multiplierReading{}, false
	}
	parts := strings.Fields(left)
	if len(parts) != 2 {
		return multiplierReading{}, false
	}
	value, err := ParseCurrency(parts[0])
	if err != nil {
		return multiplierReading{}, false
	}
	count, err := strconv.Atoi(parts[1])
	if err != nil {
		return multiplierReading{}, false
	}
	return multiplierReading{value: value, count: count}, true
}

// readingC interprets the split as "<count> x <value>", e.g. "4x405,25" — the order
// Brazilian retail writes. It costs nothing to support: where reading A also fits, the
// token was ambiguous anyway and is rejected either way.
func readingC(left, right string) (multiplierReading, bool) {
	count, err := strconv.Atoi(left)
	if err != nil {
		return multiplierReading{}, false
	}
	value, err := ParseCurrency(right)
	if err != nil {
		return multiplierReading{}, false
	}
	return multiplierReading{value: value, count: count}, true
}

// plausibleCount reports whether n is an installment count this parser accepts.
func plausibleCount(n int) bool {
	return n > 0 && n <= maxInstallments
}

// countRangeError returns the SAME error the divisor path returns for a count outside the
// accepted range, and nil when the count is fine. Reusing that wording is deliberate: a
// user who mistypes a count should read the same objection whichever notation they used.
func countRangeError(n int) error {
	if n <= 0 {
		return fmt.Errorf("installment count must be positive, got %d", n)
	}
	if n > maxInstallments {
		return fmt.Errorf("installment count too large: %d (max %d)", n, maxInstallments)
	}
	return nil
}

// parseDivisorNotation resolves the "total/N" form, where the written number is the TOTAL
// and is divided by N.
func parseDivisorNotation(s string) (perInstallment float64, count int, err error) {
	parts := strings.Split(s, "/")
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("invalid installment format: %s", s)
	}

	total, err := ParseCurrency(parts[0])
	if err != nil {
		return 0, 0, fmt.Errorf("invalid total value in installment: %w", err)
	}

	countStr := strings.TrimSpace(parts[1])
	count, err = strconv.Atoi(countStr)
	if err != nil {
		return 0, 0, fmt.Errorf("invalid installment count '%s': must be a number", countStr)
	}

	if err := countRangeError(count); err != nil {
		return 0, 0, err
	}

	return total / float64(count), count, nil
}
