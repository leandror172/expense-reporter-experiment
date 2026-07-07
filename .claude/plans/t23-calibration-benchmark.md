# T-23 Calibration Benchmark — Plan (option "c")

**Status:** REFRAMED session 50. The "Route decision" section immediately below supersedes the
session-49 token-signal framing in "Objective/Signals under test" (retained, not deleted — it is
now the *model-introspection* detail, scoped). Gates WS-D (T-09). Strategic read:
`.claude/t23-strategic-implications.md`.

---

## Route decision (session 50) — external validators vs. model-introspection

**READ FIRST.** Written session 49, this plan assumed a *token-derived confidence signal* was the
answer. Session 50's leaf-first A/B (D4 = HOLD — no production accuracy gain, `leaf-first-classification.md`
"Execution log") + the recurrence data + an advisor review reshaped the route. Everything from
"## Objective" down is kept verbatim (no detail lost) but is now scoped as the introspection detail.

### The empirical picture forcing the reframe
- Self-confidence is dead as a gate (T-14: 86–91% of WRONG answers carry ≥0.85). Unchanged.
- **Recurrence is the single dominant discriminator:** leaky/recurring ≈70% vs clean/novel ≈45–49%
  full-path — a **~25pp gap, larger than any prompt/enum/think effect in the entire benchmark**, and
  ORTHOGONAL to the model's broken self-assessment.
- Leaf-first (T-30) gave no production accuracy gain → the logprob route lost its bundled accuracy
  co-justification; leaf-first is parked.

### The axis (the organizing idea)
- **External validators** — computed from data OUTSIDE the model's output, so they cannot be fooled
  by confident-wrong:
  - **(E1) Recurrence-strength** — keyword-specificity match. Already computed in `SelectExamples`
    then DISCARDED (see As-is). Cheapest; no new inference.
  - **(E2) Value-range plausibility (task 5.R3)** — per-subcategory `value_ranges` in
    `feature_dictionary_enhanced.json`. "R$5000 → Diarista (max 220)" is implausible. Use as a
    NEGATIVE flag / exclusion, NOT a positive gate (plausible ≠ correct). Data already present.
  - **(E3) Retrieval-generation agreement** — does the predicted leaf match the dominant subcategory
    the keyword index expected? **CAVEAT (advisor):** few-shot INJECTS the matched keyword's examples,
    nudging the model toward the retrieved answer → agreement is NOT two independent methods, and
    disagreement is ambiguous (smart multi-word override like "VA compras" vs. error). This is the
    exact non-independence that sign-flipped `type_crosscheck`. TEST stability across all 4 (fs×think)
    cells BEFORE building on it.
- **Model-introspection** — reads the model's own certainty; shares confidence's failure-mode risk
  but might work where the model is genuinely uncertain (variance) rather than confidently-wrong (bias):
  - **(M1) Ensemble agreement** — K samples, vote share of the top complete leaf. Cheapest
    introspection signal: needs NO leaf-first, NO engine (extract leaves from K sampled full-paths).
    Catches variance-uncertainty, not confident-consistent-wrong.
  - **(M2) Answer perplexity / logprob-margin** — needs leaf-first (legibility) + margin needs a
    teacher-forcing engine (llama.cpp/vLLM PoC, see Tooling/subagent below). Most infra, least-justified now.
  - Self-confidence (dead) = the degenerate M0 baseline.

### The decision (sequenced; the curve decides — NOTHING pre-shelved)
1. **First study — cheapest, unblocked TODAY: recurrence-strength (E1) risk–coverage, stratified
   recurrent/novel**, with value-range (E2) as a negative flag in the same pass. It is the first
   *study*, NOT the decided route — 70% recurrent is still 30% wrong, so the curve must PROVE the
   top of the specificity range (uber/spotify ≈1.0) reaches usable precision-at-coverage.
2. **The curve DECIDES whether model-introspection signals are still needed.** Do NOT pre-shelve
   them. If E1(+E2) can't reach usable precision even on the recurrent pool, a per-item signal is
   required after all.
3. **Model-introspection's UNIQUE test is the NOVEL pool** (advisor correction — I originally
   mis-targeted them at "refining the recurrent pool," which recurrence already handles). Their only
   defensible value is whether ANY signal separates correct-from-wrong among *novel* items — the sole
   path to auto-inserting them. So stratify risk–coverage recurrent/novel and ask, per introspection
   signal: does it work on the NOVEL subset? **"Can't gate a coin flip" is too strong** — 49% is an
   AVERAGE; an informative signal can carve a high-precision subset out of a low-average pool. Whether
   ensemble/logprob can is the empirical question — do NOT pre-answer "no."
