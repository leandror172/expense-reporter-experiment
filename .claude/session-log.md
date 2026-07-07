# Session Log — Expense Reporter

**Current Session:** 2026-07-06 — Session 52: 649-replay measures T-23 gate + retrieval on real data — confidence dead, specificity+agreement gate, 5.R2 lever
**Current Layer:** Layer 5 — classifier gate & retrieval (T-23 measured → 5.R2 next)
Most recent entry first. Run `.claude/tools/rotate-session-log.sh` when this grows beyond ~3 sessions.

---
## 2026-07-06 - Session 52: 649-replay measures T-23 gate + retrieval on real data — confidence dead, specificity+agreement gate, 5.R2 lever

### Context

Resumed from session 51 (T-23 route reframe). User asked to discuss next steps besides PR #43, then chose to run the settled unblocked measurement — the 649-replay — as a retrieval probe, weighing it against the grand vision (chat-tap / messaging workflow, review-first).

### What Was Done

- Built a faithful in-package build-tagged replay harness (`//go:build replay`) that drives the REAL production retrieval + classifier (not a Python reimplementation): `replay_retrieval_test.go` (Ollama-free) + `replay_model_test.go` (Ollama, resumable, LOO by item, env-tunable band/think).
- Retrieval half: keyword miss rate 24.7%, 100% `no_keyword_match`, 160/160 misses have an in-pool same-subcat neighbor; specificity monotone (90.5% keyword-top1 at spec=1.0), frequency not additive, singleton-masquerade absent. self_match=0 tripwire clean.
- Model half (649 rows, no-think): confidence DEAD as gate (52.5→56.3% flat); specificity `top_score` real monotone discriminator (52→87% all / 44→83% clean); AGREEMENT gate (spec=1.0 ∧ model==keyword-top1) → 95.0% subcat (201 rows); keyword-top1 beats the 8B model on subcat at the top band (91.0 vs 87.8).
- think-on top-band re-run (316 rows, 58 min): discriminator shape robust to think mode (band-lift spread 0.2pp), level −2.3pp → gate can run no-think.
- Advisor stress-test → caught the review-subset representativeness bug (via `expenses_log.jsonl`: 725 unique expenses, only 347 reviewed, 378 bypassed+unlabeled → absolute precision UNMEASURABLE), corrected the "model beats keyword" claim, softened 5.R2 to a hypothesis.
- Durable findings + memory: `.claude/scratch/replay-649/{FINDINGS,FINDINGS-model}.md` + analysis scripts + folder QUICK.md; updated classifier + repo-root memories, index.md, project memories. Committed c0a3015; PR #43 title/body updated.

### Decisions Made

- SHIP vs HOLD split is load-bearing: SHIP (bias-robust, same-sample) = confidence dead + specificity replaces it + agreement gate + gate runs no-think. HOLD (depend on absolute precision the confidence-selected subset can't establish) = "no unattended auto-insert justified" + WS-D silent-insert scoping.
- Gate design candidate = AGREEMENT (spec=1.0 ∧ model==keyword-top1), possibly keyword-first/model-fallback hybrid; runs no-think.
- 5.R1 TF-IDF RULED OUT (misses are semantic zero-overlap, not vocab); 5.R2 embeddings is the lever but SOFTENED — gated on a cheap NN-retrieval precondition before building.
- Faithful Go build-tagged harness chosen over Python reimplementation (divergence risk on tokenizer/specificity) — advisor-blessed.

### Next

- **5.R2 embeddings — settled next point, gated on the NN-retrieval precondition (T-31):** embed the 160 no-keyword-match misses + their pool-mates (Ollama `/api/embeddings`, ~minutes), measure nearest-neighbor subcat hit-rate. Weak → novel items are permanent-review, not 5.R2; strong → build 5.R2.
- Then wire the specificity/agreement gate into `IsAutoInsertable` (T-32) — the WS-D unblock (no new inference; expose match-strength from `SelectExamples` → thread out of `Classify` → widen gate → 3 call sites).
- T-20 log dedup remains a clean independent pick-up.

### Gotchas

- Go-test cwd = PACKAGE dir; `REPLAY_*_OUT` relative paths resolve from `internal/classifier/` (default `../../../.claude/...`), and O_CREATE does NOT mkdir parents — a wrong path fails fast (that killed the first think-run).
- `classify` does NOT set FeedbackPath (training-only few-shot); `auto`/`batch-auto` DO → the gate lives on the auto path; the model harness replicates the `auto` config.
- The 649 = confidence-selected review subset (378/725 bypassed, unlabeled) → never claim absolute precision off it; only relative (same-sample) findings are safe.
- T-14's "+3pp think-on" is a full-sample average — it goes slightly NEGATIVE (−2.3pp) on the high-specificity band where the gate operates.
