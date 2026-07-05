# Session Log — Expense Reporter

**Current Session:** 2026-07-05 — Session 50: T-30 leaf-first A/B — D4 = HOLD (enum gain washes out under few-shot; scoped to T-23 logprob precondition)
**Current Layer:** Classifier calibration & accuracy — T-23 gate rethink / leaf-first
Most recent entry first. Run `.claude/tools/rotate-session-log.sh` when this grows beyond ~3 sessions.

---
## 2026-07-05 - Session 50: T-30 leaf-first A/B — D4 = HOLD (enum gain washes out under few-shot; scoped to T-23 logprob precondition)

### Context

Continued session 49's leaf-first thread. User caught an early over-reach — I proposed a production Go `--leaf-first` flag; corrected to a benchmark-only probe (the plans' stated approach) and saved a feedback memory. GPU usage was gated tightly (paused between arms for another task). All work is benchmark + docs — no product code changed (D3 = benchmark-only).

### What Was Done

- Re-read every leaf-first/T-23 companion doc; **reframed the work from "production feature" to "standalone benchmark probe"** and saved `feedback_probe_not_feature` memory.
- **Verified the GBNF property-order assumption FIRST** (highest risk): a two-field schema `leaf → type → confidence` makes q3 *generate* leaf before type (decl order → GBNF generation order pinned) → D1's two-field design is sound, no single-string-enum fallback needed.
- Built the benchmark-only A/B harness `.claude/scratch/leaf-first-ab/` — `run_leaf_ab.py` (both arms via direct Ollama, emits the exact `score.py` record shape, resumable, `--fewshot`/`--think` axes), `analyze_leaf_ab.py` (paired McNemar + right-sheet-wrong-leaf rate + `type_crosscheck`), symlinked sample/ood, reused `score.py`/`sigtest` unchanged. Ported `SelectExamples` few-shot (arm-shaped: `{leaf,type}` JSON vs `{path}` string).
- Ran the full **4-cell A/B** (few-shot OFF/ON × no-think/think-on, 300 items each, 0 failures): FS-A +12.7 / +8.7 pp full-path (both p<0.001); FS-B +3.3 (p=0.064) / **+1.7 (p=0.46) = production, not sig**.
- Wrote the **T-30 report section** (`.claude/t14-benchmark-report.md`) + the plan execution log; updated classifier QUICK/KNOWLEDGE, `project_logprob_confidence_leaf_first`, harness QUICK.
- Advisor-reviewed the HOLD verdict; applied 3 sharpenings (underpowered ≠ proven null; scoped T-23 precondition; T-22 descriptions held absent) + the T-27 coupling flag.

### Decisions Made

- **D4 = HOLD — do NOT adopt leaf-first for accuracy.** Production config (few-shot ON + think-on) shows no significant gain on any metric; the FS-A gain is real but few-shot retrieval already fixes the same "wrong-leaf" errors (sub-additive stacking). Not measurably worse, but the +1.7pp point estimate is underpowered — "too small to justify a rewrite," not a proven null.
- **Leaf-first remains the T-23 precondition but SCOPED:** proven only for the logprob-margin/perplexity signals (2a/2b); OPEN/possibly-unneeded for the ensemble-agreement signal (3, extractable from K sampled full-paths). If T-23 goes ensemble, leaf-first is fully on the shelf.
- **`type_crosscheck` rejected as a calibration signal** — disagree→worse in 3/4 runs, inverted once, disagreements → 0 as accuracy rises. T-23 footnote, not a gate.
- **Resolved D2=tree, D3=benchmark-only; few-shot is its own axis (FS-A/FS-B), not D3.**
- **T-27 (leaf-level descriptions) motivation undercut** — few-shot already recovers most wrong-leaf errors; re-justify on few-shot-ON evidence or drop.

### Next

- **T-23 calibration benchmark — decide the signal route FIRST:** logprob/perplexity (2a/2b, needs a leaf-first base) vs **ensemble-agreement** (3, needs NO leaf-first) vs recurrence-first fallback. Risk-coverage on the T-14 sample, stratify leaky/clean. This choice decides whether leaf-first is ever revived.
- Independent pick-up: **T-20** expense-log dedup (deterministic, no Ollama).
- Housekeeping: merge **PR #42** (T-22); rename `Apoia-se` in the workbook Referência (durability, or re-export wipes it).

### Gotchas

- **FS-A's +12.7pp is misleading in isolation** — it evaporates to +1.7pp (ns) once few-shot is on. Always gate adoption on the few-shot-ON cell, never the enum-isolated one.
- **Client-kill does NOT stop Ollama server-side generation** (queue-starve) — used `pkill` to break the chained `&&` job so the second arm wouldn't auto-start, then resumed the first arm (resumable by id).
- **GBNF pins schema property-declaration order → generation order** (leaf-then-type verified); Python `json.dumps` preserves dict order — the Go map-key-sorting trap is Go-only, irrelevant to this pure-Python harness.
- Harness reuse: `score.py` globs `results-*.jsonl` in `--dir`; the new harness symlinks `sample.jsonl`/`ood.jsonl` and emits the matching record shape, so `score.py`/`sigtest_t26.py` run unchanged. `path-first+few-shot` = 59.3% matched the production baseline (~58.7%) → few-shot port validated.
