# Session Context — Expense Reporter

**Purpose:** User preferences and working context across Claude Code sessions.

---

<!-- ref:user-prefs -->
## User Preferences

### Interaction Style
- **Output style:** Explanatory (educational insights with task completion)
- **Pacing:** Interactive — pause after each phase for user input
- **Explanations:** Explain the "why" for each step, like a practical tutorial

### Configuration Files
- **Build incrementally:** Never dump full config files at once
- **Explain each setting:** Add a setting, explain what it does, then add the next
- **Ask before proceeding:** Give user options before making non-obvious choices

### Local Model Usage
- **Try local models first** for Go boilerplate, simple functions, test stubs
- **Preferred model:** `my-go-qcoder` (qwen3-coder:30b via ollama-bridge MCP) — benchmarked session 23 (2026-05-18); verdicts 2/2/1/1 on test + cobra generation. Fallback: `my-go-q25c14` (qwen2.5-coder:14b) if qcoder unavailable (qcoder HTTP 500 = host-RAM ENOMEM from 9p/no-mmap, not VRAM — see benchmark note below).
- **Speed vs cost:** 30s for correct 14B output beats 6s for wrong 8B output. Local inference cost is latency only.
- **Verdict pattern:** Always record verdict **2/1/0** after local model output (2=accepted, 1=improved, 0=rejected)

### Shell Scripts
- Always use `./script.sh` not `bash script.sh` — `./` form is whitelistable per-script in Claude Code
<!-- /ref:user-prefs -->

---

## File Management

### Sensitive Data
- **Location:** `.claude/local/` (gitignored) and `data/classification/*.json` (personal expense data)
- **Rule:** Real expense descriptions, training data, personal financial info → always gitignored

### Log Rotation
- **Tool:** `.claude/tools/rotate-session-log.sh` — run at session end via session-handoff skill
- **Policy:** Keep 3 most recent sessions in `session-log.md`; archive the rest

---

<!-- ref:current-status -->

