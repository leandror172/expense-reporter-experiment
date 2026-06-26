package classifier

import (
	"encoding/json"
	taxonomy "expense-reporter/internal/taxonomy"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testSheets is a small taxonomy tree used across the classifier tests. It includes
// a slash-containing subcategory (Uber/Taxi) to prove paths round-trip via the map.
func testSheets() []taxonomy.ExpenseType {
	return []taxonomy.ExpenseType{
		{Name: "Variáveis", Cats: []taxonomy.Category{
			{Name: "Transporte", Subs: []taxonomy.Subcat{{Name: "Uber/Taxi"}}},
			{Name: "Alimentação", Subs: []taxonomy.Subcat{{Name: "Supermercado"}}},
		}},
		{Name: "Fixas", Cats: []taxonomy.Category{
			{Name: "Habitação", Subs: []taxonomy.Subcat{{Name: "Diarista"}}},
		}},
	}
}

// --- buildSystemPrompt ---

func TestBuildSystemPrompt_RendersTree(t *testing.T) {
	prompt := buildSystemPrompt(testSheets(), 3)

	assert.Contains(t, prompt, "Brazilian personal finance")
	assert.Contains(t, prompt, "3 candidates")
	assert.Contains(t, prompt, "Type/Category/Subcategory")
	// Type headers and their subcategories appear.
	assert.Contains(t, prompt, "Variáveis:")
	assert.Contains(t, prompt, "Transporte: Uber/Taxi")
	assert.Contains(t, prompt, "Fixas:")
	assert.Contains(t, prompt, "Habitação: Diarista")
}

// --- buildResponseSchema ---

func TestBuildResponseSchema_EmbedsEnumAndRequiresPath(t *testing.T) {
	enum, err := taxonomy.PathEnum(testSheets())
	require.NoError(t, err)

	raw := buildResponseSchema(enum)

	var schema map[string]any
	require.NoError(t, json.Unmarshal(raw, &schema))

	// Drill into results.items.properties.path.enum and confirm the paths are embedded.
	results := schema["properties"].(map[string]any)["results"].(map[string]any)
	item := results["items"].(map[string]any)
	props := item["properties"].(map[string]any)
	pathSchema := props["path"].(map[string]any)

	assert.Equal(t, "string", pathSchema["type"])
	assert.ElementsMatch(t, []string{"path", "confidence"}, toStringSlice(item["required"]))
	assert.Contains(t, toStringSlice(pathSchema["enum"]), "Variáveis/Transporte/Uber/Taxi")
}

func toStringSlice(v any) []string {
	raw, ok := v.([]any)
	if !ok {
		return nil
	}
	out := make([]string, len(raw))
	for i, e := range raw {
		out[i], _ = e.(string)
	}
	return out
}

// --- Classify (with mock Ollama server) ---

func ollamaHandler(responseContent string, statusCode int) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		w.WriteHeader(statusCode)
		if statusCode == http.StatusOK {
			resp := map[string]any{
				"message": map[string]string{"content": responseContent},
			}
			json.NewEncoder(w).Encode(resp) //nolint:errcheck
		}
	}
}

func TestClassify_HappyPathSplitsPredictedPath(t *testing.T) {
	responseContent := `{
		"results": [
			{"path": "Variáveis/Transporte/Uber/Taxi", "confidence": 0.92},
			{"path": "Fixas/Habitação/Diarista",       "confidence": 0.05},
			{"path": "Variáveis/Alimentação/Supermercado", "confidence": 0.03}
		]
	}`
	srv := httptest.NewServer(ollamaHandler(responseContent, http.StatusOK))
	defer srv.Close()

	cfg := Config{OllamaURL: srv.URL, Model: "test-model", TopN: 3}
	results, err := Classify("Uber Centro", 35.50, "15/04", testSheets(), cfg)
	require.NoError(t, err)
	require.Len(t, results, 3)

	top := results[0]
	assert.Equal(t, "Variáveis", top.Type)
	assert.Equal(t, "Transporte", top.Category)
	assert.Equal(t, "Uber/Taxi", top.Subcategory)
	assert.Equal(t, 0.92, top.Confidence)
}

