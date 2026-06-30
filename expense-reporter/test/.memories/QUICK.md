# test/ — Quick Memory

*Working memory for the acceptance test harness. Injected into agents. Keep under 30 lines.*

## Status
Harness operational. 15+ fixture directories covering classify, auto, batch-auto, feedback,
correct, installments, rollover, exclusion, add-with-prediction, apply, review, and
**generate-workbook** (`generate-basic` — NEW fixture sub-format: taxonomy.json +
entries.jsonl + oracle-frozen `expected-dump-*/`, NOT config.json+input.csv; see PATTERNS.md
"Generate-Workbook Fixture Sub-Format"). Workbook structure assertions live:
`verify.WorkbookStructureMatches(expectedDumpDir)` on `internal/inspect` dumps.
`generate-income` fixture + `TestGenerateWorkbook_IncomeRoute` (WS-C, session 38) cover the
3-level income route: nested `incomeCategories` taxonomy + `income-entries.jsonl` (extractor
schema) via `--income-entries`, oracle-frozen `expected-dump-data`. Asserts signed sums land
(Salário Jan 4150 etc.) + per-Block Listas rollup. NOTE: changing the income/summary render
re-freezes BOTH this AND `generate-basic`'s data+skeleton oracles (income summary shifts Listas).
`type_routing_cycle_test.go` (`type-routing-cycle` fixture) is an INCREMENTAL full-cycle
suite (batch-auto→review→apply→generate-workbook): each test = one CLI step, each folds prior
steps into Given; the last seeds the cumulative typed log and runs only generate-workbook,
asserting the ambiguous leaf (Dentista ∈ Variáveis+Extras) routes by type. T1 is Ollama-gated
(~38s); T2-T4 are deterministic (<0.05s). apply's skeleton workbook is built hermetically via
generate-workbook in the Given.
**WS-B slice 3 (session 43) — batch-auto acceptance retargeted to the LOG.** batch-auto fixtures now assert
`expenses_log.jsonl` via `verify.ExpenseLogMatches(<fixDir>/expected-expenses_log.jsonl)` (field-subset,
line-exact, skips id/timestamp), NOT workbook rows. `batch-auto-typed` = canonical NON-dry-run append anchor
(workbook gate dropped). `batch-auto-installments` asserts the N expanded dated log lines; `batch-auto-rollover`
INVERTED → `verify.NoRolloverFileCreated()` + next-year dates in the log (rollover.csv retired). The
failure-downgrade is a **unit** test (`cmd.TestAppendClassified_DowngradesRowOnAppendFailure`), NOT acceptance —
the pre-flight makes an acceptance-level append failure unreachable; acceptance covers the pre-flight via
`TestBatchAuto_UnwritableLogPath_FailsFastBeforeClassification` (deterministic, no RequireOllama: parent-is-a-file
log path → MkdirAll fails before classifying).

## Structure
```
harness/             # Domain-agnostic BDD engine (extractable to other repos)
  scenario.go        # Context, Scenario, Run() — Given/When/Then with function injection
  fixture.go         # FixtureConfig, CopyFixtureToWorkDir, DiscoverFixtures
  comparator.go      # CSV comparison (semicolon-delimited, comment-aware)
  ollama.go          # RequireOllama — t.Skip if Ollama unavailable
actions/             # When functions — CLI command runners
  commands.go        # RunClassify, RunAuto, RunAdd, RunCorrect, RunBatchAuto, RunReview, RunApply, RunGenerateWorkbook
verify/              # Then functions — composable assertion helpers
  csv.go             # Structural: row count, column count, file existence, exit code (CommandSucceeded/Failed)
  accuracy.go        # Soft accuracy + JSON drift reports
  feedback.go        # JSONL log verification (classifications + expense log)
  workbook.go        # Stub for future workbook content checks
fixtures/            # Test data directories (config.json + input.csv + expected-*.csv)
results/             # Gitignored — accuracy drift reports across runs
```

## Key Rules
- **Build tag `acceptance`** — never runs with `go test ./...`, requires `-tags=acceptance`
- **Live Ollama required** — tests skip gracefully via `t.Skipf` if Ollama is unreachable
- **Binary built once** — `TestMain` in `setup_test.go` builds to temp dir, shared by all tests
- **Composable Then** — helpers return `[]func(*Context)`, combined with `slices.Concat`
- **Fixture threshold 0.0** — mechanics tests (installments, rollover) use threshold 0.0
  to decouple from classifier non-determinism
- **Explicit-year fixture inputs (slice 3):** the append path reformats dates to `DD/MM/YYYY`
  (`ParseDateFlexible` fills bare `DD/MM` with `time.Now().Year()` — a year time-bomb). Non-dry-run
  batch-auto/auto/add fixtures MUST use `DD/MM/YYYY` inputs + explicit-year expected logs, and clean-dividing
  installment values (e.g. `90,00/3`→30) to avoid float JSON drift in `expected-expenses_log.jsonl`.
- **Dry-run hides the append path:** a fixture with `--dry-run` in `extra_args` (batch-auto-basic/exclusions/
  type-routing-cycle) never appends — those tests prove CSV production ONLY. Append coverage needs a NON-dry-run
  anchor (`batch-auto-typed`, `batch-auto-feedback`). Check `extra_args` before trusting a green batch-auto test
  to cover the log. (`RequireWorkbook` also SKIPs when the test workbook is absent — that hid a stale
  non-dry-run feedback test through the whole session-42 sweep.)
- **Taxonomy config is MANDATORY since T-13 (session 41+):** `classify`/`auto`/`batch-auto`/`add`/`correct`
  now hard-require a configured `taxonomy_path` (loaded before classification). Every `Given` that
  invokes them must call `withFeedbackAndTaxonomyConfig(ctx, fixDir)` (reads `fixDir/fixture-taxonomy.json`,
  copies it to WorkDir, writes binary config) OR `SetupBinaryConfig` with `taxonomy_path`. Missing it →
  `Error: taxonomy path not configured`. The fixture taxonomy must cover the input CSV's expected leaves
  (accuracy tests compare subcategory only). Session 42 added `fixture-taxonomy.json` to batch-auto-basic,
  batch-auto-exclusions, correct-*, etc.
- **`requireDataDir(t)`** (fewshot_test.go) — skip-guard mirroring `RequireOllama` for tests that read
  the gitignored `data/classification` (feature dict / training data). NOTE: that dir DOES exist at the
  **repo root** (`expense-reporter/../data/classification`), not under `expense-reporter/`.
- **Build-tag hides regressions (session 42 lesson):** because `//go:build acceptance` excludes these from
  `go test ./...`, a whole class of breakage (e.g. T-13 making taxonomy mandatory) can ship green. After any
  change to a mandatory field / config contract, run `-tags=acceptance` explicitly. [[feedback_rename_json_tag_acceptance]]
- **Infrastructure timeout (refreshed session 42):** q3 is slow (~12s/classify → a 10-row batch ≈ 2 min),
  so the full suite blows the **600s default** `go test` timeout. Run with `-timeout 30m` (single shot) or
  in sub-groups during dev. The classify-heavy batch tests dominate wall-clock.

## Deeper Memory → KNOWLEDGE.md
- **Harness design** — domain-agnostic engine vs domain-specific actions/verify
- **Fixture format** — config.json schema, CSV conventions, expected file patterns
- **Accuracy tracking** — soft vs hard assertions, drift detection strategy
