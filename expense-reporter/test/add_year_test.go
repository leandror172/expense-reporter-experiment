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

// TestAddDryRunJSON_YearFlagResolvesBareDate pins the precedence of --year flag
// when input date is bare (no year). The flag value should be used as the year.
func TestAddDryRunJSON_YearFlagResolvesBareDate(t *testing.T) {
	harness.Run(t, harness.Scenario{
		Name:  "add --dry-run --json with --year flag resolves bare date to that year",
		Given: classifierForJSON(),
		When:  actions.RunAddDryRun("Uber Centro;15/04;35,50;Uber/Taxi", "--json", "--year", "2024"),
		Then: slices.Concat(
			thenJSONSucceeded(),
			thenJSONDateIs("15/04/2024"),
		),
	})
}

// TestAddDryRunJSON_ExplicitYearBeatsYearFlag pins the precedence of explicit year
// in input string over --year flag. The input's year should always win.
func TestAddDryRunJSON_ExplicitYearBeatsYearFlag(t *testing.T) {
	harness.Run(t, harness.Scenario{
		Name:  "add --dry-run --json with explicit year in input beats --year flag",
		Given: classifierForJSON(),
		When:  actions.RunAddDryRun("Uber Centro;15/04/2023;35,50;Uber/Taxi", "--json", "--year", "2024"),
		Then: slices.Concat(
			thenJSONSucceeded(),
			thenJSONDateIs("15/04/2023"),
		),
	})
}

// TestAddDryRunJSON_ConfigDateYearResolvesBareDate pins the precedence of config date_year
// as fallback when neither input string nor --year provides a year.
func TestAddDryRunJSON_ConfigDateYearResolvesBareDate(t *testing.T) {
	harness.Run(t, harness.Scenario{
		Name:  "add --dry-run --json with config date_year resolves bare date to that year",
		Given: binaryWithDateYearConfig(2024),
		When:  actions.RunAddDryRun("Uber Centro;15/04;35,50;Uber/Taxi", "--json"),
		Then: slices.Concat(
			thenJSONSucceeded(),
			thenJSONDateIs("15/04/2024"),
		),
	})
}

// TestAddDryRunJSON_YearFlagBeatsConfigDateYear pins the precedence of --year flag
// over config date_year. The flag should always override the config.
func TestAddDryRunJSON_YearFlagBeatsConfigDateYear(t *testing.T) {
	harness.Run(t, harness.Scenario{
		Name:  "add --dry-run --json with --year flag beats config date_year",
		Given: binaryWithDateYearConfig(2022),
		When:  actions.RunAddDryRun("Uber Centro;15/04;35,50;Uber/Taxi", "--json", "--year", "2024"),
		Then: slices.Concat(
			thenJSONSucceeded(),
			thenJSONDateIs("15/04/2024"),
		),
	})
}

// thenJSONDateIs returns a verification that the JSON output has the given date.
func thenJSONDateIs(date string) []func(*harness.Context) {
	return []func(*harness.Context){
		expect.OutputJSONHasDate(date),
	}
}

// binaryWithDateYearConfig sets up context with config including date_year.
func binaryWithDateYearConfig(dateYear int) func(*harness.Context) {
	return func(ctx *harness.Context) {
		ctx.BinaryPath = binaryPath
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
