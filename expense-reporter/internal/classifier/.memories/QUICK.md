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
miss 24.7% all `no_keyword_match` → **5.R1 ruled out, 5.R2 embeddings is the lever**. **T-31 NN
precondition (s53): GO for 5.R2** — hit@5 on the 160 misses 62–65% multilingual (arctic-embed2 1.2 GB
the practical pick, best hit@1 50%; nomic collapses 41%); K=5 mandatory (hit@1 ≤50%); ~35%
permanent-review residue. `FINDINGS-nn.md`. T-30 leaf-first A/B → HOLD (few-shot already fixes
wrong-leaf); T-23 logprob probe (s49) parked. Taxonomy: `Apoia-se 4i20`→`Apoia-se` (workbook re-export
pending). Few-shot layer 1 (keyword) done. Corpus 1788 (5.R4). **5.R2 Phase 1+2 landed (s54):
`embedding.go` (Embedder iface + OllamaEmbedder `/api/embeddings`, per-model JSONL cache
`embeddings-<model>.jsonl`, dim-check fail-loud, embed-only-missing reconcile) +
`embedding_retriever.go` (brute-force cosine top-K → `[]Example`, slate-dedup by lowercased key,
LOO, source-priority tiebreak, deterministic index-permutation sort). Unit-only; Phase 3 wiring
into `selectExamples` NOT done. Embed model default `snowflake-arctic-embed2` (D7). Plan:
`.claude/plans/5r2-embedding-retrieval.md`.**

## Structure
```
classifier.go   # Classify() — Ollama client, prompt, response parsing
decision.go     # IsAutoInsertable() — threshold + exclusion check
examples.go     # SelectExamples() — keyword-based few-shot selection
loader.go       # Training data, feedback examples, keyword index
embedding.go    # Embedder + Ollama client + JSONL embedding cache/reconcile (5.R2)
embedding_retriever.go # cosineSimilarity + TopKEmbeddingExamples (5.R2)
```

## Key Rules
- **Ollama structured output** — `format` param compiles to GBNF (forces valid enum)
- **Source priority** — Corrected > Training > Confirmed feedback
- **Specificity threshold 0.7** — above = single-subcategory; below = interleave top-2
- **Feedback pool merges** — feedback overrides training for the same item

## Deeper Memory → KNOWLEDGE.md
Few-shot algorithm · prompt architecture (T-13) · grammar-enforced enum finding ·
empirical findings + corpus expansion
