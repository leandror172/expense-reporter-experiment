# Knowledge Index — Expense Reporter

**Purpose:** Map of where all project information lives. Read this to find anything.

<!-- ref:indexing-convention -->
### Indexing Conventions (Two-Tier System)

| Tier | Notation | When to Use | Lookup Method |
|------|----------|-------------|---------------|
| **Active reference** | `<!-- ref:KEY -->` + `[ref:KEY]` | Agent needs this during work; CLAUDE.md rules point here | `.claude/tools/ref-lookup.sh KEY` (machine-lookupable) |
| **Navigation pointer** | `§ "Heading"` | Index/docs pointing to sections for background reading | Open file, find heading (human/agent reads) |

**Active refs** are for high-frequency, runtime lookups (go package layout, test conventions, classification schema).
**§ pointers** are for low-frequency, "read when needed" navigation (research findings, planning docs, historical context).

**Single-responsibility rule:** One ref block per concept.
<!-- /ref:indexing-convention -->

---

## Quick Pointers (Active Work)

| What | Where |
|------|-------|
| Current layer tasks & progress | `.claude/tasks.md` |
| Session log (current) | `.claude/session-log.md` |
| Agent preferences & resume checklist | `.claude/session-context.md` |
| Project rules & constraints | `CLAUDE.md` (repo root) |
| Cross-repo context (LLM infra) | `/mnt/i/workspaces/llm/.claude/` |
| Implementation plan (session 4) | `.claude/plans/acceptance-harness-batch-auto.md` |
| Implementation plan (session 10 — 5.7) | `.claude/plans/5.7-few-shot-injection.md` |
| Implementation plan (session 20 — workbook path fix) | `.claude/plans/fix-workbook-path-resolution.md` |
| Implementation plan (`review` command) | `.claude/plans/review-command.md` |
| T-23 calibration benchmark plan (leaf-first confidence signal, session 49) | `.claude/plans/t23-calibration-benchmark.md` |
| Leaf-first classification plan (standalone accuracy A/B + T-23 precondition, session 49) | `.claude/plans/leaf-first-classification.md` |
| Review UI design brief (for claude.ai/design) | `docs/plans/review-ui-design-brief.md` + `docs/plans/review-ui-fixtures/` |
| Lovable companion suggestion (superseded by local design) | `docs/plans/lovable-suggestion-plan.md` |
| Workbook mapping plan (3-layer, session 26) | `.claude/plans/workbook-mapping-plan.md` |
| Workbook mapping Layer 3 cowork brief | `.claude/plans/workbook-layer3-instructions.md` |
| Workbook structural map (JSON + cell notes, session 25) | `.claude/workbook-map.md` |
| Workbook visual notes (annotated screenshots, session 26) | `.claude/workbook-visual-notes.md` |
| Workbook generator spec v2 (Layer 3 + hand-review, session 27) | `.claude/plans/workbook-generator-spec.md` |
| Workbook generator implementation plan + next-session brief | `.claude/plans/workbook-generator-implementation-plan.md` |
| Taxonomy identity key (full-path decision + ambiguity guard, task #5 deferred) | `.claude/plans/taxonomy-identity-key.md` [ref:taxonomy-identity-key] |
| Generator build order + row counter + oracle coupling | `expense-reporter/internal/generate/.memories/KNOWLEDGE.md` [ref:generate-build-order] [ref:generate-row-counter] [ref:generate-oracle-coupling] |
| Generator column layout (data sheets vs Listas) | `expense-reporter/internal/generate/.memories/KNOWLEDGE.md` [ref:generate-colmap] |
| Generator block sizing (calculateBlockRows) | `expense-reporter/internal/generate/.memories/KNOWLEDGE.md` [ref:generate-block-sizing] |
| Generator package-level state (dataYear/headroomRows/perGroupPctRows) | `expense-reporter/internal/generate/.memories/KNOWLEDGE.md` [ref:generate-package-state] |
| Generator excelize gotchas | `expense-reporter/internal/generate/.memories/KNOWLEDGE.md` [ref:generate-excelize-gotchas] |
| Generator income 3-level model + dual-format taxonomy | `expense-reporter/internal/generate/.memories/KNOWLEDGE.md` [ref:generate-income-symmetry] |
| Plan A — persist expense *type* + rename/JSON migration + backfill (session 32) | `.claude/plans/persist-expense-type.md` |
| Plan B — full-path entry routing (T-04, session 32) | `.claude/plans/full-path-entry-routing.md` |
| Bf real-data verification runbook (execute next session — Bf1/Bf3 + routing proof) | `.claude/plans/bf-real-data-verification-runbook.md` |
| Retire workbook insertion, keep only generation (logs = source of truth; couples T-11/T-09/T-03) | `.claude/plans/retire-insertion-keep-generation.md` |
| WS-B slice 3 — `batch-auto` → log-append (decisions locked, session 43; not yet implemented) | `.claude/plans/ws-b-slice3-batch-auto-log-append.md` |
| WS-B slice 4 — `apply` → log-append (decisions locked, session 44; not yet implemented) | `.claude/plans/ws-b-slice4-apply-log-append.md` |
| T-13 (revised) — classifier predicts full (type,category,subcategory) path; WS-D prerequisite | `.claude/plans/t13-classifier-full-path.md` |
| 5.R2 embedding retrieval — embed-on-miss cascade layer (DONE + ADOPTED session 54; A/B miss full-path 18.8%→52.5%) | `.claude/plans/5r2-embedding-retrieval.md` |
| Harness extraction plan (T2 — standalone `acceptance-harness` repo, career-search repoint, expenses migration, LTG path; session 56) | `.claude/plans/harness-extraction-plan.md` |
| acceptance-test vs harness comparison (companion: old-job pipeline framework — adopt/adapt/discard analysis; session 56) | `.claude/plans/harness-extraction-acceptance-test-comparison.md` |
| Harness extraction Session B — executable migration plan (expenses `test/` → module; measured inventory, D1 verify-namespace + D2 Context-carrier open; session 59) | `.claude/plans/harness-extraction-session-b.md` |
| Parse boundary — structured input, parse-once (year precedence, installments field, T-37/T-38/T-40/T-21 resolutions; FINAL — design session 63, awaiting implementation go) | `.claude/plans/parse-boundary.md` |
| Per-sheet structural digests (Layer 3 inputs, Sonnet fan-out) | `.claude/workbook-dump/digests/*.md` (gitignored with dump) |
| Template golden master (user-curated, fake data) | `.claude/workbook-template/template-reviewed.xlsx` + `template.xlsx` (generated) |
| Template build/convergence reports | `.claude/workbook-template/{ambiguities,review-diff,convergence-report}.md` + `diff.py` |
| Phase B data extract (template-data.xlsx block/column model) | `.claude/workbook-template/phaseB-data-extract.md` |
| Phase B re-review report (data-bearing template, PASS) | `.claude/workbook-template/phaseB-rereview.md` |
| Template builder (SUPERSEDED by `internal/generate`, kept as Phase A/B history) | `.claude/scratch/template-builder/` (standalone Go module) |
| T-14 benchmark report (accuracy/calibration/OOD + --think findings, session 46) | `.claude/t14-benchmark-report.md` |
| T-23 logprob-confidence probe (leaf-first; pre-mask logprobs, session 49) | `.claude/t23-logprob-confidence-probe.md` |
| T-23 strategic implications (gate route → workbook plan + grand vision, session 50) | `.claude/t23-strategic-implications.md` |
| T-32 agreement-gate report (work + findings: replay validation, verifier false-pass, fixture coupling, session 57) | `.claude/t32-agreement-gate-report.md` |
| T2 closeout report (v1.0.0 tag rationale + shape decision, session 60) | `.claude/t2-closeout-report.md` |
| T-34/T-12 red-test repair report (5 reds → 49/49 green; latent join-ID divergence finding, session 60) | `.claude/t34-t12-red-test-repair-report.md` |
| Advisor review — T-20 `--resume`/dedup design (Opus xhigh; adopted contract, session 60) | `.claude/advisor-t20-resume-dedup.md` |
| T-20 implementation report (`batch-auto --resume` + duplicate warning; [ref:batch-auto-resume] semantics block, session 60) | `.claude/t20-resume-implementation-report.md` |
| batch-auto resume acceptance (A1–A5: seeded-skip, one-of-two-dups, warning; deterministic seeds via real append path) | `expense-reporter/test/batch_auto_resume_test.go` + `test/expect/resume.go` + `test/fixtures/batch-auto-resume-*`, `batch-auto-dup-warning` |
| Join-id acceptance (T-35: both logs share one `GenerateID`) | `expense-reporter/test/apply_test.go` + `test/auto_log_append_test.go` (`*_JoinIDMatchesAcrossLogs`), `test/expect/feedback.go` (`JoinIDMatchesAcrossLogs`), fixture `test/fixtures/apply-join-id/` — fixtures use SHORT `DD/MM` on purpose; a full date hides the bug |
| Year-precedence acceptance (T-41 slice 1: `--year` flag, `date_year` fallback, explicit-year wins — deterministic, no Ollama) | `expense-reporter/test/add_year_test.go` + `test/expect/json.go` `OutputJSONHasDate` |
| Correct `--year` join acceptance (bare date + flag → GenerateID hits prior-year seed) + BR thousands CLI pin (`1.234,56`) | `expense-reporter/test/correct_year_test.go` + fixture `test/fixtures/correct-year-flag/`; `test/json_output_test.go` `TestAdd_ReadsBrazilianThousandsAmount` |
| Date/year semantics across the system (T-35 survey — six parsers, year sources, what NOT to "fix") | `.claude/t35-date-year-semantics.md` [ref:date-year-semantics] |
| T-35 implementation report (join-id fix: scope correction, RED evidence, T-37/T-38 findings, session 61) | `.claude/t35-implementation-report.md` |
| Session log archive (sessions 1–2) | `.claude/archive/session-log-2026-03-02-to-2026-03-02.md` |
| Session log archive (sessions 3–5) | `.claude/archive/session-log-2026-03-13-to-2026-03-02.md` |
| Session log archive (session 6 — 2026-03-03) | `.claude/archive/session-log-2026-03-03-to-2026-03-03.md` |
| Session log archive (session 7 — 2026-03-11) | `.claude/archive/session-log-2026-03-11-to-2026-03-11.md` |
| Session log archive (session 17 — 2026-04-23) | `.claude/archive/session-log-2026-04-23-to-2026-04-23.md` |
| Session log archive (session 18 — 2026-04-24) | `.claude/archive/session-log-2026-04-24-to-2026-04-24.md` |
| Run acceptance tests | `expense-reporter/run-acceptance.sh` — deterministic group by default (`-short`, no Ollama); `-full` = whole suite, 3600s ceiling |
| Generate-workbook acceptance fixture (G3, oracle-frozen dumps) | `expense-reporter/test/fixtures/generate-basic/` + `test/expect/workbook_structure.go` |
| Full type-routing cycle acceptance (batch-auto→review→apply→generate-workbook, incremental) | `expense-reporter/test/type_routing_cycle_test.go` + `test/fixtures/type-routing-cycle/` |
| Advisor review — G3 acceptance design | `.claude/advisor-G3-acceptance-design.md` |
| Advisor reviews — apply phase 3, Phase B builder, session 24 | `.claude/advisor-{apply-phase3,phaseB-builder,session24-review}.md` |
| Doc audit (2026-06-09) | `.claude/doc-audit-2026-06-09.md` |
| QUICK.md memory audit — 46K→16K consolidation into KNOWLEDGE.md (2026-07-01) | `.claude/quick-memory-audit-2026-07-01.md` |
| Session 42 postmortem — PR #36 review, model revert (qcoder→q3), acceptance repair (T-17/T-18) | `.claude/session42-postmortem.md` |
| WS-B slice 3 work report — `batch-auto` → log-append (session 43) | `.claude/ws-b-slice3-implementation-report.md` |
| Acceptance test patterns | `expense-reporter/test/PATTERNS.md` — [ref:acceptance-patterns] effort table + ref index |
| Acceptance test architecture | `expense-reporter/test/README.md` — [ref:acceptance-harness], [ref:acceptance-fixtures], [ref:acceptance-verify], [ref:acceptance-run] |

---

<!-- ref:go-structure -->
## Go Package Structure

| Package | Path | Purpose |
|---------|------|---------|
| `main` | `expense-reporter/cmd/expense-reporter/main.go` | Entry point |
| `cmd` | `expense-reporter/cmd/expense-reporter/cmd/` | Cobra CLI subcommands (add, batch, version; +classify/auto/batch-auto in Layer 5; +generate-workbook in Phase G) |
| `workbook-inspect` | `expense-reporter/cmd/workbook-inspect/` | Thin CLI wrapper over `internal/inspect` (workbook mapping L1); see its `.memories/QUICK.md` |
| `internal/inspect` | `expense-reporter/internal/inspect/` | Structural-dump core (values/formulas/styles/merges/rowType classifier) shared by workbook-inspect and test verifiers |
| `internal/generate` | `expense-reporter/internal/generate/` | Workbook generator (spec v2 port of the scratch builder): `Generate(Options)`, taxonomy JSON + entries JSONL loader, layout/styles/listas logic |
| `internal/batch` | `expense-reporter/internal/batch/` | Batch import orchestration |
| `internal/cli` | `expense-reporter/internal/cli/` | CLI output helpers |
| `internal/excel` | `expense-reporter/internal/excel/` | Excel workbook read/write (excelize library) |
| `internal/logger` | `expense-reporter/internal/logger/` | Logging abstraction |
| `internal/models` | `expense-reporter/internal/models/` | Domain structs: Expense, Category, SubCategory, etc. |
| `internal/parser` | `expense-reporter/internal/parser/` | CSV parsing, date/decimal normalization (BR format) |
| `internal/resolver` | `expense-reporter/internal/resolver/` | Category resolution: matches expense → (category, subcategory) |
| `internal/workflow` | `expense-reporter/internal/workflow/` | Multi-step workflow: parse → resolve → insert |
| `pkg/utils` | `expense-reporter/pkg/utils/` | Public utility functions (currency, date, format) |
| `config` | `expense-reporter/config/` | Configuration files and constants; `type-descriptions.json` (tracked, non-sensitive sidecar: expense-type → English description, merged into the classifier prompt — T-22) |
| `internal/classifier` | `expense-reporter/internal/classifier/` | Ollama classifier + `IsAutoInsertable` decision logic; `examples.go` (SelectExamples, KeywordIndex, tokenization); `loader.go` (LoadTrainingExamples, LoadFeedbackExamples, LoadKeywordIndex, MergeExamplePools) |
| `internal/taxonomy` | `expense-reporter/internal/taxonomy/` | Pure-input taxonomy domain types + loader + full-path routing (`path.go`); `descriptions.go` (`LoadTypeDescriptions`, `ApplyDescriptions` — type-description sidecar overlay, T-22) |
| `internal/config` | `expense-reporter/internal/config/` | Config struct + `Load()` + `ClassificationsFilePath()` + `ExpensesLogFilePath()` + `TaxonomyFilePath()` + `TypeDescriptionsFilePath()` |
| `internal/feedback` | `expense-reporter/internal/feedback/` | JSONL feedback logging: `Entry`, `GenerateID`, `Append`, `NewConfirmedEntry`, `NewManualEntry`; `ExpenseEntry`, `NewExpenseEntry`, `AppendExpense` (slim insert log → `expenses_log.jsonl`); `LoadExpenseIDCounts` (id-multiplicity ledger for `--resume`/dup warning — T-20) |
| `internal/appender` | `expense-reporter/internal/appender/` | Log-append path (WS-B): `ExpandAndAppend` (installment expansion → `expenses_log.jsonl`), `PredictEntryIDs` (same `expandEntries` step — T-20 resume drift guard) |
| `internal/parse` | `expense-reporter/internal/parse/` | **The T-41 parse boundary** — the ONE string→struct conversion: `ParsedExpense` (single stored `time.Time` date + `DateString()` = sole identity-bytes site), `Fields` (field-wise core; year ladder explicit > `--year` > `date_year` > most-recent-non-future, grace 0; BR thousands normalization), `ExpenseString` (semicolon wrapper, returns subcategory alongside). Slice 1 consumers: `add`/`correct` |
| `internal/review` | `expense-reporter/internal/review/` | Review command package: `ReadQueue` (7-field CSV reader), `BuildTaxonomy` (3-level tree from workbook mappings), `Render` (placeholder injection), `TemplateHTML` (go:embed); types in `types.go` |
| `harness` (module) | `github.com/leandror172/acceptance-harness` v1.0.0 | **External dep since T2 Session B** — the acceptance engine (Context, Scenario, Run, fixtures, FindModuleRoot/BuildBinary) + generic `verify` Then-closures. `test/harness/` no longer exists. Engine changes = a PR upstream + a version bump here |
| `test/actions` | `expense-reporter/test/actions/` | When-closures: RunClassify, RunAuto, RunBatchAuto, RunAdd; `runCommand` forwards `ctx.Env` |
| `test/expect` | `expense-reporter/test/expect/` | **DOMAIN** Then-closures (was `test/verify/`, renamed so the module owns `verify.*`): FeedbackFile*, ExpenseLogMatches, ClassificationsMatch, OutputFileHas*, SoftAccuracy, HTML*, WorkbookStructureMatches; ResumeSkipCount, DuplicateWarningCount (T-20) |
| `test/domain` | `expense-reporter/test/domain/` | Expense-specific test helpers the lean-core module excludes: RequireWorkbook + CopyWorkbookToWorkDir, SetupBinaryConfig, ExpenseFixtureConfig (`Raw` decode), ctx.Env accessors (DataDir/WorkbookPath) |
| `test/extern` | `expense-reporter/test/extern/` | External-service gates: RequireOllama (module core is LLM-free/network-free by design) |
| `test/fixtures` | `expense-reporter/test/fixtures/` | Fixture data per functional slice (classify-basic, batch-auto-basic, batch-auto-exclusions, batch-auto-feedback) |
<!-- /ref:go-structure -->

---

<!-- ref:testing -->
## Testing Conventions

- **Test runner:** `cd expense-reporter && go test ./...`
- **Framework:** `testing` package (stdlib) + testify (`assert`/`require`) — both in use
- **Unit test style:** testify preferred for new tests; existing stdlib tests left as-is
- **Coverage:** 263 top-level test functions / 647 including table-driven subtests, all green
  (counted 2026-07-21 via `go test ./... -v | grep -c '^=== RUN'`)
- **Pattern:** Each `internal/X/` has `X_test.go` with table-driven cases
- **Test data:** `expense-reporter/expenses.example.csv` — safe, no personal data
- **TDD approach:** Write failing test first, implement to pass (established pattern from Phases 1-4)

**Do NOT:**
- Use `testing.T.Fatal` where `t.Errorf` suffices
- Skip table-driven approach for new commands
<!-- /ref:testing -->

---

<!-- ref:classification -->
## Classification Data

See `data/classification/classification_algorithm.md` for the full algorithm description.
See `data/classification/reproducibility_guide.md` for how to reproduce results.

**Algorithm summary:** Hybrid rule-based + statistical scoring. Priority: precision over recall.
**Training corpus** (`training_data_complete.json`): **1,788** labeled expenses — 15 categories,
81 subcategories. By year: 2022:127 / 2023:853 / 2024:349 / 2025:459. Built 2026-06-20 by the
5.R4 historical extraction (2022–2025 workbooks deduped + merged into the original 694 corpus).
**Keyword dictionary** (`feature_dictionary_enhanced.json`): 229 keywords, 68 subcategories,
71 user-correction override rules.

⚠️ **The two artifacts are OUT OF SYNC.** The dictionary was never regenerated after 5.R4, so it
still describes the 694-era taxonomy: **13 of the corpus's 81 subcategories have no keyword entry
at all** and can never produce a keyword match. Do not quote a single "coverage" number for both —
they are different files with different scopes. This gap feeds the keyword-miss stratum that 5.R2
embedding retrieval targets ([ref:embedding-retrieval]).
**Layer 5 strategy:** Feature dictionary as system context + top-K few-shot examples per request.

**Key ref blocks for Layer 5 agents:**
- `ref-lookup.sh training-data-schema` — JSON schema for `training_data_complete.json` and `feature_dictionary_enhanced.json`; classify command input format
- `ref-lookup.sh confidence-thresholds` — auto-insert AGREEMENT gate (T-32) + legacy confidence scoring
- `ref-lookup.sh classification-overview` — executive summary + algorithm performance stats
<!-- /ref:classification -->

---

<!-- ref:training-data-schema -->
## Training Data Schema

### `training_data_complete.json` (1,788 labeled expenses — gitignored)

```
{
  "metadata": {
    "total_expenses": 1788,       // post-5.R4; was 694 pre-2026-06-20
    "unique_categories": 15,
    "unique_subcategories": 81,
    "by_year": {"2022": 127, "2023": 853, "2024": 349, "2025": 459},
    "extraction_date": "ISO 8601",
    "provenance": "free text — how this corpus was assembled"
  },
  "expenses": [
    {
      "id": 1,                    // sequential integer
      "item": "Diarista Letícia", // raw description as it appears in the spreadsheet
      "date": "2024-01-05",       // ISO 8601 (YYYY-MM-DD); source data is DD/MM/YYYY
      "value": 160.0,             // float, BRL
      "subcategory": "Diarista",  // ground truth label
      "category": "Habitação",    // parent category
      "source": "Filename.xlsx:SheetName",
      "year": 2024
    }
  ]
}
```

### `feature_dictionary_enhanced.json` (229 keywords, 68 subcategories — gitignored)

⚠️ Built from the **694-era** corpus and never regenerated after 5.R4. 13 of the training
corpus's 81 subcategories are absent from `category_mapping` entirely — those leaves cannot
produce a keyword match, and therefore cannot pass the T-32 agreement gate
([ref:confidence-thresholds]).

```
{
  "lexical_features": {
    "keywords": {
      "<lowercase_keyword>": {
        "frequency": 17,                    // occurrence count in training set
        "dominant_subcategory": "Diarista", // most common subcategory for this keyword
        "dominant_count": 17,               // how many times the dominant mapping occurred
        "specificity": 1.0,                 // dominant_count / frequency (1.0 = unambiguous)
        "idf": 3.652,                       // inverse document frequency (rarity signal)
        "subcategories": ["Diarista"]       // all subcategories this keyword appears in
      }
    }
  },
  "value_ranges": {
    "<subcategory>": { "min": 160.0, "max": 219.8, "mean": 176.19,
                       "median": 172.0, "q1": 160.0, "q3": 172.0, "count": 17 }
  },
  "category_mapping": {
    "<subcategory>": "<parent_category>"    // 68 entries; flat lookup
  },
  "user_corrections": {
    "<normalized_item_lowercase>": {
      "category": "Lazer",
      "subcategory": "Delivery",
      "full_item": "Delivery brod's"        // original casing
    }
  }
}
```

### Classify command input (for 5.2)

Three fields passed to the classifier:
- `item` — expense description string (free text, Portuguese)
- `value` — float (BRL)
- `date` — string DD/MM/YYYY (Brazilian format; normalize before lookup)

### How these files are used in Layer 5

| File | Role at classify time |
|------|----------------------|
| `training_data_complete.json` | Source of few-shot examples (top-K by keyword similarity, injected into prompt) |
| `feature_dictionary_enhanced.json` | Fast pre-filter: keyword lookup → candidate subcategory before calling Ollama; also value range plausibility check |
| `algorithm_parameters.json` | Threshold values (tracked) — HIGH ≥ 0.85, MEDIUM ≥ 0.50, LOW < 0.50. **These no longer gate anything** — they are the desktop-era scoring bands, retained as the design record. Auto-insert is decided by the agreement gate ([ref:confidence-thresholds]) |
<!-- /ref:training-data-schema -->

---

<!-- ref:confidence-thresholds -->
## Auto-Insert Gate (agreement gate — T-32)

**As of T-32, auto-insert is decided by keyword AGREEMENT, not confidence.** The
649-replay (session 52) measured that the model's self-reported confidence is
uninformative as a gate (the whole 0.85–0.95 band is a coin flip), so
`IsAutoInsertable` (`internal/classifier/decision.go`) no longer consults confidence.

A candidate is auto-insertable (`auto`/`batch-auto` append it without review) only when
ALL hold:
- a keyword matched the item (a specificity signal exists)
- the match is unambiguous (no tie at the top specificity)
- the top keyword specificity is maximal (`top_score >= 1.0`)
- the model's predicted subcategory equals the keyword's dominant subcategory (agreement)
- the subcategory is not in `auto_insert_excluded` (e.g. `Diversos`)

Everything else routes to review. Confidence is still emitted per candidate (and shown in
`--json`) but does NOT gate; `--threshold` on `batch-auto` is deprecated/ignored.

**Precision caveat:** the ~95% subcat precision this gate showed on the 649-replay is a
same-sample RELATIVE validation (shipped predicate == measured predicate). The 649 is a
confidence-selected subset, so ABSOLUTE production precision is unmeasured — which is why
unattended silent auto-insert (WS-D) stays held.

**Legacy (pre-T-32) confidence scoring** — retained as the classifier's internal scoring,
no longer a gate. Source `data/classification/algorithm_parameters.json`: HIGH ≥ 0.85,
MEDIUM ≥ 0.50, LOW < 0.50; fallback (no keyword + inconclusive value range) → `Diversos`,
confidence `0.30`. Feature weights: keyword_match 0.50, semantic_similarity 0.30,
value_proximity 0.20.
<!-- /ref:confidence-thresholds -->

---

<!-- ref:classification-overview -->
## Classification Overview

**Dataset:** 1,788 labeled expenses, 2022–2025, Brazilian Portuguese descriptions, BRL values.
**Taxonomy (corpus):** 15 categories → 81 subcategories.
**Keyword dictionary:** 229 terms (lowercase, with IDF and specificity scores) covering only
68 subcategories — 694-era, not regenerated after 5.R4. See the warning in
[ref:training-data-schema].
**User corrections:** 71 override rules (exact normalized item → forced subcategory).

⚠️ The algorithm description below is the **desktop-era hybrid scorer**, retained as the design
record. It is NOT the shipped auto-insert path: since T-32 the gate is keyword agreement, not a
confidence band — see [ref:confidence-thresholds].

**Algorithm (hybrid, priority order):**
1. User correction lookup (exact match on normalized item string) → confidence 1.0
2. Keyword match scoring (TF-IDF weighted, specificity boosted) → up to 0.85
3. Value range plausibility (Gaussian kernel against per-subcategory IQR) → modifier ±0.20
4. Fallback → Diversos, 0.30

**Key preprocessing:**
- Lowercase, remove special chars, normalize whitespace
- Do NOT strip accents (Portuguese keywords retain accents: `gás`, `habitação`)
- Minimum word length: 2 chars

**Performance (from Desktop-era analysis):**
- High-specificity keywords (specificity = 1.0): unambiguous mapping, no LLM needed
- Ambiguous keywords (e.g., `va` maps to 5 subcategories): value range resolves most cases
- Remaining ambiguous: few-shot LLM call via Ollama
<!-- /ref:classification-overview -->

---

<!-- ref:feedback-system -->
## Feedback System & Corrections

**File:** `docs/FEEDBACK_SYSTEM.md`

The feedback system logs classification decisions to `classifications.jsonl` for training, tuning, and audit purposes.

**Key ref blocks:**
- `[ref:feedback-entry-structure]` — JSON entry format, status values (confirmed/corrected/manual)
- `[ref:feedback-sources]` — Which commands create which feedback entries (add, auto, batch-auto)
- `[ref:feedback-training]` — How feedback becomes training examples for next run
- `[ref:feedback-correction-workflow]` — Correction workflow via `correct` command (status="corrected"), closed in Layer 5.9
- `[ref:feedback-file-path]` — Configuration and file resolution logic
- `[ref:feedback-cold-start]` — Behavior when classifications.jsonl doesn't exist

**Quick summary:**
- `add` command → `status="manual"` (user entered, no model)
- `auto`/`batch-auto` commands → `status="confirmed"` (model auto-appended past the agreement gate)
- `correct` command (`cmd/correct.go`) → `status="corrected"` — requires a prior entry to override
- `apply` (review-UI ingestion) → `status="confirmed"` or `"corrected"` per the reviewer's action;
  entries it writes carry `model: "review"`

**Closed in Layer 5.9** — the old "no way to create `corrected`" feature gap no longer exists.
In the live log, `apply` is in fact the dominant producer of both statuses.

<!-- /ref:feedback-system -->

---

## Retrieval Strategy Docs

| File | Tracked | Purpose |
|------|---------|---------|
| `data/classification/retrieval-strategy.md` | ✅ Yes | High-level retrieval pipeline: cascade diagram, token budget, data source strategy — [ref:retrieval-strategy], [ref:retrieval-token-budget] |
| `data/classification/tfidf-retrieval.md` | ✅ Yes | TF-IDF retrieval layer: existing artifacts, implementation approach, decision criteria — [ref:tfidf-retrieval] |
| `data/classification/embedding-retrieval.md` | ✅ Yes | Embedding/RAG retrieval layer: Ollama API, vector store, multilingual considerations — [ref:embedding-retrieval] |

---

## Classification Data Files

| File | Tracked | Purpose |
|------|---------|---------|
| `data/classification/classification_algorithm.md` | ✅ Yes | Algorithm description — no personal data |
| `data/classification/reproducibility_guide.md` | ✅ Yes | How to reproduce results |
| `data/classification/algorithm_parameters.json` | ✅ Yes | Parameter values — no personal data |
| `data/classification/AUTO_CATEGORY_README.md` | ✅ Yes | Overview of the auto-categorization effort |
| `data/classification/FINAL_SUMMARY.md` | ✅ Yes | Final analysis summary |
| `data/classification/classification_reasoning.md` | ✅ Yes | Reasoning behind category assignments |
| `data/classification/research_insights.md` | ✅ Yes | Research findings |
| `data/classification/llm_reasoning_meta.md` | ✅ Yes | LLM reasoning analysis |
| `data/classification/feature_dictionary_enhanced.json` | ❌ Gitignored | Personal expense keywords |
| `data/classification/feature_dictionary.json` | ❌ Gitignored | Personal expense keywords |
| `data/classification/training_data_complete.json` | ❌ Gitignored | Labeled expenses (personal) |
| `data/classification/training_data.json` | ❌ Gitignored | Labeled expenses (personal) |
| `data/classification/final_classifications.json` | ❌ Gitignored | Classification results |
| `data/classification/*.csv` | ❌ Gitignored | Personal expense CSVs |
| `data/classification/similarity_matrix.json` | ❌ Gitignored | Computed similarity data |
| `data/classification/vector_representations.json` | ❌ Gitignored | Feature vectors |
| `data/classification/statistical_summary.json` | ❌ Gitignored | Statistical results |
| `data/classification/confusion_analysis.json` | ❌ Gitignored | Per item analysis (may contain real descriptions) |
| `data/classification/extraction-aliases.json` | ❌ Gitignored | 5.R4 source→taxonomy alias map (taxonomy-label fragments); loaded by `extract_old_workbooks.py` |

---

## Vision & Planning Docs (docs/)

| File | Content |
|------|---------|
| `docs/expense-classifier-vision.md` | End-to-end Layer 5–6+ vision: user scenario, domain boundaries, iterative build plan (Phases 0–5), technical notes on structured output, persistence, queue. **Primary reference for understanding scope and architecture.** |
| `docs/expense-classifier-data-inventory.md` | Inventory of all auto-category analysis artifacts (feature dict, training data, confusion analysis, etc.) and their priority/role at build time. Also documents expense-reporter architecture as of Layer 5 start. |

---

## Historical Docs (docs/archive/)

Desktop-era planning documents — read for context, do not modify.

| File | Content |
|------|---------|
| `ARCHITECTURE_PLAN_Expense_Automation.md` | Original automation architecture plan |
| `ARCHITECTURE_PLAN_Go_Implementation.md` | Go implementation architecture |
| `AUTO_CATEGORIZATION_PROMPT.md` | Original auto-categorization prompt design |
| `DECISIONS_FINALIZED.md` | Finalized design decisions |
| `GO_vs_PYTHON_Comparison.md` | Language selection rationale |
| `PRE_IMPLEMENTATION_CHECKLIST.md` | Pre-implementation verification |
| `PRE_PHASE1_VERIFICATION_CHECKLIST.md` | Phase 1 checklist |
| `QUICK_REFERENCE_Implementation_Summary.md` | Implementation quick reference |
| `SHEET_NORMALIZATION_PROMPT.md` | Sheet normalization prompt |
| `USAGE_CALCULATION.md` | Usage/quota calculations |
| `DOCUMENTATION_INDEX.md` | Original doc index (superseded by this file) |
| `PROJECT_COMPLETE.md` | Phase completion summary |
| `BATCH_IMPORT_COMPLETE.md` | Batch import feature completion |
| `PHASE1-4_COMPLETE.md` | Combined phase completion doc |
| `FEATURES_INVESTIGATION_REPORT.md` | Feature investigation results |
| `FEATURE_INVESTIGATION.md` | Feature investigation notes |
| `TESTING.md` | Testing documentation (current: expense-reporter/README.md) |

---

## Per-Folder Memories

| Folder | Files | Content |
|--------|-------|---------|
| `.memories/` | QUICK.md, KNOWLEDGE.md | Repo-wide status, architecture, cross-repo relationships |
| `expense-reporter/.memories/` | QUICK.md, KNOWLEDGE.md | Go app structure, command hierarchy, batch pipeline, config |
| `expense-reporter/internal/apply/.memories/` | QUICK.md | Review UI ingestion, workbook row insertion, feedback + expense log output; entry type drop point (Plan A) |
| `expense-reporter/internal/classifier/.memories/` | QUICK.md, KNOWLEDGE.md | Few-shot algorithm, prompt architecture, empirical findings |
| `expense-reporter/internal/feedback/.memories/` | QUICK.md, KNOWLEDGE.md | Two-file JSONL structure (classifications.jsonl + expenses_log.jsonl), GenerateID join key, type persistence gap (Plan A) |
| `expense-reporter/internal/parse/.memories/` | QUICK.md | T-41 parse boundary: contract (single time.Time + DateString identity bytes), year ladder + grace 0, BR thousands rule, migration status |
| `expense-reporter/internal/review/.memories/` | QUICK.md | HTML review page builder, taxonomy re-derivation from CSV, localStorage state, type-aware UI already in place |
| `expense-reporter/internal/taxonomy/.memories/` | QUICK.md, KNOWLEDGE.md | Full-path identity, bare-name routing + ambiguous fallback, classifier/generator taxonomy disconnect (Plan B two-tier routing) |
| `expense-reporter/test/.memories/` | QUICK.md, KNOWLEDGE.md | BDD harness design, fixture format, soft/hard assertions |
| `mcp-server/.memories/` | QUICK.md, KNOWLEDGE.md | Thin wrapper decisions, binary resolution, data-dir fix |
| `expense-reporter/internal/excel/.memories/` | QUICK.md, KNOWLEDGE.md | Reference-sheet columns, boundary detection, and the workbook structural map (sheet families, palette, fill-down vs merge, separators, cross-sheet wiring) |
| `expense-reporter/cmd/workbook-inspect/.memories/` | QUICK.md | workbook-inspect tool: usage, output schema, classifier + row-fill design |
| `expense-reporter/internal/generate/.memories/` | QUICK.md, KNOWLEDGE.md | generator entry points, sheet-order rule, re-freeze discipline, excelize gotchas; KNOWLEDGE: build order, accumulating row counter, cross-fixture oracle coupling, column layout, block sizing, income 3-level model |
| `.claude/workbook-dump/` | *.json (gitignored) | Raw JSON dumps from workbook-inspect; input for Layer 2 visual annotation and Layer 3 spec |

---

## READMEs

| File | Content |
|------|---------|
| `README.md` | Repo root — project overview, components, quick start |
| `expense-reporter/README.md` | Full CLI documentation — all commands, classification system, testing |
| `expense-reporter/test/README.md` | Acceptance test harness architecture, fixtures, verifiers |

---

## Tools

| Tool | Path | Purpose |
|------|------|---------|
| `sonnet-max-subagent.js` | `.claude/workflows/sonnet-max-subagent.js` | Workflow harness: run ONE Sonnet 5 subagent at max/xhigh effort (the effort knob the Agent tool lacks). `Workflow({name:'sonnet-max-subagent', args:'<prompt>'})` or `args:{prompt,effort}` |
| `impl-opus-med` agent | `.claude/agents/impl-opus-med.md` | Opus medium-effort implementation subagent (TDD, directed reading, local-model delegation w/ per-session persona override; copied from latent-topic-graph, adapted session 54) |
| `impl-opus-xhigh` agent | `.claude/agents/impl-opus-xhigh.md` | Opus xhigh-effort implementation subagent for the hardest TDD tasks (multi-file features, invariant-heavy/advisor-contracted designs); same contract as impl-opus-med, advisor rule made availability-conditional (session 60) |
| `resume.sh` | `.claude/tools/resume.sh` | Session-start context summary |
| `ref-lookup.sh` | `.claude/tools/ref-lookup.sh` | Resolve [ref:KEY] tags |
| `rotate-session-log.sh` | `.claude/tools/rotate-session-log.sh` | Archive old session log entries |
| `reconstruct-csvs.py` | `.claude/tools/reconstruct-csvs.py` | Reconstruct classified/review CSVs from batch-auto log + original input CSV (line-matched) |
| `lookup-category.py` | `.claude/tools/lookup-category.py` | Look up canonical category for one or more subcategories (`<sub> [...]` or `--list`) |
| `backfill-type.py` | `.claude/tools/backfill-type.py` | Backfill expense type into log files from reviewed.json exports (Plan A Phase B-fill recovery) |
| 5.R4 extraction scripts | `.claude/scratch/{extract_old_workbooks,dedup_corpus,build_corpus,build_logs}.py` | One-off (session 35): 2022–2024 workbooks → deduped corpus + per-year logs. Alias map externalized to gitignored `extraction-aliases.json`. |
| WS-A.3 merge script | `.claude/scratch/merge_year_logs.py` |
| T-14 benchmark harness | `.claude/scratch/t14-benchmark/{build_sample,run_benchmark,score}.py` + `ood.jsonl` (sample/results JSONLs gitignored — real expense data). Sample builder, resumable runner (drives `classify --json`; `--prefix`/`--extra-args`), scorer (accuracy/leakage/calibration/OOD). | One-off (session 37): merge per-year `expenses_log-{2022,2023,2024}.jsonl` + base 2025 → one `expenses_log-allyears.jsonl`, rewriting `DD/MM`→`DD/MM/YYYY`. Output gitignored. Verified byte-identical (excl. manifest source) vs per-year `generate-workbook --year N` for all 4 years. |
| 649-replay harness (T-23 gate + 5.R1) | `expense-reporter/internal/classifier/replay_{retrieval,model}_test.go` (`//go:build replay`) + `.claude/scratch/replay-649/{FINDINGS,FINDINGS-model,FINDINGS-nn,FINDINGS-5r2}.md` + `analyze*.py`/`compare_think.py`/`nn_precondition.py` | Session 52: replays the 649 real labels through production retrieval + classifier (LOO). Verdict: confidence dead as gate, specificity+agreement is the gate, 5.R2 (not 5.R1) is the lever. Session 53: T-31 NN precondition on the 160 misses → GO for 5.R2 (hit@5 62–65% multilingual; arctic-embed2 practical pick). Raw `*.jsonl` gitignored (real data). |
| session-handoff skill | `.claude/skills/session-handoff/SKILL.md` | End-of-session tracking workflow |
