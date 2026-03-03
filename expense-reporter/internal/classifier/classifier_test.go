package classifier

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// --- LoadTaxonomy ---

func TestLoadTaxonomy(t *testing.T) {
	tests := []struct {
		name    string
		content string
		wantLen int
		wantErr bool
	}{
		{
			name: "valid file",
			content: `{
				"category_mapping": {
					"Diarista": "Habitação",
					"Supermercado": "Alimentação",
					"Uber/Taxi": "Transporte"
				}
			}`,
			wantLen: 3,
		},
		{
			name:    "invalid json",
			content: `not json`,
			wantErr: true,
		},
		{
			name:    "empty category_mapping",
			content: `{"category_mapping": {}}`,
			wantLen: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "feature_dictionary_enhanced.json")
			if err := os.WriteFile(path, []byte(tt.content), 0644); err != nil {
				t.Fatal(err)
			}

			got, err := LoadTaxonomy(dir)
			if (err != nil) != tt.wantErr {
				t.Fatalf("LoadTaxonomy() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err == nil && len(got) != tt.wantLen {
				t.Errorf("LoadTaxonomy() len = %d, want %d", len(got), tt.wantLen)
			}
		})
	}
}

func TestLoadTaxonomy_FileNotFound(t *testing.T) {
	_, err := LoadTaxonomy("/nonexistent/path")
	if err == nil {
		t.Error("expected error for missing file, got nil")
	}
}

// --- buildSystemPrompt ---

func TestBuildSystemPrompt(t *testing.T) {
	taxonomy := Taxonomy{
		"Diarista":     "Habitação",
		"Supermercado": "Alimentação",
	}
	prompt := buildSystemPrompt(taxonomy, 3)

	if !strings.Contains(prompt, "Brazilian personal finance") {
		t.Error("prompt missing domain context")
	}
	if !strings.Contains(prompt, "3 candidates") {
		t.Error("prompt missing topN instruction")
	}
	if !strings.Contains(prompt, "Diarista") || !strings.Contains(prompt, "Habitação") {
		t.Error("prompt missing taxonomy entries")
	}
	if !strings.Contains(prompt, "Supermercado") || !strings.Contains(prompt, "Alimentação") {
		t.Error("prompt missing taxonomy entries")
	}
}

func TestBuildSystemPrompt_Sorted(t *testing.T) {
	taxonomy := Taxonomy{
		"Cerveja":      "Lazer",
		"Supermercado": "Alimentação",
		"Diarista":     "Habitação",
	}
	prompt := buildSystemPrompt(taxonomy, 3)

	// Categories should appear in alphabetical order
	alimentacaoPos := strings.Index(prompt, "Alimentação")
	habitacaoPos := strings.Index(prompt, "Habitação")
	lazerPos := strings.Index(prompt, "Lazer")

	if alimentacaoPos > habitacaoPos || habitacaoPos > lazerPos {
		t.Error("taxonomy entries not sorted by category")
	}
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
				"message": map[string]string{
					"content": responseContent,
				},
			}
			json.NewEncoder(w).Encode(resp)
		}
	}
}

func testTaxonomy() Taxonomy {
	return Taxonomy{
		"Diarista":     "Habitação",
		"Supermercado": "Alimentação",
		"Uber/Taxi":    "Transporte",
	}
}

func TestClassify_HappyPath(t *testing.T) {
	responseContent := `{
		"results": [
			{"subcategory": "Uber/Taxi",    "category": "Transporte",  "confidence": 0.92},
			{"subcategory": "Diarista",     "category": "Habitação",   "confidence": 0.05},
			{"subcategory": "Supermercado", "category": "Alimentação", "confidence": 0.03}
		]
	}`
	srv := httptest.NewServer(ollamaHandler(responseContent, http.StatusOK))
	defer srv.Close()

	cfg := Config{OllamaURL: srv.URL, Model: "test-model", TopN: 3}
	results, err := Classify("Uber Centro", 35.50, "15/04", testTaxonomy(), cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(results))
	}
	if results[0].Subcategory != "Uber/Taxi" {
		t.Errorf("expected top result Uber/Taxi, got %s", results[0].Subcategory)
	}
	if results[0].Confidence != 0.92 {
		t.Errorf("expected confidence 0.92, got %f", results[0].Confidence)
	}
}

