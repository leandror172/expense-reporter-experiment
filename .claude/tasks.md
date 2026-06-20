# Task Progress — Expense Reporter

**Last Updated:** 2026-06-09 (session 27)
**Active Layer:** Workbook Mapping
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
- [x] **5.R4** Historical workbook extraction: extract labeled expenses from pre-2024
  workbooks to expand training set beyond 694 entries. Benefits all retrieval layers
  equally (more examples = better keyword coverage, TF-IDF vocabulary, embedding space).
  DONE session 35 (one-off Python; 2022–2025 → corpus 694→1788 + per-year expense logs).
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

### Review UI & `review` Command (session 21)

Local-first review surface — supersedes the Lovable cloud plan
(`docs/plans/lovable-suggestion-plan.md`).

- [x] **RUI-1** Implement `expense-reporter review` command — CSV + workbook taxonomy
  baked into a self-contained `review.html`. Full plan: `.claude/plans/review-command.md`
  **Complete (session 23, 2026-05-18):** All phases done. PR #22 open
  (`worktree-feat+review-command` → `master`). 17 unit tests + acceptance test green.
  Smoke: 349 rows, 23 need review against real classified.csv.
- [x] **RUI-1a** Merge PR #22 (`worktree-feat+review-command` → `master`).
- [x] **RUI-2** Build the `review.html` template via claude.ai/design — brief:
  `docs/plans/review-ui-design-brief.md` + fixtures in `docs/plans/review-ui-fixtures/`.
  Done (session 22): template at `expense-reporter/internal/review/template/review.html`;
  dev-preview copy with sample data kept at repo root `review.html`.
- [x] **RUI-3** `apply` command: ingest the UI's `reviewed.json` back into the workbook
  + feedback logs (`feedback.Entry` confirmed/corrected, `expenses_log.jsonl`).
  **Complete (session 24, 2026-05-29):** Phases 0–3 done. Acceptance test green.
  PR #23 (`feat/apply-command` → `master`). Plan: `.claude/plans/apply-command.md`.
- [x] **RUI-3a** Phase 4 smoke: run `apply` against a real `reviewed.json` from a prior
  review session. Exercises `insertNewRows` (workbook insertion path), which has zero
  acceptance test coverage — the fixture only covers already-inserted and pending entries.
  Index-aliasing bug and dry-run leak already fixed (see commit after Phase 3).
- [x] **RUI-4** Emit the full 3-level path (sheet,category,subcategory) into the
  classified CSV. When done, `review`'s `ReadQueue` populates `Predicted.Sheet` and the
  UI pre-fill becomes unambiguous (the `Predicted` struct already has the optional field).

### Key Decisions (from LLM repo session 32 design)
- Classification logic in **expense-reporter** (Go) — product feature, not LLM infrastructure
- MCP wrapper in **LLM repo** — thin layer, calls the Go binary as subprocess
- Training data strategy: hybrid (feature dict + correction rules as system + top-K few-shot per request)
- Structured output via Ollama `format` param — already proven reliable in LLM infra work
- Model to benchmark: Qwen3-8B (`my-classifier-q3`) vs Qwen2.5-Coder-7B (speed)
- Confidence threshold: HIGH ≥ 0.85 (auto-insert), LOW < 0.85 (present candidates)

<!-- ref:deferred -->
## Deferred / Backlog

