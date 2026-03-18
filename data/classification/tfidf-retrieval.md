# TF-IDF Retrieval Layer — Expense Classifier

**Purpose:** Reference document for the TF-IDF-based few-shot retrieval layer.
This is the second layer in the retrieval pipeline (§ `data/classification/retrieval-strategy.md`
for the full cascade). Not implemented in 5.7 — this documents findings for future sessions.

**Created:** 2026-03-18 (session 10 planning)

---

<!-- ref:tfidf-retrieval -->
## TF-IDF for Few-Shot Retrieval

### What It Does

TF-IDF (Term Frequency–Inverse Document Frequency) converts text into numerical vectors
where each dimension is a term weighted by how important it is to that document relative
to the corpus. Cosine similarity between vectors measures how "close" two texts are.

For the classifier: given an input like "VA compras supermercado", compute its TF-IDF
vector and find the training examples with the highest cosine similarity — those become
the few-shot examples injected into the prompt.

### Why It Helps Beyond Keywords

Keywords match individual tokens. TF-IDF captures **multi-token context**:

| Input | Keyword Match | TF-IDF Advantage |
|-------|--------------|------------------|
| "VA compras supermercado" | "va" → specificity 0.36 (ambiguous, 5 subcategories) | Full phrase similarity weights "supermercado" higher (IDF), resolving ambiguity |
| "compras mercado municipal" | "mercado" might not be in keyword dict | TF-IDF matches against training entries containing "mercado" with proper weighting |
| "pagamento cartão crédito" | "cartão" and "crédito" match separately | Joint vector captures the compound concept |

### Existing Artifacts

The Desktop-era analysis already produced TF-IDF data:

| Artifact | Location | Contents | Usable? |
|----------|----------|----------|---------|
| `vector_representations.json` | `data/classification/` (gitignored) | TF-IDF vectors, 74 dimensions, 5 sample inputs + 5 training | **No** — sample only, not full corpus |
| `similarity_matrix.json` | `data/classification/` (gitignored, 38KB) | Pre-computed similarity between items | **Maybe** — depends on coverage |
| `feature_dictionary_enhanced.json` → `lexical_features.keywords[].idf` | `data/classification/` (gitignored) | IDF weight per keyword (229 terms) | **Yes** — IDF weights ready to use |
| `algorithm_parameters.json` → `similarity_methods.text` | `data/classification/` (tracked) | Documents that `keyword_tfidf` was the chosen method | **Yes** — design reference |
| `algorithm_parameters.json` → `preprocessing` | `data/classification/` (tracked) | Tokenization rules (lowercase, no accent strip, min word length 2) | **Yes** — same preprocessing for consistency |

### Implementation Approach

**Vocabulary:** The 229 keywords in `feature_dictionary_enhanced.json` already have IDF
weights. This is the vocabulary. No need to recompute from scratch.

**Dimensions:** 229 (one per keyword in the dictionary). The Desktop-era analysis used
74 dimensions — that was a reduced vocabulary. Using the full 229 gives better coverage.

**Steps to implement:**
1. **Build vocabulary index:** Load all 229 keywords + IDF weights from feature dict
2. **Vectorize training data:** For each of the 694 training entries, tokenize `item`,
   compute TF-IDF vector (229 dims). Cache these vectors (one-time computation on startup).
3. **Vectorize input:** Tokenize the input `item`, compute its TF-IDF vector
4. **Cosine similarity:** Compare input vector against all 694 training vectors
5. **Top-K selection:** Return the K training entries with highest cosine similarity

**Computational cost:**
- Vectorization: O(V) per item, V=229 — trivial
- Cosine similarity: O(N×V) per classify call, N=694, V=229 — ~159K multiplications — trivial
- Total: sub-millisecond on any modern CPU, no caching needed for this dataset size

**Go implementation notes:**
- No external library needed — TF-IDF + cosine similarity is straightforward in Go
- Sparse vectors (most items have <10 tokens) — could use map[int]float64 but dense
  []float64 is fine at 229 dimensions
- Pre-compute training vectors at startup, store in memory (~694 × 229 × 8 bytes ≈ 1.2MB)

### When to Add This Layer

**Trigger:** After 5.7 (keyword matching) ships, measure:
1. **Keyword miss rate** — what % of classify calls have zero keyword matches?
2. **Accuracy on misses** — when keywords don't match, does the model still classify correctly?
3. **Multi-word ambiguity** — are there cases where individual keywords are ambiguous but
   the full phrase would disambiguate?

If keyword miss rate > 10% or accuracy on misses < 70%, TF-IDF is worth adding.

The `classifications.jsonl` feedback loop provides this data naturally:
- Entries with `status: corrected` where keyword matching found examples → keyword quality issue
- Entries with `status: corrected` where keyword matching found nothing → retrieval gap (TF-IDF candidate)

### Relationship to Other Layers

- **Does not replace keywords:** Keywords with specificity 1.0 are faster and more reliable
  than cosine similarity. TF-IDF is a fallback for when keywords fail.
- **Does not replace embeddings:** TF-IDF is still lexical — it cannot bridge semantic gaps
  like "99 Taxi" ≈ "Uber Centro". See § `data/classification/embedding-retrieval.md`.
- **Complements both:** In the cascade (§ [ref:retrieval-strategy]), TF-IDF sits between
  keywords and embeddings, handling the "multi-word context" gap that keywords miss but
  that doesn't require full semantic understanding.

### Preprocessing Rules

Must match the rules in `algorithm_parameters.json` → `preprocessing`:
- Lowercase: yes
- Remove accents: **no** (Portuguese keywords retain accents: `gás`, `habitação`)
- Remove special chars: yes
- Normalize whitespace: yes
- Minimum word length: 2 characters

### Vocabulary Growth

When historical workbooks are extracted (future task), the vocabulary will grow beyond 229.
The TF-IDF implementation should:
- Recompute IDF weights when vocabulary changes (IDF depends on document frequency)
- Support incremental vocabulary updates without full recomputation of all training vectors
- Or simply recompute all vectors on startup — at 694 entries × 229 dims, this is <1ms
<!-- /ref:tfidf-retrieval -->
