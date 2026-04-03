# internal/classifier/ — Quick Memory

*Working memory for the classifier package. Injected into agents. Keep under 30 lines.*

## Status
Keyword-based few-shot injection (layer 1 of 3) complete.
TF-IDF retrieval (layer 2) planned as 5.R1 — would replace keyword specificity
with corpus-level term frequency for better example selection.
Embedding retrieval (layer 3) deferred — requires vector storage.

## Structure
```
classifier.go       # Classify() — Ollama client, prompt construction, response parsing
decision.go         # IsAutoInsertable() — threshold + exclusion check
examples.go         # SelectExamples() — keyword-based few-shot example selection
loader.go           # Load training data, feedback examples, keyword index from JSON files
*_test.go           # Table-driven tests with testify
```

## Key Rules
- **Ollama structured output** — `format` param with JSON schema forces valid response structure
- **Three example sources** — training data (static), confirmed feedback, corrected feedback
- **Source priority** — Corrected > Training > Confirmed (corrected examples are most valuable)
- **Specificity threshold 0.7** — above = single-subcategory examples; below = interleave top-2
- **Feedback pool merges** — feedback entries override training entries for same item

## Deeper Memory → KNOWLEDGE.md
- **Few-shot selection algorithm** — tokenize → keyword lookup → score → select
- **Prompt architecture** — system prompt + few-shot pairs + user query
- **Empirical findings** — what works and what doesn't in classification accuracy
