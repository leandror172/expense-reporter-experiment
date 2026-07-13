# acceptance-test vs. BDD Harness — Thorough Comparison

**Companion to:** `.claude/plans/harness-extraction-plan.md` (§1 lists it as the "idea quarry")
**Date:** 2026-07-12 (session 56)
**Subject:** `~/workspaces/acceptance-test` — a Go acceptance framework the user built at a
previous job for an async data pipeline (GCS → Cloud Function → Pub/Sub → pusher-app →
Aerospike → dsp-bidder REST). Compared against `expense-reporter/test/harness` to decide
what the extracted `acceptance-harness` module adopts, adapts, or discards.

---

## 1. How acceptance-test actually works

The test never talks to the pipeline directly — it drops a file at one end and
interrogates the other end.

Lifecycle of one test case (`main_test.go:119` `testCaseDir`):

1. **Setup** — a `sync.Once`-guarded `testFixture` (`main_test.go:45`) initializes viper
   config (`properties/config.<env>.yml`, selected by `-env` flag or `ENV` var), a GCS
   client, an HTTP client, and a zerolog logger wired into `go test -v` output via
   `zerolog.NewTestWriter(t)`. Shared once across all parallel tests.
2. **Load** — `loadTestCase` (`testrunner.go:168`) finds *the first file* in
   `data_types/<type>/test_data/` (exactly one input file supported) and parses
   `expected_result/` — a stream of JSON objects, each
   `{"line_number": N, "data": {...}, "ignore_order_on": [...],
   "expected_http_result_code": ...}`. The API lookup key is extracted from inside
   `data` via a second unmarshal pass (`KeyExtractor`, falls back `key` → `id`).
3. **Trigger** — upload the input file to `input/<data_type>/<filename>` in GCS
   (`testrunner.go:69`). The upload *is* the trigger; the pipeline is event-driven.
4. **"Observe"** — `WaitForProcessing()` (`testrunner.go:234`): a **flat `time.Sleep`**
   of a config-driven duration (20s in dev). No polling, no readiness probe.
5. **Assert** — `RunAssertions` (`assertion.go:20`): a worker pool of
   `Concurrency.AssertionWorkers` goroutines (default 20) pulls assertions off a channel;
   each does one HTTP GET (`<endpoint-for-type>/<key>`) and compares. A watchdog
   goroutine polls a failure counter every 100ms and **cancels the shared context once
   failures exceed `MaxFailedTestsToPrint`** — a circuit breaker that stops burning
   requests on a clearly-broken run.
6. **Report** — results (line number, key, expected/actual JSON, failure reason)
   aggregate into a `Summary`; `PrintDetailedSummary` logs failures up to a cap, then a
   "…and more" suppression. The Go test fails via `require.Zero(summary.Failed)`.
7. **Inverse phase** — re-upload the same file to `input/<type>/DELETE/<file>.DELETE`,
   wait again, assert against `expected_result_delete/` (usually "record is gone /
   default values"). Every data type runs create *and* delete.

Concurrency exists at **two levels**: `t.Parallel()` across the nine per-data-type test
functions, and the worker pool *within* each test's assertion phase (this is where
"tens of thousands of requests" get absorbed).

**Correction to institutional memory:** cobra never actually shipped. A comment in
`pkg/config/models.go` mentions it and the plan files show a CLI mode was intended, but
task 17 ("Refactor for Go Test Framework Compatibility") is the last recorded milestone —
the project ended up pure `go test`, same as our harness. A vestigial `RunAllTests`
self-discovery mode (`testrunner.go:95`) is left over from the pre-refactor design, no
longer wired to anything (and has a latent bug — `wg.Go` combined with a manual
`defer wg.Done()`, plus concurrent map writes to `finalSummary.Results`).

## 2. The fundamental difference in kind

The two projects answer **different questions**; almost every design divergence follows
from that:

