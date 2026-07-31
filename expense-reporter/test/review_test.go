//go:build acceptance

package acceptance_test

import (
	"encoding/json"
	"path/filepath"
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"

	"expense-reporter/test/actions"
	"expense-reporter/test/expect"
	"github.com/leandror172/acceptance-harness/harness"
	"github.com/leandror172/acceptance-harness/verify"
)

func TestReview_ProducesHTMLWithQueueAndTaxonomy(t *testing.T) {
	fixDir := filepath.Join(fixturesDir(), "review-basic")
	csvPath := filepath.Join(fixDir, "input.csv")

	harness.Run(t, harness.Scenario{
		Name:    "review command produces HTML with queue and taxonomy",
		Fixture: fixDir,
		Given:   expensesReceivedForReview(),
		When:    actions.RunReview(csvPath),
		Then: slices.Concat(
			reviewHTMLProduced(),
			reviewDataEmbedded(),
			pendingExpensesQueued(5),
			expenseTypesOfferedForPicking([]string{"Fixas", "Variáveis", "Extras", "Adicionais"}),
		),
	})
}

// expensesReceivedForReview: since S2 the picker's vocabulary comes from
// config/taxonomy.json — the same file every other command reads — so publishing the
// taxonomy is the ENTIRE precondition. It used to build a synthetic workbook and point
// review at it, because review derived its types from the workbook's Referência sheet.
// That made the close cycle run on two taxonomies that had measurably drifted apart
// (7 leaves, incl. IRFF/IRRF), so a reviewer could pick a leaf that generate-workbook
// could not route — and the row vanished silently.
func expensesReceivedForReview() func(*harness.Context) {
	return taxonomyPublished()
}

func reviewHTMLProduced() []func(*harness.Context) {
	return []func(*harness.Context){
		verify.CommandSucceeded(),
		verify.OutputFileExists("review.html"),
	}
}

func reviewDataEmbedded() []func(*harness.Context) {
	return []func(*harness.Context){
		expect.HTMLFileContainsScript("review.html", "review-data"),
	}
}

type reviewQueueOnly struct {
	Queue []json.RawMessage `json:"queue"`
}

func pendingExpensesQueued(expectedCount int) []func(*harness.Context) {
	return []func(*harness.Context){
		func(ctx *harness.Context) {
			ctx.T.Helper()
			var data reviewQueueOnly
			expect.HTMLFileEmbeddedJSON("review.html", "review-data", &data)(ctx)
			assert.Len(ctx.T, data.Queue, expectedCount, "queue should have %d rows", expectedCount)
		},
	}
}

type reviewTaxonomyOnly struct {
	Taxonomy struct {
		Types []struct {
			Name string `json:"name"`
		} `json:"types"`
	} `json:"taxonomy"`
}

func expenseTypesOfferedForPicking(expectedTypes []string) []func(*harness.Context) {
	return []func(*harness.Context){
		func(ctx *harness.Context) {
			ctx.T.Helper()
			var data reviewTaxonomyOnly
			expect.HTMLFileEmbeddedJSON("review.html", "review-data", &data)(ctx)
			actualNames := make([]string, len(data.Taxonomy.Types))
			for i, s := range data.Taxonomy.Types {
				actualNames[i] = s.Name
			}
			assert.ElementsMatch(ctx.T, expectedTypes, actualNames,
				"the picker must offer exactly the types in config/taxonomy.json")
		},
	}
}
