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
- **Preferred model:** `my-go-q25c14` (qwen2.5-coder:14b via ollama-bridge MCP)
- **Speed vs cost:** 30s for correct 14B output beats 6s for wrong 8B output. Local inference cost is latency only.
- **Verdict pattern:** Always record ACCEPTED / IMPROVED / REJECTED after local model output

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
- **Active layer:** Layer 5 — Expense Classifier (first Claude Code session)
- **Last checkpoint:** Session 1 (2026-02-27) — scaffolding bootstrap complete
  - `.claude/` tools, CLAUDE.md, index.md, session-log.md created
  - Classification docs migrated from `~/workspaces/expenses/auto-category/`
  - Desktop-era planning docs moved to `docs/archive/`
- **Branch:** `feature/claude-code-scaffolding`
- **Next:** Layer 5 task 5.1 (verify training data in `data/classification/`), 5.2 (`classify` command)
- **Cross-repo:** LLM infra at `/mnt/i/workspaces/llm/` — contains personas, MCP server, platform docs
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

### Domain Boundary (decided session 32 in LLM repo context)
- **Classification logic in expense-reporter (Go)** — it's a product feature, not LLM infrastructure
- **MCP thin wrapper in LLM repo** — 3 tools: `classify_expense`, `add_expense`, `auto_add`; calls Go binary as subprocess
- **Training data strategy:** hybrid — feature dictionary as system context + top-K few-shot examples per request
- **Structured output:** Ollama `format` param (proven reliable in LLM infra work)

### Classification Strategy
- **Model candidates:** `my-classifier-q3` (Qwen3-8B) vs Qwen2.5-Coder-7B (speed). Benchmark at 5.2.
- **Confidence threshold:** HIGH ≥ 0.85 (auto-insert), LOW < 0.85 (present candidates to user)
- **Few-shot injection (5.7):** keyword pre-match against training data, inject top-K examples into prompt

### Go Conventions
- **Cobra pattern:** Each subcommand is a `.go` file in `cmd/expense-reporter/cmd/`
- **Brazilian format:** DD/MM/YYYY dates, comma decimal separator (`1.234,56` notation)
- **Error pattern:** `fmt.Errorf("context: %w", err)` — wrap with context, not bare return
- **Table-driven tests:** Standard approach — any new command gets table-driven test coverage

### Classification Data
- `confusion_analysis.json` gitignored (may contain real expense descriptions as test cases)
- `algorithm_parameters.json` tracked (no personal data, pure algorithm config)
<!-- /ref:active-decisions -->