4. **Introspection ordering if pursued:** ensemble (M1) first (no leaf-first/engine); logprob-margin
   (M2) last (needs leaf-first revival + engine PoC). **Leaf-first is parked but revivable — ONLY if
   the novel-pool study shows logprob is the one signal that works there.**

### As-is gate (verified session 50 — the change surface)
- **`classifier.IsAutoInsertable(result, threshold, excluded)`** (`internal/classifier/decision.go`)
  checks ONLY `result.Confidence >= 0.85 && result.Subcategory ∉ excluded`. That is the ENTIRE gate.
  It carries a `TODO(T-19)`: this threshold is the only guard and the model (GBNF-constrained) can't decline.
- **The recurrence signal is computed then THROWN AWAY.** `SelectExamples` (`examples.go`) computes
  per-subcategory keyword-specificity, branches on `sorted[0].score >= 0.7`, returns only `[]Example`
  — the score never leaves the function (grep-confirmed: `subcatScore`/`.score` appear ONLY in
  `examples.go`). It never reaches `Result`, the command, or the gate.
- **Wiring:** `auto.go` (2 sites) / `batch_auto.go` (1) call `classifier.Classify(...)` → `top :=
  results[0]` → `IsAutoInsertable(top, 0.85, excluded)` → append (`appender.ExpandAndAppend`) or review.
- **Plumbing cost for E1/E3 (no new inference, ~5 files):** expose match-strength (top specificity +
  dominant subcategory) from `SelectExamples`; thread it out of `Classify` (per-ITEM → a return-shape
  change, since `Result` is per-candidate); widen `IsAutoInsertable` to take the recurrence score/flags;
  update the 3 call sites. Contained, mechanical.
- **Value-range (E2):** `feature_dictionary_enhanced.json` already carries per-subcategory `value_ranges`
  (min/max/mean; I edited the Apoia-se block during the rename) — E2 = load them + compare to the
  predicted subcategory's range. Task 5.R3.

### Decided vs. open
- **DECIDED:** external validators are the primary-gate family; recurrence-strength (E1) is the first,
  cheapest, unblocked study; value-range (E2) folds in as a negative flag; self-confidence is retired
  as the sole gate.
- **OPEN (curve decides):** whether E1(+E2) alone reaches usable precision; whether any
  model-introspection signal works on the NOVEL pool; ensemble vs. logprob; whether to revive leaf-first
  (only if logprob wins novel).
- **DO NOT:** pre-shelve introspection signals; build on E3 before the 4-cell stability test; treat
  value-range as a positive gate.

---

## Objective & the decision it drives

> **SCOPE NOTE (session 50):** everything from here down was written session 49 as if a
> token-derived signal were THE answer. It is retained in full as the **model-introspection
> (M1/M2)** detail — the risk–coverage method, cost, tooling, and engine-PoC subagent all still
> apply *if* the novel-pool study (Route decision §3) shows an introspection signal is needed.
> The primary gate is now external validators (E1 recurrence-strength / E2 value-range); read the
> Route decision first.

Decide whether a **token-derived confidence signal** can replace the uninformative
self-reported `confidence` at the auto-insert gate. T-14 showed 86–91% of WRONG answers carry
self-confidence ≥0.85, so the 0.85 gate barely filters. This benchmark measures whether an
alternative signal actually separates the model's correct answers from its wrong ones.

