# expense-reporter/ — Knowledge (Semantic Memory)

*Go application accumulated decisions. Read on demand by agents.*

## Command Hierarchy (2026-02 → 2026-03)
Five user-facing commands, each building on the previous:
- `add "item;DD/MM;value;subcategory"` — manual insert with known subcategory
- `batch "file.csv"` — bulk manual inserts from CSV
- `classify "item" value DD/MM` — LLM classification only, no insert (read-only)
- `auto "item" value DD/MM` — classify + auto-insert if confidence ≥ 0.85
- `batch-auto "file.csv"` — classify CSV batch + auto-insert high-confidence rows
**Rationale:** Incremental trust — `classify` lets users verify the model before `auto`
inserts anything. `batch-auto` is the production workflow.
**Implication:** `auto` and `batch-auto` share the same classifier + decision logic.
The only difference is input source (args vs CSV) and output format.

## Batch Pipeline Optimization (2026-03)
`workflow.InsertBatchExpenses` opens the workbook once, inserts all rows, saves once.
This is 20–28x faster than calling `InsertExpense` per row.
Installments are expanded before insertion; the index mapping tracks which expanded
row maps back to which original input for error reporting.
**Rationale:** Excel file I/O (open + save) dominates per-row cost. Amortizing it across
a batch eliminates the bottleneck.
**Implication:** Single-expense `InsertExpense` delegates to `InsertBatchExpenses` with
a one-element slice — both paths share identical pipeline logic.

## JSON Output Mode (2026-03)
`--json` flag on root command; checked by `classify`, `auto`, and `add` subcommands.
Output structs: `ClassifyOutput`, `AutoOutput`, `AddOutput` — all in `cmd/output.go`.
`auto --json` is read-only (never inserts) and returns `action: "would_insert"|"review"|"excluded"`.
**Rationale:** MCP server needs structured, parseable output. Human-readable output
(confidence bars, Unicode symbols) breaks machine consumption.
**Implication:** JSON mode on `auto` is intentionally non-destructive — the MCP server
decides whether to call `add` separately.

## Resolver — Fuzzy Subcategory Matching (2026-02)
`internal/resolver` builds a hierarchical index from the Excel reference sheet.
Matching strategy: exact match → case-insensitive → prefix match → contains match.
The reference sheet is the canonical source of valid subcategories.
**Rationale:** User input varies ("Orion - Consultas" vs "Orion"). Fuzzy matching
reduces friction without requiring exact category knowledge.
**Implication:** Adding a new subcategory only requires adding it to the Excel reference
sheet — no code changes needed.

## Excel Integration (2026-02)
Uses excelize library. Key design: month columns are detected by header pattern
(month names in Portuguese), not hardcoded column indices.
Workbook path resolution: `--workbook` flag → `EXPENSE_WORKBOOK_PATH` env → default path.
Automatic backup before any write operation.
**Rationale:** The workbook layout may change year-to-year. Dynamic column detection
makes the tool resilient to layout changes.
**Implication:** The reference sheet must follow the expected format (category/subcategory
columns), but data sheet column positions are flexible.

## Feedback Persistence Design (2026-03)
Two append-only JSONL files:
- `classifications.jsonl` — status: confirmed/corrected/manual; includes model name,
  predicted vs actual subcategory. Source for few-shot feedback examples.
- `expenses_log.jsonl` — slim: item, date, value, subcategory, category, timestamp.
  Source for expense history queries.
Shared ID: `sha256(item+date+value+subcategory)[:12]` — correlates entries across files.
**Rationale:** Separate concerns — classification feedback (ML improvement) vs expense
history (data queries). The shared hash enables joins without a database.
**Implication:** Both files grow monotonically. No update/delete operations.

## Config Design (2026-02)
`config/config.json` with `internal/config` loader. Key fields:
- `date_year` (2025) — applied to DD/MM dates during parsing
- `auto_insert_excluded` (["Diversos"]) — subcategories blocked from auto-insert
- `classifications_path`, `expenses_log_path` — JSONL output locations
**Rationale:** Runtime behavior that changes between years or users belongs in config,
not code. The exclusion list was added after discovering "Diversos" false positives.
**Implication:** Year rollover requires updating `date_year` in config (and likely the
workbook path). No code changes needed.
