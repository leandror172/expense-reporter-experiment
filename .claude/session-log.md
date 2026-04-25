# Session Log — Expense Reporter

**Previous logs:** `.claude/archive/session-log-2026-02-27-to-2026-02-27.md`, `.claude/archive/session-log-2026-03-02-to-2026-03-02.md`, `.claude/archive/session-log-2026-03-13-to-2026-03-02.md`, `.claude/archive/session-log-2026-03-03-to-2026-03-03.md`, `.claude/archive/session-log-2026-03-11-to-2026-03-11.md`, `.claude/archive/session-log-2026-03-13-to-2026-03-13.md`, `.claude/archive/session-log-2026-03-13-to-2026-03-13.md`, `.claude/archive/session-log-2026-03-14-to-2026-03-14.md`, `.claude/archive/session-log-2026-03-18-to-2026-03-18.md`, `.claude/archive/session-log-2026-03-18-to-2026-03-18.md`, `.claude/archive/session-log-2026-03-23-to-2026-03-23.md`
Most recent entry first. Run `.claude/tools/rotate-session-log.sh` when this grows beyond ~3 sessions.

---

## 2026-04-23 — Session 16: Planning MCP-Layer Corrections

### Context
Short planning session. User asked about the "Telegram-flow corrections at MCP layer" item from session 15's "Next" pointer. Read `docs/expense-classifier-vision.md` mid-discussion to ground the plan in the documented Phase 3/4 seam.

### What Was Done
- Discussed scope, signal shape, and semantics of closing the Telegram correction loop
- Confirmed MCP wrapper lives in **this repo** at `mcp-server/` (not in the llm repo — user corrected an earlier assumption)
- Discovered during scoping grep: feedback schema already supports `status: confirmed | corrected | manual`, `predicted_*` / `actual_*` fields, and `GenerateID`; `auto.go` and `batch_auto.go` already write `NewConfirmedEntry`; `correct.go` writes `NewCorrectedEntry`; only `add.go` is the gap (writes `NewManualEntry` with no prediction context)
- Produced a plan written to `docs/plans/mcp-layer-corrections.md` for execution next session

### Decisions Made
- **Signal shape (c):** both `--classification-id` AND `--predicted-subcategory` on `add` — ID for linkage per vision's lifecycle model, predicted-subcategory as authoritative signal that survives log rotation
- **Confirmation logging: deferred** (user choice 2.C) — but `auto`'s existing confirmed-writes keep working. Open question logged for execution: when `add --predicted-subcategory X` has `chosen == X`, write `confirmed` (lean: yes, consistency with `auto`) or skip.
- **Double-logging is fine:** `expenses_log.jsonl` (insert event) and `classifications.jsonl` (correction event) legitimately record different events, joined by shared ID; no double-count risk since few-shot reads only `classifications.jsonl` (verify at execution start)
- **Out of scope:** status lifecycle, SQLite migration, Telegram bot implementation, 5.R1 retrieval work

### Next
- Execute `docs/plans/mcp-layer-corrections.md` — start with Step 1 (verify no double-count risk) then acceptance tests
- Still open from session 15: PR #17 merge status vs. PR #16; gitignore cleanup for `expense-reporter` binary + `expenses_failed_*.csv`

---

## 2026-04-22 — Session 15: `correct` Command — Closing the Feedback Loop (Layer 5.9)

### Context
Resumed via `.claude/tools/resume.sh` from session 14's "Next" pointer. Discussed Option B (correction workflow) over Option A (TF-IDF retrieval) because the feedback loop being broken is the bigger value gap. Adopted "acceptance tests first" as a durable workflow rule.

### What Was Done
- **Acceptance-first workflow established** (saved to memory) — discuss scenarios, write acceptance tests, then drop into TDD inner loop for unit tests
- **Naming conventions documented in `expense-reporter/test/PATTERNS.md`**:
  - **Given:** Event Modeling style — past-tense events that happened (e.g., `expenseAutoConfirmed`, `expenseConfirmedThenCorrected`); state-only exception for empty event streams (e.g., `noClassificationsRecorded`)
  - **Then:** composable `[]func(*Context)` slices joined via `slices.Concat`
- **Test scaffolding additions:**
  - `verify.CommandFailed` (inverse of CommandSucceeded)
  - `harness.SeedFileFromFixture` (copy one named fixture file into WorkDir)
  - `actions.RunCorrect` (no `--workbook` flag; passes `--data-dir`)
