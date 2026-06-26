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
**Sessions 33–35:** Plan A (T-05) + Plan B (T-04) — expense **type** persisted end-to-end;
`ExpenseSheet`→`ExpenseType` rename + JSON migration; generator **two-tier routing**
(full-path for typed, transitional bare-name fallback for type-less, NFC keys). 5.R4
historical extraction (2022–2025 old workbooks → corpus 694→1788 + per-year expense logs).
**Major pivot decided (session 36):** **retire workbook insertion, keep only generation**
— JSONL logs become the single source of truth, `generate-workbook` the only writer. Plan:
`.claude/plans/retire-insertion-keep-generation.md`. WS-0 diff validated the premise
(expenses reproduce; income is the sole gap); WS-0b extracted historical income.
**Session 37 (2026-06-23) — WS-A/T-11 DONE** (branch `chore/income-extraction-tooling`):
`parseDate` accepts `DD/MM`+`DD/MM/YYYY`; `LoadTaxonomy`/`scanEntries` filter by target year;
acceptance + unit tests green; throwaway merge script (`merge_year_logs.py`) → one multi-year
log, byte-identical gate passed all years. Income decisions locked (3-level symmetric income,
separate `--income-entries`, signed values). Currency formatting confirmed a **no-op**
(generator already numeric + `R$ #,##0.00`; the WS-0 "bare string" was a dump artifact).
Next: **WS-C** (income route — model→loader→router→generator, planned/not started, bigger
than WS-A); then WS-B (commands→log-append), WS-D (retire fallback), WS-E (delete dead code).
Open: promote merged log to canonical + retire per-year split (deferred, user's call).

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
