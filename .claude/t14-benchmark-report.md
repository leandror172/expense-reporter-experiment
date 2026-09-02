# T-14 Benchmark Report — Classifier Accuracy, Speed, Calibration (session 46, 2026-07-02)

**Question:** Is the classifier accurate enough to retire the bare-name fallback (WS-D),
and which model/config should be the default?

**Answer in one line:** q3 is ~63% full-path accurate but its confidence is
uninformative (91% of errors at ≥0.85), so the 0.85 auto-insert gate does NOT protect
against wrong inserts — WS-D should not lean harder on the model path until calibration
or exclusion rules compensate. `--think=false` gives 10× speed for −3.3 pp accuracy.

## Setup

- **Sample:** 300 items, stratified over the (category, subcategory) leaves of the
  1713 corpus entries whose `(sheet, category, subcategory)` matches the 112-path
  taxonomy exactly (74 user_corrections + 7 label-drift entries excluded). 80 of 112
  leaves have labeled history; 32 leaves are unrepresented.
- **Ground-truth type** derived from the corpus `source` sheet name.
- **Leakage flag:** items whose normalized description occurs 2+ times in the corpus
  (the few-shot pool) — 184/300. The few-shot pool also merges live
  `classifications.jsonl` feedback, so results are not reproducible from the corpus alone.
- **OOD set:** 20 hand-written out-of-domain probes (`ood.jsonl`) to measure the T-19
  sentinel-decline rate.
- **Harness:** `.claude/scratch/t14-benchmark/{build_sample,run_benchmark,score}.py`,
  driving the real binary (`classify --json --top 3`), resumable JSONL results.

## Results (q3 = my-classifier-q3, qwen3:8b)

| condition | full-path | cat+sub | type | leaky | clean | mean lat | p90 |
|---|---|---|---|---|---|---|---|
| q3 think-on | 63.0% | 63.7% | 77.0% | 71.7% | 49.1% | 14.4 s | 29.4 s |
| q3 no-think (`--think=false`) | 59.7% | 60.3% | 77.0% | 67.4% | 47.4% | **1.5 s** | 2.1 s |

- **Calibration (both conditions):** mean confidence 0.95 on correct vs 0.90–0.91 on
  wrong; **~91% of wrong answers sit at ≥0.85** → the auto-insert threshold passes
  nearly every error. This is the headline finding.
- **Leakage split:** recurring items (few-shot twins retrievable) 67–72%; novel items
  **~48–49% — a coin flip**. Production skews recurring, so effective accuracy is
  nearer the leaky number; any new merchant is 50/50.
- **OOD / T-19 sentinel:** think-on declines 2/20 (10%) to `Diversos`@0.30; no-think
  declines **0/20** and produces the most absurd confident picks (detective →
  `Supermercado VA`@0.90, taxidermy → `Extras/Saúde/Hospital`@0.90). T-19 remains
  structurally present but behaviorally near-dead; out-of-domain safety must come from
  review flow / exclusion rules, not confidence.
- Errors lose the whole leaf, not the type prefix: full-path ≈ cat+sub ≈ sub accuracy,
  type accuracy 14 pp higher ("right sheet, wrong leaf" is the dominant error class).

## Model matrix outcome

- **q35 (qwen3.5:9b) is disqualified as default** on Ollama 0.17.5:
  - Thinking-on: unbounded thinking runs (70–240 s/item, timeouts) — a full run ≈ 8 h.
  - `think:false`: **the `format` grammar is silently dropped** (verified at API level:
    persona-schema JSON + out-of-enum labels). Structured output broken → unusable.
  - The `/no_think` prompt soft switch is ignored by qwen3.5 (qwen3-era feature).
  - Upstream bug candidate: qwen3.5 renderer + `think:false` + `format` on Ollama 0.17.5.
- **q3 honors `think:false` AND keeps the grammar** — the no-think fast lane is q3-only.
- **qcoder subset not run** (q35 investigation consumed the slot; qcoder still doesn't
  fit VRAM and nothing here suggests a 30B fixes calibration).

