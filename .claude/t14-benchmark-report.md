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
