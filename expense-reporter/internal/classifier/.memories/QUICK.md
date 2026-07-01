# internal/classifier/ — Quick Memory

*Working memory for the classifier package. Injected into agents. Keep under 30 lines.*

## Status
**Predicts the FULL path `Type/Category/Subcategory` (T-13).** `Classify` takes
`[]taxonomy.ExpenseType`; structured output = atomic `path` enum (`taxonomy.PathEnum`,
112 values), split via `PathMap.Split`; off-enum dropped. Flat feature-dict taxonomy
DELETED — feature dict is keyword-only. **Default model `my-classifier-q3`** (enum
validity is grammar-enforced, not model-dependent — KNOWLEDGE.md). T-14 benchmark open.
Few-shot layer 1 (keyword) done; TF-IDF planned (5.R1). Corpus 1788 examples (5.R4).

## Structure
```
classifier.go   # Classify() — Ollama client, prompt, response parsing
decision.go     # IsAutoInsertable() — threshold + exclusion check
examples.go     # SelectExamples() — keyword-based few-shot selection
loader.go       # Training data, feedback examples, keyword index
```

## Key Rules
- **Ollama structured output** — `format` param compiles to GBNF (forces valid enum)
- **Source priority** — Corrected > Training > Confirmed feedback
- **Specificity threshold 0.7** — above = single-subcategory; below = interleave top-2
- **Feedback pool merges** — feedback overrides training for the same item

## Deeper Memory → KNOWLEDGE.md
Few-shot algorithm · prompt architecture (T-13) · grammar-enforced enum finding ·
empirical findings + corpus expansion
