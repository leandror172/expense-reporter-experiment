# expense-reporter repo — Quick Memory

*Working memory for the repo root. Injected into agents. Keep under 30 lines.*

## Status
Layer 5 (Expense Classifier) done through 5.9; workbook generator COMPLETE.
**Pivot: JSONL logs are the single source of truth; `generate-workbook` is the only
workbook writer.** Landed: full-path typed classification (T-13), multi-year logs
(WS-A), income route (WS-C), commands→log-append (WS-B). Next: WS-D (retire bare-name
fallback) → WS-E (delete dead insert code). History → KNOWLEDGE.md "Milestone Log".

## Repo Structure
```
expense-reporter/         # Go module — CLI application (Cobra, excelize, Ollama)
mcp-server/               # Python MCP server — thin wrapper calling Go binary
data/classification/      # Taxonomy, training data (JSONs gitignored)
docs/archive/             # Historical planning docs (read-only)
.claude/                  # Session tracking, tools, overlays
```

## Key Rules
- **Go CLI is the product** — all classification logic lives in Go, not the MCP wrapper
- **Brazilian locale** — DD/MM/YYYY dates, comma decimal (1.234,56), BRL currency
- **Hierarchical categories** — Type → Category → Subcategory; identity = full path
- **Local-first ML** — Ollama models for classification; no cloud API calls
- **Sensitive data gitignored** — training JSONs, expense CSVs, personal financial data

## Deeper Memory → KNOWLEDGE.md
Architecture layers · classification strategy · cross-repo relationships · milestone log
