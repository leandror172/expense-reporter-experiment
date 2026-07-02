# expense-reporter repo — Knowledge (Semantic Memory)

*Repo-wide accumulated decisions. Read on demand by agents.*

## Project Identity (2026-02)
Go CLI that reads CSV bank exports, classifies expenses using local LLMs (Ollama),
and inserts them into an Excel budget workbook. Built for personal finance automation
with Brazilian financial conventions (BRL, DD/MM/YYYY, comma decimals).
**Rationale:** Automates a tedious manual process — copying bank statement lines into a
categorized Excel budget. Local LLMs keep financial data private.
**Implication:** All ML inference is local (Ollama). No cloud APIs for classification.

## Architecture — Pipeline Layers (2026-02 → 2026-03)
The expense processing pipeline has distinct stages, each handled by a Go package:
1. **Parse** (`internal/parser`) — semicolon-delimited input → `models.Expense` struct
2. **Resolve** (`internal/resolver`) — fuzzy-match subcategory against reference sheet
3. **Classify** (`internal/classifier`) — LLM-based categorization with confidence scores
4. **Decide** (`internal/classifier/decision.go`) — threshold + exclusion list → auto-insert or review
5. **Insert** (`internal/workflow` + `internal/excel`) — write to Excel workbook with backup
6. **Log** (`internal/feedback`) — append to `classifications.jsonl` and `expenses_log.jsonl`
**Rationale:** Each stage is independently testable. The pipeline can stop at any stage
(e.g., `classify` stops at step 3; `auto` goes through step 6; `batch-auto` adds CSV I/O).
**Implication:** New features (like TF-IDF retrieval) slot into the pipeline at a specific
stage without affecting others.

## Installment Handling (2026-03)
Expenses with notation like "99,90/3" are expanded into 3 monthly installments.
Installments crossing the year boundary produce rollover rows written to `rollover.csv`.
**Rationale:** Brazilian credit card statements show installment totals; the budget needs
individual monthly entries.
**Implication:** Batch processing must track original→expanded index mapping for error reporting.

## Classification Strategy — Three-Layer Cascade (2026-03)
1. **Layer 1 (keyword):** Tokenize item → look up in feature dictionary → find dominant
   subcategory by specificity score. Implemented in `classifier/examples.go`.
2. **Layer 2 (TF-IDF):** Planned (5.R1) — corpus-level term frequency for better retrieval.
3. **Layer 3 (embedding):** Deferred — vector similarity for semantic matching.
Each layer feeds few-shot examples to the LLM prompt, improving classification accuracy.
**Rationale:** Empirical finding — LLM resolves multi-word context ("VA compras") better
than keyword specificity alone, but keywords select which few-shot examples to inject.
**Implication:** The keyword layer is a retrieval mechanism, not a classifier itself.

## Confidence Threshold Design (2026-03)
- HIGH ≥ 0.85 → auto-insert into workbook
- LOW < 0.85 → print candidates, require manual review
- Excluded subcategories (e.g., "Diversos") are never auto-inserted regardless of confidence
**Rationale:** "Diversos" (miscellaneous) at 90% confidence is a false positive — the model
is confident it doesn't know. Discovered empirically during integration testing.
**Implication:** Exclusion list is in `config.json`, not hardcoded. New problematic categories
can be added without code changes.

## Feedback Loop (2026-03)
Two JSONL files persist classification results:
- `classifications.jsonl` — full classification context (predicted vs actual, model, status)
- `expenses_log.jsonl` — slim insert log (item, date, value, subcategory, category, hash ID)
Both share a sha256[:12] ID for cross-file correlation.
Confirmed/corrected entries from `classifications.jsonl` are loaded back as few-shot examples,
creating a self-improving feedback loop.
**Rationale:** The system learns from corrections without retraining. Corrected examples get
highest priority in few-shot selection (SourceCorrected > SourceTraining > SourceConfirmed).
**Implication:** Classification accuracy improves with use as the feedback pool grows.

