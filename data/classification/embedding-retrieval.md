# Embedding Retrieval Layer — Expense Classifier

**Purpose:** Reference document for the embedding-based few-shot retrieval layer.
This is the third (and most advanced) layer in the retrieval pipeline
(§ `data/classification/retrieval-strategy.md` for the full cascade).
Not planned for near-term implementation — this documents findings and design
considerations for future sessions.

**Created:** 2026-03-18 (session 10 planning)

---

<!-- ref:embedding-retrieval -->
## Embedding-Based Retrieval for Few-Shot Examples

### What It Does

An embedding model converts text into dense vectors in a high-dimensional semantic space
(typically 384–1536 dimensions). Texts with similar **meaning** — not just similar words —
end up close together. Vector similarity (cosine or dot product) retrieves the most
semantically relevant few-shot examples for a given input.

This is the classic RAG (Retrieval-Augmented Generation) pattern applied to few-shot
example selection rather than document retrieval.

### Why It Helps Beyond TF-IDF

TF-IDF is lexical — it can only match texts that share actual words. Embeddings capture
**conceptual similarity**:

| Input | TF-IDF Result | Embedding Advantage |
|-------|--------------|---------------------|
| "99 Taxi Centro" | Low similarity to "Uber Leblon" (no shared words) | High similarity — both are ride-hailing expenses |
| "Netflix mensal" | Matches "Netflix" keyword but no multi-word context | Clusters with "Spotify", "Disney+", "Amazon Prime" — subscription concept |
| "Farmácia Drogasil" | Matches "farmácia" keyword | Also finds "Drogaria São Paulo", "Droga Raia" — pharmacy concept regardless of brand |
| "Almoço restaurante japonês" | "almoço" + "restaurante" match separately | Clusters with "Sushi delivery", "Japa" — cuisine/dining concept |

### When Embeddings Make Sense for This Project

**Likely valuable when:**
- Training set grows beyond ~1000 entries (more historical workbooks extracted)
- Keyword vocabulary can't keep up with description variety
- New vendors/brands appear that don't match existing keywords
- Correction rate remains high despite TF-IDF layer being active

**Likely overkill when:**
- Training set is small (~694) and keyword coverage is good (142/229 = 62% perfect specificity)
- Most expenses use consistent descriptions (same vendors, same phrasing)
- The model already classifies correctly with just taxonomy + keyword-matched examples

**Current assessment (2026-03-18):** Premature. Keyword matching + eventual TF-IDF
should cover the majority of cases. Revisit after measuring retrieval quality with
real classification data.

### Ollama Embedding Endpoint

Ollama provides a local embedding endpoint — no external API needed:

```
POST http://localhost:11434/api/embeddings
{
  "model": "nomic-embed-text",   // or any embedding model
  "prompt": "Uber Centro"
}

Response:
{
  "embedding": [0.0123, -0.0456, ...]   // dense vector, dims depend on model
}
```

**Available embedding models (Ollama):**

| Model | Dimensions | Size | Notes |
|-------|-----------|------|-------|
| `nomic-embed-text` | 768 | 274MB | Good general-purpose, fast |
| `mxbai-embed-large` | 1024 | 670MB | Higher quality, slower |
| `all-minilm` | 384 | 46MB | Smallest, fastest, decent quality |
| `snowflake-arctic-embed` | 1024 | 670MB | Strong multilingual support |

**Multilingual consideration:** Expense descriptions are in Portuguese. Models with
explicit multilingual support (`snowflake-arctic-embed`, `nomic-embed-text` v1.5+)
would perform better than English-only models. This needs benchmarking — the expense
vocabulary is domain-specific (financial terms, brand names, Brazilian Portuguese).

### Architecture Considerations

#### Vector Store

For 694 entries (or even a few thousand after historical extraction), a dedicated vector
database is overkill. Options:

| Approach | Complexity | Performance | When to Use |
|----------|-----------|-------------|-------------|
| **In-memory brute force** | Trivial — `[][]float64` | Sub-ms at <5K entries | Current dataset size |
| **Pre-computed + cached to disk** | Low — JSON/binary file | Sub-ms, no startup embedding cost | After first run, vectors are stable |
| **SQLite + vector extension** | Medium — needs CGo or pure-Go port | Fast, persistent, queryable | >10K entries |
| **Dedicated vector DB (Chroma, Qdrant)** | High — new dependency | Optimized for scale | >100K entries (not this project) |

