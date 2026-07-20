# Acceptance Test Suite

File-driven BDD harness for end-to-end testing of CLI commands against a real Ollama instance.

---

<!-- ref:acceptance-harness -->
## Harness Architecture

The Given/When/Then engine is the **`github.com/leandror172/acceptance-harness` module**
(pinned in `go.mod`), not local code. It was extracted in T2 Session B once a second
consumer (career-search's `roles` CLI) wanted the same engine. Everything expense-shaped
stayed here:

```
                    --- from the module ---
  harness           -- Context, Scenario, Run, fixture plumbing,
                       FindModuleRoot/BuildBinary, HARNESS_KEEP_* retention
  verify            -- GENERIC Then: CommandSucceeded/Failed, OutputContains/NotContains,
                       OutputFileExists, JSON assertions, file-state assertions

                    --- ours, under test/ ---
  actions/          -- When: command runners
    commands.go     -- RunClassify, RunAuto, RunBatchAuto, RunBatchAutoWithFixture
  expect/           -- DOMAIN Then: what our artifacts mean
    csv.go          -- classified/review CSV structure; semicolon+comment-aware reader
    feedback.go     -- classifications.jsonl / expenses_log.jsonl
    accuracy.go     -- soft accuracy + drift tracking to test/results/
    html.go         -- review.html embedded JSON
    workbook_structure.go -- generated-workbook dump comparison
  domain/           -- expense-specific helpers the module excludes
    workbook.go     -- RequireWorkbook, CopyWorkbookToWorkDir
    config.go       -- SetupBinaryConfig (config next to the binary)
    fixture.go      -- ExpenseFixtureConfig (model/threshold/... via the module's Raw)
    env.go          -- DataDir/WorkbookPath accessors over ctx.Env
  extern/           -- RequireOllama (the module core is LLM-free by design)
  fixtures/         -- per-scenario data; results/ gitignored
```

**Two assertion packages, both imported by scenarios:** `verify.*` (module, generic) and
`expect.*` (ours, domain). The module owns the `verify` name — never add a local package
called `verify`.

**Context** holds per-scenario state: `BinaryPath`, `WorkDir` (temp), `FixtureDir`,
`Env map[string]string`, `Artifacts map[string]string`, `Stdout`, `Stderr`, `ExitCode`.
It carries no domain fields — `DataDir`/`WorkbookPath` ride in `Env` via `domain`'s
accessors, and `runCommand` forwards `Env` to the command.

**Scenario** has three phases:
- `Given func(*Context)` — set up: copy fixtures, set binary path, prepare workbook
- `When func(*Context)` — action: run the CLI command
- `Then []func(*Context)` — assertions: composable, order-independent checks

**Run(t, Scenario)** executes the scenario **directly on `t`, not as a subtest** — so `t.Log`
output flushes in real time under `-v`, which matters when a step is waiting ~12s on Ollama.
It creates a fresh temp `WorkDir` per scenario and removes it on cleanup unless retention is
requested (see § Running).
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

### Generic — from the module (`verify.*`)

Full list in the module's godoc; the ones this suite leans on:

| Function | Signature | Assertion |
|----------|-----------|-----------|
| `CommandSucceeded` | `() func(*Context)` | Last command exited 0 |
| `CommandFailed` | `() func(*Context)` | Last command exited non-zero |
| `OutputFileExists` | `(artifactKey) func(*Context)` | Artifact file exists on disk |
| `OutputContains` | `(substr) func(*Context)` | stdout+stderr contains substr |
| `OutputNotContains` | `(substr) func(*Context)` | stdout+stderr does not contain substr |
| `OutputIsValidJSON` | `() func(*Context)` | stdout parses as JSON |
| `OutputJSONHasKey` | `(key) func(*Context)` | stdout JSON has top-level key |
| `OutputJSONHasValue` | `(key, expected) func(*Context)` | stdout JSON key equals expected (EqualValues) |

### Structural — ours (`test/expect/csv.go`)

| Function | Signature | Assertion |
|----------|-----------|-----------|
| `OutputFileHasRows` | `(artifactKey, n) func(*Context)` | CSV has exactly n rows |
| `OutputFileHasAtLeastRows` | `(artifactKey, n) func(*Context)` | CSV has >= n rows |
| `OutputFileHasColumns` | `(artifactKey, n) func(*Context)` | Every row has n columns |
| `AllClassificationScoresValid` | `(artifactKey) func(*Context)` | Confidence column values in [0.0, 1.0] |
| `NoExpenseInBothFiles` | `(artifact1, artifact2) func(*Context)` | No row in both files |
| `NoRolloverFileCreated` | `() func(*Context)` | rollover.csv absent (retired path) |

### Accuracy (`test/expect/accuracy.go`)

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
cd expense-reporter && ./run-acceptance.sh              # deterministic group (no Ollama needed)
cd expense-reporter && ./run-acceptance.sh -full        # whole suite (requires Ollama)
cd expense-reporter && ./run-acceptance.sh 'TestAdd_'   # filter by regex (composes with -full)
```
The default is the deterministic group: it passes `-short`, under which
`extern.RequireOllama` skips every Ollama-gated test, so it runs on any machine
with a 600s timeout and no Ollama pre-flight. `-full` runs everything: Ollama
pre-flight + a 3600s timeout (a full run measured 1989s with ~2.4x machine
variance from Ollama model-load contention). Both modes derive
`EXPENSE_WORKBOOK_PATH` from the repo-root workbook when unset — so
workbook-gated tests run rather than skip if that file is present.

**Directly:**
```bash
cd expense-reporter && go test -tags=acceptance -short -v ./test/...          # deterministic group
cd expense-reporter && go test -tags=acceptance -v -timeout 60m ./test/...    # whole suite
```
The whole suite takes ~15–35 min depending on Ollama contention, so the default
600s timeout is not enough without `-short`. If Ollama is not running, gated
tests skip via `extern.RequireOllama`; under `-short` they skip unconditionally.

**Keeping the work dir for inspection:**
```bash
./run-acceptance.sh 'TestFoo' -keep-artifacts       # flags, via harness.RegisterFlags()
HARNESS_KEEP_ON_FAILURE=1 go test -tags=acceptance ./test/...   # or env vars
```
The module reads `HARNESS_KEEP_ARTIFACTS` / `HARNESS_KEEP_ON_FAILURE`; `TestMain` calls
`harness.RegisterFlags()` to also bind the `-keep-artifacts` / `-keep-on-failure` flags the
script forwards. The preserved path is logged.

**Binary lifecycle:** `TestMain` in `setup_test.go` builds the binary once via
`harness.BuildBinary` into a temp dir and removes it after `m.Run()` (not via `defer` — 
`os.Exit` skips defers). All test files share it via the package-level `binaryPath`.

**Drift tracking:** `SoftAccuracy` writes JSON reports to `test/results/` (gitignored). Compare across runs to track classification accuracy changes over model/prompt updates.
<!-- /ref:acceptance-run -->
