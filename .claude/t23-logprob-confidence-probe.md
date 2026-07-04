# T-23 Probe — Token Logprobs as a Confidence Signal (leaf-first)

**Session 49 (2026-07-04). Model `my-classifier-q3` (qwen3:8b), Ollama 0.17.5, `think:false`.**
Exploratory probe for T-23 (calibration / auto-insert gate rethink). Feeds T-27, T-28.

## Why

T-14/T-26 proved q3's **self-reported** `confidence` is uninformative: ~86–91% of WRONG
full-path answers still carry confidence ≥0.85, so the 0.85 auto-insert gate filters almost
nothing (the WS-D blocker). Idea: the model already produces a probability distribution over
next-token candidates; when output is grammar-constrained to the taxonomy enum, those
candidates *are* the available labels. Can we read a real confidence off the token
distribution instead of trusting the self-reported scalar?

## Setup

- Ollama native `/api/chat`, `stream:false`, `format` = JSON-schema `enum` (→ GBNF grammar).
- Request `logprobs: true`, `top_logprobs: N` — **both supported in 0.17.5** and returned
  even while the `format` grammar is active. Feasibility gate PASSES.

## Finding 1 — Ollama reports PRE-grammar-mask logprobs (overturns the initial theory)

The reported logprobs are the model's **raw, unconstrained** next-token distribution, NOT the
grammar-renormalized distribution over legal tokens. Proof — the JSON key token, where the
schema makes exactly one key legal:

```
tok 'leaf'  p=0.000   top=[category=1.000, .category=0.000, item=0.000]
```

The grammar *forced* `"leaf"` (only legal key), but Ollama reports the chosen token's raw
prob (~0) and shows the model actually wanted `"category"` (1.0). Consequences:

- You cannot read "P(model over the legal paths)" directly — the numbers aren't renormalized
  over the legal set.
- Legal tokens are frequently **outside top-k**, so you can't even reconstruct the
  renormalized distribution from the truncated `top_logprobs`.

→ The clean "grammar renormalizes, token-dist == taxonomy-dist" story is **false on this
stack**. Any logprob-confidence approach must work *with* the raw distribution.

## Finding 2 — the TYPE-FIRST surface form fights tokenization (accuracy + legibility)

Enum = full `Type/Category/Subcategory` strings, so the grammar makes the model commit to
**Type first** — the most abstract, least-informative decision — with the discriminating
**leaf last**. Probe: `item: Uber Centro` → chose `Extras/Lazer/Cinema` (WRONG):

```
tok 'E'    p=0.000   top=[item=0.496, ..., Uber=0.089]   # wanted to say "Uber"
tok 'La'   p=0.000   top=[Uber=0.993, ...]               # STILL wants "Uber"
tok 'Ci'   p=0.000   top=[Uber=0.956, ...]               # STILL wants "Uber"
```

The model's semantic signal ("this is Uber") is loud and consistent, but the grammar forces a
Type guess before the leaf is reachable — it picks the wrong subtree and locks in. Chosen
tokens have raw p≈0 (the model is being dragged onto a rail it didn't want) → the margin is
**unreadable**.

## Finding 3 — LEAF-FIRST aligns the surface form; the signal becomes legible

Enum = the 104 unique bare leaf names (5 cross-type collisions collapse; disambiguate upward
via `taxonomy.ResolveLeaf`, exactly as `add` already does). Single `leaf` field. Same expense
→ chose `Uber/Taxi` (RIGHT):

```
tok 'Uber'  p=0.986   top=[Uber=0.986, Cent=0.007, Leaf=0.003]   # the leaf decision, CLEAN
tok '/'     p=0.000   ...   # grammar completing the unique "Uber/Taxi" leaf — NOT a decision
tok 'Tax'   p=0.000   ...   # (same)
```

Because `Uber` is now a legal leaf token the model *wants* to emit, the decision token is the
model's own high-probability token with readable competitors. The real decision concentrates
at ONE branch point (tokens after it just complete the unique leaf). **Mechanism confirmed:
leaf-first makes the pre-mask logprobs usable.**

Leaf-first also mirrors the already-proven manual path: `add` takes a human leaf and derives
type/category upward (`resolveFullPath`→`ResolveLeaf`, add.go:87,206). The MODEL path is the
outlier doing it top-down.

## Finding 4 — a single token is NOT the leaf margin (methodological)

Reading one token overstates confidence for multi-token leaves. The first character measures
"which initial letter," not "which leaf":

```
Aula de paraquedismo -> Apoia-se 4i20   'A' p=0.968   # confident it starts with A; many leaves do
Ferradura de cavalo  -> Ferramentas     'F' p=0.877   # same first-letter artifact
Drone DJI Mavic      -> Diversos         'D' p=0.563 (Dr=0.402)  # leaf short => token ~= decision
Netflix assinatura   -> Netflix         'Netflix' p=0.646        # 1-token leaf; rivals are numbering frags
```

The correct signal is the **sequence probability of the whole leaf** (product of its token
probs) and the **margin to the next-most-probable complete leaf**, not any single token.

Encouraging (but n=1) calibration anecdote: `Drone DJI Mavic` — a genuinely novel item — showed
real first-token uncertainty (0.56) and routed to the catch-all `Diversos`. That is the "I don't
know" behavior a gate wants. One data point, not a result.

## What's proven vs. not

- **PROVEN:** logprobs are exposed under the grammar; leaf-first makes the decision token
  legible; type-first destroys legibility; Ollama reports pre-mask logprobs.
- **NOT proven:** that the aggregated leaf margin is *calibrated* (separates correct from
  wrong). The single-token probes can't show this. Needs the T-14 labeled set.

## Side flags

- **Junk taxonomy leaf:** `Apoia-se 4i20` looks like a leaked expense description masquerading
  as a taxonomy leaf. Audit `config/taxonomy.json`.
- All probes were `think:false`. Think-on prepends reasoning tokens that change the stream
  shape (the JSON — and the decision token — arrive after the `<think>` block); aggregation
  must locate the JSON span first.

## Implications for T-23

1. **Leaf-first classification** is a low-risk architectural win independent of calibration:
   attacks the "right sheet, wrong leaf" error class and is the precondition for any legible
   confidence signal. Enum = 104 leaves; derive type/category via `ResolveLeaf`.
2. **Confidence signal — two routes** (the real T-23 experiment, "option c"):
   - **Logprob margin**: sequence logprob of the chosen leaf + margin to runner-up. Fiddly
     given multi-token leaves + pre-mask reporting; may need a prefix-tree/beam reconstruction.
   - **Ensemble agreement**: sample K times at temperature > 0, measure vote concentration
     over *complete* leaves. Immune to the tokenization/aggregation problem; trivially
     interpretable. Possibly the more robust signal.
   - **Unconstrained-first** (couples to T-28): classify free-form, read that generation's
     clean logprobs, map to taxonomy, grammar-constrain only on off-tree fallback.
3. **Benchmark before adopting:** over the T-14 sample, test whether margin and/or agreement
   separates correct from wrong (AUC; does a threshold recover the precision 0.85-confidence
   couldn't?). This is the gate-decision experiment.

## Reproduce

Probes: `/tmp/.../scratchpad/{lp_test,leaf_test,probe}.json` (throwaway). To rebuild the
leaf-first enum: extract `types[].cats[].subs[].name` from `config/taxonomy.json`, dedup.
