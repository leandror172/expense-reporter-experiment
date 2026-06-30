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
- **Preferred model:** `my-go-qcoder` (qwen3-coder:30b via ollama-bridge MCP) — benchmarked session 23 (2026-05-18); verdicts 2/2/1/1 on test + cobra generation. Fallback: `my-go-q25c14` (qwen2.5-coder:14b) if qcoder unavailable.
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
- **WS-A / T-11 — DONE (s37):** multi-year log support. **WS-C / income route — DONE & validated on real 2024 data (s38):** 3-level income model, dual-format loader, `--income-entries`, signed values, 3-level Receitas render.
- **T-13 — DONE (session 41), PR #36 MERGED to master, reviewed session 42.** The classifier predicts the full `(type,category,subcategory)` path against `taxonomy.json` via an atomic 112-path enum; `Result.Type`; `classifier.Taxonomy`/`LoadTaxonomy` deleted (feature dict now keyword-only). New `internal/taxonomy/path.go` helpers (`PathEnum`/`Split`/`PathFor`/`ResolveLeaf`/`TypesForLeaf`/`CategoryForLeaf`). `add` resolves via taxonomy + `--type`/prompt/non-interactive-error hybrid; `auto`/`batch-auto` write the predicted type. Report: `.claude/t13-implementation-report.md`.
- **PR #36 review follow-ups — session 42:** (1) default model reverted qcoder→`my-classifier-q3` for ALL commands — enum validity is GRAMMAR-enforced (Ollama `format`→GBNF, model-independent), so qcoder (20.7 GB, doesn't fit 12 GB GPU) added only latency; q3 validated end-to-end. q3 accurate but slow (~12 s/call → T-14). (2) T-15 Slice 1: predicted `type` surfaced in `CandidateOutput`/`AddOutput` JSON. (3) Acceptance repaired (T-17×11 + T-18×2) — taxonomy_path newly mandatory but un-wired in many `Given`s (build-tag hid it). Postmortem: `.claude/session42-postmortem.md`.
- **WS-B (commands → log-append) — slices 1–3 DONE:** `add` (s1) + `auto` (s2) + **`batch-auto` (s3, session 43, branch `feat/ws-b-slice3-batch-auto-log-append`)** all append via `internal/appender.ExpandAndAppend` (single append-time writer; installments expand at append; no workbook writes). Slice 3: rollover.csv retired; append-failure downgrades the row (honest count + non-zero exit); pre-flight fails fast on an unwritable log; `workflow.InsertBatchExpensesFromClassified` deprecated (plain `batch` keeps `InsertBatchExpenses` + machinery LIVE). Plan: `.claude/plans/ws-b-slice3-batch-auto-log-append.md`. **Remaining: slice 4 `apply` (delete write-half).**
- **Next:** review + merge **PR #37** (slice-3 branch, 7 commits, full suite 15/15 green), then **WS-B slice 4 `apply` → log-append**; then WS-D (retire bare-name fallback, T-09), WS-E (delete dead insert code). PR #36 follow-ups still open: **T-14** (model accuracy+speed benchmark), **T-19** (enum "none-of-these" escape hatch); T-16 doc cleanup done. Open: promote merged `expenses_log-allyears.jsonl` to canonical + retire per-year split (deferred); **T-20** expense-log dedup (no idempotency on re-run).
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
- **`my-go-qcoder` first benchmark (session 23, 2026-05-18):** Used for `cmd/review.go` (verdict 1), `render_test.go` (verdict 2), `taxonomy_test.go` (verdict 2), `queue_test.go` (verdict 1). Struggled with intermediate Go map types in a prior session (verdict 0 on `taxonomy.go`) — passes cleanly when types are pre-defined in context files. Test generation is its strongest use; cobra command wiring is solid. Preferred over `my-go-q25c14` going forward for single-file codegen tasks. (Session T-02: verdict 0 on a loader edit with subtle algorithm + data-aware separator — wrote from scratch; conceptual/data-shaped defects remain a weak spot.) NOTE (session 43): qcoder returned HTTP 500 on `warm_model` twice (VRAM contention) — fell back to `my-go-q25c14` (verdict 1).
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
| **Merge PR #37 (slice-3 branch, do FIRST)** | PR #37 (`feat/ws-b-slice3-batch-auto-log-append` → master); `.claude/ws-b-slice3-implementation-report.md` | Open + green (suite 15/15, 864 s). Review + merge. Point at master, don't fuss over base ([[feedback_pr_base_no_rebase_fuss]]). |
| **WS-B slice 4 `apply` → log-append (START HERE)** | retire-insertion plan "WS-B"; `cmd/.../apply.go`; `internal/apply` writer; slice-3 plan `.claude/plans/ws-b-slice3-batch-auto-log-append.md` (the pattern to mirror) | Slices 1–3 done (add/auto/batch-auto append via `appender.ExpandAndAppend`). Keep the log-writing half; delete the `excel.WriteBatchExpenses`/`AllocateEmptyRows`/`FindSubcategoryRowBatch` write half. Mirror slice 3's failure-honesty downgrade + pre-flight. **Run target tests in GROUPS with `-v` — q3 ~12 s/call, full suite exceeds the 600 s default; use `-timeout 30m`.** |
| **PR #36 review follow-ups (T-14 / T-19)** | `.claude/session42-postmortem.md`; classifier `.memories/KNOWLEDGE.md` (grammar-enforcement) | T-14: benchmark accuracy+speed across q3/q35/qcoder on real labeled data, set the smallest-that-fits default. T-19: design a sentinel/`Diversos` escape so the enum can decline out-of-domain inputs (currently force-mapped at high confidence). T-16 doc cleanup is DONE. |
| Slice-2/3 loose ends (T-12) + stale memory | plan "WS-B progress" (LOOSE ENDS); `test/auto_test.go` | `auto.go` `✓ Inserted`→`✓ Appended` rename DONE (session 43); `logExpense` deleted. Still: rewire `test/auto_test.go`'s `RequireWorkbook`-gated cases (LOW/ambiguous path skips → uncovered); add `internal/appender/.memories`; refresh `internal/feedback/.memories` (no-dedup note). |
| Retire bare-name fallback (T-09 / WS-D) | `[ref:taxonomy-identity-key]`; `internal/taxonomy/loader.go` `scanEntries`; stderr fallback count; **tasks.md T-19** | **Unblocked by T-13.** Gate on real-data type-less surface ~0. **Coupled to T-19:** retiring the fallback leans harder on the model path, which (post-T-13) can no longer decline novel inputs — resolve the escape-hatch gap first. The now-dead `BuildTypeIndex`/`LookupType` may be removed in WS-E. |
| Delete dead insert code (WS-E) | plan "WS-E"; `internal/workflow`, `internal/excel` write side, `internal/batch` insert path | Only after WS-B/WS-D. **NARROW scope:** delete only the deprecated `InsertBatchExpensesFromClassified` + its test; `InsertBatchExpenses`/rollover/excel stay LIVE under plain `batch`. Verify `internal/inspect` reader needs before deleting `internal/excel/reader.go`. Also: unused `taxonomy.BuildTypeIndex`/`LookupType`. |
| Expense-log dedup (T-20) | `internal/feedback/expense_log.go` `AppendExpense`; appender | `AppendExpense` is a plain `O_APPEND` write with no hash-ID dedup → a re-run after a partial failure double-appends. Add dedup-on-ID or `--resume` semantics before batch-auto runs at volume on real data. |
| Promote merged log to canonical (deferred) | `.claude/scratch/merge_year_logs.py`; gitignored `expenses_log-allyears.jsonl` | User decides when to swap canonical + delete per-year files. |
<!-- /ref:session-reading-guide -->
