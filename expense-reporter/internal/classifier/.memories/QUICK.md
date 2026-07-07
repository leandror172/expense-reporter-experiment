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
confirm deferred.** **T-23 GATE + RETRIEVAL MEASURED on real data (649-replay, session 52).** Report
`.claude/scratch/replay-649/FINDINGS{,-model}.md`; harness `replay_{retrieval,model}_test.go`
(`//go:build replay`). SHIP: **confidence DEAD as gate** (52→56% flat); **specificity `top_score`
replaces it** (monotone, 52→87%); **best gate = AGREEMENT `spec==1.0` ∧ model==keyword-top1 → 95% subcat**;
gate runs NO-THINK (shape robust, 10× faster). HOLD (649 = confidence-selected review subset, 378/725
bypassed+unlabeled → absolute levels unmeasurable): "no silent auto-insert" + WS-D scoping. RETRIEVAL:
miss 24.7% all `no_keyword_match` → **5.R1 ruled out, 5.R2 embeddings is the lever** (softened; NEXT =
cheap NN-retrieval precondition on the 160 misses). T-30 leaf-first A/B → HOLD (few-shot already fixes
wrong-leaf); T-23 logprob probe (s49) parked. Taxonomy: `Apoia-se 4i20`→`Apoia-se` (workbook re-export
pending). Few-shot layer 1 (keyword) done. Corpus 1788 (5.R4).

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
