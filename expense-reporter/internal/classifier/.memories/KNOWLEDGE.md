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
  **RESOLVED —** the sentinel path shipped (T-19; `SentinelPath` appended to the enum). The
  "defeats the 0.85 threshold" half is moot: that threshold no longer exists (T-32 replaced it
  with the agreement gate, which routes any model/keyword disagreement to review).

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
  **SUPERSEDED (T-32) —** calibration was not fixed; the confidence gate was *deleted*. The
  649-replay confirmed the finding on real data (52.5% → 56.3% across the whole confidence
  range) and the gate switched to keyword agreement. WS-D remains held, but now on
  representativeness (the 649 is a confidence-selected subset), not on calibration.
- **`--think` flag (Config.NoThink → request `"think":false`, omitted by default):**
  q3 no-think = 59.7% at **1.5 s/item (10×)**, grammar intact, zero parse failures —
  but OOD sentinel-decline drops 2/20 → 0/20 with maximally absurd confident picks.
  Default stays think-on; batch runs can opt into speed. **[SUPERSEDED s62/T-24:
  default is now `--think=false` on classify/auto/batch-auto — gate band measured
  −2.3pp WITH think, 5.R2 miss path +2.5pp at 5.5× latency, sentinel covered by the
  T-32 agreement gate. `--think` opts thinking back on.]**
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

## Leaf-First Classification — planned + D1 locked (session 49)
Plan: `.claude/plans/leaf-first-classification.md`. Standalone accuracy change AND the
precondition that makes the T-23 confidence signal legible. Split from T-23 so it can be judged
on accuracy alone.
- **Change:** enum = the 104 unique **leaf** names (not 112 full paths); derive `(type,category)`
  upward via `taxonomy.ResolveLeaf`. Mirrors the manual `add` path (which already does this).
  Attacks T-14's dominant "right sheet, wrong leaf" error class; puts the model's strong-signal
  decision (the leaf, cf. "Uber") first instead of forcing an abstract Type guess first.
- **D1 LOCKED = second `type` field.** Schema emits **`leaf` THEN `type`** (order load-bearing:
  leaf first keeps the leaf-first commitment; type second is a real *recurrence* judgment).
  Always-predict `type`; `ResolveLeaf` consults it ONLY for the 5 cross-type collisions and
  cross-checks the 99 unique leaves as a free calibration signal. Grounded in the audit: the
  collision axis IS recurrence (pets `Ambos`/`Lilly`/`Orion` ×3, `Dentista`/`Estacionamento` ×2
  span Fixas/Variáveis/Extras) — not derivable from the leaf name.
- **A/B DONE (session 50) → D4 = HOLD (do not adopt for accuracy).** D2=tree, D3=benchmark-only
  (`.claude/scratch/leaf-first-ab/`), GBNF property-order (leaf-then-type) empirically confirmed.
  4 cells (few-shot OFF/ON × no-think/think-on, n=300 paired, McNemar): full-path Δ decays
  monotonically +12.7 → +8.7 → +3.3 → **+1.7 (production = few-shot ON + think-on, p=0.46, NOT
  sig)**. The FS-A gain was real but **few-shot retrieval + thinking already fix the same
  "wrong-leaf" errors** (sub-additive stacking). Type Δ survives longest (derived lookup off the
  leaf: sig in 3/4 cells) but also washes out in prod (+2.3, p=0.30). Not measurably *worse* (all
  prod deltas mildly +), but underpowered to prove a true null — "too small to justify a rewrite,"
  not "costs nothing". Report: `.claude/t14-benchmark-report.md` "T-30". Full log:
  `.claude/plans/leaf-first-classification.md`.
- **Still valuable ONLY as the T-23 precondition, SCOPED:** proven for the logprob-margin/perplexity
  signals (2a/2b); OPEN/possibly-unneeded for the ensemble-agreement signal (3). If T-23 goes
  ensemble, leaf-first is fully on the shelf. `type_crosscheck` is NOT a usable calibration signal
  (sign-flips across runs; disagreements → 0 as accuracy rises). Couples to T-27 (few-shot already
  recovers most wrong-leaf errors → re-justify T-27 or drop).
- **Taxonomy audit DONE (session 49):** all 112 leaves clean except `Apoia-se 4i20` (a leaked
  description auto-minted as a leaf in 5.R4 — appeared identically as `item` AND `subcategory`).
  Generalized to vendor leaf `Apoia-se`; full surgical propagation (labels only, real `item`
  text kept) across taxonomy + feature dict + training + logs; `*.bak-apoia-rename` backups.
  **Durability gap:** the leaf must also be renamed in the workbook Referência (T-02 export
  source) or the next re-export wipes it.

