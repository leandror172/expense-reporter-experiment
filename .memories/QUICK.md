# expense-reporter repo — Quick Memory

*Working memory for the repo root. Injected into agents. Keep under 30 lines.*

## Status
Layer 5 (Expense Classifier) milestones done through 5.9 (classifier, auto/batch-auto, feedback,
few-shot, JSON+MCP, `apply`, `review` UI). **Workbook generator COMPLETE** (sessions 26–29,
branch `feat/workbook-generator`, PR #27): mapping L1–L3 → spec v2 → Phase A/B convergence
(user-blessed data-bearing golden master) → Phase G: `internal/inspect` (dump core),
`internal/generate` + **`generate-workbook` command** (taxonomy JSON + expenses_log.jsonl →
full workbook), acceptance-first with oracle-frozen dumps (3/3 green, deterministic, no Ollama).
PR #27 review comments addressed (English identifiers, SOLID extraction); latent hardcoded
sheet-order bug fixed (registry `sheetOrder`). Scratch builder SUPERSEDED.
Sessions 30–31: `internal/generate` internal refactor (styles vocabulary + English renames;
loader/revenue/summary step-extraction; shared helpers → `util.go`/new `data_sheet.go`; unified
block-sizing + data-band writers) — behavior-preserving, oracle dumps unchanged, 2 commits on the
branch (uncommitted-then-committed). See generate `.memories/QUICK.md` for the conventions.
**Session 33 (2026-06-19) — Plan A (T-05) + Plan B (T-04) implemented** (PRs #29 + #30,
branches `feat/persist-expense-type` → stacked `feat/full-path-entry-routing`): expense
**type** persisted end-to-end (feedback structs carry `Type`, set on the apply path);
`ExpenseSheet`→`ExpenseType` rename + JSON migration (`sheets`→`types`, `sheet`→`type`
with legacy read-compat); generator **two-tier routing** — full-path for typed entries,
**transitional** bare-name fallback (+ambiguous-skip) for type-less — with NFC-normalized
keys; `backfill-type.py` recovery tool. Also remediated a `git add -A` data leak (runtime
logs now gitignored). PR #27 already merged.
Next: classifier full-path label (5.R4/RUI-4, closes the type-less producer gap);
year-rollover workflow; TF-IDF (5.R1). Real-data proof of the A→B chain pending Bf3
(export reviewed.json → backfill → confirm `expensePath` is a taxonomy key).

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
