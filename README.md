# Expense Reporter

Automated personal expense management — classifies bank statement entries using
local LLMs and inserts them into an Excel budget workbook.

## Components

### [`expense-reporter/`](expense-reporter/)
Go CLI application. Parses CSV bank exports, classifies expenses via Ollama,
and writes to an Excel workbook with hierarchical category resolution.

Commands: `add`, `batch`, `classify`, `auto`, `batch-auto`, `review`, `apply`,
`generate-workbook`, `telegram-import`, `correct`, `version`.

```bash
cd expense-reporter && go build ./cmd/expense-reporter
./expense-reporter auto "Uber Centro" 35,50 15/04
# ✓ Inserted: Uber Centro → Uber/Taxi (Transporte) — 95% confidence
```

### [`mcp-server/`](mcp-server/)
Python MCP server wrapping the Go binary. Exposes `classify_expense` and
`add_expense` as MCP tools for Claude Code integration.

```bash
cd mcp-server && uv run expense-mcp
```

### `data/classification/`
Taxonomy definitions, feature dictionaries, and training data for the classifier.
Algorithm documentation and research notes. Training JSONs are gitignored
(contain personal financial data).

## How Classification Works

1. Expense description is tokenized and matched against a feature dictionary
2. Keyword matches select few-shot training examples (up to 5). On a keyword **miss**, an
   embedding retriever fills in instead — cosine top-5 over the training corpus
3. System prompt with full taxonomy + few-shot pairs sent to local Ollama model, with the
   response schema constraining every candidate to a valid taxonomy path
4. Model returns ranked subcategory candidates with confidence scores
5. **Agreement gate** decides auto-append: the model's predicted subcategory must agree with an
   unambiguous, maximum-specificity keyword match, and not be on the exclusion list. Everything
   else goes to manual review. (Confidence is reported but does not gate — it was measured
   uninformative on 649 real labels and dropped in T-32.)
6. Results logged to JSONL feedback files, feeding back into future classifications —
   corrections outrank training data in few-shot selection

## Requirements

- **Go 1.25.5** — for the CLI
- **Python 3.10+** + **uv** — for the MCP server
- **Ollama** — local LLM runtime (required for classification commands)
- **Excel workbook** — with a reference sheet defining the category taxonomy

## Quick Start

```bash
# Build the CLI
cd expense-reporter && go build -o expense-reporter ./cmd/expense-reporter

# Set workbook path
export EXPENSE_WORKBOOK_PATH="/path/to/budget.xlsx"

# Classify an expense (read-only)
./expense-reporter classify "Uber Centro" 35,50 15/04

# Classify and auto-insert if confident
./expense-reporter auto "Uber Centro" 35,50 15/04

# Batch classify a CSV file
./expense-reporter batch-auto expenses.csv
```

See [expense-reporter/README.md](expense-reporter/README.md) for full documentation.

## Testing

```bash
cd expense-reporter && go test ./...              # 263 test funcs / 647 with subtests
cd expense-reporter && ./run-acceptance.sh        # deterministic acceptance tests (no Ollama)
cd expense-reporter && ./run-acceptance.sh -full  # full acceptance suite (requires Ollama)
```

## License

Personal use project.