| Dimension | acceptance-test | expenses harness |
|---|---|---|
| System under test | Remote async pipeline (5 hops, cloud) | Local CLI binary |
| Trigger | GCS file upload (side effect) | `exec.Command` (direct invocation) |
| Observation | Poll-free sleep, then HTTP GETs against a *different* service | Process exit code, stdout/stderr, files in a temp WorkDir |
| Test unit | 1 case = 1 input file → **thousands** of keyed assertions | 1 scenario = 1 command → a handful of composable assertions |
| Assertion volume strategy | Worker pool + circuit breaker | `[]func(*Context)` run sequentially |
| Nondeterminism source | Timing (did the pipeline finish?) | Content (LLM output varies) |
| Nondeterminism answer | Fixed wait | Soft assertions + accuracy floors + canonical items |
| Fixture shape | `test_data/` + `expected_result{,_delete}/` per data type | Directory with `config.json` + inputs + expected files |
| Readability model | None — assertions are data rows | **Given/When/Then discipline** — the harness's core asset |
| Environment | Multi-env (`-env dev/stg/prod`), real cloud, secrets | Hermetic temp dirs, local Ollama only |
| Extensibility | Hardcoded (`allowedDataTypes()` map + one test func per type + endpoint config, 3 places to touch) | Drop a fixture dir + write an action/verifier |

**Where meaning lives** is the deepest structural difference. In the harness, meaning
lives in *code*: a scenario is legible Go (`Given: tenMixedExpensesReadyForBatch`,
`Then: installmentExpandedToNDatedLogLines`). In acceptance-test, meaning lives in
*data*: the Go code is a fixed engine, and each of thousands of `.ljson` lines is one
assertion. That's why acceptance-test scales to 10,000 assertions but has no vocabulary
for expressing a *behavior*, while the harness expresses behaviors beautifully but has
no machinery for bulk. They are complementary, not competing.

Also worth naming: acceptance-test is an **observe-elsewhere** design — the assertion
target (bidder API) is a different system from the trigger target (GCS). The harness's
implicit assumption is that the thing you invoke is the thing you observe. LTG's
GIVEN→RUN→OBSERVE→ASSERT formalism is the generalization that covers both.

## 3. What we can use (adopt — mostly roadmap items in the extraction plan)

**a. The bulk-assertion executor pattern** (`assertion.go:20-105`). Worker pool with
config-driven width + context-cancel circuit breaker + progress logging every 100 items
+ "print first N failures, suppress the rest." The single most valuable *code* idea.
The harness has nothing for "one scenario produces 10,000 checkable facts" — if a future
consumer (a batch pipeline, a corpus tool like LTG) needs it, this is the template. Two
details worth keeping exactly: failures **become results, not errors** (an assertion
failure is data; only infrastructure failure aborts), and the breaker threshold reuses
the reporting cap — after N failures you learn nothing new from the rest.

**b. `ignore_order_on`** (`assertion.go:168-227`). Dot-path-targeted array sorting via
gjson/sjson before JSON comparison — surgical order-insensitivity, applied only where
declared, instead of a blunt "sort all arrays." Directly portable into the lib's
`verify/json.go` someday; the gjson path syntax even matches the dotted-path walker
career-search built.

**c. `expected_http_result_code`** (`models.go:39`). An assertion that checks *only* the
status and deliberately skips the body — the clean way to express "this key should 404
after delete." Generalizes to "assert on the outcome channel, ignore the payload" — the
harness analog is exit-code-only assertions (`CommandSucceeded/Failed`, already
present). The HTTP variant becomes relevant only if the lib ever grows an HTTP observe
mode.

**d. Line-number traceability** (`models.go:18`). Every assertion carries the source
line of the *input* row it came from, so a failure names the offending input line out of
thousands. Worth stealing whenever a bulk expected-file format arrives.

**e. The create/delete inverse pairing.** Every mutation test has a paired inverse
asserting the mutation can be undone/is gone (`expected_result` vs
`expected_result_delete`, the `.DELETE` path convention). Adopt as a *convention*
(PATTERNS.md material, not code): "a write scenario earns a paired inverse scenario."
The apply idempotency test is the same instinct, ad hoc.

**f. Interface-first clients.** `gcs.Uploader` and the HTTP getter are small interfaces
injected into the Runner — the reminder for any future `extern/` package: make the
external system an interface from day one.

## 4. What we'd change (right ideas, wrong implementations)

