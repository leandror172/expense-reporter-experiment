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

## Taxonomy Loading (2026-03)
`LoadTaxonomy` reads `category_mapping` from `feature_dictionary_enhanced.json`.
Returns `map[subcategory]category`. Used by both the classifier (system prompt) and
the `add` command (resolve category from subcategory for logging).
**Rationale:** The feature dictionary is the single source of truth for the category
hierarchy. Maintaining it separately from the Excel reference sheet allows offline
classification (without opening the workbook).
**Implication:** The taxonomy must stay in sync with the Excel reference sheet.
Discrepancies cause resolution failures in the `add` pipeline.

## Empirical Findings (2026-03)
- **Multi-word context beats keyword specificity:** "VA compras" classifies correctly
  at 100% despite "va" having specificity=0.36 — the LLM understands the phrase
- **"Diversos" false positive:** The model confidently assigns the catch-all category.
  Blocked via exclusion list rather than prompt engineering — more reliable
- **Cold-start timeouts:** First Ollama call after model load takes 10–30s. This is
  normal (model loading to GPU). Retry policy: first timeout = retry, not rejection
- **Specificity=1.0 keywords** (e.g., "uber", "spotify") are near-perfect retrievers.
  The few-shot examples they select almost always lead to correct classification
