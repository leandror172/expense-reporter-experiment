package classifier

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeEmbedder is a deterministic Embedder for tests — no Ollama.
type fakeEmbedder struct {
	vectors map[string][]float64
	calls   []string
	err     error
}

func (f *fakeEmbedder) Embed(text string) ([]float64, error) {
	f.calls = append(f.calls, text)
	if f.err != nil {
		return nil, f.err
	}
	v, ok := f.vectors[text]
	if !ok {
		return nil, fmt.Errorf("fakeEmbedder: no vector for %q", text)
	}
	return v, nil
}

func writeCacheLines(t *testing.T, path string, lines []string) {
	t.Helper()
	require.NoError(t, os.WriteFile(path, []byte(strings.Join(lines, "\n")+"\n"), 0o644))
}

func TestEmbeddingCachePath_SanitizesModelName(t *testing.T) {
	assert.Equal(t, filepath.Join("/data", "embeddings-snowflake-arctic-embed2.jsonl"),
		EmbeddingCachePath("/data", "snowflake-arctic-embed2"))
	// ":" and "/" in the model name must be sanitized so the path stays a single file.
	assert.Equal(t, filepath.Join("/data", "embeddings-qwen3-embedding-8b.jsonl"),
		EmbeddingCachePath("/data", "qwen3-embedding:8b"))
}

func TestLoadEmbeddingCache_RoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "c.jsonl")
	writeCacheLines(t, path, []string{
		`{"item":"Uber","vector":[0.1,0.2,0.3]}`,
		`{"item":"Netflix","vector":[0.4,0.5,0.6]}`,
	})

	store, err := LoadEmbeddingCache(path)
	require.NoError(t, err)
	require.Len(t, store, 2)
	assert.Equal(t, []float64{0.1, 0.2, 0.3}, store["Uber"])
	assert.Equal(t, []float64{0.4, 0.5, 0.6}, store["Netflix"])
}

func TestLoadEmbeddingCache_MissingFileReturnsEmpty(t *testing.T) {
	// A cold start (no cache yet) must not be an error — reconcile builds it.
	store, err := LoadEmbeddingCache(filepath.Join(t.TempDir(), "nope.jsonl"))
	require.NoError(t, err)
	assert.Empty(t, store)
}

func TestLoadEmbeddingCache_SkipsTornLastLine(t *testing.T) {
	// A process killed mid-append leaves a truncated final line. Skip it (it gets
	// re-embedded on the next reconcile); the earlier good lines still load.
	path := filepath.Join(t.TempDir(), "c.jsonl")
	require.NoError(t, os.WriteFile(path,
		[]byte(`{"item":"Uber","vector":[0.1,0.2]}`+"\n"+`{"item":"Netfl`), 0o644))

	store, err := LoadEmbeddingCache(path)
	require.NoError(t, err)
	require.Len(t, store, 1)
	assert.Equal(t, []float64{0.1, 0.2}, store["Uber"])
}

func TestLoadEmbeddingCache_DimensionMismatchFailsLoud(t *testing.T) {
	// Inconsistent vector dimensions within a file would produce garbage cosines —
	// fail loud (return an error), never silently.
	path := filepath.Join(t.TempDir(), "c.jsonl")
	writeCacheLines(t, path, []string{
		`{"item":"Uber","vector":[0.1,0.2,0.3]}`,
		`{"item":"Netflix","vector":[0.4,0.5]}`,
	})

	_, err := LoadEmbeddingCache(path)
	require.Error(t, err)
}

