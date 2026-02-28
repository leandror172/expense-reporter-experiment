# Knowledge Index — Expense Reporter

**Purpose:** Map of where all project information lives. Read this to find anything.

<!-- ref:indexing-convention -->
### Indexing Conventions (Two-Tier System)

| Tier | Notation | When to Use | Lookup Method |
|------|----------|-------------|---------------|
| **Active reference** | `<!-- ref:KEY -->` + `[ref:KEY]` | Agent needs this during work; CLAUDE.md rules point here | `.claude/tools/ref-lookup.sh KEY` (machine-lookupable) |
| **Navigation pointer** | `§ "Heading"` | Index/docs pointing to sections for background reading | Open file, find heading (human/agent reads) |

**Active refs** are for high-frequency, runtime lookups (go package layout, test conventions, classification schema).
**§ pointers** are for low-frequency, "read when needed" navigation (research findings, planning docs, historical context).

**Single-responsibility rule:** One ref block per concept.
<!-- /ref:indexing-convention -->

---

## Quick Pointers (Active Work)

| What | Where |
|------|-------|
| Current layer tasks & progress | `.claude/tasks.md` |
| Session log (current) | `.claude/session-log.md` |
| Agent preferences & resume checklist | `.claude/session-context.md` |
| Project rules & constraints | `CLAUDE.md` (repo root) |
| Cross-repo context (LLM infra) | `/mnt/i/workspaces/llm/.claude/` |

---

<!-- ref:go-structure -->
## Go Package Structure

| Package | Path | Purpose |
|---------|------|---------|
| `main` | `expense-reporter/cmd/expense-reporter/main.go` | Entry point |
| `cmd` | `expense-reporter/cmd/expense-reporter/cmd/` | Cobra CLI subcommands (add, batch, version; +classify/auto/batch-auto in Layer 5) |
| `internal/batch` | `expense-reporter/internal/batch/` | Batch import orchestration |
| `internal/cli` | `expense-reporter/internal/cli/` | CLI output helpers |
| `internal/excel` | `expense-reporter/internal/excel/` | Excel workbook read/write (excelize library) |
| `internal/logger` | `expense-reporter/internal/logger/` | Logging abstraction |
| `internal/models` | `expense-reporter/internal/models/` | Domain structs: Expense, Category, SubCategory, etc. |
| `internal/parser` | `expense-reporter/internal/parser/` | CSV parsing, date/decimal normalization (BR format) |
| `internal/resolver` | `expense-reporter/internal/resolver/` | Category resolution: matches expense → (category, subcategory) |
| `internal/workflow` | `expense-reporter/internal/workflow/` | Multi-step workflow: parse → resolve → insert |
| `pkg/utils` | `expense-reporter/pkg/utils/` | Public utility functions |
| `config` | `expense-reporter/config/` | Configuration files and constants |
<!-- /ref:go-structure -->

---

<!-- ref:testing -->
## Testing Conventions

- **Test runner:** `cd expense-reporter && go test ./...`
- **Framework:** Standard library (`testing` package), table-driven tests
- **Coverage:** 190+ tests across all packages
- **Pattern:** Each `internal/X/` has `X_test.go` with table-driven cases
- **Test data:** `expense-reporter/expenses.example.csv` — safe, no personal data
- **TDD approach:** Write failing test first, implement to pass (established pattern from Phases 1-4)
- **No external test framework** — only stdlib `testing` + `testify` if already present in go.mod

**Do NOT:**
- Use `testing.T.Fatal` where `t.Errorf` suffices
- Skip table-driven approach for new commands
<!-- /ref:testing -->

---

<!-- ref:classification -->
## Classification Data

See `data/classification/classification_algorithm.md` for the full algorithm description.
See `data/classification/reproducibility_guide.md` for how to reproduce results.

**Algorithm summary:** Hybrid rule-based + statistical scoring. Priority: precision over recall.
**Training data:** 694 labeled expenses (304 from 2024, 303 from 2025, 87 user corrections).
**Coverage:** 16 categories, 68 subcategories, 229 keywords.
**Layer 5 strategy:** Feature dictionary as system context + top-K few-shot examples per request.

