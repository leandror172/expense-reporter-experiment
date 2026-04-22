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
| Implementation plan (session 4) | `.claude/plans/acceptance-harness-batch-auto.md` |
| Implementation plan (session 6 — 5.5) | `.claude/plans/polished-knitting-simon.md` |
| Implementation plan (session 10 — 5.7) | `.claude/plans/5.7-few-shot-injection.md` |
| Session log archive (sessions 1–2) | `.claude/archive/session-log-2026-03-02-to-2026-03-02.md` |
| Session log archive (sessions 3–5) | `.claude/archive/session-log-2026-03-13-to-2026-03-02.md` |
| Session log archive (session 6 — 2026-03-03) | `.claude/archive/session-log-2026-03-03-to-2026-03-03.md` |
| Session log archive (session 7 — 2026-03-11) | `.claude/archive/session-log-2026-03-11-to-2026-03-11.md` |
| Run acceptance tests | `expense-reporter/run-acceptance.sh` — pre-flight + `go test -tags=acceptance ./test/...` |
| Acceptance test patterns | `expense-reporter/test/PATTERNS.md` — [ref:acceptance-patterns] effort table + ref index |
| Acceptance test architecture | `expense-reporter/test/README.md` — [ref:acceptance-harness], [ref:acceptance-fixtures], [ref:acceptance-verify], [ref:acceptance-run] |

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
| `pkg/utils` | `expense-reporter/pkg/utils/` | Public utility functions (currency, date, format) |
| `config` | `expense-reporter/config/` | Configuration files and constants |
| `internal/classifier` | `expense-reporter/internal/classifier/` | Ollama classifier + `IsAutoInsertable` decision logic; `examples.go` (SelectExamples, KeywordIndex, tokenization); `loader.go` (LoadTrainingExamples, LoadFeedbackExamples, LoadKeywordIndex, MergeExamplePools) |
| `internal/config` | `expense-reporter/internal/config/` | Config struct + `Load()` + `ClassificationsFilePath()` + `ExpensesLogFilePath()` |
| `internal/feedback` | `expense-reporter/internal/feedback/` | JSONL feedback logging: `Entry`, `GenerateID`, `Append`, `NewConfirmedEntry`, `NewManualEntry`; `ExpenseEntry`, `NewExpenseEntry`, `AppendExpense` (slim insert log → `expenses_log.jsonl`) |
| `test/harness` | `expense-reporter/test/harness/` | Acceptance test engine (Context, Scenario, fixtures, Ollama check, SetupBinaryConfig) |
| `test/actions` | `expense-reporter/test/actions/` | When-closures: RunClassify, RunAuto, RunBatchAuto, RunAdd |
| `test/verify` | `expense-reporter/test/verify/` | Then-closures: ExitCodeZero, RowCount, AllConfidencesInRange, SoftAccuracy, FeedbackFile* |
| `test/fixtures` | `expense-reporter/test/fixtures/` | Fixture data per functional slice (classify-basic, batch-auto-basic, batch-auto-exclusions, batch-auto-feedback) |
<!-- /ref:go-structure -->

---

<!-- ref:testing -->
## Testing Conventions

- **Test runner:** `cd expense-reporter && go test ./...`
- **Framework:** `testing` package (stdlib) + testify (`assert`/`require`) — both in use
- **Unit test style:** testify preferred for new tests; existing stdlib tests left as-is
- **Coverage:** 220+ tests across all packages
- **Pattern:** Each `internal/X/` has `X_test.go` with table-driven cases
- **Test data:** `expense-reporter/expenses.example.csv` — safe, no personal data
- **TDD approach:** Write failing test first, implement to pass (established pattern from Phases 1-4)

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

<!-- ref:training-data-schema -->
## Training Data Schema

### `training_data_complete.json` (694 labeled expenses — gitignored)

```
{
  "metadata": {
    "total_expenses": 694,        // 304 from 2024 + 303 from 2025 + 87 user corrections
    "unique_categories": 16,
    "unique_subcategories": 68,
    "extraction_date": "ISO 8601"
  },
  "expenses": [
    {
      "id": 1,                    // sequential integer
      "item": "Diarista Letícia", // raw description as it appears in the spreadsheet
      "date": "2024-01-05",       // ISO 8601 (YYYY-MM-DD); source data is DD/MM/YYYY
      "value": 160.0,             // float, BRL
      "subcategory": "Diarista",  // ground truth label
      "category": "Habitação",    // parent category
      "source": "Filename.xlsx:SheetName",
      "year": 2024
    }
  ]
}
```

