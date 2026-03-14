  Phase 1 — Shared logic extraction

  internal/classifier/decision.go (new)
  - const DefaultHighConfidenceThreshold = 0.85
  - func IsAutoInsertable(result Result, threshold float64, excluded []string) bool — moved out of cmd/auto.go where it was private and hardcoded to 0.85; now takes threshold as a parameter so batch-auto can
  reuse it with its own --threshold flag

  pkg/utils/format.go (new)
  - func FormatBRValue(v float64) string — 35.5 → "35,50"
  - func BuildInsertString(item, date string, value float64, subcategory string) string — "item;date;35,50;subcategory"

  cmd/auto.go (modified) — calls classifier.IsAutoInsertable(...) and utils.BuildInsertString(...) instead of local private functions; local duplicates deleted.

  cmd/auto_test.go (modified) — TestFormatBRValue, TestBuildInsertString, TestIsAutoInsertable removed (migrated to their new packages with extended test cases).

  ---
  Phase 2 — Acceptance test harness

  All files under test/ carry //go:build acceptance and belong to package acceptance_test. A single TestMain in setup_test.go builds the binary once via go build before any test runs.

  test/harness/
  - scenario.go — Context struct (BinaryPath, WorkDir, Artifacts map, Stdout/Stderr/ExitCode) + Scenario struct (Name, Given, When, Then[]) + Run(t, Scenario) that calls Given→When→Then[] inside t.Run
  - fixture.go — FixtureConfig (JSON-deserialized from config.json), LoadFixtureConfig, CopyFixtureToWorkDir, DiscoverFixtures
  - comparator.go — ReadCSVFile (semicolon-delimited, comment-aware), CompareCSVExact, CompareCSVFuzzy, ConfidenceInRange
  - ollama.go — RequireOllama(t, url) — GETs /api/tags with 3s timeout, calls t.Skipf on failure

  test/actions/commands.go
  - RunClassify(args...), RunAuto(args...), RunBatchAuto(args...) — return func(*Context) closures that exec the binary and capture stdout/stderr/exitcode
  - RunBatchAutoWithFixture(fixtureDir) — reads config.json, builds full batch-auto args, registers classified.csv and review.csv in ctx.Artifacts

  test/verify/csv.go
  - ExitCodeZero, FileExists, RowCount, RowCountAtLeast, ColumnCount, AllConfidencesInRange, NoOverlap, StdoutContains, StdoutNotContains

  test/verify/accuracy.go
  - SoftAccuracy(artifactKey, expectedPath, floor, subcatCol, resultsDir) — compares actual vs expected subcategory column, fails only if accuracy < floor, writes AccuracyReport JSON to test/results/
  - AllInReview(artifactKey, autoInsertedCol) — asserts every row has auto_inserted == "false"

  test/verify/workbook.go — stub placeholder

  ---
  Phase 2 — Fixtures & test scenarios

  ┌───────────────────────────────────────────────────┬──────┬───────────────────────────────────────────────────────────────────────────────────┐
  │                      Fixture                      │ Rows │                                      Purpose                                      │
  ├───────────────────────────────────────────────────┼──────┼───────────────────────────────────────────────────────────────────────────────────┤
  │ fixtures/classify-basic/input.csv                 │ 5    │ Scenario table for TestClassify_Basic — read row-by-row, one harness.Run per item │
  ├───────────────────────────────────────────────────┼──────┼───────────────────────────────────────────────────────────────────────────────────┤
  │ fixtures/auto-basic/input.csv                     │ 3    │ Scenario table for TestAuto_Basic — same pattern                                  │
  ├───────────────────────────────────────────────────┼──────┼───────────────────────────────────────────────────────────────────────────────────┤
  │ fixtures/batch-auto-basic/input.csv               │ 10   │ File input for batch-auto dry-run tests                                           │
  ├───────────────────────────────────────────────────┼──────┼───────────────────────────────────────────────────────────────────────────────────┤
  │ fixtures/batch-auto-basic/expected-classified.csv │ 10   │ Soft accuracy reference for future SoftAccuracy assertions                        │
  ├───────────────────────────────────────────────────┼──────┼───────────────────────────────────────────────────────────────────────────────────┤
  │ fixtures/batch-auto-exclusions/input.csv          │ 3    │ Rows with "Diversos"-like items for exclusion tests                               │
  └───────────────────────────────────────────────────┴──────┴───────────────────────────────────────────────────────────────────────────────────┘

  End-to-end test scenarios:
  - TestClassify_Basic — iterates fixture rows, asserts exit 0 + % in output per item
  - TestAuto_Basic — same pattern for auto command
  - TestAuto_AmbiguousItemStaysInReview — vague item, asserts ✓ Inserted absent
  - TestBatchAuto_Basic — 10-row dry-run, asserts both CSVs exist, ≥1 row, 7 columns, confidences in [0,1]
  - TestBatchAuto_MixedConfidence — stricter: exactly 11 rows (1 header + 10 data)

  run-acceptance.sh — checks Ollama reachability, builds, runs go test -tags=acceptance -v -timeout 300s ./test/...

  ---
  Phase 3 — batch-auto command (5.4)

  cmd/batch_auto.go — Cobra command batch-auto <csv_file> with flags: --model, --data-dir, --threshold (0.85), --top (3), --dry-run, --output-dir

  Pipeline:
  1. Read 3-field semicolon CSV (item;DD/MM;value) via batch.NewCSVReader
  2. Load taxonomy + app config (exclusion list)
  3. Classify each row via classifier.Classify; on Ollama failure → confidence=0, mark for review, continue
  4. --dry-run skips workbook insert; otherwise backs up workbook, calls workflow.InsertExpense per high-confidence row, marks insert errors back to review
  5. Writes classified.csv (7 cols: item, date, value, subcategory, category, confidence, auto_inserted) and review.csv (same schema, only non-inserted rows)
  6. Prints summary: auto-inserted / for review / errors

  cmd/batch_auto_test.go — unit tests (no Ollama): TestParse3FieldLine, TestBatchAutoCommand_Flags, TestWriteClassifiedCSV, TestWriteReviewCSV_OnlyLowConfidence

  ---
  Housekeeping

  - test/results/ added to .gitignore
  - .claude/index.md updated with all new packages and the run-acceptance.sh entry
  - All 190+ pre-existing unit tests still pass; go vet clean; acceptance harness compiles with -tags=acceptance