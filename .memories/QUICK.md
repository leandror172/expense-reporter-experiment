# expense-reporter repo — Quick Memory

*Working memory for the repo root. Injected into agents. Keep under 30 lines.*

## Status
Layer 5 (Expense Classifier) done through 5.9; workbook generator COMPLETE.
**Pivot: JSONL logs are the single source of truth; `generate-workbook` is the only
workbook writer.** Landed: full-path typed classification (T-13), multi-year logs
(WS-A), income route (WS-C), commands→log-append (WS-B). **T-23/5.R1 MEASURED on real
data (649-replay, s52): confidence dead as gate; specificity+agreement is the gate;
5.R2 embeddings (not 5.R1) is the accuracy lever. T-31 NN precondition (s53): GO for
5.R2 — hit@5 62–65% multilingual on the 160 misses; arctic-embed2 the practical pick.
5.R2 BUILT + ADOPTED (s54): embed-on-miss cascade layer in `internal/classifier`
(arctic-embed2, K=5, disk cache); replay A/B miss full-path 18.8%→52.5% no-think
(+33.8pp). Reports `.claude/scratch/replay-649/`. T-32 agreement gate SHIPPED (s57).
T2 harness extraction DONE (s58 module+career-search; s59 Session B): the acceptance
engine is now the `github.com/leandror172/acceptance-harness` module — `test/harness/`
is gone, domain split into `test/{expect,domain,extern}/`; v1.0.0 tagged s60.
T-20 DONE (s60): `batch-auto --resume` + always-on dup warning — id-count ledger,
`PredictEntryIDs` pre-LLM skip, partial series → review.** Next = WS-D (retire
bare-name fallback, HELD: gate-to-review not silent insert) → WS-E.
History → KNOWLEDGE.md "Milestone Log".

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
