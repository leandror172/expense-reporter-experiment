//go:build acceptance

package acceptance_test

// Incremental acceptance suite for the full type-routing cycle that validated the
// `sheets`→`types` review-serialization fix (session 34, Option A integration test).
//
// The cycle has six conceptual steps; the four that are CLI commands each get a test,
// and each successive test folds the prior steps into its Given as preparation. The two
// non-CLI steps are represented as fixtures and noted where they fold in:
//
//   1. batch-auto            → TestTypeRoutingCycle_1_BatchAutoEmitsType        (When)
//   2. bridge true/false→1/0 → folded into the `review-input.csv` fixture (a known
//                              CSV-format gap between writeClassifiedCSV and ReadQueue)
//   3. review                → TestTypeRoutingCycle_2_ReviewRendersTypes        (When)
//   4. browser pick + export → folded into the `reviewed.json` fixture (can't drive a
//                              browser from the harness)
//   5. apply                 → TestTypeRoutingCycle_3_ApplyBackfillsType        (When)
//   6. generate-workbook     → TestTypeRoutingCycle_4_GeneratedWorkbookRoutesByType (When)
//
// The last test seeds the cumulative post-apply state and runs ONLY generate-workbook as
// its When — all earlier steps are preparation. It proves the payoff: a typed entry routes
// by full path to its sheet, where the bare ambiguous name ("Dentista" ∈ Variáveis+Extras)
// alone could not.

import (
	"fmt"
	"path/filepath"
	"slices"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/xuri/excelize/v2"

	"expense-reporter/test/actions"
	"expense-reporter/test/harness"
	"expense-reporter/test/verify"
)

func typeRoutingFixtureDir() string {
	return filepath.Join(fixturesDir(), "type-routing-cycle")
}

// ---------------------------------------------------------------------------
// Step 1 — batch-auto emits the type column (RUI-4). Ollama-gated; the only
// non-deterministic step, so it asserts the structural type-column contract, not
// classifier-chosen values.
// ---------------------------------------------------------------------------

func TestTypeRoutingCycle_1_BatchAutoEmitsType(t *testing.T) {
	harness.RequireOllama(t, "")

	fixDir := typeRoutingFixtureDir()

	harness.Run(t, harness.Scenario{
		Name:  "batch-auto writes a type column into classified.csv",
		Given: expensesAwaitingClassification(fixDir),
		When:  actions.RunBatchAutoWithFixture(fixDir),
		Then: slices.Concat(
			commandSucceeded(),
			classifiedCsvCarriesTypeColumn(),
		),
	})
}

// ---------------------------------------------------------------------------
// Step 3 — review renders the taxonomy as `types` and pre-fills predicted.type
// (THE FIX). Deterministic: input is the bridged review-input.csv (step 2 folded in)
// and an ambiguous-Dentista reference workbook.
// ---------------------------------------------------------------------------

func TestTypeRoutingCycle_2_ReviewRendersTypes(t *testing.T) {
	fixDir := typeRoutingFixtureDir()
	reviewInput := filepath.Join(fixDir, "review-input.csv")

	harness.Run(t, harness.Scenario{
		Name:  "review embeds taxonomy.types and pre-fills predicted.type per row",
		Given: expensesClassifiedWithTypes(fixDir),
		When:  actions.RunReview(reviewInput),
		Then: slices.Concat(
			reviewHTMLProduced(),
			reviewDataEmbedded(),
			pendingExpensesQueued(3),
			workbookSheetsInTaxonomy([]string{"Fixas", "Variáveis", "Extras"}),
			predictedTypesPrefilled(map[string]string{
				"Uber Centro":      "Variáveis",
				"Diarista Letícia": "Fixas",
				"Dentista Dra Ana": "", // ambiguous → left for the human to pick
			}),
		),
	})
}

// ---------------------------------------------------------------------------
// Step 5 — apply backfills the human-chosen type into the expense log.
// Deterministic: the browser export (step 4) is the reviewed.json fixture; apply's
// new-row insert needs a workbook, so the Given builds a hermetic skeleton with
// generate-workbook (no committed binary fixture).
// ---------------------------------------------------------------------------