**Exit decision:**
- A signal *bends the risk–coverage curve* (precision climbs materially as coverage drops,
  beating the self-confidence baseline with bootstrap CIs that don't overlap) → adopt it as the
  T-23 gate; WS-D unblocks.
- No signal beats baseline → retire the logprob idea; T-23 goes **recurrence-first** (auto-insert
  only few-shot/keyword-matched items, review everything novel — the direction T-14 already
  favors).

## Signals under test

| # | Signal | Cost/item | Notes |
|---|--------|-----------|-------|
| 1 | **Self-reported `confidence`** (BASELINE) | 1 call | current gate; the thing to beat |
| 2a | **Answer perplexity** — length-normalized raw sequence logprob of the chosen leaf | 1 call | computable from Ollama TODAY (sum of generated-token logprobs); leaf-first |
| 2b | **Logprob margin** — 2a minus the runner-up leaf's sequence logprob | 1 call + scoring | needs teacher-forcing (score an arbitrary alt string); **Ollama can't** — engine question (see Tooling) |
| 3 | **Ensemble agreement** — K samples at temp>0, vote share of the top *complete* leaf | K calls | immune to tokenization; needs no special lib; pushes toward no-think (cost) |

All non-baseline signals run **leaf-first** (enum = the 104 unique leaf names; type/category
derived upward via `taxonomy.ResolveLeaf`). Baseline stays on the current full-path enum so the
comparison is signal-vs-signal, not confounded by the enum change — but ALSO record 2a on the
full-path enum to isolate "leaf-first accuracy gain" from "signal quality."

## Sample

Two options, both already labeled:
- **T-14 stratified set** — 300 corpus items (80/112 leaves) + 20 OOD (`.claude/scratch/t14-benchmark/`).
  Balanced for coverage; NOT representative of the real expense distribution.
- **REAL usage set (session-51 finding, preferred for the gate decision)** — `classifications.jsonl`
  = **649 human-verified `(item → actual)` labels** from real 2025 usage (review/apply). This is the
  real distribution, so the risk–coverage it produces is the one that actually decides WS-D readiness.
  **Method = REPLAY these 649 through the current classifier** (the stored `confidence`/`status` are
  review DEFAULTS — `model="review"`, 0.95×502 — and selection-biased, so they are NOT usable directly;
  a naive read gives an artifactual 7.2% flat curve). **Leakage caveat:** many of the 649 are now in the
  example pool (`MergeExamplePools`) → they self-retrieve as few-shot twins → exclude self-matches /
  split leaky-vs-clean or the curve flatters the model. This same replay also yields the **5.R1 keyword
  miss rate** in one pass ([[project_r1_evaluation_procedure]]) — cheap part (miss rate) needs no Ollama
  (replay `SelectExamples` only); the confidence part needs the full run. See
  `.claude/t23-strategic-implications.md` §6 + [[project_t23_gate_route_strategy]].

## Metrics

1. **Risk–coverage curve (PRIMARY).** Sweep the signal threshold; plot coverage (fraction
   auto-inserted) vs precision (accuracy of the auto-inserted set). The gate metric. Report the
   operating point: **precision at 40% coverage** and **coverage at 95% precision**, per signal,
   vs baseline. Bootstrap 95% CIs (n≈300 → resample).
2. **AUROC** of signal vs correctness — single-number discrimination summary.
3. **Reliability curve + ECE** — is the signal's *magnitude* meaningful, or only its ordering?
   (Ordering is enough for a gate; calibration is a bonus.)

## Stratification (report each metric split by)

- **Leaky vs clean** — the load-bearing split. A signal only earns its keep on **clean/novel**
  items (T-14: 49.1%, coin flip); leaky items are already caught by recurrence. Do NOT let good
  leaky-item numbers mask a useless clean-item signal.
- **In-domain vs OOD** — does the signal drop on the 20 OOD probes (the safety case)?
- **think-on vs no-think** — accuracy and (for signal 3) cost both swing; couples to T-24.

## Procedure

0. **Tooling decision** (see Tooling section) — ensemble (3) + perplexity (2a) run on Ollama as
   is; margin (2b) is gated on picking a teacher-forcing engine. Ship 1/2a/3 first; 2b only if 0
   resolves cheaply.
1. **Leaf-first probe harness** — standalone script (Python, mirrors `run_t26.sh`) that builds the
   leaf-first enum from `config/taxonomy.json`, classifies each sample item, and records per item:
   predicted leaf, correct?, self-confidence, chosen-leaf token logprobs (→ 2a), and (if enabled)
   K ensemble votes (→ 3). **No production classifier change** — benchmark reads the same Ollama
   endpoint. Resume keys on a fresh results filename (T-26 gotcha).
2. **Run** — think-on primary; add a no-think pass for signals 2a/3 (cheap enough) and T-24 input.
   Watch the abandoned-request queue-starve ops gotcha (keep timeouts above worst case).
3. **Analyze** — `sigtest`-style script computes the three metrics + bootstrap CIs, stratified.
4. **Report** → new "T-23 Calibration" section in `.claude/t14-benchmark-report.md`; update
   classifier KNOWLEDGE.md + [[project_logprob_confidence_leaf_first]].

## Cost estimate

- Signals 1 + 2a: 1 call/item = 300 calls ≈ think-on 300×14 s ≈ 70 min; no-think ≈ 8 min.
- Signal 3 (K=7): 7 calls/item. Think-on ≈ 8 h (untenable for iteration) → **no-think only**
  (7×1.5 s×300 ≈ 53 min). This is why 3 couples to T-24.
- Total realistic first pass: ~2–2.5 h wall (mostly signal 3 no-think + signal 1/2a think-on).

## Risks / confounds

- **Pre-mask logprobs (proven):** 2a is the chosen path's own raw sequence logprob — valid. 2b's
  runner-up score is the hard part and the reason for the engine question.
- **Length bias:** normalize sequence logprobs by token count before comparing leaves.
- **Ensemble temperature:** too low → no diversity (fake agreement); sweep temp ∈ {0.5,0.7,1.0}.
- **Junk taxonomy leaves** (`Apoia-se 4i20`): audit `config/taxonomy.json` before the run so a
  garbage leaf doesn't distort the enum / OOD routing.
- **n≈300 resolves ~±5 pp; subgroup splits are noisier** — lean on bootstrap CIs, don't over-read
  a single stratum (T-26 lesson).

## Deliverables

- Probe harness + analysis script under `.claude/scratch/t23-calibration/`.
- Per-item results JSONL (gitignored — carries expense descriptions).
- Report section with the risk–coverage curves and the adopt/retire recommendation.

## Tooling — library rundown (session 49)

**Signals 1 / 2a / 3 need nothing beyond Ollama + Python — start there.** Only signal **2b**
(logprob margin to a runner-up leaf) needs capability Ollama lacks: renormalized (post-mask) or
teacher-forced logprobs. Engine options if 2b is pursued:

- **llama.cpp `llama-server`** — *recommended default.* Lightest, and it's literally the engine
  *under* Ollama (same GGUF weights, fits the 12 GB GPU). Native GBNF grammars; `/completion`
  with `n_probs` returns top-N per-token probs; driving the sampler yourself yields the **masked**
  distribution (the thing Ollama hides). Also supports supplying a continuation to score.
- **vLLM** — *most capable, heavier.* OpenAI-compatible `logprobs` on output **and**
  `prompt_logprobs` — that's teacher-forcing: feed a candidate leaf string, read its per-token
  logprobs → the runner-up sequence score for 2b. Plus `guided_decoding` (outlines/lm-format-
  enforcer backends) for the grammar. qwen3:8b fits 12 GB but VRAM headroom is tighter than
  llama.cpp.
- **Outlines** (Python) — structured-gen lib where *you own the logit processor*, so you can read
  the **renormalized "P over legal leaves"** directly — the cleanest fix for the pre-mask problem.
  Wraps transformers/vLLM/llama.cpp. Siblings: **lm-format-enforcer**, **guidance**,
  **llama-cpp-python** (in-process bindings with `logits_processor` hooks).

**Metrics — Python, not Go.** scikit-learn (`roc_auc_score`, `calibration_curve`) + scipy
(bootstrap CIs) + **netcal** (ECE/reliability). Matches the existing harness (`sigtest_t26.py` is
already Python). Go/gonum has AUROC but you'd hand-roll risk-coverage + calibration — not worth it
for a research script.

## Deferred subagent — engine PoC for signal 2b (only if 2b is pursued)

A self-contained, install-heavy investigation worth isolating from the main context. **Ask the
user which model to run it on before spawning** ([[feedback_ask_subagent_model]]).
- **Question it answers:** can we get, on *our* model (qwen3:8b) and the *actual 12 GB GPU*, clean
  masked and/or teacher-forced logprobs — and at what setup/VRAM/latency cost?
- **Tasks:** (1) stand up llama.cpp `llama-server` with the qwen3:8b GGUF; verify `/completion`
  `n_probs` + a driven GBNF grammar returns a *masked* per-token distribution (does it solve the
  pre-mask problem?); (2) test teacher-forced scoring of a supplied candidate leaf string →
  sequence score; (3) fall back to vLLM `prompt_logprobs` only if llama.cpp can't; (4) measure
  VRAM headroom (coexist with / replace running Ollama?), cold-start + per-call latency, setup
  friction.
- **Deliverable:** a short verdict doc — which engine, exact command/flags, does it produce usable
  masked + teacher-forced logprobs on our model, fits the GPU y/n, latency numbers, minimal repro.
  Enough to classify 2b as "cheap, do it" vs "rabbit hole, drop it."
- **Constraints:** no sudo (user-space installs / existing binaries only — sudo can't run through
  Claude Code); do NOT disrupt the running Ollama (GPU contention is a known ops gotcha); the
  pre-mask finding is the bar to clear; read-only w.r.t. the Go codebase.
- **Why not now:** gated on the user committing to the 2b arm. Signals 1/2a/3 run on Ollama today,
  so the natural order is run those first and only send the PoC if 2b is the piece worth chasing.
