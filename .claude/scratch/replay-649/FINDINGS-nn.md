# 649-Replay — NN-Retrieval Precondition (T-31 / 5.R2 go-no-go) — Findings

*Session 53 (2026-07-07). Embedding nearest-neighbor probe over the 160 no-keyword-match
miss rows from the retrieval replay (`retrieval.jsonl`, `n_examples==0`), against the
FULL production example pool (1,744 — replicates `MergeExamplePools` exactly, including
its keep-training-duplicates behavior). Leave-one-out by the same dedup key. Text-only
embeddings (no value/date). Harness: `nn_precondition.py`. Raw: `nn-<model>.jsonl` +
`nn-summary.json` (gitignored — real descriptions).*

Run: `python3 nn_precondition.py` (per-model JSONL embedding caches → re-runs are seconds).

## Question

The retrieval replay showed 160/649 rows get ZERO few-shot examples (100% `no_keyword_match`)
and that pool is the accuracy sinkhole (18.8% full-path). "160/160 have a same-subcat
pool-mate" ≠ retrievable. **Precondition:** can embedding NN actually surface a same-subcat
neighbor into a top-5 slate? Pre-committed rule: hit@5 ≥60% → 5.R2 justified; ≤40% → novel
items are permanent-review; between → judge hit quality.

## Result — GO (borderline-clear)

| model | size | hit@1 | hit@3 | hit@5 |
|---|---|---|---|---|
| **qwen3-embedding:8b** | 4.7 GB | 38.8% | 57.5% | **65.0%** |
| snowflake-arctic-embed2 | 1.2 GB | **50.0%** | 60.0% | 62.5% |
| bge-m3 | 1.2 GB | 40.0% | 56.2% | 61.3% |
| embeddinggemma | 622 MB | 37.5% | 50.0% | 57.5% |
| nomic-embed-text | 274 MB | 18.8% | 32.5% | 41.2% |

Robustness cuts (qwen3-embedding:8b): excl-Diversos 61.6%; unique-item basis (78 items)
65.4%; unique excl-Diversos 62.0%. **The hit rate is NOT grab-bag inflation** (Diversos was
14/14 but removing it barely moves the number) and NOT recurrence double-counting.

## Reads

1. **5.R2 is justified** — embeddings convert ~62–65% of the zero-example sinkhole into
   slates containing the right subcat. Ceiling read: the miss pool's 18.8% full-path could
   move meaningfully toward the keyword-hit band (~67%); actual gain depends on the model
   using the injected examples (that's the 5.R2 build's own A/B to run).
2. **Multilingual training matters more than size** — nomic (English-leaning) collapses to
   41.2%; every multilingual model clears 57%. Answers the embedding-retrieval doc's open
   question #1.
3. **Model choice:** qwen3-embedding:8b tops hit@5 but costs 4.7 GB VRAM next to the
   classifier. **snowflake-arctic-embed2 (1.2 GB) is the practical pick** — best hit@1
   (50.0%), −2.5pp hit@5, fits alongside q3. bge-m3 is an acceptable same-size alternate.
4. **~35–38% of misses have NO same-subcat neighbor in the top 5 even for the best model**
   — a permanent-review residue remains inside the novel pool; 5.R2 shrinks the sinkhole, it does
   not eliminate it. Aligns with the review-first framing (T-23/T-32: gate-to-review).

## Caveats

- Retrieval metric only — upper bound on 5.R2. Whether injected embedding-retrieved
  examples improve the model's final accuracy is the 5.R2 build's A/B (same replay harness).
- hit@1 ≤50% everywhere: a top-1-only injection would be wrong half the time; keep K=5.
- Same confidence-selected 649 subset caveat as the parent replay: relative reads only.

## Feeds

- **T-31 CLOSED — GO for 5.R2** (`.claude/tasks.md`). Build = embedding cascade layer in
  `internal/classifier` (embed-on-miss, cache, cosine top-K → few-shot injection), then A/B
  on this same replay.
- Model default for 5.R2: `snowflake-arctic-embed2` (VRAM-practical), benchmark alternate
  `qwen3-embedding:8b` in the A/B if VRAM allows.