func TestClassify_TopNCap(t *testing.T) {
	responseContent := `{
		"results": [
			{"path": "Variáveis/Transporte/Uber/Taxi", "confidence": 0.90},
			{"path": "Fixas/Habitação/Diarista",       "confidence": 0.07},
			{"path": "Variáveis/Alimentação/Supermercado", "confidence": 0.03}
		]
	}`
	srv := httptest.NewServer(ollamaHandler(responseContent, http.StatusOK))
	defer srv.Close()

	cfg := Config{OllamaURL: srv.URL, Model: "test-model", TopN: 1}
	results, err := Classify("Uber", 35.50, "15/04", testSheets(), cfg)
	require.NoError(t, err)
	assert.Len(t, results, 1)
}

func TestClassify_SortedByConfidenceDesc(t *testing.T) {
	responseContent := `{
		"results": [
			{"path": "Variáveis/Alimentação/Supermercado", "confidence": 0.03},
			{"path": "Variáveis/Transporte/Uber/Taxi", "confidence": 0.90},
			{"path": "Fixas/Habitação/Diarista",       "confidence": 0.07}
		]
	}`
	srv := httptest.NewServer(ollamaHandler(responseContent, http.StatusOK))
	defer srv.Close()

	cfg := Config{OllamaURL: srv.URL, Model: "test-model", TopN: 3}
	results, err := Classify("something", 50.00, "10/03", testSheets(), cfg)
	require.NoError(t, err)
	for i := 1; i < len(results); i++ {
		assert.LessOrEqual(t, results[i].Confidence, results[i-1].Confidence)
	}
}

func TestClassify_DropsOffEnumPath(t *testing.T) {
	// The model violates the enum and returns a path not in the taxonomy: that
	// candidate must be dropped, not turned into a half-populated Result.
	responseContent := `{
		"results": [
			{"path": "Variáveis/Transporte/Uber/Taxi", "confidence": 0.80},
			{"path": "Imaginário/Inventado/Fantasma",  "confidence": 0.95}
		]
	}`
	srv := httptest.NewServer(ollamaHandler(responseContent, http.StatusOK))
	defer srv.Close()

	cfg := Config{OllamaURL: srv.URL, Model: "test-model", TopN: 3}
	results, err := Classify("Uber", 20.00, "01/02", testSheets(), cfg)
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, "Uber/Taxi", results[0].Subcategory)
}

func TestClassify_DefaultConfig(t *testing.T) {
	responseContent := `{"results": [{"path": "Fixas/Habitação/Diarista", "confidence": 0.9}]}`
	srv := httptest.NewServer(ollamaHandler(responseContent, http.StatusOK))
	defer srv.Close()

	cfg := Config{OllamaURL: srv.URL} // zero-value: defaults applied
	results, err := Classify("Diarista Leticia", 160.00, "05/01", testSheets(), cfg)
	require.NoError(t, err)
	assert.NotEmpty(t, results)
}

func TestClassify_OllamaError(t *testing.T) {
	srv := httptest.NewServer(ollamaHandler("", http.StatusInternalServerError))
	defer srv.Close()

	cfg := Config{OllamaURL: srv.URL, Model: "test-model", TopN: 3}
	_, err := Classify("item", 10.00, "01/01", testSheets(), cfg)
	assert.Error(t, err)
}

func TestClassify_MalformedContent(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]any{"message": map[string]string{"content": "not valid json"}}
		json.NewEncoder(w).Encode(resp) //nolint:errcheck
	}))
	defer srv.Close()

	cfg := Config{OllamaURL: srv.URL, Model: "test-model", TopN: 3}
	_, err := Classify("item", 10.00, "01/01", testSheets(), cfg)
	assert.Error(t, err)
}

func TestClassify_NetworkError(t *testing.T) {
	cfg := Config{OllamaURL: "http://127.0.0.1:1", Model: "test-model", TopN: 3}
	_, err := Classify("item", 10.00, "01/01", testSheets(), cfg)
	assert.Error(t, err)
}