func TestReconcileEmbeddings_EmbedsOnlyMissing(t *testing.T) {
	path := filepath.Join(t.TempDir(), "embeddings-fake.jsonl")
	writeCacheLines(t, path, []string{`{"item":"Uber","vector":[1,0]}`})

	emb := &fakeEmbedder{vectors: map[string][]float64{"Netflix": {0, 1}}}
	store, err := ReconcileEmbeddings(path, []string{"Uber", "Netflix"}, emb)
	require.NoError(t, err)

	// Only the missing item is embedded; the cached one is reused.
	assert.Equal(t, []string{"Netflix"}, emb.calls)
	assert.Equal(t, []float64{1, 0}, store["Uber"])
	assert.Equal(t, []float64{0, 1}, store["Netflix"])

	// The new embedding is persisted (appended) — a reload sees both.
	reloaded, err := LoadEmbeddingCache(path)
	require.NoError(t, err)
	require.Len(t, reloaded, 2)
	assert.Equal(t, []float64{0, 1}, reloaded["Netflix"])
}

func TestReconcileEmbeddings_EmbedsEachUniqueTextOnce(t *testing.T) {
	// One line per unique raw item text (D3/D6) — duplicates in the input embed once.
	path := filepath.Join(t.TempDir(), "e.jsonl")
	emb := &fakeEmbedder{vectors: map[string][]float64{"A": {1, 1}, "B": {2, 2}}}

	_, err := ReconcileEmbeddings(path, []string{"A", "B", "A"}, emb)
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"A", "B"}, emb.calls)

	reloaded, err := LoadEmbeddingCache(path)
	require.NoError(t, err)
	assert.Len(t, reloaded, 2)
}

func TestReconcileEmbeddings_WritesLowercaseJSONKeys(t *testing.T) {
	// D3/D6 mandate the exact on-disk shape {"item":...,"vector":...}. LoadEmbeddingCache's
	// unmarshal is case-insensitive, so assert the literal bytes to catch a regression to
	// capitalized Go field names.
	path := filepath.Join(t.TempDir(), "e.jsonl")
	emb := &fakeEmbedder{vectors: map[string][]float64{"A": {1, 0}}}

	_, err := ReconcileEmbeddings(path, []string{"A"}, emb)
	require.NoError(t, err)

	raw, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, `{"item":"A","vector":[1,0]}`, strings.TrimSpace(string(raw)))
}

func TestReconcileEmbeddings_FromEmptyCache(t *testing.T) {
	path := filepath.Join(t.TempDir(), "e.jsonl") // no file yet
	emb := &fakeEmbedder{vectors: map[string][]float64{"A": {1, 0}, "B": {0, 1}}}

	store, err := ReconcileEmbeddings(path, []string{"A", "B"}, emb)
	require.NoError(t, err)
	require.Len(t, store, 2)
	assert.Equal(t, []float64{1, 0}, store["A"])
	assert.Equal(t, []float64{0, 1}, store["B"])
}

func TestReconcileEmbeddings_PropagatesEmbedderError(t *testing.T) {
	// Ollama down / model missing → error surfaces to the caller (which degrades, Phase 3).
	path := filepath.Join(t.TempDir(), "e.jsonl")
	emb := &fakeEmbedder{err: errors.New("ollama down")}

	_, err := ReconcileEmbeddings(path, []string{"A"}, emb)
	require.Error(t, err)
}

func TestOllamaEmbedder_Embed(t *testing.T) {
	var gotBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/embeddings", r.URL.Path)
		body, _ := io.ReadAll(r.Body)
		require.NoError(t, json.Unmarshal(body, &gotBody))
		_, _ = w.Write([]byte(`{"embedding":[0.5,0.25,0.125]}`))
	}))
	defer srv.Close()

	e := &OllamaEmbedder{URL: srv.URL, Model: "test-model"}
	vec, err := e.Embed("Uber Centro")
	require.NoError(t, err)
	assert.Equal(t, []float64{0.5, 0.25, 0.125}, vec)
	assert.Equal(t, "test-model", gotBody["model"])
	assert.Equal(t, "Uber Centro", gotBody["prompt"])
}

func TestOllamaEmbedder_Embed_NonOKStatusIsError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	e := &OllamaEmbedder{URL: srv.URL, Model: "m"}
	_, err := e.Embed("x")
	require.Error(t, err)
}
