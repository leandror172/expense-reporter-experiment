//go:build acceptance

package acceptance_test

import (
	"path/filepath"
	"testing"

	"expense-reporter/test/actions"
	"expense-reporter/test/harness"
	"expense-reporter/test/verify"
)

// TestClassify_Basic reads each row from the classify-basic fixture as a scenario
// and runs the classify command against the real binary + Ollama.
func TestClassify_Basic(t *testing.T) {
	harness.RequireOllama(t, "")

	fixDir := filepath.Join(fixturesDir(), "classify-basic")

	// Load the fixture's input.csv as a scenario table.
	// Each non-comment row: item;DD/MM;value
	rows := harness.ReadCSVFile(t, filepath.Join(fixDir, "input.csv"))

	for _, row := range rows {
		row := row // capture
		if len(row) < 3 {
			t.Logf("classify-basic: skipping malformed row %v", row)
			continue
		}
		item, date, value := row[0], row[1], row[2]

		harness.Run(t, harness.Scenario{
			Name: "classify: " + item,
			Given: func(ctx *harness.Context) {
				ctx.BinaryPath = binaryPath
				ctx.FixtureDir = fixDir
			},
			When: actions.RunClassify(item, value, date),
			Then: []func(*harness.Context){
				verify.ExitCodeZero(),
				verify.OutputContains("%"),
			},
		})
	}
}
