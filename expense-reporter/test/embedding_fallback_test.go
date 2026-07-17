//go:build acceptance

package acceptance_test

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"expense-reporter/test/actions"
	"expense-reporter/test/domain"
	"expense-reporter/test/extern"
	"github.com/leandror172/acceptance-harness/harness"
	"github.com/leandror172/acceptance-harness/verify"
)

// TestEmbeddingFallback_KeywordMissRetrievesByEmbedding verifies the 5.R2
// embed-on-miss cascade end to end: an item whose tokens match NO keyword still
// gets few-shot examples (retrieved by embedding similarity), and the per-model
// embedding cache is populated in the data dir as a side effect.
// Requires a live Ollama with the snowflake-arctic-embed2 model pulled.
func TestEmbeddingFallback_KeywordMissRetrievesByEmbedding(t *testing.T) {
	extern.RequireOllama(t, "")

	harness.Run(t, harness.Scenario{
		Name:  "keyword-miss item gets embedding-retrieved few-shot examples + cache populated",
		Given: smallPoolRecordedWithNoKeywordForQuery(),
		When:  actions.RunClassify("--verbose", "Pastel de feira", "25,00", "15/04"),
		Then: slices.Concat(
			commandSucceeded(),
			embeddingFallbackInjectedExamples(),
			embeddingCachePopulatedForPool(),
		),
	})
}

// --- Given helpers ---

// smallPoolRecordedWithNoKeywordForQuery builds a temp data dir holding a 3-item
// fake training pool and a keyword index whose keywords (uber/supermercado/aluguel)
// do NOT match the query item — forcing the keyword layer to miss so the embedding
// fallback is the only example source. Fake data lives in fixtures/embedding-fallback.
func smallPoolRecordedWithNoKeywordForQuery() func(*harness.Context) {
	return func(ctx *harness.Context) {
		ctx.BinaryPath = binaryPath
		fixDir := filepath.Join(fixturesDir(), "embedding-fallback")

		tmpDir := ctx.T.TempDir()
		if err := copyFile(filepath.Join(fixDir, "mini-training.json"),
			filepath.Join(tmpDir, "training_data_complete.json")); err != nil {
			ctx.T.Fatalf("copy mini training pool: %v", err)
		}
		if err := copyFile(filepath.Join(fixDir, "mini-feature-dict.json"),
			filepath.Join(tmpDir, "feature_dictionary_enhanced.json")); err != nil {
			ctx.T.Fatalf("copy mini feature dict: %v", err)
		}
		domain.SetDataDir(ctx, tmpDir)
		withFeedbackAndTaxonomyConfig(ctx, filepath.Join(fixturesDir(), "json-output")) // T-13: classify needs a taxonomy
	}
}

// --- Then helpers ---

// embeddingFallbackInjectedExamples asserts the --verbose output shows the
// embedding-fallback debug line — proof the keyword layer missed and the
// embedding layer supplied the examples.
func embeddingFallbackInjectedExamples() []func(*harness.Context) {
	return []func(*harness.Context){
		verify.OutputContains("few-shot: embedding fallback",
			"embedding fallback debug line should appear in --verbose output"),
	}
}

// embeddingCachePopulatedForPool asserts the per-model embedding cache file was
// created in the data dir — the durable side effect of the first reconcile
// (pool vectors written once, reused by subsequent runs).
func embeddingCachePopulatedForPool() []func(*harness.Context) {
	return []func(*harness.Context){
		func(ctx *harness.Context) {
			cache := filepath.Join(domain.DataDir(ctx), "embeddings-snowflake-arctic-embed2.jsonl")
			info, err := os.Stat(cache)
			if err != nil {
				ctx.T.Errorf("embedding cache not created at %s: %v", cache, err)
				return
			}
			if info.Size() == 0 {
				ctx.T.Errorf("embedding cache at %s is empty — pool was not embedded", cache)
			}
		},
	}
}
