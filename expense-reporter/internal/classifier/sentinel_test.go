package classifier

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	taxonomy "expense-reporter/internal/taxonomy"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildSystemPrompt_MentionsSentinel(t *testing.T) {
	prompt := buildSystemPrompt(testSheets(), 3)
	assert.Contains(t, prompt, SentinelPath)
}

func TestClassify_RequestEnumIncludesSentinel(t *testing.T) {
	var capturedBody []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedBody, _ = io.ReadAll(r.Body)
		defer r.Body.Close()

		w.WriteHeader(http.StatusOK)
		resp := map[string]any{
			"message": map[string]string{"content": `{"results":[{"path":"Variáveis/Transporte/Uber/Taxi","confidence":0.9}]}`},
		}
		json.NewEncoder(w).Encode(resp) //nolint:errcheck
	}))
	defer srv.Close()

	cfg := Config{OllamaURL: srv.URL, Model: "test-model", TopN: 3}
	_, err := Classify("item", 10.00, "01/01", testSheets(), cfg)
	require.NoError(t, err)

	assert.Contains(t, string(capturedBody), SentinelPath)
}

func TestClassify_SentinelMapsToDiversosWithFlooredConfidence(t *testing.T) {
	sheets := append(testSheets(), taxonomy.ExpenseType{
		Name: "Adicionais",
		Cats: []taxonomy.Category{
			{Name: "Outros", Subs: []taxonomy.Subcat{{Name: "Diversos"}}},
		},
	})

	srv := httptest.NewServer(ollamaHandler(`{"results":[{"path":"NENHUMA DAS OPÇÕES","confidence":0.95},{"path":"Variáveis/Transporte/Uber/Taxi","confidence":0.40}]}`, http.StatusOK))
	defer srv.Close()

	cfg := Config{OllamaURL: srv.URL, Model: "test-model", TopN: 3}
	results, err := Classify("item", 10.00, "01/01", sheets, cfg)
	require.NoError(t, err)
	assert.Len(t, results, 2)

	// First result should be Uber/Taxi (higher confidence)
	assert.Equal(t, "Variáveis", results[0].Type)
	assert.Equal(t, "Transporte", results[0].Category)
	assert.Equal(t, "Uber/Taxi", results[0].Subcategory)
	assert.Equal(t, 0.40, results[0].Confidence)

	// Second result should be Diversos with floored confidence
	assert.Equal(t, "Adicionais", results[1].Type)
	assert.Equal(t, "Outros", results[1].Category)
	assert.Equal(t, "Diversos", results[1].Subcategory)
	assert.Equal(t, 0.30, results[1].Confidence)
}

func TestClassify_SentinelDroppedWhenNoDiversosLeaf(t *testing.T) {
	srv := httptest.NewServer(ollamaHandler(`{"results":[{"path":"NENHUMA DAS OPÇÕES","confidence":0.95},{"path":"Variáveis/Transporte/Uber/Taxi","confidence":0.40}]}`, http.StatusOK))
	defer srv.Close()

	cfg := Config{OllamaURL: srv.URL, Model: "test-model", TopN: 3}
	results, err := Classify("item", 10.00, "01/01", testSheets(), cfg)
	require.NoError(t, err)
	assert.Len(t, results, 1)

	// Only Uber/Taxi should remain
	assert.Equal(t, "Variáveis", results[0].Type)
	assert.Equal(t, "Transporte", results[0].Category)
	assert.Equal(t, "Uber/Taxi", results[0].Subcategory)
}