- **Production code** (TDD: red acceptance → impl → green):
  - `feedback.NewCorrectedEntry` — predicted from prior, actual from user, status=corrected
  - `feedback.FindLatestEntry(path, id)` — scans JSONL, returns last matching entry
  - `cmd/correct.go` cobra command — parses 4-field input, looks up prior, writes corrected entry; fails clearly when no prior entry exists (hint to use `add`); does NOT touch the workbook
- **2 fixtures + 3 acceptance tests** — overrides-confirmed, fails-when-no-prior, latest-wins-on-duplicate-IDs (all 3 green in <1s, no Ollama needed)
- **Tooling:** `.claude/tools/lookup-category.py` — subcategory→category lookup against `feature_dictionary_enhanced.json`
- **Persona/tier-list update:** added `my-go-g3-12b` (gemma3:12b) at #2 in CLAUDE.md Go codegen tier list
- **Memory updates:**
  - New: `feedback_acceptance_first.md`
  - Sharpened `feedback_ollama_timeouts.md` — 3 parallel Ollama codegen calls is too much for non-trivial prompts (VRAM ceiling)
  - Updated `feedback_system_findings.md` — flipped "missing corrections" section to documented closed gap; added correct command to entry points
- **Doc sweep:** `expense-reporter/README.md`, root `README.md`, `expense-reporter/.memories/QUICK.md`, `test/.memories/QUICK.md`, `docs/FEEDBACK_SYSTEM.md` (renamed ref block `feedback-missing-feature` → `feedback-correction-workflow`); `.claude/index.md` ref pointer updated
- **Branch & PRs:**
  - Created `feature/correct-command` branch (off `docs/feedback-system-csv-reconstruction`)
  - 2 commits: `3ddd46f` (feat), `4b97c12` (docs)
  - **PR #17** opened to master: https://github.com/leandror172/expense-reporter-experiment/pull/17 — includes 2 commits from PR #16 (still open) as ancestors

### Decisions Made
- **`correct` is feedback-only** — no `--workbook` flag, does not move cells. Workbook cell relocation is bundled responsibility; keep `correct` single-purpose. User fixes the workbook manually for now.
- **`correct` requires a prior entry** — fails with hint to use `add` if none exists. Matches the design: corrections always override a model prediction. `review.csv` items (never auto-inserted) belong to `add`.
- **Telegram-flow corrections deferred to MCP/bot layer** — writing `corrected` at insert time when user picks a non-top candidate needs an MCP-layer extension (e.g., `--predicted-subcategory` flag on `add`); not part of CLI scope.
- **Gemma persona used as-is** (`my-go-g3-12b`, role differs from qcoder) rather than copied from qcoder — avoid spec-overwrite when an existing active persona works.
- **Ollama parallelization ceiling**: 3 parallel codegen calls only safe for tiny near-identical prompts; default to serial otherwise.

### Next
- Verify PR #17 in CI; merge order with PR #16 (rebase if #17 lands first)
- After merge: 5.R1 TF-IDF retrieval (better few-shot example selection) OR Telegram-flow `corrected` extension at MCP layer — user pick
- Consider gitignore cleanup for `expense-reporter/expense-reporter` binary and `expenses_failed_*.csv` artifacts (separate small commit)

---

## 2026-04-20 — Session 14: Feedback System Documentation & CSV Reconstruction Tool

### Context
Resumed on master branch. User extracted 1601 expense entries from large JSON file (ChatExport), hit batch-auto bug during classification workflow, then pivoted to understanding feedback system architecture and creating tooling to recover from such failures.

### What Was Done
- **Bug investigation & reporting** — batch-auto failed to write CSV output files when workbook insertion failed; loss of all classification work; logged to `BUG_REPORT.md`
- **Feedback system research** — discovered and documented that system CAN read `status=corrected` entries but nothing writes them (critical missing feature for learning loop)
- **Comprehensive documentation** (`docs/FEEDBACK_SYSTEM.md`):
  - 6 REF blocks: entry structure, command flows (add/auto/batch-auto), training integration, missing correction feature, file paths, cold-start behavior
  - Indexed in `.claude/index.md` for future reference via `ref-lookup.sh`