## Auto-insert gate — AGREEMENT (T-32 SHIPPED) + T-23 route history (session 50)
Full decision: `.claude/plans/t23-calibration-benchmark.md` "Route decision"; strategy:
`.claude/t23-strategic-implications.md`. [[project_logprob_confidence_leaf_first]].
- **SHIPPED (T-32): the gate is now AGREEMENT, confidence dropped.** `IsAutoInsertable(result,
  MatchSignal, excluded)` (`decision.go`) requires: a keyword matched, unambiguous, `top_score>=1.0`,
  model subcategory == keyword dominant subcategory, and not excluded. The old confidence-only gate
  (`Confidence >= 0.85 && ∉ excluded`) is gone — T-14/replay proved confidence anti-informative
  (86–91% of wrong ≥0.85). **Implementation note:** the E1 signal is surfaced as a PURE
  `MatchStrength(item, keywords) MatchSignal` in `examples.go` computed at the gate sites — NOT the
  planned `Classify` return-shape change (per advisor: the signal is model-independent, so a pure
  function avoids threading + case-(b) contamination where a high-spec keyword falls to the embedding
  path). Validated on the 649-replay: shipped `top_score` == harness `top_score` on all 649 rows,
  reproducing FINDINGS 34.2%/86.9% band + 94.9% agreement subcat precision (same-sample relative).
  **Absolute precision MEASURED s72 at 94.1%** (1 of 17 gate-passing rows corrected on the first
  real close, vs 50.0% error on the 48 gate-refused rows) — see
  `.claude/b1-gate-precision-measurement.md`. WS-D stays held anyway: n=17, and confirms on
  auto-inserted rows are weak-positive because the page shows them as already handled.
- **The recurrence signal is computed then DISCARDED.** `SelectExamples` (`examples.go`) derives
  per-subcategory keyword-specificity, branches on `sorted[0].score >= 0.7`, returns only
  `[]Example`; the score never escapes the function (grep: `subcatScore`/`.score` live only in
  `examples.go`) — never reaches `Result`, the command, or the gate.
- **Route reframe: external validators > model-introspection.** The dominant discriminator is
  RECURRENCE (leaky ≈70% vs clean ≈45–49%, ~25pp, orthogonal to self-confidence), not any token
  signal. Gate family = **external validators**: E1 recurrence-strength (specificity — exists,
  discarded), E2 value-range plausibility (`feature_dictionary_enhanced.json` `value_ranges`; task
  5.R3; use as a NEGATIVE flag), E3 retrieval-generation agreement (CAVEAT: few-shot injection
  breaks independence like `type_crosscheck` — test 4-cell stability first). Model-introspection
  (ensemble M1 cheapest / logprob M2 needs leaf-first+engine) is tested ONLY on the NOVEL pool,
  where recurrence gives nothing — do NOT assume it can't work there ("can't gate a coin flip" is
  too strong; 49% is an average).
- **First study (unblocked):** recurrence-strength (E1) risk–coverage stratified recurrent/novel,
  +E2 negative flag. The curve decides whether any introspection signal is still needed. Plumbing
  to wire E1: expose match-strength from `SelectExamples` → thread out of `Classify` (return-shape
  change) → widen `IsAutoInsertable` → 3 call sites (`auto.go` ×2, `batch_auto.go`). No new inference.
- **Critical path:** this gate is the WS-D (T-09) unblock — recurrence-first means novel rows never
  reach the append path, so retiring the bare-name fallback stops risking confident-wrong novel rows.
