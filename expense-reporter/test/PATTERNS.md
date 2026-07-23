<!-- ref:acceptance-patterns -->
## Adding New Acceptance Scenarios

The BDD engine is the `github.com/leandror172/acceptance-harness` module, and so are the
generic assertions (`verify.*`). This file covers what is expense-specific; for the generic
half — the three-phase model, the zero-copy-read vs copy-to-WorkDir-write rule, fixture
plumbing — read the module's `docs/PATTERNS.md` and `docs/ADOPTION.md`.

| What's needed | Where it lives |
|---|---|
| New fixture dir + CSV | `test/fixtures/` |
| New test function | `test/*_test.go` |
| New DOMAIN assertion (what our artifacts mean) | `test/expect/` |
| New GENERIC assertion (any CLI would want it) | upstream: a PR to the module's `verify/` |
| New CLI invocation | `test/actions/` |
| External-service gate | `test/extern/` |
| Workbook / binary-config / fixture-config helper | `test/domain/` |
| Engine capability gap (Context, Scenario, Run, fixtures) | upstream: a PR to the module + version bump here |
| New feature to test | `cmd/` + `internal/` |

**Two assertion packages.** Scenarios import both: `verify.*` is the module's generic Then
(CommandSucceeded, CommandFailed, OutputContains, OutputNotContains, OutputFileExists,
OutputIsValidJSON, OutputJSONHasKey, OutputJSONHasValue); `expect.*` is ours. Never add a
local package named `verify` — the module owns that name.

## Then Composition Pattern

Then helpers are composable slices combined with `slices.Concat` at the call site:

```go
Then: slices.Concat(
    commandSucceeded(),
    classificationsMatchExpected(fixDir),
    expenseLogMatchesExpected(fixDir),
),
```

Each helper returns `[]func(*harness.Context)` scoped to one concern. This allows mixing and matching at the test level without monolithic helpers.

**Naming rule (Then):** helpers describe the concern they assert, not the overall test
scenario. Go one step further than "names the artifact": the name should functionally
describe the **specific expected result**, so a reader can infer the behavior from the name
alone without opening the helper or the fixture.

