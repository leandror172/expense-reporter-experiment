# test/ — Knowledge (Semantic Memory)

*Acceptance test harness accumulated decisions. Read on demand by agents.*

## Harness Design — Domain-Agnostic Engine (2026-03)
The `harness/` package contains zero expense-reporter knowledge. It provides:
- `Context` — per-scenario state bag (binary path, work dir, artifacts, stdout/stderr, exit code)
- `Scenario` — three-phase struct: Given (setup), When (action), Then (assertions)
- `Run(t, Scenario)` — wraps in `t.Run` for named subtests
Domain knowledge lives in `actions/` (how to invoke CLI commands) and `verify/`
(what to assert about output).
**Rationale:** The engine is intentionally extractable — it could test any CLI tool
that produces file artifacts. Keeping domain out of the harness makes it reusable.
**Implication:** Adding a new command to test means adding a new action function and
possibly new verify helpers — the harness itself doesn't change.

## Fixture Format (2026-03)
Each fixture is a directory with:
- `config.json` — command, model, threshold, assertion_type, extra_args, accuracy_floor, top_n
- `input.csv` — semicolon-delimited, `#` comments supported
- `expected-classified.csv` — optional, for soft accuracy comparison
- `expected-feedback.jsonl` / `expected-expenses_log.jsonl` — for JSONL log verification
**Key distinction:** classify/auto fixtures use `input.csv` as a scenario table (each row
becomes a separate test invocation). batch-auto fixtures pass `input.csv` directly to the binary.
**Rationale:** Different commands have different input models. Scenario tables let
classify/auto tests run multiple items in one fixture without batch machinery.
**Implication:** The fixture format is a contract — changing it requires updating both
the test code and all existing fixture directories.

## Soft vs Hard Assertions (2026-03)
- **Hard** (`assertion_type: "hard"`) — exact match required, test fails on mismatch
- **Soft** (`assertion_type: "soft"`) — calculates accuracy percentage, fails only below
  `accuracy_floor`. Writes JSON reports to `test/results/` for drift tracking.
**Rationale:** LLM classification is non-deterministic. Hard assertions on classifier
output make tests flaky. Soft assertions with a floor catch regressions without
requiring exact reproducibility.
**Implication:** Mechanics tests (installments, rollover) use hard assertions — they
test deterministic logic. Classification tests use soft assertions.

## Threshold 0.0 Strategy (2026-03)
Fixtures testing structural mechanics (installments, rollover) set `threshold: 0.0`.
This means every classified row is auto-inserted regardless of confidence.
**Rationale:** These tests verify installment expansion, rollover detection, and CSV
output format — not classification accuracy. A non-zero threshold would make them
dependent on the classifier's confidence, adding false-negative risk.
**Implication:** Only use threshold 0.0 for tests where the classification result
doesn't matter. Classification quality tests should use realistic thresholds.

## Canonical Test Items (2026-03)
- **"Uber Centro"** — the most reliable test item. Consistently returns Uber/Taxi
  subcategory across models and runs. Used as the baseline in auto and feedback tests.
- **"Diarista Letícia"** — reliable for Diarista subcategory. Used in batch tests.
**Rationale:** Empirically discovered that some items are nearly deterministic across
Ollama model versions, while others are sensitive to model changes.
**Implication:** Use canonical items for structural tests. Use diverse items only in
soft-assertion tests where accuracy drift is tracked.

## Composable Then Pattern (2026-04)
Then helpers return `[]func(*harness.Context)`, not single functions. Combined with
`slices.Concat` at the test site. Each helper is scoped to one concern (e.g.,
"classified output has correct columns" vs "accuracy meets floor").
**Rationale:** Monolithic assertion functions hide what's being tested and make it
hard to compose different assertion sets for different fixtures.
**Implication:** When adding a new assertion concern, create a new helper function
that returns `[]func(*Context)`. Never add assertions to existing helpers unless
they're truly part of the same concern.

**Convention (2026-04):** `Then:` blocks must contain only named `then*` helpers —
never raw `verify.*` calls directly. `verify.*` calls belong inside helper bodies.
This keeps test intent readable at the scenario level and keeps assertion details
encapsulated. `then*` helpers live in the same `*_test.go` file as their tests;
`verify.*` functions live in `verify/` and return single `func(*harness.Context)`.
`commandSucceeded()` (feedback_test.go, same package) is the shared base — use it
via `slices.Concat` rather than calling `verify.CommandSucceeded()` directly.

