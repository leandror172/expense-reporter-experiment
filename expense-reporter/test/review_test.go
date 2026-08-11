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
			cleanQueueReportsNothingUnreviewable(),
			installmentCountOfferedToTheReviewer("Financiamento carro", 24),
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

type reviewQueueInstallments struct {
	Queue []struct {
		Item         string `json:"item"`
		Installments int    `json:"installments"`
	} `json:"queue"`
}

// installmentCountOfferedToTheReviewer asserts the count parsed out of the raw "total/N"
// token reaches the page's embedded data, which is where the export JS reads it from.
//
// This is the Go half of the T-21 producer→consumer chain. The remaining half — that
// exportReviewed() actually copies the field into reviewed.json — runs in the browser and
// no Go test can reach it; see the note in test/README.md. Without this assertion the
// whole JS hop would be unguarded, since the only other evidence is a fixture someone
// hand-authored to contain the field.
func installmentCountOfferedToTheReviewer(item string, want int) []func(*harness.Context) {
	return []func(*harness.Context){
		func(ctx *harness.Context) {
			ctx.T.Helper()
			var data reviewQueueInstallments
			expect.HTMLFileEmbeddedJSON("review.html", "review-data", &data)(ctx)
			for _, row := range data.Queue {
				if row.Item == item {
					assert.Equal(ctx.T, want, row.Installments,
						"queue row %q should carry its installment count", item)
					return
				}
			}
			ctx.T.Errorf("queue has no row for %q", item)
		},
	}
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

// TestReview_UnparsedRowsAreSkippedNotFatal pins the S1 contract found by the T-42 scout:
// batch-auto DELIBERATELY records a row it could not parse in classified.csv with empty
// date/value cells (see cmd.TestWriteClassifiedCSV_UnparsedRowKeepsItsRawLine), and
// ReadQueue used to hard-error on exactly that shape. One such row in 69 killed the whole
// review step on real data — producer and consumer each deliberately contradicting the
// other, the same defect family as T-54 in a different column.
//
// The queue must therefore carry the 3 reviewable rows and leave out the 2 unparsed ones,
// and review must SAY so — silently dropping them would trade a loud failure for a quiet
// one, which is the trade this codebase keeps getting wrong.
func TestReview_UnparsedRowsAreSkippedNotFatal(t *testing.T) {
	fixDir := filepath.Join(fixturesDir(), "review-malformed-rows")
	csvPath := filepath.Join(fixDir, "input.csv")

	harness.Run(t, harness.Scenario{
		Name:    "review skips rows batch-auto could not parse and reports them",
		Fixture: fixDir,
		Given:   expensesReceivedForReview(),
		When:    actions.RunReview(csvPath),
		Then: slices.Concat(
			reviewHTMLProduced(),
			pendingExpensesQueued(3),
			unreviewableRowsReportedToUser(2),
		),
	})
}

func unreviewableRowsReportedToUser(n int) []func(*harness.Context) {
	return []func(*harness.Context){expect.UnreviewableRowsReported(n)}
}

func cleanQueueReportsNothingUnreviewable() []func(*harness.Context) {
	return []func(*harness.Context){expect.NoUnreviewableRowsReported()}
}
