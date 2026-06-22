# test/ — Quick Memory

*Working memory for the acceptance test harness. Injected into agents. Keep under 30 lines.*

## Status
Harness operational. 15+ fixture directories covering classify, auto, batch-auto, feedback,
correct, installments, rollover, exclusion, add-with-prediction, apply, review, and
**generate-workbook** (`generate-basic` — NEW fixture sub-format: taxonomy.json +
entries.jsonl + oracle-frozen `expected-dump-*/`, NOT config.json+input.csv; see PATTERNS.md
"Generate-Workbook Fixture Sub-Format"). Workbook structure assertions live:
`verify.WorkbookStructureMatches(expectedDumpDir)` on `internal/inspect` dumps.
`type_routing_cycle_test.go` (`type-routing-cycle` fixture) is an INCREMENTAL full-cycle
suite (batch-auto→review→apply→generate-workbook): each test = one CLI step, each folds prior
steps into Given; the last seeds the cumulative typed log and runs only generate-workbook,
asserting the ambiguous leaf (Dentista ∈ Variáveis+Extras) routes by type. T1 is Ollama-gated
(~38s); T2-T4 are deterministic (<0.05s). apply's skeleton workbook is built hermetically via
generate-workbook in the Given.

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
- **Infrastructure timeout gotcha (session 18):** Full 600s acceptance suite times out mid-flight. Basic test ~286s + MixedConfidence ~299s = 585s remaining. New tests should be fast (<5s) or run separately. Run individual test classes, not full suite, during development.

## Deeper Memory → KNOWLEDGE.md
- **Harness design** — domain-agnostic engine vs domain-specific actions/verify
- **Fixture format** — config.json schema, CSV conventions, expected file patterns
- **Accuracy tracking** — soft vs hard assertions, drift detection strategy
