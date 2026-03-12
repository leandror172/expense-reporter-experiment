# CLAUDE.md — Expense Reporter

This file provides guidance to Claude Code when working in this repository.

<!-- overlay:ref-indexing v1 -->
## Reference Indexing Convention

Rules in this file may include `[ref:KEY]` tags pointing to detailed reference material
stored as `<!-- ref:KEY -->` blocks in `*.md` files.

**To look up a ref:** `.claude/tools/ref-lookup.sh KEY` — prints that section.
Run with no args to list all known keys.
**To check integrity:** `.claude/tools/check-ref-integrity.py` — finds broken `[ref:KEY]`
tags and malformed blocks across the repo.

### Two-Tier Notation

| Tier | Notation | When to Use | How to Resolve |
|------|----------|-------------|----------------|
| **Active reference** | `[ref:KEY]` | Agent needs this content during work | `ref-lookup.sh KEY` |
| **Navigation pointer** | `§ "Heading"` | Background reading, archive, rationale | Open the file, find the heading |

Use `ref:KEY` for content agents need at runtime. Use `§ "Heading"` for background or
archive navigation. Do not use `ref:KEY` for content that is only occasionally needed.

### Hard Requirements When Modifying Files

1. **New ref blocks** — wrap with `<!-- ref:KEY -->` / `<!-- /ref:KEY -->`; one concept
   per block; never wrap an entire file in one block
2. **New `[ref:KEY]` tag in CLAUDE.md** — add a corresponding block somewhere in `*.md`
3. **New scripts/tools** — add to `.claude/index.md` under the scripts/tools table
4. **New files of any kind** — add to `.claude/index.md` under the appropriate table

The full indexing convention (examples, block format, § pointer usage) is documented in
`.claude/index.md` under the "Indexing Conventions" section.
<!-- /overlay:ref-indexing -->
## Project Identity

**Go CLI expense reporter** — reads CSV expense exports, classifies them, and inserts rows into an Excel workbook.

**Layer 5 goal:** Auto-classifier that runs local Ollama models to categorize expenses with confidence scoring, then auto-inserts high-confidence entries into the workbook.

**Scope:** Feature work (classify command, auto command, batch-auto command) lives here.
The MCP thin wrapper (5.8) lives in the LLM infrastructure repo:
`/mnt/i/workspaces/llm/` — see `.claude/session-context.md` for cross-repo context.

## Codebase Structure

```
expense-reporter/          ← Go module root (module: expense-reporter)
├── cmd/expense-reporter/
│   ├── main.go            ← Entry point
│   └── cmd/               ← Cobra subcommands (add, batch, classify, auto, batch-auto)
├── internal/              ← Private packages
│   ├── batch/             ← Batch import logic
│   ├── cli/               ← CLI helpers
│   ├── excel/             ← Excel workbook read/write (excelize)
│   ├── logger/            ← Logging
│   ← models/             ← Domain structs (Expense, Category, etc.)
│   ├── parser/            ← CSV parsing
│   ├── resolver/          ← Category resolution
│   └── workflow/          ← Multi-step orchestration
├── pkg/utils/             ← Public utility functions
└── config/                ← Configuration files

data/classification/       ← Classification algorithm docs + training data
  (tracked: *.md, algorithm_parameters.json)
  (gitignored: *.json training data, *.csv personal data)

docs/archive/              ← Desktop-era planning docs (read-only history)
.claude/                   ← Claude Code scaffolding (session tracking, tools)
```

[ref:go-structure] — detailed package table

## Build & Test Commands

```bash
cd expense-reporter && go build ./...        # Build all packages
cd expense-reporter && go test ./...         # Run all tests (190+ tests)
cd expense-reporter && go vet ./...          # Lint
```

No Makefile magic needed for normal work. Tests use standard `go test` with table-driven patterns.

[ref:testing] — test conventions

## WORKFLOW RULES (HARD REQUIREMENTS)

1. **DO NOT proceed to the next phase automatically** — Always wait for explicit user permission
2. **Step-by-step configuration** — Build config files incrementally, explaining each setting
3. **Explanatory mode active** — Use "Explanatory" output style with Insight boxes
4. **Licensing compliance** — When using or referencing external code/projects, check and honor their license. Track attributions in `docs/ATTRIBUTIONS.md`.

## Troubleshooting Approach

1. **Ask what's been tried** before suggesting solutions
2. **Check prior context** — read session logs and handoff files first
3. **Propose before executing** — explain intent for diagnostic commands with side effects

## Environment Context

- **Claude Code runs in:** WSL2 (Ubuntu-22.04) natively
- **Go version:** 1.25.5
- **Ollama endpoint:** `http://localhost:11434` — use `/api/chat` with `stream: false`, not CLI
- **sudo commands:** Cannot run through Claude Code. Ask the user.
- **LLM repo:** `/mnt/i/workspaces/llm/` — platform docs, personas, MCP server

## Local Model Usage (Layer 5+)

**Default: always try local model first for any code generation task.**
If the first call gets a REJECTED verdict, and there is a next model in the tier list
(below), try that model before escalating.
Escalate to Claude only after a second REJECTED verdict, or when the task explicitly requires
architectural reasoning across 3+ files simultaneously.

