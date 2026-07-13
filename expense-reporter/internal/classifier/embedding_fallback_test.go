package classifier

import (
	"errors"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// countingEmbedder returns fixed vectors per text and counts Embed calls.
type countingEmbedder struct {
	vectors map[string][]float64
	calls   int
	err     error
}

func (c *countingEmbedder) Embed(text string) ([]float64, error) {
	c.calls++
	if c.err != nil {
		return nil, c.err
	}
	v, ok := c.vectors[text]
	if !ok {
		return []float64{0, 0, 1}, nil
	}
	return v, nil
}

func fallbackPool() []Example {
	return []Example{
		{Item: "Uber Leblon", Subcategory: "Uber/Taxi", Category: "Transporte", Source: SourceTraining},
		{Item: "Farmácia Drogasil", Subcategory: "Farmácia", Category: "Saúde", Source: SourceTraining},
		{Item: "Netflix mensal", Subcategory: "Streaming", Category: "Lazer", Source: SourceConfirmed},
	}
}

func fallbackVectors() map[string][]float64 {
	return map[string][]float64{
		"99 Taxi Centro":    {1, 0, 0}, // query — closest to Uber Leblon
		"Uber Leblon":       {0.9, 0.1, 0},
		"Farmácia Drogasil": {0, 1, 0},
		"Netflix mensal":    {0, 0.9, 0.1},
	}
}

func TestFallbackExamples_RanksByEmbeddingSimilarity(t *testing.T) {
	emb := &countingEmbedder{vectors: fallbackVectors()}
	cachePath := filepath.Join(t.TempDir(), "embeddings-test.jsonl")
	st := &embedState{}

	got := fallbackExamples("99 Taxi Centro", fallbackPool(), emb, cachePath, 2, st)

	require.Len(t, got, 2)
	assert.Equal(t, "Uber Leblon", got[0].Item)
}

func TestFallbackExamples_ReconcilesPoolOnlyOnce(t *testing.T) {
	emb := &countingEmbedder{vectors: fallbackVectors()}
	cachePath := filepath.Join(t.TempDir(), "embeddings-test.jsonl")
	st := &embedState{}

	fallbackExamples("99 Taxi Centro", fallbackPool(), emb, cachePath, 2, st)
	callsAfterFirst := emb.calls // 3 pool items + 1 query
	fallbackExamples("99 Taxi Centro", fallbackPool(), emb, cachePath, 2, st)

	assert.Equal(t, 4, callsAfterFirst)
	assert.Equal(t, 5, emb.calls, "second call must only embed the query, not the pool")
}

func TestFallbackExamples_DegradesToNilOnReconcileError(t *testing.T) {
	emb := &countingEmbedder{err: errors.New("ollama down")}
	cachePath := filepath.Join(t.TempDir(), "embeddings-test.jsonl")
	st := &embedState{}

	got := fallbackExamples("99 Taxi Centro", fallbackPool(), emb, cachePath, 2, st)

	assert.Nil(t, got)
}

func TestFallbackExamples_DegradesToNilOnQueryEmbedError(t *testing.T) {
	emb := &countingEmbedder{vectors: fallbackVectors()}
	cachePath := filepath.Join(t.TempDir(), "embeddings-test.jsonl")
	st := &embedState{}
	// Prime the store successfully, then break the embedder for the query.
	fallbackExamples("99 Taxi Centro", fallbackPool(), emb, cachePath, 2, st)
	emb.err = errors.New("ollama down")

	got := fallbackExamples("99 Taxi Centro", fallbackPool(), emb, cachePath, 2, st)

	assert.Nil(t, got)
}

func TestEmbeddingFallback_DisabledByConfig(t *testing.T) {
	cfg := Config{DataDir: t.TempDir(), NoEmbedRetrieval: true}
	st := &embedState{}

	got := embeddingFallback("99 Taxi Centro", fallbackPool(), cfg, st)

	assert.Nil(t, got)
}

func TestEmbeddingFallback_NilWithoutDataDir(t *testing.T) {
	cfg := Config{}
	st := &embedState{}

	got := embeddingFallback("99 Taxi Centro", fallbackPool(), cfg, st)

	assert.Nil(t, got)
}
