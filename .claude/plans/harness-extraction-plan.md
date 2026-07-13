# Acceptance-Harness Extraction Plan (T2)

**Date:** 2026-07-12 (session 56 planning)
**Goal:** extract the `test/harness` BDD engine into a standalone public Go module,
repoint career-search (unblocks its TC-EXT), give expense-reporter a migration path,
and document an adoption path for latent-topic-graph.
**Decisions locked with user this session:** public GitHub repo · lean core v1 (no LLM
extension packages) · LTG = design influence + adoption-path doc only · split
sequencing (repo + career-search first; expenses migration is its own session).

---

## 1. Inputs (read these before executing)

| Source | Role | Key artifacts |
|---|---|---|
| **expense-reporter** (this repo) | Origin; second migration target | `expense-reporter/test/harness/{scenario,fixture,comparator,ollama}.go`, `test/setup_test.go`, `test/harness-external-usage.md` (5 known leaks), `test/PATTERNS.md`, `test/.memories/KNOWLEDGE.md` (methodology) |
| **career-search** | First external consumer; **the seed** | `dashboard/test/harness/{scenario,fixture,buildbin}.go` (already de-domained), `dashboard/test/verify/{cli,json}.go` (generic base), `dashboard/test/EXTRACTION-NOTES.md` (**the de-facto spec**), `.claude/plans/tracker-cli-acceptance-plan.md` §2, tasks.md TC-EXT (BLOCKED on this module) |
| **acceptance-test** (old job project) | Idea quarry only — no code reuse | `pkg/testrunner/assertion.go` (worker pool + circuit breaker), `pkg/models/models.go` (`.ljson` assertion schema, `ignore_order_on`, `expected_http_result_code`), `main_test.go` (create→wait→assert→delete flow). **Full adopt/adapt/discard analysis: companion doc `harness-extraction-acceptance-test-comparison.md`** |
| **latent-topic-graph** | Future consumer; formalism donor | `docs/plans/ltg-l06-incremental-refresh.md` §"Acceptance tests" (GIVEN→RUN→OBSERVE→ASSERT + `MANUAL`), `probes/*-acceptance.md` (hand-run records a harness would automate) |
| **llm repo** | Surveyed; NOT a consumer | Scenario specs + pytest only; no harness desire expressed |

## 2. The one design thesis

career-search's EXTRACTION-NOTES: *"The methodology is the treasure; the code is mostly
generic with three expense-shaped leaks."* The repo ships **methodology docs as
first-class content** (the Given/Then discipline, result-named Then helpers, no raw
`verify.*` in `Then:` blocks, `slices.Concat` composition, zero-copy-read vs
copy-to-WorkDir-write) alongside ~500 lines of code. If only one thing ships well, it's
the docs.

## 3. New repo design (v1)

- **Location:** `~/workspaces/acceptance-harness`
- **Module:** `github.com/leandror172/acceptance-harness` — public, MIT license
- **Seed:** career-search's de-domained copy (NOT the expenses original — the second
  consumer already shook out the boundary; that was the whole point of T2's sequencing
  note). Backport career-search's generic verify additions too.

### Package layout

```
acceptance-harness/
  harness/          # engine: Context, Scenario, Run, fixtures, buildbin
    scenario.go     #   Context{T, FixtureDir, WorkDir, BinaryPath, Env, Artifacts,
                    #           Stdout, Stderr, ExitCode} — nothing domain
    fixture.go      #   FixtureConfig{Command, ExtraArgs, Raw map[string]json.RawMessage},
                    #   CopyTreeToWorkDir (DEFAULT, recursive), CopyFixtureToWorkDir
                    #   (flat, the special case), DiscoverFixtures, SeedFileFromFixture
    buildbin.go     #   BuildBinary(moduleRoot, pkgPath, binName), FindModuleRoot
  verify/           # generic Then base — separate package, no domain mixing
    cli.go          #   CommandSucceeded/Failed, OutputContains/NotContains, OutputFileExists
    json.go         #   nested by default: JSONFieldEquals (dotted path + array index),
                    #   JSONArrayLen, JSONArrayEvery, OutputIsValidJSON; assert.EqualValues
    file.go         #   FileUnchanged, FileAbsent, WorkFileExists, FileContains
  docs/
    PATTERNS.md     # the methodology, de-domained (examples from a neutral toy CLI)
    ADOPTION.md     # consumer how-to: TestMain + BuildBinary, actions/ + verify/ split,
                    #   zero-copy-read vs copy-write pattern, build-tag ownership
    LTG-PATH.md     # §7 below
  README.md         # what it is, the three-phase model, quick start
  LICENSE           # MIT
```

