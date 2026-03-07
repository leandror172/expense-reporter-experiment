# Session Log — Expense Reporter

Most recent entry first. Run `.claude/tools/rotate-session-log.sh` when this grows beyond ~3 sessions.

---

## 2026-03-03 — Session 4: Debt fix + acceptance test harness & batch-auto design

### Context
Resumed from session 3. User asked to recontextualize, address tech debt, then discuss
design for both an acceptance test harness and the 5.4 batch-auto command.

### What Was Done
- **Fixed `runtime.Caller` debt** (commit `405953e`): replaced `runtime.Caller(0)` with
  `os.Executable()` in `internal/config/config.go` — config path now resolves relative to
  the installed binary, not the compile-time source path.
- **Designed acceptance test harness** — BDD-style Given/When/Then with function injection:
  - Domain-agnostic `test/harness/` package (Context, Scenario, Run, fixture loader, CSV comparator, Ollama check)
  - Domain-specific `test/actions/` (command runners) + `test/verify/` (composable assertions)
  - `//go:build acceptance` tag, `run-acceptance.sh` wrapper, `test/results/` for drift tracking
  - Dir-per-fixture with representative batches, soft vs hard assertions for non-deterministic outputs
- **Designed batch-auto command (5.4)** — 3-field CSV input, classify via Ollama, split to
  classified.csv + review.csv, dry-run flag, continue-on-error, workbook backup before insert
- **Evaluated testscript** (rogpeppe/go-internal) — decided against: script DSL conflicts with
  user's function-injection BDD vision. Custom harness on stdlib is more aligned.
- **Evaluated testify** — decided against: project uses stdlib `testing` throughout, no value added.
- **Created comprehensive implementation plan:** `.claude/plans/acceptance-harness-batch-auto.md`
- **Design decisions for shared code extraction:**
  - `isAutoInsertable` → `internal/classifier/decision.go` (shared by auto + batch-auto)
  - `buildInsertString`/`formatBRValue` → `pkg/utils/format.go` (alongside existing currency/date utils)

### Decisions Made
- **No testscript, no testify** — custom harness on stdlib, consistent with existing 190+ tests
- **Dir-per-fixture** with representative batches (10-20 rows), not single-row scenarios
- **Continue-on-error (option b)** for Ollama failures mid-batch — failed rows get confidence=0, go to review
- **Dry-run flag** for batch-auto — classify + write CSVs, skip workbook insertion
- **Multiple test files per command** in `package acceptance_test` (setup_test.go + per-command files)
- **Local model candidates** for bounded Go tasks during implementation (decision.go, scenario.go, etc.)

### Next
- [ ] Implement plan from `.claude/plans/acceptance-harness-batch-auto.md`:
  - Phase 1: Extract shared logic (decision.go, format.go)
  - Phase 2: Acceptance test harness skeleton + initial fixtures
  - Phase 3: batch-auto command (5.4)
  - Phase 4: Polish (run-acceptance.sh, drift tracking, docs)

---

## 2026-03-02 — Session 3: Integration testing + Diversos auto-insert fix

### Context
Continuation of session 2 (same day). After the earlier handoff, user asked to run live test
cases against real Ollama using `Planilha_Normalized_Final_copy.xlsx`.

### What Was Done
- **Integration testing (4 cases):**
  - "Diarista Letícia" 160 05/01 → 95% Diarista ✓ (--confirm, user declined)
  - "Uber Centro" 35.50 15/04 → 100% Uber/Taxi ✓ (--confirm, user declined)
  - "VA compras" 85.00 10/03 → 100% Supermercado VA — auto-inserted into workbook copy
    (finding: LLM resolves multi-word ambiguity better than keyword specificity alone; "va"
    has specificity=0.36 in feature dict but "VA compras" was unambiguous to the model)
  - "TechCorp SaaS assinatura" 49.90 01/03 → 95% Diversos — auto-inserted (**bug found**)
- **Fix: configurable auto-insert exclusion list** (commit `1ac43dd`):
  - `internal/config/config.go`: `Config` struct + `Load()` reads `config/config.json`
    (known debt: uses `runtime.Caller` for path resolution; should use `os.Executable`)
  - `config/config.json`: added `"auto_insert_excluded": ["Diversos"]`
  - `cmd/auto.go`: extracted `isAutoInsertable(result, excluded []string) bool`; distinct ⚠
    messages for threshold vs exclusion rejection
  - `cmd/auto_test.go`: 9 tests for `isAutoInsertable` including empty-exclusion-list case
- **Deferred items logged:**
  - `tasks.md` (this repo): `runtime.Caller → os.Executable` config reader debt
  - LLM `tasks.md`: `ollama-bridge file_path input` (token efficiency gap) + same config debt
- **Local model verdict:** `my-go-q25c14` used for test update — IMPROVED (wrong package name
  `expense_reporter_test` → `cmd`; syntax typo `0,.95` → `0.95`; structure was correct)

### Decisions Made
- **Exclusion list in config.json:** Not hardcoded — configurable so users can add subcategories
  without recompiling. Empty list = no exclusions (Diversos would pass through).
- **Distinct ⚠ messages:** "below threshold" vs "excluded from auto-insert" — different root
  causes deserve different messages.
- **runtime.Caller acknowledged as debt:** Logged and deferred; not blocking for development use.
- **ollama-bridge file_path:** Token efficiency gap identified — file content must pass through
  Claude context twice (read + embed in prompt). Logged in LLM tasks for future bridge enhancement.