**Refinement — name Then helpers by expected RESULT, not artifact (2026-06, PR #35):**
A `Then` helper name should let a reader infer the scenario's behavior without opening the
helper or the fixture. Prefer the *outcome* over the *mechanism*:
`installmentExpandedToNDatedLogLines(fixDir)` over `expenseLogMatchesExpected(fixDir)`;
`crossYearInstallmentNotDivertedToRollover()` over an inline `verify.NoRolloverFileCreated()`.
- **Split on variance:** keep genuinely *invariant* concerns generic (`commandSucceeded()`,
  `classificationsMatchExpected()`); only the *scenario-varying* concern needs the
  outcome-describing name. This reconciles "one concern per helper" with "name the result".
- **Per-scenario wrappers:** when several scenarios share one verifier, wrap it in named
  helpers that just `return expenseLogMatchesExpected(fixDir)`. The body is one line — the
  wrapper's value IS its name; the differing fixture encodes the differing result.
- **Worked example:** `add_log_append_test.go` (typedExpenseRecordedAsSingleLogLine,
  installmentExpandedToNDatedLogLines, crossYearInstallmentLoggedWithNextYearDate,
  crossYearInstallmentNotDivertedToRollover). Rule also documented in `PATTERNS.md`.
- **PENDING SWEEP (not yet done):** only `add_log_append_test.go` was renamed (the file PR #35
  flagged). Other files still call the mechanism-named `expenseLogMatchesExpected(` /
  `classificationsMatchExpected(` and may carry the same smell — candidates: `apply_test.go`,
  `correct_test.go`, `fewshot_test.go`, `generate_income_test.go`, `generate_test.go`,
  `json_output_test.go`, `typed_log_test.go`, `type_routing_cycle_test.go`. Not yet read or
  judged; rename only the scenario-varying call sites, leave the generic invariants alone.

## JSONL Verification Design (2026-03)
File-specific verifiers (not generic string-keyed):
- `verify.ClassificationsMatch(expectedPath)` — checks `classifications.jsonl`
- `verify.ExpenseLogMatches(expectedPath)` — checks `expenses_log.jsonl`
Expected files omit non-deterministic fields (`id`, `timestamp`). For classifier-dependent
tests, `subcategory`/`category` are also omitted from expected files.
**Rationale:** JSONL logs include auto-generated fields (hash IDs, timestamps) that
differ every run. Expected files contain only the deterministic contract.
**Implication:** When adding new fields to JSONL output, update the verifier's skip
list if the field is non-deterministic.
**WS-B slice 3 (2026-06–30):** the log-append path (`add`/`auto`/`batch-auto` via `appender.ExpandAndAppend`)
writes the **`date` field as `DD/MM/YYYY`** (was bare `DD/MM` under the old `logExpense`). Bare-`DD/MM` inputs
get `time.Now().Year()` (`ParseDateFlexible`) — a year time-bomb — so `expected-expenses_log.jsonl` fixtures use
**explicit-year inputs** and pin the deterministic subset `{item, date, value, type}` (subcategory/category left
out where classifier-dependent). Installment rows assert the expanded `item (i/N)` + incremented dates; cross-year
rows pin the real next-year date (e.g. `01/01/2027`) and pair with `NoRolloverFileCreated`.

## Generate-Workbook Acceptance Design (2026-06, session 29)
The `generate-basic` fixture is a NEW sub-format: `taxonomy.json` + `entries.jsonl`
(+ `entries-with-unmapped.jsonl`) + committed `expected-dump-skeleton/` and
`expected-dump-data/` (per-sheet `internal/inspect` JSON dumps). No config.json, no input.csv,
no Ollama (fully deterministic, <1s — safe for the suite-timeout budget).
**Oracle-frozen expectations:** dumps were frozen from the convergence-verified scratch
builder run on the same fixture BEFORE the generator existed, making G2 a converge-to-green
port (healthy RED = command-absent, not expectation-fake).
**Rationale:** placeholder expectations assert nothing and freezing a generator's own output
against itself is circular; an independent already-trusted producer is the only real oracle.
**Limit:** oracle and port can share a bug (hardcoded sheet order emitted invalid D0 refs in
the frozen dumps). When the contract changes, re-freeze and MANUALLY REVIEW the dump delta —
acceptance can't distinguish "both fixed" from "both broken".
**Normalized-subset comparison:** `verify.WorkbookStructureMatches` asserts exact equality on
values/formulas/merges/dims/rowType/rowFill/bgColor/bold/borders and ignores column widths,
row heights, and manifest source (excelize float/serialization noise). Full deep-equality
would turn cosmetic excelize quirks into red tests.
**Implication:** when extending the generator, run the acceptance tests first; if a deliberate
output change is intended, regenerate `expected-dump-*` with the fixed binary + workbook-inspect
and diff old-vs-new dumps before committing.

## Incremental Full-Cycle Test (type-routing-cycle, session 34)
`type_routing_cycle_test.go` proves the batch-auto→review→apply→generate-workbook chain that
validated the `sheets`→`types` fix. Design decisions worth reusing:
- **One CLI step per test, Given accumulates.** Each test's When is one command; prior steps
  become Given preparation. The last test (`_4_…RoutesByType`) seeds the cumulative typed log
  and runs ONLY generate-workbook — its Then is the payoff (ambiguous leaf Dentista ∈
  Variáveis+Extras routes to its chosen sheet, and is ABSENT from the other candidate).
- **Non-CLI steps fold into fixtures**, documented at the fold point: the true/false→1/0 CSV
  bridge lives in `review-input.csv`; the browser pick+export lives in `reviewed.json`. The
  harness can't run a transform or drive a browser, so don't model them as When steps.
- **Hermetic skeleton trick:** apply's new-row insert needs a workbook with the target slot.
  Instead of committing a binary `.xlsx`, the Given builds one with `generate-workbook` (no
  `--entries`). So generate-workbook is both the *subject* of the last test and a *setup tool*
  in apply's Given — empty skeleton vs `--entries`-filled.
- **Determinism split:** only the batch-auto step needs Ollama (gated, ~38s, asserts just the
  8-column type contract — not LLM values). Review/apply/generate are deterministic (<0.05s),
  so the fix's regression guard never depends on Ollama.
- **Routing assertion without a frozen oracle:** for a single-value routing check, scan the
  generated sheet's cells for the entry's unique value (inline excelize) rather than
  `WorkbookStructureMatches` — cheaper than freezing/maintaining dump fixtures.
