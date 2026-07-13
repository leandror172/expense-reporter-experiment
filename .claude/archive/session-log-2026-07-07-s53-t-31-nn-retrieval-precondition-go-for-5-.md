## 2026-07-07 - Session 53: T-31 NN-retrieval precondition — GO for 5.R2 (5-model embedding bake-off on the 160 misses)

### Context

Resumed after PR #43 (session-52 docs) merged; picked T-31 (the settled next step) — the cheap NN-retrieval precondition gating the 5.R2 embeddings build.

### What Was Done

- feat(t31): NN-retrieval precondition — GO for 5.R2 (hit@5 62-65% multilingual on the 160 misses) — PR #44 (`feat/t31-nn-precondition` → master)
- Built `nn_precondition.py` (`.claude/scratch/replay-649/`) via local-model codegen (verdicts 0 → 1 + inline patches): faithful Python port of `MergeExamplePools` (incl. keep-training-duplicates → pool=1,744 exactly matching the replay), LOO by the same dedup key, per-model resumable embedding caches, hit@1/3/5 + per-subcat + neighbor-source breakdowns
- Ran the 5-model bake-off: qwen3-embedding:8b 65.0% hit@5 / snowflake-arctic-embed2 62.5% (best hit@1 50.0%, 1.2 GB) / bge-m3 61.3% / embeddinggemma 57.5% / nomic-embed-text 41.2%; pulled embeddinggemma + snowflake-arctic-embed2
- Robustness cuts: excl-Diversos 61.6%, unique-item (78) 65.4%, unique-excl-Div 62.0% → signal is real, not grab-bag/recurrence inflation; report `FINDINGS-nn.md`
- Tracking updated: index.md replay row, replay/classifier/repo QUICKs, repo KNOWLEDGE cascade section (5.R1 RULED OUT / 5.R2 GO)

### Decisions Made

- GO for 5.R2 — clears the pre-committed ≥60% hit@5 bar (set this session: ≥60% GO, ≤40% no-go, between = judge hit quality)
- Model pick for 5.R2: snowflake-arctic-embed2 (1.2 GB, best hit@1, −2.5pp hit@5 vs the 8B) — VRAM-practical next to my-classifier-q3; benchmark qwen3-embedding:8b as alternate in the build A/B
- Probe embeds item text only (value belongs to E2, would muddy the measurement); full 1,744 production pool (what 5.R2 actually retrieves from)
- K=5 injection mandatory — hit@1 ≤50% for every model, top-1-only would be wrong half the time

### Next

- Build 5.R2: embed-on-miss cascade layer in `internal/classifier` (Ollama /api/embeddings, disk cache, cosine top-K → few-shot injection), then A/B final classifier accuracy via the same replay harness
- T-32 (independent, parallel-able): wire the specificity/agreement gate into `IsAutoInsertable`
- Merge PR #44; T-20 dedup remains a clean independent pick-up

### Gotchas

- Go `MergeExamplePools` only adds FEEDBACK keys to `seen` — duplicate training items are all kept (pool 1,744, not 1,179); a faithful port must NOT dedup training-vs-training (first probe run was wrong for this reason)
- Multilingual embedding training matters more than size on PT expense text: 274MB nomic collapses to 41% while every multilingual model clears 57% (closes embedding-retrieval.md open question #1)
- ~35% of misses have NO same-subcat top-5 neighbor even with the best embedder → permanent-review residue inside the novel pool; 5.R2 shrinks the sinkhole, doesn't eliminate it
- The probe is a retrieval UPPER BOUND — whether the model uses the injected examples is the 5.R2 build's own A/B