### `feature_dictionary_enhanced.json` (229 keywords — gitignored)

```
{
  "lexical_features": {
    "keywords": {
      "<lowercase_keyword>": {
        "frequency": 17,                    // occurrence count in training set
        "dominant_subcategory": "Diarista", // most common subcategory for this keyword
        "dominant_count": 17,               // how many times the dominant mapping occurred
        "specificity": 1.0,                 // dominant_count / frequency (1.0 = unambiguous)
        "idf": 3.652,                       // inverse document frequency (rarity signal)
        "subcategories": ["Diarista"]       // all subcategories this keyword appears in
      }
    }
  },
  "value_ranges": {
    "<subcategory>": { "min": 160.0, "max": 219.8, "mean": 176.19,
                       "median": 172.0, "q1": 160.0, "q3": 172.0, "count": 17 }
  },
  "category_mapping": {
    "<subcategory>": "<parent_category>"    // 68 entries; flat lookup
  },
  "user_corrections": {
    "<normalized_item_lowercase>": {
      "category": "Lazer",
      "subcategory": "Delivery",
      "full_item": "Delivery brod's"        // original casing
    }
  }
}
```

### Classify command input (for 5.2)

Three fields passed to the classifier:
- `item` — expense description string (free text, Portuguese)
- `value` — float (BRL)
- `date` — string DD/MM/YYYY (Brazilian format; normalize before lookup)

### How these files are used in Layer 5

| File | Role at classify time |
|------|----------------------|
| `training_data_complete.json` | Source of few-shot examples (top-K by keyword similarity, injected into prompt) |
| `feature_dictionary_enhanced.json` | Fast pre-filter: keyword lookup → candidate subcategory before calling Ollama; also value range plausibility check |
| `algorithm_parameters.json` | Threshold values (tracked) — HIGH ≥ 0.85, MEDIUM ≥ 0.50, LOW < 0.50 |
<!-- /ref:training-data-schema -->

---

<!-- ref:confidence-thresholds -->
## Confidence Thresholds

Source: `data/classification/algorithm_parameters.json` (tracked).

| Level | Threshold | Behavior |
|-------|-----------|----------|
| HIGH | ≥ 0.85 | Auto-insert into workbook (`auto` command proceeds without prompt) |
| MEDIUM | ≥ 0.50 | Print top candidates, ask user to confirm |
| LOW | < 0.50 | Flag for manual review; do not insert |

**Fallback:** if no keyword matches and value range is inconclusive → category `Diversos`, confidence `0.30`, `require_manual_review: true`.

**Feature weights** (for scoring inside the classifier):
- keyword_match: 0.50
- semantic_similarity: 0.30
- value_proximity: 0.20
<!-- /ref:confidence-thresholds -->

---

<!-- ref:classification-overview -->
## Classification Overview

**Dataset:** 694 labeled expenses, 2024–2025, Brazilian Portuguese descriptions, BRL values.
**Taxonomy:** 16 categories → 68 subcategories.
**Keyword dictionary:** 229 terms (lowercase, with IDF and specificity scores).
**User corrections:** 71 override rules (exact normalized item → forced subcategory).

**Algorithm (hybrid, priority order):**
1. User correction lookup (exact match on normalized item string) → confidence 1.0
2. Keyword match scoring (TF-IDF weighted, specificity boosted) → up to 0.85
3. Value range plausibility (Gaussian kernel against per-subcategory IQR) → modifier ±0.20
4. Fallback → Diversos, 0.30

**Key preprocessing:**
- Lowercase, remove special chars, normalize whitespace
- Do NOT strip accents (Portuguese keywords retain accents: `gás`, `habitação`)
- Minimum word length: 2 chars

**Performance (from Desktop-era analysis):**
- High-specificity keywords (specificity = 1.0): unambiguous mapping, no LLM needed
- Ambiguous keywords (e.g., `va` maps to 5 subcategories): value range resolves most cases
- Remaining ambiguous: few-shot LLM call via Ollama
<!-- /ref:classification-overview -->

---