## Operational notes

- A client-side timeout does NOT stop Ollama's server-side generation; abandoned
  requests queue-starve everything behind them (GPU pegged, all calls hang). Keep the
  runner timeout above worst case, or restart Ollama to clear orphans.
- `my-classifier-q35-nothink` persona (LLM repo registry) was created on the false
  premise that `/no_think` works — it is inert; delete or repurpose.

## Implications

1. **WS-D gate: NOT passed.** Retiring the bare-name fallback would put ~37% wrong
   full paths behind a threshold that stops ~9% of them. Blockers: calibration or a
   stronger exclusion/review net.
2. **`--think` default:** left at `true` (accuracy + residual sentinel). The 10× speed
   is one flag away for batch runs; revisit after T-22.
3. **T-22 (taxonomy descriptions A/B)** is now cheap (~8 min/condition at no-think
   speed) — next session. Measure: full-path accuracy, clean-subset accuracy, and
   OOD decline rate, same sample.

---

# T-22 Addendum — Type-Description A/B (session 47, 2026-07-03)

**Question:** Does adding a short, type-level description (Fixas/Variáveis/Extras/
Adicionais) to the classifier prompt improve accuracy — and in which language?

**Change under test:** optional `Description` on `taxonomy.ExpenseType`, rendered
parenthetically on the type header line of the system prompt (`writeTaxonomyTree`).
Strictly additive — no enum/grammar/routing change; the GBNF path grammar is untouched,
so the 112 valid paths and their validity are unaffected. Three conditions differ ONLY by
the four description lines: `nodesc` (no `description` keys → byte-identical prompt to the
pre-change baseline), `descen` (English), `descpt` (Portuguese).

## Results — no-think, 300 items, q3 (same T-14 sample)

| condition | full-path | **type** | leaky/recurring (184) | clean/novel (116) | mean lat | HC-wrong |
|---|---|---|---|---|---|---|
| nodesc | 58.7% | 74.3% | 66.3% | 46.6% | 1.5 s | 91% |
| **descen** (English) | **61.3%** (+2.6) | **80.3%** (+6.0) | **72.3%** (+6.0) | 44.0% (−2.6) | 1.4 s | 94% |
| descpt (Portuguese) | 59.7% (+1.0) | 79.0% (+4.7) | 70.1% (+3.8) | 43.1% (−3.5) | 1.4 s | 95% |

- **Headline signal is TYPE accuracy: +6 pp (English), and BOTH languages lift it ~5–6 pp.**
  That consistency across two independently-authored description sets is strong evidence
  it is a real effect, not q3 run-to-run noise. Mechanism matches T-14's dominant error
  class ("right sheet, wrong leaf"): type-level text helps pick the right *type*; leaf
  disambiguation is untouched, so full-path lifts less (+2.6 pp). Argues for a future
  leaf-level description extension.
- **English > Portuguese** on every metric → English adopted; Portuguese dropped.
- **Small cost on novel items** (−2.6 pp): descriptions help recurring items (which
  dominate production) but slightly mislead genuinely-new merchants. Net positive.
- **Calibration UNCHANGED** — HC-wrong still ~91–95% everywhere. Descriptions are NOT a
  fix for the auto-insert gate; that remains T-23.
- **OOD: 0/20 decline in all three conditions** — expected, since no-think kills the T-19
  sentinel regardless. Descriptions did shift *where* OOD items land (English pulls
  oddities toward `Extras` = "irregular/unexpected", less absurd than baseline's
  `Detetive→Supermercado`), but produced no actual declines. Whether descriptions restore
  sentinel declines only shows up in think-on — deferred.
- The `nodesc` re-run (58.7%) matches T-14's prior no-think baseline (59.7%) within noise
  → harness validated. Result files: `.claude/scratch/t14-benchmark/results-{nodesc,descen,descpt}-my-classifier-q3.jsonl`.

