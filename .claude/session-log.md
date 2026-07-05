# Session Log — Expense Reporter

**Current Session:** 2026-07-05 — Session 51: T-23 route reframe — external validators over model-introspection (recurrence-first); validated leaf-first HOLD; PR #43
**Current Layer:** Classifier calibration & accuracy — T-23 gate route (external validators)
Most recent entry first. Run `.claude/tools/rotate-session-log.sh` when this grows beyond ~3 sessions.

---
## 2026-07-05 - Session 51: T-23 route reframe — external validators over model-introspection (recurrence-first); validated leaf-first HOLD; PR #43

### Context

Resumed after session 50's handoff to (a) validate its leaf-first D4=HOLD and (b) decide the T-23 signal route it unblocked. Pure analysis/docs — no product code.

### What Was Done

- **Independently validated session-50's leaf-first D4=HOLD:** recomputed the production cell (full-path +1.7 p=0.46) and the load-bearing novel stratum (+2.6 n.s.) from the cached A/B results — HOLD confirmed (the novel number was the one that could have overturned it).
- **Read the classifier end-to-end** (`decision.go` / `examples.go` / `loader.go` + `auto`/`batch-auto` wiring): the gate is confidence-only (`IsAutoInsertable`), and the recurrence signal (keyword specificity) is computed in `SelectExamples` then **DISCARDED** — grep-confirmed it never leaves the function.
- **Reframed the T-23 route (advisor-reviewed)** around external validators vs. model-introspection; wrote it into `.claude/plans/t23-calibration-benchmark.md` "Route decision" (session-49 token framing retained + scoped) + a new `.claude/t23-strategic-implications.md`.
- Persisted: classifier KNOWLEDGE (as-is gate + reframe), memories (updated `project_logprob_confidence_leaf_first`, created `project_t23_gate_route_strategy`), index.
- **Created PR #43** (`docs/t23-calibration-probe` → master; #42 merged so the base is clean, docs-only diff).

### Decisions Made

- **Gate route = EXTERNAL VALIDATORS (recurrence-strength / value-range) primary, MODEL-INTROSPECTION (ensemble → logprob) secondary.** Recurrence (recurring ≈70% vs novel ≈49%, ~25pp) dwarfs any token signal and is orthogonal to the model's anti-informative self-confidence. First study = recurrence-strength risk–coverage stratified recurrent/novel; the curve decides whether any introspection signal is needed. **NOT a final gate decision — deferred.**
- Advisor corrections integrated: it's the first STUDY, not the decided route; introspection's unique test is the NOVEL pool (not "refining the recurrent pool"); "can't gate a coin flip" is too strong (49% is an average); E3 retrieval-agreement carries the `type_crosscheck` circularity (few-shot injection breaks independence → 4-cell stability test first); value-range is a NEGATIVE flag only; nothing pre-shelved.
- Strategic: the auto-insert vision **bifurcates by recurrence, not confidence**; the review UI is first-class; the review→feedback loop amortizes the novel-review cost; retrieval (5.R2 embeddings > 5.R1 TF-IDF) is the real accuracy lever; deprioritize prompt/enum tuning (T-27 re-justify or drop).

### Next

- **T-23 recurrence-strength study** (E1 risk–coverage stratified recurrent/novel, + E2 value-range negative flag) — the cheapest, unblocked first move and the WS-D critical-path opener.
- **T-20** dedup — independent, deterministic pick-up.
- Then **WS-D (T-09)** retire bare-name fallback → **WS-E**.

### Gotchas

- The recurrence signal EXISTS but is thrown away in `SelectExamples` (never reaches `Result`/gate). Wiring it in = ~5-file plumbing (expose → thread out of `Classify` → widen `IsAutoInsertable` → 3 call sites), no new inference.
- E3 retrieval-generation agreement is NOT two independent methods — few-shot injects the matched keyword's examples, nudging the model toward the retrieved answer (the same non-independence that sign-flipped `type_crosscheck`). Test 4-cell stability before relying on it.
