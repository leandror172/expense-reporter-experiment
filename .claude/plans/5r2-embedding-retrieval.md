# 5.R2 — Embedding Retrieval Layer (embed-on-miss cascade)

**Status:** PLANNED (session 54, 2026-07-10). Decisions locked with user; implementation not started.
**Branch:** `feat/5r2-embedding-retrieval`
**Precondition:** T-31 GO (session 53) — NN hit@5 62–65% on the 160 no-keyword-match misses.
Reports: `.claude/scratch/replay-649/FINDINGS-nn.md` (+ `FINDINGS.md`, `FINDINGS-model.md`).
Design background: `data/classification/embedding-retrieval.md` [ref:embedding-retrieval].

## What it is

Today `selectExamples` (`internal/classifier/classifier.go:91`) is keyword-only and returns
**nil on a keyword miss** — 24.7% of real traffic, classifying at 18.8% full-path (the
accuracy sinkhole). 5.R2 adds a fallback layer: on keyword miss, embed the item via Ollama
`/api/embeddings`, cosine top-K against a disk-cached embedding of the merged example pool,
inject the neighbors as few-shot examples. Downstream (`resolveExamplePaths`, prompt
rendering, enum grammar) is unchanged — the fallback produces `[]Example`.

The T-31 probe is a **retrieval upper bound**: whether the model *uses* the injected
neighbors is this build's own A/B (Phase 4).

## Locked decisions

| # | Decision | Choice |
|---|----------|--------|
| D1 | Trigger scope | **Miss-only** — fallback fires only when `SelectExamples` returns nil (zero keyword matches). Weak/ambiguous matches (top score < 0.7) keep today's keyword path. |
| D2 | Enable switch | Config field (`classifier.Config`), default **on** once shipped; replay harness toggles it directly for the A/B. No CLI flag initially. |
| D3 | Cache format | Per-model JSONL in `DataDir`: `embeddings-<model>.jsonl`, one `{item, vector}` line per **unique raw item text**. Gitignored (real descriptions). Matches `nn_precondition.py`'s cache shape. |
| D4 | Cache freshness | **Per-item keying** — no snapshot hash. Model change → different file. Pool growth → diff unique pool texts vs cache keys, embed only missing, append. Pool is append-only so shrink/mutate can't happen; stale lines sit unused (cache ≈10–15 MB ceiling). |
| D5 | Reconcile timing | **Lazy + synchronous (blocking)** — first keyword miss in a process triggers reconcile-then-retrieve (`sync.Once`-style); keyword-hit runs pay zero. NO async: the triggering item needs the vectors, and async would classify early vs late misses inconsistently within one batch (poisons the A/B). Progress log line so batches don't look hung. |
| D6 | Cache key normalization | **Exact raw item text** — embedded text is byte-identical to pool text and to what T-31 measured. Near-duplicate casings cost redundant ~50ms embeds, nothing in quality. |
| D7 | Embedding model | **`snowflake-arctic-embed2`** (1.2 GB, best hit@1 50%, fits beside q3). Alternate `qwen3-embedding:8b` (+2.5pp hit@5, 4.7 GB) in the A/B if VRAM allows. |
| D8 | K | **K=5 mandatory** — hit@1 ≤50% everywhere; top-1-only injection would be wrong half the time. Same topK as the keyword path. |
| D9 | Failure mode | Ollama down / embed model missing / cache unreadable → warn once, return nil examples (today's behavior). Classification never blocks on the retrieval layer. |
| D10 | Robustness | Tolerate torn last JSONL line (skip + re-embed that item); per-file vector-dimension sanity check (fail loud, not garbage cosines). |
| D11 | Vector search | In-memory brute-force cosine over ~1.7K vectors (sub-ms). No vector DB, no SQLite. |
| D12 | Tests | Unit (testify, table-driven) against an **embedder interface** (fake vectors): cache diff, torn-line, dimension check, cosine top-K, miss-only trigger, failure degradation. ONE Ollama-gated acceptance scenario (event-style Given, outcome-named Then, `withFeedbackAndTaxonomyConfig`). Real proof = replay A/B. |

## Check-later items (post-A/B revisions)

- [ ] **CL1 — Trigger widening:** revisit firing the fallback also on weak/ambiguous keyword
  matches (top score < 0.7) once the miss-only A/B gives a baseline. Widening earlier would
  contaminate the A/B.
- [ ] **CL2 — Burst mitigation:** the accepted worst cases — first-ever activation ≈1,744
  embeds (~2–3 min, one-time; the A/B run itself pre-populates the cache) and post-review
  bursts (~300 new feedback items ≈15–30 s). If it annoys, add a pre-warm (`warm-embeddings`
  command or hook at end of `apply`). Do NOT "skip-missing" retrieval — it biases away from
  the newest Corrected examples, the highest-priority source.

## Phases

**Do not proceed to the next phase without explicit user permission.** Codegen via local
model (`my-go-qcoder` tier) per CLAUDE.md; verdict protocol applies.

### Phase 1 — Embedding client + disk cache
`internal/classifier/embedding.go` (+ `embedding_test.go`):
- Embedder interface (e.g. `Embed(text string) ([]float64, error)`) + Ollama
  `/api/embeddings` implementation (`POST {model, prompt}` → `{embedding}`).
- Cache load (skip torn/malformed lines, dimension check), diff-against-pool,
  embed-missing, append (O_APPEND, one line per item).
- Unit tests with a fake embedder; no Ollama.

### Phase 2 — Cosine top-K retriever
- Cosine similarity + top-K selection over the in-memory store, returning `[]Example`
  (pool rows joined back by item text; preserve `Source`/`TypeHint` so
  `resolveExamplePaths` and source-priority semantics keep working).
- Duplicate pool rows for one item text: dedup the *slate* by dedup key so K=5 means 5
  distinct examples (mirror the probe's LOO/dedup semantics — check `nn_precondition.py`).
- Unit tests: known fake vectors → deterministic ranking.

### Phase 3 — Wire the fallback into `selectExamples`
- `classifier.go`: on `SelectExamples` nil → lazy-init (sync.Once) reconcile → embed query
  → top-K inject. Config gate (D2), degrade-to-nil (D9), progress + debug logging
  (`few-shot: embedding fallback`, count, item).
- Config plumbing: embedding model name + enable flag through `classifier.Config` (and
  `internal/config` if a file-level setting is wanted — decide at build time).
- Acceptance scenario (Ollama-gated): no-keyword-match item → debug/JSON evidence that
  examples were injected. Given/Then naming per `test/PATTERNS.md`.

### Phase 4 — Replay A/B (the decision gate)
- Rerun `replay_model_test.go` (`//go:build replay`) on the 649 with fallback ON vs OFF;
  stratify the 160-miss pool vs keyword-hit band. The A/B run populates the cache.
- Metrics: full-path / type / subcat accuracy on the miss stratum (baseline 18.8%
  full-path; keyword-hit band ≈67% is the ceiling read), overall delta, latency.
- Optional cell: `qwen3-embedding:8b` as embedder (D7 alternate).
- Report → `.claude/scratch/replay-649/FINDINGS-5r2.md`; adopt/revert default per result.
- Caveat carried from the parent replay: the 649 is a confidence-selected review subset —
  relative reads only.

## Out of scope

- 5.R1 TF-IDF (ruled out by the 649-replay — 100% of misses are `no_keyword_match`).
- T-32 gate wiring (independent track; `decision.go`).
- Value/date in the embedding text (open question #2 in embedding-retrieval.md — text-only
  is what T-31 validated).
- Vector DB / ANN indexing.
