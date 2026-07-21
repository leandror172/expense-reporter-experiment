# Expense Reporter

Go CLI that automates personal expense management — reads bank statement exports,
classifies them using local LLMs, and inserts entries into an Excel budget workbook.

## Overview

Built for Brazilian personal finance: DD/MM/YYYY dates, comma decimal separators
(`1.234,56`), BRL currency, and a hierarchical category taxonomy (Category → Subcategory).

Classification runs entirely on local Ollama models — no cloud API calls, keeping
financial data private.

**Current version:** 2.1.0  
**Tests:** 263 unit test functions (647 with subtests), 56 acceptance tests across 31 fixtures  
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

### `auto` — Classify and auto-append if the agreement gate passes

```bash
# gate passes — "posto"/"ipiranga" have specificity 1.0 → Combustível, and the model agrees
expense-reporter auto "Posto Ipiranga" 180,00 15/04
# ✓ Appended: Posto Ipiranga → Combustível (Variáveis) — 95% confidence

# gate fails — "uber" has specificity 0.80 (splits Viagens / Uber-Taxi), so despite a
# confident-looking prediction the row is held back for review
expense-reporter auto "Uber Centro" 35,50 15/04
# ⚠  Not appended — keyword specificity 0.80 is below the maximum required for auto-insert
```

Classifies the expense, then applies the **agreement gate** (`IsAutoInsertable`):
- **Gate passes** — a keyword matched, the match is unambiguous, its specificity is maximal
  (`top_score >= 1.0`), the model's predicted subcategory equals the keyword's top-1, and the
  subcategory is not excluded → appends to `expenses_log.jsonl`
- **Gate fails** (keyword miss, ambiguous match, sub-maximal specificity, or model/keyword
  disagreement) → prints candidates for manual review, appends nothing
- **Excluded subcategory** (e.g., "Diversos") → prints warning, appends nothing

Confidence is still reported per candidate but does **not** gate — it was measured
uninformative on 649 real labels and dropped in T-32. Like `batch-auto` and `apply`, `auto`
appends to the log rather than writing the workbook (WS-B pivot); run `generate-workbook`
to materialize the spreadsheet.

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
and auto-appends rows that pass the agreement gate.

Output files:
- `classified.csv` — all rows with classification results
- `review.csv` — rows not auto-inserted (agreement gate not met or excluded)

Auto-insert uses the **agreement gate**: a row is auto-inserted only when the model's
predicted subcategory agrees with an unambiguous, maximum-specificity keyword match
(and is not in the exclusion list). All other rows go to `review.csv`.

Flags:
- `--dry-run` — classify only, skip workbook insertion
- `--resume` — idempotent re-run: skip rows whose expense-log entry ids are ALL already
  present (printed as `SKIP … already logged`), before the model is called.
  Partially-logged installment series route to `review.csv` for manual resolution
  instead of being auto-completed. Bare `DD/MM` dates infer the current year, so a
  resume crossing a year boundary may not match — use `DD/MM/YYYY` for December batches.
- `--threshold` — **deprecated, ignored** (the gate no longer uses confidence)
- `--model`, `--data-dir`, `--output-dir`, `--top`

Independently of `--resume`, batch-auto always warns on stderr when an appended entry's
id already exists in the log (a likely duplicate append).

### `review` — Generate an interactive HTML review page

```bash
expense-reporter review classified.csv
expense-reporter review classified.csv --output review.html --workbook /path/to/workbook.xlsx
# wrote review.html — 349 rows (23 need review)
```

Takes the `classified.csv` output from `batch-auto` and bakes it into a self-contained
`review.html` file. The HTML contains the full expense queue and workbook taxonomy as
embedded JSON — open it directly in a browser, no server needed.

**Workflow position:** `batch-auto` → `classified.csv` → **`review`** → `review.html`
→ (browser review) → `reviewed.json` → `apply` (appends to `expenses_log.jsonl`)
→ `generate-workbook`

> Since the WS-B log-append pivot, `apply` (and `auto`/`batch-auto`) no longer write the
> workbook directly — they append typed entries to `expenses_log.jsonl`, and
> `generate-workbook` is the sole workbook writer.

In the browser:

- Rows are pre-filled with the classifier's predicted Sheet / Category / Subcategory
- "Needs review" filter (default) shows only rows that weren't auto-inserted
- Three cascading dropdowns per row — changing Sheet resets Category/Subcategory if
  incompatible; an amber hint flags subcategories that exist in multiple sheets
- Accept (`a`) or Skip (`s`) each row; `j`/`k` navigate; `1`–`4` set sheet by hotkey
- "Accept auto-inserted" bulk-confirms all already-classified rows at once
- Progress auto-saves to `localStorage` — reloading the file resumes where you left off
- **Shift+E** exports `reviewed.json` with every row's final action (`confirmed` /
  `corrected` / `skipped`) and the resolved Sheet / Category / Subcategory

Flags:
- `--output` / `-o` — output path (default: `review.html`)
- `--workbook` — workbook path override (for taxonomy; uses config/env otherwise)

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

### `generate-workbook` — Generate a complete workbook from data

```bash
expense-reporter generate-workbook -o 2026.xlsx --taxonomy taxonomy.json \
    --entries expenses_log.jsonl --year 2026
```

Builds a full expense workbook (Listas de itens + Receitas + one sheet per expense
group) from a JSON taxonomy file, optionally filled with entries from an
`expenses_log.jsonl` log. The workbook is **regenerated from data, never inserted
into** — rerun the command after the log changes.

