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

## Current Status

- **Pre-history (Claude Desktop):** Phases 1–11 complete — full CLI (add/batch/version), 190+ tests, v2.1.0
- **Classification analysis:** Complete (auto-category work) — results in `data/classification/`
- **Active layer:** Workbook Generator — **COMPLETE**; **PR #27 MERGED to master**. Follow-on (T-02)
  on branch `refactor/internal-taxonomy` (not yet merged): `internal/taxonomy` extracted + real
  `config/taxonomy.json` authored + full-path identity key.
- **Last checkpoint (this session):** PLANNING ONLY (no commits). Authored two advisor-reviewed
  implementation plans + a proposed-memories file, all on `refactor/internal-taxonomy`:
  - `.claude/plans/persist-expense-type.md` (Plan A / **T-05**) — persist expense type end-to-end,
    rename ExpenseSheet→ExpenseType, migrate JSON keys (`sheet`→`type`), partial backfill.
  - `.claude/plans/full-path-entry-routing.md` (Plan B / **T-04**) — two-tier routing (typed →
    full-path key; type-less → retained bare-name map + ambiguous-skip fallback).
  - `.claude/plans/proposed-memories.md` — memories to author + updates to existing ones.
- **Identity decision:** a subcategory's identity is its FULL PATH (sheet/category/sub; income
  group/label), not the bare leaf name — only an exact full-path repeat errors; a bare name shared
  by >1 path is *ambiguous* → warn+skip (`[ref:taxonomy-identity-key]`,
  `.claude/plans/taxonomy-identity-key.md`).
- **Type-persistence decision (this session):** option 1 (full path as model label / struct
  identity); rename "Expense Sheet"→"Expense Type" with JSON key migration; the ambiguity guard
  is a **permanent fallback** for type-less entries, NOT removed by T-04.
- **Input contract (spec §1.1):** taxonomy JSON (sheets→cats→subcats; incomeCategories→blocks)
  + `expenses_log.jsonl` entries (date `DD/MM`, no year — `--year` supplies it); unknown
  subcategory → warn+skip exit 0; taxonomy wins on category mismatch.
- **Next:** merge PR for `refactor/internal-taxonomy`; then (awaiting go-ahead) implement Plan A
  (T-05) → Plan B (T-04) → classifier full-path label (later, 5.R4/RUI-4); T-03 year-rollover;
  then TF-IDF (5.R1). **No implementation until the user says so.**
- **Cross-repo:** LLM infra at `/mnt/i/workspaces/llm/` — personas, MCP server, platform docs
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
- **Bare-name routing kept but guarded** — a bare name shared by >1 full path is *ambiguous* and
  dropped from routing (entry warn+skip, exit 0; never a silent misroute). `registerTarget` keeps
  the ambiguous set sticky (guards the 3× re-add trap); full paths join on a null byte (names
  contain `/`, e.g. `Uber/Taxi`).
- **T-02 reframed** — NO export command/writer; the real taxonomy is a one-shot hand-authored file
  (`config/taxonomy.json`, 112 subs, gitignored). Long-term direction is DB ingestion, not workbook
  insertion.

### Type Persistence & Full-Path Routing (this session — Plans A & B, NOT yet implemented)
- **Option 1 chosen** — the classifier emits the FULL PATH (type/category/subcategory) as its label /
  struct identity, not a bare name + ambiguity-only sheet. Cleaner; re-training data exists (607/694
  training examples carry the type in `source`; reviewed entries carry it). Tracked 5.R4 / RUI-4.
- **"Expense Sheet" → "Expense Type"** domain rename WITH JSON key migration (`sheet`→`type`,
  `sheets`→`types`); "sheet" stays reserved for Excel worksheet addressing only
  (`models.SheetLocation.SheetName`, `internal/inspect`, `sheetOrder`). Plan A.
- **Type is currently dropped at every log-write layer** — captured in review + apply (workbook
  insertion) but absent from `expenses_log.jsonl`, `classifications.jsonl`, and the 7-field
  classified CSV. This is a retraining data loss. Plan A Phase F captures it; B-fill backfills
  (partial — only reviewed entries carry the type).
- **Routing is two-tier, guard is PERMANENT (T-04, Plan B)** — typed entries route by full-path key
  (`expensePath`); type-less entries (auto.go:172 + batch_auto.go:296 + ~355 existing log lines)
  fall back to the retained bare-name map with ambiguous-skip. The earlier "guard removal / DEFERRED"
  framing is RETRACTED — the guard is a fallback, not removed. Advisor-caught: deleting it would drop
  the auto-inserted majority.
- **Sequencing** — merge branch → Plan A (T-05) Phase F + recover → Plan B (T-04) → classifier
  full-path label (later). No impl until user authorizes.

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
- **Workbook write out of scope for `review`** — UI emits `reviewed.json`; a separate
  future `apply` command (RUI-3) ingests it into workbook + feedback logs.