## Decision
- **Adopted English type descriptions** via a tracked, non-sensitive sidecar
  `config/type-descriptions.json`, merged into the loaded taxonomy at classify-time
  (`taxonomy.LoadTypeDescriptions`/`ApplyDescriptions`, applied in the classifier-scoped
  `loadTaxonomyTree`). The gitignored `config/taxonomy.json` is left untouched — the
  sidecar is the durable, version-controlled authoring home, surviving any future taxonomy
  regeneration. `generate-workbook` never renders descriptions, so it is unaffected.
- **Deferred:** the think-on confirmation run (descen vs the existing T-14 think-on
  baseline, ~72 min) — to confirm the type-accuracy gain holds in the production-default
  mode AND to measure OOD sentinel decline *with* descriptions (the mode where the
  sentinel is alive). Couples to T-24 (`--think` default) and T-23 (gate rethink).
  **Runner caveat:** `run_t22.sh` swaps `taxonomy.json` (pre-adoption mechanism, used only
  for the no-think A/B above). Post-adoption the think-on run must toggle the **sidecar**
  instead — `descen` = `config/type-descriptions.json` present, `nodesc` = sidecar
  removed/renamed. Rework the runner before that run (a warning is in its header).

## Results — think-on (T-26 confirmation run, 2026-07-03)

Runner: `.claude/scratch/t14-benchmark/run_t26.sh think` (sidecar-toggle mechanism;
`nodesc` = `config/type-descriptions.json` removed, `descen` = present). 300-item accuracy
+ 20-item OOD per condition, q3, think-on default. Result files:
`.claude/scratch/t14-benchmark/results-think-{nodesc,descen}-my-classifier-q3.jsonl`
(+ `ood-results-think-*`). Portuguese dropped after the no-think A/B.

| condition | full-path | **type** | leaky/recurring (184) full | clean/novel (116) full | novel **type** | OOD sentinel | mean lat | HC-wrong |
|---|---|---|---|---|---|---|---|---|
| think-nodesc | 64.7% | 79.0% | 76.6% | 45.7% | 70.7% | **4/20 (20%)** | 15.1 s | 86% |
| **think-descen** (English) | 64.0% (−0.7) | **81.3%** (+2.3) | 72.8% (−3.8) | **50.0%** (+4.3) | **79.3%** (+8.6) | **0/20 (0%)** | 12.8 s | 88% |

**⚠ Significance-tested (paired McNemar / Fisher exact, `sigtest_t26.py`, zero model
calls). This is a SINGLE run; almost every delta below is within noise. Treat the
think-on numbers as suggestive, not established — the cross-run *consistency* is the only
real evidence, no single-run delta here clears p<0.05.**

Paired McNemar (nodesc vs descen, same 300 items):

| metric | Δ | McNemar p | verdict |
|---|---|---|---|
| full-path, all | −0.7 pp | 0.86 | noise |
| full-path, novel (n=116) | +4.3 pp | 0.30 | not sig |
| full-path, recurring (n=184) | −3.8 pp | 0.17 | not sig |
| type, all | +2.3 pp | 0.36 | not sig (this run alone) |
| type, novel | +8.6 pp | 0.052 | borderline, misses 0.05 |
| type, recurring | −1.6 pp | 0.66 | noise |
| OOD sentinel 4/20 vs 0/20 | — | 0.11 (Fisher) | not sig |

**Q1 — Does the +6 pp type gain hold in think-on? Direction-consistent, but not
independently significant here.**
- Aggregate type: +2.3 pp (vs +6.0 no-think), **p=0.36 — not significant on this run.**
  The evidence for a real type effect is the *consistency of direction across two
  independent runs* (+6.0 no-think, +2.3 think-on), NOT this run's own p-value. Think-on
  has less headroom (nodesc type already 79.0%), which is the plausible reason it is smaller.
- **RETRACTED (was stated as a finding, is noise):** the "gain concentrates on novel
  items / redistributes easy→hard" story. The novel-subgroup deltas are non-significant
  (full-path p=0.30, type p=0.052), AND the leaky/novel effect **flips sign between the
  no-think and think-on runs** (no-think: +leaky/−novel; think-on: −leaky/+novel) — a real
  mechanism cannot flip sign, so this is run-to-run noise, not a redistribution mechanism.