- Prefer the *outcome* over the *mechanism*: `installmentExpandedToNDatedLogLines(fixDir)`,
  not `expenseLogMatchesExpected(fixDir)`; `crossYearInstallmentNotDivertedToRollover()`,
  not a raw `expect.NoRolloverFileCreated()` (which also violates the "no raw `verify.*`/`expect.*` in
  `Then`" rule).
- When several scenarios share one verifier, wrap it in per-scenario named helpers that
  delegate to the shared verifier (e.g. `typedExpenseRecordedAsSingleLogLine` →
  `expenseLogMatchesExpected`). The wrapper's value **is** its name — the differing fixture
  encodes the differing result, and the name documents what that result means.
- Keep genuinely *invariant* concerns generic (`commandSucceeded()`,
  `classificationsMatchExpected()`); only the scenario-varying concern needs the
  outcome-describing name.

See `add_log_append_test.go` for the worked example.

**Call site reads as an English phrase (Given and Then alike):** the name plus its
argument should form one grammatical sentence — slot the argument into the grammar
instead of letting it dangle after a complete phrase. End the name with the word that
receives the argument (`As`, `To`, …): `defaultYearConfiguredAs(2024)`,
`flagYearResolvedBareDateTo("15/04/2024")`, `thousandsAmountReadAs(1234.56)` — not
`defaultYearConfigured(2024)` or `bareDateResolvedToFlagYear("15/04/2024")`, where the
argument sits outside the sentence.

## Given Naming Pattern

Given helpers use **Event Modeling** style: past-tense events that happened in the system,
not technical setup descriptions or state predicates.

**Good (event that happened):**
- `expenseAutoConfirmed(fixDir)`
- `expenseConfirmedThenCorrected(fixDir)`
- `paymentReceivedForOrder(orderID)`

**Bad (technical / state):**
- `setupClassificationsFile(fixDir)` — names the mechanism, not the event
- `previouslyConfirmedExpenseExists(fixDir)` — state predicate, not event
- `withFeedbackConfig` — internal plumbing, not a domain event (OK as a private
  helper called by Given functions; not OK as the Given itself)
- `classifierForJSON()` — the archetype: names a mechanism (*a classifier*) plus an
  output format (*for JSON*), neither of which is a domain fact. Renamed to
  `taxonomyAuthoredWithTrainingData()`, which states what is true in the domain.
- `batchReadyForLogAppend(fixDir)` — "ready for &lt;internal step&gt;" is a state predicate
  named after machinery. Renamed to `expenseBatchSubmittedForClassification(fixDir)`.

**Duplicate bodies are the tell.** Mechanism names describe *how the context was
assembled*, so two different assembly stories can hide one identical fact and nobody
notices: `classifierForJSON`/`classifierWithDataDir` and three separate
`*ReadyForLogAppend` helpers each turned out to be byte-identical. Event names collide
loudly when they mean the same thing. When a scenario genuinely needs its own name,
make it a one-line wrapper over the shared Given (e.g.
`knownExpenseBatchSubmittedForClassification`), never a second copy of the body.

**Exception:** absence of any event (an empty event stream) is a state, not an event.
Name it pragmatically — e.g. `noClassificationsRecorded()`.

**Why:** Given/When/Then reads like a story of what happened in the domain. Event-style
Given names align with how the system actually evolves over time (a sequence of recorded
events) and keep the test description domain-focused.

### Composing a Given from events

`Scenario.Given` holds ONE function while `Scenario.Then` holds a slice. That asymmetry is
why Given helpers duplicated for so long: a scenario needing three facts had no way to say
so except to write a new body containing all three, so every combination grew its own copy.

`given(...)` (in `givens_test.go`) restores the missing composition — it folds a list of
events into the single function the harness expects:

```go
Given: given(binaryBuilt(), fixtureAvailable(fixDir), trainingCorpusRecorded()),
```

Atomic events each carry ONE fact and compose in any combination: `binaryBuilt`,
`fixtureAvailable`, `trainingCorpusRecorded`, `taxonomyPublished`, `feedbackLogsConfigured`,
`inputBatchStaged`.

**This works only because `domain.SetupBinaryConfig` MERGES keys** across calls within a
scenario (it accumulates per `*harness.Context`). Before that, two events each writing part
of config.json would silently erase each other — the second `os.WriteFile` won. Any new
event that writes config must go through `SetupBinaryConfig`, never write config.json itself.

### Canonical Givens — reuse, don't re-copy

Three implementations cover nearly every scenario. A new Given should be a one-line
wrapper over one of them, named for its own scenario; write a new body only when the
setup genuinely differs.

| Canonical | What is true | Fixture copied to WorkDir? |
|---|---|---|
| `taxonomyAuthoredWithTrainingData(fixDir)` | taxonomy + real training corpus | no (read-only) |
| `taxonomyAuthoredWithoutTrainingData(fixDir)` | taxonomy only — for commands that never classify | no |
| `expenseBatchSubmittedForClassification(fixDir)` | taxonomy + corpus + a submitted input CSV | yes — batch-auto writes beside its input |

`jsonOutputFixture()` is the shared taxonomy-only fixture for scenarios that author no
fixture data of their own. Existing wrappers: `noExpensesLoggedYet`,
`knownExpenseNotYetLogged`, `expenseManuallyAdded`, `expenseClassifiedByModel`,
`tenMixedExpensesSubmittedForClassification`, `expensesWithExcludedCategoryMarkers`.

## Generate-Workbook Fixture Sub-Format (G3, 2026-06-11)

`generate-basic` does NOT follow the config.json+input.csv format. It is file-in/file-out:

| File | Role |
|------|------|
| `taxonomy.json` | workbook skeleton (sheets→categories→subcategories, incomeCategories→blocks); schema: spec §1.1 |
| `entries.jsonl` | `feedback.ExpenseEntry` lines; `date` is `DD/MM` (no year) |
| `entries-with-unmapped.jsonl` | variant with a subcategory absent from the taxonomy (warn+skip contract) |
| `expected-dump-skeleton/`, `expected-dump-data/` | committed `internal/inspect` dumps frozen from the scratch template-builder oracle |

Verifier: `expect.WorkbookStructureMatches(expectedDumpDir)` — compares a NORMALIZED SUBSET
(values, formulas, merges, dims, rowType/rowFill, bgColor/bold/borders); deliberately ignores
column widths, row heights, manifest source (excelize serialization noise). No Ollama —
these tests are deterministic and fast.

## JSONL Log Verification

For commands that write JSONL log files on insert, use file-specific verifiers (not generic string-keyed ones):

| Verifier | Artifact checked |
|----------|-----------------|
| `expect.ClassificationsMatch(expectedPath)` | `classifications.jsonl` |
| `expect.ClassificationsNotCreated()` | `classifications.jsonl` |
| `expect.ExpenseLogMatches(expectedPath)` | `expenses_log.jsonl` |
| `expect.ExpenseLogNotCreated()` | `expenses_log.jsonl` |

**Fixture files:** each fixture dir that exercises an insert command must have both:
- `expected-feedback.jsonl` — for `classifications.jsonl` verification
- `expected-expenses_log.jsonl` — for `expenses_log.jsonl` verification

**Field selection in expected files:** include only deterministic fields. Omit `id` and `timestamp` always (they are implementation details — the verifier already skips them). Omit `subcategory`/`category` for classifier-dependent tests (LLM output is non-deterministic across runs); include them for `add` command tests where the subcategory is passed explicitly.

## README Refs

| Key | Contains |
|-----|----------|
| `ref:acceptance-harness` | Context/Scenario/Run types; Given/When/Then execution flow; directory layout |
| `ref:acceptance-fixtures` | Fixture dir structure; config.json schema with all fields; CSV format rules |
| `ref:acceptance-verify` | All verifiers with signatures; column index table for batch-auto output |
| `ref:acceptance-run` | Build tag, run-acceptance.sh, go test invocation; binary lifecycle; drift tracking |
<!-- /ref:acceptance-patterns -->
