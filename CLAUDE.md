# CLAUDE.md — Expense Reporter

This file provides guidance to Claude Code when working in this repository.

## Reference Lookup Convention

Rules in this file may include `[ref:KEY]` tags pointing to detailed reference material.
**To look up:** `.claude/tools/ref-lookup.sh KEY` — prints the referenced section. Run with no args to list all keys.

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

When implementing Layer 5 classifier features, **try local models first** for code generation.
This generates training data for future distillation.

**Use `mcp__ollama-bridge__generate_code` or `mcp__ollama-bridge__ask_ollama` for:**
- Go structs, interfaces, simple functions, test stubs
- Simple transformations (parsing, formatting, serialization)
- Cobra command scaffolding

**After receiving local model output, evaluate it explicitly:**
- `ACCEPTED` — used as-is (note the prompt that worked)
- `IMPROVED` — used with modifications (note what changed and why)
- `REJECTED` — not usable (note the failure reason: logic error / wrong API / off-task)

**Preferred local model:** `my-go-q25c14` (qwen2.5-coder:14b) — ~25-32s, ACCEPTED quality.

**Do NOT use local models for:**
- Architectural decisions or multi-file reasoning
- Security-sensitive code
- Tasks requiring understanding of large context (>400 tokens of output needed)

## Documentation Rules (HARD REQUIREMENTS)

When creating or modifying files:
1. **New scripts/tools** — add to `[ref:bash-wrappers]` in `.claude/index.md`
2. **New runtime-relevant doc sections** — wrap with `<!-- ref:KEY -->` / `<!-- /ref:KEY -->` blocks; one concept per block
3. **New files of any kind** — add to `.claude/index.md` under the appropriate table
4. **§ vs ref:KEY** — use `ref:KEY` for content agents need at runtime; use `§ "Heading"` for background/archive pointers

[ref:indexing-convention]

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
