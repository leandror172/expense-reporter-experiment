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
- **Plan A (T-05) + Plan B (T-04):** MERGED. Expense `Type` persisted end-to-end; two-tier routing — typed by full path (`byPath`), type-less via the **transitional** bare-name fallback (`byName` + ambiguous-skip); NFC keys; stderr fallback count.
- **5.R4 historical extraction — DONE.** 2022–2025 old workbooks → 1808 deduped records; corpus 694→1788; per-year expense logs.
- **PIVOT (session 36): retire workbook insertion, keep only generation** — JSONL logs become the single source of truth, `generate-workbook` the only writer. Plan: `.claude/plans/retire-insertion-keep-generation.md`.
- **WS-A / T-11 — DONE (s37):** multi-year log support. **WS-C / income route — DONE & validated on real 2024 data (s38).**
- **T-13 — DONE (session 41), PR #36 MERGED.** Classifier predicts the full `(type,category,subcategory)` path via an atomic 112-path enum. Default model `my-classifier-q3`.
- **WS-B (commands → log-append) — COMPLETE & MERGED (slices 1–4, PR #38):** `add`/`auto`/`batch-auto`/`apply` all append via `internal/appender.ExpandAndAppend`; no classifier-fed command writes the workbook (plain `batch` still does).
- **T-19 — DONE (session 45), PR #40 MERGED.** Sentinel decline path (`Diversos`@0.30 fixed). Residual confident-wrong risk → T-23.
- **T-14 — DONE (session 46), PR MERGED.** Report: `.claude/t14-benchmark-report.md`. q3 = 63.0% full-path (clean 49.1%); **calibration broken (91% of errors ≥0.85 → the auto-insert gate does not protect; WS-D gate NOT passed)**. `--think` flag: no-think 10× faster (1.5 s vs 14.4 s) at −3.3 pp but kills the T-19 sentinel; q35 disqualified (think:false drops the format grammar); default think-on.
- **T-22 — DONE (session 47), PR #42 OPEN** (`feat/t22-type-descriptions`). Type-level taxonomy descriptions merged into the classifier prompt from a tracked sidecar `config/type-descriptions.json`. No-think A/B: **English +6.0 pp TYPE accuracy** (74.3→80.3), +2.6 pp full-path; Portuguese dropped; **calibration UNCHANGED — not a gate fix**. Leaf-level extension = T-27. Report addendum "T-22 Addendum".
- **T-26 — DONE (session 48).** Think-on confirmation of T-22. Type lift direction-consistent across runs (adoption holds); single-run deltas within noise. **Calibration still broken (86–88% HC-wrong) → gate untouched; T-23 stands.**
- **T-23 EXPLORED (session 49): logprob-confidence probe + leaf-first spin-off.** Probed token logprobs as a gate signal: Ollama 0.17.5 exposes logprobs but reports them PRE-grammar-mask (raw, not renormalized) → can't read P-over-legal-paths directly; type-first enum makes the margin unreadable, LEAF-FIRST makes it legible (Uber leaf p=0.986). Calibration NOT yet proven. Spun off **leaf-first classification** as a standalone accuracy change + T-23 precondition (plan + **D1 LOCKED = second `type` field**). Taxonomy audit: 1 junk leaf `Apoia-se 4i20`→`Apoia-se` (full propagation; workbook re-export pending). Report `.claude/t23-logprob-confidence-probe.md`; plans `.claude/plans/{leaf-first-classification,t23-calibration-benchmark}.md`. Branch `docs/t23-calibration-probe` (unpushed).
- **Next:** **leaf-first A/B** (T-30 — D1 locked, build next; decide D2/D3/D4 at build time) → **T-23** calibration benchmark (rides on leaf-first; signals 1/2a/3 on Ollama, 2b engine-PoC subagent deferred) → **WS-D (T-09)** retire bare-name fallback → **WS-E** delete dead insert code. Independent: **T-20** dedup (pick up while PR #42 reviews). Open: merge PR #42; durability — rename `Apoia-se` leaf in workbook Referência; T-27/T-28/T-21/T-24/T-25; promote all-years log (deferred).
- **Cross-repo:** LLM infra at `/mnt/i/workspaces/llm/` — personas, MCP server, platform docs.
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

### Leaf-First & Token-Confidence (T-23 exploration, session 49)
- **Ollama 0.17.5 reports PRE-grammar-mask logprobs** — the raw model distribution, NOT renormalized over grammar-legal tokens (proven: a schema-forced key reports the chosen token at p≈0 while the model's wanted token shows 1.0; legal tokens are often outside `top_logprobs`). So you cannot read "P over legal paths" directly. Overturns the initial "grammar renormalizes → token-dist == taxonomy-dist" theory. Report: `.claude/t23-logprob-confidence-probe.md`.
- **Leaf-first split from T-23** — leaf-first classification (enum = 104 leaf names, derive type/cat via `ResolveLeaf`) is now a STANDALONE accuracy change judged on accuracy first (attacks T-14's dominant "right sheet, wrong leaf"), AND the precondition that makes any token-confidence signal legible (type-first makes the margin unreadable; leaf-first: Uber leaf p=0.986). Plan: `.claude/plans/leaf-first-classification.md`.
- **D1 LOCKED = second `type` field** — schema emits `leaf` THEN `type` (order load-bearing: leaf first keeps the leaf-first commitment; type second is a real recurrence judgment). Always-predict; `ResolveLeaf` consults it only for the 5 cross-type collisions and cross-checks the 99 unique leaves as a free calibration signal. Grounded in the audit: the collision axis IS recurrence (pets `Ambos`/`Lilly`/`Orion` ×3, `Dentista`/`Estacionamento` ×2 span Fixas/Variáveis/Extras — not in the leaf name). Open: D2 (tree vs flat prompt), D3 (benchmark-only vs prod flag), D4 (adopt threshold).
- **T-23 calibration benchmark** (`.claude/plans/t23-calibration-benchmark.md`) — 4 signals: self-confidence baseline / answer-perplexity (Ollama today) / logprob-margin (needs teacher-forcing engine — DEFERRED) / ensemble-agreement (no new lib). Primary metric = risk-coverage curve; stratify leaky/clean (must work on CLEAN/novel). Signals 1/2a/3 run on Ollama first; the 2b engine-PoC subagent (llama.cpp/vLLM, ask model before spawning) is only sent if 2b earns it.
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
- **Confidence threshold:** HIGH ≥ 0.85 (auto-insert), LOW < 0.85 (print candidates + ⚠ signal)
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
- **Threshold 0.0** in mechanics-testing fixtures (installments, rollover) — decouples from
  classifier confidence non-determinism; other fixtures use 0.85
- **Uber Centro** is the canonical reliable test item — consistently returns Uber/Taxi subcategory
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
## Pre-Session Reading Guide

*What to read before each pending work item.*

| Task | Read first | Notes |
|------|-----------|-------|
| **Leaf-first classification A/B (T-30, do FIRST — precondition for T-23)** | `.claude/plans/leaf-first-classification.md`; `internal/classifier/classifier.go` (`buildResponseSchema`/`writeTaxonomyTree`); `internal/taxonomy` `ResolveLeaf` | Enum = 104 leaf names, derive type/cat upward; attacks T-14's dominant "right sheet, wrong leaf". **D1 LOCKED = second `type` field, schema order leaf-THEN-type.** Still open: D2 (tree vs flat prompt), D3 (benchmark-only vs prod — lean benchmark-only first), D4 (adopt threshold). Reuse the T-14 harness; A/B vs current full-path enum on the 300-item sample; negative result is valid (then judged as the T-23 enabler). |
| **T-23 — calibration/gate rethink (blocks WS-D; rides on leaf-first)** | `.claude/plans/t23-calibration-benchmark.md`; `.claude/t23-logprob-confidence-probe.md`; `.claude/t14-benchmark-report.md` (calibration + "Results — think-on"); `internal/classifier/decision.go`; `config.json` `auto_insert_excluded` | Confidence uninformative (86–95% of errors ≥0.85); T-22/T-26 confirmed descriptions don't fix it. Benchmark 4 signals (self-confidence / answer-perplexity / logprob-margin / ensemble) on the T-14 sample; risk-coverage primary, stratify leaky/clean. Ollama reports PRE-mask logprobs (per the probe). Signals 1/2a/3 on Ollama; 2b engine-PoC subagent deferred (ask model before spawning). Fallback if no signal wins: recurrence-first gating. |
| **T-27 — leaf-level descriptions (extends T-22; sequence AFTER leaf-first)** | T-22 addendum; `writeTaxonomyTree`/`descriptions.go`; `config/type-descriptions.json` | Type-level descriptions left "wrong leaf" untouched. Per-leaf (×112) targets those — **motivate via T-14's robust "right sheet, wrong leaf", NOT T-26's noisy novel-type number.** Needs prompt-size/latency measurement; extends the sidecar schema beyond the flat type→desc map. Composes with leaf-first. |
| **T-28 — alternative classification strategies** | `internal/classifier/classifier.go` (prompt + `format`/GBNF); T-13 report; t14 addendum | Explore beyond one-shot atomic-enum: (a) two-stage — leaf-first (now T-30) IS one variant; (b) two-pass grammar — classify UNCONSTRAINED first, then re-ask WITH the grammar only if the free-form answer doesn't match the tree (couples to T-23's "unconstrained-first" confidence route). Measure vs baseline on the T-14 sample. |
| **T-24 — --think default decision** | `.claude/t14-benchmark-report.md` "Results — think-on"; T-22 addendum; `sigtest_t26.py` | Data now exists. Descriptions on → think-on's sentinel advantage gone → accuracy-vs-speed call: ~+2.7 pp full-path for ~10× latency (likely not sig on n=300). If T-23 moves safety off confidence (or ensemble needs no-think for cost), the no-think case strengthens. |
| **WS-D — retire bare-name fallback (T-09, after T-23)** | `[ref:taxonomy-identity-key]`; `internal/taxonomy/loader.go` `scanEntries`; stderr fallback count | **Gate NOT passed in T-14/T-26** — ~37% wrong full paths behind a gate that stops ~9% of them. Requires T-23 + real-data type-less surface ~0. |
| Delete dead insert code (WS-E) | plan "WS-E"; `internal/workflow`, `internal/excel` write side, `internal/batch` insert path | Only after WS-D. **NARROW scope:** delete only deprecated `InsertBatchExpensesFromClassified` + its test; `InsertBatchExpenses`/rollover/excel stay LIVE under plain `batch`. Also unused `taxonomy.BuildTypeIndex`/`LookupType`. |
| Slice-2/3 loose ends (T-12) + stale memory | plan "WS-B progress" (LOOSE ENDS); `test/auto_test.go` | Rewire `test/auto_test.go`'s `RequireWorkbook`-gated cases (LOW/ambiguous path skips → uncovered). `internal/appender/.memories` still absent. |
| Expense-log dedup (T-20) | `internal/feedback/expense_log.go` `AppendExpense`; appender | Plain `O_APPEND`, no hash-ID dedup → re-run after partial failure double-appends. Add dedup-on-ID or `--resume` before volume use. Good independent pick-up while PR #42 reviews. |
| Reviewed-installment under-recording (T-21) | `internal/review/queue.go` `ReadQueue`; review.html export JS; `internal/apply/types.go`; `cmd/.../apply.go` | Count discarded at `ReadQueue:63`, absent from `reviewed.json`. Thread count/rawValue through the chain. UX: 3 rows vs ×3 badge. |
| q35 grammar bug + persona cleanup (T-25) | `.claude/t14-benchmark-report.md` "Model matrix outcome"; Ollama issue tracker | Reproduce minimal case (qwen3.5 + `think:false` + `format`), check newer Ollama, file upstream. Delete inert `my-classifier-q35-nothink` from the LLM-repo registry. |
| Taxonomy `Apoia-se` durability (session 49) | `.claude/plans/leaf-first-classification.md` § Prerequisite; workbook Referência sheet | The `Apoia-se 4i20`→`Apoia-se` rename was applied to `config/taxonomy.json` (gitignored) + data files, but the taxonomy is exported from the workbook — rename it in the Referência sheet (or export step) so re-export doesn't wipe it. |
| Promote merged log to canonical (deferred) | `.claude/scratch/merge_year_logs.py`; gitignored `expenses_log-allyears.jsonl` | User decides when to swap canonical + delete per-year files. |
<!-- /ref:session-reading-guide -->