func TestTypeRoutingCycle_3_ApplyBackfillsType(t *testing.T) {
	fixDir := typeRoutingFixtureDir()
	reviewedPath := filepath.Join(fixDir, "reviewed.json")

	harness.Run(t, harness.Scenario{
		Name:  "apply inserts the confirmed entry and logs its type to expenses_log.jsonl",
		Given: expenseTypedDuringBrowserReview(fixDir),
		When:  actions.RunApply(reviewedPath),
		Then: slices.Concat(
			commandSucceeded(),
			expenseLogMatchesExpected(fixDir),
		),
	})
}

// ---------------------------------------------------------------------------
// Step 6 (last) — generate-workbook routes the typed entry by full path.
// All prior steps are preparation: the Given seeds the cumulative typed expense log
// (batch-auto + review + apply) and the only When is generate-workbook.
// ---------------------------------------------------------------------------

func TestTypeRoutingCycle_4_GeneratedWorkbookRoutesByType(t *testing.T) {
	fixDir := typeRoutingFixtureDir()
	taxonomyPath := filepath.Join(fixDir, "taxonomy.json")
	entriesPath := filepath.Join(fixDir, "typed-expenses_log.jsonl")

	harness.Run(t, harness.Scenario{
		Name:  "generate-workbook routes the ambiguous-leaf entry to its typed sheet",
		Given: cycleCompletedThroughApply(fixDir),
		When:  actions.RunGenerateWorkbook(taxonomyPath, entriesPath),
		Then: slices.Concat(
			commandSucceeded(),
			typedEntryRoutedToSheet(250, "Variáveis"), // Dentista → its chosen type
			valueAbsentFromSheet(250, "Extras"),       // not the other ambiguous candidate
		),
	})
}

// --- Given helpers (Event Modeling style — past-tense events) ---

func expensesAwaitingClassification(fixDir string) func(*harness.Context) {
	return func(ctx *harness.Context) {
		ctx.BinaryPath = binaryPath
		ctx.DataDir = dataDir
		ctx.FixtureDir = fixDir
		if err := harness.CopyFixtureToWorkDir(ctx, fixDir); err != nil {
			ctx.T.Fatalf("CopyFixtureToWorkDir: %v", err)
		}
	}
}

func expensesClassifiedWithTypes(fixDir string) func(*harness.Context) {
	return func(ctx *harness.Context) {
		ctx.BinaryPath = binaryPath
		ctx.FixtureDir = fixDir
		ctx.WorkbookPath = createAmbiguousReferenceWorkbook(ctx.T, ctx.WorkDir)
	}
}

func expenseTypedDuringBrowserReview(fixDir string) func(*harness.Context) {
	return func(ctx *harness.Context) {
		ctx.BinaryPath = binaryPath
		withFeedbackConfig(ctx) // classifications.jsonl + expenses_log.jsonl in WorkDir
		// apply's new-row insert needs a workbook; build a hermetic skeleton from the
		// same taxonomy that generate-workbook uses, so the Saúde/Dentista slot exists.
		buildSkeletonWorkbook(ctx, filepath.Join(fixDir, "taxonomy.json"))
	}
}

func cycleCompletedThroughApply(fixDir string) func(*harness.Context) {
	return func(ctx *harness.Context) {
		ctx.BinaryPath = binaryPath
		// The typed log is supplied directly to the When via the entries flag; nothing
		// else is needed — this models the cumulative state after apply.
		ctx.FixtureDir = fixDir
	}
}

// buildSkeletonWorkbook runs generate-workbook (no entries) to produce an empty workbook
// and points ctx.WorkbookPath at it. Fails the test if generation does not succeed.
func buildSkeletonWorkbook(ctx *harness.Context, taxonomyPath string) {
	ctx.T.Helper()
	actions.RunGenerateWorkbook(taxonomyPath, "")(ctx)
	if ctx.ExitCode != 0 {
		ctx.T.Fatalf("buildSkeletonWorkbook: generate-workbook failed (exit %d): %s", ctx.ExitCode, ctx.Stderr)
	}
	ctx.WorkbookPath = ctx.Artifacts["generated-workbook"]
}

