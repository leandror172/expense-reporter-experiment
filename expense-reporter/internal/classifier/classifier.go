package classifier

import (
	"bytes"
	"encoding/json"
	"expense-reporter/internal/logger"
	taxonomy "expense-reporter/internal/taxonomy"
	"fmt"
	"net/http"
	"sort"
	"strings"
)

// Result is a single classification candidate. Since T-13 the model predicts a
// full taxonomy path, so a Result carries the expense Type alongside category and
// subcategory — every field comes from one validated path, never from independent
// lookups that could disagree.
type Result struct {
	Type        string
	Category    string
	Subcategory string
	Confidence  float64
}

// Config controls classifier behaviour.
type Config struct {
	OllamaURL    string // default: http://localhost:11434
	Model        string // default: my-classifier-q3 (fits VRAM; enum validity is grammar-enforced, not model-dependent)
	DataDir      string // path to data/classification/
	FeedbackPath string // path to classifications.jsonl (optional; skipped when empty)
	TopN         int    // number of candidates to return (default: 3)
}

// Classify sends item/value/date to Ollama and returns top-N full-path candidates.
// date must be in DD/MM format. sheets is the expense taxonomy tree
// (config/taxonomy.json), rendered into the prompt and used to constrain the model
// to valid Type/Category/Subcategory paths via a structured-output enum.
// When cfg.DataDir is set, few-shot examples are loaded and injected into the prompt.
func Classify(item string, value float64, date string, sheets []taxonomy.ExpenseType, cfg Config) ([]Result, error) {
	if cfg.OllamaURL == "" {
		cfg.OllamaURL = "http://localhost:11434"
	}
	if cfg.Model == "" {
		// q3 (qwen3:8b, ~5.2 GB) fits VRAM and is the proven baseline. Output
		// validity (every candidate is a valid 112-path enum member) is enforced by
		// Ollama's grammar-constrained decoding, so it does NOT depend on a larger
		// model — qcoder (qwen3-coder:30b, 20.7 GB) only added CPU-offload latency and
		// load-time 500s on a 12 GB GPU without buying validity. Accuracy across
		// q3/q35/qcoder is benchmarked in T-14.
		cfg.Model = "my-classifier-q3"
	}
	if cfg.TopN <= 0 {
		cfg.TopN = 3
	}

	pm, err := taxonomy.BuildPathMap(sheets)
	if err != nil {
		return nil, fmt.Errorf("building taxonomy path map: %w", err)
	}

	examples := resolveExamplePaths(selectExamples(item, cfg), sheets, pm)

	body, err := buildRequest(item, value, date, sheets, pm.Enum(), examples, cfg)
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

	return parseResponse(resp.Body, pm, cfg.TopN)
}

// selectExamples loads and selects the few-shot example pool for item. Returns nil
// when no data directory is configured or the keyword index is unavailable.
func selectExamples(item string, cfg Config) []Example {
	if cfg.DataDir == "" {
		return nil
	}
	keywords, err := LoadKeywordIndex(cfg.DataDir)
	if err != nil {
		logger.Debug("few-shot: keyword index unavailable", "err", err)
		return nil
	}
	training, _ := LoadTrainingExamples(cfg.DataDir)
	var feedback []Example
	if cfg.FeedbackPath != "" {
		feedback, _ = LoadFeedbackExamples(cfg.FeedbackPath)
	}
	pool := MergeExamplePools(training, feedback)
	examples := SelectExamples(item, pool, keywords, 5)
	logger.Debug("few-shot", "count", len(examples), "item", item)
	return examples
}

// fewShotExample is a training example whose subcategory has been resolved to a
// canonical full path, ready to render as a synthetic assistant message.
type fewShotExample struct {
	Item  string
	Value float64
	Date  string
	Path  string
}

// resolveExamplePaths attaches a canonical taxonomy path to each example and drops
// any that cannot be resolved (ambiguous leaf without a usable type hint, or a
// subcategory absent from the taxonomy). Resolving through ResolveLeaf + PathFor
// guarantees every rendered example uses a path that is actually in the enum — an
// example built from the training file's own category string could disagree with
// the taxonomy spelling and teach the model an off-enum answer.
func resolveExamplePaths(examples []Example, sheets []taxonomy.ExpenseType, pm taxonomy.PathMap) []fewShotExample {
	var out []fewShotExample
	for _, ex := range examples {
		typ, cat, err := taxonomy.ResolveLeaf(sheets, ex.Subcategory, ex.TypeHint)
		if err != nil {
			continue
		}
		path, ok := pm.PathFor(typ, cat, ex.Subcategory)
		if !ok {
			continue
		}
		out = append(out, fewShotExample{Item: ex.Item, Value: ex.Value, Date: ex.Date, Path: path})
	}
	return out
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

// classifyResponse is the structured payload the model returns: each candidate is
// one full taxonomy path plus a confidence.
type classifyResponse struct {
	Results []struct {
		Path       string  `json:"path"`
		Confidence float64 `json:"confidence"`
	} `json:"results"`
}

// buildResponseSchema constructs the Ollama format schema constraining each
// candidate's "path" to one of the enum members. Built by marshalling Go values so
// the enum slice is embedded safely (no string concatenation).
func buildResponseSchema(enum []string) json.RawMessage {
	pathSchema := map[string]any{"type": "string", "enum": enum}
	item := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"path":       pathSchema,
			"confidence": map[string]any{"type": "number"},
		},
		"required": []string{"path", "confidence"},
	}
	schema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"results": map[string]any{"type": "array", "items": item},
		},
		"required": []string{"results"},
	}
	data, _ := json.Marshal(schema) // map[string]any with string/[]string values cannot fail
	return data
}

