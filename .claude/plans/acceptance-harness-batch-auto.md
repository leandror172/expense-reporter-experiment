# Plan: Acceptance Test Harness + batch-auto Command

## Context

Layer 5 (classifier) has `classify` and `auto` commands working, integration-tested manually in session 3. Two gaps remain:

1. **No automated acceptance tests** — end-to-end validation of CLI commands against real Ollama relies on manual testing. This blocks confident iteration.
2. **No `batch-auto` command (5.4)** — the next feature milestone, which processes a full CSV through the classifier pipeline.

Building the test harness first lets us validate existing commands and then develop `batch-auto` with fixtures ready (TDD-style).

---

## Phase 1: Foundation — Extract shared logic

### Step 1.1: Extract `isAutoInsertable` to `internal/classifier/decision.go`

**Why:** Both `auto` and `batch-auto` need this logic. Currently private in `cmd/auto.go:107-117`.

**New file:** `internal/classifier/decision.go`
```go
func IsAutoInsertable(result Result, threshold float64, excluded []string) bool
const DefaultHighConfidenceThreshold = 0.85
```

**New file:** `internal/classifier/decision_test.go`
- Migrate test cases from `cmd/auto_test.go:73-101` (TestIsAutoInsertable)
- Add threshold parameter to test cases (original hardcodes 0.85)

