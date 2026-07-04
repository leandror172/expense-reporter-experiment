# T-23 Calibration Benchmark — Plan (option "c")

**Status:** proposed (session 49). Depends on the leaf-first probe finding
(`.claude/t23-logprob-confidence-probe.md`). Gates WS-D (T-09).

**PRECONDITION — do leaf-first FIRST.** The non-baseline signals here assume a leaf-first
classify path (that's what makes any token signal legible — see the probe report). Leaf-first
is now its own change with its own accuracy A/B: `.claude/plans/leaf-first-classification.md`.
Run/adopt that first; this benchmark then rides on a leaf-first base. Sequencing intent: split
so leaf-first can be judged on accuracy alone, and the calibration signal on a base that already
uses the enum it needs.

## Objective & the decision it drives

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

Reuse the T-14 stratified set: **300 labeled corpus items** (80/112 leaves) + **20 OOD probes**
(`.claude/scratch/t14-benchmark/`). Labels = the known-correct leaf. No new labeling needed.

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
