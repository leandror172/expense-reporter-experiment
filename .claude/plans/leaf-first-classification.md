# Leaf-First Classification — Plan

**Status:** proposed (session 49). Standalone accuracy change + precondition for T-23.
Companion docs: `.claude/t23-logprob-confidence-probe.md` (the probe that surfaced this),
`.claude/plans/t23-calibration-benchmark.md` (the confidence work that rides on top).
[[project_logprob_confidence_leaf_first]].

---

## Intent — why this thread exists (read this first)

This did NOT start as an accuracy task. It started as **T-23** (fix the auto-insert gate,
the WS-D blocker) because T-14/T-26 proved q3's self-reported `confidence` is uninformative
(~86–91% of WRONG answers carry ≥0.85). During that discussion the user raised a sharp idea:
a model already produces a probability distribution over next-token candidates, and when the
output is grammar-constrained to the taxonomy enum, those candidates *are* the label set — so
maybe a real confidence can be read off the token distribution instead of the self-reported
scalar.

Probing that (session 49) produced two surprises:
1. **Ollama 0.17.5 reports PRE-grammar-mask logprobs** — the raw distribution, not renormalized
   over legal tokens — so you cannot directly read "P over legal paths." (Overturned the initial
   theory.)
2. **The current full-path enum fights the model's tokenization.** Enum members are
   `Type/Category/Subcategory` strings, so the grammar forces the model to commit to **Type
   first** — the most abstract decision — with the discriminating **leaf last**. Probe: for
   `Uber Centro` the model *wanted* to emit "Uber" (raw 0.95+ at multiple positions) but was
   dragged onto a Type rail, guessed the wrong subtree (`Extras`), and locked in. Chosen tokens
   had raw p≈0.

Reversing to **leaf-first** (model picks the leaf; derive type/category upward) fixed BOTH: the
decision token became the model's own high-probability token (`Uber` leaf p=0.986, competitors
readable), i.e. the confidence signal became legible. But in doing so it revealed that leaf-first
is **also a plausible accuracy win on its own**, independent of the confidence idea — and that
the manual `add` path *already works this way*. So we split the threads:

- **This plan** = leaf-first as a standalone accuracy change (lower-risk, possibly a real gain,
  and the precondition for legible confidence). Do this first.
- **T-23 calibration benchmark** = the confidence-signal experiment, which rides on a leaf-first
  base once it lands.

**The intent to preserve:** leaf-first is worth doing even if the confidence idea never pans out,
because it attacks T-14's *dominant* error class directly. Don't relitigate that framing; the
open question is execution + measurement, not whether it's worth exploring.

---

## Background / what we know

- **Current (T-13):** classifier predicts a full path via an atomic 112-path enum
  (`taxonomy.PathEnum`), left-to-right Type→Category→Subcategory. Grammar-enforced valid.
