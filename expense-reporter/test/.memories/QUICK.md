# test/ — Quick Memory

*Working memory for the acceptance test harness. Injected into agents. Keep under 30 lines.*

## Status
**Engine is the `github.com/leandror172/acceptance-harness` module (v1.1.0 since session
64).** `test/harness/` is gone: Context/Scenario/Run, fixture plumbing and BuildBinary come
from the module; everything expense-shaped stayed here. Since v1.1 the **engine owns
scenario plumbing** — `Scenario.Fixture` sets `ctx.FixtureDir`, `harness.UseBinary` (TestMain)
sets `ctx.BinaryPath`, so no Given wires either or takes a fixture path. 38 fixture dirs
cover all commands incl. apply, review, generate-workbook. Details → KNOWLEDGE.md.

## Structure
```
givens_test.go  # Given vocabulary: 4 atomic events + 3 canonical Givens (start here)
actions/    # When — CLI command runners (+ runCommand, forwards ctx.Env)
expect/     # Then — DOMAIN assertions (feedback/expense logs, CSV, HTML, workbook)
domain/     # expense-specific test helpers the module excludes: workbook gate +
            #   copy, SetupBinaryConfig, fixture-config decode, ctx.Env accessors
extern/     # RequireOllama (module core is LLM-free by design)
fixtures/   # Test data dirs; results/ gitignored
```

## Key Rules
- **READ THE DSL GUIDE BEFORE WRITING OR CHANGING ANY TEST** — `[ref:acceptance-dsl]`
  (`.claude/tools/ref-lookup.sh acceptance-dsl`). It lists what to read in what order
  (`test/PATTERNS.md` → `givens_test.go` → `actions/commands.go` header → the module's docs
  and `harness/scenario.go`) and the invariants a new scenario must not break. Skipping it is
  how the suite reached 27 Given bodies for 3 real preconditions.
- **Two assertion packages, both imported by scenarios** — `verify.*` = the MODULE's
  generic Then (CommandSucceeded, OutputContains, OutputJSONHasKey…); `expect.*` = ours.
  Never re-add a `verify`-named local package.
- **`then*` helper bodies call `verify.*`/`expect.*`** — never raw in a `Then:` block
- **Build tag `acceptance`** — hidden from `go test ./...`; after config-contract changes
  run explicitly: `./run-acceptance.sh` = deterministic group (`-short`, ~8s, no Ollama);
  `-full` = whole suite (~2 min at the think-off default, 3600s ceiling) — T-39 split
- **Live Ollama required only for `-full`** (`extern.RequireOllama` skips when down AND
  unconditionally under `-short`); binary built once in TestMain
- **Taxonomy config MANDATORY (T-13)** — every classify-family Given needs a taxonomy
- **Non-dry-run fixtures need `DD/MM/YYYY` inputs**; check `extra_args` for `--dry-run`.
  **T-41 sharpens this for add/correct:** a bare date strictly in the future now resolves
  to LAST year (rung 4, grace 0) — a late-December bare date in a fixture flips years
  depending on the run date. Year-precedence tests: `add_year_test.go` (deterministic;
  local Given `defaultYearConfiguredAs(year)` writes `date_year`; `expect.OutputJSONHasDate`).
  **Exception (T-35): join-id tests MUST use short `DD/MM`** — a full date makes the raw and
  normalized strings identical, so it hides exactly the bug they guard. Safe only because
  they assert id EQUALITY, never a literal date, so no year is pinned. Do NOT "fix" the
  dates in `apply-join-id` or `TestAuto_JoinIDMatchesAcrossLogs` to a full year: that
  silently disables the test rather than breaking it.
- **Auto-append fixtures need a gate-PASSING item** (T-32) — `Posto Ipiranga`, not `Uber Centro`
- `requireDataDir(t)` guards gitignored `data/classification` reads (at REPO root)
- **An all-malformed fixture is deterministic (s71)** — the parse precedes classification, so
  `batch-auto-failed-rows` calls no model and runs in the `-short` group, same reason
  `batch-auto-resume-all-seeded` is Ollama-free. It therefore CANNOT prove a parseable row
  stays out of `failed.csv`; that claim is pinned by a pure-function unit test instead. Do
  not "improve" it by adding a good row — see the fixture README.

## Deeper Memory → KNOWLEDGE.md
Module split · fixture formats · WS-B log retarget · type-routing-cycle ·
traps (dry-run/RequireWorkbook masking, explicit-year time-bomb, build-tag lessons)