### Next
- [ ] 5.4 — `batch-auto` command: classify a CSV, write `classified.csv` (HIGH) + `review.csv` (LOW)
- [ ] Consider: `Transporte` appearing as subcategory at 90% in case 2 — taxonomy oddity, not urgent

---

## 2026-03-02 — Session 2: Layer 5 tasks 5.1–5.3

### Context
First active feature session. Resumed from session 1 (scaffolding). Started with recontextualization
(memory + resume.sh + ref_lookup), then proceeded with 5.1 → 5.2 → 5.3 in order.

### What Was Done
- **5.1 (docs):** Added three ref blocks to `.claude/index.md`:
  `ref:training-data-schema`, `ref:confidence-thresholds`, `ref:classification-overview`.
  All three were already referenced from `ref:classification` but had no content.
- **5.2 (classify command):** `internal/classifier/` package with `Classify()`, `LoadTaxonomy()`,
  `buildSystemPrompt()`; calls Ollama `/api/chat` with structured JSON format param.
  `cmd/classify.go` — 3 positional args (item, value, DD/MM), `--model`, `--top`, `--data-dir` flags.
  11 tests covering LoadTaxonomy, buildSystemPrompt, and Classify via httptest mock server.
- **5.3 (auto command):** `cmd/auto.go` — classify + auto-insert if confidence ≥ 0.85;
  `--confirm` flag prompts before inserting; `⚠ Not inserted` signal on low confidence; exit 0 always.
  Tests for `formatBRValue`, `buildInsertString`, `confirmInsert` (y/Y/yes + n/N/no/empty).
- **classify fix:** Swapped `strconv.ParseFloat` → `utils.ParseCurrency` so both `35.50` and `35,50` accepted.
- **LLM repo notes:** Added session-37 entry to `session-log.md` and deferred task for `ref_lookup`
  cross-repo support to `tasks.md` in `/mnt/i/workspaces/llm/`.
- **Branch:** `feature/layer5-classifier` (3 commits: b4e4c61, d623cd7, bd8aebe)

### Decisions Made
- **classify input format:** Positional args (`classify "item" value DD/MM`), not semicolon string.
  Chosen for CLI idiom and standard float; `utils.ParseCurrency` added to accept both `.` and `,`.
- **auto exit code:** Always exit 0 on successful run; non-zero only on actual errors. Signal via stdout `⚠`.
- **auto --confirm:** Prompts even on HIGH confidence when flag is set; default no-insert on empty/non-y input.
- **Feature dictionary in 5.2:** Skipped — 5.2 is pure LLM path; pre-filter optimization deferred to 5.7.
- **Local model use:** Cobra command scaffold generated with `my-go-q25c14` (verdict: IMPROVED —
  dropped spurious date parsing and context arg; structure and flag registration were correct).
- **TDD note:** Tests were written after implementation for 5.2 (not red-first); corrected for 5.3.

### Next
- [ ] 5.4 — `batch-auto` command: classify a CSV, write `classified.csv` (HIGH) + `review.csv` (LOW)
- [ ] 5.5 — Correction logging: `corrections.jsonl`
- [ ] Update tasks.md in this repo to mark 5.1–5.3 complete

---

## 2026-02-27 — Session 1 (Claude Code bootstrap)

### Pre-history (Claude Desktop, sessions 1–N)

All work before this session was done in Claude Desktop (separate context, no session tracking).

**What was built (Phases 1–11 in Desktop):**
- Phase 1-2: Domain model — `Expense`, `SubCategory`, `Category` structs; CSV parser (DD/MM/YYYY, comma decimals)
- Phase 3-4: Excel writer — inserts expense row into workbook using excelize; auto-detects sheet
- Phase 5-6: Category resolver — keyword matching against category/subcategory taxonomy
- Phase 7-8: Batch import — process CSV file, report successes/failures
- Phase 9-11: Cobra CLI — `add` (single expense), `batch` (CSV import), `version` commands; 190+ tests
- v2.1.0 reached: full CLI working, all tests passing, Excel integration verified

**Classification work (separate Claude Desktop context):**
- Auto-categorization analysis of ~N historical expenses using LLM-assisted feature extraction
- Feature dictionary (keyword → category mapping), training data, similarity matrices
- Results in `~/workspaces/expenses/auto-category/` (original); copied to `data/classification/`

### Done this session
- Bootstrap `.claude/` scaffolding in this repo (tools, skills, CLAUDE.md, index.md, session-log.md, session-context.md, tasks.md)
- Migrated classification docs from `~/workspaces/expenses/auto-category/` → `data/classification/`
- Moved Desktop-era planning docs to `docs/archive/`
- Set up `.gitignore` for personal data files
- Confirmed: `cd expense-reporter && go test ./...` still passes

### Key decisions
- Layer 5 feature work (classify/auto/batch-auto commands) happens in this repo
- MCP thin wrapper (5.8) stays in LLM repo (`/mnt/i/workspaces/llm/`)
- Training data (personal expense data) gitignored; docs-only tracked in `data/classification/`
- `confusion_analysis.json` gitignored (may contain real expense descriptions)

### Blockers
- None

### Next
- [ ] 5.1 — Port training data: confirm `feature_dictionary_enhanced.json` + `training_data_complete.json` are in `data/classification/` and document their format
- [ ] 5.2 — `classify` command: 3-field input → Ollama HTTP → structured JSON → top-N subcategories with confidence
- [ ] 5.3 — `auto` command: classify + insert if HIGH confidence (≥0.85)

---
