# Acceptance Test Suite

File-driven BDD harness for end-to-end testing of CLI commands against a real Ollama instance.

---

<!-- ref:acceptance-harness -->
## Harness Architecture

The harness follows a Given/When/Then pattern with function injection:

```
test/
  harness/          -- domain-agnostic engine (extractable to another repo)
    scenario.go     -- Context, Scenario, Run()
    fixture.go      -- FixtureConfig, CopyFixtureToWorkDir, DiscoverFixtures
    comparator.go   -- ReadCSVFile (semicolon, comment-aware), CompareCSVExact/Fuzzy
    ollama.go       -- RequireOllama (t.Skip on failure)
  actions/          -- domain: When functions (command runners)
    commands.go     -- RunClassify, RunAuto, RunBatchAuto, RunBatchAutoWithFixture
  verify/           -- domain: Then functions (composable assertions)
    csv.go          -- structural assertions (rows, columns, files, exit code)
    accuracy.go     -- soft accuracy + drift tracking to test/results/
    workbook.go     -- stub for future workbook content assertions
```

**Context** holds per-scenario state: `BinaryPath`, `WorkDir` (temp), `FixtureDir`, `Artifacts map[string]string`, `Stdout`, `Stderr`, `ExitCode`.

**Scenario** has three phases:
- `Given func(*Context)` — set up: copy fixtures, set binary path, prepare workbook
- `When func(*Context)` — action: run the CLI command
- `Then []func(*Context)` — assertions: composable, order-independent checks

**Run(t, Scenario)** wraps everything in `t.Run(s.Name, ...)` so each scenario is a named subtest.
<!-- /ref:acceptance-harness -->

---

<!-- ref:acceptance-fixtures -->
## Fixture Format

Each fixture is a directory under `test/fixtures/` containing at minimum:

```
fixtures/<name>/
  config.json             -- required: command, model, threshold, assertion_type, extra_args
  input.csv               -- test data (3-field for batch-auto, scenario table for classify/auto)
  expected-classified.csv -- optional: soft accuracy reference
```

**config.json schema:**

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `command` | string | — | `"classify"`, `"auto"`, or `"batch-auto"` |
| `model` | string | — | Ollama model name (e.g. `"my-classifier-q3"`) |
| `threshold` | float | 0.85 | Confidence threshold for auto-insert |
| `assertion_type` | string | `"hard"` | `"hard"` (fail on mismatch) or `"soft"` (warn above floor) |
| `accuracy_floor` | float | 0.0 | Minimum accuracy for soft assertions |
| `top_n` | int | 3 | Classification candidates |
| `extra_args` | []string | [] | Additional CLI flags (e.g. `["--dry-run"]`) |

**CSV format:** Semicolon-delimited. Lines starting with `#` are comments (stripped by `ReadCSVFile`).

**classify/auto fixtures** use `input.csv` as a scenario table — each row becomes a separate `harness.Run` call with CLI args extracted from the row. The file is NOT passed to the binary.

**batch-auto fixtures** pass `input.csv` directly to the binary as the `<csv_file>` argument.
<!-- /ref:acceptance-fixtures -->

---

<!-- ref:acceptance-verify -->
## Available Verifiers

### Structural (`test/verify/csv.go`)

| Function | Signature | Assertion |
|----------|-----------|-----------|
| `CommandSucceeded` | `() func(*Context)` | Last command exited 0 |
| `OutputFileExists` | `(artifactKey) func(*Context)` | Artifact file exists on disk |
| `OutputFileHasRows` | `(artifactKey, n) func(*Context)` | CSV has exactly n rows |
| `OutputFileHasAtLeastRows` | `(artifactKey, n) func(*Context)` | CSV has >= n rows |
| `OutputFileHasColumns` | `(artifactKey, n) func(*Context)` | Every row has n columns |
| `AllClassificationScoresValid` | `(artifactKey) func(*Context)` | Confidence column values in [0.0, 1.0] |
| `NoExpenseInBothFiles` | `(artifact1, artifact2) func(*Context)` | No row in both files |
| `OutputContains` | `(substr) func(*Context)` | stdout+stderr contains substr |
| `OutputNotContains` | `(substr) func(*Context)` | stdout+stderr does not contain substr |

### Accuracy (`test/verify/accuracy.go`)

| Function | Signature | Assertion |
|----------|-----------|-----------|
| `ClassificationAccuracyAtLeast` | `(artifactKey, expectedPath, floor, resultsDir) func(*Context)` | Category accuracy >= floor; writes JSON report to resultsDir |
| `NoneWereAutoInserted` | `(artifactKey) func(*Context)` | Every row has `auto_inserted == "false"` |

### Column indices for batch-auto output (classified.csv / review.csv)

| Col | 0 | 1 | 2 | 3 | 4 | 5 | 6 |
|-----|---|---|---|---|---|---|---|
| Field | item | date | value | subcategory | category | confidence | auto_inserted |
<!-- /ref:acceptance-verify -->

---

<!-- ref:acceptance-run -->
## How to Run

**Build tag:** All acceptance test files use `//go:build acceptance`. They never run with `go test ./...`.

**Via script (recommended):**
```bash
cd expense-reporter && ./run-acceptance.sh
```
Pre-flight: checks Ollama reachability, verifies `go build`, then runs tests with 300s timeout.

**Directly:**
```bash
cd expense-reporter && go test -tags=acceptance -v -timeout 300s ./test/...
```
If Ollama is not running, tests skip gracefully via `t.Skipf`.

**Binary lifecycle:** `TestMain` in `setup_test.go` builds the binary once into a temp dir. All test files share it via the package-level `binaryPath` variable.

**Drift tracking:** `SoftAccuracy` writes JSON reports to `test/results/` (gitignored). Compare across runs to track classification accuracy changes over model/prompt updates.
<!-- /ref:acceptance-run -->
