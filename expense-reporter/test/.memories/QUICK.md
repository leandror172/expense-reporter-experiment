# test/ — Quick Memory

*Working memory for the acceptance test harness. Injected into agents. Keep under 30 lines.*

## Status
Harness operational; 15+ fixture dirs cover all commands incl. apply, review,
generate-workbook. WS-B complete: batch-auto and apply acceptance assert
`expenses_log.jsonl` (`verify.ExpenseLogMatches`), not workbook rows. Details → KNOWLEDGE.md.

## Structure
```
harness/    # Domain-agnostic BDD engine
actions/    # When functions — CLI command runners
verify/     # Then functions — csv, accuracy, logs, workbook structure
fixtures/   # Test data dirs; results/ gitignored
```

## Key Rules
- **Build tag `acceptance`** — hidden from `go test ./...`; after config-contract changes
  run `-tags=acceptance -timeout 30m` explicitly (q3 ≈12 s/classify)
- **Live Ollama required** (`t.Skipf` gate); binary built once in `TestMain`
- **Taxonomy config MANDATORY (T-13)** — every classify-family Given needs
  `withFeedbackAndTaxonomyConfig(ctx, fixDir)` or `taxonomy_path` in config
- **Non-dry-run fixtures need `DD/MM/YYYY` inputs**; check `extra_args` for `--dry-run`
  before trusting a green test to cover the append path
- **Composable Then** — helpers return `[]func(*Context)`, joined with `slices.Concat`
- `requireDataDir(t)` guards gitignored `data/classification` reads (at REPO root)

## Deeper Memory → KNOWLEDGE.md
Harness design · fixture formats · WS-B log retarget · type-routing-cycle ·
traps (dry-run/RequireWorkbook masking, explicit-year time-bomb, build-tag lessons)
