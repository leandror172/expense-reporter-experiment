# Session Log — Expense Reporter

**Current Session:** 2026-07-10 — Session 54: 5.R2 embedding retrieval — built, A/B'd, ADOPTED (miss full-path 18.8%→52.5%); PR #45
**Current Layer:** Layer 5 — retrieval & gate (5.R2 DONE; T-32 next)
Most recent entry first. Run `.claude/tools/rotate-session-log.sh` when this grows beyond ~3 sessions.

---
## 2026-07-10 - Session 54: 5.R2 embedding retrieval — built, A/B'd, ADOPTED (miss full-path 18.8%→52.5%); PR #45

### Context

Started from the T-31 GO verdict (session 53). Planned 5.R2 interactively (12 locked decisions), then executed all 4 phases in one session: subagent for Phases 1+2, in-session Phases 3+4.

### What Was Done

- docs(5r2): embedding retrieval plan — decisions locked (D1–D12), phases defined (`.claude/plans/5r2-embedding-retrieval.md`)
- feat(5r2): embedding client + JSONL cache + cosine top-K retriever (Phases 1+2) — built by an `impl-opus-med` subagent (new agent definition copied from latent-topic-graph, adapted to this repo at `.claude/agents/impl-opus-med.md`); 24 unit tests
- feat(5r2): wire embed-on-miss fallback into `selectExamples` (Phase 3) — `embedding_fallback.go`, `Config.EmbedModel`/`NoEmbedRetrieval` (zero-value = ON), 6 unit tests + 1 live-verified acceptance test (`test/embedding_fallback_test.go` + fake mini-pool fixture); embedding caches gitignored
- feat(5r2): Phase 4 replay A/B — ADOPTED: miss-stratum full-path 18.8%→52.5% no-think (+33.8pp, 29 fixed/2 broke), 55.0% think-on; `REPLAY_MISS_ONLY`/`REPLAY_NO_EMBED` harness knobs; report `.claude/scratch/replay-649/FINDINGS-5r2.md`
- docs(5r2): adoption propagated to QUICK/KNOWLEDGE/README/index + `embedding-retrieval.md` status header
- Created persona `my-go-q3-14b` (qwen3:14b, copied from my-go-qcoder) and ran it as the session's sole Go codegen model — informal benchmark: verdicts 0/1/0/1 (subagent, Phases 1+2) + 1 (Phase 3) vs qcoder's 2/2/1/1; pulled `snowflake-arctic-embed2`
- Opened PR #45 (`feat/5r2-embedding-retrieval` → master); merged PR #44 (user, session start)

### Decisions Made

- 5.R2 ADOPTED as production default (fallback ON by zero-value config) — the +33.8pp miss-stratum lift with 29 fixed/2 broke is decisive; think-on adds only +2.5pp churn at 5.5× latency, so no-think is right for the miss path (feeds T-24)
- D1–D12 locked in the plan; notably: miss-only trigger (CL1 revisits widening), lazy BLOCKING sync.Once reconcile (async would classify early/late misses inconsistently within a batch), raw-text cache keys, K=5 mandatory
- CL1 (trigger widening) + CL2 (pre-warm/burst) deliberately deferred to post-A/B — recorded in the plan's Check-later section, not on the tasks board
- qwen3:14b below qcoder for Go on this sample (needed a directed retry per file); model strategy reverted to cascade (q3-14b first, then previous tiers on repeated 0s)

### Next

- Merge PR #45 (user)
- T-32: wire the specificity/agreement gate into `IsAutoInsertable` (WS-D unblock) — independent, parallel-able; design gate-to-review, not silent insert
- Still open: rename `Apoia-se` in workbook Referência; promote all-years log; T-20 dedup; T-24/T-25/T-27/T-28

### Gotchas

- Workflow tool `args` must be a real JSON object, not a stringified one — a string arg made `args.prompt` undefined and the subagent received only boilerplate (34K tokens wasted)
- A workflow/background agent's final message ENDS its run — the first Phase-1+2 attempt died by ending its turn "waiting for the GPU"; also its abandoned qwen3:14b thinking calls queue-starved Ollama (the T-14 ops gotcha, now baked into the agent prompt as guardrails)
- `generate_code` with a RELATIVE `output_file` resolves against the ollama-bridge REPO_ROOT (the LLM repo), not the target repo — always pass absolute paths (stray file was created and deleted)
- `GenerateID(item,date,value)` collides for repeat expenses → replay outputs carry ~2 lines per id; score on unique ids (n=80, not 160)
- `/data/classification/embeddings-*.jsonl` was NOT gitignored until this session — the cache stores real expense descriptions; rule added before any real cache was written