<!-- ref:feedback-system -->
## Feedback System & Corrections

**File:** `docs/FEEDBACK_SYSTEM.md`

The feedback system logs classification decisions to `classifications.jsonl` for training, tuning, and audit purposes.

**Key ref blocks:**
- `[ref:feedback-entry-structure]` — JSON entry format, status values (confirmed/corrected/manual)
- `[ref:feedback-sources]` — Which commands create which feedback entries (add, auto, batch-auto)
- `[ref:feedback-training]` — How feedback becomes training examples for next run
- `[ref:feedback-missing-feature]` — Missing correction workflow (status="corrected" not implemented)
- `[ref:feedback-file-path]` — Configuration and file resolution logic
- `[ref:feedback-cold-start]` — Behavior when classifications.jsonl doesn't exist

**Quick summary:**
- `add` command → `status="manual"` (user entered, no model)
- `auto`/`batch-auto` commands → `status="confirmed"` (model accepted automatically)
- No way to create → `status="corrected"` (model was wrong; user corrected it) — **feature gap**

<!-- /ref:feedback-system -->

---

## Retrieval Strategy Docs

| File | Tracked | Purpose |
|------|---------|---------|
| `data/classification/retrieval-strategy.md` | ✅ Yes | High-level retrieval pipeline: cascade diagram, token budget, data source strategy — [ref:retrieval-strategy], [ref:retrieval-token-budget] |
| `data/classification/tfidf-retrieval.md` | ✅ Yes | TF-IDF retrieval layer: existing artifacts, implementation approach, decision criteria — [ref:tfidf-retrieval] |
| `data/classification/embedding-retrieval.md` | ✅ Yes | Embedding/RAG retrieval layer: Ollama API, vector store, multilingual considerations — [ref:embedding-retrieval] |

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

## Vision & Planning Docs (docs/)

| File | Content |
|------|---------|
| `docs/expense-classifier-vision.md` | End-to-end Layer 5–6+ vision: user scenario, domain boundaries, iterative build plan (Phases 0–5), technical notes on structured output, persistence, queue. **Primary reference for understanding scope and architecture.** |
| `docs/expense-classifier-data-inventory.md` | Inventory of all auto-category analysis artifacts (feature dict, training data, confusion analysis, etc.) and their priority/role at build time. Also documents expense-reporter architecture as of Layer 5 start. |

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

## Per-Folder Memories

| Folder | Files | Content |
|--------|-------|---------|
| `.memories/` | QUICK.md, KNOWLEDGE.md | Repo-wide status, architecture, cross-repo relationships |
| `expense-reporter/.memories/` | QUICK.md, KNOWLEDGE.md | Go app structure, command hierarchy, batch pipeline, config |
| `expense-reporter/internal/classifier/.memories/` | QUICK.md, KNOWLEDGE.md | Few-shot algorithm, prompt architecture, empirical findings |
| `expense-reporter/test/.memories/` | QUICK.md, KNOWLEDGE.md | BDD harness design, fixture format, soft/hard assertions |
| `mcp-server/.memories/` | QUICK.md, KNOWLEDGE.md | Thin wrapper decisions, binary resolution, data-dir fix |

---

## READMEs

| File | Content |
|------|---------|
| `README.md` | Repo root — project overview, components, quick start |
| `expense-reporter/README.md` | Full CLI documentation — all commands, classification system, testing |
| `expense-reporter/test/README.md` | Acceptance test harness architecture, fixtures, verifiers |

---

## Tools

| Tool | Path | Purpose |
|------|------|---------|
| `resume.sh` | `.claude/tools/resume.sh` | Session-start context summary |
| `ref-lookup.sh` | `.claude/tools/ref-lookup.sh` | Resolve [ref:KEY] tags |
| `rotate-session-log.sh` | `.claude/tools/rotate-session-log.sh` | Archive old session log entries |
| `reconstruct-csvs.py` | `.claude/tools/reconstruct-csvs.py` | Reconstruct classified/review CSVs from batch-auto log + original input CSV (line-matched) |
| `lookup-category.py` | `.claude/tools/lookup-category.py` | Look up canonical category for one or more subcategories (`<sub> [...]` or `--list`) |
| session-handoff skill | `.claude/skills/session-handoff/SKILL.md` | End-of-session tracking workflow |
