package classifier

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func exampleItems(exs []Example) []string {
	out := make([]string, len(exs))
	for i, e := range exs {
		out[i] = e.Item
	}
	return out
}

func TestCosineSimilarity(t *testing.T) {
	assert.InDelta(t, 1.0, cosineSimilarity([]float64{1, 0}, []float64{1, 0}), 1e-9)
	assert.InDelta(t, 0.0, cosineSimilarity([]float64{1, 0}, []float64{0, 1}), 1e-9)
	assert.InDelta(t, -1.0, cosineSimilarity([]float64{1, 0}, []float64{-1, 0}), 1e-9)
	assert.InDelta(t, 0.7071067811865476,
		cosineSimilarity([]float64{1, 0}, []float64{1, 1}), 1e-9)
}

func TestCosineSimilarity_DegenerateReturnsZero(t *testing.T) {
	assert.Equal(t, 0.0, cosineSimilarity([]float64{0, 0}, []float64{1, 1}))    // zero magnitude
	assert.Equal(t, 0.0, cosineSimilarity([]float64{1, 2, 3}, []float64{1, 2})) // length mismatch
}

func TestTopKEmbeddingExamples_RanksByCosine(t *testing.T) {
	pool := []Example{
		{Item: "C", Subcategory: "sc", Source: SourceTraining},
		{Item: "A", Subcategory: "sc", Source: SourceTraining},
		{Item: "B", Subcategory: "sc", Source: SourceTraining},
	}
	store := map[string][]float64{
		"A": {1, 0},
		"B": {0.7071067811865476, 0.7071067811865476},
		"C": {0, 1},
	}
	got := TopKEmbeddingExamples("query", []float64{1, 0}, pool, store, 5)
	assert.Equal(t, []string{"A", "B", "C"}, exampleItems(got))
}

func TestTopKEmbeddingExamples_DedupsSlateByKey(t *testing.T) {
	// "Uber" and "uber " share a dedup key → collapse to one slate entry, so K=5
	// yields DISTINCT examples, not a slate padded with duplicates.
	pool := []Example{
		{Item: "Uber", Subcategory: "Transporte", Source: SourceTraining},
		{Item: "uber ", Subcategory: "Transporte", Source: SourceTraining},
		{Item: "Spotify", Subcategory: "Assinaturas", Source: SourceTraining},
	}
	store := map[string][]float64{
		"Uber":    {1, 0},
		"uber ":   {1, 0},
		"Spotify": {0, 1},
	}
	got := TopKEmbeddingExamples("query", []float64{1, 0}, pool, store, 5)
	require.Len(t, got, 2)
	assert.Equal(t, "Uber", got[0].Item) // first-seen representative for equal source priority
	assert.Equal(t, "Spotify", got[1].Item)
}

func TestTopKEmbeddingExamples_PrefersHighestSourcePriority(t *testing.T) {
	// Duplicate-key rows: keep the highest-priority source (Corrected > Training > Confirmed).
	pool := []Example{
		{Item: "Uber", Subcategory: "X", Source: SourceTraining},
		{Item: "uber", Subcategory: "X", Source: SourceCorrected},
	}
	store := map[string][]float64{"Uber": {1, 0}, "uber": {1, 0}}
	got := TopKEmbeddingExamples("query", []float64{1, 0}, pool, store, 5)
	require.Len(t, got, 1)
	assert.Equal(t, SourceCorrected, got[0].Source)
	assert.Equal(t, "uber", got[0].Item)
}

func TestTopKEmbeddingExamples_TruncatesToK(t *testing.T) {
	pool := []Example{
		{Item: "A"}, {Item: "B"}, {Item: "C"}, {Item: "D"},
	}
	store := map[string][]float64{
		"A": {1, 0}, "B": {0.9, 0.1}, "C": {0.5, 0.5}, "D": {0, 1},
	}
	got := TopKEmbeddingExamples("q", []float64{1, 0}, pool, store, 2)
	require.Len(t, got, 2)
	assert.Equal(t, []string{"A", "B"}, exampleItems(got))
}

func TestTopKEmbeddingExamples_ExcludesQueryOwnKey(t *testing.T) {
	// Leave-one-out: a same-item feedback row could exist in the pool; the query must
	// not retrieve itself. "netflix " matches "Netflix" by dedup key.
	pool := []Example{
		{Item: "Netflix", Subcategory: "Assinaturas", Source: SourceTraining},
		{Item: "Spotify", Subcategory: "Assinaturas", Source: SourceTraining},
	}
	store := map[string][]float64{"Netflix": {1, 0}, "Spotify": {0.9, 0.1}}
	got := TopKEmbeddingExamples("netflix ", []float64{1, 0}, pool, store, 5)
	require.Len(t, got, 1)
	assert.Equal(t, "Spotify", got[0].Item)
}

func TestTopKEmbeddingExamples_SkipsRowsWithoutVector(t *testing.T) {
	pool := []Example{
		{Item: "A"}, {Item: "NoVec"}, {Item: "B"},
	}
	store := map[string][]float64{"A": {1, 0}, "B": {0.5, 0.5}}
	got := TopKEmbeddingExamples("q", []float64{1, 0}, pool, store, 5)
	assert.Equal(t, []string{"A", "B"}, exampleItems(got))
}

func TestTopKEmbeddingExamples_NonPositiveKReturnsNil(t *testing.T) {
	got := TopKEmbeddingExamples("q", []float64{1, 0},
		[]Example{{Item: "A"}}, map[string][]float64{"A": {1, 0}}, 0)
	assert.Nil(t, got)
}

func TestTopKEmbeddingExamples_TiesKeepFirstSeenOrder(t *testing.T) {
	// Determinism contract (load-bearing for the Phase-4 A/B replay): when several
	// distinct items have IDENTICAL similarity, ranking must preserve first-seen pool
	// order — never depend on map iteration. All three vectors are identical here.
	pool := []Example{
		{Item: "X", Subcategory: "s", Source: SourceTraining},
		{Item: "Y", Subcategory: "s", Source: SourceTraining},
		{Item: "Z", Subcategory: "s", Source: SourceTraining},
	}
	store := map[string][]float64{"X": {1, 0}, "Y": {1, 0}, "Z": {1, 0}}
	got := TopKEmbeddingExamples("query", []float64{1, 0}, pool, store, 5)
	assert.Equal(t, []string{"X", "Y", "Z"}, exampleItems(got))
}

func TestTopKEmbeddingExamples_PreservesSourceAndTypeHint(t *testing.T) {
	// Retrieved examples must carry Source/TypeHint through so resolveExamplePaths and
	// source-priority semantics keep working downstream.
	pool := []Example{
		{Item: "Consulta", Subcategory: "Médico", Source: SourceCorrected, TypeHint: "Variáveis"},
	}
	store := map[string][]float64{"Consulta": {1, 0}}
	got := TopKEmbeddingExamples("q", []float64{1, 0}, pool, store, 5)
	require.Len(t, got, 1)
	assert.Equal(t, SourceCorrected, got[0].Source)
	assert.Equal(t, "Variáveis", got[0].TypeHint)
	assert.Equal(t, "Médico", got[0].Subcategory)
}
