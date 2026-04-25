# test/ — Quick Memory

*Working memory for the acceptance test harness. Injected into agents. Keep under 30 lines.*

## Status
Harness operational. 10 fixture directories covering classify, auto, batch-auto, feedback,
correct, installments, rollover, and exclusion scenarios. Drift tracking active via
`test/results/`. Future: workbook content assertions (stub exists).

## Structure
```
harness/             # Domain-agnostic BDD engine (extractable to other repos)
  scenario.go        # Context, Scenario, Run() — Given/When/Then with function injection
  fixture.go         # FixtureConfig, CopyFixtureToWorkDir, DiscoverFixtures
  comparator.go      # CSV comparison (semicolon-delimited, comment-aware)
  ollama.go          # RequireOllama — t.Skip if Ollama unavailable
actions/             # When functions — CLI command runners
  commands.go        # RunClassify, RunAuto, RunBatchAuto, RunBatchAutoWithFixture
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

## Deeper Memory → KNOWLEDGE.md
- **Harness design** — domain-agnostic engine vs domain-specific actions/verify
- **Fixture format** — config.json schema, CSV conventions, expected file patterns
- **Accuracy tracking** — soft vs hard assertions, drift detection strategy