- Adoption still holds on the type signal (direction-consistent across runs). T-27
  (leaf-level descriptions) should be motivated by **T-14's robust "right sheet, wrong
  leaf" dominant error class**, NOT by this run's noisy novel-type +8.6 pp.

**Q2 — Does the T-19 sentinel survive with descriptions? Observed drop 20%→0%, but
UNDERPOWERED (n=20, Fisher p=0.11) — do not treat as established.**
- think-`nodesc` 4/20 vs think-`descen` 0/20. The *mechanism* is visible (OOD items route
  to a real `Adicionais/Outros/Diversos` leaf @0.95 with descriptions, so the model never
  emits the explicit decline), but 4-vs-0 on n=20 does not clear significance and q3's
  confidence is run-to-run unstable (T-25). **A bigger OOD probe set is needed to settle
  this.**
- What IS observable (not statistical): the sentinel is a **weak** net in *every* condition
  — even at its best (think-nodesc 20%), confident-wrong non-`Diversos` OOD picks
  (paraquedas→`Extras/Casa`@0.95, taxidermia→`Extras/Casa`@0.90) slip through. `Diversos`
  routings are separately caught by `auto_insert_excluded`. **This** — not "descriptions
  drive declines to 0" — is the defensible basis for the T-24 conclusion.

**Calibration:** still broken — HC-wrong 86–88% in both conditions (marginally better than
no-think's 91–94%, still unusable as a gate). This null is the only ordering-relevant output
of T-26: nothing here touches the gate. Descriptions do NOT fix calibration — **T-23 stands
as the WS-D blocker, unchanged.**

---

# T-30 Leaf-First A/B (session 50, 2026-07-04/05)

**Question:** Does making the classifier predict the LEAF first (enum = 104 unique leaf names,
derive type/category upward via `taxonomy.ResolveLeaf`) beat the current TYPE-first full-path
enum (112 `Type/Category/Subcategory` paths)? Spun off from the session-49 T-23 logprob probe,
which found type-first fights the model's tokenization (drags it onto a Type rail before the
discriminating leaf is reachable). Judged on accuracy first; also the precondition that makes any
token-confidence signal legible (T-23).

**Harness:** `.claude/scratch/leaf-first-ab/` — `run_leaf_ab.py` runs both arms via direct
Ollama `/api/chat` calls (D3 = benchmark-only, NO production change), emitting the exact `score.py`
record shape so `score.py`/`sigtest`-style analysis run unchanged. `analyze_leaf_ab.py` adds the
right-sheet-wrong-leaf rate + paired McNemar. Same 300-item stratified T-14 sample.

**GBNF property-order confirmed FIRST** (highest-risk assumption): a two-field schema declaring
`leaf → type → confidence` makes q3 *generate* `leaf` before `type` (schema decl order → GBNF
generation order is pinned). So D1's two-field design (leaf enum + separate `type` field, the
latter disambiguating the 5 cross-type collision leaves `Ambos`/`<person E>`/`Orion`/`Dentista`/
`Estacionamento`) works — no fallback to a concatenated single-string enum needed.

**Two axes, 4 cells** (both n=300 paired, McNemar exact): **FS** = few-shot OFF (FS-A, enum-isolated
signal check) vs ON (FS-B, arm-shaped examples — `{leaf,type}` JSON for treatment vs `{path}` string
for control, same `SelectExamples` port for both; the production-realistic verdict). **think** =
no-think vs think-on (production default = few-shot ON + think-on).

## Full-path accuracy: control (path-first) → treatment (leaf-first)

| condition | control | treat | Δpp | McNemar p |
|---|---|---|---|---|
| FS-A no-think | 39.7% | 52.3% | +12.7 | <0.001 |
| FS-A think-on | 46.7% | 55.3% | +8.7 | <0.001 |
| FS-B no-think | 59.3% | 62.7% | +3.3 | 0.064 |
| **FS-B think-on (PROD)** | **59.3%** | **61.0%** | **+1.7** | **0.458** |

