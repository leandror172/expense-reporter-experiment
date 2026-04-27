# Task Progress — Expense Reporter

**Last Updated:** 2026-03-13 (session 6)
**Active Layer:** Layer 5 — Expense Classifier
**Full history:** `.claude/session-log.md` (session log), `docs/archive/` (Desktop-era docs)

---

## Pre-History Summary (Claude Desktop — complete)

All work below was done in Claude Desktop before Claude Code was set up for this repo.

- **Phases 1–11:** Full CLI built — CSV parser, Excel writer, category resolver, batch import, Cobra CLI
- **Tests:** 190+ tests, table-driven, all passing
- **Commands:** `add` (single expense), `batch` (CSV import), `version`
- **Version:** v2.1.0
- **Classification analysis:** Auto-categorization of historical expenses using LLM feature extraction
  Results in `data/classification/` — algorithm docs, training data (gitignored), parameters

---

## Two-Repo Workflow

- **This repo** (`~/workspaces/expenses/code/`): Layer 5 feature work (classify, auto, batch-auto commands)
- **LLM repo** (`/mnt/i/workspaces/llm/`): MCP thin wrapper (5.8), ollama-bridge MCP server, personas
- **Scaffolding template:** `/mnt/i/workspaces/llm/docs/scaffolding-template.md`

---

## Layer 5: Expense Classifier

**Goal:** Local model classifies expenses, auto-inserts into Excel via expense-reporter Go tool.
**Context:** `/mnt/i/workspaces/llm/docs/vision/expense-classifier-vision.md` (full vision + iterative plan)
**Data inventory:** `/mnt/i/workspaces/llm/docs/vision/expense-classifier-data-inventory.md`
**Classification data:** `data/classification/` (this repo)

### Pre-work — COMPLETE (LLM repo sessions 32–35)
- [x] **5.0a** ollama-bridge JSONL logging: call logs to `~/.local/share/ollama-bridge/calls.jsonl`
- [x] **5.0b** CLAUDE.md in LLM repo: Layer 5+ local-model-first instruction (ACCEPTED/IMPROVED/REJECTED)
- [x] **5.0c** Model audit: qwen2.5-coder:14b, qwen3:8b-q8_0, qwen3:30b-a3b pulled; personas created
- [x] **5.0d** Multi-model comparison tooling: `run-compare-models.sh` + `run-record-verdicts.sh`
- [x] **5.0e** Fix `think: false` in ollama_client.py: top-level payload, not options{}. 82% token reduction.
- [x] **5.0f** num_ctx 16384→10240 in go-qwen25c14.Modelfile. No OOM.
- [x] **5.0g** my-java-q25c14 persona created (qwen2.5-coder:14b, Java 21 + Spring Boot 3.x)
- [x] **5.0h** Scaffolding bootstrap: this repo now has .claude/ tools, CLAUDE.md, index.md (session 1)

### Layer 5 Tasks (next)
- [x] **5.1** Port training data into expense-reporter: verify `feature_dictionary_enhanced.json` + `training_data_complete.json` are in `data/classification/`; document their JSON format in index.md
- [x] **5.2** `classify` command: 3-field input (date, description, amount) → Ollama HTTP → structured JSON → top-N subcategories with confidence scores
- [x] **5.3** `auto` command: classify + insert into Excel if HIGH confidence (≥0.85), else print candidates
- [x] **5.4** `batch-auto` command: classify a CSV file → classified.csv (HIGH confidence) + review.csv (LOW confidence)
- [X] **5.5** Classification feedback logging: `classifications.jsonl` — `{id, item, date, value, predicted_*, actual_*, confidence, model, status, 
timestamp}` appended on insert (status: confirmed/corrected/manual). Plan: `.claude/plans/polished-knitting-simon.md`
- [x] **5.6** Expense persistence: hash ID (sha256[:12] of normalized item+date+value), `expenses_log.jsonl` appended on each insert
- [x] **5.7** Few-shot injection: keyword pre-match against training data, inject top-K examples into classify prompt
- [x] **5.8** MCP thin wrapper (lives in THIS repo at `mcp-server/`, not LLM repo): `classify_expense` + `add_expense` tools, calls Go binary as subprocess. `auto_add` dropped by design — see `mcp-server/.memories/KNOWLEDGE.md` "Two Tools, Not Three" (2026-03-27).

### Deferred Technical Debt