**Key ref blocks for Layer 5 agents:**
- `ref-lookup.sh training-data-schema` — JSON schema for `training_data_complete.json` and `feature_dictionary_enhanced.json`; classify command input format
- `ref-lookup.sh confidence-thresholds` — HIGH/MEDIUM/LOW thresholds (0.85/0.50) and per-level behavior
- `ref-lookup.sh classification-overview` — executive summary + algorithm performance stats
<!-- /ref:classification -->

---

## Classification Data Files

| File | Tracked | Purpose |
|------|---------|---------|
| `data/classification/classification_algorithm.md` | ✅ Yes | Algorithm description — no personal data |
| `data/classification/reproducibility_guide.md` | ✅ Yes | How to reproduce results |
| `data/classification/algorithm_parameters.json` | ✅ Yes | Parameter values — no personal data |
| `data/classification/AUTO_CATEGORY_README.md` | ✅ Yes | Overview of the auto-categorization effort |
| `data/classification/FINAL_SUMMARY.md` | ✅ Yes | Final analysis summary |
| `data/classification/classification_reasoning.md` | ✅ Yes | Reasoning behind category assignments |
| `data/classification/research_insights.md` | ✅ Yes | Research findings |
| `data/classification/llm_reasoning_meta.md` | ✅ Yes | LLM reasoning analysis |
| `data/classification/feature_dictionary_enhanced.json` | ❌ Gitignored | Personal expense keywords |
| `data/classification/feature_dictionary.json` | ❌ Gitignored | Personal expense keywords |
| `data/classification/training_data_complete.json` | ❌ Gitignored | Labeled expenses (personal) |
| `data/classification/training_data.json` | ❌ Gitignored | Labeled expenses (personal) |
| `data/classification/final_classifications.json` | ❌ Gitignored | Classification results |
| `data/classification/*.csv` | ❌ Gitignored | Personal expense CSVs |
| `data/classification/similarity_matrix.json` | ❌ Gitignored | Computed similarity data |
| `data/classification/vector_representations.json` | ❌ Gitignored | Feature vectors |
| `data/classification/statistical_summary.json` | ❌ Gitignored | Statistical results |
| `data/classification/confusion_analysis.json` | ❌ Gitignored | Per item analysis (may contain real descriptions) |

---

## Historical Docs (docs/archive/)

Desktop-era planning documents — read for context, do not modify.

| File | Content |
|------|---------|
| `ARCHITECTURE_PLAN_Expense_Automation.md` | Original automation architecture plan |
| `ARCHITECTURE_PLAN_Go_Implementation.md` | Go implementation architecture |
| `AUTO_CATEGORIZATION_PROMPT.md` | Original auto-categorization prompt design |
| `DECISIONS_FINALIZED.md` | Finalized design decisions |
| `GO_vs_PYTHON_Comparison.md` | Language selection rationale |
| `PRE_IMPLEMENTATION_CHECKLIST.md` | Pre-implementation verification |
| `PRE_PHASE1_VERIFICATION_CHECKLIST.md` | Phase 1 checklist |
| `QUICK_REFERENCE_Implementation_Summary.md` | Implementation quick reference |
| `SHEET_NORMALIZATION_PROMPT.md` | Sheet normalization prompt |
| `USAGE_CALCULATION.md` | Usage/quota calculations |
| `DOCUMENTATION_INDEX.md` | Original doc index (superseded by this file) |
| `PROJECT_COMPLETE.md` | Phase completion summary |
| `BATCH_IMPORT_COMPLETE.md` | Batch import feature completion |
| `PHASE1-4_COMPLETE.md` | Combined phase completion doc |
| `FEATURES_INVESTIGATION_REPORT.md` | Feature investigation results |
| `FEATURE_INVESTIGATION.md` | Feature investigation notes |
| `TESTING.md` | Testing documentation (current: expense-reporter/README.md) |

---

## Tools

| Tool | Path | Purpose |
|------|------|---------|
| `resume.sh` | `.claude/tools/resume.sh` | Session-start context summary |
| `ref-lookup.sh` | `.claude/tools/ref-lookup.sh` | Resolve [ref:KEY] tags |
| `rotate-session-log.sh` | `.claude/tools/rotate-session-log.sh` | Archive old session log entries |
| session-handoff skill | `.claude/skills/session-handoff/SKILL.md` | End-of-session tracking workflow |
