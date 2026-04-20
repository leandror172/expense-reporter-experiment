# Session Log — Expense Reporter

**Previous logs:** `.claude/archive/session-log-2026-02-27-to-2026-02-27.md`, `.claude/archive/session-log-2026-03-02-to-2026-03-02.md`, `.claude/archive/session-log-2026-03-13-to-2026-03-02.md`, `.claude/archive/session-log-2026-03-03-to-2026-03-03.md`, `.claude/archive/session-log-2026-03-11-to-2026-03-11.md`, `.claude/archive/session-log-2026-03-13-to-2026-03-13.md`, `.claude/archive/session-log-2026-03-13-to-2026-03-13.md`, `.claude/archive/session-log-2026-03-14-to-2026-03-14.md`, `.claude/archive/session-log-2026-03-18-to-2026-03-18.md`
Most recent entry first. Run `.claude/tools/rotate-session-log.sh` when this grows beyond ~3 sessions.

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

## 2026-03-23 — Session 12: 5.8a — JSON output for classify/auto

### Context
Resumed after 5.7 merge to master. Recontextualized via `resume.sh`. Discussed 5.8 architecture extensively — originally planned as MCP thin wrapper in LLM repo, but decided the MCP server belongs in this repo since it's a tool of this app. Reviewed the grand vision (`docs/expense-classifier-vision.md`) which reshaped 5.8 into two sub-tasks: 5.8a (Go-side `--json` flag) and 5.8b (Python MCP wrapper). This session implemented 5.8a.

### What Was Done
- **5.7 merged** — user merged `feature/5.7-few-shot-injection` to master before session work began
- **5.8 architecture discussion** — read LLM repo's MCP server (`server.py`, `registry.py`), all Cobra commands, and concurrency control docs; decided MCP server lives in expense-reporter repo, not LLM repo
- **5.8a implementation** — `--json` persistent flag on root command; new `output.go` with `ClassifyOutput`, `CandidateOutput`, `AutoOutput` structs + `printJSON`/`toCandidates` helpers
- **classify --json** — outputs structured JSON with item/value/date/candidates array
- **auto --json** — read-only mode: classifies, evaluates `IsAutoInsertable`, returns recommendation (`would_insert`/`review`/`excluded`) but **never inserts** into workbook. Matches vision flow: classify → user picks → insert
- **Unit tests** — `output_test.go`: 5 tests (toCandidates, ClassifyOutput serialization, AutoOutput serialization, nil Result omitempty, printJSON stdout capture)
- **Acceptance tests** — `test/json_output_test.go`: 2 scenarios (classify --json valid JSON with keys, auto --json returns recommendation without inserting); `test/verify/json.go`: `OutputIsValidJSON()`, `OutputJSONHasKey()` verify helpers
- **All tests pass** — 18 acceptance (16 existing + 2 new), 220+ unit tests, clean build + vet
- **Deferred task T1** — resume context loading improvement documented in `.claude/ideas/resume-context-loading.md`
- **Ollama usage** — used `my-go-qcoder` (qwen3-coder:30b) for `output.go` and `output_test.go` generation; identified gap where later files were written without Ollama (feedback saved to memory)

### Decisions Made
- **MCP server lives in expense-reporter repo** — it's a tool of this app, not generic LLM infra; uses Python/FastMCP (same stack as LLM repo's ollama-bridge)
- **5.8 split into 5.8a + 5.8b** — vision doc showed MCP is one of several consumers (Telegram next); Go binary `--json` is the real API surface, MCP wrapper becomes trivially thin
- **auto --json never inserts** — matches vision flow (classify → present → user picks → add); action field is `would_insert` (recommendation), not `inserted`
- **`--json` is persistent on root** — future commands (batch-auto, add) can adopt without plumbing changes
- **TDD violation acknowledged** — tests written after implementation instead of red-first; noted for future sessions

