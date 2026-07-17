# test/ — Quick Memory

*Working memory for the acceptance test harness. Injected into agents. Keep under 30 lines.*

## Status
**Engine is now the `github.com/leandror172/acceptance-harness` module (v0.1.1, T2
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
  run `-tags=acceptance -timeout 30m` explicitly (q3 ≈12 s/classify)
- **Live Ollama required** (`extern.RequireOllama` skip gate); binary built once in TestMain
- **Taxonomy config MANDATORY (T-13)** — every classify-family Given needs a taxonomy
- **Non-dry-run fixtures need `DD/MM/YYYY` inputs**; check `extra_args` for `--dry-run`
- **Auto-append fixtures need a gate-PASSING item** (T-32) — `Posto Ipiranga`, not `Uber Centro`
- `requireDataDir(t)` guards gitignored `data/classification` reads (at REPO root)

## Deeper Memory → KNOWLEDGE.md
Module split · fixture formats · WS-B log retarget · type-routing-cycle ·
traps (dry-run/RequireWorkbook masking, explicit-year time-bomb, build-tag lessons)
