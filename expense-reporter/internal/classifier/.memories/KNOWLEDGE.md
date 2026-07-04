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

## T-13 Mechanics + Model Default (sessions 41–42, consolidated from QUICK.md 2026-07-01)
Few-shot examples resolve their canonical path via `ResolveLeaf`+`PathFor` (dropped if
unresolvable); `Example.TypeHint` comes from the training `source` sheet-name. The system
prompt renders the 3-level tree. See `.claude/t13-implementation-report.md`.
Default model unified to `my-classifier-q3` for ALL commands (session 42): the earlier
qcoder default was reverted because the session-40 "qcoder honors the enum 100%" smoke test
measured a property of the GBNF grammar, not the model — qcoder (qwen3-coder:30b, 20.7 GB)
only added CPU-offload latency + load-time 500s on the 12 GB GPU. q3 validated end-to-end
on the real 112-path taxonomy (Uber→Uber/Taxi 100%) but is slow (~12 s/call); T-14
benchmarks accuracy+speed across q3/q35/qcoder.

## Training Corpus Expansion (2026-06-20, 5.R4)
Historical workbook extraction 2022–2025 deduped + merged into
`training_data_complete.json`: 694→1788 examples, 15 cats / 81 subs, multi-year. Bigger
few-shot/keyword pool; classifier now emits expense `type`.
See [[project_workbook_extraction_5r4]].

## T-14 Benchmark Results + --think Flag (session 46, 2026-07-02)
Full report: `.claude/t14-benchmark-report.md`; harness `.claude/scratch/t14-benchmark/`.
300 stratified corpus items (80/112 leaves) + 20 out-of-domain probes, real binary path.
- **q3 think-on: 63.0% full-path / 77% type / mean 14.4 s.** Leaky (few-shot twin
  exists) 71.7% vs clean 49.1% — novel items are a coin flip.
- **Calibration is the blocker, not accuracy:** ~91% of WRONG answers carry
  confidence ≥0.85 in BOTH think modes → the 0.85 auto-insert threshold filters
  almost nothing. WS-D (retire bare-name fallback) gated on fixing this, not on
  raw accuracy.
- **`--think` flag (Config.NoThink → request `"think":false`, omitted by default):**
  q3 no-think = 59.7% at **1.5 s/item (10×)**, grammar intact, zero parse failures —
  but OOD sentinel-decline drops 2/20 → 0/20 with maximally absurd confident picks.
  Default stays think-on; batch runs can opt into speed.
- **qwen3.5 + think:false SILENTLY DROPS the `format` grammar (Ollama 0.17.5)** —
  structured output broken (verified at API level); `/no_think` soft switch also
  ignored by qwen3.5. With thinking on, q35 runs 70–240 s/item (unbounded thinking).
  **q35 disqualified as default**; the no-think fast lane is q3-only.
- **Ops gotcha:** client timeouts don't stop server-side generation — abandoned
  grammar+thinking requests queue-starve Ollama (GPU pegged, later calls hang until
  restart). Keep benchmark timeouts above worst case.
- T-19 sentinel is structurally present but behaviorally near-dead (10% decline rate
  on OOD at best); out-of-domain safety must come from review/exclusion, not confidence.

## T-22 Type Descriptions Adopted (session 47, 2026-07-03)
Optional per-type `Description` rendered parenthetically on the type header line of the
system prompt (`writeTaxonomyTree`) — additive, no enum/grammar/routing change. Sourced
from a tracked sidecar `config/type-descriptions.json`, overlaid at classify-time
(`taxonomy.LoadTypeDescriptions`/`ApplyDescriptions`, applied in cmd `loadTaxonomyTree`;
`generate-workbook` unaffected). No-think A/B (300 items, q3): **English +6.0 pp TYPE
accuracy** (74.3→80.3), +2.6 pp full-path; both EN and PT lift type ~5–6 pp (→ real, not
q3 noise); mechanism = T-14's dominant "right sheet, wrong leaf" error, and descriptions
are type-level so leaf disambiguation is untouched (argues for a future leaf-level
extension). English > Portuguese (dropped). Small −2.6 pp on novel items. **Calibration
UNCHANGED** (HC-wrong ~91–95%) — NOT a gate fix (that's T-23). OOD 0/20 at no-think
(sentinel dead without thinking). Deferred: think-on confirmation (only mode with a live
sentinel; couples to T-23/T-24). Report addendum in `.claude/t14-benchmark-report.md`.
[[project_t22_type_descriptions]].

## T-23 Probe — Token Logprobs as a Confidence Signal (session 49, 2026-07-04)
Full report: `.claude/t23-logprob-confidence-probe.md`. [[project_logprob_confidence_leaf_first]].
Exploring whether the model's token distribution can replace the uninformative self-reported
`confidence` at the auto-insert gate (the WS-D blocker). Empirical (Ollama 0.17.5, q3, think:false):
- **Logprobs ARE exposed** on `/api/chat` (`logprobs:true` + `top_logprobs:N`) under the active
  `format` enum grammar. Feasibility gate passes.
- **BUT Ollama reports PRE-grammar-mask logprobs** — the raw unconstrained distribution, not the
  one renormalized over legal tokens. Proof: a schema-forced key (`"leaf"`/`"path"`) reports the
  chosen token at p≈0 while the model's wanted token (`"category"`) shows 1.0. Legal tokens are
  often outside `top_logprobs`, so the renormalized distribution can't be reconstructed. This
  **overturns** the initial "grammar renormalizes → token-dist == taxonomy-dist" assumption.
- **Type-first enum fights tokenization** — forces the abstract Type decision first (leaf last);
  the model wants to emit the leaf token ("Uber") but is dragged onto a Type rail (chosen tokens
  raw p≈0), picks the wrong subtree, locks in. Margin unreadable.
- **Leaf-first enum fixes it** — enum = bare leaf names, derive type/cat upward via `ResolveLeaf`
  (mirrors the `add` path). The decision token is the model's own high-prob token with readable
  competitors (`Uber` leaf p=0.986). Real decision concentrates at ONE branch; trailing tokens
  just complete the unique leaf.
- **A single token ≠ the margin** — first char = "which initial letter" (many leaves share it),
  not "which leaf." Correct signal = sequence logprob of the whole leaf + margin to runner-up.
- **Calibration NOT proven** by these probes (single-token, n≈4). Needs the T-14 labeled set.
  Encouraging n=1: `Drone DJI Mavic` (novel) showed real uncertainty (0.56) and routed to
  `Diversos`. Two signals to benchmark: logprob-margin vs K-sample ensemble agreement.
- Side flags: junk taxonomy leaf `Apoia-se 4i20` (audit `config/taxonomy.json`); think-on
  prepends reasoning tokens that move the JSON/decision span (aggregation must find it first).
