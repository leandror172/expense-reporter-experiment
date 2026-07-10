package classifier

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Embedder is an interface for embedding text into vectors.
type Embedder interface {
	Embed(text string) ([]float64, error)
}

// OllamaEmbedder embeds text using the Ollama API.
type OllamaEmbedder struct {
	URL    string
	Model  string
	Client *http.Client
}

// Embed sends a request to the Ollama API to get an embedding for the given text.
func (e *OllamaEmbedder) Embed(text string) ([]float64, error) {
	if e.Client == nil {
		e.Client = http.DefaultClient
	}

	reqBody := map[string]any{
		"model":  e.Model,
		"prompt": text,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal request body: %w", err)
	}

	resp, err := e.Client.Post(e.URL+"/api/embeddings", "application/json", strings.NewReader(string(body)))
	if err != nil {
		return nil, fmt.Errorf("post request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("non-200 status code: %d", resp.StatusCode)
	}

	var result struct {
		Embedding []float64 `json:"embedding"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return result.Embedding, nil
}

// EmbeddingCachePath returns the path to the embedding cache file with sanitized model name.
func EmbeddingCachePath(dir, model string) string {
	sanitized := strings.ReplaceAll(strings.ReplaceAll(model, ":", "-"), "/", "-")
	return filepath.Join(dir, fmt.Sprintf("embeddings-%s.jsonl", sanitized))
}

// embeddingCacheLine is one JSONL record: the raw item text and its vector.
type embeddingCacheLine struct {
	Item   string    `json:"item"`
	Vector []float64 `json:"vector"`
}

// LoadEmbeddingCache reads a JSONL file of item vectors into a map.
func LoadEmbeddingCache(path string) (map[string][]float64, error) {
	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return make(map[string][]float64), nil
		}
		return nil, fmt.Errorf("open cache file: %w", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 1<<20), 1<<20) // 1MB buffer

	var (
		store       = make(map[string][]float64)
		vectorLen   int
		firstVector bool
	)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		var entry embeddingCacheLine
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			continue
		}

		if entry.Item == "" || len(entry.Vector) == 0 {
			continue
		}

		if !firstVector {
			vectorLen = len(entry.Vector)
			firstVector = true
		} else if len(entry.Vector) != vectorLen {
			return nil, fmt.Errorf("vector length mismatch: expected %d, got %d", vectorLen, len(entry.Vector))
		}

		store[entry.Item] = entry.Vector
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scan cache file: %w", err)
	}

	return store, nil
}

// ReconcileEmbeddings updates the embedding cache with new embeddings for missing items.
func ReconcileEmbeddings(path string, items []string, embedder Embedder) (map[string][]float64, error) {
	store, err := LoadEmbeddingCache(path)
	if err != nil {
		return nil, fmt.Errorf("load cache: %w", err)
	}

	missing := make([]string, 0)
	seen := make(map[string]bool)
	for _, item := range items {
		if _, exists := store[item]; exists || seen[item] {
			continue
		}
		seen[item] = true
		missing = append(missing, item)
	}
	sort.Strings(missing)

	filePath := path
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("mkdir all: %w", err)
	}

	file, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return nil, fmt.Errorf("open file: %w", err)
	}
	defer file.Close()

	for _, item := range missing {
		vec, err := embedder.Embed(item)
		if err != nil {
			return nil, fmt.Errorf("embed %q: %w", item, err)
		}

		store[item] = vec

		line, err := json.Marshal(embeddingCacheLine{Item: item, Vector: vec})
		if err != nil {
			return nil, fmt.Errorf("marshal line: %w", err)
		}

		if _, err := file.Write(append(line, '\n')); err != nil {
			return nil, fmt.Errorf("write line: %w", err)
		}
	}

	return store, nil
}