- **Pre-history (Claude Desktop):** Phases 1–11 complete — full CLI (add/batch/version), 190+ tests, v2.1.0
- **Classification analysis:** Complete (auto-category work) — results in `data/classification/`
- **Workbook Generator:** COMPLETE; PR #27 merged. `internal/taxonomy` extracted (T-02) + real `config/taxonomy.json` (gitignored) + full-path identity key.
- **Plan A (T-05) + Plan B (T-04):** MERGED. Expense `Type` persisted end-to-end; two-tier routing — typed by full path (`byPath`), type-less via the **transitional** bare-name fallback (`byName` + ambiguous-skip).
- **5.R4 historical extraction — DONE.** 2022–2025 old workbooks → corpus 694→1788 + per-year expense logs.
- **PIVOT (session 36): retire workbook insertion, keep only generation** — JSONL logs are the single source of truth, `generate-workbook` the only writer.
- **WS-A/T-11, WS-C income route — DONE.** T-13 (full-path 112-enum classifier), WS-B (commands→log-append), T-19 (sentinel decline), T-14/T-22/T-26 (accuracy/descriptions/think-on benchmarks), T-30 (leaf-first A/B → HOLD) — all DONE & merged.
- **T-23 ROUTE REFRAMED (session 51):** gate = **external validators** (E1 recurrence-strength / E2 value-range / E3 retrieval-agreement) over **model-introspection**. Recurrence dwarfs any token signal. PR #43 (docs) merged. NOT decided — deferred.
- **T-23 GATE + RETRIEVAL MEASURED ON REAL DATA (session 52) — the 649-replay.** Reports `.claude/scratch/replay-649/{FINDINGS,FINDINGS-model}.md`; harness `internal/classifier/replay_{retrieval,model}_test.go` (`//go:build replay`). SHIP: confidence DEAD as gate; **specificity (`top_score`) replaces it**; **best gate = AGREEMENT `spec==1.0` ∧ model==keyword-top1 → 95.0% subcat**; gate runs no-think. HOLD: the 649 is a confidence-selected review subset (378/725 bypassed, unlabeled) → absolute levels unmeasurable; "no unattended auto-insert" + WS-D scoping HELD.
- **T-31 NN PRECONDITION (session 53) — GO for 5.R2.** hit@5 62–65% multilingual on the 160 misses; arctic-embed2 the practical pick. `FINDINGS-nn.md`. PR #44 merged.
- **5.R2 BUILT + ADOPTED (session 54) — PR #45 MERGED (session 57).** Embed-on-miss cascade layer in `internal/classifier` (`embedding{,_retriever,_fallback}.go`): keyword miss → arctic-embed2 embed + cosine top-5 few-shot injection; per-model JSONL disk cache (gitignored), lazy BLOCKING sync.Once reconcile, degrade-to-nil on any failure. `Config.EmbedModel`/`NoEmbedRetrieval` — zero values = ON (no command changes). **Replay A/B (miss stratum, n=80 unique): full-path 18.8%→52.5% no-think (+33.8pp, 29 fixed/2 broke); think-on 55.0% (+2.5pp churn at 5.5× latency → no-think right for the miss path, feeds T-24).** Report `FINDINGS-5r2.md`; plan `.claude/plans/5r2-embedding-retrieval.md` (check-later: CL1 trigger widening, CL2 pre-warm). ~35% permanent-review residue remains inside the novel pool (T-31 prediction held). **Session 55: 5.R2 files refactored into named step helpers (method-extraction pattern) — pure refactor, pushed to PR #45.**
- **Method-extraction convention ADOPTED module-wide (session 55):** multi-step function bodies read as named delegated steps ≤~15 lines; step-comments promote to helper doc comments; action-verb names for step helpers, no unwarranted abbreviations. Recorded in `expense-reporter/.memories/KNOWLEDGE.md` "Method-Extraction Convention" + QUICK Key Rule.
- **T-32 SHIPPED (session 57, PR #46 MERGED session 58).** `IsAutoInsertable` gates on keyword AGREEMENT (`matched ∧ ¬ambiguous ∧ top_score>=1.0 ∧ predicted==keyword_top1 ∧ ¬excluded`); confidence dropped (measured dead). Signal = pure `MatchStrength(item, keywords)` in `examples.go`. `--threshold` deprecated. Replay-validated: shipped `top_score` == harness on all 649 rows → 94.9% agreement subcat (30.5% coverage). **CAVEAT: same-sample RELATIVE; 649 confidence-selected → ABSOLUTE precision UNMEASURED → WS-D silent-insert HELD.** Report `.claude/t32-agreement-gate-report.md`.
- **T2 HARNESS EXTRACTION — CLOSED (Sessions A s58, B s59, closeout s60).** `github.com/leandror172/acceptance-harness` published (public, MIT); career-search repointed (PR #7 merged); expenses migrated (PR #48 merged) — `test/harness/` gone, domain split into `test/{expect,domain,extern}/`, scenarios import module `verify.*` + local `expect.*`. **v1.0.0 TAGGED (s60)** at `11beb2a`; expenses bumped v0.1.1→v1.0.0 (master). Engine changes = PR upstream + version bump here.
- **T-20 MERGED (s61).** `batch-auto --resume` + always-on duplicate-append warning on master (PR #49).
- **SESSION 61 — T-35 + T-36 SHIPPED (PRs #50, #51 open).** **T-35 join-id divergence:** the two JSONL logs join ONLY on `GenerateID` = sha256(item|date|value)[:12], which hashes the date **as raw bytes** — and `auto`/`batch-auto`/**`apply`** each fed one log a raw `DD/MM` and the other a normalized `DD/MM/YYYY`, so one expense sat under two ids and every cross-file join silently broke (incl. `correct`, which looked up the normalized id and missed every auto-logged row). `add`/`correct` were already correct. Fix = canonicalize ONCE at each boundary (`entriesWithCanonicalDates` in apply; one re-derivation in auto/batch-auto); `appendNewRows` moved `ParseDateWithYear`→`ParseDateFlexible` because the former REJECTS full dates. Guard = `expect.JoinIDMatchesAcrossLogs` + fixtures that MUST stay short-`DD/MM` (a full date makes raw==normalized and hides the bug). Both tests verified RED first. Docs: `.claude/t35-date-year-semantics.md` [ref:date-year-semantics] + `.claude/t35-implementation-report.md`. **T-36:** post-T-32 stale test names, ghost `auto-basic/input.csv`, README rollover/confidence lines, + isolated 15-file gofmt sweep (verified comment/whitespace-only by token diff). **Suite green on both branches (840s / 1989s, 56 / 54 tests).**
- **Next: merge #50 then #51.** Product: **WS-D (T-09)** — HELD, gate-to-review not silent insert; then WS-E. New backlog: T-37 (dead `date_year` config), T-38 (`ParseDate` hardcodes 2025, live under plain `batch`), T-39 (suite 1989s > run-acceptance.sh's 1800s ceiling), T-40 (batch-auto join-id coverage gap). Also: T-24/T-25/T-27/T-28; rename `Apoia-se` in workbook Referência; promote all-years log.
- **Cross-repo:** LLM infra at `/mnt/i/workspaces/llm/` — personas, MCP server, platform docs. New Go persona `my-go-q3-14b` (qwen3:14b) — Go verdicts 0/1/0/1+1, below qcoder's 2/2/1/1; keep qcoder tier list, q3-14b useful when VRAM-sharing with another qwen3:14b session.
<!-- /ref:current-status -->

---

<!-- ref:resume-steps -->
## Quick Resume

Run `.claude/tools/resume.sh` for a compact session-start summary.

Or manually:
1. `ref-lookup.sh current-status` — current layer, next task, branch state
2. Tail of `.claude/session-log.md` — "Next" pointer from most recent session
3. `git log --oneline -3` — recent commits
4. `.claude/index.md` — find any specific file/topic on demand
<!-- /ref:resume-steps -->

---

<!-- ref:active-decisions -->
## Active Decisions
### batch-auto --resume (T-20, session 60, PR #49)
- **Option B locked (user):** explicit `--resume` flag + ALWAYS-ON stderr duplicate-append warning; the expense log NEVER dedups silently — identical `(item,date,value)` triples can be legitimate distinct expenses, so recovery from partial failure is an explicit, observable operation.
- **Ledger = one `map[id]int`** (`feedback.LoadExpenseIDCounts`; counts, not a set), consumed classify-phase full-skips first, then append-phase appends/warnings; partial matches consume nothing. `appender.PredictEntryIDs` shares `expandEntries` with `ExpandAndAppend` (advisor BLOCKER: prediction from raw strings would mismatch every DD/MM + installment id). Partially-logged installment series route to REVIEW, never auto-completed (advisor: a divergent run-2 reclassification would silently split a series across categories). DD/MM year-inference caveat documented in flag help (December batches → `DD/MM/YYYY`).
- **Advisor substitute (Fable):** no `advisor()` tool in Fable sessions — spawn an Opus subagent at xhigh effort instead (Workflow harness or the new `.claude/agents/impl-opus-xhigh.md` for implementation). Review: `.claude/advisor-t20-resume-dedup.md`.
### Agreement Gate — SHIPPED (T-32, session 57, PR #46)
- **`IsAutoInsertable` gates on keyword AGREEMENT, not confidence.** `matched ∧ ¬ambiguous ∧ top_score>=1.0 ∧ predicted_subcat==keyword_top1 ∧ ¬excluded`. Confidence dropped (649-replay proved it dead: 0.85–0.95 band a coin flip). Signal = PURE `MatchStrength(item, keywords) MatchSignal` in `examples.go` computed at the gate sites — NOT the planned `Classify` return-shape change (advisor: model/pool-independent; a pure fn avoids the case-(b) embedding-fallback contamination where a high-spec keyword with no pool examples would be mis-tagged a miss). Deterministic tiebreak in `sortSubcategoriesByScore`; a top-score tie → `Ambiguous` → review. `--threshold` deprecated (cobra `MarkDeprecated`), not removed.
- **Replay-validated (shipped==measured):** `gate_validation_test.go` (`//go:build replay`) recomputes the shipped gate over the frozen 649 `model.jsonl` — shipped `top_score` == harness `top_score` on ALL 649 rows (0 mismatch), reproducing FINDINGS 34.2%/86.9% band + **94.9% agreement subcat / 30.5% coverage**. **CAVEAT:** same-sample RELATIVE only; the 649 is a confidence-selected subset → ABSOLUTE production precision UNMEASURED → **WS-D unattended silent-insert stays HELD** (T-32 unblocks scoping, not silent insert).
- **Latent verifier false-pass fixed** — `test/verify/feedback.go` `FeedbackMatchesExpected` early-returned on an empty log (batch-auto's `O_CREATE` preflight always creates one), silently passing vs a non-empty expected. Removed → empty actual now fails the count check. The gate exposed it on the installment tests.
- **Fixture item swap** — `Uber Centro` (spec 0.8, ambiguous Viagens/Uber-Taxi) FAILS the gate; auto-append fixtures (feedback/installments/rollover/typed + auto-basic) now use `Posto Ipiranga` (spec 1.0 → Combustível, model agrees, stays Variáveis). `add` tests keep Uber Centro (no gate). **Session 60 note:** the swap now also covers `auto_log_append_test.go`'s literal `When:` args (T-34), and the T-20 one-of-two-dups resume test uses `Uber Centro` deliberately (gate-failing → deterministic review routing).
### Gate Route & Leaf-First (T-23 → T-30 → route reframe, sessions 49–51)
- **Ollama 0.17.5 reports PRE-grammar-mask logprobs** — the raw model distribution, NOT renormalized over grammar-legal tokens (proven: a schema-forced key reports the chosen token at p≈0 while the model's wanted token shows 1.0; legal tokens are often outside `top_logprobs`). So you cannot read "P over legal paths" directly. Overturns the initial "grammar renormalizes → token-dist == taxonomy-dist" theory. Report: `.claude/t23-logprob-confidence-probe.md`.
- **Leaf-first split from T-23** — leaf-first classification (enum = 104 leaf names, derive type/cat via `ResolveLeaf`) was a STANDALONE accuracy change judged on accuracy first (attacks T-14's dominant "right sheet, wrong leaf"), AND the precondition that makes any token-confidence signal legible (type-first makes the margin unreadable; leaf-first: Uber leaf p=0.986). Plan: `.claude/plans/leaf-first-classification.md`.
- **D1 (schema) = second `type` field, leaf THEN type** — order load-bearing; leaf first, type second as a recurrence judgment; `ResolveLeaf` consults type only for the 5 cross-type collisions, cross-checks the 99 uniques. **GBNF confirmed the declaration order pins generation order (session 50).** D2=tree, D3=benchmark-only, D4 RESOLVED below.
- **T-30 A/B OUTCOME (session 50) = D4 HOLD (do not adopt for accuracy).** 4 cells (few-shot OFF/ON × no-think/think-on, n=300 paired): full-path Δ decays +12.7 → +8.7 → +3.3 → **+1.7 (production, p=0.46 NOT sig)**. Few-shot retrieval already fixes the same wrong-leaf errors → the enum gain is real in isolation but sub-additive with few-shot, washing to noise in production. Not measurably worse (all prod deltas mildly +), but underpowered → "too small to justify a rewrite," not a proven null. Type Δ survives longest (sig 3/4) but not in prod. `type_crosscheck` NOT a usable calibration signal (sign-flips). T-22 descriptions held absent in both arms (strengthens HOLD). Report `.claude/t14-benchmark-report.md` "T-30". Validated session 51 (prod cell + novel stratum +2.6 n.s. recomputed).
- **Leaf-first is PARKED, revivable ONLY for the logprob route.** Precondition proven for logprob-margin/answer-perplexity (M2) only; for ensemble-agreement (M1) it is untested/possibly unnecessary (leaves extractable from K sampled full-paths). Revive only if the T-23 novel-pool study shows logprob is the one signal that works there. Couples to T-27 (few-shot already recovers most wrong-leaf errors → re-justify or drop T-27).
<!-- /ref:active-decisions -->

<!-- ref:session-reading-guide -->

| Task | Read first | Notes |
|------|-----------|-------|
| **Merge PR #50 (T-35) then #51 (T-36) — first thing next session** | PR #50; PR #51; `.claude/t35-implementation-report.md` | **Order matters:** T-36 renames tests in `auto_log_append_test.go`, the file T-35 appends a test to. Different regions so git should merge, but #50 first is lower-risk. |
| **T-39 — acceptance suite exceeds its own timeout (arguably urgent)** | `run-acceptance.sh:49,51`; session-61 log gotchas | Suite ran **1989s**; the script hardcodes `-timeout 1800s`, so the sanctioned invocation would now TIME OUT (my green came from calling `go test` directly with 2400s). Runtime varies 840s–1989s on one machine (Ollama load/contention). Decide: raise the ceiling, or split the Ollama-gated tests into their own group. T-08 is the adjacent single-test flake. |
| **Date/year semantics — read BEFORE touching any date code** | `.claude/t35-date-year-semantics.md` [ref:date-year-semantics] | Six parsers with six different missing-year rules; two were reachable from one command. Also lists the year behavior that is CORRECT and must not be "fixed" (installments crossing years, installment ids legitimately differing between logs, review's year-less session id, taxonomy's `year=0` sentinel). |
| **T-37 / T-38 — the two date findings** | `t35-date-year-semantics.md` §5 | T-37: `config.json` `date_year` is DEAD — one reference in the codebase (the declaration), read by nothing, while KNOWLEDGE.md says rollover requires updating it. T-38: `utils.ParseDate` hardcodes 2025 (`date.go:41`), live via plain `batch` → `parser.ParseExpenseString` → `models.NewExpense`. T-38 touches workbook data. |
| **WS-D — retire bare-name fallback (T-09) — NEXT product step** | `.claude/t32-agreement-gate-report.md` §7; `[ref:taxonomy-identity-key]`; `internal/taxonomy/loader.go` `scanEntries`; stderr fallback count | T-32 agreement gate LANDED, so novel rows are protected. But absolute precision UNMEASURED (649 = confidence-selected) → design **gate-to-review**, NOT silent insert. Session-52 replay confirms recurrence-first (leaky 63% vs clean 44%). |
| **Adding any join-id / date fixture** | `test/.memories/QUICK.md` Key Rules; `test/fixtures/apply-join-id/README.md` | Join-id fixtures MUST use short `DD/MM` — the OPPOSITE of the standing explicit-year rule. A full date makes raw==normalized and **silently disables** the test instead of breaking it. Safe only because these assert id EQUALITY, never a literal date. |
| **5.R2 follow-ups (CL1 trigger widening / CL2 pre-warm / alternate embedder)** | `.claude/scratch/replay-649/FINDINGS-5r2.md`; `.claude/plans/5r2-embedding-retrieval.md` § Check-later | ADOPTED s54 (miss full-path 18.8→52.5 no-think). CL1: widen the fallback to weak/ambiguous keyword matches (top score < 0.7). CL2: pre-warm if the burst annoys. qwen3-embedding:8b (+2.5pp hit@5) unbenchmarked in-model. |
| **T-23 — gate route (measured; decision still deferred)** | `.claude/plans/t23-calibration-benchmark.md` "Route decision"; `.claude/t23-strategic-implications.md`; `.claude/scratch/replay-649/FINDINGS-model.md` | Route = external validators over model-introspection (s51). Measured s52: E1 specificity works; agreement is the strongest gate; confidence dead. 5.R2 BUILT; T-32 gate wiring feeds the final call. |
| Delete dead insert code (WS-E) | plan "WS-E"; `internal/workflow`, `internal/excel` write side, `internal/batch` insert path | Only after WS-D. NARROW scope: delete only deprecated `InsertBatchExpensesFromClassified` + its test; `InsertBatchExpenses`/rollover/excel stay LIVE under plain `batch`. |
| **T-24 — --think default decision** | `.claude/t14-benchmark-report.md` "T-30"; `.claude/scratch/replay-649/FINDINGS-{model,5r2}.md` | Gate band: think-on −2.3pp. On the miss path with 5.R2 ON, think adds only +2.5pp churn at 5.5× latency. Both strata favor no-think. Couples to T-39 (suite runtime). |
| **T-27 — leaf-level descriptions (RE-JUSTIFY or drop)** | T-22 addendum; T-30 report; `descriptions.go` | Undercut by T-30 (few-shot already recovers most wrong-leaf). Re-justify on few-shot-ON evidence or drop. |
| **T-28 — alternative classification strategies** | `internal/classifier/classifier.go` (prompt + GBNF); T-13 report | Two-stage / two-pass-grammar (unconstrained-first) vs one-shot enum. Measure on the T-14 sample. |
| Reviewed-installment under-recording (T-21) | `internal/review/queue.go` `ReadQueue`; review.html export JS; `internal/apply/types.go` | Count discarded at `ReadQueue:63`, absent from `reviewed.json`. Thread count/rawValue through. |
| q35 grammar bug + persona cleanup (T-25) | `.claude/t14-benchmark-report.md` "Model matrix outcome" | Reproduce minimal (qwen3.5 + `think:false` + `format`), check newer Ollama, file upstream. |
| Taxonomy `Apoia-se` durability (session 49) | `.claude/plans/leaf-first-classification.md` § Prerequisite; workbook Referência sheet | Rename `Apoia-se 4i20`→`Apoia-se` in the Referência sheet (export source) or re-export wipes it. |
| Promote merged log to canonical (deferred) | `.claude/scratch/merge_year_logs.py`; gitignored `expenses_log-allyears.jsonl` | User decides when to swap canonical + delete per-year files. |
<!-- /ref:session-reading-guide -->
