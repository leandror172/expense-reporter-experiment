# internal/classifier/ — Quick Memory

*Working memory for the classifier package. Injected into agents. Keep under 30 lines.*

## Status
**Predicts the FULL path `Type/Category/Subcategory` (T-13).** `Classify` takes
`[]taxonomy.ExpenseType`; structured output = atomic `path` enum (`taxonomy.PathEnum`,
112 values), split via `PathMap.Split`; off-enum dropped. Flat feature-dict taxonomy
DELETED — feature dict is keyword-only. **Default model `my-classifier-q3`** (enum
validity is grammar-enforced, not model-dependent — KNOWLEDGE.md). **T-14 benchmarked
(s46): 63% full-path, confidence UNINFORMATIVE (91% of errors ≥0.85 — the auto-insert
gate does not protect); `--think=false` = 10× faster at −3.3 pp but kills the T-19
sentinel; q35 disqualified (think:false drops the grammar). WS-D gate NOT passed.**
**T-22 (s47): English type-level descriptions adopted via tracked sidecar
`config/type-descriptions.json` — +6pp TYPE accuracy, calibration unchanged; think-on
confirm deferred.** **T-23 probe (s49): token logprobs as a confidence signal — Ollama 0.17.5
exposes logprobs but reports them PRE-grammar-mask (raw, not renormalized); type-first enum
makes the margin unreadable, LEAF-FIRST enum makes it legible (Uber leaf p=0.986). Calibration
NOT yet proven. Report `.claude/t23-logprob-confidence-probe.md`; KNOWLEDGE.md.** Few-shot layer
1 (keyword) done; TF-IDF planned (5.R1). Corpus 1788 examples (5.R4).

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
