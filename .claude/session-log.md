# Session Log — Expense Reporter

**Current Session:** 2026-07-04 — Session 49: T-23 logprob-confidence probe + leaf-first spin-off (D1 locked) + taxonomy audit
**Current Layer:** Classifier calibration & accuracy — T-23 gate rethink / leaf-first
Most recent entry first. Run `.claude/tools/rotate-session-log.sh` when this grows beyond ~3 sessions.

---
## 2026-07-04 - Session 49: T-23 logprob-confidence probe + leaf-first spin-off (D1 locked) + taxonomy audit

### Context

Started mid-review of PR #42 (T-22) with a request to discuss next steps. A T-23 (gate rethink) discussion — sparked by the user's idea of reading confidence off the token distribution — turned into an empirical Ollama-logprob probe, which spun off leaf-first classification as its own thread. All output this session is docs/plans/memory (no product code); the taxonomy rename touched gitignored data files (not in git).

### What Was Done

- Probed Ollama 0.17.5 logprobs empirically: feasibility gate PASSES (logprobs exposed under the `format` grammar), but the reported logprobs are PRE-grammar-mask (raw, not renormalized) — overturned the initial "grammar renormalizes → token-dist == taxonomy-dist" theory.
- Ran the leaf-first legibility probe: type-first enum makes the margin unreadable (chosen tokens raw p≈0); LEAF-FIRST makes it legible (Uber leaf p=0.986, readable competitors). Wrote it up: `.claude/t23-logprob-confidence-probe.md`.
- Authored the T-23 calibration-benchmark plan (`.claude/plans/t23-calibration-benchmark.md`): 4 signals (self-confidence baseline / answer-perplexity / logprob-margin / ensemble-agreement), risk-coverage as primary metric, engine/library rundown + deferred 2b engine-PoC subagent scope.
- Authored the leaf-first classification plan (`.claude/plans/leaf-first-classification.md`) as a standalone accuracy change + T-23 precondition, with full intent narrative.
- Ran the taxonomy junk-leaf audit: 1 junk leaf `Apoia-se 4i20` → generalized to `Apoia-se`; full surgical propagation (labels only, real `item` text kept) across taxonomy + feature dict + training + logs (backups `*.bak-apoia-rename`). The 5 cross-type collisions confirmed as real recurrence structure.
- Updated classifier QUICK.md + KNOWLEDGE.md, wrote memory `project_logprob_confidence_leaf_first`, created internal tasks (#1 audit DONE, #2 leaf-first, #3 = T-23 benchmark, #4 = T-20). Branch `docs/t23-calibration-probe` (4 commits, unpushed).

### Decisions Made

- **D1 LOCKED** for leaf-first collision handling = second `type` field, schema order `leaf` THEN `type` (order load-bearing), always-predict, `ResolveLeaf` disambiguates the 5 collisions + cross-checks the 99 unique leaves as a free calibration signal. Grounded in the audit: the collision axis IS recurrence (pets/dentista/parking span Fixas/Variáveis/Extras — not derivable from the leaf name).
- **Split leaf-first from T-23** — judged on accuracy alone first (attacks T-14's dominant "right sheet, wrong leaf"); the confidence benchmark rides on a leaf-first base.
- **`Apoia-se 4i20` → `Apoia-se`** (vendor granularity, NOT an `Assinaturas` bucket — keeps future Netflix/HBO/Spotify generalizations consistent).
- **2b engine work deferred** — run signals 1/2a/3 on Ollama first; only send the llama.cpp/vLLM engine-PoC subagent if 2b earns it (ask model before spawning).

### Next

- Build the **leaf-first A/B** (new T-30) — D1 settled; decide D2 (tree vs flat prompt) / D3 (benchmark-only vs prod) / D4 (adopt threshold) at build time.
- **T-20** dedup as an independent pick-up while PR #42 is under review (deterministic, no Ollama).
- **T-23** calibration benchmark rides on the leaf-first base once it lands.
- Durability: rename the `Apoia-se` leaf in the workbook Referência (or export step) so it survives taxonomy re-export (T-02).

### Gotchas

- Ollama 0.17.5 reports PRE-grammar-mask logprobs (raw distribution, not renormalized over legal tokens; legal tokens often outside top_logprobs) — you cannot read "P over legal paths" directly.
- A single token is NOT the leaf margin — the first char measures "which initial letter" (many leaves share it), not "which leaf"; the real signal is the whole-leaf sequence logprob + margin to runner-up.
- The taxonomy rename touched gitignored data files (not in git); `*.bak-apoia-rename` backups sit in the working tree; the docs branch is unpushed; classifier QUICK/KNOWLEDGE edits are uncommitted.