| Flag | Required | Meaning |
|------|----------|---------|
| `-o, --output` | yes | output `.xlsx` path |
| `--taxonomy` | yes | taxonomy JSON file (sheets → categories → subcategories, plus income categories/blocks) |
| `--entries` | no | entries JSONL; omitted = skeleton workbook |
| `--year` | no (current year) | year applied to entry dates (`DD/MM` in the log has no year) |
| `--headroom` | no (0) | spare data rows per block beyond the busiest month |

Entries whose subcategory is not in the taxonomy are skipped with a warning
(exit stays 0). On category disagreement the taxonomy wins.

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
2. **TF-IDF layer** (RULED OUT, session 52) — the 649-replay showed 100% of retrieval
   misses share zero tokens with the pool, so lexical similarity cannot bridge them
3. **Embedding layer** (implemented — 5.R2, session 54) — on a keyword miss, the item is
   embedded via Ollama (`snowflake-arctic-embed2` by default) and the top-5 cosine
   neighbors from a disk-cached pool embedding are injected as few-shot examples.
   Measured on real data: miss-stratum full-path accuracy 18.8% → 52.5%.
   Config: `Config.EmbedModel` / `Config.NoEmbedRetrieval` (on by default); cache at
   `data/classification/embeddings-<model>.jsonl` (gitignored — real descriptions)

### How it works

1. Load taxonomy from `feature_dictionary_enhanced.json` (subcategory → category mapping)
2. Select up to 5 few-shot examples via keyword matching
3. Build prompt: system instruction + taxonomy + few-shot pairs + user query
4. Send to Ollama with structured output (JSON schema in `format` param)
5. Parse response, apply the **agreement gate** (`IsAutoInsertable`: keyword matched ∧
   unambiguous ∧ `top_score >= 1.0` ∧ model prediction == keyword top-1) and the exclusion list.
   Confidence is emitted but does not gate (T-32).
6. Append to the log or route to review

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

Installments crossing into the next year simply carry their real next-year date. The
separate `rollover.csv` was retired in the WS-B log-append pivot — `generate-workbook`
selects entries by year, so a next-year row needs no special handling.

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
  review/                  # review command: CSV reader, taxonomy builder, HTML renderer,
                           #   go:embed template; types: QueueEntry, Taxonomy, ReviewData
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

Note: `date_year` is currently not read by any code path — bare `DD/MM` dates fall back
to the current year (classifier-era commands). Wiring it as the configured fallback year
is planned (parse boundary).

## Testing

### Unit tests

```bash
cd expense-reporter && go test ./...    # 263 test funcs / 647 with subtests, ~60s
cd expense-reporter && go vet ./...     # lint
```

Table-driven tests using [testify](https://github.com/stretchr/testify) (`assert`/`require`).

### Acceptance tests

File-driven BDD harness in `test/` with build tag `//go:build acceptance`.
The default run is the deterministic group (no Ollama needed); `-full` runs the
whole suite and requires a live Ollama instance.

```bash
cd expense-reporter && ./run-acceptance.sh        # deterministic group
cd expense-reporter && ./run-acceptance.sh -full  # whole suite (requires Ollama)
```

11 fixture directories: classify-basic, auto-basic, batch-auto-basic, batch-auto-exclusions,
batch-auto-feedback, batch-auto-installments, batch-auto-rollover, add-feedback,
correct-overrides-confirmed, correct-uses-latest-entry, review-basic.

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
**Layer 5.3:** Decision logic — confidence threshold + exclusion list for auto-insert *(the
threshold half was later removed — see T-32 below; the exclusion list survives)*  
**Layer 5.4–5.5:** Auto/batch-auto commands — single and batch classification workflows  
**Layer 5.6:** Feedback persistence — classifications.jsonl + expenses_log.jsonl  
**Layer 5.7:** Few-shot injection — keyword-based example selection from training + feedback data  
**Layer 5.8:** JSON output + MCP server — machine-readable output, Python MCP wrapper
**Layer 5.9:** Correction workflow — `correct` command closes the feedback loop by writing `status="corrected"` entries that take priority in few-shot retrieval  
**RUI-1:** Review command — `review` bakes `classified.csv` + workbook taxonomy into a self-contained `review.html`; browser UI with cascading pickers, localStorage persistence, and `reviewed.json` export  
**Workbook generator + pivot:** `generate-workbook` builds the workbook from the JSONL logs; the logs become the single source of truth and generation the only writer (direct workbook insertion retired)  
**T-13:** Classifier predicts the full `Type/Category/Subcategory` path via a grammar-enforced 112-member enum, plus a sentinel path so the model can decline  
**5.R2:** Embed-on-miss retrieval — on a keyword miss, embed the item and inject cosine top-5 few-shot examples; degrades to nil if the embedder is unavailable  
**T-32:** Agreement gate — `IsAutoInsertable` drops confidence (measured uninformative on 649 real labels) and gates on keyword agreement instead; `--threshold` deprecated  
**T2:** Acceptance harness extracted to the standalone public module `github.com/leandror172/acceptance-harness` (v1.0.0, MIT)  
**T-20:** `batch-auto --resume` + always-on duplicate-append warning  
**T-35:** Join-id fix — dates canonicalized once per boundary so both JSONL logs share one `GenerateID`  
**T-39 / T-24:** Acceptance suite split into a deterministic (no-Ollama) group and a full group; `--think=false` becomes the default on all three model-facing commands

## License

Personal use project.