// --- resolveExamplePaths ---

func TestResolveExamplePaths_ResolvesAndDrops(t *testing.T) {
	sheets := testSheets()
	pm, err := taxonomy.BuildPathMap(sheets)
	require.NoError(t, err)

	examples := []Example{
		{Item: "Uber Centro", Value: 25.5, Date: "15/04", Subcategory: "Uber/Taxi"}, // resolves
		{Item: "Mystery", Value: 9.9, Date: "01/01", Subcategory: "DoesNotExist"},   // dropped (absent)
	}
	got := resolveExamplePaths(examples, sheets, pm)
	require.Len(t, got, 1)
	assert.Equal(t, "Variáveis/Transporte/Uber/Taxi", got[0].Path)
	assert.Equal(t, "Uber Centro", got[0].Item)
}

// --- formatExampleMessages ---

func TestFormatExampleMessages(t *testing.T) {
	ex := fewShotExample{Item: "Uber Centro", Value: 25.50, Date: "15/04", Path: "Variáveis/Transporte/Uber/Taxi"}

	t.Run("nil examples", func(t *testing.T) {
		assert.Nil(t, formatExampleMessages(nil))
	})

	t.Run("single example", func(t *testing.T) {
		msgs := formatExampleMessages([]fewShotExample{ex})
		require.Len(t, msgs, 2)

		assert.Equal(t, "user", msgs[0].Role)
		assert.Contains(t, msgs[0].Content, "item: Uber Centro")
		assert.Contains(t, msgs[0].Content, "value: 25.50")
		assert.Contains(t, msgs[0].Content, "date: 15/04")

		assert.Equal(t, "assistant", msgs[1].Role)
		assert.Contains(t, msgs[1].Content, `"path":"Variáveis/Transporte/Uber/Taxi"`)
		assert.Contains(t, msgs[1].Content, `"confidence":0.95`)
	})
}

// --- buildRequest ---

func TestBuildRequest_FewShot(t *testing.T) {
	sheets := testSheets()
	enum, err := taxonomy.PathEnum(sheets)
	require.NoError(t, err)
	cfg := Config{Model: "my-test-model", TopN: 3}
	ex := fewShotExample{Item: "Uber Centro", Value: 25.50, Date: "15/04", Path: "Variáveis/Transporte/Uber/Taxi"}

	t.Run("no examples", func(t *testing.T) {
		body, err := buildRequest("Actual Item", 100.0, "20/05", sheets, enum, nil, cfg)
		require.NoError(t, err)

		var req ollamaRequest
		require.NoError(t, json.Unmarshal(body, &req))
		require.Len(t, req.Messages, 2)
		assert.Equal(t, "system", req.Messages[0].Role)
		assert.Equal(t, "user", req.Messages[1].Role)
		assert.Contains(t, req.Messages[1].Content, "item: Actual Item")
		assert.False(t, req.Stream)
		assert.Equal(t, "my-test-model", req.Model)
		assert.Contains(t, string(req.Format), "enum")
	})

	t.Run("two examples", func(t *testing.T) {
		body, err := buildRequest("Actual Item", 100.0, "20/05", sheets, enum, []fewShotExample{ex, ex}, cfg)
		require.NoError(t, err)

		var req ollamaRequest
		require.NoError(t, json.Unmarshal(body, &req))
		require.Len(t, req.Messages, 6)
		assert.Equal(t, "system", req.Messages[0].Role)
		assert.Equal(t, "assistant", req.Messages[2].Role)
		assert.Contains(t, req.Messages[5].Content, "item: Actual Item")
		assert.Contains(t, req.Messages[2].Content, `"confidence":0.95`)
	})
}

// guard against accidental removal of the slash-in-name round-trip guarantee.
func TestClassify_PathWithSlashSubcategoryRoundTrips(t *testing.T) {
	assert.True(t, strings.Contains("Variáveis/Transporte/Uber/Taxi", "Uber/Taxi"))
}
