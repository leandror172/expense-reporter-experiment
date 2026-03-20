package classifier

import (
	"bytes"
	"encoding/json"
	"expense-reporter/internal/logger"
	"fmt"
	"net/http"
	"os"
	"sort"
	"strings"
)

// Result is a single classification candidate.
type Result struct {
	Subcategory string
	Category    string
	Confidence  float64
}

// Config controls classifier behaviour.
type Config struct {
	OllamaURL    string // default: http://localhost:11434
	Model        string // default: my-classifier-q3
	DataDir      string // path to data/classification/
	FeedbackPath string // path to classifications.jsonl (optional; skipped when empty)
	TopN         int    // number of candidates to return (default: 3)
}

// Taxonomy maps subcategory name → parent category name.
type Taxonomy map[string]string

// LoadTaxonomy reads category_mapping from feature_dictionary_enhanced.json.
func LoadTaxonomy(dataDir string) (Taxonomy, error) {
	path := dataDir + "/feature_dictionary_enhanced.json"
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading feature dictionary: %w", err)
	}
	var dict struct {
		CategoryMapping map[string]string `json:"category_mapping"`
	}
	if err := json.Unmarshal(data, &dict); err != nil {
		return nil, fmt.Errorf("parsing feature dictionary: %w", err)
	}
	return Taxonomy(dict.CategoryMapping), nil
}

// Classify sends item/value/date to Ollama and returns top-N subcategory candidates.
// date must be in DD/MM format.
// When cfg.DataDir is set, few-shot examples are loaded and injected into the prompt.
func Classify(item string, value float64, date string, taxonomy Taxonomy, cfg Config) ([]Result, error) {
	if cfg.OllamaURL == "" {
		cfg.OllamaURL = "http://localhost:11434"
	}
	if cfg.Model == "" {
		cfg.Model = "my-classifier-q3"
	}
	if cfg.TopN <= 0 {
		cfg.TopN = 3
	}

	// Load and select few-shot examples when a data directory is configured.
	var examples []Example
	if cfg.DataDir != "" {
		keywords, err := LoadKeywordIndex(cfg.DataDir)
		if err != nil {
			logger.Debug("few-shot: keyword index unavailable", "err", err)
		} else {
			training, _ := LoadTrainingExamples(cfg.DataDir)
			var feedback []Example
			if cfg.FeedbackPath != "" {
				feedback, _ = LoadFeedbackExamples(cfg.FeedbackPath)
			}
			pool := MergeExamplePools(training, feedback)
			examples = SelectExamples(item, pool, keywords, 5)
			logger.Debug("few-shot", "count", len(examples), "item", item)
		}
	}

	body, err := buildRequest(item, value, date, taxonomy, examples, cfg)
	if err != nil {
		return nil, err
	}

	resp, err := http.Post(cfg.OllamaURL+"/api/chat", "application/json", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("calling Ollama: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Ollama returned status %d", resp.StatusCode)
	}

	return parseResponse(resp.Body, cfg.TopN)
}

// --- internal types ---

type ollamaRequest struct {
	Model    string          `json:"model"`
	Stream   bool            `json:"stream"`
	Format   json.RawMessage `json:"format"`
	Messages []ollamaMessage `json:"messages"`
}

type ollamaMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ollamaResponse struct {
	Message struct {
		Content string `json:"content"`
	} `json:"message"`
}

type classifyResponse struct {
	Results []struct {
		Subcategory string  `json:"subcategory"`
		Category    string  `json:"category"`
		Confidence  float64 `json:"confidence"`
	} `json:"results"`
}

// responseSchema is the JSON schema passed to Ollama's format param.
var responseSchema = json.RawMessage(`{
	"type": "object",
	"properties": {
		"results": {
			"type": "array",
			"items": {
				"type": "object",
				"properties": {
					"subcategory": {"type": "string"},
					"category":    {"type": "string"},
					"confidence":  {"type": "number"}
				},
				"required": ["subcategory", "category", "confidence"]
			}
		}
	},
	"required": ["results"]
}`)

func buildRequest(item string, value float64, date string, taxonomy Taxonomy, examples []Example, cfg Config) ([]byte, error) {
	messages := []ollamaMessage{
		{Role: "system", Content: buildSystemPrompt(taxonomy, cfg.TopN)},
	}
	messages = append(messages, formatExampleMessages(examples)...)
	messages = append(messages, ollamaMessage{
		Role:    "user",
		Content: fmt.Sprintf("item: %s\nvalue: %.2f\ndate: %s", item, value, date),
	})

	req := ollamaRequest{
		Model:    cfg.Model,
		Stream:   false,
		Format:   responseSchema,
		Messages: messages,
	}
	data, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshaling request: %w", err)
	}
	return data, nil
}

// formatExampleMessages converts examples into user/assistant message pairs for few-shot injection.
func formatExampleMessages(examples []Example) []ollamaMessage {
	if len(examples) == 0 {
		return nil
	}
	msgs := make([]ollamaMessage, 0, len(examples)*2)
	for _, ex := range examples {
		userContent := fmt.Sprintf("item: %s\nvalue: %.2f\ndate: %s", ex.Item, ex.Value, ex.Date)
		// Synthetic assistant response matching the response schema with high confidence.
		assistantContent := fmt.Sprintf(
			`{"results":[{"subcategory":%q,"category":%q,"confidence":0.95}]}`,
			ex.Subcategory, ex.Category,
		)
		msgs = append(msgs,
			ollamaMessage{Role: "user", Content: userContent},
			ollamaMessage{Role: "assistant", Content: assistantContent},
		)
	}
	return msgs
}

func parseResponse(body interface{ Read([]byte) (int, error) }, topN int) ([]Result, error) {
	var ollamaResp ollamaResponse
	if err := json.NewDecoder(body).Decode(&ollamaResp); err != nil {
		return nil, fmt.Errorf("decoding Ollama response: %w", err)
	}

	var classified classifyResponse
	if err := json.Unmarshal([]byte(ollamaResp.Message.Content), &classified); err != nil {
		return nil, fmt.Errorf("parsing classification JSON: %w", err)
	}

	sort.Slice(classified.Results, func(i, j int) bool {
		return classified.Results[i].Confidence > classified.Results[j].Confidence
	})

	results := make([]Result, 0, topN)
	for i, r := range classified.Results {
		if i >= topN {
			break
		}
		results = append(results, Result{
			Subcategory: r.Subcategory,
			Category:    r.Category,
			Confidence:  r.Confidence,
		})
	}
	return results, nil
}

func buildSystemPrompt(taxonomy Taxonomy, topN int) string {
	type entry struct{ sub, cat string }
	entries := make([]entry, 0, len(taxonomy))
	for sub, cat := range taxonomy {
		entries = append(entries, entry{sub, cat})
	}
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].cat != entries[j].cat {
			return entries[i].cat < entries[j].cat
		}
		return entries[i].sub < entries[j].sub
	})

	var sb strings.Builder
	sb.WriteString("You are an expense classifier for Brazilian personal finance.\n")
	sb.WriteString("Classify the given expense into one of the known subcategories below.\n")
	sb.WriteString(fmt.Sprintf("Return exactly %d candidates ranked by confidence (highest first).\n", topN))
	sb.WriteString("Confidence is a float between 0.0 and 1.0.\n\n")
	sb.WriteString("Known subcategories (subcategory → category):\n")
	for _, e := range entries {
		sb.WriteString(fmt.Sprintf("  %s → %s\n", e.sub, e.cat))
	}
	return sb.String()
}
