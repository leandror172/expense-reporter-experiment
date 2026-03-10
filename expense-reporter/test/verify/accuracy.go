//go:build acceptance

package verify

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/stretchr/testify/assert"

	"expense-reporter/test/harness"
)

// AccuracyTuple records a single row comparison.
type AccuracyTuple struct {
	Item     string `json:"item"`
	Expected string `json:"expected"`
	Got      string `json:"got"`
	Match    bool   `json:"match"`
}

// AccuracyReport is written to resultsDir for drift tracking.
type AccuracyReport struct {
	Total    int             `json:"total"`
	Correct  int             `json:"correct"`
	Accuracy float64         `json:"accuracy"`
	Details  []AccuracyTuple `json:"details"`
}

// ClassificationAccuracyAtLeast compares the subcategory column of the actual
// classified.csv against an expected reference file. Fails only if accuracy < floor.
// Writes an AccuracyReport JSON to resultsDir for drift tracking over time.
func ClassificationAccuracyAtLeast(artifactKey, expectedPath string, floor float64, resultsDir string) func(*harness.Context) {
	return func(ctx *harness.Context) {
		ctx.T.Helper()
		actual := readArtifact(ctx, artifactKey)
		if actual == nil {
			return
		}
		expected := harness.ReadCSVFile(ctx.T, expectedPath)

		// Skip header row in actual (classified.csv has a real CSV header).
		// expected-classified.csv uses # comments (stripped by ReadCSVFile) — no header row.
		if len(actual) > 0 {
			actual = actual[1:]
		}

		total := len(actual)
		if len(expected) < total {
			total = len(expected)
		}

		const subcatCol = 3
		var details []AccuracyTuple
		correct := 0
		for i := 0; i < total; i++ {
			item := ""
			if len(actual[i]) > 0 {
				item = actual[i][0]
			}
			got := ""
			if subcatCol < len(actual[i]) {
				got = actual[i][subcatCol]
			}
			want := ""
			if subcatCol < len(expected[i]) {
				want = expected[i][subcatCol]
			}
			match := got == want
			if match {
				correct++
			}
			details = append(details, AccuracyTuple{Item: item, Expected: want, Got: got, Match: match})
		}

		var accuracy float64
		if total > 0 {
			accuracy = float64(correct) / float64(total)
		}

		report := AccuracyReport{Total: total, Correct: correct, Accuracy: accuracy, Details: details}
		if err := os.MkdirAll(resultsDir, 0o755); err == nil {
			name := ctx.T.Name()
			reportPath := filepath.Join(resultsDir, fmt.Sprintf("%s.json", name))
			if data, err := json.MarshalIndent(report, "", "  "); err == nil {
				_ = os.WriteFile(reportPath, data, 0o644)
			}
		}

		assert.GreaterOrEqual(ctx.T, accuracy, floor,
			"classification accuracy %.0f%% is below required floor %.0f%% (%d/%d correct)\nSee %s for details",
			accuracy*100, floor*100, correct, total, resultsDir)
	}
}

// NoneWereAutoInserted asserts all data rows in the artifact have auto_inserted == "false".
// Skips the header row. Use to verify that no expense passed the auto-insert threshold.
func NoneWereAutoInserted(artifactKey string) func(*harness.Context) {
	const autoInsertedCol = 6
	return func(ctx *harness.Context) {
		ctx.T.Helper()
		rows := readArtifact(ctx, artifactKey)
		if rows == nil {
			return
		}
		if len(rows) > 0 {
			rows = rows[1:] // skip header
		}
		for i, row := range rows {
			if !assert.Greater(ctx.T, len(row), autoInsertedCol,
				"%q row %d missing auto_inserted column", artifactKey, i) {
				continue
			}
			assert.Equal(ctx.T, "false", row[autoInsertedCol],
				"%q row %d: expense was auto-inserted but should have been kept for manual review",
				artifactKey, i)
		}
	}
}
