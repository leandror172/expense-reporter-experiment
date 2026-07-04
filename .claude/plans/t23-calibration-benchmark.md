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

## Tooling — see the session-49 library rundown in the chat log / handoff.
Ensemble + perplexity need nothing beyond Ollama + Python (scikit-learn/scipy for metrics).
The logprob-**margin** arm (2b) is the only piece that may justify moving the benchmark off
Ollama to an engine that exposes renormalized/teacher-forced logprobs (llama.cpp server `n_probs`,
or vLLM `prompt_logprobs`, or Outlines-controlled masking). Decide in Procedure step 0.
