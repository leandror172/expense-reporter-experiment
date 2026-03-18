# Few-Shot Retrieval Strategy — Expense Classifier

**Purpose:** High-level design for how the classifier selects and injects few-shot
examples into the Ollama prompt. Covers the layered retrieval pipeline, token budget
analysis, and data source strategy.

**Context:** This document was produced during 5.7 planning (session 10, 2026-03-18).
For implementation details, see the 5.7 plan file (TBD).

---

<!-- ref:retrieval-strategy -->
## Layered Retrieval Pipeline

Few-shot example retrieval uses a **cascade of methods**, not a single strategy.
Each layer handles a different class of input. The pipeline short-circuits at the
first layer that produces confident results.

```
Input: "Uber Centro"
         │
    ┌────▼─────────┐
    │   Keywords    │──→ specificity ≥ 0.7 → direct hit, grab examples → done
    └────┬─────────┘
         │ no match / ambiguous (specificity < 0.7)
    ┌────▼─────────┐
    │   TF-IDF     │──→ cosine > threshold → grab nearest examples → done
    └────┬─────────┘
         │ no good match
    ┌────▼─────────┐
    │  Embedding   │──→ semantic neighbors → grab examples → done
    └────┬─────────┘
         │ still nothing
    (send to model with no examples — taxonomy-only prompt)
```

**Key principle:** layers are **complementary, not mutually exclusive**. Each captures
a different signal type:

| Layer | Signal | Strength | Blind Spot |
|-------|--------|----------|------------|
| Keyword | Exact/partial lexical match | Fast, deterministic, interpretable | Synonyms, novel descriptions |
| TF-IDF | Term importance + multi-word similarity | Multi-token context, no model needed | Still lexical — can't bridge semantic gaps |
| Embedding | Semantic meaning (concept-level) | Handles synonyms, novel phrasing | Heavier, needs embedding model call |

**5.7 scope:** Keyword layer only + injection plumbing.
**Future layers:** TF-IDF (§ `data/classification/tfidf-retrieval.md`),
Embedding (§ `data/classification/embedding-retrieval.md`).
<!-- /ref:retrieval-strategy -->

---

<!-- ref:retrieval-token-budget -->
## Token Budget Analysis

Measured 2026-03-18 against current prompt and data files.

| Component | Tokens (est.) | Notes |
|-----------|---------------|-------|
| Current system prompt (taxonomy) | ~462 | 68 subcategory→category lines |
| Response schema (JSON) | ~80 | Structured output format spec |
| User message (item/value/date) | ~20 | Single expense input |
| **Baseline total** | **~562** | What we send today |
| Per few-shot example | ~20 | `item: X \| value: Y \| date: Z → Subcategory (Category)` |
| 5 examples | ~100 | |
| 10 examples | ~200 | |
| All 694 training entries | ~13,577 | Would consume 41% of 32K context |
| **32K context window** | **32,768** | qwen3-coder:30b (`my-classifier-qcoder`) |

**Conclusion:** Token budget is not the bottleneck. Even 10 examples (~200 tokens)
brings us to ~760 tokens — 2.3% of the context window. The constraint is
**signal-to-noise**: too many examples dilute attention or introduce contradictions.

**Recommendation:** Start with 3–5 examples. Measure accuracy. Increase only if
accuracy improves with more examples (diminishing returns expected after ~7).
<!-- /ref:retrieval-token-budget -->

---

## Data Sources for Examples

Two pools, used together:

### 1. `training_data_complete.json` — Static Training Set

- 694 labeled expenses (304 from 2024, 303 from 2025, 87 user corrections)
- Always available — no cold-start problem
- Schema: `{id, item, date, value, subcategory, category, source, year}`
- [ref:training-data-schema] for full schema

### 2. `classifications.jsonl` — Runtime Feedback