**Recommendation:** In-memory brute force with disk cache. Same pattern as TF-IDF
vectors but higher dimensionality (768 vs 229).

**Storage estimate:**
- 694 entries × 768 dims × 8 bytes = ~4.3MB (in-memory)
- 694 entries × 768 dims × 8 bytes = ~4.3MB (disk cache, binary format)
- Negligible for this dataset size

#### Embedding Pipeline

```
Startup:
  1. Check disk cache: embeddings.bin exists + matches training data hash?
     → yes: load from cache
     → no: embed all 694 entries via Ollama /api/embeddings, save cache

Classify-time:
  1. Embed input item via Ollama /api/embeddings (~50-100ms)
  2. Cosine similarity against all cached training vectors
  3. Top-K selection → few-shot examples
```

**Latency impact:**
- Embedding call: ~50-100ms (local Ollama, small text)
- Similarity search: <1ms (brute force over 694 vectors)
- Total added latency: ~50-100ms per classify call
- Compare: current classify call is ~2-30s (LLM generation) — embedding overhead is negligible

#### Incremental Updates

When `classifications.jsonl` gains new entries (corrected or confirmed), their embeddings
should be added to the cache:
- Embed the new entry
- Append to the in-memory vector store + disk cache
- No need to re-embed existing entries (embeddings are deterministic per model version)

When the embedding model is changed/updated, **all vectors must be recomputed** —
embeddings from different models are not comparable.

### Relationship to Other Layers

In the retrieval cascade (§ [ref:retrieval-strategy]):

- **Keywords first:** Unambiguous keyword matches (specificity ≥ 0.7) short-circuit —
  no need to compute embeddings for "Diarista Letícia" when "diarista" maps 1:1.
- **TF-IDF second:** Multi-word lexical similarity handles most ambiguous cases without
  an embedding model call.
- **Embeddings third:** Only invoked when keywords miss AND TF-IDF similarity is below
  threshold. This is the "last resort before taxonomy-only" layer.

**Does embedding supersede TF-IDF?** In theory, a good embedding captures everything
TF-IDF does and more. In practice, TF-IDF has advantages for this dataset:
- **Deterministic** — same input always gives same result (embedding models can vary by version)
- **Interpretable** — you can see which terms contributed to the match
- **No model dependency** — pure math, no Ollama call needed
- **Faster** — no network round-trip for the embedding call

The practical answer: keep both. TF-IDF handles the "multi-word but still lexical" gap
cheaply. Embeddings handle the "semantic gap" cases that TF-IDF can't reach.

### Decision Criteria for Adding This Layer

After TF-IDF is implemented, measure:

1. **TF-IDF miss rate** — what % of classify calls fall through both keyword AND TF-IDF
   retrieval with no good examples?
2. **Semantic gap cases** — are there corrected entries where the input is semantically
   similar to training data but lexically different? (e.g., brand synonyms, rephrased descriptions)
3. **Training set diversity** — after historical workbook extraction, is there enough
   variety that lexical methods can't cover it?

If semantic gap cases account for >5% of corrections, embeddings are worth adding.

### Open Questions (for future investigation)

1. **Which embedding model for Portuguese financial text?** Needs benchmarking with actual
   expense descriptions. `nomic-embed-text` is the default candidate but multilingual
   quality varies.
2. **Should the embedding include value and date, or just item text?** Value ranges might
   help disambiguate ("R$15" is more likely transport than "R$500"), but mixing numeric
   and text features in embeddings is non-trivial.
3. **Embedding model VRAM impact** — running an embedding model alongside the classifier
   model. Both need to fit in GPU memory. `all-minilm` (46MB) is negligible;
   `nomic-embed-text` (274MB) may matter depending on GPU.
4. **Cache invalidation** — when to re-embed the training set? On model change (mandatory),
   on training data change (append new, keep existing).
<!-- /ref:embedding-retrieval -->
