# 649-Replay — Retrieval Half (T-23 / 5.R1) — Findings

*Session 52 (2026-07-06). Ollama-free replay of the 649 human-verified real labels
(`classifications.jsonl`) through the PRODUCTION few-shot retrieval path
(`SelectExamples` + real tokenizer/specificity), leave-one-out. Harness:
`expense-reporter/internal/classifier/replay_retrieval_test.go` (`//go:build replay`).
Analysis: `analyze.py`. Raw: `retrieval.jsonl`.*

Run: `cd expense-reporter && go test -tags=replay -run TestReplayRetrieval649 ./internal/classifier/ -v`
then `python3 .claude/scratch/replay-649/analyze.py`.

## Provenance / validity
- Pool wired exactly like `classifier.go:95-106`: `MergeExamplePools(training=1788, feedback=649)` = 1744, `topK=5`.
- Feature dict = **229 keywords** = the original 694-training count → **NOT rebuilt** on the 5.R4 corpus or the 649 reviews. So specificity/frequency are **clean of review leakage** (dict mtime 07-04 = surgical Apoia-se edit, not a rebuild).
- **Leave-one-out** removes each query's own pool entry using the exact `MergeExamplePools` dedup key. **`self_match = 0`** → leakage tripwire passes.
- Denominator = **per-row** (misses/649), stream-weighted toward recurring items (stated, per memory).

## Headline

| Metric | Value | Meaning |
|---|---|---|
| **Miss rate** (0 examples) | **24.7%** (160/649) | > 10% → **5.R1/5.R2 trigger MET** |
| Miss reason | **100% `no_keyword_match`** | zero `no_examples_in_bucket` → semantic gap, not vocab gap |
| retrieval_hit | 47.1% | right subcat surfaced among few-shot examples |
| keyword_top1_correct | 42.7% | pure-keyword top-1 (all rows) |
| self_match | **0** | LOO faithful |

## Finding 1 — the lever is 5.R2 embeddings, not 5.R1 TF-IDF
Every miss is `no_keyword_match` (no token hit the index at all), AND **160/160 missed items
belong to a subcategory that HAS other examples in the 649.** The neighbors exist in the pool;
lexical retrieval can't reach them (zero token overlap). TF-IDF is still lexical → cannot bridge a
zero-overlap gap. **Embeddings (5.R2) directly attack this 24.7% pool; TF-IDF (5.R1) mostly can't.**
→ Re-rank the retrieval roadmap: **5.R2 > 5.R1** (matches strategic doc §2c).

## Finding 2 — specificity (`top_score`) is a real, monotone gate signal; frequency is NOT additive
Risk–coverage among the 489 keyword hits (accuracy = keyword_top1_correct, an OPTIMISTIC proxy):

| specificity ≥ | coverage | precision |
|---|---|---|
| 1.0 | 45.4% | **90.5%** |
| 0.85 | 64.6% | 76.6% |
| 0.7 | 72.6% | 71.0% |
| 0.5 | 92.6% | 60.0% |
| 0.0 | 100% | 56.6% |

Monotone — higher specificity → higher precision. This is E1 recurrence-strength **working** as a
stratifier. **Frequency-aware gating adds nothing:** freq≥2 == freq≥1 (no singletons exist), and
higher freq thresholds HURT (freq≥10 → 55%, freq≥25 bucket → 38.6%) because high-frequency tokens
are generic/ambiguous. **Use specificity, not frequency.**

## Finding 3 — the advisor's singleton-masquerade risk does NOT occur in this dict
Concern: a freq==1 token scores specificity 1.0 (novel masquerading as recurrent), invisible to
pool-LOO. Emitted `driver_freq`/`idf` to defend. Result: **0 items with `driver_freq≤1` at any
`top_score` threshold** — the dict carries no selection-driving singletons (min bucket 2–4). Concern
retired empirically; the field stays as the proof.

## Caveats (hold the line — advisor #3)
- **`keyword_top1_correct` is the KEYWORD layer, not the model.** The model tops at 63% full-path and
  diverges from keyword. The 90.5%@spec=1.0 is an OPTIMISTIC upper bound on what a specificity gate buys.
- **The real gate risk–coverage needs the model.** Join actual model output (predicted path) to
  `top_score` — that is the SEPARATE Ollama accuracy half (~16 min no-think), not done here.
- This half decides the RETRIEVAL question (5.R1/5.R2). It only *suggests* the gate; it does not close it.

## Decisions this unblocks
1. **5.R1 trigger met, but re-pointed to 5.R2** — miss pool is semantic, not lexical. (5.R2 > 5.R1.)
2. **E1 specificity gate is viable pending model confirmation** — monotone curve; spec≥1.0 is the high-precision band.
3. **Next measurement:** Ollama accuracy half — replay the 649 through `classify --json`, join predicted
   path to `top_score`, get the REAL (model) risk–coverage → closes the T-23 gate → WS-D readiness.
