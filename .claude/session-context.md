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
- **SESSION 62 — T-39 + T-24 SHIPPED (PR #52 merged s63) + STRATEGIC REFRAME + PARSE-BOUNDARY PLAN (T-41).** **T-39:** suite split via `testing.Short()` in `extern.RequireOllama` (per-test seam); `run-acceptance.sh` defaults to the deterministic group (~8s, 30 pass/26 skip, no Ollama, 600s), `-full` = whole suite (3600s ceiling; **measured 128s at the new think-off default — was 840–1989s**). **T-24:** `--think=false` default on all 3 commands (gate band −2.3pp WITH think; 5.R2 miss +2.5pp at 5.5× latency; sentinel covered by the agreement gate). Full suite 56/56 green under the new default. **Vision re-read → project reframed PRE-FIRST-USE; organizing milestone = first real monthly close on 2026 data (T-42).** **Parse-boundary plan drafted (`.claude/plans/parse-boundary.md`, T-41):** structured input, parse-once — single-string input is the common ancestor of the date/data debt. Memories/README staleness sweep on the branch (dead `date_year` doc corrected; root QUICK compressed, Milestone Log s60–62 added).
- **SESSION 63 — T-41 DESIGN LOCKED + SLICE 1 SHIPPED (PR #54 open).** Design session closed all plan-§10 opens: home = new `internal/parse`; struct = SINGLE stored `time.Time` + `DateString()` method (sole identity-bytes site — an inconsistent date pair is unrepresentable); field-wise core (`Fields`) + semicolon wrapper (`ExpenseString`) returning subcategory alongside; grace window **0**; boundary owns ALL input normalization; internal-first (MCP `parse_expense` waits for Layer 6); order add/correct → auto → batch-auto → apply, one PR per slice. **Slice 1:** `add`/`correct` repointed (`parseExpenseForFeedback` deleted), `--year` on both (`correct` included — T-16 parity), `date_year` LIVE as rung 3 (T-37 partially closed), rung 4 = most-recent-non-future (bare strictly-future date → LAST year — deliberate change), BR thousands (`1.234,56`) parse at the boundary (never worked anywhere before). Tests: 4 year-precedence acceptance + `correct --year` join test (seed id `26138364fcc9`) + thousands CLI pin + 10 unit funcs (injected clock pins both sides of the grace-0 edge). Full suite green (`-full` 114s). Pattern-conformance pass vs llm-repo conventions (single-home validation; `parseOptions` extraction deferred to slice 2 = third caller). **Cross-repo find: Fable transcripts store NO assistant text → verdict-capture hook inert in these sessions; 4 verdicts appended directly to `calls.jsonl`; llm T-109 (uncommitted there) + memory `feedback_verdict_transcript_gap`.**
- **Next: merge PR #54 → slice 2 (`auto` + cmd-level `parseOptions` extraction) → slice 3 (`batch-auto` parse-once-per-row + T-40 boundary test) → slice 4 (`apply`, retires `canonicalDate`) → T-21 threading → T-42 monthly close.** T-43 (correct fixtures explicit year) = 2-line quickie. WS-D (HELD: gate-to-review, not silent insert) → WS-E after (includes plain `batch` + `utils.ParseDate` retirement). Standing queue: T-25/T-27/T-28/T-29, Apoia-se rename, promote all-years log.
- **Cross-repo:** LLM infra at `/mnt/i/workspaces/llm/` — personas, MCP server, platform docs. New Go persona `my-go-q3-14b` (qwen3:14b) — Go verdicts 0/1/0/1+1, below qcoder's 2/2/1/1; keep qcoder tier list, q3-14b useful when VRAM-sharing with another qwen3:14b session. **T-109 noted there (uncommitted): 5 verdict/ollama-bridge findings from this repo's s63.**
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
### Parse boundary — design locks (session 63, plan §10 FINAL)
- **Struct shape (user proposal):** single stored `Date time.Time` + `DateString()` method — an inconsistent date pair is unrepresentable; formatting cannot fail; `DateString()` is the SOLE identity-bytes site (GenerateID + both logs). Supersedes the "both fields" lean; do not add a stored string date.
- **Home:** new `internal/parse` (NOT `internal/parser` — dying pre-pivot code, wholesale-deletable under WS-E; NOT `models` — leaf package, `NewExpense` also dies).
- **API:** field-wise core `Fields(item, dateStr, valueStr, Options)` + semicolon wrapper `ExpenseString` returning subcategory ALONGSIDE (classification ≠ parsing; the semicolon form exists only at add/correct).
- **Grace window = 0 (user):** never enters future-dated transactions; bare date strictly after Now → LAST year; same-day is not future (candidate is midnight UTC). Genuinely future entries carry an explicit year (rung 1 wins).
- **`correct` gets `--year` too** (beyond the s62 list) — T-16 parity; its GenerateID lookup must resolve years identically to what `add` wrote.
- **`parseOptions(yearFlag, cfg)` extraction deferred to slice 2** — no seam below 3 callers (extract-keep-divergence rule 3); cmd-level so `parse` never imports `config`.
- **BR thousands rule:** token with BOTH '.' and ',' → dots stripped; dot-only keeps legacy decimal-dot meaning. Boundary owns ALL input normalization (Q3).
- **Internal-first (Q4):** no MCP `parse_expense` until Layer 6; existing MCP tools benefit via the commands they shell out to.
### Parse boundary + suite/think defaults (session 62)
- **Parse-once architecture (user):** chat-era input starts with a parse step producing structured fields; downstream tools consume fields, never raw strings. The single-string format is the measured common ancestor of the date/data debt (T-35 class, T-18, T-21, six parsers, `GenerateID` format-sensitivity). Plan: `.claude/plans/parse-boundary.md` (T-41); boundary stays deterministic Go — free-form text is Layer 6's job.
- **Year precedence LOCKED:** explicit year in string > `--year` arg > config `date_year` (T-37 wired here) > most-recent-non-future (grace window 0, locked s63). `--year` added uniformly to `auto`/`batch-auto`/`add` (+`correct`, s63).
- **T-21 UX:** review UI shows 1 row + ×N badge; `apply` expands to N log entries (expansion is a write-time concern, mirrors batch-auto's auto path).
- **T-38 RESOLVED:** plain `batch` is pre-pivot (writes the workbook directly; no vision path uses it) — `utils.ParseDate` (hardcoded 2025) is deleted WITH it under WS-E, not repaired. WS-E scope widened accordingly.
- **Suite modes (T-39):** `run-acceptance.sh` default = deterministic (`-short`, no Ollama, 600s); `-full` = whole suite (3600s). Seam = `testing.Short()` in `extern.RequireOllama`. Deterministic tests inside Ollama-heavy files still run (per-test seam).
- **Think default (T-24):** `--think=false` on all three commands, uniform (T-16 parity lesson). Rationale: gate band −2.3pp WITH think; 5.R2 miss path +2.5pp at 5.5× latency; OOD sentinel loss covered by the T-32 agreement gate routing disagreements to review. `--think` opts back in. Latency claims in docs should name their think mode.
### batch-auto --resume (T-20, session 60, PR #49)
- **Option B locked (user):** explicit `--resume` flag + ALWAYS-ON stderr duplicate-append warning; the expense log NEVER dedups silently — identical `(item,date,value)` triples can be legitimate distinct expenses, so recovery from partial failure is an explicit, observable operation.
- **Ledger = one `map[id]int`** (`feedback.LoadExpenseIDCounts`; counts, not a set), consumed classify-phase full-skips first, then append-phase appends/warnings; partial matches consume nothing. `appender.PredictEntryIDs` shares `expandEntries` with `ExpandAndAppend` (advisor BLOCKER: prediction from raw strings would mismatch every DD/MM + installment id). Partially-logged installment series route to REVIEW, never auto-completed (advisor: a divergent run-2 reclassification would silently split a series across categories). DD/MM year-inference caveat documented in flag help (December batches → `DD/MM/YYYY`).
- **Advisor substitute (Fable):** no `advisor()` tool in Fable sessions — spawn an Opus subagent at xhigh effort instead (Workflow harness or the new `.claude/agents/impl-opus-xhigh.md` for implementation). Review: `.claude/advisor-t20-resume-dedup.md`. **Fable verdict caveat (s63):** [VERDICT] blocks in message text are never captured in Fable sessions (transcript has no assistant text) — append records directly to `calls.jsonl` per memory `feedback_verdict_transcript_gap`.
<!-- /ref:active-decisions -->

<!-- ref:session-reading-guide -->
| Task | Read first | Notes |
|------|-----------|-------|
| **T-41 parse boundary — NEXT: slice 2 (`auto`)** | `.claude/plans/parse-boundary.md` §5 (per-slice mechanics) + §10 (decision record); `expense-reporter/internal/parse/.memories/QUICK.md` | Slice 1 on PR #54. Slice 2 = repoint auto.go:51/:58 to `parse.Fields` + extract cmd-level `parseOptions(yearFlag, cfg)` (third caller — deferred from slice 1 by the no-seam-below-3 rule; cmd-level, `parse` must NOT import `config`). Net: `expect.JoinIDMatchesAcrossLogs` + short-`DD/MM` fixtures. Slice 3 takes T-40 as a boundary unit test; slice 4 retires `canonicalDate` and precedes T-21. Deterministic suite ~8s; `-full` ~2 min. |
| **T-42 — first real monthly close on 2026 data (organizing milestone)** | `docs/expense-classifier-vision.md`; `.claude/t23-strategic-implications.md`; parse-boundary plan §9 | Project is PRE-FIRST-USE (s62 reframe). After T-41 + T-21: run a real bank CSV through `batch-auto → review → apply → generate-workbook --year 2026`; everything that breaks is the real backlog. Pre-work also: T-03 rollover check, promote all-years log. |
| **Date/year semantics — read BEFORE touching any date code** | `.claude/t35-date-year-semantics.md` [ref:date-year-semantics]; `internal/parse/.memories/QUICK.md` | The T-41 boundary now owns add/correct's year resolution (ladder + grace 0); auto/batch-auto/apply still use the six-parser world until their slices land. The survey also lists year behavior that is CORRECT and must not be "fixed" (installments crossing years, installment ids differing between logs, review's year-less session id, taxonomy's `year=0` sentinel). |
| **WS-D — retire bare-name fallback (T-09)** | `.claude/t32-agreement-gate-report.md` §7; `[ref:taxonomy-identity-key]`; `internal/taxonomy/loader.go` `scanEntries`; stderr fallback count | T-32 agreement gate LANDED, so novel rows are protected. But absolute precision UNMEASURED (649 = confidence-selected) → design **gate-to-review**, NOT silent insert. Session-52 replay confirms recurrence-first (leaky 63% vs clean 44%). |
| **Adding any join-id / date fixture** | `test/.memories/QUICK.md` Key Rules; `test/fixtures/apply-join-id/README.md` | Join-id fixtures MUST use short `DD/MM` — the OPPOSITE of the standing explicit-year rule. A full date makes raw==normalized and **silently disables** the test instead of breaking it. Safe only because these assert id EQUALITY, never a literal date. T-41 sharpens the explicit-year side: a bare date strictly in the future now resolves to LAST year for add/correct (rung 4, grace 0) — see T-43 for the 2027 time bomb in the correct fixtures. |
| **5.R2 follow-ups (CL1 trigger widening / CL2 pre-warm / alternate embedder)** | `.claude/scratch/replay-649/FINDINGS-5r2.md`; `.claude/plans/5r2-embedding-retrieval.md` § Check-later | ADOPTED s54 (miss full-path 18.8→52.5 no-think). CL1: widen the fallback to weak/ambiguous keyword matches (top score < 0.7). CL2: pre-warm if the burst annoys. qwen3-embedding:8b (+2.5pp hit@5) unbenchmarked in-model. |
| **T-23 — gate route (measured; decision still deferred)** | `.claude/plans/t23-calibration-benchmark.md` "Route decision"; `.claude/t23-strategic-implications.md`; `.claude/scratch/replay-649/FINDINGS-model.md` | Route = external validators over model-introspection (s51). Measured s52: E1 specificity works; agreement is the strongest gate; confidence dead. 5.R2 BUILT; T-32 gate wiring feeds the final call. |
| Delete dead insert code (WS-E) | plan "WS-E"; `internal/workflow`, `internal/excel` write side, `internal/batch` insert path | Only after WS-D. **Scope WIDENED s62 (T-38 resolution): plain `batch` is a pre-pivot artifact — its insert chain AND `utils.ParseDate` (hardcoded 2025) go with it.** `internal/parser` and `models.NewExpense` go in the same sweep (superseded by `internal/parse`); consider absorbing the utils date-parser split then too. |
| **T-27 — leaf-level descriptions (RE-JUSTIFY or drop)** | T-22 addendum; T-30 report; `descriptions.go` | Undercut by T-30 (few-shot already recovers most wrong-leaf). Re-justify on few-shot-ON evidence or drop. |
| **T-28 — alternative classification strategies** | `internal/classifier/classifier.go` (prompt + GBNF); T-13 report | Two-stage / two-pass-grammar (unconstrained-first) vs one-shot enum. Measure on the T-14 sample. |
| Reviewed-installment under-recording (T-21) | `internal/review/queue.go` `ReadQueue`; review.html export JS; `internal/apply/types.go` | Count discarded at `ReadQueue:63`, absent from `reviewed.json`. **First CONSUMER slice of T-41, right after slice 4** — `Installments`/`RawValue` are already boundary fields; UX locked s62: 1 row + ×N badge, `apply` expands. |
| q35 grammar bug + persona cleanup (T-25) | `.claude/t14-benchmark-report.md` "Model matrix outcome" | Reproduce minimal (qwen3.5 + `think:false` + `format`), check newer Ollama, file upstream. |
| Taxonomy `Apoia-se` durability (session 49) | `.claude/plans/leaf-first-classification.md` § Prerequisite; workbook Referência sheet | Rename `Apoia-se 4i20`→`Apoia-se` in the Referência sheet (export source) or re-export wipes it. |
| Promote merged log to canonical (deferred) | `.claude/scratch/merge_year_logs.py`; gitignored `expenses_log-allyears.jsonl` | User decides when to swap canonical + delete per-year files. Feeds T-42. |
<!-- /ref:session-reading-guide -->
