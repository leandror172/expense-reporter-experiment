# Harness — External Usage Report (second consumer)

**Date:** 2026-06-27
**Subject:** the `test/harness/` BDD engine is being adopted by a **different repo** — the
`career-search` `roles` CLI — making it the harness's first external consumer.

This matters because `harness/` was *designed* to be domain-agnostic and extractable
(`test/.memories/KNOWLEDGE.md` "Harness Design — Domain-Agnostic Engine"). A second,
unrelated consumer is the real test of that claim — and it surfaces exactly where the
engine is still accidentally expense-shaped.

---

## The consumer

- **Repo:** `career-search` (private), branch `feat/tracker-cli`.
- **Tool:** `roles` — a Go CLI that reads a markdown tracker table + YAML sidecars and emits
  `--json` / human output (role status, fit, geo, resolved artifacts).
- **Profile that stresses the boundary:** **deterministic, no LLM.** Where this repo's suite
  leans on soft/accuracy assertions and Ollama gating, the tracker suite is fully hermetic,
  sub-second, all-hard-assertions. Nothing about LLM flakiness can mask a domain leak — so it
  is a clean probe of what is truly generic.
- **Consumer-side plan:** `career-search/.claude/plans/tracker-cli-acceptance-plan.md`.

## What is cleanly generic (reusable as-is)

| Symbol | Note |
|---|---|
| `harness/scenario.go` — `Context`, `Scenario`, `Run` | Given/When/Then engine; no domain knowledge |
| `harness/fixture.go` — `CopyFixtureToWorkDir`, `DiscoverFixtures`, `SeedFileFromFixture`, `copyFile` | generic file plumbing |
| `verify/json.go` — `OutputIsValidJSON`, `OutputJSONHasValue`, `OutputJSONHasKey` | directly reusable for any CLI's `--json` output |
| `verify/csv.go` — `CommandSucceeded`, `CommandFailed`, `OutputContains`, `OutputNotContains` | generic process/stdout assertions |
| `setup_test.go` — `TestMain` build-once + `findModuleRoot` | generalizes to `BuildBinary(moduleRoot, pkgPath)` |

## Expense-shaped leaks the consumer reveals (fix during extraction)

1. **`FixtureConfig` carries domain fields** — `Model`, `Threshold`, `AssertionType`,
   `AccuracyFloor`, `TopN` live in the *core* struct. The tracker needs none of them.
   → Genericize to `{Command string; ExtraArgs []string; Raw map[string]json.RawMessage}`;
   each consumer decodes its own fields from `Raw`.
2. **`CopyFixtureToWorkDir` is shallow** (`if entry.IsDir() { continue }`). This repo's fixtures
   are flat; the tracker's fixture is a *tree* (`data/`, `evaluations/`, `resumes/`).
   → Add a recursive `CopyTreeToWorkDir` to the core.
3. **`ollama.go` (`RequireOllama`) sits in `harness/` core.** A generic CLI-test lib must not
   assume an LLM backend. The tracker importing the core *without* it is the proof.
   → Move to an optional `extern` package (or keep it in this repo's domain layer).
4. **`comparator.go` is expense-flavored** — semicolon CSV + `ConfidenceInRange`. The tracker
   reads markdown tables + JSON + YAML, not semicolon CSV.
   → Make CSV an optional lib subpackage; `ConfidenceInRange` is pure expense (domain `verify`).
5. **`Context` has expense fields** — `WorkbookPath`, `DataDir`, `WorkbookDir`.
   → Replace with a generic `Env map[string]string` (or a small typed extension per consumer);
   keep the core `Context` to binary/workdir/fixture/stdout/stderr/exit/artifacts.

## Recommended extraction shape

Lift the **de-domained generic subset** into its own module (e.g.
`github.com/leandror/acceptance-harness`): the engine (`Context`/`Scenario`/`Run`), fixture
plumbing (+ `CopyTreeToWorkDir`), the generic `verify` base (incl. `json.go`), and
`BuildBinary`/`FindModuleRoot`. Both repos then depend on it; each keeps its **domain**
`actions/` + `verify/` (this repo retains CSV/JSONL/workbook/ollama; the tracker adds
markdown-table + YAML-sidecar verifiers).

**Sequencing note:** do the extraction *after* the tracker suite has been built against a
de-domained copy — letting a real second consumer shake out the boundary first avoids freezing
this repo's accidental shape into the "generic" lib.

## Tracked as

`.claude/tasks.md` → Deferred Tooling Improvements → **T2** (harness extraction; second consumer).
