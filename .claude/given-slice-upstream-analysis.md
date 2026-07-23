# Should `harness.Scenario.Given` become a slice? — repo-wide analysis

**Date:** 2026-07-23 (session 64) · **Repo surveyed:** `expense-reporter/test/` (62 scenarios)
**Question:** the local `given(...)` compose helper landed this session. Should the change go
upstream to `github.com/leandror172/acceptance-harness` as `Given []func(*Context)`?

**Recommendation: NO to the slice. YES to upstreaming the compose helper** as a non-breaking
`harness.Given(...)` utility (v1.1.0). Evidence below.

---

## 1. The asymmetry that started this

```go
type Scenario struct {
    Given func(*Context)      // one
    When  func(*Context)      // one
    Then  []func(*Context)    // many
}
```

The diagnosis was correct and is worth recording: **the single Given slot caused the
duplication this session removed.** A scenario needing "taxonomy authored AND corpus
recorded AND batch submitted" had no way to express it except a new function containing all
three, so every distinct combination grew its own body — 27 helper bodies for 3 real
preconditions. Composition was the missing primitive.

But the primitive was missing *at the helper level*, not the scenario level. That distinction
decides the upstream question.

## 2. Survey data

| Measure | Value |
|---|---|
| Scenarios | 62 |
| Distinct Given expressions at call sites | 43 |
| Scenarios passing **more than one** Given at the call site | **0** |
| Given helpers defined | 48 |
| …that are wrappers/compositions with no own body | 20 |
| …that actually seed state (write/copy/seed files) | 15 |
| Helper names encoding ≥2 facts (`With`, `Then`, `Before`, `Already`, …) | 20 |

The critical row is the third: **every one of the 62 scenarios names exactly one Given.**
Not one would be written as a list today, including after this session's refactor.

## 3. Why the call site stays single-valued

The suite's convention (PATTERNS.md, "Given Naming Pattern") is one domain-named event per
scenario, because *the name is the documentation*: `Given: expenseConfirmedThenCorrected(fixDir)`
tells a reader what world the test runs in. Decomposing that into

```go
Given: given(binaryBuilt(), fixtureAvailable(f), trainingCorpusRecorded(),
             taxonomyPublished(f), feedbackLogsConfigured(), priorEntriesSeeded(f)),
```

replaces a domain sentence with a setup checklist. That is plumbing leaking upward — the
same failure the naming sweep just spent a session removing.

The 20 multi-fact names are **not** decomposition candidates for this reason.
`taxonomyAuthoredWithTrainingData` is better as one name than as two composed atoms at every
call site: "taxonomy authored with training data" is a single domain concept, not a
coincidence of two.

## 4. The counter-example that settles it

`batch_auto_resume_test.go` already solved composition without any harness change:

```go
func expenseLogSeededWith(fixDir string, rows []seedRow) func(*harness.Context)   // parameterized
func allInputRowsAlreadyLogged(fixDir string) ...  { return expenseLogSeededWith(fixDir, []seedRow{...}) }
func oneOfTwoDuplicatesAlreadyLogged(fixDir) ...   { return expenseLogSeededWith(fixDir, []seedRow{...}) }
func oneRowAlreadyLoggedOtherIsNew(fixDir) ...     { return expenseLogSeededWith(fixDir, []seedRow{...}) }
func expenseAlreadyLoggedThenReappended(fixDir)... { return expenseLogSeededWith(fixDir, []seedRow{...}) }
```

This is the best-designed Given family in the suite: one implementation, four scenario names,
variation carried by **data** rather than by copied bodies. It needed neither a slice nor the
`given(...)` helper — parameterization was enough. Where a Given varies by *content*,
parameterize; where it varies by *which facts hold*, compose. Neither case wants the
scenario struct to change.

## 5. What the local helper already delivers

`given(...)` (in `test/givens_test.go`) folds N events into the one function the harness
expects. It is used **inside** helper definitions, which is where composition belongs:

```go
func taxonomyAuthoredWithTrainingData(fixDir string) func(*harness.Context) {
    return given(binaryBuilt(), fixtureAvailable(fixDir), trainingCorpusRecorded(),
                 taxonomyPublished(fixDir), feedbackLogsConfigured())
}
```

Result: 3 canonical Givens built from 6 atomic events; every other helper is a one-line
wrapper. Full suite 62/62 green. **The slice would add nothing this does not already do.**

### The real precondition was not the slice

Composition was blocked by `domain.SetupBinaryConfig` doing a whole-file `os.WriteFile` —
two events each configuring part of `config.json` silently erased each other. Fixing that to
merge per-scenario was the actual enabling change. Any consumer attempting composition hits
this same class of bug in its own domain layer, which is a **consumer** concern, not an
engine one — further evidence the engine is not where the fix belongs.

## 6. Cost of the breaking change, for completeness

- `Given []func(*Context)` is source-breaking → **v2.0.0** of a published module.
- Two known consumers migrate: this repo (62 scenarios) and career-search.
- Every existing `Given: helper(fix)` becomes `Given: []func(*harness.Context){helper(fix)}`
  or needs a `slices.Concat` wrapper — noise on 62 call sites to enable a pattern zero of
  them want.

## 7. Recommendation

1. **Do not** change `Scenario.Given` to a slice.
2. **Do** upstream the compose helper itself as `harness.Given(events ...func(*Context)) func(*Context)`
   — additive, non-breaking, **v1.1.0**, no consumer migration. It makes the composition
   primitive discoverable to every consumer instead of each reinventing it, and it documents
   the intended shape of a Given (a sequence of events) without forcing it into the struct.
3. **Do** carry the merge-config lesson into the harness docs (`docs/PATTERNS.md` upstream):
   *if your Givens compose, your domain layer's config writer must merge, not overwrite.*
   That is the trap, and it is invisible until a scenario mysteriously loses a config key.

## 8. If the slice is wanted anyway

The one honest argument for it: it makes the *wrong* thing (a single monolithic Given) harder
to write, which is what let this suite drift to 27 bodies. If that enforcement is judged worth
a major version, the migration is mechanical (wrap every call site in `slices.Concat`) and the
`given(...)` helper becomes redundant. But the enforcement is weak — nothing stops a consumer
from putting one fat function in the slice, which is exactly what a drifting suite would do.

---

## Appendix — remaining local cleanups (not blocking, previously proposed)

- **generate family** (6 helpers, `generate_test.go` + `generate_income_test.go` +
  `type_routing_cycle_test.go`): identical bodies; four share the fixture `generate-basic`
  and differ only in the entries file the **When** selects. Proposal: keep the names, make
  five wrappers over `taxonomyAuthored`.
- **correct pair** (`expenseAutoConfirmed`, `expenseConfirmedThenCorrected`): identical
  bodies, different fixtures. Proposal: new canonical `priorClassificationsRecorded(fixDir)`,
  both become wrappers — pairs with the existing `noClassificationsRecorded()`.
- `.claude/tasks.md:190` still cites the renamed `classifierForJSON()` (handoff-only file).