func TestClassify_TopNCap(t *testing.T) {
	responseContent := `{
		"results": [
			{"subcategory": "Uber/Taxi",    "category": "Transporte",  "confidence": 0.90},
			{"subcategory": "Diarista",     "category": "Habitação",   "confidence": 0.07},
			{"subcategory": "Supermercado", "category": "Alimentação", "confidence": 0.03}
		]
	}`
	srv := httptest.NewServer(ollamaHandler(responseContent, http.StatusOK))
	defer srv.Close()

	cfg := Config{OllamaURL: srv.URL, Model: "test-model", TopN: 1}
	results, err := Classify("Uber", 35.50, "15/04", testTaxonomy(), cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 1 {
		t.Errorf("expected 1 result (TopN=1), got %d", len(results))
	}
}

func TestClassify_SortedByConfidenceDesc(t *testing.T) {
	// Model returns results in wrong order — we must sort them
	responseContent := `{
		"results": [
			{"subcategory": "Supermercado", "category": "Alimentação", "confidence": 0.03},
			{"subcategory": "Uber/Taxi",    "category": "Transporte",  "confidence": 0.90},
			{"subcategory": "Diarista",     "category": "Habitação",   "confidence": 0.07}
		]
	}`
	srv := httptest.NewServer(ollamaHandler(responseContent, http.StatusOK))
	defer srv.Close()

	cfg := Config{OllamaURL: srv.URL, Model: "test-model", TopN: 3}
	results, err := Classify("something", 50.00, "10/03", testTaxonomy(), cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for i := 1; i < len(results); i++ {
		if results[i].Confidence > results[i-1].Confidence {
			t.Errorf("results not sorted: results[%d].Confidence %f > results[%d].Confidence %f",
				i, results[i].Confidence, i-1, results[i-1].Confidence)
		}
	}
}

func TestClassify_DefaultConfig(t *testing.T) {
	responseContent := `{"results": [{"subcategory": "Diarista", "category": "Habitação", "confidence": 0.9}]}`
	srv := httptest.NewServer(ollamaHandler(responseContent, http.StatusOK))
	defer srv.Close()

	// Zero-value Config — defaults should be applied
	cfg := Config{OllamaURL: srv.URL}
	results, err := Classify("Diarista Leticia", 160.00, "05/01", testTaxonomy(), cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) == 0 {
		t.Error("expected at least one result with default config")
	}
}

func TestClassify_OllamaError(t *testing.T) {
	srv := httptest.NewServer(ollamaHandler("", http.StatusInternalServerError))
	defer srv.Close()

	cfg := Config{OllamaURL: srv.URL, Model: "test-model", TopN: 3}
	_, err := Classify("item", 10.00, "01/01", testTaxonomy(), cfg)
	if err == nil {
		t.Error("expected error for non-200 response, got nil")
	}
}

func TestClassify_MalformedContent(t *testing.T) {
	// Ollama returns 200 but message.content is not valid JSON
	responseContent := `not valid json`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]any{
			"message": map[string]string{"content": responseContent},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	cfg := Config{OllamaURL: srv.URL, Model: "test-model", TopN: 3}
	_, err := Classify("item", 10.00, "01/01", testTaxonomy(), cfg)
	if err == nil {
		t.Error("expected error for malformed content JSON, got nil")
	}
}

func TestClassify_NetworkError(t *testing.T) {
	cfg := Config{OllamaURL: "http://127.0.0.1:1", Model: "test-model", TopN: 3}
	_, err := Classify("item", 10.00, "01/01", testTaxonomy(), cfg)
	if err == nil {
		t.Error("expected error for unreachable Ollama, got nil")
	}
}