### 5.8b Design Decisions (discussed, not yet implemented)
- **Three tools:** `classify_expense` (candidates only), `add_expense` (insert), `auto_add` (candidates + recommendation) — all three kept; auto_add is superset of classify but both useful
- **Thin wrapper:** ~150 lines server.py, no OllamaClient/registry/persona reuse from ollama-bridge
- **Binary path:** `EXPENSE_REPORTER_BIN` env var primary, fallback builds from Go module root
- **Workbook path:** MCP param forwarded as `--workbook` (not hidden, not env-resolved by MCP)
- **Exposed params:** `model` and `top` only (caller-meaningful); `data_dir` hidden (infrastructure)
- **Testing:** Integration tests (build binary, call with `--json`); minimal; use `--dry-run` for add
- **5.8b-prep task:** Add `--dry-run` flag to Go `add` command before building MCP server

### Next
- [ ] **Merge `feature/5.8a-json-output`** to master
- [ ] **5.8b-prep:** Add `--dry-run` to `add` command (Go)
- [ ] **5.8b:** Python MCP server in `mcp-server/` — `classify_expense`, `add_expense`, `auto_add`

---

## 2026-03-18 — Session 11: 5.7 implementation — few-shot injection

### Context
Resumed immediately after Session 10 (planning session). Plan at `.claude/plans/5.7-few-shot-injection.md` was approved as-is. Full implementation in one session using local Ollama models (qwen3-coder:30b) for code generation.

### What Was Done
- **Phase 1 — example selection engine** (`internal/classifier/examples.go`): `Example`, `ExampleSource`, `KeywordEntry`, `KeywordIndex` types; `SelectExamples` (keyword lookup → high/ambiguous specificity branching → source priority sort); `tokenize` (unicode-aware, accents preserved)
- **Phase 2 — data loading** (`internal/classifier/loader.go`): `LoadTrainingExamples` (YYYY-MM-DD → DD/MM normalization, nil,nil on missing), `LoadFeedbackExamples` (JSONL, skips manual entries), `LoadKeywordIndex`, `MergeExamplePools` (feedback wins on item dedup)
- **Phase 3 — prompt wiring** (`classifier.go`): `buildRequest` accepts `[]Example`; `formatExampleMessages` produces user/assistant pairs (confidence 0.95); `Classify` orchestrates loading from `cfg.DataDir`/`cfg.FeedbackPath`, logs via `logger.Debug("few-shot", ...)`; `Config` gains `DataDir` + `FeedbackPath`; all three commands wired
- **Phase 3 unit tests** — `TestFormatExampleMessages` + `TestBuildRequest_FewShot` added to existing `classifier_test.go` (testify; no new file — right call)
- **Phase 4 — acceptance tests** (`test/fewshot_test.go`): 3 scenarios — training data present, training data absent (graceful degradation), batch-auto unchanged; `RunClassify` updated to auto-inject `--data-dir`
- **Phase 5** — all 16 acceptance tests pass (~15 min total); all 220+ unit tests pass
- **Phase 6** — `tasks.md` (5.7 → done), `session-context.md`, `index.md` updated
- **Tooling** — replaced `.claude/tools/ref-lookup.sh` bash calls with `mcp__ollama-bridge__ref_lookup` for doc lookups

### Decisions Made
- **`FeedbackPath` field in `classifier.Config`** — `classify` command only loads training data; `auto`/`batch-auto` pass `appCfg.ClassificationsFilePath()` as `FeedbackPath`
- **No new test file for `buildRequest` tests** — added to existing `classifier_test.go`; splitting was only justified by style mixing, not domain/size
- **Ollama timeouts = cold start** — truly unavailable Ollama fails immediately (connection refused); retry more aggressively before falling back to writing code manually

### Next
- [ ] **Merge `feature/5.7-few-shot-injection`** to master (PR open)
- [ ] **Start 5.8** — MCP thin wrapper in LLM repo (`/mnt/i/workspaces/llm/`): 3 tools `classify_expense`, `add_expense`, `auto_add` calling Go binary as subprocess

---

