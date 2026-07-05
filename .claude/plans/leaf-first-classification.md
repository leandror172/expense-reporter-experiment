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
- **Cross-type collisions** — 5 leaf names are ambiguous by bare name (audit-confirmed
  multiplicity, 13 instances → 112−8=104 unique): `Ambos`/`Lilly`/`Orion` (Pets, ×3 each across
  Fixas/Variáveis/Extras — pet names), `Dentista` (Saúde ×2), `Estacionamento` (Transporte ×2).
  Bare leaf-first cannot resolve these. **DECISION (D1) — LOCKED = (b), session 49.**
  - **(b) Second `type` field.** Schema emits **`leaf` (104-enum) THEN `type` (3–4 types)** — field
    order is load-bearing: leaf first preserves the leaf-first commitment (the model's strong
    signal), type second is a genuine *recurrence* judgment (the audit showed the collision axis
    IS Fixas/Variáveis/Extras — not derivable from the leaf name; "Lilly" can't say monthly-vet
    vs one-off-toy). Same recurrence axis T-23 flags as a better safety signal than confidence.
  - **Always predict `type`** (a static grammar can't make a field conditionally required on the
    leaf value); `ResolveLeaf` uses it ONLY for the 5 collisions and ignores it for the 99 unique
    leaves. For unique leaves, **cross-check** model-type vs the leaf's real type and log
    disagreements as a free extra calibration signal (don't gate on it yet).
  - Rejected: (a) hybrid enum reintroduces the surface-form problem for the 5; (c) route-to-review
    is lossy (pets span all 3 types, likely frequent); (d) amount heuristic is fragile.
  - Mirrors how the manual `add` path already uses `--type`.
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

**DONE (session 49).** Audit of all 112 leaves flagged exactly one junk leaf: `Apoia-se 4i20`
(a campaign-specific description captured as a leaf during 5.R4 extraction — confirmed: it
appeared identically as both `item` and `subcategory` in training data). **Resolved:**
generalized to the vendor-level leaf **`Apoia-se`** (user chose vendor granularity over an
`Assinaturas` bucket, to stay consistent with future Netflix/HBO/Spotify generalizations).
**Full propagation applied** (surgical — label positions only, real `item` text preserved):
`config/taxonomy.json` (1), `feature_dictionary_enhanced.json` (4), `training_data_complete.json`
(12 subcategory, 12 items kept), `expenses_log.jsonl` + `expenses_log-allyears.jsonl` (12 each).
Backups: `*.bak-apoia-rename`. All JSON re-validated; 112 leaves (count unchanged).
**REMAINING durability step:** the taxonomy is exported from the workbook (T-02), so the local
`config/taxonomy.json` edit is wiped on re-export — the leaf must also be renamed in the workbook
Referência sheet (or the export step) to survive regeneration. Frozen T-14 benchmark artifacts
(`.claude/scratch/t14-benchmark/`) were intentionally NOT touched (they record what was run).

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

- **D1** — LOCKED (session 49) = second `type` field, schema order `leaf` then `type`,
  always-predict, ResolveLeaf disambiguates the 5 collisions + cross-checks the 99 as a free
  signal. See Design § collisions.
- **D2** — prompt shows the tree vs a flat leaf list.
- **D3** — benchmark-only script vs production flag for the A/B. (Lean: benchmark-only first.)
- **D4** — adopt-for-accuracy threshold: how big a clean-item gain justifies productionizing.

## Deliverables

- This plan; the taxonomy-audit outcome; the A/B harness (under `.claude/scratch/`); a report
  section (append to `.claude/t14-benchmark-report.md`); an adopt/hold decision.

---

## Execution log (session 50, 2026-07-04)

- **GBNF property-order assumption — EMPIRICALLY CONFIRMED (highest-risk item, checked FIRST).**
  A two-field schema declaring properties `leaf → type → confidence` makes q3 *generate* `leaf`
  before `type` in the raw stream (3/3 probes: Uber→`Uber/Taxi`, Netflix→`Netflix`,
  dentista→`Dentista`, all `{"leaf":…,"type":…}`). Schema declaration order → generation order is
  pinned by the schema→GBNF compile. **D1's two-field mechanism is sound — no fallback to a
  concatenated `leaf|type` single-string enum needed.** The `Dentista` probe also showed the
  collision case working: `type:"Variáveis"` came back as a separate field for `ResolveLeaf` to
  consume. Probe: `scratchpad/gbnf_order_probe.py`.
- **Label fix (carried from review):** the **few-shot axis is SEPARATE from D3.** D3 = where the
  arm lives (benchmark-only vs prod flag) — locked benchmark-only. The few-shot on/off choice is
  its own axis: **FS-A** (few-shot OFF, both arms — the signal check) vs **FS-B** (few-shot ON,
  arm-shaped assistant messages: leaf+type JSON for treatment vs path string for control — the
  production-realistic verdict).
- **D4 is gated on FS-B, not FS-A.** FS-A answers only "does the enum direction move accuracy,
  all else equal." Production always runs few-shot ON, and few-shot may wash out or interact with
  the enum effect, so a promising FS-A delta ALONE must not trigger "adopt." FS-A = signal; FS-B =
  verdict.
- **Harness built (D3=benchmark-only, D2=tree):** `.claude/scratch/leaf-first-ab/`
  (`run_leaf_ab.py` two arms + direct Ollama calls; `analyze_leaf_ab.py` adds the
  right-sheet-wrong-leaf rate + paired McNemar + the `type_crosscheck` free calibration signal;
  `sample.jsonl`/`ood.jsonl` symlinked from `../t14-benchmark`; `score.py`/`sigtest_t26.py`
  reused unchanged — the runner emits their exact record shape). Both arms few-shot OFF, same tree
  prompt, NO T-22 descriptions (held constant/absent).

### FS-A result (no-think, few-shot OFF, enum-isolated, n=300 paired) — LARGE + SIGNIFICANT

| metric | control (path-first) | treat (leaf-first) | Δ | McNemar p |
|---|---|---|---|---|
| full-path | 39.7% | 52.3% | **+12.7** | <0.001 |
| type | 60.7% | 71.7% | +11.0 | <0.001 |
| leaf | 41.3% | 53.7% | +12.3 | <0.001 |

- **Holds on the load-bearing CLEAN/novel stratum:** full-path +12.9pp (20.7→33.6%), p=0.001
  (17 treatment-wins vs 2 control-wins). Leaky +12.5pp, p<0.001. Decisively clears noise (unlike
  the T-26 within-noise deltas).
- **Mechanism correction (honest):** the "right sheet, wrong leaf" *rate* barely moved
  (−1.7pp, not sig). The gain is NOT from fixing wrong-leaf-within-right-sheet; it's that the
  **type decision itself improved** because leaf-first *derives* type from a confident leaf
  (`ResolveLeaf`) instead of guessing it first — path-first type 60.7% (predicted first) vs
  leaf-first 71.7% (derived). The probe's core insight, confirmed at scale.
- **`type_crosscheck` is a real free calibration signal** (feeds T-23): of 256 unique-leaf
  predictions, 42 had the model's field-2 `type` guess disagree with the leaf's owner →
  full-path acc **60.8% (agree) vs 40.5% (disagree)**, a 20pp spread self-confidence never gave.
- **Binding caveat (why FS-A ≠ adoption):** control few-shot-OFF (39.7%) sits far below production
  path-first few-shot-ON (~58.7%) — **few-shot is worth more than the enum direction**. Leaf-first
  few-shot-OFF (52.3%) is still *below* path-first few-shot-ON. FS-A proves the enum gain is real
  and large in isolation; it can't say whether it SURVIVES with few-shot on. → FS-B decides D4.

### Think-on FS-A result (few-shot OFF, n=300 paired) — enum gain HOLDS across think mode

| metric | control | treat | Δ | McNemar p | (no-think Δ) |
|---|---|---|---|---|---|
| full-path | 46.7% | 55.3% | +8.7 | <0.001 | +12.7 |
| type | 63.3% | 75.0% | +11.7 | <0.001 | +11.0 |
| leaf | 48.3% | 57.0% | +8.7 | <0.001 | +12.3 |

- **Aggregate + type gains are robust and significant in BOTH think modes, direction-consistent**
  (the real evidence, per the T-26 cross-run-consistency lesson — not any single p-value).
- **Two honest corrections to the no-think writeup:**
  1. **Clean/novel win is weaker than first stated.** Think-on clean full-path +6.0pp is
     **p=0.143 (not sig)**; only clean *type* clears (+12.1pp, p=0.007). Clean full-path is
     positive in both modes (+12.9 no-think / +6.0 think-on — no sign flip) but individually
     significant only in no-think. Robust effects = aggregate + type.
  2. **`type_crosscheck` calibration signal WALKED BACK — does not replicate.** No-think:
     disagree→worse (60.8 vs 40.5). Think-on: disagree→*better* (60.1 vs 77.8, n=18). Sign flip
     across modes on tiny n = noise, not a mechanism (T-26 rule). NOT an established signal — a
     hypothesis for T-23 to test properly, not a finding. (Removes the earlier "free calibration
     signal" claim.)

### FS-B block (the D4 adoption gate) — few-shot ON, arm-shaped examples

FS-B setup: `--fewshot` injects arm-shaped example pairs — `{"leaf","type"}` JSON for treatment vs
`{"path"}` string for control; same `SelectExamples` port feeds both arms (training pool only, for
reproducibility); `fs-` filename segment. User authorized the whole block.

**FS-B no-think result (few-shot ON, n=300 paired) — enum full-path gain largely WASHES OUT:**

| metric | control (path+fs) | treat (leaf+fs) | Δ | McNemar p | (FS-A Δ) |
|---|---|---|---|---|---|
| full-path | 59.3% | 62.7% | +3.3 | 0.064 (NOT sig) | +12.7 |
| type | 75.7% | 81.3% | +5.7 | 0.008 (sig) | +11.0 |
| leaf | 60.3% | 63.7% | +3.3 | 0.064 (NOT sig) | +12.3 |

- **The caveat held:** few-shot captures most of the leaf-first full-path benefit. Few-shot lifted
  the control +19.6pp (39.7→59.3) but the treatment only +10.4pp (52.3→62.7) — the two mechanisms
  (retrieval vs generation-order) fix overlapping errors, so stacking is sub-additive and the
  marginal full-path gain drops to +3.3pp / not significant.
- **TYPE gain survives** (+5.7pp, p=0.008) — type is a *derived* lookup off the leaf, a structural
  gain few-shot can't replicate by example. Clean-subset type also sig (+8.6pp, p=0.041); clean
  full-path +6.0pp borderline (p=0.065).
- **Harness validated:** path-first+few-shot = 59.3% matches the production T-14/T-22 no-think
  baseline (~58.7%) → the few-shot port faithfully reproduces production.
- `type_crosscheck`: disagree→worse again here (69.8 vs 34.8, n=23) — but it flipped in FS-A
  think-on, so still not established (awaiting FS-B think-on as the 4th datapoint).

**FS-B think-on result (few-shot ON + think-on = PRODUCTION default, n=300 paired) — DECISIVE:**

| metric | control (path+fs) | treat (leaf+fs) | Δ | McNemar p |
|---|---|---|---|---|
| full-path | 59.3% | 61.0% | +1.7 | 0.458 (NOT sig) |
| type | 77.0% | 79.3% | +2.3 | 0.296 (NOT sig) |
| leaf | 61.0% | 62.0% | +1.0 | 0.701 (NOT sig) |

**In the production configuration, leaf-first has NO significant advantage on any metric.**

### D4 DECISION — HOLD (do not adopt leaf-first for accuracy). Session 50.

Full-path Δ decays monotonically across the 4 cells: +12.7 (FS-A nothink) → +8.7 (FS-A think) →
+3.3 (FS-B nothink) → **+1.7 (FS-B think = prod, not sig)**. Type Δ: +11.0 → +11.7 → +5.7 → +2.3
(sig in 3/4, not in prod). Every mechanism that lets the path-first control catch up (few-shot
retrieval, then thinking) erodes the leaf-first edge; the two mechanisms fix overlapping errors, so
stacking is sub-additive.

- **HOLD for accuracy** — the plan admitted a negative result was valid; this is a clean one.
- **Point estimate too small to justify a change (NOT a proven null)** — prod-cell full-path +1.7pp
  on ~29 discordant pairs is underpowered; n=300 can't resolve a true +2–3pp effect. Leaf-first is
  not measurably worse, but "costs nothing" would overclaim. Conservative read: a ≤+1.7pp effect
  isn't worth a production rewrite.
- **Still the T-23 precondition — SCOPED (advisor sharpening).** Proven only for the
  **logprob-margin / answer-perplexity** route (2a/2b; type-first drags chosen tokens to raw p≈0).
  For the **ensemble-agreement** route (signal 3), leaf-first is **untested/possibly unnecessary**
  (leaves can be extracted from K sampled full-paths without a leaf-first enum). So: precondition
  for the logprob route; OPEN for ensemble. If T-23 goes ensemble → leaf-first fully on the shelf.
- **T-22 descriptions held absent** in both arms (isolates the enum); prod has them. Strengthens
  HOLD — leaf-first got its best shot (no competing type booster) and its type edge still washed out.
- **`type_crosscheck` NOT a usable signal** — disagree→worse in 3/4 runs, inverted in FS-A think,
  and disagreements collapse toward 0 as accuracy rises (n=42→18→23→4). A T-23 footnote, not a gate.
- **Couples to T-27** — few-shot already recovers most "wrong-leaf" errors in production, undercutting
  T-27's (leaf-level descriptions) "right sheet, wrong leaf" motivation. Re-justify T-27 on few-shot-ON
  evidence or drop it.
- Report section: `.claude/t14-benchmark-report.md` "T-30 Leaf-First A/B".