func buildRequest(item string, value float64, date string, sheets []taxonomy.ExpenseType, enum []string, examples []fewShotExample, cfg Config) ([]byte, error) {
	messages := []ollamaMessage{
		{Role: "system", Content: buildSystemPrompt(sheets, cfg.TopN)},
	}
	messages = append(messages, formatExampleMessages(examples)...)
	messages = append(messages, ollamaMessage{
		Role:    "user",
		Content: formatQuery(item, value, date),
	})

	req := ollamaRequest{
		Model:    cfg.Model,
		Stream:   false,
		Format:   buildResponseSchema(enum),
		Messages: messages,
	}
	data, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshaling request: %w", err)
	}
	return data, nil
}

// formatQuery renders the item/value/date block shared by the query and examples.
func formatQuery(item string, value float64, date string) string {
	return fmt.Sprintf("item: %s\nvalue: %.2f\ndate: %s", item, value, date)
}

// formatExampleMessages converts resolved examples into user/assistant message pairs
// for few-shot injection. The synthetic assistant response matches the path-based
// response schema with high confidence.
func formatExampleMessages(examples []fewShotExample) []ollamaMessage {
	if len(examples) == 0 {
		return nil
	}
	msgs := make([]ollamaMessage, 0, len(examples)*2)
	for _, ex := range examples {
		assistant := fmt.Sprintf(`{"results":[{"path":%q,"confidence":0.95}]}`, ex.Path)
		msgs = append(msgs,
			ollamaMessage{Role: "user", Content: formatQuery(ex.Item, ex.Value, ex.Date)},
			ollamaMessage{Role: "assistant", Content: assistant},
		)
	}
	return msgs
}

func parseResponse(body interface{ Read([]byte) (int, error) }, pm taxonomy.PathMap, topN int) ([]Result, error) {
	var ollamaResp ollamaResponse
	if err := json.NewDecoder(body).Decode(&ollamaResp); err != nil {
		return nil, fmt.Errorf("decoding Ollama response: %w", err)
	}

	var classified classifyResponse
	if err := json.Unmarshal([]byte(ollamaResp.Message.Content), &classified); err != nil {
		return nil, fmt.Errorf("parsing classification JSON: %w", err)
	}

	results := splitResults(classified, pm)
	sort.Slice(results, func(i, j int) bool {
		return results[i].Confidence > results[j].Confidence
	})

	if len(results) > topN {
		results = results[:topN]
	}
	return results, nil
}

// splitResults turns each predicted path into a typed Result via the path map.
// A candidate whose path is not in the taxonomy (a model violation of the enum) is
// logged and dropped rather than producing a half-populated Result.
func splitResults(classified classifyResponse, pm taxonomy.PathMap) []Result {
	results := make([]Result, 0, len(classified.Results))
	for _, r := range classified.Results {
		typ, cat, sub, ok := pm.Split(r.Path)
		if !ok {
			logger.Debug("classify: dropping off-enum path", "path", r.Path)
			continue
		}
		results = append(results, Result{
			Type:        typ,
			Category:    cat,
			Subcategory: sub,
			Confidence:  r.Confidence,
		})
	}
	return results
}

// buildSystemPrompt renders the taxonomy as a type→category→subcategories tree and
// instructs the model to return full paths chosen from it.
func buildSystemPrompt(sheets []taxonomy.ExpenseType, topN int) string {
	var sb strings.Builder
	sb.WriteString("You are an expense classifier for Brazilian personal finance.\n")
	sb.WriteString("Classify the given expense into exactly one full path from the taxonomy below.\n")
	sb.WriteString(fmt.Sprintf("Return exactly %d candidates ranked by confidence (highest first).\n", topN))
	sb.WriteString("Each candidate's \"path\" must be a string copied verbatim from the taxonomy, in the form Type/Category/Subcategory.\n")
	sb.WriteString("Confidence is a float between 0.0 and 1.0.\n\n")
	sb.WriteString("Taxonomy (choose one full path):\n")
	writeTaxonomyTree(&sb, sheets)
	return sb.String()
}

// writeTaxonomyTree writes each type as a header line followed by its categories and
// their comma-joined subcategories.
func writeTaxonomyTree(sb *strings.Builder, sheets []taxonomy.ExpenseType) {
	for _, sheet := range sheets {
		sb.WriteString(sheet.Name)
		sb.WriteString(":\n")
		for _, cat := range sheet.Cats {
			sb.WriteString("  ")
			sb.WriteString(cat.Name)
			sb.WriteString(": ")
			sb.WriteString(strings.Join(subcatNames(cat.Subs), ", "))
			sb.WriteString("\n")
		}
	}
}

// subcatNames extracts the display names of a category's subcategories.
func subcatNames(subs []taxonomy.Subcat) []string {
	names := make([]string, len(subs))
	for i, s := range subs {
		names[i] = s.Name
	}
	return names
}
