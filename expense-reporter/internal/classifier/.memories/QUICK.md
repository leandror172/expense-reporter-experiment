# internal/classifier/ — Quick Memory

*Working memory for the classifier package. Injected into agents. Keep under 30 lines.*

## Status
Keyword-based few-shot injection (layer 1 of 3) complete.
TF-IDF retrieval (layer 2) planned as 5.R1 — would replace keyword specificity
with corpus-level term frequency for better example selection.
Embedding retrieval (layer 3) deferred — requires vector storage.
**Training corpus expanded 694→1788 (2026-06-20, 5.R4):** historical workbook
extraction 2022–2025 deduped + merged into `training_data_complete.json` (now
multi-year, 15 cats / 81 subs). Bigger few-shot/keyword pool. See
[[project_workbook_extraction_5r4]]. Classifier now emits expense `type` (5.R4).

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
