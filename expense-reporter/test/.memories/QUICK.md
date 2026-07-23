# test/ — Quick Memory

*Working memory for the acceptance test harness. Injected into agents. Keep under 30 lines.*

## Status
**Engine is now the `github.com/leandror172/acceptance-harness` module (v1.0.0, T2
Session B).** `test/harness/` is gone: Context/Scenario/Run, fixture plumbing and
BuildBinary come from the module; everything expense-shaped stayed here. 15+ fixture
dirs cover all commands incl. apply, review, generate-workbook. Details → KNOWLEDGE.md.

## Structure
```
actions/    # When — CLI command runners (+ runCommand, forwards ctx.Env)
expect/     # Then — DOMAIN assertions (feedback/expense logs, CSV, HTML, workbook)
domain/     # expense-specific test helpers the module excludes: workbook gate +
            #   copy, SetupBinaryConfig, fixture-config decode, ctx.Env accessors
extern/     # RequireOllama (module core is LLM-free by design)
fixtures/   # Test data dirs; results/ gitignored
```

## Key Rules
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
  local Given `defaultYearConfiguredAs` writes `date_year`; `expect.OutputJSONHasDate`).
  **Exception (T-35): join-id tests MUST use short `DD/MM`** — a full date makes the raw and
  normalized strings identical, so it hides exactly the bug they guard. Safe only because
  they assert id EQUALITY, never a literal date, so no year is pinned. Do NOT "fix" the
  dates in `apply-join-id` or `TestAuto_JoinIDMatchesAcrossLogs` to a full year: that
  silently disables the test rather than breaking it.
- **Auto-append fixtures need a gate-PASSING item** (T-32) — `Posto Ipiranga`, not `Uber Centro`
- `requireDataDir(t)` guards gitignored `data/classification` reads (at REPO root)

## Deeper Memory → KNOWLEDGE.md
Module split · fixture formats · WS-B log retarget · type-routing-cycle ·
traps (dry-run/RequireWorkbook masking, explicit-year time-bomb, build-tag lessons)
