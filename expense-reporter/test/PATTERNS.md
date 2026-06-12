<!-- ref:acceptance-patterns -->
## Adding New Acceptance Scenarios

| What's needed | Where it lives |
|---|---|---|
| New fixture dir + CSV | `test/fixtures/` |
| New test function | `test/*_test.go` |
| New verify function | `test/verify/` |
| Harness capability gap | `test/harness/` or `test/actions/` |
| New feature to test | `cmd/` + `internal/` |

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

**Naming rule (Then):** helpers describe the concern they assert, not the overall test scenario.

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

**Exception:** absence of any event (an empty event stream) is a state, not an event.
Name it pragmatically — e.g. `noClassificationsRecorded()`.

**Why:** Given/When/Then reads like a story of what happened in the domain. Event-style
Given names align with how the system actually evolves over time (a sequence of recorded
events) and keep the test description domain-focused.

## Generate-Workbook Fixture Sub-Format (G3, 2026-06-11)

`generate-basic` does NOT follow the config.json+input.csv format. It is file-in/file-out:

| File | Role |
|------|------|
| `taxonomy.json` | workbook skeleton (sheets→categories→subcategories, incomeCategories→blocks); schema: spec §1.1 |
| `entries.jsonl` | `feedback.ExpenseEntry` lines; `date` is `DD/MM` (no year) |
| `entries-with-unmapped.jsonl` | variant with a subcategory absent from the taxonomy (warn+skip contract) |
| `expected-dump-skeleton/`, `expected-dump-data/` | committed `internal/inspect` dumps frozen from the scratch template-builder oracle |

Verifier: `verify.WorkbookStructureMatches(expectedDumpDir)` — compares a NORMALIZED SUBSET
(values, formulas, merges, dims, rowType/rowFill, bgColor/bold/borders); deliberately ignores
column widths, row heights, manifest source (excelize serialization noise). No Ollama —
these tests are deterministic and fast.

## JSONL Log Verification

For commands that write JSONL log files on insert, use file-specific verifiers (not generic string-keyed ones):

| Verifier | Artifact checked |
|----------|-----------------|
| `verify.ClassificationsMatch(expectedPath)` | `classifications.jsonl` |
| `verify.ClassificationsNotCreated()` | `classifications.jsonl` |
| `verify.ExpenseLogMatches(expectedPath)` | `expenses_log.jsonl` |
| `verify.ExpenseLogNotCreated()` | `expenses_log.jsonl` |

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
