# Session Log — Expense Reporter

**Current Session:** 2026-07-12 — Session 56: Harness extraction plan (T2) — 4-source exploration; plan + acceptance-test comparison companion
**Current Layer:** Layer 5 — retrieval & gate (5.R2 DONE; T-32 next) + T2 extraction planned
Most recent entry first. Run `.claude/tools/rotate-session-log.sh` when this grows beyond ~3 sessions.

---
## 2026-07-12 - Session 56: Harness extraction plan (T2) — 4-source exploration; plan + acceptance-test comparison companion

### Context

Planning session, no product code. User asked to plan the T2 harness extraction: explore the current harness, how career-search used its copy, latent-topic-graph's acceptance-test concepts (+ a haiku sweep of the llm repo), and the old-job `~/workspaces/acceptance-test` framework — then write a detailed plan for a next session. Exploration ran as 4 parallel subagents (3 sonnet Explore + 1 haiku) feeding targeted direct reads.

### What Was Done

- docs(t2): harness extraction plan + acceptance-test comparison companion — `.claude/plans/harness-extraction-plan.md` + `.claude/plans/harness-extraction-acceptance-test-comparison.md`, both indexed in `.claude/index.md`
- Explored all four sources: this repo's `test/harness-external-usage.md` + harness source; career-search's de-domained copy (`dashboard/test/harness/` + `EXTRACTION-NOTES.md` — the de-facto extraction spec; its TC-EXT task is BLOCKED on our module); old-job `acceptance-test` (worker-pool + circuit-breaker assertions, `ignore_order_on`, status-only asserts, blind-sleep observe anti-lesson; cobra never actually shipped there); LTG's GIVEN→RUN→OBSERVE→ASSERT + MANUAL formalism (`ltg-l06-incremental-refresh.md`); llm repo surveyed → NOT a consumer (pytest + manual scenarios, no harness desire)
- Wrote the full comparison companion doc (adopt/adapt/discard analysis of acceptance-test, with file:line anchors)

### Decisions Made

- New repo: `~/workspaces/acceptance-harness`, module `github.com/leandror172/acceptance-harness`, PUBLIC, MIT
- v1 = lean core only (no LLM extension packages; Ollama gate + soft-assert/drift stay in expense-reporter's domain layer; `extern/llm` is roadmap)
- Seed from career-search's de-domained copy, NOT the expenses original (second consumer already shook out the boundary)
- LTG = design influence + adoption-path doc only (Artifacts documented as the OBSERVE named-capture register); no LTG code in v1
- Split sequencing: Session A = create repo + repoint career-search (closes its TC-EXT); Session B = migrate expense-reporter (`-tags=acceptance -timeout 30m` + replay-tag compile as gates)
- Library design corrections beyond the field reports: strip `//go:build acceptance` from lib code (consumer policy), replace package-level `flag.Bool` with env vars + opt-in `RegisterFlags()`

### Next

- Execute harness extraction Session A per `.claude/plans/harness-extraction-plan.md` §4 (scaffold repo from career-search seed → docs → unit tests → publish v0.1.0 → repoint career-search)
- Merge PR #45 (user) — now also carries this session's docs commit
- T-32 (specificity/agreement gate into `IsAutoInsertable`) still the settled next product step — independent of the extraction track

### Gotchas

- The session-56 docs commit landed on `feat/5r2-embedding-retrieval` (current branch), so it rides in PR #45 — fine if merging soon, cherry-pick to master if not
- `ref-lookup.sh` has no `--paths` flag (silently no-ops) — known modes: bare KEY, `--list`, no-args
- career-search's harness copy was adapted at old expense branch `feat/t13-classifier-full-path` — diff against current master before seeding the new repo (plan §4 caveats)