- **CSV reconstruction tool** (`.claude/tools/reconstruct-csvs.py`):
  - Parses batch-auto logs + original CSV (line-matched indexing)
  - Reconstructs `classified.csv` and `review.csv` from 373-line run (326 auto-inserted, 23 review, 24 skipped)
  - Efficient non-I/O approach (no reading full files into memory)
- **Personal memory** — saved feedback system findings to user memory for cross-session reference
- **Commits** — `docs: add feedback system architecture documentation` (22cdd33) on new branch `docs/feedback-system-csv-reconstruction`

### Decisions Made
- **Document findings instead of implementing** — feedback system is complex and worth understanding before adding features; created searchable reference for future work
- **Script-based recovery** — better to provide reconstruction tool than auto-save via side effects
- **REF-based documentation** — organized by concept (entry structure, flows, training, gaps) not by file

### What's Staged
- `.claude/tools/reconstruct-csvs.py` — CSV reconstruction from logs
- `docs/FEEDBACK_SYSTEM.md` — Feedback system architecture docs
- `.claude/index.md` — Updated tools table + feedback section
- `BUG_REPORT.md` — Bug report for workbook insertion failure

### Next
- Create PR for this branch (docs/feedback-system-csv-reconstruction → master)
- Possible future work: implement `NewCorrectedEntry()` + `correct` command to enable feedback loop closure
- Consider 5.R1 TF-IDF retrieval or surface-level feedback in review flow

---

## 2026-03-27 — Session 13: 5.8b — Python MCP server + dry-run + data-dir fix

### Context
Resumed from session 12. Recontextualized, opened PR for 5.8a, then worked through 5.8b-prep and 5.8b. Discussed whether to use Opus→Sonnet delegation pattern (decided not to for this size of task). Explored MCP tool interfaces against the grand vision, reducing 3 tools to 2 (`classify_expense` + `add_expense`). Used Ollama for code gen with explicit verdicts. Discovered and fixed latent `--data-dir` bug via live MCP testing.

### What Was Done
- **PR #11** — 5.8a JSON output merged to remote (push + PR)
- **5.8b-prep** (`feature/5.8b-prep-add-dry-run`, PR #12) — `--dry-run` flag on `add` command; early parse refactor; `AddOutput` struct; 5 unit tests + 1 acceptance test; `RunAddDryRun` action + `OutputJSONHasValue` verify helper
- **5.8b** (`feature/5.8b-mcp-server-impl`, PR #13) — new `mcp-server/` Python project (uv + FastMCP); `binary.py` (find_binary, run_binary, error types); `server.py` (classify_expense + add_expense tools); 7 integration tests; `run-server.sh`; registered with Claude Code
- **fix(add)** (`feature/5.8b-add-data-dir`, PR #14) — `--data-dir` flag on `add` command; MCP server passes it explicitly; category now resolves correctly in `add_expense` response; new acceptance test `TestAddDryRunJSON_ResolvesCategory`
- **End-to-end test** — MCP tools smoke-tested live; both tools confirmed working after restart

### Decisions Made
- **2 tools not 3** — dropped separate `classify_expense` (raw) vs `auto_add` (recommendation); `classify_expense` now maps to `auto --json` (candidates + recommendation); `add_expense` maps to `add --json`
- **`auto_add` name was misleading** — renamed to `classify_expense` since it never adds
- **Python justified in Go repo** — MCP server is thin subprocess wrapper, Go MCP SDK maturity unknown; Python + FastMCP is proven pattern
- **Opus→Sonnet delegation** — analyzed as not worth it for this task size; fits better for larger mechanical tasks (e.g. 5.R1 TF-IDF)
- **Ollama for code gen** — test file: REJECTED (truncation + wrong approach); server.py: IMPROVED (right shape, FastMCP API bugs); binary.py written by Claude directly (Ollama skipped)
- **`--data-dir` on `add`** — resolved latent bug where taxonomy lookup silently failed when called from MCP server (cwd mismatch); default preserved as `"data/classification"` for CLI use

### Next
- Merge PR chain: 5.8a → 5.8b-prep → 5.8b → 5.8b-add-data-dir → master
- Consider deferred task: `TestBatchAuto_SameYearInstallmentsExpanded` scope reduction (tasks.md)
- Next feature work: 5.R1 TF-IDF retrieval layer or T1 resume context loading

---

