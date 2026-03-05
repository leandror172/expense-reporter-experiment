//go:build acceptance

package acceptance_test

import (
	"path/filepath"
	"runtime"
	"testing"

	"expense-reporter/test/actions"
	"expense-reporter/test/harness"
	"expense-reporter/test/verify"
)

func fixturesDir() string {
	_, file, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(file), "fixtures")
}

func TestBatchAuto_Basic(t *testing.T) {
	harness.RequireOllama(t, "")

	fixDir := filepath.Join(fixturesDir(), "batch-auto-basic")

	harness.Run(t, harness.Scenario{
		Name: "batch-auto basic — 10 rows dry-run",
		Given: func(ctx *harness.Context) {
			ctx.BinaryPath = binaryPath
			ctx.FixtureDir = fixDir
			if err := harness.CopyFixtureToWorkDir(ctx, fixDir); err != nil {
				ctx.T.Fatalf("CopyFixtureToWorkDir: %v", err)
			}
		},
		When: actions.RunBatchAutoWithFixture(fixDir),
		Then: []func(*harness.Context){
			verify.ExitCodeZero(),
			verify.FileExists("classified.csv"),
			verify.FileExists("review.csv"),
			verify.RowCountAtLeast("classified.csv", 1), // header + at least 1 data row
			verify.ColumnCount("classified.csv", 7),
			verify.AllConfidencesInRange("classified.csv", 5), // col 5 = confidence
		},
	})
}

func TestBatchAuto_MixedConfidence(t *testing.T) {
	harness.RequireOllama(t, "")

	fixDir := filepath.Join(fixturesDir(), "batch-auto-basic")

	harness.Run(t, harness.Scenario{
		Name: "batch-auto — classified.csv has 11 rows (1 header + 10 data), 7 columns",
		Given: func(ctx *harness.Context) {
			ctx.BinaryPath = binaryPath
			ctx.FixtureDir = fixDir
			if err := harness.CopyFixtureToWorkDir(ctx, fixDir); err != nil {
				ctx.T.Fatalf("CopyFixtureToWorkDir: %v", err)
			}
		},
		When: actions.RunBatchAutoWithFixture(fixDir),
		Then: []func(*harness.Context){
			verify.ExitCodeZero(),
			verify.FileExists("classified.csv"),
			verify.FileExists("review.csv"),
			verify.RowCount("classified.csv", 11), // 1 header + 10 data rows
			verify.ColumnCount("classified.csv", 7),
			verify.AllConfidencesInRange("classified.csv", 5),
		},
	})
}
