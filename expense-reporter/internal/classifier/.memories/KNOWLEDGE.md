# internal/classifier/ — Knowledge (Semantic Memory)

*Classifier package accumulated decisions. Read on demand by agents.*

## Few-Shot Selection Algorithm (2026-03-20)
`SelectExamples` implements a keyword-based retrieval pipeline:
1. **Tokenize** — lowercase, strip non-alphanumeric, keep tokens with ≥2 runes
2. **Keyword lookup** — each token checked against `feature_dictionary_enhanced.json`;
   accumulate per-subcategory max-specificity scores
3. **Branch on specificity:**
   - High (top score ≥ 0.7) → examples from dominant subcategory only
   - Ambiguous (< 0.7) → interleave examples from top-2 subcategories
4. **Sort by source** — Corrected first, then Training, then Confirmed
5. **Truncate** to topK (default 5)
**Rationale:** High-specificity keywords (e.g., "uber" → Uber/Taxi at 1.0) give clean
signal. Ambiguous keywords benefit from showing the model both plausible categories.
**Implication:** The 0.7 threshold was chosen empirically. Items with all-ambiguous
keywords get diverse examples; items with one strong keyword get focused examples.

## Prompt Architecture (2026-03-20)
Messages sent to Ollama's `/api/chat` endpoint:
1. **System** — "You are an expense classifier" + full taxonomy (subcategory → category list)
2. **Few-shot pairs** — user message (item/value/date) + synthetic assistant response
   (subcategory/category at 0.95 confidence). Up to 5 pairs.
3. **User query** — the actual item to classify
The `format` parameter sends a JSON schema that forces Ollama to return valid structured
output matching `{"results": [{"subcategory", "category", "confidence"}]}`.
**Rationale:** Ollama's structured output (`format` param) eliminates parsing failures.
Few-shot pairs as user/assistant messages (not system prompt examples) follow the
chat completion convention and empirically produce better results.
**Implication:** The full taxonomy is injected every request (~2K tokens). This is
acceptable for local models but would need chunking for API-based models.

## Example Pool Merging (2026-03-20)
`MergeExamplePools` combines training data (static JSON) with feedback (JSONL).
Feedback entries take precedence: if the same item appears in both pools, the
feedback version is kept. Deduplication key is `strings.ToLower(strings.TrimSpace(item))`.
**Rationale:** Feedback entries are more recent and may contain corrections. A corrected
entry should replace the original training example, not coexist with it.
**Implication:** As the feedback pool grows, training data gradually gets superseded
by real-world confirmed/corrected classifications.

## Taxonomy Loading — SUPERSEDED by T-13 (session 41)
~~`LoadTaxonomy` reads `category_mapping` from `feature_dictionary_enhanced.json`.~~
`classifier.LoadTaxonomy` and the `Taxonomy` type are **deleted**. The classifier now
loads the 3-level tree from `config/taxonomy.json` via `internal/taxonomy.LoadTaxonomy`
(passed in as `[]taxonomy.ExpenseType`) and predicts the full path against it. The feature
dictionary is now **keyword-only** (few-shot selection); it is no longer a category/type
authority. Category and type both come from the predicted path — they can never disagree.
`add` resolves its path via `taxonomy.ResolveLeaf` (+ `--type` for ambiguous leaves).

## Prompt Architecture — UPDATED by T-13
The system prompt now renders the `type → category → subcategory` tree (not a flat
`sub→cat` list), and the `format` schema constrains each candidate to a `path` enum string
(`Type/Category/Subcategory`) drawn from `taxonomy.PathEnum`. Few-shot assistant messages
emit `{"results":[{"path":...,"confidence":0.95}]}`. `parseResponse` splits each path back
to `(type,cat,sub)` via `PathMap.Split` and drops any off-enum path.

## Enum validity is GRAMMAR-enforced, not model-dependent (session 42 — corrects the above)
The session-40 claim "qcoder honors the 112-enum 100%, so commands default to qcoder" was a
**measurement artifact**. Ollama's `format` schema with an `enum` compiles to a GBNF grammar that
constrains the sampler, so **any** model is forced to emit valid enum members — verified: a 9B model
(`my-classifier-q35`) returns valid 112-enum paths AND is forced to map an out-of-domain item
("airplane to Tokyo") into the enum. So:
- **Default reverted to `my-classifier-q3`** for all commands (qcoder = qwen3-coder:30b, 20.7 GB, does
  NOT fit a 12 GB GPU → CPU-offload + load-time 500s, with zero validity benefit). q3 validated
  end-to-end on the real 112-path taxonomy (Uber→Uber/Taxi 100%). q3 is **accurate but slow** (~12 s/call,
  likely qwen3 "thinking" tokens) — accuracy+speed across q3/q35/qcoder is the open benchmark (T-14).
- `splitResults`'s off-enum drop is now near-dead code (the grammar rarely lets an off-enum through).
- **Lost safety net (T-19):** the atomic enum gives the model no "none of these" option — novel/out-of-domain
  expenses are forced into a leaf, sometimes at high confidence, which can defeat the 0.85 auto-insert
  threshold. The pre-T-13 algorithm had an explicit `Diversos`/`require_manual_review` escape. Consider a
  sentinel path.

## Empirical Findings (2026-03)
- **Multi-word context beats keyword specificity:** "VA compras" classifies correctly
  at 100% despite "va" having specificity=0.36 — the LLM understands the phrase
- **"Diversos" false positive:** The model confidently assigns the catch-all category.
  Blocked via exclusion list rather than prompt engineering — more reliable
- **Cold-start timeouts:** First Ollama call after model load takes 10–30s. This is
  normal (model loading to GPU). Retry policy: first timeout = retry, not rejection
- **Specificity=1.0 keywords** (e.g., "uber", "spotify") are near-perfect retrievers.
  The few-shot examples they select almost always lead to correct classification

## Empirical Findings (2026-03)
- **Multi-word context beats keyword specificity:** "VA compras" classifies correctly
  at 100% despite "va" having specificity=0.36 — the LLM understands the phrase
- **"Diversos" false positive:** The model confidently assigns the catch-all category.
  Blocked via exclusion list rather than prompt engineering — more reliable
- **Cold-start timeouts:** First Ollama call after model load takes 10–30s. This is
  normal (model loading to GPU). Retry policy: first timeout = retry, not rejection
- **Specificity=1.0 keywords** (e.g., "uber", "spotify") are near-perfect retrievers.
  The few-shot examples they select almost always lead to correct classification