## Type accuracy Δ (same cells): +11.0 / +11.7 / +5.7(p=.008) / +2.3(p=.296)

**Monotonic decay to nothing.** Every mechanism that helps the path-first control catch up —
few-shot retrieval, then thinking — erodes the leaf-first advantage. In the **production
configuration (FS-B think-on), leaf-first shows NO significant gain on any metric** (full-path
+1.7 p=0.46, type +2.3 p=0.30, leaf +1.0 p=0.70; all deltas mildly positive, within noise).

**Mechanism (why the FS-A gain was real but evaporates):** leaf-first fixes "wrong-leaf" errors by
letting the model commit its strongest token (the leaf) first; few-shot fixes the SAME errors by
retrieving a near-twin with the right answer. Overlapping fixes → sub-additive when stacked.
Few-shot lifted the control +19.6pp (39.7→59.3) but the treatment only +10.4pp (52.3→62.7),
because leaf-first had already recovered part of what few-shot recovers. The one effect with a
structural (not example-substitutable) basis — TYPE, a derived lookup off the leaf — survives
longest (sig in 3/4 cells) but still washes to non-significance in the full production config.

**Harness validation:** path-first + few-shot = 59.3% (no-think) matches the production T-14/T-22
no-think baseline (~58.7%) → the few-shot port faithfully reproduces the production path.

**Caveat — T-22 descriptions held absent:** both arms rendered the taxonomy tree WITHOUT the
adopted `descen` type descriptions (held constant/absent to isolate the enum). Production runs
*with* them. This does not threaten HOLD — it strengthens it: leaf-first was given its best shot
(no competing type-boost mechanism) and its type edge still washed out; adding `descen`, itself a
type-accuracy booster, would only erode the leaf-first type Δ further.

**`type_crosscheck` (leaf-first's field-2 type vs the leaf's real owner):** proposed as a free
calibration signal. NOT established — disagree→worse in 3/4 runs but inverted in FS-A think-on,
and the disagreement count collapses toward zero as accuracy rises (n=42→18→23→**4** across cells),
so it cannot serve as a gate. A footnote for T-23, not a finding.

## Decision (D4): HOLD — do NOT adopt leaf-first *for accuracy*

- **Point estimate too small to justify a production change.** Prod-cell full-path is +1.7pp with
  only ~29 discordant pairs (12 vs 17) — n=300 cannot resolve a genuine +2–3pp effect at p<0.05,
  so this is "too small to be worth a rewrite," NOT a proven null. All prod-adjacent deltas are
  mildly positive; leaf-first is not measurably *worse*, but "costs nothing" would overclaim.
- **Still the T-23 precondition — but SCOPED.** Leaf-first is proven to make the **logprob-margin /
  answer-perplexity** signals (2a/2b) legible (type-first drags chosen tokens to raw p≈0; session-49).
  For the **ensemble-agreement** route (signal 3 — the calibration plan's "possibly more robust"
  option), leaf-first's necessity is **untested**: you can extract leaves from K sampled full-paths
  and count votes without a leaf-first enum. So: precondition for the logprob route; **open** for the
  ensemble route. If T-23 goes ensemble, leaf-first's justification is an open question (accuracy null
  + possibly-unneeded → fully on the shelf), not an established need.
- **Recommendation:** don't productionize as an accuracy change; adopt *only* if/when T-23 pursues a
  logprob-based signal, at negligible accuracy cost.
- **Couples to T-27 (leaf-level descriptions):** this probe's core finding — few-shot retrieval
  already recovers most "wrong-leaf" errors in production — undercuts T-27's stated motivation (the
  same "right sheet, wrong leaf" error class). T-27 should be re-justified on production-realistic
  (few-shot ON) evidence, or dropped.
- Full record + per-stratum tables: `.claude/plans/leaf-first-classification.md` "Execution log".
