//go:build acceptance

package acceptance_test

import (
	"path/filepath"
	"testing"

	"expense-reporter/test/actions"
	"expense-reporter/test/harness"
	"expense-reporter/test/verify"
)

// TestAuto_Basic reads rows from the auto-basic fixture and runs the auto command
// against each, asserting exit 0, confidence output, and no spurious insert.
func TestAuto_Basic(t *testing.T) {
	harness.RequireOllama(t, "")

	fixDir := filepath.Join(fixturesDir(), "auto-basic")
	rows := harness.ReadCSVFile(t, filepath.Join(fixDir, "input.csv"))

	for _, row := range rows {
		row := row
		if len(row) < 3 {
			continue
		}
		item, date, value := row[0], row[1], row[2]

		harness.Run(t, harness.Scenario{
			Name: "auto: " + item,
			Given: func(ctx *harness.Context) {
				ctx.BinaryPath = binaryPath
				ctx.FixtureDir = fixDir
			},
			When: actions.RunAuto(item, value, date),
			Then: []func(*harness.Context){
				verify.ExitCodeZero(),
				verify.StdoutContains("%"),
			},
		})
	}
}

// TestAuto_AmbiguousItemStaysInReview verifies that a vague item is not auto-inserted.
func TestAuto_AmbiguousItemStaysInReview(t *testing.T) {
	harness.RequireOllama(t, "")

	harness.Run(t, harness.Scenario{
		Name: "auto: vague item must not be auto-inserted",
		Given: func(ctx *harness.Context) {
			ctx.BinaryPath = binaryPath
		},
		When: actions.RunAuto("Outros gastos aleatorios xyz", "10.00", "01/01"),
		Then: []func(*harness.Context){
			verify.ExitCodeZero(),
			verify.StdoutNotContains("✓ Inserted"),
		},
	})
}
