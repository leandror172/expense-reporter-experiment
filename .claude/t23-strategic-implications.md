# T-23 / Leaf-First — Strategic Implications for the Project

*Session 50 (2026-07-05). What the calibration/accuracy findings mean beyond the gate —
for the current workbook plan and for the project's grand vision. Companion to
`.claude/plans/t23-calibration-benchmark.md` (the decision) and
`.claude/plans/leaf-first-classification.md` (the A/B that produced HOLD).*

---

## 1. The empirical reality these benchmarks collectively established

Across T-14 (accuracy/calibration), T-22/T-26 (type descriptions), and T-30 (leaf-first),
a consistent picture has hardened — treat this as ground truth for planning:

1. **The local classifier tops out around 63% full-path accuracy** (q3 / qwen3:8b), and prompt/
   enum engineering moves it only a few points that wash out in production (descriptions +2.6pp,
   leaf-first +1.7pp n.s.). **The model is near its ceiling for this task via prompting.**
2. **The model's self-reported confidence is anti-informative** — 86–91% of WRONG answers sit
   ≥0.85. It cannot gate. (T-19: it's confidently wrong on out-of-domain items.)
3. **Recurrence is the dominant signal, by far.** Recurring items ≈70% vs novel items ≈45–49%
   (a coin flip) — a ~25pp gap that dwarfs every model-side lever. And recurrence is *external*
   to the model, so it can't be fooled by confident-wrong.
4. **Accuracy is bimodal, split by recurrence, not by confidence.** "Have we seen this before"
   predicts correctness; "how confident is the model" does not.

**One-line takeaway:** the leverage in this system is *retrieval / recurrence*, not model
cleverness or confidence.

---

## 2. What it means for the GRAND VISION

Vision recap: local-first personal-finance pipeline — expenses in (CSV, correction flows) →
classify (local LLM) → high-confidence auto-insert / low-confidence review → JSONL logs as the
single source of truth → `generate-workbook` (and, long-term, DB ingestion). Layer-5 goal as
written: "auto-classify + auto-insert high-confidence entries."

The findings **refine the vision rather than break it** — but the refinement is real:

### 2a. The vision bifurcates by recurrence, not confidence
The original "auto-insert high-confidence" framing implicitly assumed confidence discriminates.
It doesn't. The honest, achievable system is:
- **Recurring expenses (rent, utilities, subscriptions, regular merchants) — the bulk of a
  monthly personal stream — auto-insert at high precision** (gated by recurrence-strength, not
  confidence).
- **Novel expenses (new merchants, one-offs) — a coin flip — routed to review.**

This is a *good* reframe: a personal expense stream is dominated by recurring items, so gating on
recurrence automates the majority and turns the minority into a fast human pass. **The review UI
(RUI, already built) is not a fallback — it is a first-class half of the system.** The vision
shifts from "auto-classify everything" to **"auto-classify what recurs, assist-review what's new."**

### 2b. The review cost is a bootstrapping tax that AMORTIZES — the key virtuous cycle
Every reviewed novel item becomes a feedback example (`MergeExamplePools`: feedback > training),
so **today's novel item is tomorrow's recurring item.** The novel pool — and thus the review
burden — *shrinks over time* as the user's personal expense vocabulary gets covered. This is what
makes "review all novel" acceptable: it's front-loaded and self-decaying, not a permanent tax.
**This feedback loop is arguably the most important strategic asset in the project** and should be
protected/strengthened (it's why correction logging + example-pool merging matter more than any
single accuracy point).

### 2c. The bottleneck is RETRIEVAL, not the model — re-prioritize accordingly
Since prompt engineering has hit diminishing returns and recurrence is the lever, the
high-value track for *raising* accuracy (shrinking the coin-flip pool) is **better retrieval**:
- **5.R1 (TF-IDF)** and especially **5.R2 (embeddings)** are the real accuracy investments —
  embeddings turn "novel" into "recurring-like" by finding semantic near-neighbors, directly
  attacking the 49% pool.
- Further prompt/enum tuning (T-27 leaf descriptions, more leaf-first variants) is **low-ROI** —
  T-30 showed few-shot already recovers most of what those target. T-27 in particular is
  undercut and must be re-justified on few-shot-ON evidence or dropped.
- A bigger/better *model* is a possible lever but hits VRAM limits (qcoder doesn't fit) and buys
  less than retrieval would; deprioritize.

### 2d. An external-validator gate is DB-ready and model-agnostic
Building the gate from external validators (recurrence-strength, value-range) rather than
model-introspection (logprobs/ensemble) is not just more robust — it **aligns with the long-term
"logs → DB" direction.** External validators are deterministic, auditable rules over structured
data (keyword stats, value ranges, history) — exactly what a DB-backed system does well, and they
survive a model swap. Model-introspection signals are local-LLM-specific and would not. So
external-first future-proofs the gate.

---

## 3. What it means for the CURRENT WORKBOOK PLAN

Workbook-plan state: `generate-workbook` is the only writer; JSONL logs are the source of truth;
WS-B (commands append to logs) shipped; **WS-D (retire the transitional bare-name routing fallback)
is BLOCKED on the gate**; WS-E (delete dead insert code) follows WS-D.

### 3a. This work IS the WS-D unblock — and the critical path is now concrete
WS-D was gated on "the gate protects" + "type-less count ≈0." The **recurrence-first gate is the
'gate protects' piece**: once auto-insert requires a strong recurrence match (and novel → review),
retiring the bare-name fallback no longer risks dumping confident-wrong novel rows into the log,
because novel rows never reach the append path. So the critical path is:

> **recurrence-strength study (T-23, cheap, unblocked) → recurrence-first gate → WS-D → WS-E.**

The first step is actionable today (no new inference, ~5-file plumbing, the T-14 harness already
supports the risk–coverage analysis).

### 3b. The workbook-generation architecture is VALIDATED and stable
`generate-workbook` reads the logs; the gate only decides what enters the logs automatically vs.
via review. Nothing in these findings touches the generator, the taxonomy, or the log schema — so
the "logs → generate" spine is confirmed sound. The open question is *purely* the ingestion gate.

### 3c. Downstream items are unaffected but benefit
Year-rollover (T-03) and the DB direction don't depend on the gate choice, but they inherit a
deterministic/auditable gate rather than an opaque confidence threshold — a net simplification.

---

## 4. Honest risks / ceilings

- **The recurrence gate might not reach usable precision even on recurring items.** 70% recurring
  is still 30% wrong; the risk–coverage curve must prove the *top* of the specificity range
  (uber/spotify ≈1.0) is ~95%+. If it isn't, auto-insertion of even recurring items is in
  question and the system leans further toward review. The study decides.
- **Novel items may be un-gateable.** If no signal separates correct-from-wrong on the novel pool,
  novel = always review, permanently — capping automation at the recurring fraction. For a
  personal tool that's acceptable (and 2b's amortization softens it); for a "full automation"
  ambition it's a ceiling only better retrieval (5.R2) or a better model can lift.
- **The amortization loop assumes the user actually reviews + the feedback is captured.** If
  review is skipped or corrections aren't logged, the virtuous cycle stalls and the novel pool
  never shrinks. The review→feedback→example-pool path is load-bearing and must stay frictionless.

---

## 5. Recommended re-prioritization (for a later decision — not yet acted on)

1. **T-23 recurrence-strength study** — cheap, unblocked, and the WS-D critical-path opener. Do first.
2. **Protect/strengthen the feedback loop** (correction logging, example-pool merging) — the
   amortization engine; highest strategic ROI, mostly already built.
3. **Retrieval investment (5.R2 embeddings > 5.R1 TF-IDF)** — the real lever to shrink the novel pool.
4. **Deprioritize** further prompt/enum accuracy work (T-27 leaf descriptions unless re-justified;
   leaf-first parked; more descriptions) — proven low-ROI.
5. **Model-introspection signals (ensemble, then logprob)** — only if the novel-pool study shows an
   external gate leaves safe novel auto-insertion on the table.

*Nothing here is decided — it's the strategic frame for the gate decision the user is deferring.*

---

## 6. Real-data grounding (session 51) — the measurement is unblocked; the labels already exist

An earlier draft of §5 implied we'd need to *generate* real data (run `batch-auto` on a fresh month).
Wrong — it already exists: **`classifications.jsonl` = 649 human-verified `(item → actual category)`
labels** from real 2025 usage (review/apply pipeline, WS-B slice 4).

- **Observation that matters strategically:** every one of the 649 went through the **review-first**
  path (`model="review"`), so the auto-insert gate was never battle-tested on real data. You've been
  running the recurrence/review-first vision *by hand* already — corroborating §2a.
- **Do NOT measure the gate directly off this file.** The confidences are review DEFAULTS (0.95×502,
  0.85×38) and the queue is selection-biased (602 corrected / 47 confirmed) — a naive "accuracy" reads
  7.2% with a flat risk–coverage curve, which is an **artifact**, not a finding. (Lesson: check the
  `model` field before trusting any confidence in a feedback log.)
- **The real move — REPLAY the 649 labeled items through the current classifier.** One pass yields the
  real gate risk–coverage (→ WS-D readiness) *and* the real keyword miss rate (→ 5.R1 TF-IDF go/no-go).
  Keyword-miss rate is cheap (replay `SelectExamples`, no Ollama); real confidence needs the full run
  (~16 min no-think / ~2.5 h think-on). **Leakage caveat:** many of the 649 are now in the example pool
  → exclude self-matches / split leaky-vs-clean or the numbers flatter the classifier.

**Bottom line for the roadmap question:** insertion removal (WS-D/E) and TF-IDF are each *one
measurement away from a decision*, and that measurement no longer needs fresh data collection — the
expensive part (labeling 649 real expenses) is already done. Replay-and-measure is the unblocking step.
See [[project_t23_gate_route_strategy]] + [[project_r1_evaluation_procedure]].
