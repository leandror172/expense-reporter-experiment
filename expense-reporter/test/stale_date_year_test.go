//go:build acceptance

package acceptance_test

import (
	"slices"
	"testing"
	"time"

	"expense-reporter/test/actions"
	"expense-reporter/test/expect"
	"github.com/leandror172/acceptance-harness/harness"
)

// TestStaleDateYear_WarnsWhenConfiguredYearDatedTheEntry covers the hazard the
// warning exists for: a date_year left over from a past year silently back-dates
// entries, and a wrong year is invisible downstream — the row hashes and writes
// cleanly, then generate-workbook for the real year simply never routes it.
func TestStaleDateYear_WarnsWhenConfiguredYearDatedTheEntry(t *testing.T) {
	harness.Run(t, harness.Scenario{
		Name:  "add warns when a configured year below the current one dated the entry",
		Given: defaultYearConfiguredAs(previousYear()),
		When:  actions.RunAddDryRun("Uber Centro;15/04;35,50;Uber/Taxi", "--json"),
		Then: slices.Concat(
			thenJSONSucceeded(),
			staleConfiguredYearWarned(),
		),
	})
}

// TestStaleDateYear_SilentWhenConfiguredYearIsCurrent guards against warning on
// the normal case. A date_year naming the year actually being closed is the
// supported way to use the setting, and a warning that fires there would train
// the reader to ignore it.
func TestStaleDateYear_SilentWhenConfiguredYearIsCurrent(t *testing.T) {
	harness.Run(t, harness.Scenario{
		Name:  "add stays silent when the configured year is the current year",
		Given: defaultYearConfiguredAs(currentYear()),
		When:  actions.RunAddDryRun("Uber Centro;15/04;35,50;Uber/Taxi", "--json"),
		Then: slices.Concat(
			thenJSONSucceeded(),
			noStaleConfiguredYearWarning(),
		),
	})
}

// TestStaleDateYear_SilentWhenOutrankedByExplicitYear is the scenario that
// separates a correct implementation from a naive one. Comparing date_year to the
// clock alone would warn here too — but the input carried its own year, so the
// configured year dated nothing and there is no hazard to report.
func TestStaleDateYear_SilentWhenOutrankedByExplicitYear(t *testing.T) {
	harness.Run(t, harness.Scenario{
		Name:  "add stays silent when an explicit year in the input outranked the configured year",
		Given: defaultYearConfiguredAs(previousYear()),
		When:  actions.RunAddDryRun("Uber Centro;15/04/2023;35,50;Uber/Taxi", "--json"),
		Then: slices.Concat(
			thenJSONSucceeded(),
			noStaleConfiguredYearWarning(),
		),
	})
}

// TestStaleDateYear_WarnsOnCorrectDespiteFailure proves the warning is wired into
// correct as well as add, and covers the case where a stale year is most baffling:
// the wrong year is precisely what makes the lookup miss, so without the warning
// the user sees only "no prior classification found" for an expense they did log.
func TestStaleDateYear_WarnsOnCorrectDespiteFailure(t *testing.T) {
	harness.Run(t, harness.Scenario{
		Name:  "correct warns about the stale configured year even though the lookup fails",
		Given: defaultYearConfiguredAs(previousYear()),
		When:  actions.RunCorrect("Uber Centro;15/04;35,50;Combustível"),
		Then: slices.Concat(
			commandFailed(),
			staleConfiguredYearWarned(),
		),
	})
}

// --- Clock-relative years -------------------------------------------------------
// The warning's predicate is relative to "now", so pinning a literal year would
// make these scenarios mean something different every January.

func currentYear() int {
	return time.Now().Year()
}

func previousYear() int {
	return currentYear() - 1
}

// --- Then helpers ---------------------------------------------------------------

// staleConfiguredYearWarned asserts the user was told the configured year is out
// of date and dated this entry.
func staleConfiguredYearWarned() []func(*harness.Context) {
	return []func(*harness.Context){
		expect.StaleConfiguredYearWarned(),
	}
}

// noStaleConfiguredYearWarning asserts nothing was said about the configured year.
func noStaleConfiguredYearWarning() []func(*harness.Context) {
	return []func(*harness.Context){
		expect.NoStaleConfiguredYearWarning(),
	}
}
