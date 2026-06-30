# internal/classifier/ — Quick Memory

*Working memory for the classifier package. Injected into agents. Keep under 30 lines.*

## Status
**T-13 (session 41): predicts the FULL path `(type,category,subcategory)`.** `Classify`
takes `[]taxonomy.ExpenseType` (this package now depends on `internal/taxonomy`); the
system prompt renders the 3-level tree; structured output is an atomic `path` enum
(`taxonomy.PathEnum`, 112 values) per candidate, split back via `PathMap.Split`; off-enum
paths are dropped. `Result` carries `Type`. `Taxonomy`/`LoadTaxonomy` (the flat feature-dict
`category_mapping`) are **deleted** — the feature dict is now keyword-only. Few-shot examples
resolve their canonical path via `ResolveLeaf`+`PathFor` (drop if unresolvable);
`Example.TypeHint` comes from the training `source` sheet-name. See `.claude/t13-implementation-report.md`.
**Default model = `my-classifier-q3` (session 42, all commands unified).** The earlier qcoder
default was reverted: enum validity is **grammar-enforced** (Ollama compiles the `format` enum to
a GBNF grammar — verified: a 9B model is forced into valid 112-enum paths), so it is NOT
model-dependent; the session-40 "qcoder honors the enum 100%" smoke test measured a property of the
grammar, not the model. qcoder (qwen3-coder:30b, 20.7 GB) only added CPU-offload latency + load-time
500s on a 12 GB GPU. q3 validated end-to-end on the real 112-path taxonomy (Uber→Uber/Taxi 100%).
Accuracy across q3/q35/qcoder is the open question → T-14.

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
