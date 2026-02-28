# Task Progress — Expense Reporter

**Last Updated:** 2026-02-27 (session 1)
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
- [ ] **5.1** Port training data into expense-reporter: verify `feature_dictionary_enhanced.json` + `training_data_complete.json` are in `data/classification/`; document their JSON format in index.md
- [ ] **5.2** `classify` command: 3-field input (date, description, amount) → Ollama HTTP → structured JSON → top-N subcategories with confidence scores
- [ ] **5.3** `auto` command: classify + insert into Excel if HIGH confidence (≥0.85), else print candidates
- [ ] **5.4** `batch-auto` command: classify a CSV file → classified.csv (HIGH confidence) + review.csv (LOW confidence)
- [ ] **5.5** Correction logging: `corrections.jsonl` — `{input, predicted, actual, confidence}` appended on user override
- [ ] **5.6** Expense persistence: hash ID (sha256[:12] of normalized item+date+value), `expenses_log.jsonl` appended on each insert
- [ ] **5.7** Few-shot injection: keyword pre-match against training data, inject top-K examples into classify prompt
- [ ] **5.8** MCP thin wrapper in LLM repo: `classify_expense` / `add_expense` / `auto_add` tools (calls Go binary as subprocess)

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