**a. Sleep → poll.** `WaitForProcessing`'s flat sleep is the project's confessed
weakness: always too long (wasted minutes when the pipeline is fast) and sometimes too
short (flaky when it's slow). The concept — "an observe phase that waits for the system
to reach a state" — is exactly right. The fix is the roadmap's
`WaitFor(ctx, timeout, interval, predicate)`: poll a readiness predicate with backoff,
fail on timeout with the last observed state. That one helper subsumes the sleep and
makes the whole trigger-async/observe-later class of consumers viable.

**b. Type-dispatch → strategy.** `runAssertion` picks its comparison by
`strings.Contains(dataType, "logistic_regression")` (`assertion.go:118`) — a
stringly-typed special case, with `getJsonAPIResponse`/`getValueAPIResponse` as
near-duplicate twins differing only in response type. The right shape is a declared
comparison strategy per assertion (or per fixture), the way our fixtures declare
`assertion_type: hard|soft`. Same lesson as the harness's own `FixtureConfig` leak:
don't hardcode the first two consumers' shapes into the engine.

**c. Hardcoded registry → discovery.** Adding a data type means touching
`allowedDataTypes()`, a new `TestDataType*` func, and the endpoint config map. Our
`DiscoverFixtures` (any dir with a `config.json`) is already the better answer; nothing
to import except the cautionary tale. Same for `findFirstFile` — "exactly one file per
directory" is a rigidity the harness never had.

**d. Epsilon comparison, generalized.** `assert.InEpsilon(..., 1e-9)` for float weights
is correct instinct (never `==` floats) but special-cased. Career-search's `EqualValues`
switch fixed int-vs-float64; a numeric-tolerance option on `JSONFieldEquals` would be
the general form — add only when a consumer needs it.

**e. The `t`-in-the-runner smell.** The Runner holds `*testing.T` and calls
`assert.JSONEqf(r.t, ...)` *inside worker goroutines* — every failed comparison logs
through testify immediately (noisy at volume; `t` isn't goroutine-safe for all
operations), *and* the result is separately accumulated into the Summary. Pick one
channel: compare pure (return a result), report at the end. The harness already does
this right — verifiers receive `*Context` and fail through `ctx.T` sequentially.

## 5. What we discard

- **The GCS/cloud trigger layer** (`pkg/gcs`, upload-path conventions) —
  pipeline-specific. The generic residue is "When can be any side effect, not only
  exec," which the harness's `When func(*Context)` already permits.
- **The org-internal dependencies** — `github.com/thinknear/dsp-go` (auth, `jsonhttp`,
  `secure.SensitiveString`), basic-auth config. Not portable, not needed.
- **Viper multi-env config layering** (`config.dev.yml` per environment). Real need
  there (dev/stg/prod clouds); meaningless for hermetic local CLI tests where the
  fixture *is* the environment. If a consumer ever needs env selection, `Context.Env`
  is the seam.
- **The vestigial `RunAllTests` CLI-mode remnant** and the never-built plan items
  (Pub/Sub emulator local mode, auto-generation of expected results from inputs). The
  auto-generation idea is the only tempting one, and the generate-workbook fixtures
  already embody its correct form: **oracle-freezing** — expected output must come from
  an independently trusted producer, never from the system under test, or the test is
  circular.
- **`pkg/reporter` as a package.** Structured pass/fail aggregation matters at 10k
  assertions; at harness scale, `t.Errorf` + the existing drift-tracking JSON cover it.
  If the bulk executor ever lands in the lib, a minimal Summary comes with it — as part
  of that feature, not as a standalone reporting layer.
- **The Planner/Executor `plan/rules/` meta-process files** — archaeology (the project
  was built Cursor-agent-driven with an execution-history ledger); the current
  session-tracking system is a strict superset of that idea.

## Bottom line

Architecturally the two projects share almost no code-shaped overlap — one is a
data-driven bulk validator for a remote async system, the other a code-driven behavior
spec for a local binary. What acceptance-test contributes to the extraction is a
**capability roadmap**: the four things the harness will need the day a consumer
outgrows "run a CLI, check a few files" — polling-based observation, bulk concurrent
assertion with a circuit breaker, declared-per-assertion comparison options
(`ignore_order_on`, status-only, epsilon), and input-line traceability. All four are in
the extraction plan's roadmap table, deliberately deferred until a real consumer demands
them — the same discipline that made the harness extractable in the first place.