- **Review UI is the only type producer (this session)** — the type/sheet choice is made in the
  review page and exported in `reviewed.json`; saved corrections live in localStorage and are
  recoverable by re-export (no page change needed for Plan A backfill).
- **Taxonomy source** — workbook's "Referência de Categorias" sheet via
  `excel.LoadReferenceSheet`, grouped sheet→category→subcategory at runtime.

### Classification Strategy
- **Model candidates:** `my-classifier-q3` (Qwen3-8B) vs Qwen2.5-Coder-7B (speed). Benchmark deferred.
- **Confidence threshold:** HIGH ≥ 0.85 (auto-insert), LOW < 0.85 (print candidates + ⚠ signal)
- **Feature dictionary pre-filter:** skipped in 5.2; deferred to 5.7 (few-shot injection task)
- **Few-shot injection (5.7):** implemented — keyword layer (layer 1 of 3-layer cascade) complete; SelectExamples in `internal/classifier/examples.go`; loaders in `loader.go`; injected as user/assistant pairs in buildRequest; TF-IDF/embeddings deferred to future sessions
- **`expenses_log.jsonl`** — slim insert log (`id`, `item`, `date`, `value`, `subcategory`, `category`, `timestamp`); separate from `classifications.jsonl`; ID is sha256[:12] shared across both files for cross-file correlation. NOTE: `date` is `DD/MM` (no year). Plan A adds an optional `type` field.

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
- **`my-go-qcoder` first benchmark (session 23, 2026-05-18):** Used for `cmd/review.go` (verdict 1), `render_test.go` (verdict 2), `taxonomy_test.go` (verdict 2), `queue_test.go` (verdict 1). Struggled with intermediate Go map types in a prior session (verdict 0 on `taxonomy.go`) — passes cleanly when types are pre-defined in context files. Test generation is its strongest use; cobra command wiring is solid. Preferred over `my-go-q25c14` going forward for single-file codegen tasks. (Session T-02: verdict 0 on a loader edit with subtle algorithm + data-aware separator — wrote from scratch; conceptual/data-shaped defects remain a weak spot.)
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
| **Merge `refactor/internal-taxonomy` (START HERE)** | the open PR; `expense-reporter/internal/taxonomy/.memories/QUICK.md`; `.claude/plans/taxonomy-identity-key.md` | T-02 work (render-config relocation → package extraction → real `taxonomy.json` → full-path identity key). All green; CSV↔JSON fidelity verified (symmetric-difference empty), oracle dumps unchanged. Review + merge. |
| **T-05 persist expense type (Plan A — do AFTER merge)** | `.claude/plans/persist-expense-type.md` (full step list); `internal/feedback/feedback.go` + `expense_log.go`; `internal/apply/types.go`; `internal/review/template/review.html` (export) | Persist the expense type end-to-end + rename ExpenseSheet→ExpenseType + migrate JSON keys (`sheet`→`type`) + partial backfill. Phase F first (add `type,omitempty`, legacy-`sheet` UnmarshalJSON fallback). Hard prereq for Plan B/T-04. Advisor-reviewed. No impl until user authorizes. |
| **T-04 full-path entry routing (Plan B — AFTER Plan A)** | `.claude/plans/full-path-entry-routing.md` (full step list); `[ref:taxonomy-identity-key]`; `internal/taxonomy/loader.go` (`buildSubcategoryMap`/`scanEntries`); fixtures `test/fixtures/generate-basic/` | Two-tier routing: typed entries → full-path key; type-less → retained bare-name map + ambiguous-skip. The guard is a PERMANENT fallback (not removed) — type-less entries are the auto-inserted majority. Keep a typed/type-less fixture MIX. Advisor-reviewed. |
| Year-rollover workflow (T-03) | spec §1.1 + `internal/generate/.memories/QUICK.md`; `.claude/plans/workbook-generator-implementation-plan.md` §4 | Generate year N+1 from taxonomy alone (skeleton); decide fate of `apply`/`add` against generated workbooks. Real `config/taxonomy.json` (gitignored) is the input. |
| Classifier full-path label (5.R4 / RUI-4) | `internal/excel/reader.go` `LoadReferenceSheet`; `internal/models/`; `cmd/expense-reporter/cmd/classify.go`; `.claude/plans/persist-expense-type.md` follow-on | Make the classifier emit the type for *auto* entries (option 1, full-path label) using type-labeled training data + backfilled gold corrections. Out of scope of Plans A/B. |
| 5.R1 (TF-IDF layer) | `project_r1_evaluation_procedure.md` memory; `data/classification/research_insights.md` | Instrumentation prerequisite still open |
<!-- /ref:session-reading-guide -->