- [x] (RUI-WM1) **Workbook mapping Layer 1** — Rewrite `cmd/workbook-inspect/main.go` to output full JSON per sheet: all cells, formulas, cell styles (fill color, bold, borders). Schema in `.claude/plans/workbook-mapping-plan.md`.
- [x] (RUI-WM2) **Workbook mapping Layer 2** — Chrome automation screenshot pass via Google Sheets. Use Layer 1 block list to target sections. See plan.
- [x] (RUI-WM3) **Workbook mapping Layer 3** — claude.ai synthesis of Layer 1+2 into `workbook-generator-spec.md`. Must complete before 2026-07-05 (2× usage expires).
- [x] (RUI-WG) **Workbook generator command** — Implement `generate` command that produces workbook from `classifications.jsonl` + `expenses_log.jsonl`. Blocked on RUI-WM3 (spec).
- [ ] (T-01) **Merge PR #27** — workbook generator branch; user submits/discards pending review first (drafts addressed in session 29)
- [x] (T-02) **Real-taxonomy export** — one-time: Referência (113 subcats) → `taxonomy.json` per spec §1.1; compose sub-item splits into col-B strings; validate via skeleton generation
  - **Addendum (decided session 30, 2026-06-12):** when implementing this export, split `loader.go` (+ the input-only parts of `taxonomy.go`) into a sibling `internal/taxonomy` package — pure input layer, zero excelize — and land the export writer there. Explicit decision NOT to subpackage styles/sheets (Go flat-package idiom; too coupled). **Prerequisite (found session 31):** `taxonomy.go` mixes domain types with mutable RENDER config (`dataYear`/`headroomRows`/`perGroupPctRows`, set by `Generate()`, read by builders); relocate those vars into `generate` before the split, and decide whether the domain types live in `internal/taxonomy` or stay in `generate` (cycle risk to check).
- [ ] (T-03) **Year-rollover workflow** — generate year N+1 skeleton from taxonomy alone; decide `apply`/`add` fate against generated workbooks
- [x] (T-04) **Full-path entry routing** — route logged entries by full sheet/category/subcategory path instead of bare name; changes the entry contract (`expenses_log.jsonl` carries only a bare `subcategory`), the classifier that emits entries, `scanEntries`, and entry-fed fixtures. Advisor-reviewed. Removes the interim ambiguity guard. See `[ref:taxonomy-identity-key]` §6 / `.claude/plans/taxonomy-identity-key.md`.
- [x] (T-05) **Persist expense type end-to-end (Plan A)** — add `type,omitempty` to feedback.Entry + ExpenseEntry, rename ExpenseSheet→ExpenseType, migrate JSON keys (`sheet`→`type`, `sheets`→`types`) with legacy-read fallback, partial backfill from re-exported reviewed.json. Plan: `.claude/plans/persist-expense-type.md`. Hard prereq for T-04. Advisor-reviewed; not yet implemented.
- [x] (T-06) **Bf real-data verification** — execute `.claude/plans/bf-real-data-verification-runbook.md`: export reviewed.json → backfill-type.py → generate-workbook against real config/taxonomy.json; confirm typed entries route (no `not in taxonomy` warnings on typed lines), ambiguous leaves land in the right sheet. The only end-to-end proof of the Plan A→B chain on real data.
- [ ] (T-08) **Generate full-suite acceptance flake (low priority)** — `TestGenerateWorkbook_Skeleton` fails under full-suite timeout (partial actual-dump → nil panic) but passes in isolation. Confirmed benign timeout artifact, not a Phase-R/Type regression. Investigate `test/verify` parallel-safety / temp-dir isolation only if it recurs when run alone.
- [ ] (T-09) **Retire bare-name routing fallback** — now that the classifier emits type (5.R4-classifier, done), shrink/retire the transitional bare-name `byName` fallback in `internal/taxonomy` `scanEntries` as typed coverage approaches 100%; gate retirement on the stderr type-less fallback count reaching ~0. Advisor-flagged risk: deleting it prematurely drops the auto-inserted majority.
- [x] (T-10) **Disambiguate the `5.R4` id collision** — RESOLVED session 35: `5.R4` stays the historical-extraction task (its original owner in the 5.R1–5.R5 series, now done); the classifier-emits-type work is referred to as **`5.R4-classifier`** (adopted in current-status/reading-guide and T-09). Boards now agree.
- [ ] (T-11) **Year adaptation for expenses_log** — `parseDate` requires exactly `DD/MM`, forcing per-year log files (5.R4). Accept `DD/MM/YYYY` + have `generate` use a per-entry year (fallback `--year`) so one multi-year log routes directly and the per-year split retires.
<!-- /ref:deferred -->