- [ ] **TestBatchAuto_SameYearInstallmentsExpanded — reduce scope to structural validation**: The current test uses a real workbook insert to verify installment expansion, but this is fragile — the classifier may return a category name instead of a subcategory name (e.g. "Saúde" instead of "Academia"), causing a resolution error independent of installment logic. The installment expansion is already covered by `workflow_test.go` unit tests. Better acceptance test: use `--dry-run` and assert the classified.csv has the expense with installment notation preserved in the `value` column, or just assert structural validity (`classifiedAndReviewFilesProduced()`). Avoids dependency on classifier subcategory quality for a test whose purpose is installment mechanics.

- [x] **config reader: replace runtime.Caller with os.Executable** — Fixed session 4 (commit 405953e). `internal/config/config.go` now uses `os.Executable()` to resolve `config/config.json` relative to the binary location.

### Deferred — Retrieval Pipeline Evolution (from 5.7 planning, session 10)

These build on the keyword matching layer delivered in 5.7. Each is a separate cascade
layer — see `data/classification/retrieval-strategy.md` for the full pipeline design.

- [ ] **5.R1** TF-IDF retrieval layer: second cascade layer for multi-word similarity matching.
  Uses existing IDF weights from `feature_dictionary_enhanced.json` (229 keywords).
  Pure Go implementation, no external dependency. ~1.2MB memory for 694 training vectors.
  **Trigger:** keyword miss rate > 10% or accuracy on misses < 70%.
  Reference: `data/classification/tfidf-retrieval.md`
- [ ] **5.R2** Embedding retrieval layer (RAG): third cascade layer for semantic matching.
  Uses Ollama `/api/embeddings` endpoint (local, no external API). Handles synonym/brand
  gaps that lexical methods can't bridge. Needs embedding model benchmarking for Portuguese.
  **Trigger:** semantic gap cases > 5% of corrections after TF-IDF is active.
  Reference: `data/classification/embedding-retrieval.md`
- [ ] **5.R3** Value-range plausibility: use `value_ranges` from feature dict to pre-filter
  or post-validate subcategory candidates. Example: R$5000 expense matching "Diarista"
  (max 220) should be flagged as implausible. Orthogonal to few-shot injection — can be
  a separate scoring modifier or pre-filter on the taxonomy sent to the model.
- [ ] **5.R4** Historical workbook extraction: extract labeled expenses from pre-2024
  workbooks to expand training set beyond 694 entries. Benefits all retrieval layers
  equally (more examples = better keyword coverage, TF-IDF vocabulary, embedding space).
- [ ] **5.R5** Correction-weighted example selection: when `classifications.jsonl` has
  enough corrected entries, prioritize them as few-shot examples over confirmed/training
  entries. Simple sort order initially (corrected > training > confirmed). Refine to
  weighted scoring if correction volume warrants it.

### Deferred Tooling Improvements

- [ ] **T1** Resume context loading: session-context.md not auto-loaded on resume, so TDD/user-pref directives are missed. Options documented in `.claude/ideas/resume-context-loading.md`. Likely solution: `/resume` skill.

### Deferred Improvements (from AUTO_CATEGORY_README.md Next Steps)

These items are not blocking for Layer 5.1–5.8 but should be revisited after the core classify/auto commands are working:

- [ ] **5.D1** Add missing subcategories: Utilities (água/luz/gás), Insurance, Credit cards, Subscriptions (identified during initial classification run — see `data/classification/AUTO_CATEGORY_README.md` § "Next Steps")
- [ ] **5.D2** Fuzzy string matching: Levenshtein distance for typo handling (e.g., "Shoope" → "Shopee")
- [ ] **5.D3** Vendor database: canonical vendor name list to normalize inconsistent spellings
- [ ] **5.D4** Seasonal/temporal patterns: day-of-month and month-of-year features for recurring bills
- [ ] **5.D5** ML model experiment: train Random Forest or fine-tune a small BERT on labeled data once enough corrections accumulate (see `data/classification/research_insights.md` § 5 for approach)

### Key Decisions (from LLM repo session 32 design)
- Classification logic in **expense-reporter** (Go) — product feature, not LLM infrastructure
- MCP wrapper in **LLM repo** — thin layer, calls the Go binary as subprocess
- Training data strategy: hybrid (feature dict + correction rules as system + top-K few-shot per request)
- Structured output via Ollama `format` param — already proven reliable in LLM infra work
- Model to benchmark: Qwen3-8B (`my-classifier-q3`) vs Qwen2.5-Coder-7B (speed)
- Confidence threshold: HIGH ≥ 0.85 (auto-insert), LOW < 0.85 (present candidates)