- Accumulates as the classifier runs (`auto`, `batch-auto`, `add` commands)
- Contains model predictions + user corrections (`status: confirmed/corrected/manual`)
- Schema: `{id, item, date, value, predicted_*, actual_*, confidence, model, status, timestamp}`
- Filter: exclude `status == "manual"` (no model prediction to learn from)
- [ref:training-data-schema] for full schema

### Merge Strategy

```
Example Pool = training_data_complete ∪ classifications.jsonl(status != manual)
```

When both sources have examples for the same subcategory, prefer:
1. **Corrected** entries from `classifications.jsonl` (highest signal — teaches failure modes)
2. **Training data** entries (pre-labeled ground truth)
3. **Confirmed** entries from `classifications.jsonl` (model agreed with itself — lower marginal value)

§ "Correction-Weighted Prioritization" below for rationale.

---

## Keyword Matching (5.7 Implementation Layer)

Uses `feature_dictionary_enhanced.json` → `lexical_features.keywords`.

**Current keyword statistics:**
- 229 keywords total
- 142 with perfect specificity (1.0) — unambiguous, single subcategory
- 60 with low specificity (<0.7) — ambiguous, maps to multiple subcategories
- 27 in between (0.7–0.99) — mostly unambiguous

**Algorithm:**
1. Tokenize input `item` (lowercase, remove special chars, split on whitespace)
2. Look up each token in the keyword dictionary
3. Collect candidate subcategories with their specificity scores
4. For high-specificity matches (≥0.7): select examples from the dominant subcategory
5. For ambiguous matches: select examples from top-2 candidate subcategories (diversity)
6. If no keyword matches: send to model with no examples (taxonomy-only fallback)

**Example format in prompt** (as user/assistant conversation turns):
```
user: item: Uber Centro\nvalue: 15.90\ndate: 05/03
assistant: {"results": [{"subcategory": "Uber", "category": "Transporte", "confidence": 0.95}, ...]}
```

---

## Correction-Weighted Prioritization

**Rationale:** Corrected entries in `classifications.jsonl` represent cases where the
model failed — showing these as examples directly addresses known failure modes.

**Pros:**
- Targets the model's specific weaknesses, not generic cases
- Small number of corrections can have outsized accuracy impact
- Data already exists with both predicted and actual labels

**Cons:**
- Cold-start: corrections accumulate slowly
- Over-representing corrections could make the model second-guess itself on cases it handles well
- Adds ranking complexity beyond simple keyword matching

**Decision:** Implement as a simple sort order (corrected > training > confirmed) within
keyword-matched results. No weighting scores. Revisit if correction volume becomes
significant enough to warrant more sophisticated prioritization.

---

## Prompt Extraction

The system prompt is currently hardcoded in `internal/classifier/classifier.go:buildSystemPrompt()`.
As part of 5.7, extract to a template file for tracking and iteration:

- **Location:** `data/classification/prompts/classify.tmpl` (tracked, no personal data)
- **Format:** Go `text/template` with `{{.Taxonomy}}`, `{{.TopN}}`, `{{.Examples}}` placeholders
- **Benefits:** Prompt changes visible in git history; can iterate without recompiling

---

## Deferred Enhancements

These are tracked as future tasks in `.claude/tasks.md`:

- **Value-range plausibility** — use `value_ranges` from feature dict to pre-filter or
  post-validate subcategory candidates. Orthogonal to few-shot injection.
  § `data/classification/algorithm_parameters.json` for threshold values.
- **TF-IDF retrieval layer** — second cascade layer for multi-word similarity.
  § `data/classification/tfidf-retrieval.md` for full analysis.
- **Embedding retrieval layer** — third cascade layer for semantic matching.
  § `data/classification/embedding-retrieval.md` for full analysis.
- **Historical workbook extraction** — extract labeled expenses from pre-2024 workbooks
  to expand the training set beyond 694 entries.
- **Retrieval-method-agnostic interface** — design the example selection as a pluggable
  function (`func SelectExamples(item, value, date) []Example`) so retrieval layers
  can be swapped/composed without touching classifier or prompt code.
