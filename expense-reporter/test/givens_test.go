//go:build acceptance

package acceptance_test

// Shared Given vocabulary for the whole suite.
//
// Scenario plumbing belongs to the engine (harness v1.1): `Scenario.Fixture` seeds
// ctx.FixtureDir and `harness.UseBinary` (TestMain) seeds ctx.BinaryPath, so no Given
// wires either — and no Given takes a fixture path just to pass it along. Events read
// the fixture from the context.
//
// Two layers live here:
//   1. atomic events    — one fact each, composable in any combination
//   2. canonical Givens — the three combinations nearly every scenario wants
//
// Scenario-specific Givens elsewhere should be one-line wrappers over a canonical (or a
// harness.Events(...) of atoms), never a fresh body. See PATTERNS.md "Canonical Givens".

import (
	"os"
	"path/filepath"

	"expense-reporter/test/domain"
	"github.com/leandror172/acceptance-harness/harness"
)

// --- Atomic events -------------------------------------------------------------
// One fact each. Compose with harness.Events(...); do not inline into new bodies.

// trainingCorpusRecorded: the real training corpus + keyword dictionary exist, so
// few-shot retrieval and the keyword agreement gate have something to work with.
func trainingCorpusRecorded() func(*harness.Context) {
	return func(ctx *harness.Context) {
		domain.SetDataDir(ctx, dataDir)
	}
}

// taxonomyPublished: the scenario's taxonomy was published where the binary reads it —
// copied into the WorkDir and pointed at by config. Since T-13 every classifying command
// resolves its full path through this file.
func taxonomyPublished() func(*harness.Context) {
	return func(ctx *harness.Context) {
		taxonomyDest := filepath.Join(ctx.WorkDir, "taxonomy.json")

		taxData, err := os.ReadFile(filepath.Join(ctx.FixtureDir, "fixture-taxonomy.json"))
		if err != nil {
			ctx.T.Fatalf("reading fixture taxonomy: %v", err)
		}
		if err := os.WriteFile(taxonomyDest, taxData, 0o644); err != nil {
			ctx.T.Fatalf("writing taxonomy to workdir: %v", err)
		}
		if err := domain.SetupBinaryConfig(ctx, map[string]interface{}{
			"taxonomy_path": taxonomyDest,
		}); err != nil {
			ctx.T.Fatalf("SetupBinaryConfig: %v", err)
		}
	}
}

// feedbackLogsConfigured: both JSONL logs have a home in the WorkDir and are registered
// as artifacts. They start absent — a scenario needing prior entries seeds them.
func feedbackLogsConfigured() func(*harness.Context) {
	return func(ctx *harness.Context) {
		classificationsPath := filepath.Join(ctx.WorkDir, "classifications.jsonl")
		expensesLogPath := filepath.Join(ctx.WorkDir, "expenses_log.jsonl")

		if err := domain.SetupBinaryConfig(ctx, map[string]interface{}{
			"classifications_path": classificationsPath,
			"expenses_log_path":    expensesLogPath,
		}); err != nil {
			ctx.T.Fatalf("SetupBinaryConfig: %v", err)
		}
		ctx.Artifacts["classifications.jsonl"] = classificationsPath
		ctx.Artifacts["expenses_log.jsonl"] = expensesLogPath
	}
}

// inputBatchStaged: the fixture's files were copied into the WorkDir, because the
// command writes its output beside its input and must not touch the fixture.
func inputBatchStaged() func(*harness.Context) {
	return func(ctx *harness.Context) {
		if err := harness.CopyFixtureToWorkDir(ctx, ctx.FixtureDir); err != nil {
			ctx.T.Fatalf("CopyFixtureToWorkDir: %v", err)
		}
	}
}

// --- Canonical Givens ----------------------------------------------------------

// jsonOutputFixture is the taxonomy-only fixture used by scenarios that author no
// fixture data of their own — they need a configured taxonomy and nothing else.
func jsonOutputFixture() string {
	return filepath.Join(fixturesDir(), "json-output")
}

// taxonomyAuthoredWithTrainingData: a taxonomy exists and the real training corpus
// (+ keyword index) was recorded — the ordinary state of a working install, and the
// most common Given in the suite. Read-only: the fixture is NOT staged in the WorkDir,
// so scenarios whose command writes beside its input want
// expenseBatchSubmittedForClassification instead.
func taxonomyAuthoredWithTrainingData() func(*harness.Context) {
	return harness.Events(
		trainingCorpusRecorded(),
		taxonomyPublished(),
		feedbackLogsConfigured(),
	)
}

// taxonomyAuthoredWithoutTrainingData: a taxonomy exists but no training corpus was
// ever recorded — no data dir, and therefore no few-shot examples or keyword index.
// The honest Given for commands that never classify (add), and for dry-run paths that
// resolve a full path from the taxonomy but call no model.
func taxonomyAuthoredWithoutTrainingData() func(*harness.Context) {
	return harness.Events(
		taxonomyPublished(),
		feedbackLogsConfigured(),
	)
}

// expenseBatchSubmittedForClassification: a CSV of expenses was submitted for
// classification — staged in the WorkDir because batch-auto writes its output files
// beside the input. Shared by every batch scenario; per-scenario wrappers elsewhere
// name what their own fixture's batch contains.
func expenseBatchSubmittedForClassification() func(*harness.Context) {
	return harness.Events(
		trainingCorpusRecorded(),
		inputBatchStaged(),
		taxonomyPublished(),
		feedbackLogsConfigured(),
	)
}

// withFeedbackAndTaxonomyConfig is the pre-composition plumbing form, kept for the
// Givens that still assemble their own context. Prefer composing the atoms above.
func withFeedbackAndTaxonomyConfig(ctx *harness.Context) {
	taxonomyPublished()(ctx)
	feedbackLogsConfigured()(ctx)
}

// withFeedbackConfig gives both JSONL logs a home without publishing a taxonomy.
func withFeedbackConfig(ctx *harness.Context) {
	feedbackLogsConfigured()(ctx)
}
