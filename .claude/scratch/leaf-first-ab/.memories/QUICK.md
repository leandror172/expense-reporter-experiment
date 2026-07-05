# QUICK — leaf-first A/B harness (T-30, session 50)

Benchmark-only A/B: leaf-first enum (104 leaves + `type` field, derive via `ResolveLeaf`) vs the
current type-first full-path enum (112 paths). Direct Ollama calls, NO production change.

## Outcome — D4 = HOLD (do not adopt for accuracy)
Full-path Δ (leaf-first − path-first) decays across 4 cells: +12.7 (fs OFF, no-think) → +8.7 (fs
OFF, think) → +3.3 (fs ON, no-think) → **+1.7 (fs ON + think-on = PRODUCTION, p=0.46 NOT sig)**.
Few-shot retrieval already fixes the same "wrong-leaf" errors. Report: `../../t14-benchmark-report.md`
"T-30"; full log: `../../plans/leaf-first-classification.md`.

## Running
- `python3 run_leaf_ab.py --arm {pathfirst|leaffirst} [--think] [--fewshot]` — resumable by id;
  file = `results-<arm>-[fs-]<think|nothink>-<model>.jsonl`. Emits the exact `score.py` record shape.
- `python3 analyze_leaf_ab.py --mode {nothink|think} [--fewshot]` — paired McNemar + right-sheet-
  wrong-leaf rate + `type_crosscheck`. Reuses `../t14-benchmark/score.py`.
- `sample.jsonl`/`ood.jsonl` are symlinks to `../t14-benchmark/`. GPU: serialize arms (one q3, one GPU).

## Gotchas
- GBNF pins schema-declaration order → generation order (leaf-then-type verified); Python `json.dumps`
  preserves dict order (unlike Go maps, which sort keys — that trap is Go-only).
- `type_crosscheck` is NOT a usable calibration signal (sign-flips across runs; n→0 as accuracy rises).
- Same T-26 lesson: single 300-run ≈ ±5pp, subgroup splits noisy; trust cross-cell consistency.
