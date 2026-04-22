# Expense Reporter

Go CLI that automates personal expense management — reads bank statement exports,
classifies them using local LLMs, and inserts entries into an Excel budget workbook.

## Overview

Built for Brazilian personal finance: DD/MM/YYYY dates, comma decimal separators
(`1.234,56`), BRL currency, and a hierarchical category taxonomy (Category → Subcategory).

Classification runs entirely on local Ollama models — no cloud API calls, keeping
financial data private.

**Current version:** 2.1.0  
**Tests:** 190+ unit tests, 8 acceptance test fixtures  
**Go version:** 1.25.5

## Commands

### `add` — Insert a single expense

```bash
expense-reporter add "Uber Centro;15/04;35,50;Uber/Taxi"
# ✓ Expense added successfully!
```

The expense string format is `<item>;<DD/MM>;<value>;<subcategory>`.
Values use Brazilian format (`35,50` = thirty-five reais and fifty centavos).
Installment notation is supported: `300,00/3` divides into 3 monthly payments.

Flags:
- `--dry-run` — validate and parse without inserting
- `--data-dir` — path to classification data (for category resolution)
- `--json` — structured JSON output

### `classify` — Classify an expense (read-only)

```bash
expense-reporter classify "Uber Centro" 35,50 15/04
#   1. Uber/Taxi           Transporte     ████████████████ 95%
#   2. 99/Taxi             Transporte     ████████         52%
#   3. Combustível         Transporte     ████             28%
```

Sends the expense to a local Ollama model and returns ranked subcategory candidates
with confidence scores. Does not insert anything.

Flags:
- `--model` — Ollama model override (default: `my-classifier-q3`)
- `--data-dir` — path to classification data directory
- `--json` — structured JSON output

### `auto` — Classify and auto-insert if confident

```bash
expense-reporter auto "Uber Centro" 35,50 15/04
# ✓ Inserted: Uber Centro → Uber/Taxi (Transporte) — 95% confidence
```

Classifies the expense, then:
- **Confidence ≥ 85%** and subcategory not excluded → auto-inserts into workbook
- **Confidence < 85%** → prints candidates for manual review, does not insert
- **Excluded subcategory** (e.g., "Diversos") → prints warning, does not insert

Flags:
- `--confirm` — always ask for confirmation before inserting
- `--model`, `--data-dir`, `--json` — same as `classify`

In JSON mode, `auto` is read-only — it returns a recommendation
(`would_insert` / `review` / `excluded`) without inserting.

### `batch` — Bulk manual import from CSV

```bash
expense-reporter batch expenses.csv --backup --report=import_report.txt
```

Imports multiple expenses from a semicolon-delimited CSV file.
Each row must include a known subcategory (no classification involved).

Flags:
- `--backup` — create timestamped backup before processing
- `--report` — report output path (default: `batch_report.txt`)
- `--silent` — suppress progress bar

### `batch-auto` — Classify and auto-insert a CSV batch

```bash
expense-reporter batch-auto expenses.csv
expense-reporter batch-auto expenses.csv --dry-run --output-dir /tmp/out
```

Reads a 3-field CSV (`item;DD/MM;value`), classifies each row via Ollama,
and auto-inserts rows exceeding the confidence threshold.

Output files:
- `classified.csv` — all rows with classification results
- `review.csv` — rows not auto-inserted (low confidence or excluded)
- `rollover.csv` — installment rows crossing into next year

Flags:
- `--dry-run` — classify only, skip workbook insertion
- `--threshold` — confidence threshold (default: 0.85)
- `--model`, `--data-dir`, `--output-dir`, `--top`

### `correct` — Override a prior auto-classification

```bash
expense-reporter correct "Uber Centro;15/04;35,50;Combustível"
# ✓ Correction logged: Uber Centro → Combustível (was Uber/Taxi)
```

When a previously confirmed (auto-inserted) classification was wrong, `correct` logs
a `corrected` entry to `classifications.jsonl`, preserving the original `predicted_*`
and `model` fields so future few-shot retrieval can prioritize learning from this
mistake. Requires a prior entry — for expenses with no model prediction, use `add`.

This command does **not** modify the workbook; it only writes to the feedback log.

Flags:
- `--data-dir` — path to classification data (for resolving the corrected category)

### `version` — Print version

```bash
expense-reporter version
# expense-reporter version 2.1.0
```

## Classification System

The classifier uses local Ollama models with a three-layer retrieval cascade
for few-shot example selection:

1. **Keyword layer** (implemented) — tokenizes the expense description, looks up tokens
   in a feature dictionary with specificity scores, selects training examples from
   the most relevant subcategories
2. **TF-IDF layer** (planned) — corpus-level term frequency for better retrieval
3. **Embedding layer** (deferred) — vector similarity for semantic matching

### How it works

1. Load taxonomy from `feature_dictionary_enhanced.json` (subcategory → category mapping)
2. Select up to 5 few-shot examples via keyword matching
3. Build prompt: system instruction + taxonomy + few-shot pairs + user query
4. Send to Ollama with structured output (JSON schema in `format` param)
5. Parse response, apply confidence threshold and exclusion list
6. Insert or present for review

### Feedback loop

Two JSONL files persist classification results:
- `classifications.jsonl` — full classification context (predicted vs actual, model, status)
- `expenses_log.jsonl` — slim insert log (item, date, value, subcategory, category)

Confirmed and corrected entries are loaded back as few-shot examples, so classification
accuracy improves with use. Corrected examples get highest priority in selection.

Three commands write to the log:
- `add` → `manual` (no model prediction)
- `auto` / `batch-auto` → `confirmed` (model prediction accepted)
- `correct` → `corrected` (user overrode a prior `confirmed` entry)

