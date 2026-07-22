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
4. **Decide** (`internal/classifier/decision.go`) — keyword-agreement gate + exclusion list → auto-insert or review (T-32; replaced the confidence threshold)
5. **Append** (`internal/appender`) — installment expansion → append to `expenses_log.jsonl`
6. **Log** (`internal/feedback`) — append to `classifications.jsonl` and `expenses_log.jsonl`
7. **Generate** (`internal/generate`) — build the Excel workbook FROM the logs (`generate-workbook`)

**Post-pivot note (session 36):** the JSONL logs are the single source of truth and
`generate-workbook` is the **only** workbook writer. The old direct-insert path
(`internal/workflow` + the `internal/excel` write side, reached via plain `batch`) is retired
but not yet deleted — removal is WS-E, which also takes `utils.ParseDate` with it. Do not treat
that path as the live pipeline.
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
2. **Layer 2 (TF-IDF):** RULED OUT (5.R1, session 52) — the 649-replay showed all retrieval
   misses are zero-lexical-overlap; TF-IDF cannot bridge them.
3. **Layer 3 (embedding):** **BUILT + ADOPTED, default-on** (5.R2, session 54; PR #45 merged
   session 57). On a keyword miss: embed the item (arctic-embed2) → cosine top-5 → inject as
   few-shot. Per-model JSONL disk cache; degrades to nil on any failure. Implemented in
   `internal/classifier/embedding{,_retriever,_fallback}.go`; `Config.EmbedModel` /
   `Config.NoEmbedRetrieval`, zero values = ON. Measured A/B on the miss stratum (n=80 unique):
   full-path 18.8% → 52.5% no-think. T-31 precondition (session 53) had shown NN hit@5 62–65%,
   multilingual models only.
Each layer feeds few-shot examples to the LLM prompt, improving classification accuracy.
**Rationale:** Empirical finding — LLM resolves multi-word context ("VA compras") better
than keyword specificity alone, but keywords select which few-shot examples to inject.
**Implication:** The keyword layer is a retrieval mechanism, not a classifier itself.

## Auto-Insert Gate — Agreement (T-32; was Confidence 2026-03)
`IsAutoInsertable` (`decision.go`) auto-inserts only when the model's predicted subcategory
AGREES with an unambiguous, maximum-specificity keyword match, and the subcategory is not
excluded. Confidence NO LONGER gates — the 649-replay (session 52) proved it uninformative
(the whole 0.85–0.95 band is a coin flip). All other rows route to review.
- Excluded subcategories (e.g., "Diversos") are never auto-inserted regardless of the gate.
- The ~95% subcat precision is a same-sample RELATIVE validation (shipped predicate ==
  measured predicate), NOT an absolute production guarantee — the 649 is a
  confidence-selected subset. This is why unattended silent auto-insert (WS-D) stays held.
**Rationale:** confidence was measured dead, so the gate switched to keyword agreement
(specificity + model⊕keyword concurrence). "Diversos" exclusion still guards the vaguest leaf.
**Implication:** Exclusion list is in `config.json`. `--threshold` on `batch-auto` is deprecated.

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
- **Sessions 58–59 — T2 harness extraction done.** The acceptance engine left this repo:
  `github.com/leandror172/acceptance-harness` (public, MIT; v0.1.1 at migration time,
  **v1.0.0 tagged session 60** — that is the version `go.mod` pins today) now provides
  Context/Scenario/Run, fixture plumbing, BuildBinary, and the generic `verify.*` Then
  assertions to both this repo and career-search's `roles` CLI. Session B (59) migrated
  expenses: `test/harness/` deleted; local `verify` → `test/expect/` (the module owns the
  `verify` name — scenarios import both); expense-shaped helpers → `test/{domain,extern}/`;
  `Context.{DataDir,WorkbookPath}` → `ctx.Env` accessors. Gate: the full acceptance roster
  is byte-identical pre/post (44 pass / 5 fail), the 5 reds being pre-existing bugs filed
  separately. Details → `expense-reporter/test/.memories/KNOWLEDGE.md`.
  **Rationale:** a second consumer is the only real test of a "domain-agnostic" claim, and
  two copies drift — the module's own unit tests immediately caught a latent `Run` bug that
  had been sitting in both copies.
  **Implication:** engine changes are now a PR upstream + a version bump here, which is the
  point: the boundary can no longer erode quietly.
- **Sessions 60–61 — T-20, T-35/T-36.** T-20: `batch-auto --resume` + always-on duplicate
  warning (id-count ledger `LoadExpenseIDCounts`, `PredictEntryIDs` pre-LLM skip, partially
  logged installment series route to review, never auto-completed). T-35: the two JSONL logs
  joined only on `GenerateID` (sha256 of raw bytes incl. the date string); `auto`/`batch-auto`/
  `apply` fed one log raw `DD/MM` and the other normalized `DD/MM/YYYY` → two ids per expense.
  Fix: canonicalize once per boundary; join-id guard tests MUST use short `DD/MM` fixtures
  (full dates hide the bug class). T-36 staleness sweep. Survey `.claude/t35-date-year-semantics.md`
  [ref:date-year-semantics] mapped six parsers/six year rules → T-37..T-40 filed.
- **Session 62 — T-39/T-24 (PR #52) + strategic reframe + parse-boundary plan.** Suite split:
  `testing.Short()` skip in `extern.RequireOllama`; `run-acceptance.sh` defaults to the
  deterministic group (~8s, no Ollama), `-full` = whole suite (3600s ceiling; measured 128s at
  think-off — was 840–1989s). `--think=false` now default on classify/auto/batch-auto (gate band
  −2.3pp WITH think; 5.R2 miss +2.5pp at 5.5× latency; sentinel covered by the agreement gate).
  Vision re-read → project reframed as **pre-first-use**; organizing milestone = first real
  monthly close on 2026 data. Parse-boundary plan drafted (`.claude/plans/parse-boundary.md`):
  structured input, parse-once; year precedence string > `--year` > config `date_year` >
  most-recent-non-future (grace window TBD); T-21 UX = 1 row + ×N badge; `--year` uniform;
  T-38 resolved = delete `utils.ParseDate` with plain `batch` under WS-E, don't repair.
