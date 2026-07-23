//go:build acceptance

package acceptance_test

import (
	"path/filepath"
	"slices"
	"testing"

	"expense-reporter/test/actions"
	"github.com/leandror172/acceptance-harness/harness"
)

// TestCorrect_YearFlagResolvesBareDateToPriorEntry pins the join property behind
// --year: correct looks up the prior entry by GenerateID(item, date, value), so a
// bare 15/04 with --year 2024 must canonicalize to 15/04/2024 and hash to the very
// id the seed carries — without the flag, the fallback rungs would resolve to a
// different year and the lookup would miss.
func TestCorrect_YearFlagResolvesBareDateToPriorEntry(t *testing.T) {
	fixDir := filepath.Join(fixturesDir(), "correct-year-flag")

	harness.Run(t, harness.Scenario{
		Name:    "correct --year joins a bare date to the prior year's entry",
		Fixture: fixDir,
		Given:   expenseAutoConfirmedInAPriorYear(),
		When:    actions.RunCorrect("Uber Centro;15/04;35,50;Combustível", "--year", "2024"),
		Then: slices.Concat(
			commandSucceeded(),
			correctionJoinedThePriorYearEntry(fixDir),
		),
	})
}

// expenseAutoConfirmedInAPriorYear returns the same auto-confirmed event as
// expenseAutoConfirmed, distinguished because the seed is dated in a PRIOR year (2024)
// while the correction arrives with a bare date.
func expenseAutoConfirmedInAPriorYear() func(*harness.Context) {
	return expenseAutoConfirmed()
}

// correctionJoinedThePriorYearEntry asserts the corrected entry was appended with
// the 2024 canonical date, proving the id join hit the seeded prior-year entry.
func correctionJoinedThePriorYearEntry(fixDir string) []func(*harness.Context) {
	return classificationsMatchExpected(fixDir)
}