## MCP Server

`mcp-server/` contains a Python MCP server that wraps the Go binary for integration
with Claude Code and other MCP clients. See [mcp-server/](../mcp-server/) for details.

Two tools:
- `classify_expense` — calls `auto --json` (read-only, returns recommendation)
- `add_expense` — calls `add --json` (inserts into workbook)

## Input Format

### Expense string

```
<item>;<DD/MM>;<value>;<subcategory>
```

- **Item:** free text (no semicolons)
- **Date:** DD/MM (year from `config.json`, default 2025)
- **Value:** Brazilian format — `150,00` for single payment, `300,00/3` for installments
- **Subcategory:** must exist in the Excel reference sheet

### CSV format

Semicolon-delimited. Lines starting with `#` are comments.

For `batch` (4 fields):
```csv
Uber Centro;15/04;35,50;Uber/Taxi
Compras Carrefour;03/01;150,00;Supermercado
```

For `batch-auto` (3 fields — subcategory is classified):
```csv
Uber Centro;15/04;35,50
Compras Carrefour;03/01;150,00
```

### Hierarchical subcategory paths

When a subcategory appears in multiple sheets, disambiguate with paths:
```
Diarista                      # may be ambiguous
Habitação,Diarista            # 2-level path
Fixas,Habitação,Diarista      # full path
```

Resolution is progressive — tries the shortest path first, adds levels if ambiguous.

## Installment Payments

Input `300,00/3` produces 3 monthly entries of `100,00` each:
```
Compra parcelada (1/3) — Feb 20 — 100,00
Compra parcelada (2/3) — Mar 20 — 100,00
Compra parcelada (3/3) — Apr 20 — 100,00
```

Installments crossing into the next year are written to a separate rollover file.

## Project Structure

```
cmd/expense-reporter/
  main.go                  # Entry point
  cmd/                     # Cobra subcommands: add, auto, batch, batch-auto,
                           #   classify, correct, version, root, output
internal/
  batch/                   # CSV reading, installment expansion, progress, reports
  classifier/              # LLM classification — Ollama client, few-shot selection,
                           #   decision logic, training data loaders
  cli/                     # CLI formatting (confidence bars)
  config/                  # config.json loader
  excel/                   # Excelize wrapper — reference sheet, column mapping, writer
  feedback/                # JSONL persistence (classifications + expense log)
  logger/                  # Debug logging
  models/                  # Domain types: Expense, BatchError, ClassifiedExpense
  parser/                  # Semicolon-delimited expense string parser
  resolver/                # Fuzzy subcategory matching against reference sheet
  workflow/                # Orchestration: parse → resolve → expand → insert pipeline
pkg/utils/                 # Currency parsing, date formatting, string building
config/config.json         # Runtime config (workbook path, exclusion list, log paths)
test/                      # Acceptance test suite (BDD harness, live Ollama)
```

## Configuration

`config/config.json`:
```json
{
  "workbook_path": "../workbook.xlsx",
  "reference_sheet": "Referência de Categorias",
  "date_year": 2025,
  "auto_insert_excluded": ["Diversos"],
  "classifications_path": "classifications.jsonl",
  "expenses_log_path": "expenses_log.jsonl"
}
```

Workbook path resolution: `--workbook` flag → `EXPENSE_WORKBOOK_PATH` env → config default.

## Testing

### Unit tests

```bash
cd expense-reporter && go test ./...    # 190+ tests, ~50s
cd expense-reporter && go vet ./...     # lint
```

Table-driven tests using [testify](https://github.com/stretchr/testify) (`assert`/`require`).

### Acceptance tests

File-driven BDD harness in `test/` with build tag `//go:build acceptance`.
Requires a live Ollama instance.

```bash
cd expense-reporter && ./run-acceptance.sh
```

10 fixture directories: classify-basic, auto-basic, batch-auto-basic, batch-auto-exclusions,
batch-auto-feedback, batch-auto-installments, batch-auto-rollover, add-feedback,
correct-overrides-confirmed, correct-uses-latest-entry.

Soft accuracy assertions track classification drift across model/prompt updates
without requiring exact reproducibility.

## Dependencies

```go
require (
    github.com/spf13/cobra v1.10.2               // CLI framework
    github.com/xuri/excelize/v2 v2.10.0           // Excel operations
    github.com/schollz/progressbar/v3 v3.18.0     // Progress bar
    github.com/stretchr/testify v1.11.1            // Test assertions
)
```

**Runtime:** [Ollama](https://ollama.com/) for local LLM classification
(required for `classify`, `auto`, `batch-auto` commands).

## Development History

**Phases 1–4:** Foundation — parser, models, Excel integration, Cobra CLI (131 tests)  
**Phases 5–9:** Batch import — CSV reader, processor, progress, reports, ambiguous handling (179 tests)  
**Phase 10:** Installment payments — expansion, rollover, partial failure handling  
**Phase 11:** Hierarchical subcategory paths — disambiguation for multi-sheet subcategories  
**Layer 5.2:** LLM classifier — Ollama integration, structured output, confidence scoring  
**Layer 5.3:** Decision logic — threshold + exclusion list for auto-insert  
**Layer 5.4–5.5:** Auto/batch-auto commands — single and batch classification workflows  
**Layer 5.6:** Feedback persistence — classifications.jsonl + expenses_log.jsonl  
**Layer 5.7:** Few-shot injection — keyword-based example selection from training + feedback data  
**Layer 5.8:** JSON output + MCP server — machine-readable output, Python MCP wrapper
**Layer 5.9:** Correction workflow — `correct` command closes the feedback loop by writing `status="corrected"` entries that take priority in few-shot retrieval

## License

Personal use project.