### v1 design rules (from EXTRACTION-NOTES, verbatim intent)

1. **Core is LLM-free, network-free, git-free.** No `ollama.go`, no soft-assertion /
   accuracy-floor / drift machinery, no CSV comparator. Those stay in
   expense-reporter's domain layer (lean-core decision). Grep gate: `grep -ri ollama`
   over the repo returns nothing — including **log strings** (the "waiting for Ollama"
   leak class).
2. **Recursive tree copy is the default**; flat copy is the special case.
3. **Nested JSON from day one**; numeric comparison via `EqualValues`, never `!=`.
4. **`config.json` is optional** — "every fixture has a config.json" was an expense
   assumption. `DiscoverFixtures` keys on it, but actions may pass flags directly.
5. **No `//go:build acceptance` in the library.** The build tag is consumer policy
   (each consumer tags its own `*_test.go`); a library behind a tag can't be vetted or
   tested normally. (Both current copies carry the tag — strip it during seeding.)
6. **Flag registration is a lib gotcha.** `keepArtifacts`/`keepOnFailure` are
   package-level `flag.Bool` today — fine in a test package, rude in a library
   (init-time registration can collide with consumer flags). v1: read env vars
   (`HARNESS_KEEP_ARTIFACTS`, `HARNESS_KEEP_ON_FAILURE`) with an optional
   `RegisterFlags()` for consumers who want CLI flags. Keep the `os.MkdirTemp` +
   manual-cleanup design (it exists to honor keep-on-failure; `t.TempDir()` can't).
7. **OBSERVE influence (LTG):** `Artifacts map[string]string` is already
   "named captures" — document it as the OBSERVE register (When populates named
   artifacts; Then predicates consume them by name). No API change needed in v1; the
   framing goes in PATTERNS.md. Add `Context.Env` semantics doc (env injection seam).
8. **The lib gets its own unit tests** (fixture copy, config parse, JSON walker,
   file verifiers — all deterministic) so consumers aren't its only safety net.

## 4. Session A — create the repo + repoint career-search

1. **Scaffold:** new repo at `~/workspaces/acceptance-harness`; `git init`; go.mod
   `github.com/leandror172/acceptance-harness` (Go 1.25); MIT LICENSE.
2. **Seed code** from `career-search/dashboard/test/harness/` + `verify/{cli,json}.go`
   + the file-state verifiers from its write suite. Strip build tags; apply rules
   §3.5–3.6; split `verify` into `cli/json/file`. Dependency: testify only.
3. **Port the methodology docs**: rewrite `expense-reporter/test/PATTERNS.md` +
   the Given/Then conventions from `test/.memories/KNOWLEDGE.md` into
   `docs/PATTERNS.md` with neutral examples; write `docs/ADOPTION.md` (TestMain
   pattern, actions/verify split, zero-copy-read vs copy-write, build-tag ownership).
4. **Unit tests** for the lib (§3.8); `go vet` + `go test ./...` green.
5. **Publish:** create public GitHub repo `leandror172/acceptance-harness`, push,
   tag `v0.1.0`.
6. **Repoint career-search** (TC-EXT): delete `dashboard/test/harness/`, add the
   module dependency, update imports in `actions/`, `verify/tracker.go`, test files;
   fold its local `verify/{cli,json}.go` usage onto the lib package (its domain
   `tracker.go` stays). Gate: its full acceptance suite (26 tests, deterministic,
   sub-second) green. Close TC-EXT in career-search's tasks.md; note the module in
   its EXTRACTION-NOTES header.
7. **Record here:** session-handoff updates T2 with "repo created, career-search
   repointed; expenses migration next".

**Caveats for session A**
- career-search adapted at an old expense branch (`feat/t13-classifier-full-path`);
  diff its `scenario.go`/`fixture.go` against current expenses master before seeding —
  expenses may have grown helpers since (e.g. `SeedFileFromFixture` changes).
- Public repo → double-check no personal data rides along in examples/fixtures
  (use synthetic toy-CLI examples only; career-search fixtures stay in career-search).
- Don't over-absorb: `SetupBinaryConfig` (config-next-to-binary) is expense-specific —
  it encodes expense's `os.Executable()` config resolution; leave it behind.

## 5. Session B — migrate expense-reporter

1. Add the module dep; delete `test/harness/` **except** `ollama.go` +
   `comparator.go`, which move into the domain layer (e.g. `test/extern/ollama.go`,
   `test/verify/csv_compare.go`) since lean-core v1 excludes them.
2. Update imports across `test/*.go`, `test/actions/`, `test/verify/` — the mechanical
   bulk. `Context` field fallout: `WorkbookPath`/`DataDir`/`WorkbookDir` disappear from
   core `Context` → carry them via `Env` (keys like `DATA_DIR`) or a small domain
   wrapper struct; decide at migration time by count of touch points.
3. `FixtureConfig` fallout: expenses' `Model/Threshold/AssertionType/AccuracyFloor/TopN`
   move to `Raw` decoding via a domain `LoadExpenseFixtureConfig(dir)` helper that
   wraps the lib loader.
4. Keep expenses' own build tag + `run-acceptance.sh` unchanged.
5. **Gate:** full `-tags=acceptance -timeout 30m` suite green (needs live Ollama,
   q3 ≈12 s/classify) AND `-tags=replay` still compiles (replay files import nothing
   from test/, but grep anyway — build-tag-hidden breakage is this repo's known trap
   class [[feedback_rename_json_tag_acceptance]]).
6. Update `test/README.md`, `test/PATTERNS.md` (now points at the lib's docs for the
   generic half), `test/.memories/*`, `.claude/index.md`, `harness-external-usage.md`
   (mark superseded/executed), T2 in tasks.md at handoff.

## 6. Roadmap (post-v1, in rough priority order)

| Item | Source | Notes |
|---|---|---|
| **`extern/llm` extension package** | expenses | Ollama liveness gate + soft-assertion/accuracy-floor/drift-tracking, as an optional package — promotes expenses' copies into the lib once a second LLM consumer appears (LTG would be it). |
| **Polling-based OBSERVE** | acceptance-test's blind `time.Sleep` anti-lesson | `WaitFor(ctx, timeout, interval, predicate)` helper for async CLIs/pipelines; the missing piece for any trigger→eventually-observable consumer. |
| **Order-insensitive + status-only JSON assertions** | acceptance-test | `ignore_order_on` dot-path array normalization; `expected_http_result_code`-style "status only" checks. Cheap adds to `verify/json.go` when a consumer needs them. |
| **Concurrent assertion executor + circuit breaker** | acceptance-test | Worker pool with config-driven concurrency, context-cancel after N failures. Only worth it at tens-of-thousands-of-assertions scale — wait for a real consumer. |
| **`MANUAL` step convention** | LTG | A `Then` marker that logs "requires human judgment: <text>" and records rather than asserts — the automatable half of LTG's probe records. Docs-first (PATTERNS.md), code later. |
| **`.ljson` line-numbered expected files** | acceptance-test | Traceable many-assertion expected files; consider only if a consumer outgrows per-file expected fixtures. |
| **Harness self-CI** | new repo | GitHub Actions: vet + unit tests on push. Trivial once public. |

## 7. LTG adoption path (documented, not built)

LTG's acceptance tests today are hand-run probe records (`probes/*-acceptance.md`)
following GIVEN→RUN→OBSERVE→ASSERT. The harness fits because LTG's OBSERVE captures are
already subprocess-shaped: exit codes, stdout lines, files on disk, row counts —
exactly what `Context{ExitCode, Stdout, Artifacts}` holds. Path when LTG wants it:

1. A tiny Go module inside LTG (e.g. `acceptance/go.mod`) importing the harness;
   `actions/` shells to the `run-*.sh` wrappers (CLAUDE.md there mandates driving via
   the shims); `verify/` asserts on exit codes, stdout, JSONL/table row counts.
2. Determinism vocabulary maps onto existing tiers: structural-equivalence assertions
   are hard; `ollama_calls == 0` (AT-2's defining incremental assertion) is an
   `OutputContains`-class negative check against instrumented counters; judgment
   steps use the `MANUAL` convention (roadmap).
3. Alternative LTG's docs imagine — a Python-native runner bound into
   `ltg_inspect.py` (T-34 `acceptance_mode` residue) — remains valid; the Go harness
   is an option, not a mandate. Decide there, not here.

## 8. Risks / open questions

- **Public-module churn:** v0.x tags signal instability; both consumers pin exact
  versions. Don't chase a stable v1.0.0 API until expenses has migrated (the second
  consumer's fallout is the last boundary test).
- **Two consumers, one author:** breaking changes are cheap now — prefer fixing the
  API in v0.x over compatibility shims.
- **Env-var vs flag for keep-artifacts** (§3.6) is a guess; if it annoys, revisit in
  v0.2 with `RegisterFlags()`.
- **`Context.T` couples the lib to `*testing.T`.** Fine — this is a go-test library by
  design (acceptance-test's history shows the CLI-runner idea died there too). Do not
  abstract it speculatively.