- **T-14:** 63% full-path / 77% type. The **dominant error class is "right sheet, wrong leaf"**
  — the model gets the type/category region right but the specific leaf wrong. T-22 type-level
  descriptions lifted TYPE accuracy (+6 pp) but by construction left leaf disambiguation
  untouched (→ that's why T-27 leaf-level descriptions exists).
- **`add` already does leaf-first:** a human supplies the leaf, and `resolveFullPath` →
  `taxonomy.ResolveLeaf(sheets, subcategory, typeFlag)` (add.go:87,206) derives type/category by
  lookup. The MODEL path (`auto`/`batch-auto`/`classify`) is the outlier doing it top-down.
- **Probe finding:** type-first is a poor match for how the model wants to emit tokens; the
  semantic signal lives at the leaf but the grammar demands it last.

## Hypothesis

Making the model predict the **leaf** first and deriving type/category by lookup will:
(a) raise full-path accuracy by eliminating "wrong-type-then-locked-in" errors, and
(b) spend the model's highest-attention first tokens on the decision it is empirically best at
(the leaf carries the semantic signal — cf. "Uber"), and
(c) make type/category *deterministic* (a lookup), so they can never disagree with the leaf.

This is falsifiable: the A/B below could show no gain (or a regression), in which case leaf-first
survives only as the confidence-signal enabler, judged on T-23's terms instead.

## Design

- **Enum** = the **104 unique leaf names** (112 leaves − 8 duplicate instances across 5 collision
  names). Single `leaf` field in the structured-output schema (replaces the `path` enum).
- **Derive** `(type, category)` from the chosen leaf via `taxonomy.ResolveLeaf`. Reconstruct the
  full path for logging/routing exactly as today — nothing downstream of `classifier.Result`
  changes (it still carries `Type/Category/Subcategory`).
- **Cross-type collisions** — 5 leaf names are ambiguous by bare name: `Estacionamento`, `Orion`,
  `Lilly`, `Ambos`, `Dentista` (verify exact multiplicity; 112→104 implies 8 instances). Bare
  leaf-first cannot resolve these. **Open decision (D1)** — options:
  - (a) **Hybrid enum**: keep the 5 as full `Type/…/leaf` strings, rest as bare leaves. Minimal,
    preserves resolvability, but reintroduces the surface-form problem for exactly those 5.
  - (b) **Second field**: model also emits a `type` when the leaf is ambiguous (small enum of the
    2–3 owning types). Clean, one extra token only when needed.
  - (c) **Route ambiguous to review** — never auto-insert a collision leaf. Safe, lossy.
  - (d) **Amount/context heuristic** post-hoc. Fragile.
  - Lean: (b) — mirrors how `add` uses `--type`, and keeps the common 99 leaves clean.
- **Sentinel (T-19)** — still append `SentinelPath` ("NENHUMA DAS OPÇÕES") → Diversos@0.30. The
  decline path is orthogonal to leaf-vs-path.
- **Prompt** — still render the type→category→leaf **tree** for context (the model needs to see
  which leaves exist and their grouping), even though the enum is flat leaves. **Open decision
  (D2):** tree vs flat leaf list in the prompt — measure both if cheap.
- **Confidence** — unchanged for this change (model still emits self-reported `confidence`). The
  gate rethink stays in T-23; this plan does NOT touch the gate.
- **Where it lives — Open decision (D3):** benchmark-only script first (build the leaf-first enum
  from `config/taxonomy.json`, call Ollama directly, no production change) vs wire into
  `classifier.Classify` behind a flag. Lean: benchmark-only for the A/B, productionize only if it
  wins — same discipline as T-22's sidecar A/B.

## Prerequisite — taxonomy hygiene audit (also its own task)

`config/taxonomy.json` contains at least one junk leaf: **`Apoia-se 4i20`** looks like a leaked
expense *description* that got captured as a taxonomy leaf (probably from the 5.R4 historical
extraction). Junk leaves hurt BOTH the current classifier (they're valid enum members the model
can be forced into) and leaf-first (they pollute the 104-name enum and can absorb OOD items).
**Audit:** walk all 112 leaves, flag any that are descriptions/typos/duplic*semantic* entries,
and decide fix vs keep with the user (taxonomy is user-authoritative — exported from the
workbook per T-02, so a fix must survive regeneration: fix at the source/export or via a
sidecar, not by hand-editing the generated file). Do this BEFORE the A/B so garbage doesn't
distort the enum or the accuracy numbers.

## The A/B experiment

- Reuse the T-14 harness + stratified sample (300 labeled items, `.claude/scratch/t14-benchmark/`).
- **Arms:** current full-path enum (control) vs leaf-first enum (treatment). If D1=(b), include
  the ambiguous-type prediction.
- **Metrics:** full-path accuracy, TYPE accuracy, LEAF accuracy, and specifically the
  **"right sheet, wrong leaf" rate** (does leaf-first move it?). Also record whether type/category
  derivation ever fails (should be never except the 5 collisions).
- **Stratify:** leaky vs clean (the win must show on **clean/novel** items to matter — leaky are
  already easy), think-on vs no-think.
- **Significance:** bootstrap CIs; n≈300 resolves ~±5 pp, and subgroup splits are noisy (T-26
  lesson) — don't over-read a single stratum.

## Relationship to other work (don't lose these couplings)

- **Precondition for T-23** (calibration benchmark) — leaf-first is what makes the logprob signal
  legible; the benchmark's non-baseline signals assume it.
- **T-27 (leaf-level descriptions)** — composes: leaf-first + per-leaf blurbs both attack the
  wrong-leaf class. Sequence T-27 after leaf-first so descriptions land on the enum that actually
  uses them.
- **T-28 (alt classification strategies)** — leaf-first IS one of T-28's directions (inverted
  two-stage: leaf then derive up). This plan makes that concrete; T-28's *other* idea (two-pass
  unconstrained-then-grammar) stays separate and also couples to the T-23 "unconstrained-first"
  confidence route.
- **T-22 (merged)** — independent; type descriptions still render in the tree context.
- **T-24 (`--think` default)** — the A/B's no-think arm feeds this.

## Risks / open questions

- **D1 collisions** — the load-bearing design decision. Getting it wrong either reintroduces the
  surface-form problem (a) or silently drops accuracy on 5 leaves (c).
- **Junk leaves** — mitigated by the prerequisite audit.
- **ResolveLeaf uniqueness** — verify every non-collision leaf resolves to exactly one
  (type, category); if any others are ambiguous, D1 must cover them too.
- **Downstream** — `classifier.Result` still carries the full path, so logs/routing/appender are
  unchanged; verify no code path reads the raw `path` enum string directly.
- **think-on stream** — reasoning tokens precede the JSON; the benchmark harness must locate the
  JSON span (same handling the T-23 plan notes).
- **Negative result is a valid outcome** — if the A/B shows no accuracy gain, leaf-first is judged
  purely as the T-23 enabler, not adopted for accuracy on its own.

## Decision log (OPEN — resolve with user before building)

- **D1** — cross-type collision handling: hybrid enum / second type field / route-to-review /
  heuristic. (Lean: second type field.)
- **D2** — prompt shows the tree vs a flat leaf list.
- **D3** — benchmark-only script vs production flag for the A/B. (Lean: benchmark-only first.)
- **D4** — adopt-for-accuracy threshold: how big a clean-item gain justifies productionizing.

## Deliverables

- This plan; the taxonomy-audit outcome; the A/B harness (under `.claude/scratch/`); a report
  section (append to `.claude/t14-benchmark-report.md`); an adopt/hold decision.
