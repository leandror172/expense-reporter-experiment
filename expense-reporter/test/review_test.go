//go:build acceptance

package acceptance_test

import (
	"encoding/json"
	"path/filepath"
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"

	"expense-reporter/test/actions"
	"expense-reporter/test/harness"
	"expense-reporter/test/verify"
)

func TestReview_ProducesHTMLWithQueueAndTaxonomy(t *testing.T) {
	harness.RequireWorkbook(t, testWorkbook)

	fixDir := filepath.Join(fixturesDir(), "review-basic")
	csvPath := filepath.Join(fixDir, "input.csv")

	harness.Run(t, harness.Scenario{
		Name:  "review command produces HTML with queue and taxonomy",
		Given: setupReviewScenario(fixDir),
		When:  actions.RunReview(csvPath),
		Then: slices.Concat(
			thenReviewHTMLExists(),
			thenReviewDataScriptPresent(),
			thenReviewQueueHasRows(5),
			thenReviewTaxonomyHasSheets([]string{"Fixas", "Variáveis", "Extras", "Adicionais"}),
		),
	})
}

func setupReviewScenario(fixDir string) func(*harness.Context) {
	return func(ctx *harness.Context) {
		ctx.BinaryPath = binaryPath
		ctx.FixtureDir = fixDir
		ctx.DataDir = dataDir
		if err := harness.CopyWorkbookToWorkDir(ctx, testWorkbook); err != nil {
			ctx.T.Fatalf("CopyWorkbookToWorkDir: %v", err)
		}
	}
}

func thenReviewHTMLExists() []func(*harness.Context) {
	return []func(*harness.Context){
		verify.CommandSucceeded(),
		verify.OutputFileExists("review.html"),
	}
}

func thenReviewDataScriptPresent() []func(*harness.Context) {
	return []func(*harness.Context){
		verify.HTMLFileContainsScript("review.html", "review-data"),
	}
}

type reviewQueueOnly struct {
	Queue []json.RawMessage `json:"queue"`
}

func thenReviewQueueHasRows(expectedCount int) []func(*harness.Context) {
	return []func(*harness.Context){
		func(ctx *harness.Context) {
			ctx.T.Helper()
			var data reviewQueueOnly
			verify.HTMLFileEmbeddedJSON("review.html", "review-data", &data)(ctx)
			assert.Len(ctx.T, data.Queue, expectedCount, "queue should have %d rows", expectedCount)
		},
	}
}

type reviewTaxonomyOnly struct {
	Taxonomy struct {
		Sheets []struct {
			Name string `json:"name"`
		} `json:"sheets"`
	} `json:"taxonomy"`
}

func thenReviewTaxonomyHasSheets(expectedSheets []string) []func(*harness.Context) {
	return []func(*harness.Context){
		func(ctx *harness.Context) {
			ctx.T.Helper()
			var data reviewTaxonomyOnly
			verify.HTMLFileEmbeddedJSON("review.html", "review-data", &data)(ctx)
			actualNames := make([]string, len(data.Taxonomy.Sheets))
			for i, s := range data.Taxonomy.Sheets {
				actualNames[i] = s.Name
			}
			assert.ElementsMatch(ctx.T, expectedSheets, actualNames, "taxonomy sheets mismatch")
		},
	}
}
