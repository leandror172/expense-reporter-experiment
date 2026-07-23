//go:build acceptance

package acceptance_test

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"expense-reporter/test/actions"
	"expense-reporter/test/domain"
	"expense-reporter/test/expect"
	"github.com/leandror172/acceptance-harness/harness"
)

// TestAdd_BareDateResolvesToFlagYear pins the year-precedence ladder: a bare
// date (no year) takes its year from the --year flag.
func TestAdd_BareDateResolvesToFlagYear(t *testing.T) {
	harness.Run(t, harness.Scenario{
		Name:    "add --dry-run --json with --year flag resolves bare date to that year",
		Fixture: jsonOutputFixture(),
		Given:   taxonomyAuthoredWithTrainingData(),
		When:    actions.RunAddDryRun("Uber Centro;15/04;35,50;Uber/Taxi", "--json", "--year", "2024"),
		Then: slices.Concat(
			thenJSONSucceeded(),
			flagYearResolvedBareDateTo("15/04/2024"),
		),
	})
}

// TestAdd_ExplicitYearWinsOverFlag pins the top rung of the ladder: a year
// written in the input itself always beats the --year flag.
func TestAdd_ExplicitYearWinsOverFlag(t *testing.T) {
	harness.Run(t, harness.Scenario{
		Name:    "add --dry-run --json with explicit year in input beats --year flag",
		Fixture: jsonOutputFixture(),
		Given:   taxonomyAuthoredWithTrainingData(),
		When:    actions.RunAddDryRun("Uber Centro;15/04/2023;35,50;Uber/Taxi", "--json", "--year", "2024"),
		Then: slices.Concat(
			thenJSONSucceeded(),
			explicitYearKeptDateAs("15/04/2023"),
		),
	})
}

// TestAdd_BareDateFallsBackToConfiguredYear pins the fallback rung: with no
// explicit year and no flag, the configured default year dates the entry.
func TestAdd_BareDateFallsBackToConfiguredYear(t *testing.T) {
	harness.Run(t, harness.Scenario{
		Name:  "add --dry-run --json with config date_year resolves bare date to that year",
		Given: defaultYearConfiguredAs(2024),
		When:  actions.RunAddDryRun("Uber Centro;15/04;35,50;Uber/Taxi", "--json"),
		Then: slices.Concat(
			thenJSONSucceeded(),
			configuredYearResolvedBareDateTo("15/04/2024"),
		),
	})
}

// TestAdd_FlagYearOverridesConfiguredYear pins the ordering between the two
// middle rungs: the --year flag beats the configured default year.
func TestAdd_FlagYearOverridesConfiguredYear(t *testing.T) {
	harness.Run(t, harness.Scenario{
		Name:  "add --dry-run --json with --year flag beats config date_year",
		Given: defaultYearConfiguredAs(2022),
		When:  actions.RunAddDryRun("Uber Centro;15/04;35,50;Uber/Taxi", "--json", "--year", "2024"),
		Then: slices.Concat(
			thenJSONSucceeded(),
			flagYearBeatConfiguredYearResolvingTo("15/04/2024"),
		),
	})
}

// flagYearResolvedBareDateTo asserts the bare input date came out dated with
// the --year flag's year.
func flagYearResolvedBareDateTo(date string) []func(*harness.Context) {
	return []func(*harness.Context){
		expect.OutputJSONHasDate(date),
	}
}

// explicitYearKeptDateAs asserts the input's own year survived untouched
// despite a competing --year flag.
func explicitYearKeptDateAs(date string) []func(*harness.Context) {
	return []func(*harness.Context){
		expect.OutputJSONHasDate(date),
	}
}

// configuredYearResolvedBareDateTo asserts the configured default year dated
// the bare input date.
func configuredYearResolvedBareDateTo(date string) []func(*harness.Context) {
	return []func(*harness.Context){
		expect.OutputJSONHasDate(date),
	}
}

// flagYearBeatConfiguredYearResolvingTo asserts the --year flag won over the
// configured default year in dating the bare input date.
func flagYearBeatConfiguredYearResolvingTo(date string) []func(*harness.Context) {
	return []func(*harness.Context){
		expect.OutputJSONHasDate(date),
	}
}

// defaultYearConfiguredAs sets up the binary with a config whose date_year is
// the given year — the "a default year was configured" event.
func defaultYearConfiguredAs(dateYear int) func(*harness.Context) {
	return func(ctx *harness.Context) {
		taxonomyDest := filepath.Join(ctx.WorkDir, "taxonomy.json")
		fixtureTaxPath := filepath.Join(fixturesDir(), "json-output", "fixture-taxonomy.json")

		taxData, err := os.ReadFile(fixtureTaxPath)
		if err != nil {
			ctx.T.Fatalf("reading fixture taxonomy: %v", err)
		}
		if err := os.WriteFile(taxonomyDest, taxData, 0o644); err != nil {
			ctx.T.Fatalf("writing taxonomy to workdir: %v", err)
		}

		classificationsPath := filepath.Join(ctx.WorkDir, "classifications.jsonl")
		expensesLogPath := filepath.Join(ctx.WorkDir, "expenses_log.jsonl")

		if err := domain.SetupBinaryConfig(ctx, map[string]interface{}{
			"classifications_path": classificationsPath,
			"expenses_log_path":    expensesLogPath,
			"taxonomy_path":        taxonomyDest,
			"date_year":            dateYear,
		}); err != nil {
			ctx.T.Fatalf("SetupBinaryConfig: %v", err)
		}
		ctx.Artifacts["classifications.jsonl"] = classificationsPath
		ctx.Artifacts["expenses_log.jsonl"] = expensesLogPath
	}
}
