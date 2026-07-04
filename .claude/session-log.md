# Session Log — Expense Reporter

**Current Session:** 2026-07-03 — Session 48: T-26 think-on confirmation of T-22 descriptions — validated (mostly noise), T-23 stands
**Current Layer:** Classifier accuracy & calibration → T-23 gate rethink (WS-D blocker) next
Most recent entry first. Run `.claude/tools/rotate-session-log.sh` when this grows beyond ~3 sessions.

---
## 2026-07-03 - Session 48: T-26 think-on confirmation of T-22 descriptions — validated (mostly noise), T-23 stands

### Context

Resumed on branch `feat/t22-type-descriptions` (PR #42 open). User asked to discuss next steps, chose the "cheaper adjacent win" — T-26 (the deferred T-22 think-on confirmation) — over starting the harder T-23. Ordered T-26 before T-24 because T-24's `--think` decision consumes T-26's sentinel data.

### What Was Done

- Reworked the stale benchmark runner: authored `.claude/scratch/t14-benchmark/run_t26.sh` — toggles the **sidecar** `config/type-descriptions.json` (removed=nodesc, present=descen) instead of `run_t22.sh`'s pre-adoption taxonomy-swap; builds the binary itself; encodes the think-mode into result prefixes (`results-<mode>-<cond>-…`) so `run_benchmark.py`'s resume-by-id can't mistake the old no-think files for this run's work.
- Ran the think-on A/B (nodesc vs descen, 300-item accuracy + 20-item OOD each, q3, ~2.5 h wall-clock, resumable, 0 failures). Sidecar restored to adopted state by the EXIT trap.
- Significance-tested the deltas with `sigtest_t26.py` (paired McNemar + Fisher exact, reuses `score.py` predicates, zero model calls) — prompted by an advisor call.
- Updated `.claude/t14-benchmark-report.md` with a "Results — think-on" section, softened to significance-tested language.
- Created `.claude/scratch/t14-benchmark/.memories/{QUICK,KNOWLEDGE}.md`.
- No package code changed this session — benchmark/report/tooling only (hence no feature commits).

### Decisions Made

- **T-26 done; task order UNCHANGED — T-23 is still next.** The only ordering-relevant output is a null: calibration is untouched (86–88% HC-wrong), so nothing here moves the auto-insert gate.
- **Retracted two report findings as noise after significance testing** (they had been written in as fact): (1) "descriptions concentrate the win on novel items / redistribute easy→hard" — the leaky/novel effect **flips sign** between the no-think (+leaky/−novel) and think-on (−leaky/+novel) runs, and a real mechanism can't flip sign; subgroup deltas non-sig (novel full-path p=0.30, novel type p=0.052). (2) "descriptions suppress the T-19 sentinel to 0" — 4/20-vs-0/20, Fisher p=0.11, underpowered.
- **T-22 adoption still holds**, but on the *cross-run direction consistency* of the type lift (+6.0 no-think, +2.3 think-on), NOT think-on's own delta (p=0.36, not significant alone).
- **T-24 framing corrected:** with descriptions on (production default), think-on's sentinel advantage vanishes (the sentinel is a weak net in *every* condition — confident-wrong non-`Diversos` OOD picks slip through even at its best 20%). So the `--think` call is accuracy-vs-speed (~+2.7 pp full-path for ~10× latency), not "descriptions kill the sentinel."

### Next

- **T-23 — calibration / gate rethink** (the WS-D blocker). T-26 confirmed descriptions do NOT fix the gate. Directions: margin/agreement thresholding, few-shot with varied confidence, review-first for non-recurring items (recurrence beats confidence as a safety signal), stronger `auto_insert_excluded`.
- Merge PR #42 (updated with the T-26 think-on results).
- Candidate follow-up task (propose via amend): a larger OOD probe set to settle the sentinel-suppression question the n=20 run left underpowered.
- Later: T-27 (leaf-level descriptions — justify via T-14's "wrong leaf" error class, NOT this run's noisy novel-type number), T-24 (`--think` default), T-28 (alt strategies).

### Gotchas

- `go build ./...` does NOT emit the cmd binary — must `go build -o expense-reporter ./cmd/expense-reporter` before benchmarking (`run_t26.sh` does this).
- Benchmark resume keys on the results **filename** — reusing a prior run's prefix makes `done_ids` "skip everything" silently. New runs need a fresh prefix (`run_t26.sh` encodes the think-mode).
- A single 300-item run resolves only ~±5 pp; subgroup splits (n=116, n=20) can't clear noise alone, and a subgroup effect that flips sign across runs is noise. Run `sigtest_t26.py` before treating any delta as real.