**Modify:** `cmd/auto.go`
- Replace `isAutoInsertable(top, excluded)` → `classifier.IsAutoInsertable(top, highConfidenceThreshold, excluded)`
- Keep `highConfidenceThreshold` const in cmd (it's a CLI default, not domain logic)
- Remove the local `isAutoInsertable` function

**Modify:** `cmd/auto_test.go`
- Remove `TestIsAutoInsertable` (moved to classifier package)
- Tests for `formatBRValue`, `buildInsertString`, `confirmInsert` stay

**Verify:** `go test ./internal/classifier/... ./cmd/expense-reporter/cmd/...`

### Step 1.2: Extract `buildInsertString` and `formatBRValue`

**Why:** `batch-auto` also needs to build 4-field insert strings. These are pure functions with no CLI dependency.

**New file:** `pkg/utils/format.go` — alongside existing `currency.go` and `date.go`
```go
func FormatBRValue(v float64) string
func BuildInsertString(item, date string, value float64, subcategory string) string
```

**New file:** `pkg/utils/format_test.go` — migrate test cases from `cmd/auto_test.go:11-68`

**Modify:** `cmd/auto.go` — call `utils.BuildInsertString(...)` and `utils.FormatBRValue(...)` instead
**Modify:** `cmd/auto_test.go` — remove `TestFormatBRValue` and `TestBuildInsertString` (moved to utils)

---

## Phase 2: Acceptance Test Harness

### Directory structure
```
expense-reporter/
  test/
    harness/                  ← domain-agnostic engine (extractable)
      scenario.go             ← Context, Scenario, Run()
      fixture.go              ← FixtureConfig, CopyFixtureToWorkDir, CopyWorkbook, DiscoverFixtures
      comparator.go           ← ReadCSVFile, CompareCSVExact, CompareCSVFuzzy, ConfidenceInRange
      ollama.go               ← RequireOllama (t.Skip on failure)
    actions/                  ← domain: When functions
      commands.go             ← RunClassify, RunAuto, RunBatchAuto, RunBatchAutoWithFixture
    verify/                   ← domain: Then functions
      csv.go                  ← RowCount, ColumnCount, FileExists, NoOverlap, ExitCodeZero, AllConfidencesInRange
      accuracy.go             ← SoftAccuracy, AllInReview, AccuracyReport (drift tracking to JSON)
      workbook.go             ← stub for now (placeholder for future workbook assertions)
    fixtures/                 ← test data (one dir per functional slice)
      classify-basic/         ← classify command: 5+ rows, mixed categories
        input.csv             ← NOT used as file input (classify takes CLI args); serves as scenario table
        config.json           ← assertion config (command, model, threshold)
      auto-basic/             ← auto command: threshold + exclusion scenarios
        input.csv
        config.json
      batch-auto-basic/       ← batch-auto: representative 10-row batch
        input.csv             ← 3-field CSV (item;DD/MM;value)
        expected-classified.csv  ← for soft accuracy comparison
        config.json
    results/                  ← gitignored, drift tracking JSON output
    setup_test.go             ← TestMain (builds binary once), shared helpers
    classify_test.go          ← TestClassify_* scenarios
    auto_test.go              ← TestAuto_* scenarios
    batch_auto_test.go        ← TestBatchAuto_* scenarios
  run-acceptance.sh           ← pre-flight (Ollama check) + go test -tags=acceptance
```

All files under `test/` get `//go:build acceptance` build tag. Single test package `acceptance_test` — one `TestMain` builds the binary once for all tests.

### Step 2.1: Harness engine (4 files)

**`test/harness/scenario.go`** — core types and orchestrator:
- `Context` struct: `T`, `FixtureDir`, `WorkDir` (t.TempDir), `WorkbookDir`, `BinaryPath`, `Artifacts map[string]string`, `ExitCode`, `Stdout`, `Stderr`
- `Scenario` struct: `Name string`, `Given func(*Context)`, `When func(*Context)`, `Then []func(*Context)`
- `Run(t, Scenario)` — calls Given → When → Then[] sequentially inside `t.Run(s.Name, ...)`

**`test/harness/fixture.go`** — fixture discovery and setup:
- `FixtureConfig` struct: `Threshold`, `AssertionType` (hard/soft), `AccuracyFloor`, `Model`, `TopN`, `Command`, `ExtraArgs []string`
- `LoadFixtureConfig(dir)` — reads config.json with sensible defaults (0.85 threshold, top 3, hard assertions)
- `CopyFixtureToWorkDir(ctx, fixtureDir)` — shallow copy all files to ctx.WorkDir
- `CopyWorkbook(ctx, workbookSrc)` — copy baseline .xlsx to WorkDir, set ctx.WorkbookDir
- `DiscoverFixtures(baseDir)` — returns dirs containing config.json

**`test/harness/comparator.go`** — CSV comparison utilities:
- `ReadCSVFile(t, path)` — semicolon-delimited, comment-aware, returns `[][]string`
- `CompareCSVExact(t, actualPath, expectedPath)` — cell-by-cell comparison
- `CompareCSVFuzzy(t, actualPath, expectedRowCount, expectedHeaders)` — structural only
- `ConfidenceInRange(float64) bool` — checks [0.0, 1.0]

**`test/harness/ollama.go`** — environment check:
- `RequireOllama(t, url)` — GET `/api/tags` with 3s timeout, `t.Skipf` on failure

### Step 2.2: Domain actions (`test/actions/commands.go`)

Wrapper functions that return `func(*Context)` closures:

- `RunClassify(binaryPath, args...)` — executes `expense-reporter classify <args>`, captures stdout/stderr/exitcode
- `RunAuto(binaryPath, args...)` — same for `auto`
- `RunBatchAuto(binaryPath, args...)` — same for `batch-auto`
- `RunBatchAutoWithFixture(binaryPath)` — reads fixture config.json, builds args from it (input path, threshold, model, dry-run, output-dir), registers output artifacts

Internal `runCommand(ctx, binary, args...)` — uses `os/exec.Command`, captures output into `ctx.Stdout`/`ctx.Stderr`/`ctx.ExitCode`

### Step 2.3: Domain verify functions

**`test/verify/csv.go`:**
- `RowCount(artifactKey, n)` → hard assertion on row count
- `RowCountAtLeast(artifactKey, n)` → minimum rows
- `ColumnCount(artifactKey, n)` → columns per row
- `FileExists(artifactKey)` → artifact file on disk
- `NoOverlap(artifact1, artifact2)` → review.csv entries have auto_inserted=false
- `AllConfidencesInRange(artifactKey, confidenceCol)` → every confidence in [0,1]
- `ExitCodeZero()` → last command exit code

**`test/verify/accuracy.go`:**
- `SoftAccuracy(artifactKey, expectedPath, floor, subcatCol, resultsDir)` → compares actual vs expected categories, fails only if accuracy < floor, writes AccuracyReport JSON to results dir
- `AllInReview(artifactKey, subcatCol)` → all rows have auto_inserted=false
- `AccuracyReport` struct + `AccuracyTuple` for drift tracking

**`test/verify/workbook.go`:**
- Stub for now — placeholder interface for future workbook content assertions

### Step 2.4: Initial fixtures + test file

**`test/fixtures/classify-basic/config.json`:**
```json
{"command": "classify", "model": "my-classifier-q3", "assertion_type": "hard"}
```
The classify command takes CLI args (not CSV), so the fixture `input.csv` serves as a scenario table read by the test function, not passed to the binary.

**`test/fixtures/batch-auto-basic/input.csv`** — 10 representative rows, 3-field format:
```
Uber Centro;15/04;35,50
Compras Carrefour;03/01;150,00
Conta de luz;05/01;320,00
...
```

**`test/setup_test.go`:**
- `TestMain` — builds binary once via `go build -o <tmpdir>/expense-reporter ./cmd/expense-reporter`, sets package-level `binaryPath`
- `findModuleRoot()` helper

**`test/classify_test.go`:**
- `TestClassify_Basic` — RequireOllama, run classify with known item, assert exit 0 + non-empty stdout + contains "%"

**`test/batch_auto_test.go`:**
- `TestBatchAuto_Basic` — RequireOllama, load fixture, run batch-auto --dry-run, assert classified.csv/review.csv created, row counts, confidences in range

### Step 2.5: `run-acceptance.sh` wrapper script

```bash
#!/bin/bash
# Pre-flight: check Ollama, build binary, run acceptance tests
```
- `curl -s` Ollama `/api/tags` → exit 1 if not 200
- `go build` → exit 1 if fails
- `go test -tags=acceptance -v -timeout 300s ./test/...`
- Cleanup temp binary

### Step 2.6: Housekeeping
- Add `test/results/` to `.gitignore`
- Add new files to `.claude/index.md`
- Add `run-acceptance.sh` to `[ref:bash-wrappers]`

---

## Phase 3: batch-auto Command (5.4)

### Step 3.1: `cmd/expense-reporter/cmd/batch_auto.go`

**Cobra command:** `batch-auto <csv_file>`

**Flags:**
| Flag | Default | Description |
|------|---------|-------------|
| `--model` | `my-classifier-q3` | Ollama model |
| `--data-dir` | `data/classification` | Taxonomy data path |
| `--threshold` | `0.85` | Auto-insert confidence floor |
| `--top` | `3` | Classification candidates |
| `--dry-run` | `false` | Classify + write CSVs, skip workbook insert |
| `--output-dir` | (same as input) | Where to write classified.csv and review.csv |

**Input format:** 3-field, semicolon-delimited: `item;DD/MM;value` (no subcategory — classifier provides it)

**Processing pipeline:**
1. Read CSV via `batch.NewCSVReader(csvPath).Read()` → raw lines
2. Load taxonomy via `classifier.LoadTaxonomy(dataDir)`
3. Load config via `config.Load()` → exclusion list
4. For each line:
   - Split on `;` → 3 fields (item, date, value)
   - `utils.ParseCurrency(value)` → float64
   - `classifier.Classify(item, value, date, taxonomy, cfg)` → `[]Result`
   - `classifier.IsAutoInsertable(top, threshold, excluded)` → bool
   - **On Ollama failure:** mark confidence=0, auto_inserted=false, continue (option b)
   - Print progress: `[i/total] AUTO|REVIEW: item → subcategory (confidence%)`
5. If not `--dry-run` and highQueue non-empty:
   - `batch.NewBackupManager().CreateBackup(workbookPath)` before modification
   - Build 4-field insert strings via `utils.BuildInsertString()`
   - Track `highQueueSourceIdx []int` — maps highQueue[i] → results[i]
   - `workflow.InsertBatchExpenses(workbook, highQueue)` → `[]*BatchError`
   - On insert errors: set `results[idx].AutoInserted = false`
6. Write `classified.csv` — ALL rows: `item;date;value;subcategory;category;confidence;auto_inserted`
7. Write `review.csv` — rows where `auto_inserted=false`
8. Print summary: auto-inserted / for review / errors

**Internal types:**
```go
type classifiedRow struct {
    Item, Date, Value, Subcategory, Category string
    Confidence  float64
    AutoInserted bool
    Error        error
}
```

### Step 3.2: `cmd/expense-reporter/cmd/batch_auto_test.go`

Unit tests (no Ollama required):
- `TestWriteClassifiedCSV` — writes rows, verifies CSV content
- `TestWriteReviewCSV_OnlyLowConfidence` — auto_inserted=true rows excluded
- `TestBatchAutoCommand_Flags` — all flags registered on Cobra command
- `TestParse3FieldCSV` — edge cases (too few fields, empty lines)

### Step 3.3: Acceptance fixtures for batch-auto

**`test/fixtures/batch-auto-basic/`** — 10-row batch, dry-run, soft accuracy floor 0.5
**`test/fixtures/batch-auto-exclusions/`** — rows with Diversos, verify all go to review

Add scenarios to `test/acceptance_test.go`:
```go
func TestBatchAuto_MixedConfidence(t *testing.T) {
    harness.Run(t, harness.Scenario{
        Name:  "mixed batch — high confidence auto-inserted, low goes to review",
        Given: ..., When: ...,
        Then: []func(*harness.Context){
            verify.FileExists("classified.csv"),
            verify.FileExists("review.csv"),
            verify.RowCount("classified.csv", 10),
            verify.AllConfidencesInRange("classified.csv", 5),
            verify.ColumnCount("classified.csv", 7),
            verify.ExitCodeZero(),
        },
    })
}
```

---

## Phase 4: Polish

- Run full acceptance suite: `./run-acceptance.sh`
- Verify unit tests still pass: `go test ./...`
- Update `.claude/tasks.md` — mark 5.4 complete
- Session handoff documentation

---

## Local model usage during implementation

Per CLAUDE.md rules, try local models first for bounded Go tasks:
- **Good candidates for local codegen:** `decision.go` (simple function + const), `scenario.go` / `fixture.go` / `comparator.go` / `ollama.go` (straightforward structs + functions), `writeClassifiedCSV` / `writeReviewCSV` (CSV output), unit test stubs
- **Tier:** `my-go-qcoder` first, then `my-go-q25c14` on REJECTED
- **Escalate to Claude for:** the main `runBatchAuto` function (orchestration across classifier, workflow, config, batch — 3+ packages), acceptance test composition logic, any cross-file refactoring

---

## Verification plan

1. `cd expense-reporter && go test ./...` — all existing 190+ unit tests still pass
2. `cd expense-reporter && go vet ./...` — no lint issues
3. `cd expense-reporter && go build ./...` — compiles cleanly
4. `./run-acceptance.sh` — acceptance tests pass (requires running Ollama)
5. Manual smoke test: `expense-reporter batch-auto test-expenses.csv --dry-run` — verify classified.csv and review.csv output
6. Manual smoke test: `expense-reporter auto "Uber Centro" 35.50 15/04` — verify extraction didn't break existing behavior

---

## Files modified/created summary

| Action | File |
|--------|------|
| **Create** | `internal/classifier/decision.go` |
| **Create** | `internal/classifier/decision_test.go` |
| **Modify** | `cmd/expense-reporter/cmd/auto.go` — use `classifier.IsAutoInsertable` |
| **Modify** | `cmd/expense-reporter/cmd/auto_test.go` — remove migrated tests |
| **Create** | `pkg/utils/format.go` |
| **Create** | `pkg/utils/format_test.go` |
| **Create** | `cmd/expense-reporter/cmd/batch_auto.go` |
| **Create** | `cmd/expense-reporter/cmd/batch_auto_test.go` |
| **Create** | `test/harness/scenario.go` |
| **Create** | `test/harness/fixture.go` |
| **Create** | `test/harness/comparator.go` |
| **Create** | `test/harness/ollama.go` |
| **Create** | `test/actions/commands.go` |
| **Create** | `test/verify/csv.go` |
| **Create** | `test/verify/accuracy.go` |
| **Create** | `test/verify/workbook.go` (stub) |
| **Create** | `test/setup_test.go` |
| **Create** | `test/classify_test.go` |
| **Create** | `test/auto_test.go` |
| **Create** | `test/batch_auto_test.go` |
| **Create** | `test/fixtures/classify-basic/{input.csv, config.json}` |
| **Create** | `test/fixtures/batch-auto-basic/{input.csv, expected-classified.csv, config.json}` |
| **Create** | `test/fixtures/batch-auto-exclusions/{input.csv, config.json}` |
| **Create** | `run-acceptance.sh` |
| **Modify** | `.gitignore` — add `test/results/` |
| **Modify** | `.claude/index.md` — register new files |