## MCP Integration (2026-03)
Python MCP server (`mcp-server/`) wraps the Go binary as two tools: `classify_expense`
and `add_expense`. Uses FastMCP, calls Go binary as subprocess with `--json` flag.
**Rationale:** MCP provides a standard protocol for Claude Code to invoke the classifier.
The Go binary remains the single source of truth for all logic.
**Implication:** The Python layer is intentionally thin — no classification logic, no state.
Binary path is resolved at server startup (lifespan context).

## Cross-Repo Relationships (2026-03)
- **LLM infrastructure** (`/mnt/i/workspaces/llm/`) — Ollama personas, MCP bridge server,
  platform documentation. Classification models are defined as personas there.
- **Web research** (`/mnt/i/workspaces/web-research/`) — Independent project. Shares the
  `.memories/` convention and MCP-as-integration-layer pattern.
**Rationale:** Each repo owns its domain. Communication happens through MCP, not shared code.
**Implication:** Model configuration (persona definitions, prompt templates) lives in the
LLM repo; this repo only references model names.

## Key Empirical Findings (2026-03)
- LLM resolves multi-word context better than keyword specificity — "VA compras" → 100%
  despite "va" having specificity=0.36 in feature dictionary
- Fallback category "Diversos" at high confidence is a real risk — blocked via exclusion list
- `Transporte` appearing as subcategory at 90% in Uber test case — taxonomy oddity, not urgent
- Cold-start Ollama timeouts are normal — first-call timeout = retry, not a rejection

## Milestone Log (consolidated from QUICK.md, 2026-07-01)
- **Sessions 26–29 (PR #27) — workbook generator.** Mapping L1–L3 → spec v2 → Phase A/B
  convergence (user-blessed data-bearing golden master) → Phase G: `internal/inspect`
  (dump core), `internal/generate` + `generate-workbook` command; acceptance-first with
  oracle-frozen dumps (deterministic, no Ollama). Review follow-ups: English identifiers,
  SOLID extraction, hardcoded sheet-order bug fixed (registry `sheetOrder`). Scratch
  builder SUPERSEDED. Sessions 30–31: behavior-preserving internal refactor (styles
  vocabulary, shared helpers, unified block sizing) — see generate/.memories.
- **Sessions 33–35 — Plan A (T-05) + Plan B (T-04).** Expense type persisted end-to-end;
  `ExpenseSheet`→`ExpenseType` rename + JSON migration; generator two-tier routing
  (full-path for typed, transitional bare-name fallback, NFC keys). 5.R4 historical
  extraction: 2022–2025 old workbooks → corpus 694→1788 + per-year expense logs.
- **Session 36 — pivot decided:** retire workbook insertion, keep only generation. WS-0
  diff validated the premise (expenses reproduce; income was the sole gap); WS-0b
  extracted historical income.
- **Session 37 — WS-A/T-11 done** (branch `chore/income-extraction-tooling`): `parseDate`
  accepts `DD/MM`+`DD/MM/YYYY`; year filtering in `scanEntries`; throwaway
  `merge_year_logs.py` → one multi-year log, byte-identical gate passed all years. Income
  decisions locked (3-level symmetric income, separate `--income-entries`, signed values).
  Currency formatting confirmed a no-op (generator already numeric + `R$ #,##0.00`).
  Open: promote merged log to canonical + retire per-year split (user's call).
- **Session 38 — WS-C income route done** (model→loader→router→generator).
- **Sessions 41–42 — T-13:** classifier predicts full path via grammar-enforced 112-enum;
  default model `my-classifier-q3`; acceptance repairs (see session42-postmortem).
- **Sessions 43–44 — WS-B done:** batch-auto (slice 3) and apply (slice 4) →
  log-append via `appender.ExpandAndAppend`; rollover.csv retired; pre-flights + failure
  honesty. Next: WS-D (retire fallback, T-09), WS-E (delete dead insert code).
