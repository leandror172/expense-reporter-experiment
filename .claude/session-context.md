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
- **T-35 + T-36 MERGED (session 62 start, PRs #50/#51).** T-35 join-id divergence: dates canonicalized ONCE at each boundary so both JSONL logs share one `GenerateID`; guard = `expect.JoinIDMatchesAcrossLogs` + deliberately short-`DD/MM` fixtures. T-36 staleness/gofmt sweep. Docs: `.claude/t35-date-year-semantics.md` [ref:date-year-semantics] + `.claude/t35-implementation-report.md`.
- **SESSION 62 — T-39 + T-24 SHIPPED (PR #52 open) + STRATEGIC REFRAME + PARSE-BOUNDARY PLAN (T-41).** **T-39:** suite split via `testing.Short()` in `extern.RequireOllama` (per-test seam); `run-acceptance.sh` defaults to the deterministic group (~8s, 30 pass/26 skip, no Ollama, 600s), `-full` = whole suite (3600s ceiling; **measured 128s at the new think-off default — was 840–1989s**). **T-24:** `--think=false` default on all 3 commands (gate band −2.3pp WITH think; 5.R2 miss +2.5pp at 5.5× latency; sentinel covered by the agreement gate). Full suite 56/56 green under the new default. **Vision re-read → project reframed PRE-FIRST-USE; organizing milestone = first real monthly close on 2026 data (T-42).** **Parse-boundary plan drafted (`.claude/plans/parse-boundary.md`, T-41):** structured input, parse-once — single-string input is the common ancestor of the date/data debt. LOCKED: year precedence `string > --year > config date_year > most-recent-non-future` (grace window TBD); T-21 UX = 1 row + ×N badge; `--year` uniform; **T-38 RESOLVED = delete `utils.ParseDate` with plain `batch` under WS-E (widens WS-E scope), don't repair.** Memories/README staleness sweep on the branch (dead `date_year` doc corrected; root QUICK compressed, Milestone Log s60–62 added).
- **Next: merge PR #52 → T-41 parse-boundary design session (plan §10: package home, migration order, currency ownership, MCP timing) → T-21 threading → T-42 monthly close.** WS-D (HELD: gate-to-review, not silent insert) → WS-E after (now includes plain `batch` + `utils.ParseDate` retirement). Standing queue: T-25/T-27/T-28/T-29, Apoia-se rename, promote all-years log.
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
### Parse boundary + suite/think defaults (session 62)
- **Parse-once architecture (user):** chat-era input starts with a parse step producing structured fields; downstream tools consume fields, never raw strings. The single-string format is the measured common ancestor of the date/data debt (T-35 class, T-18, T-21, six parsers, `GenerateID` format-sensitivity). Plan: `.claude/plans/parse-boundary.md` (T-41); boundary stays deterministic Go — free-form text is Layer 6's job.
- **Year precedence LOCKED:** explicit year in string > `--year` arg > config `date_year` (T-37 wired here) > most-recent-non-future occurrence (bare `DD/MM` landing in the future resolves to LAST year; small grace window for post-dated card transactions TBD in design session). `--year` added uniformly to `auto`/`batch-auto`/`add`.
- **T-21 UX:** review UI shows 1 row + ×N badge; `apply` expands to N log entries (expansion is a write-time concern, mirrors batch-auto's auto path).
- **T-38 RESOLVED:** plain `batch` is pre-pivot (writes the workbook directly; no vision path uses it) — `utils.ParseDate` (hardcoded 2025) is deleted WITH it under WS-E, not repaired. WS-E scope widened accordingly.
- **Suite modes (T-39):** `run-acceptance.sh` default = deterministic (`-short`, no Ollama, 600s); `-full` = whole suite (3600s). Seam = `testing.Short()` in `extern.RequireOllama`. Deterministic tests inside Ollama-heavy files still run (per-test seam).
- **Think default (T-24):** `--think=false` on all three commands, uniform (T-16 parity lesson). Rationale: gate band −2.3pp WITH think; 5.R2 miss path +2.5pp at 5.5× latency; OOD sentinel loss covered by the T-32 agreement gate routing disagreements to review. `--think` opts back in. Latency claims in docs should name their think mode.
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
<!-- /ref:active-decisions -->

<!-- ref:session-reading-guide -->
| Task | Read first | Notes |
|------|-----------|-------|
| **T-41 parse boundary — NEXT: design session** | `.claude/plans/parse-boundary.md`; `.claude/t35-date-year-semantics.md` [ref:date-year-semantics] | LOCKED (s62): year precedence `string > --year > config date_year > most-recent-non-future` (grace window TBD); T-21 UX = 1 row + ×N badge; `--year` uniform; T-38 out of scope (dies with plain `batch`, WS-E). OPEN (§10): package home/struct shape, migration order (candidate: add/correct → auto → batch-auto → apply), currency ownership, MCP `parse_expense` timing. Deterministic suite now ~8s (`./run-acceptance.sh`) — iterate freely; `-full` ~2 min. |
| **T-42 — first real monthly close on 2026 data (organizing milestone)** | `docs/expense-classifier-vision.md`; `.claude/t23-strategic-implications.md`; parse-boundary plan §9 | Project is PRE-FIRST-USE (s62 reframe). After T-41 + T-21: run a real bank CSV through `batch-auto → review → apply → generate-workbook --year 2026`; everything that breaks is the real backlog. Pre-work also: T-03 rollover check, promote all-years log. |
| **Date/year semantics — read BEFORE touching any date code** | `.claude/t35-date-year-semantics.md` [ref:date-year-semantics] | Six parsers with six different missing-year rules; two were reachable from one command. Also lists the year behavior that is CORRECT and must not be "fixed" (installments crossing years, installment ids legitimately differing between logs, review's year-less session id, taxonomy's `year=0` sentinel). |
| **WS-D — retire bare-name fallback (T-09)** | `.claude/t32-agreement-gate-report.md` §7; `[ref:taxonomy-identity-key]`; `internal/taxonomy/loader.go` `scanEntries`; stderr fallback count | T-32 agreement gate LANDED, so novel rows are protected. But absolute precision UNMEASURED (649 = confidence-selected) → design **gate-to-review**, NOT silent insert. Session-52 replay confirms recurrence-first (leaky 63% vs clean 44%). |
| **Adding any join-id / date fixture** | `test/.memories/QUICK.md` Key Rules; `test/fixtures/apply-join-id/README.md` | Join-id fixtures MUST use short `DD/MM` — the OPPOSITE of the standing explicit-year rule. A full date makes raw==normalized and **silently disables** the test instead of breaking it. Safe only because these assert id EQUALITY, never a literal date. |
| **5.R2 follow-ups (CL1 trigger widening / CL2 pre-warm / alternate embedder)** | `.claude/scratch/replay-649/FINDINGS-5r2.md`; `.claude/plans/5r2-embedding-retrieval.md` § Check-later | ADOPTED s54 (miss full-path 18.8→52.5 no-think). CL1: widen the fallback to weak/ambiguous keyword matches (top score < 0.7). CL2: pre-warm if the burst annoys. qwen3-embedding:8b (+2.5pp hit@5) unbenchmarked in-model. |
| **T-23 — gate route (measured; decision still deferred)** | `.claude/plans/t23-calibration-benchmark.md` "Route decision"; `.claude/t23-strategic-implications.md`; `.claude/scratch/replay-649/FINDINGS-model.md` | Route = external validators over model-introspection (s51). Measured s52: E1 specificity works; agreement is the strongest gate; confidence dead. 5.R2 BUILT; T-32 gate wiring feeds the final call. |
| Delete dead insert code (WS-E) | plan "WS-E"; `internal/workflow`, `internal/excel` write side, `internal/batch` insert path | Only after WS-D. **Scope WIDENED s62 (T-38 resolution): plain `batch` is a pre-pivot artifact — its insert chain AND `utils.ParseDate` (hardcoded 2025) go with it.** Previous narrow scope (only `InsertBatchExpensesFromClassified`) superseded. |
| **T-27 — leaf-level descriptions (RE-JUSTIFY or drop)** | T-22 addendum; T-30 report; `descriptions.go` | Undercut by T-30 (few-shot already recovers most wrong-leaf). Re-justify on few-shot-ON evidence or drop. |
| **T-28 — alternative classification strategies** | `internal/classifier/classifier.go` (prompt + GBNF); T-13 report | Two-stage / two-pass-grammar (unconstrained-first) vs one-shot enum. Measure on the T-14 sample. |
| Reviewed-installment under-recording (T-21) | `internal/review/queue.go` `ReadQueue`; review.html export JS; `internal/apply/types.go` | Count discarded at `ReadQueue:63`, absent from `reviewed.json`. **Now the FIRST CONSUMER slice of T-41** — `Installments`/`RawValue` become boundary fields; UX locked s62: 1 row + ×N badge, `apply` expands. |
| q35 grammar bug + persona cleanup (T-25) | `.claude/t14-benchmark-report.md` "Model matrix outcome" | Reproduce minimal (qwen3.5 + `think:false` + `format`), check newer Ollama, file upstream. |
| Taxonomy `Apoia-se` durability (session 49) | `.claude/plans/leaf-first-classification.md` § Prerequisite; workbook Referência sheet | Rename `Apoia-se 4i20`→`Apoia-se` in the Referência sheet (export source) or re-export wipes it. |
| Promote merged log to canonical (deferred) | `.claude/scratch/merge_year_logs.py`; gitignored `expenses_log-allyears.jsonl` | User decides when to swap canonical + delete per-year files. Feeds T-42. |
<!-- /ref:session-reading-guide -->