// createAmbiguousReferenceWorkbook builds a reference-sheet-only workbook where
// Saúde/Dentista appears under BOTH Variáveis and Extras, so review's taxonomy carries
// the ambiguity. Rows 1-4 are header rows skipped by LoadReferenceSheet.
func createAmbiguousReferenceWorkbook(t testing.TB, dir string) string {
	t.Helper()

	path := filepath.Join(dir, "reference-workbook.xlsx")
	f := excelize.NewFile()
	if err := f.SetSheetName("Sheet1", "Referência de Categorias"); err != nil {
		t.Fatalf("rename sheet: %v", err)
	}

	rows := [][]string{
		{"Fixas", "Habitação", "Diarista"},
		{"Variáveis", "Transporte", "Uber/Taxi"},
		{"Variáveis", "Saúde", "Dentista"},
		{"Extras", "Saúde", "Dentista"},
	}
	for i, row := range rows {
		rowNum := i + 5
		for j, value := range row {
			cell := fmt.Sprintf("%c%d", 'A'+j, rowNum)
			if err := f.SetCellValue("Referência de Categorias", cell, value); err != nil {
				t.Fatalf("set cell %s: %v", cell, err)
			}
		}
	}
	if err := f.SaveAs(path); err != nil {
		t.Fatalf("save workbook: %v", err)
	}
	return path
}

// --- Then helpers (composable, one concern each) ---

func classifiedCsvCarriesTypeColumn() []func(*harness.Context) {
	return []func(*harness.Context){
		verify.OutputFileExists("classified.csv"),
		verify.OutputFileHasColumns("classified.csv", 8), // 7 original + type
	}
}

// reviewQueuePredicted captures only the per-row item + predicted.type from review-data.
type reviewQueuePredicted struct {
	Queue []struct {
		Item      string `json:"item"`
		Predicted struct {
			Type string `json:"type"`
		} `json:"predicted"`
	} `json:"queue"`
}

// predictedTypesPrefilled asserts each named item's predicted.type matches want (empty
// string means the type must be absent/blank — the ambiguous case).
func predictedTypesPrefilled(want map[string]string) []func(*harness.Context) {
	return []func(*harness.Context){
		func(ctx *harness.Context) {
			ctx.T.Helper()
			var data reviewQueuePredicted
			verify.HTMLFileEmbeddedJSON("review.html", "review-data", &data)(ctx)
			got := make(map[string]string, len(data.Queue))
			for _, q := range data.Queue {
				got[q.Item] = q.Predicted.Type
			}
			for item, wantType := range want {
				assert.Equal(ctx.T, wantType, got[item], "predicted.type for %q", item)
			}
		},
	}
}

// typedEntryRoutedToSheet asserts the generated workbook's named sheet contains a cell
// whose numeric value equals want — i.e. the entry routed to that type's sheet.
func typedEntryRoutedToSheet(want float64, sheet string) []func(*harness.Context) {
	return []func(*harness.Context){
		func(ctx *harness.Context) {
			ctx.T.Helper()
			assert.True(ctx.T, sheetContainsValue(ctx, sheet, want),
				"expected value %g in sheet %q (full-path routing)", want, sheet)
		},
	}
}

// valueAbsentFromSheet asserts the named sheet does NOT contain the value — proving the
// entry routed by its type, not duplicated into the other ambiguous candidate sheet.
func valueAbsentFromSheet(want float64, sheet string) []func(*harness.Context) {
	return []func(*harness.Context){
		func(ctx *harness.Context) {
			ctx.T.Helper()
			assert.False(ctx.T, sheetContainsValue(ctx, sheet, want),
				"value %g should not appear in sheet %q", want, sheet)
		},
	}
}

// sheetContainsValue opens the generated-workbook artifact and reports whether any cell
// in the named sheet parses to want (within a small epsilon).
func sheetContainsValue(ctx *harness.Context, sheet string, want float64) bool {
	ctx.T.Helper()
	path, ok := ctx.Artifacts["generated-workbook"]
	if !assert.True(ctx.T, ok, "generated-workbook artifact not registered") {
		return false
	}
	f, err := excelize.OpenFile(path)
	if err != nil {
		ctx.T.Fatalf("open generated workbook: %v", err)
	}
	defer f.Close()

	rows, err := f.GetRows(sheet)
	if err != nil {
		ctx.T.Fatalf("read sheet %q: %v", sheet, err)
	}
	for _, row := range rows {
		for _, cell := range row {
			if v, err := strconv.ParseFloat(cell, 64); err == nil && abs(v-want) < 0.001 {
				return true
			}
		}
	}
	return false
}

func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}