- **Real-data measurement (session 51):** `classifications.jsonl` = 649 human-verified `(item→actual)`
  labels from real 2025 usage. **GOTCHA — do NOT measure the gate off the stored fields:** all rows are
  `model="review"` with DEFAULTED confidence (0.95×502, 0.85×38) and the review queue is selection-biased
  (602 corrected / 47 confirmed) → a naive accuracy/risk-coverage is an ARTIFACT (7.2%, flat — invalid).
  Real gate + 5.R1 keyword-miss-rate measurement = **REPLAY the 649 through the current classifier**
  (exclude self-matches — they're in the example pool now; leaky-vs-clean split). See
  [[project_t23_gate_route_strategy]] "Real-data grounding" + [[project_r1_evaluation_procedure]].

## 649-replay results — gate + retrieval measured on real data (session 52)
Reports: `.claude/scratch/replay-649/FINDINGS.md` (retrieval) + `FINDINGS-model.md` (gate).
Harness `replay_{retrieval,model}_test.go` (`//go:build replay`, in-package, faithful — drives real
`SelectExamples`/tokenizer + `Classify`). Advisor-stress-tested; SHIP vs HOLD split is load-bearing.
- **SHIP (bias-robust): confidence is DEAD** (52.5->56.3% flat; conf>=0.85 admits 97% at 53%).
  **Specificity (`top_score`) is a real MONOTONE discriminator -> replace confidence as the gate**
  (52->87% all / 44->83% clean full-path). **Best gate = AGREEMENT: spec==1.0 AND model==keyword-top1
  -> 95.0% subcat (201 rows)** = clears the auto-insert bar. Keyword-top1 > model on subcat at spec=1.0
  (91.0 vs 87.8) -> keyword-first/model-fallback hybrid worth designing. Frequency is NOT the axis
  (specificity is; high-freq generic tokens hurt); singleton-masquerade does NOT occur (0 freq<=1 drivers).
- **think-on top-band re-run: discriminator SHAPE robust to think mode** (band-lift spread 0.2pp, uniform),
  level -2.3pp (think-on slightly worse on high-spec) -> **the gate runs no-think, 10x faster, no loss**
  (feeds T-24).
- **HOLD (absolute level not representative): the 649 is a confidence-SELECTED review subset** —
  `expenses_log.jsonl` 725 unique expenses, only 347 reviewed; 378 (52%) bypassed review UNLABELED ->
  true full-stream precision UNMEASURABLE. So "no unattended auto-insert" + WS-D silent-insert scoping are
  HELD, not proven.
- **Retrieval: keyword miss rate 24.7%, 100% `no_keyword_match`, 160/160 misses have an in-pool same-subcat
  neighbor** -> 5.R1 TF-IDF ruled out (lexical can't bridge zero-overlap), **5.R2 embeddings is the lever**
  — but SOFTENED, gated on a cheap NN-retrieval precondition (embed the 160 misses + pool-mates, measure
  neighbor subcat hit-rate) = the settled NEXT step before building 5.R2. Miss pool is the accuracy
  sinkhole (18.8% full-path). Leakage controls: feedback LOO by item (self_match=0), training kept +
  clean(362)/leaky(287) stratified (44.2% vs 63.1% = ~19pp recurrence effect).

## 5.R2 Embedding Retrieval — Phase 1+2 (session 54, 2026-07-10)
Plan: `.claude/plans/5r2-embedding-retrieval.md` (locked D1–D12). Built by an Opus-medium
subagent, codegen via `my-go-q3-14b` (verdicts 0/1/0/1 — below qcoder's 2/2/1/1 for Go).
- **Two distinct keys by design:** the disk cache is keyed by **exact raw item text** (D6 —
  casing preserved, near-dup casings embed separately), while `TopKEmbeddingExamples` dedups
  the returned slate by the **lowercased/trimmed** key (`itemKey`, mirrors `MergeExamplePools`)
  so K=5 yields 5 distinct examples; among same-key rows the highest source-priority row wins
  (first-seen on ties).
- **Deterministic ranking:** sorts an **index permutation** (not the reps slice, not map
  iteration) so equal-similarity ties — guaranteed, because training duplicates share identical
  vectors — keep reproducible first-seen order. The Phase-4 A/B depends on this.
- **Cache robustness:** within-file vector-dimension mismatch is a hard error (fail loud, not
  garbage cosines); torn/malformed JSONL lines are skipped and re-embedded on next reconcile.
- `itemKey` deliberately named to avoid colliding with the `replay`-tagged test's `dedupKey`.
- All failures return errors (D9) so Phase 3 wiring can degrade to nil examples (today's
  keyword-miss behavior) — never panic, never block classification.

## 5.R2 Phases 3+4 — wired + ADOPTED (session 54, 2026-07-10)
- **Wiring:** `selectExamples` (classifier.go) falls back to `embeddingFallback` when
  `SelectExamples` returns nil (keyword miss ONLY — D1; weak/ambiguous matches keep the
  keyword path, CL1 revisits). `embedding_fallback.go`: package-level `embedState`
  (sync.Once) reconciles the pool cache lazily on the FIRST miss in a process — blocking
  by design (the triggering item needs the vectors; async would classify early/late misses
  inconsistently within one batch). Progress via `logger.Info` so batches don't look hung.
  Every failure degrades to nil examples — classification never blocks on retrieval.
- **Config:** `EmbedModel` (default snowflake-arctic-embed2) + `NoEmbedRetrieval` — zero
  values = ON with default model, so the commands needed no changes (D2).
- **A/B (replay, miss stratum n=80 unique):** full-path 18.8%→52.5% no-think (+33.8pp,
  29 fixed/2 broke) / 55.0% think-on. Think adds +2.5pp churn (10/8) at 5.5× latency →
  no-think is right for the miss path (feeds T-24). `FINDINGS-5r2.md`. Harness gained
  `REPLAY_MISS_ONLY` + `REPLAY_NO_EMBED` env knobs. NOTE: `GenerateID` collides for
  repeat expenses → replay outputs hold ~2 lines/id; score on unique ids.
- **First-activation burst:** full-pool embed = 1,744 items ≈ 3 min, one-time (CL2 has
  the pre-warm option if it ever annoys).

## 5.R2 method-extraction refactor (session 55, 2026-07-11)
PR #45 review pass applied the module-wide method-extraction convention (see
`expense-reporter/.memories/KNOWLEDGE.md`) to the 5.R2 files — pure refactor, tests untouched:
- `TopKEmbeddingExamples` → `dedupePoolByKey` (two-key design in its doc comment: cache =
  exact raw text, slate = lowercased `itemKey`) + `rankBySimilarity` (index-permutation
  determinism rationale now lives in its doc comment — load-bearing for replay A/Bs).
- `ReconcileEmbeddings` → `missingCacheItems` + `openCacheAppend` + `embedAndAppend`
  (doc comment states the resume property: items appended before a mid-loop failure persist).
- `LoadEmbeddingCache` loop → `parseCacheLine` (torn-line skip → re-embed contract).
Helper names were checked against `replay_*.go` (build-tag-hidden) before landing.
