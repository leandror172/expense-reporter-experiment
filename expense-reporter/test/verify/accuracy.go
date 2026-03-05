//go:build acceptance

package verify

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

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

// SoftAccuracy compares the subcategory column of the actual CSV against expectedPath.
// Fails only if accuracy < floor. Writes an AccuracyReport JSON to resultsDir.
func SoftAccuracy(artifactKey, expectedPath string, floor float64, subcatCol int, resultsDir string) func(*harness.Context) {
	return func(ctx *harness.Context) {
		ctx.T.Helper()
		actual := readArtifact(ctx, artifactKey)
		if actual == nil {
			return
		}
		expected := harness.ReadCSVFile(ctx.T, expectedPath)

		// Skip header row (first row) in both if present
		if len(actual) > 0 {
			actual = actual[1:]
		}
		if len(expected) > 0 {
			expected = expected[1:]
		}

		total := len(actual)
		if len(expected) < total {
			total = len(expected)
		}

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
			if 3 < len(expected[i]) {
				want = expected[i][3]
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

		if accuracy < floor {
			ctx.T.Errorf("SoftAccuracy(%q): %.0f%% < floor %.0f%% (%d/%d correct)",
				artifactKey, accuracy*100, floor*100, correct, total)
		}
	}
}

// AllInReview asserts all rows in the artifact have auto_inserted == "false".
func AllInReview(artifactKey string, autoInsertedCol int) func(*harness.Context) {
	return func(ctx *harness.Context) {
		ctx.T.Helper()
		rows := readArtifact(ctx, artifactKey)
		if rows == nil {
			return
		}
		for i, row := range rows {
			if autoInsertedCol >= len(row) {
				ctx.T.Errorf("AllInReview(%q): row %d missing col %d", artifactKey, i, autoInsertedCol)
				continue
			}
			if row[autoInsertedCol] != "false" {
				ctx.T.Errorf("AllInReview(%q): row %d auto_inserted=%q, want \"false\"",
					artifactKey, i, row[autoInsertedCol])
			}
		}
	}
}
