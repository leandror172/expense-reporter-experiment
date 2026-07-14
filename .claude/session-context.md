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
- **T2 HARNESS EXTRACTION PLANNED (session 56):** full plan at `.claude/plans/harness-extraction-plan.md` + acceptance-test comparison companion. Decisions locked: public `github.com/leandror172/acceptance-harness` (MIT), lean core v1 seeded from career-search's de-domained copy (its TC-EXT is blocked on us), LTG = design influence only, split sequencing (Session A repo+career-search; Session B expenses migration). llm repo surveyed → not a consumer.
- **T-32 SHIPPED (session 57, PR #46 MERGED session 58).** `IsAutoInsertable` now gates on keyword AGREEMENT (`matched ∧ ¬ambiguous ∧ top_score>=1.0 ∧ predicted==keyword_top1 ∧ ¬excluded`); confidence dropped (measured dead). Signal = pure `MatchStrength(item, keywords)` in `examples.go` (NOT a `Classify` return-shape change — advisor: model/pool-independent, avoids embedding-fallback contamination). Deterministic tiebreak; `--threshold` deprecated. Replay-validated: shipped `top_score` == harness on all 649 rows → FINDINGS 34.2/86.9 band + **94.9% agreement subcat** (30.5% coverage). **CAVEAT: same-sample RELATIVE (shipped==measured); 649 is a confidence-selected subset → ABSOLUTE production precision UNMEASURED → WS-D silent-insert stays HELD.** Also fixed a latent verifier false-pass (empty-log early-return; batch-auto `O_CREATE` preflight) + swapped gate-failing `Uber Centro`→`Posto Ipiranga` in auto-append fixtures. Runtime refs describing the confidence gate all corrected. Report `.claude/t32-agreement-gate-report.md`.
- **T2 HARNESS EXTRACTION SESSION A — DONE (session 58).** `github.com/leandror172/acceptance-harness` PUBLISHED (public, MIT, **v0.1.1**, go 1.25): engine + generic verify (cli/json/file) seeded from career-search's de-domained copy; build tags stripped (consumer policy); retention flags → `HARNESS_KEEP_*` env vars + opt-in `RegisterFlags()`; methodology docs ported (`docs/{PATTERNS,ADOPTION,LTG-PATH}.md`); 30 lib unit tests (found + fixed latent `Run()` bug — `t.Name()` in MkdirTemp pattern crashes under subtests, was in BOTH consumers' copies); `.memories/` + `CLAUDE-DRAFT.md` (adoption undecided). **career-search REPOINTED (TC-EXT closed, PR #7):** module owns the `verify.*` namespace (146 call sites untouched), domain verifiers → `test/tracker/` (44 renames/3 files), go 1.24.2→1.25.5, 65 tests green; migration report `career-search/.claude/plans/tracker-cli-report-tc-ext.md`. **Session B (expenses migration) NEXT** — plan §5; no v1.0.0 until it lands.
- **Next: Harness extraction Session B** (migrate `test/` to the module — plan §5; keep `ollama.go`+`comparator.go` in the domain layer); or **WS-D (T-09)** — HELD, design gate-to-review, not silent insert; or **T-20** dedup (independent). Open: merge career-search PR #7; CLAUDE-DRAFT adoption; rename `Apoia-se` in workbook Referência; promote all-years log; T-24/T-25/T-27/T-28.
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

### Agreement Gate — SHIPPED (T-32, session 57, PR #46)
- **`IsAutoInsertable` gates on keyword AGREEMENT, not confidence.** `matched ∧ ¬ambiguous ∧ top_score>=1.0 ∧ predicted_subcat==keyword_top1 ∧ ¬excluded`. Confidence dropped (649-replay proved it dead: 0.85–0.95 band a coin flip). Signal = PURE `MatchStrength(item, keywords) MatchSignal` in `examples.go` computed at the gate sites — NOT the planned `Classify` return-shape change (advisor: model/pool-independent; a pure fn avoids the case-(b) embedding-fallback contamination where a high-spec keyword with no pool examples would be mis-tagged a miss). Deterministic tiebreak in `sortSubcategoriesByScore`; a top-score tie → `Ambiguous` → review. `--threshold` deprecated (cobra `MarkDeprecated`), not removed.
- **Replay-validated (shipped==measured):** `gate_validation_test.go` (`//go:build replay`) recomputes the shipped gate over the frozen 649 `model.jsonl` — shipped `top_score` == harness `top_score` on ALL 649 rows (0 mismatch), reproducing FINDINGS 34.2%/86.9% band + **94.9% agreement subcat / 30.5% coverage**. **CAVEAT:** same-sample RELATIVE only; the 649 is a confidence-selected subset → ABSOLUTE production precision UNMEASURED → **WS-D unattended silent-insert stays HELD** (T-32 unblocks scoping, not silent insert).
- **Latent verifier false-pass fixed** — `test/verify/feedback.go` `FeedbackMatchesExpected` early-returned on an empty log (batch-auto's `O_CREATE` preflight always creates one), silently passing vs a non-empty expected. Removed → empty actual now fails the count check. The gate exposed it on the installment tests.
- **Fixture item swap** — `Uber Centro` (spec 0.8, ambiguous Viagens/Uber-Taxi) FAILS the gate; auto-append fixtures (feedback/installments/rollover/typed + auto-basic) now use `Posto Ipiranga` (spec 1.0 → Combustível, model agrees, stays Variáveis). `add` tests keep Uber Centro (no gate).

### Gate Route & Leaf-First (T-23 → T-30 → route reframe, sessions 49–51)
- **Ollama 0.17.5 reports PRE-grammar-mask logprobs** — the raw model distribution, NOT renormalized over grammar-legal tokens (proven: a schema-forced key reports the chosen token at p≈0 while the model's wanted token shows 1.0; legal tokens are often outside `top_logprobs`). So you cannot read "P over legal paths" directly. Overturns the initial "grammar renormalizes → token-dist == taxonomy-dist" theory. Report: `.claude/t23-logprob-confidence-probe.md`.
- **Leaf-first split from T-23** — leaf-first classification (enum = 104 leaf names, derive type/cat via `ResolveLeaf`) was a STANDALONE accuracy change judged on accuracy first (attacks T-14's dominant "right sheet, wrong leaf"), AND the precondition that makes any token-confidence signal legible (type-first makes the margin unreadable; leaf-first: Uber leaf p=0.986). Plan: `.claude/plans/leaf-first-classification.md`.
- **D1 (schema) = second `type` field, leaf THEN type** — order load-bearing; leaf first, type second as a recurrence judgment; `ResolveLeaf` consults type only for the 5 cross-type collisions, cross-checks the 99 uniques. **GBNF confirmed the declaration order pins generation order (session 50).** D2=tree, D3=benchmark-only, D4 RESOLVED below.
- **T-30 A/B OUTCOME (session 50) = D4 HOLD (do not adopt for accuracy).** 4 cells (few-shot OFF/ON × no-think/think-on, n=300 paired): full-path Δ decays +12.7 → +8.7 → +3.3 → **+1.7 (production, p=0.46 NOT sig)**. Few-shot retrieval already fixes the same wrong-leaf errors → the enum gain is real in isolation but sub-additive with few-shot, washing to noise in production. Not measurably worse (all prod deltas mildly +), but underpowered → "too small to justify a rewrite," not a proven null. Type Δ survives longest (sig 3/4) but not in prod. `type_crosscheck` NOT a usable calibration signal (sign-flips). T-22 descriptions held absent in both arms (strengthens HOLD). Report `.claude/t14-benchmark-report.md` "T-30". Validated session 51 (prod cell + novel stratum +2.6 n.s. recomputed).
- **Leaf-first is PARKED, revivable ONLY for the logprob route.** Precondition proven for logprob-margin/answer-perplexity (M2) only; for ensemble-agreement (M1) it is untested/possibly unnecessary (leaves extractable from K sampled full-paths). Revive only if the T-23 novel-pool study shows logprob is the one signal that works there. Couples to T-27 (few-shot already recovers most wrong-leaf errors → re-justify or drop T-27).
- **GATE ROUTE REFRAMED (session 51, advisor-reviewed) = EXTERNAL VALIDATORS over MODEL-INTROSPECTION.** Recurrence is the dominant discriminator (recurring ≈70% vs novel ≈45–49%, ~25pp) and orthogonal to the model's anti-informative self-confidence. Primary gate = **external validators**: E1 recurrence-strength (keyword specificity — **currently computed in `SelectExamples` then DISCARDED**), E2 value-range plausibility (`feature_dictionary_enhanced.json` `value_ranges`, task 5.R3, **NEGATIVE flag** only), E3 retrieval-generation agreement (CAVEAT: few-shot injection breaks independence like `type_crosscheck` → 4-cell stability test first). **Model-introspection** (M1 ensemble cheapest / M2 logprob-margin needs leaf-first + engine) is tested ONLY on the **NOVEL pool** — do NOT pre-shelve ("can't gate a coin flip" is too strong; 49% is an average). **FIRST STUDY (unblocked): E1 recurrence-strength risk–coverage stratified recurrent/novel; the curve decides whether any introspection signal is needed.** As-is gate = `IsAutoInsertable` confidence≥0.85 + exclusion only (`decision.go`); wiring E1 = expose match-strength from `SelectExamples` → thread out of `Classify` → widen the gate → 3 call sites. DECIDED + SHIPPED as the agreement gate (T-32, session 57 — see 'Agreement Gate — SHIPPED' above). Route: `.claude/plans/t23-calibration-benchmark.md` "Route decision"; strategy: `.claude/t23-strategic-implications.md`; PR #43.
- **Real-data measurement is unblocked (session 51).** `classifications.jsonl` = 649 human-verified `(item→actual)` labels from real 2025 usage — but all `model="review"` with DEFAULTED confidence (0.95×502) and selection-biased (602 corrected/47 confirmed), so NOT a direct gate measurement (naive read = artifactual 7.2% flat curve). The gate AND the 5.R1 TF-IDF trigger (keyword miss rate) are measured by ONE replay of the 649 through the current classifier (exclude example-pool self-matches — leakage). WS-D/E and TF-IDF are each "one measurement (this replay) away from a decision"; do NOT build TF-IDF speculatively. Strategic doc §6.
- **Taxonomy audit DONE** — one junk leaf `Apoia-se 4i20` (a description auto-minted as a leaf in 5.R4) → generalized to vendor leaf `Apoia-se`; full surgical propagation (labels only, real `item` kept) across taxonomy + feature dict + training + logs (`*.bak-apoia-rename` backups). Durability gap: rename it in the workbook Referência (T-02 export source) or the next re-export wipes it.

### Log-Append Pivot (WS-B — slices 1–3 done, session 43)
- **Commands append, generate writes** — `add`/`auto`/`batch-auto` append typed entries to `expenses_log.jsonl` via the single writer `internal/appender.ExpandAndAppend`; they never touch the workbook. `generate-workbook` is the only workbook writer. Plain `batch` (manual, non-classifier) still inserts into the workbook.
- **`--dry-run` = classify + write CSVs, no append** — no workbook to skip anymore; dry-run also skips the pre-flight and writes nothing to either log.
- **Failure honesty is load-bearing** — because the log is the only durable persistence, an append failure downgrades that row in place (`AutoInserted=false` + `Error`), keeping the summary count honest, routing the row to `review.csv`, and exiting non-zero. CSVs are written AFTER the append so they reflect downgrades. Guarded by a UNIT test (`cmd.TestAppendClassified_DowngradesRowOnAppendFailure`), NOT acceptance — the pre-flight makes an acceptance-level append failure unreachable.
- **Pre-flight fails fast** — `preflightLogPath`/`verifyAppendable` opens the log in `O_APPEND` mode before classifying (saves ~12 s/row when the log is unwritable). Acceptance: `TestBatchAuto_UnwritableLogPath_FailsFastBeforeClassification` (deterministic, no Ollama).
- **rollover.csv retired** — cross-year installments are normal log lines carrying their real next-year date (`addMonths`), not a separate file.
- **Deprecate-not-delete, NARROW scope** — only `workflow.InsertBatchExpensesFromClassified` orphans (deprecated via `// Deprecated:`, kept + unit-tested for WS-E). `InsertBatchExpenses` + rollover/excel machinery stay LIVE under plain `batch` — do NOT delete with the classified variant.
- **Date reformat gotcha** — the appender writes `date` as `DD/MM/YYYY`; bare `DD/MM` gets `time.Now().Year()` (`ParseDateFlexible`). Fixtures/tests use explicit-year inputs + clean-dividing installment values to stay stable.
- **No dedup (idempotency gap)** — `feedback.AppendExpense` is a plain `O_APPEND` write with no hash-ID dedup → a re-run after a partial failure double-appends (T-20).

### Workbook Generator (sessions 27–29)
- **Spec v2 is the design authority** (`.claude/plans/workbook-generator-spec.md`) — a
  REDESIGN; where it disagrees with the original workbook, the spec wins. §1.1 carries the
  G3 input contract (taxonomy JSON + entries join rules).
- **Derived layout** — row positions computed from taxonomy + entry counts; sheet order =
  taxonomy order via registry `sheetOrder` (never hardcode the 4-sheet list — D0-refs bug).
- **Merges, not fill-down; 2 label cols everywhere; months start col C; Referência omitted**
  — hand-review reversals of source behavior.
- **Golden-master validation** — data-bearing `template.xlsx` blessed 2026-06-11; judge ONLY
  via workbook-inspect dumps, never eyeballing.
- **Oracle-frozen acceptance expectations** — `expected-dump-*` frozen from the trusted
  scratch builder before the port. Limit: oracle and port can share bugs; deliberate output
  changes require re-freezing + manually reviewing the dump delta.
- **Identifiers English, strings pt-BR** — all user-visible text in `Labels`;
  `Labels.RevenueSheet` ("Receitas") appears inside cross-sheet formulas → schema identifier.
- **Generated workbook is not an insertion target** — regenerate-don't-insert; `apply`/`add`
  keep working against hand-maintained workbooks; year-rollover + taxonomy export pending.

### Generate Package Architecture (session 30)
- **styles.go = vocabulary + registration** — named constructors say WHAT a cell is
  (`dataCell`/`grayBanner`/`columnHeader`/`totalRowCell`/`navyBand`/…) over named palette +
  numfmt constants; `styleRegistrar.family(fill,font)` mints General/currency/percent trios.
  Never inline a raw `excelize.Style{...}` in a sheet builder — extend the vocabulary.
  `styleSet` fields are English (MonthCorner; TotalText/TotalTextLeft/TotalValue/TotalValueRight).
- **File homes by domain, not first caller** — `util.go` = pure string/formula/ref helpers, no
  excelize (`cell`, `sheetRef`, `needsQuote`, `sumList/Range`, `lower`, `atoi`); `data_sheet.go` =
  the data-sheet writing vocabulary shared by the expense sheets AND Receitas, plus the unified
  `calculateBlockRows(row, maxEntries)` and `writeDataBand(..., rowHeight, lastCol)` (sole
  behavioral diff between the two sheet kinds is row height 12.75 vs 15).
- **Package stays FLAT** — no styles/sheets subpackages (Go idiom; styleSet/layoutRegistry/Labels
  too coupled). The penciled split is **DONE** (T-02, 2026-06-16, commits 07f395a + 21c6d4e):
  `internal/taxonomy` extracted as the pure input layer (domain types + loader); render config
  (`dataYear`/`headroomRows`/`perGroupPctRows`) relocated into `generate` FIRST (cycle-free).
  Builders reference `taxonomy.X` by full qualification (no alias shim).
- **Behavior-preserving refactors leaned on the oracle** — a mis-parameterized row height fails the
  frozen dump loudly and specifically; that safety net is what made aggressive cross-file merges safe.

### Taxonomy Identity Key (`[ref:taxonomy-identity-key]`)
- **Identity = full path** — a subcategory is identified by sheet/category/subcategory (income:
  group/label), NOT its bare leaf name. Only an exact repeated full path is a validation error;
  cross-path repeats are legal (real data repeats leaves: `Orion` ×3 across Pet blocks; `Aluguel`
  as expense + income).
- **Two-tier routing (T-04 landed)** — typed entries route by full-path key (`byPath`,
  `expensePath`), resolving ambiguous leaves to one block; type-less entries fall back to the
  retained bare-name map (`byName`) where a name shared by >1 full path is *ambiguous* and dropped
  (entry warn+skip, exit 0; never a silent misroute). `registerTarget` keeps the ambiguous set
  sticky (3× re-add trap); full paths join on a null byte (names contain `/`, e.g. `Uber/Taxi`).
  The bare-name tier is **transitional** — retired once the classifier emits type for all entries.
- **T-02 reframed** — NO export command/writer; the real taxonomy is a one-shot hand-authored file
  (`config/taxonomy.json`, 112 subs, gitignored). Long-term direction is DB ingestion, not workbook
  insertion.

### Type Persistence & Full-Path Routing (Plans A & B — IMPLEMENTED session 33)
- **Option 1 chosen** — the classifier emits the FULL PATH (type/category/subcategory) as its label /
  struct identity, not a bare name + ambiguity-only sheet. Cleaner; re-training data exists (607/694
  training examples carry the type in `source`; reviewed entries carry it). Tracked 5.R4 / RUI-4.
- **"Expense Sheet" → "Expense Type"** domain rename WITH JSON key migration (`sheet`→`type`,
  `sheets`→`types`); "sheet" stays reserved for Excel worksheet addressing only
  (`models.SheetLocation.SheetName`, `internal/inspect`, `sheetOrder`). Plan A — DONE.
- **Type was dropped at every log-write layer** — now captured: feedback.Entry + ExpenseEntry carry
  `Type` (`omitempty`), set post-construction on the apply path. `expenses_log.jsonl` /
  `classifications.jsonl` carry it; `backfill-type.py` recovers it into pre-existing logs (partial —
  only reviewed entries). The 7-field classified CSV still lacks a type column (RUI-4).
- **Routing is two-tier, guard is TRANSITIONAL (T-04, Plan B — IMPLEMENTED)** — typed entries route
  by full-path key (`expensePath`); type-less entries (auto.go + batch_auto.go + ~355 existing log
  lines) fall back to the retained bare-name map with ambiguous-skip. The fallback is a **bridge**, to
  be retired once the classifier emits type for every entry (5.R4/RUI-4). `scanEntries` logs a
  one-line count of type-less fallbacks so the remaining surface is measurable. Advisor-caught:
  deleting the fallback now would drop the auto-inserted majority.
- **String-equality contract** — a typed entry routes only if type/category/sub byte-match
  taxonomy.json; wrong spelling → warn+skip (never silent misroute). NFC-normalization (`normalizeKey`,
  via `golang.org/x/text`) is applied at every key boundary so accent NFC/NFD skew can't drop entries.
  So when the classifier emits type (5.R4) it must produce taxonomy-exact strings.
- **Sequencing** — Plan A (T-05) ✅ + Plan B (T-04) ✅ implemented (stacked branches `feat/persist-expense-type`
  / PR #29 → `feat/full-path-entry-routing` / PR #30). Real-data proof pending (Bf runbook / T-06).
  Next: classifier full-path label (5.R4/RUI-4) — DONE in T-13.

### Domain Boundary (decided session 32 in LLM repo context)
- **Classification logic in expense-reporter (Go)** — it's a product feature, not LLM infrastructure
- **MCP thin wrapper in this repo** (`mcp-server/`) — 2 tools: `classify_expense` (→ `auto --json`), `add_expense` (→ `add --json`); calls Go binary as subprocess; registered with Claude Code. **Layer 5.8 fully shipped** (5.8a Go `--json` + 5.8b Python MCP server, plus follow-ups: `add --data-dir`, `classification_id` surfaced, prediction flags on `add`). `auto_add` tool was dropped by design — see `mcp-server/.memories/KNOWLEDGE.md` "Two Tools, Not Three".
- **Training data strategy:** hybrid — feature dictionary as system context + top-K few-shot examples per request
- **Structured output:** Ollama `format` param (proven reliable in LLM infra work)

### Review UI (session 21)
- **Local-first, not cloud** — review UI is a single self-contained HTML file the CLI
  bakes data into. Lovable cloud plan (`docs/plans/lovable-suggestion-plan.md`) superseded.
- **`review` is a producer, not a server** — bakes queue + 3-level taxonomy into an HTML
  template via `__REVIEW_DATA__` placeholder replacement; no HTTP server, no endpoints.
- **Workbook write out of scope for `review`** — UI emits `reviewed.json`; the `apply`
  command ingests it into workbook + feedback logs.
- **Review UI is the only type producer** — the type choice is made in the review page and
  exported in `reviewed.json` (key `type`, legacy `sheet` still read); saved corrections live in
  localStorage and are recoverable by re-export (no page change needed for backfill).
- **Taxonomy source** — workbook's "Referência de Categorias" sheet via
  `excel.LoadReferenceSheet`, grouped sheet→category→subcategory at runtime.

### Classification Strategy
- **Model candidates:** `my-classifier-q3` (Qwen3-8B) vs Qwen2.5-Coder-7B (speed). Benchmark deferred.
- **Auto-insert gate:** keyword AGREEMENT (T-32, SHIPPED) — confidence measured dead, no longer gates; see 'Agreement Gate — SHIPPED'
- **Feature dictionary pre-filter:** skipped in 5.2; deferred to 5.7 (few-shot injection task)
- **Few-shot injection (5.7):** implemented — keyword layer (layer 1 of 3-layer cascade) complete; SelectExamples in `internal/classifier/examples.go`; loaders in `loader.go`; injected as user/assistant pairs in buildRequest; TF-IDF/embeddings deferred to future sessions
- **`expenses_log.jsonl`** — slim insert log (`id`, `item`, `date`, `value`, `subcategory`, `category`, `type` (omitempty), `timestamp`); separate from `classifications.jsonl`; ID is sha256[:12] shared across both files for cross-file correlation. NOTE: the append path (`appender`) now writes `date` as `DD/MM/YYYY`; legacy lines + the generator's per-entry loader still accept bare `DD/MM`.

### Go Conventions
- **Cobra pattern:** Each subcommand is a `.go` file in `cmd/expense-reporter/cmd/`
- **Brazilian format:** DD/MM/YYYY dates, comma decimal separator (`1.234,56` notation)
- **Error pattern:** `fmt.Errorf("context: %w", err)` — wrap with context, not bare return
- **Table-driven tests:** Standard approach — any new command gets table-driven test coverage
- **Unit tests use testify:** `assert`/`require` from `github.com/stretchr/testify` (convention change session 10); acceptance `test/verify/` already used testify
- **Acceptance tests:** `//go:build acceptance` tag, separate from unit tests, live Ollama required
  (EXCEPTION: generate-workbook tests are Ollama-free and deterministic)
- **Acceptance harness:** `test/harness/` (Context, Scenario, Run), `test/actions/`, `test/verify/`;
  `run-acceptance.sh` with Ollama pre-flight, workbook auto-detect, filter arg, keep-artifacts flags
- **Workbook config:** `EXPENSE_WORKBOOK_PATH` env var — script auto-detects from relative path to workbook
- **classify/auto input:** Positional args with `utils.ParseCurrency` for value (accepts both `.` and `,`)
- **TDD:** Write tests red-first before implementation (5.2 was an exception — tests written after)
- **Working directory:** Shell commands run from `expense-reporter/` — do not prefix paths with it
- **Ollama timeout policy (session 8):** 1st timeout = retry (cold start), not a rejection. Only treat as 1st rejection if the model responds with wrong output. Two rejections → escalate to Claude.
- **Ollama parallelization ceiling (session 15):** 3 parallel codegen calls only safe for tiny near-identical prompts. Default to serial for non-trivial codegen — VRAM ceiling causes silent degradation/timeouts.
- **`my-go-qcoder` first benchmark (session 23, 2026-05-18):** Used for `cmd/review.go` (verdict 1), `render_test.go` (verdict 2), `taxonomy_test.go` (verdict 2), `queue_test.go` (verdict 1). Struggled with intermediate Go map types in a prior session (verdict 0 on `taxonomy.go`) — passes cleanly when types are pre-defined in context files. Test generation is its strongest use; cobra command wiring is solid. Preferred over `my-go-q25c14` going forward for single-file codegen tasks. (Session T-02: verdict 0 on a loader edit with subtle algorithm + data-aware separator — wrote from scratch; conceptual/data-shaped defects remain a weak spot.) NOTE (session 43): qcoder returned HTTP 500 on `warm_model` twice. ROOT CAUSE (diagnosed 2026-06-30, llm repo): host-RAM ENOMEM, NOT VRAM contention. The model store is on a 9p mount (/mnt/i) → Ollama disables mmap (`UseMmap:false`) → reads the full 19.3 GiB qwen3-coder:30b blob into RAM, which exceeds WSL2's ~11 GiB free → runner panics (`exit status 2`). Mitigation (session 98): raised WSL `.wslconfig` memory=24GB — this is the actual fix and stays load-bearing. (Store later moved to a dedicated ext4 vhdx for faster loads, but that did NOT free RAM: Ollama keeps mmap off for partially-offloaded models regardless of filesystem.) Fell back to `my-go-q25c14` (verdict 1).
- **Verbatim code moves are NOT codegen (session 29):** 530-line package-rename moves go to
  sed/python — 3 warm-model timeouts proved the shape wrong for LLMs (pure transcription risk).
  Delegate synthesis (new tests, new units), not copying. Also: the model hallucinated fixture
  literals it was explicitly given — always re-check literals in generated tests.
- **Excelize formula APIs (session 27):** `SetCellFormula` takes the formula WITHOUT a leading
  `=`; stale-formula display fix = `UpdateLinkedValue()` + `SetCalcProps(FullCalcOnLoad)`.
- **Excelize API confusions to expect from local models (session 29):** `NewSheet` returns
  (int, error); no `SetCellFont` (use NewStyle+SetCellStyle); `MergeCell` not `MergeCells`.
- **`gh` on this repo (session 29):** `gh pr edit` / `pr view --comments` fail (projects-classic
  GraphQL deprecation) — use `gh api repos/.../pulls/N` REST endpoints instead.

### Test Conventions (session 15)
- **Acceptance-first** — discuss scenarios → write acceptance tests → drop into TDD inner loop for unit tests
- **Given naming** — Event Modeling style, past-tense events that happened (`expenseAutoConfirmed`); state-only exception for empty event streams (`noClassificationsRecorded`)
- **Then naming** — composable `[]func(*Context)` slices joined via `slices.Concat`; describe the concern, not the scenario
- **Doc:** `expense-reporter/test/PATTERNS.md` is the spec — send to Ollama as context when delegating test generation
- **generate-basic fixture sub-format (session 29):** taxonomy.json + entries.jsonl +
  oracle-frozen `expected-dump-*/` — NOT config.json+input.csv. See PATTERNS.md. Plan B keeps a
  MIX of typed and type-less entries here (typing every line would mask a broken fallback).
- **Log-append fixtures (session 43):** non-dry-run batch-auto/auto/add fixtures assert the log via
  `verify.ExpenseLogMatches(<fixDir>/expected-expenses_log.jsonl)`; use **explicit-year inputs**
  (`DD/MM/YYYY`) + clean-dividing installment values (the append path reformats dates). A `--dry-run`
  fixture (`extra_args`) never appends — those tests cover CSV production only.

### Correction Workflow (session 15, Layer 5.9)
- **`correct` is feedback-only** — no `--workbook` flag; user fixes workbook manually
- **Requires a prior entry** — fails with hint to use `add` if none exists (matches design: corrections always override a prediction)
- **Telegram-flow corrections shipped (session 17)** — `add --predicted-subcategory` writes `confirmed`/`corrected` entries; `classify_expense` now surfaces `classification_id`; `add_expense` MCP tool forwards all prediction flags

### Classification Data
- `confusion_analysis.json` gitignored (may contain real expense descriptions as test cases)
- `algorithm_parameters.json` tracked (no personal data, pure algorithm config)

### Acceptance Test Fixture Stability (session 5)
- **`--threshold` deprecated + inert (T-32)** — auto-append mechanics fixtures now need gate-PASSING items (spec-1.0, unambiguous, model agrees)
- **Posto Ipiranga** is the canonical auto-append test item (spec 1.0 → Combustível, gate-passing); **Uber Centro** now FAILS the agreement gate (spec 0.8, ambiguous) → `add`/structural tests only
- **Exclusions test** scoped to structural validation only — LLM routing assertions are fragile;
  exclusion logic is deterministic and covered by `classifier/decision_test.go` unit tests
- **auto command** now falls back to review (exit 0) on resolution/ambiguous errors; only IO/capacity → exit 1

### Integration Testing Findings (session 3)
- LLM resolves multi-word context better than keyword specificity alone — "VA compras" → 100%
  despite "va" having specificity=0.36 in feature dictionary
- Fallback category "Diversos" at high confidence is a real risk — now blocked via exclusion list
- `Transporte` appearing as subcategory at 90% in Uber case — taxonomy oddity, not urgent
<!-- /ref:active-decisions -->

<!-- ref:session-reading-guide -->
| Task | Read first | Notes |
|------|-----------|-------|
| **Harness extraction Session B (T2 — migrate expenses `test/` to the module)** | expenses plan §5 in `.claude/plans/harness-extraction-plan.md`; module `docs/ADOPTION.md` (consumer wiring); `test/harness/{ollama,comparator}.go` (the two files that STAY, → domain layer) | Module = `github.com/leandror172/acceptance-harness` **v0.1.1** (go 1.25; repo go is 1.25.5-ready). Add dep; delete `test/harness/` EXCEPT ollama.go+comparator.go (→ `test/extern/` + `test/verify/csv_compare.go`); `Context` workbook fields (`WorkbookPath`/`DataDir`/`WorkbookDir`) → `Env` or a domain wrapper (decide by touch-point count); `FixtureConfig` domain fields → `Raw` decode via a domain loader. Gate: full `-tags=acceptance -timeout 30m` green AND `-tags=replay` still compiles [[feedback_rename_json_tag_acceptance]]. Reference repoint: career-search PR #7 (verify-namespace strategy, RegisterFlags in TestMain). |
| **5.R2 follow-ups (CL1 trigger widening / CL2 pre-warm / alternate embedder)** | `.claude/scratch/replay-649/FINDINGS-5r2.md`; `.claude/plans/5r2-embedding-retrieval.md` § Check-later | ADOPTED s54 (miss full-path 18.8→52.5 no-think). CL1: widen the fallback to weak/ambiguous keyword matches (top score < 0.7) now a miss-only baseline exists. CL2: pre-warm (`warm-embeddings` cmd or post-`apply` hook) if the burst annoys. qwen3-embedding:8b (+2.5pp hit@5) unbenchmarked in-model — only if chasing points. |
| **T-23 — gate route (measured; decision still deferred)** | `.claude/plans/t23-calibration-benchmark.md` "Route decision"; `.claude/t23-strategic-implications.md`; `.claude/scratch/replay-649/FINDINGS-model.md` | Route = external validators over model-introspection (session 51). Measured session 52: E1 specificity works; agreement is the strongest gate; confidence is dead. 5.R2 (the accuracy lever) now BUILT — the gate wiring (T-32) feeds the final call. |
| **T-20 — expense-log dedup (good independent pick-up)** | `internal/feedback/expense_log.go` `AppendExpense`; appender | Plain `O_APPEND`, no hash-ID dedup → re-run after partial failure double-appends. Add dedup-on-ID or `--resume`. Deterministic, no Ollama. |
| **WS-D — retire bare-name fallback (T-09) — NEXT product step** | `.claude/t32-agreement-gate-report.md` §7; `[ref:taxonomy-identity-key]`; `internal/taxonomy/loader.go` `scanEntries`; stderr fallback count | T-32 agreement gate LANDED (PR #46 merged), so novel rows are protected. But absolute precision is UNMEASURED (649 = confidence-selected) → design **gate-to-review**, NOT silent insert. Session-52 replay confirms recurrence-first (leaky 63% vs clean 44%). |
| Delete dead insert code (WS-E) | plan "WS-E"; `internal/workflow`, `internal/excel` write side, `internal/batch` insert path | Only after WS-D. NARROW scope: delete only deprecated `InsertBatchExpensesFromClassified` + its test; `InsertBatchExpenses`/rollover/excel stay LIVE under plain `batch`. |
| **T-24 — --think default decision** | `.claude/t14-benchmark-report.md` "T-30"; `.claude/scratch/replay-649/FINDINGS-{model,5r2}.md` | Gate band: think-on −2.3pp. NEW (s54): on the miss path with 5.R2 ON, think adds only +2.5pp churn (10 fixed/8 broke) at 5.5× latency. Both strata now favor no-think. |
| **T-27 — leaf-level descriptions (RE-JUSTIFY or drop)** | T-22 addendum; T-30 report; `descriptions.go` | Undercut by T-30 (few-shot already recovers most wrong-leaf). Re-justify on few-shot-ON evidence or drop. |
| **T-28 — alternative classification strategies** | `internal/classifier/classifier.go` (prompt + GBNF); T-13 report | Two-stage / two-pass-grammar (unconstrained-first) vs one-shot enum. Measure on the T-14 sample. |
| Reviewed-installment under-recording (T-21) | `internal/review/queue.go` `ReadQueue`; review.html export JS; `internal/apply/types.go` | Count discarded at `ReadQueue:63`, absent from `reviewed.json`. Thread count/rawValue through. |
| q35 grammar bug + persona cleanup (T-25) | `.claude/t14-benchmark-report.md` "Model matrix outcome" | Reproduce minimal (qwen3.5 + `think:false` + `format`), check newer Ollama, file upstream. |
| Taxonomy `Apoia-se` durability (session 49) | `.claude/plans/leaf-first-classification.md` § Prerequisite; workbook Referência sheet | Rename `Apoia-se 4i20`→`Apoia-se` in the Referência sheet (export source) or re-export wipes it. |
| Promote merged log to canonical (deferred) | `.claude/scratch/merge_year_logs.py`; gitignored `expenses_log-allyears.jsonl` | User decides when to swap canonical + delete per-year files. |
<!-- /ref:session-reading-guide -->
