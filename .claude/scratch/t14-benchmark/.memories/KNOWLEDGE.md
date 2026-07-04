# KNOWLEDGE — t14-benchmark harness

## Files
- `run_benchmark.py` — one classify call per sample item → append+flush JSONL (resumable
  by id). `--model --sample --prefix --extra-args --limit --timeout`.
- `score.py` — globs `results-*.jsonl`, joins to `sample.jsonl` by id, prints accuracy /
  latency / calibration / leaky split / OOD sentinel. Pairs each with `ood-<name>`.
  Label = filename minus `results-`/`.jsonl`.
- `sigtest_t26.py` — paired McNemar (full-path + type, overall + leaky/novel subgroups) and
  Fisher exact (sentinel). Imports `score.py` predicates. Reusable for any A/B in this dir.
- `run_t26.sh` — think-on/no-think A/B via **sidecar toggle** (supersedes `run_t22.sh`).
  Builds the binary, backs up + restores the sidecar via an EXIT trap, writes
  `results-<mode>-<cond>-<model>.jsonl`. `*.bak` and `results-*.jsonl` are gitignored.
- `run_t22.sh` — STALE: swaps `taxonomy.json` (pre-adoption). Kept only as the record of the
  no-think A/B. Do not run post-adoption.

## The sidecar toggle (why it's the correct mechanism)
`config/config.json` sets `type_descriptions_path: config/type-descriptions.json`, resolved
relative to the **binary** (`expense-reporter/config/…`). `taxonomy.LoadTypeDescriptions`
returns an empty map + nil error when the file is absent → **deleting the sidecar IS the
clean `nodesc` condition** (no error, no descriptions). Present = `descen`. `generate-workbook`
never renders descriptions, so only the classifier path is affected.

## T-26 findings (think-on A/B, 2026-07-03) — what survived significance testing
Full write-up: `../t14-benchmark-report.md` "Results — think-on". The lesson worth carrying:

- **Almost every single-run delta was within noise.** Paired McNemar verdicts:
  full-path all Δ−0.7pp (p=0.86), novel Δ+4.3pp (p=0.30), type all Δ+2.3pp (p=0.36),
  type novel Δ+8.6pp (p=0.052 borderline), sentinel 4/20-vs-0/20 (Fisher p=0.11). **None
  clear p<0.05.**
- **RETRACTED as noise** (had been written into the report as findings): (1) "descriptions
  concentrate the win on novel items / redistribute easy→hard" — the leaky/novel effect
  **flips sign** between the no-think run (+leaky/−novel) and the think-on run (−leaky/+novel);
  a real mechanism can't flip sign. (2) "descriptions suppress the T-19 sentinel to 0" —
  underpowered (n=20). Do NOT let these drive T-24/T-27 reasoning.
- **What IS robust:** (a) type-accuracy lift is **direction-consistent across two independent
  runs** (+6.0pp no-think, +2.3pp think-on) — that consistency, not any single p-value, is
  the evidence T-22 adoption rests on; (b) **calibration is broken in every condition**
  (86–95% of wrong answers ≥0.85 conf) → the 0.85 auto-insert gate protects nothing; this is
  the WS-D blocker (T-23), untouched by descriptions in either think mode; (c) the T-19
  sentinel is a **weak** net in every condition (confident-wrong non-`Diversos` OOD picks slip
  through even at its best, think-nodesc 20%).
- **Ordering conclusion:** T-26 changed nothing about task order. T-23 (calibration/gate
  rethink) is still next; PR #42 merge is unaffected; T-27 should be justified by T-14's
  robust "right sheet, wrong leaf" error class, not this run's noisy novel-type number.

## Sample / labels
`sample.jsonl` (300, from the gitignored corpus — real descriptions, never commit),
`ood.jsonl` (20, synthetic, safe to commit). Sample rows carry `expected_{type,category,
subcategory}` + `leaky` (bool). `.gitignore` here: `*.jsonl` (except `ood.jsonl`), `*.bak`,
`taxonomy-*.json`, `*.log`.
