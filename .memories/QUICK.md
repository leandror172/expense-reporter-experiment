# expense-reporter repo — Quick Memory

*Working memory for the repo root. Injected into agents. Keep under 30 lines.*

## Status
Layer 5 (Expense Classifier) active. Phases 1–4 complete (CLI: add/batch/version, 190+ unit tests, v2.1.0).
Layer 5 milestones done: classifier (5.2), decision logic (5.3), auto/batch-auto commands (5.4–5.5),
feedback persistence (5.6), few-shot injection (5.7), JSON output + MCP server (5.8a/b).
Open PRs: #11–#14 (5.8 chain). Next: merge PR chain, then TF-IDF retrieval (5.R1).

## Repo Structure
```
expense-reporter/         # Go module — CLI application (Cobra, excelize, Ollama)
mcp-server/               # Python MCP server — thin wrapper calling Go binary
data/classification/      # Taxonomy, training data, algorithm docs (JSONs gitignored)
docs/archive/             # Historical planning docs (read-only)
.claude/                  # Session tracking, tools, overlays
```

## Key Rules
- **Go CLI is the product** — all classification logic lives in Go, not the MCP wrapper
- **Brazilian locale** — DD/MM/YYYY dates, comma decimal (1.234,56), BRL currency
- **Hierarchical categories** — Category → Subcategory, loaded from feature_dictionary_enhanced.json
- **Local-first ML** — Ollama models for classification; no cloud API calls
- **Sensitive data gitignored** — training JSONs, expense CSVs, personal financial data

## Deeper Memory → KNOWLEDGE.md
- **Architecture layers** — CSV → parse → classify → decide → insert → log
- **Classification strategy** — few-shot injection, confidence thresholds, exclusion list
- **Cross-repo relationship** — this repo vs LLM infra vs web-research
- **Testing strategy** — unit (testify) + acceptance (BDD harness, live Ollama)
