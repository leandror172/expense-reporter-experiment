# 649-Replay — 5.R2 Embedding Fallback A/B (Phase 4 decision gate) — Findings

*Session 54 (2026-07-10). Replays the 160 keyword-miss rows (80 unique expenses —
`GenerateID` collides for repeat expenses, duplicates carry identical predictions,
scored on unique ids) through the production classifier with the 5.R2 embed-on-miss
fallback ON, vs the session-52 OFF baseline (`model.jsonl`, `top_score==0` rows).
Harness: `replay_model_test.go` + new `REPLAY_MISS_ONLY` / `REPLAY_NO_EMBED` env knobs.
Outputs `model-embed{,-think}.jsonl` (gitignored — real data). Embedder:
snowflake-arctic-embed2, K=5, real disk cache (populated by this run, ~3 min/1,744 items).*

## Question

T-31 proved the retrieval upper bound (hit@5 62.5% on the misses). Does the model
actually USE the injected embedding-retrieved examples — does miss-pool accuracy move?

## Result — YES, decisively (ADOPT)

Miss stratum, n=80 unique expenses:

| metric | OFF (s52, no-think) | ON no-think | ON think-on |
|---|---|---|---|
| full path | 18.8% | **52.5%** | **55.0%** |
| subcategory | 21.2% | 56.2% | 57.5% |
| type | 56.2% | 73.8% | 72.5% |

- **No-think Δ +33.8pp full-path; discordant 29 fixed / 2 broke** — significant at any
  reasonable threshold (McNemar). The zero-example sinkhole (18.8%) moved most of the way
  to the T-31 retrieval ceiling (62.5% hit@5) and near the keyword-hit band (~67%).
- **Think-on adds ~nothing once embeddings are in:** +2.5pp over no-think ON (10 fixed /
  8 broke — churn, not signal), at 5.5× the latency (2,250s vs 410s for 160 rows).
  Consistent with T-30's sub-additivity finding: retrieval and thinking fix the same errors.
- Run cost: no-think 410s (incl. one-time pool embed), think-on 2,250s.

## Reads

1. **5.R2 is validated end-to-end — keep the fallback ON as the production default** (it
   already is: `NoEmbedRetrieval` zero-value). The measured lift is the largest single
   accuracy improvement in the project's history on any stratum.
2. **No-think is the right mode for the miss path too** — the embed examples do the work
   thinking would have attempted; paying 14s/item buys +2.5pp of churn. Strengthens T-24's
   no-think lean.
3. Residual: ON-arm miss accuracy (52–55%) ≈ the clean-stratum keyword-hit level; the
   ~35% permanent-review residue T-31 predicted (no same-subcat neighbor in top-5) is
   the remaining error mass. The gate (T-32) still decides what reaches auto-insert.

## Caveats

- **Think-on cell has no exact-paired OFF baseline** (s52 ran think-on only on the
  high-spec band); its OFF column above is the no-think baseline, so its Δ inherits the
  known ~+3pp think-level shift. Immaterial to the decision — the no-think pair alone
  justifies adoption, and think-on ON was measured directly.
- **sync.Once faithfulness wrinkle:** the pool embeds once from the FIRST miss row's LOO
  pool, so later rows' stores lack that first item's vectors (≤1 potential neighbor
  distortion, direction: understates ON). Query-side leakage is handled by the
  retriever's own-key exclusion.
- Same parent-replay caveat: the 649 is a confidence-selected review subset — relative
  reads only.
- Alternate embedder (qwen3-embedding:8b, +2.5pp hit@5) NOT run — arctic-embed2 already
  cleared the bar; benchmark only if chasing the last few points.

## Feeds

- **Phase 4 CLOSED — 5.R2 ADOPTED** (plan `.claude/plans/5r2-embedding-retrieval.md`).
- Check-later items live: CL1 (widen trigger to weak/ambiguous keyword matches) and CL2
  (pre-warm command) — see plan.
- T-24 (--think default): miss-path evidence now also favors no-think.
- T-32 gate wiring: unchanged priority; with misses at ~55% the gate-to-review framing
  stands (55% is far below auto-insert precision).
