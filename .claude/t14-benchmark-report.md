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