**Local model tier list for Go codegen** (benchmark in progress — use in priority order):
1. `my-go-qcoder` (qwen3-coder:30b, 32K ctx, agentic) — primary candidate, not yet benchmarked
2. `my-go-q25c14` (qwen2.5-coder:14b) — current proven baseline, ~25-32s, ~800 token budget
3. `my-go-q35-27b` (qwen3.5:27b) — benchmark candidate vs 14B baseline
4. `my-go-q35` (qwen3.5:9b) — VRAM-only, fastest, for simple tasks

**Local model tier list for classification** (for `classify`/`auto` commands — same cascade rule: try next on REJECTED):
1. `my-classifier-qcoder` (qwen3-coder:30b, 32K ctx) — primary; required for 5.7+ (few-shot injection)
2. `my-classifier-q35` (qwen3.5:9b) — VRAM-only, fast, for standard classification
3. `my-classifier-q3` (qwen3:8b) — proven baseline

**Use `mcp__ollama-bridge__generate_code` or `mcp__ollama-bridge__ask_ollama` for:**
- Go structs, interfaces, functions, test stubs, new functions
- Cobra command scaffolding
- Transformations (parsing, formatting, serialization)
- Single-file feature additions (e.g. batch-auto, correction logging, persistence)
- Any task where expected output is a bounded Go file or function
- Send necessary context - a list of files and the specific lines can be sent through the MCP call

**After receiving local model output, evaluate it explicitly:**
- `ACCEPTED` — used as-is (note the prompt that worked)
- `IMPROVED` — used with modifications (note what changed and why)
- `REJECTED` — not usable (note the failure reason: logic error / wrong API / off-task); try next model in tier before escalating

**On IMPROVED verdicts — when to fix inline vs. call Ollama again:**
- ≤3 line fixes (wrong var name, missing nil check, unused import) → fix inline, no second call
- Ollama missed entire test cases or whole functions → write stub signatures first, then call Ollama with the stub file as `context_files` and list the missing method names in the prompt; second call mainly pays off when bodies are non-trivial (table-driven, many edge cases)
- Deeper policy discussion deferred to LLM repo: `.claude/tasks.md` § "Deferred Infrastructure / Tooling"

**On ACCEPTED or IMPROVED verdicts, add a rough token estimate — do NOT read files or write code to compute it:**
- Mentally apply `(chars in your prompt + chars in response) / 4` as a ballpark of what Claude would have spent
- Note it inline in one phrase, e.g.: `ACCEPTED — ~300 est. Claude tokens saved`
- Rough is fine; the log records exact values automatically (`claude_tokens_est`, `prompt_eval_count`, `eval_count`) for later analysis

**Escalate to Claude (frontier) only when:**
- Second model in tier also returned REJECTED
- The task requires reasoning across 4+ files simultaneously

## Documentation Rules (HARD REQUIREMENTS)

When creating or modifying files:
1. **New scripts/tools** — add to `[ref:bash-wrappers]` in `.claude/index.md`
2. **New runtime-relevant doc sections** — wrap with `<!-- ref:KEY -->` / `<!-- /ref:KEY -->` blocks; one concept per block
3. **New files of any kind** — add to `.claude/index.md` under the appropriate table
4. **§ vs ref:KEY** — use `ref:KEY` for content agents need at runtime; use `§ "Heading"` for background/archive pointers

[ref:indexing-convention]

## Acceptance Testing

File-driven BDD harness in `expense-reporter/test/`. Build tag: `//go:build acceptance`.
Run: `./run-acceptance.sh` or `go test -tags=acceptance ./test/...`.
Whenever creating/updating acceptance test (or any test, including unit tests), give a complete rundown of the behavior of the tests in question
When updating, describe the previous behavior, and the new one, and the reason for the change

[ref:acceptance-patterns] — effort classification + README ref index for new scenarios
§ "Harness Architecture" in `expense-reporter/test/README.md` — full harness design

## Git Safety Protocol

For destructive git operations: explain → backup → dry-run → execute → verify.

## Key Domain Facts

- **Date format:** DD/MM/YYYY (Brazilian format, e.g., `15/03/2025`)
- **Decimal separator:** comma (e.g., `1.234,56` = one thousand two hundred thirty-four reais and 56 centavos)
- **Year context:** 2025 expense data (historical); 2026 is current year
- **Currency:** BRL (Brazilian Real)
- **Categories:** hierarchical (Category → Subcategory). See `data/classification/` for taxonomy.
- **Sensitive data:** Training JSONs contain real expense descriptions — always gitignore, never commit

## Resuming Multi-Session Work

**On session start:** run `.claude/tools/resume.sh` — outputs current status, next task, recent commits in ~40 lines.

For deeper context:
- `ref-lookup.sh current-status` — active layer, next task
- `ref-lookup.sh active-decisions` — key architectural choices
- `.claude/index.md` — map of all files

**Knowledge index:** `.claude/index.md`
**Sensitive data:** `.claude/local/` (gitignored)
