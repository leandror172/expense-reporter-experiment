<!-- ref:acceptance-patterns -->
## Adding New Acceptance Scenarios

| What's needed | Where it lives |
|---|---|---|
| New fixture dir + CSV | `test/fixtures/` |
| New test function | `test/*_test.go` |
| New verify function | `test/verify/` |
| Harness capability gap | `test/harness/` or `test/actions/` |
| New feature to test | `cmd/` + `internal/` |

## README Refs

| Key | Contains |
|-----|----------|
| `ref:acceptance-harness` | Context/Scenario/Run types; Given/When/Then execution flow; directory layout |
| `ref:acceptance-fixtures` | Fixture dir structure; config.json schema with all fields; CSV format rules |
| `ref:acceptance-verify` | All verifiers with signatures; column index table for batch-auto output |
| `ref:acceptance-run` | Build tag, run-acceptance.sh, go test invocation; binary lifecycle; drift tracking |
<!-- /ref:acceptance-patterns -->
