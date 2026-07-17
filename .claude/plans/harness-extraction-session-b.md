# Session B — Migrate expense-reporter onto the `acceptance-harness` Module (T2)

**Date:** 2026-07-17 (session 59 planning)
**Supersedes:** `harness-extraction-plan.md` §5 (six bullets — this is the executable form)
**Module:** `github.com/leandror172/acceptance-harness` v0.1.1 (public, MIT, go 1.25)
**Precedent:** career-search TC-EXT — `career-search/.claude/plans/tracker-cli-report-tc-ext.md`
**Goal:** delete `expense-reporter/test/harness/`, depend on the module, keep the full
acceptance suite green. Unblocks the module's v1.0.0 (expenses is the last boundary test).

---

## 1. Read before executing

| Source | Why |
|---|---|
| `.claude/plans/harness-extraction-plan.md` §3 (design rules), §5, §8 (risks) | The parent plan; §3.1 lean-core rule is why ollama/comparator can't migrate |
| `career-search/.claude/plans/tracker-cli-report-tc-ext.md` | Same migration, done once. §2 = the namespace strategy that this repo must decide differently |
| `~/workspaces/acceptance-harness/docs/ADOPTION.md` | Consumer wiring (TestMain, `Raw` decode, `Env` seam, retention env vars). **Note §2 has a naming bug — see §4 below** |
| `expense-reporter/test/.memories/KNOWLEDGE.md` | The Given/Then conventions the migration must not break |
| `expense-reporter/test/PATTERNS.md` | Ditto; gets repointed at the module's docs in Phase 5 |

## 2. Measured inventory (surveyed session 59 — trust these numbers, they corrected the plan)

### `test/harness/` — 354 lines, 4 files

| File | Symbol | Sites | Disposition |
|---|---|---|---|
| `scenario.go` | `Context`, `Scenario`, `Run` | 269 / 49 | **→ module** (drop-in; module's `Run` also fixes the `t.Name()` MkdirTemp bug) |
| `scenario.go` | `keepArtifacts`/`keepOnFailure` flags | — | **→ module** env vars + `RegisterFlags()` |
| `scenario.go` | `RequireWorkbook` | 3 | **→ domain** |
| `fixture.go` | `FixtureConfig`, `LoadFixtureConfig` | 3 | **→ module** + domain `Raw` decode (§5 Phase 3) |
| `fixture.go` | `CopyFixtureToWorkDir` | 23 | **→ module** (drop-in) |
| `fixture.go` | `SeedFileFromFixture` | 6 | **→ module** (drop-in) |
| `fixture.go` | `DiscoverFixtures` | **0** | module has it; expenses never used it — nothing to do |
| `fixture.go` | `CopyWorkbookToWorkDir` | 6 | **→ domain** |
| `fixture.go` | `SetupBinaryConfig` | 13 | **→ domain** (parent plan §4: "leave it behind" — encodes expense's `os.Executable()` config resolution) |
| `ollama.go` | `RequireOllama` | 19 | **→ domain** (lean-core §3.1: no LLM, no network) |
| `comparator.go` | `ReadCSVFile` | 2 | **→ domain**, both callers in `test/verify/` |
| `comparator.go` | `ConfidenceInRange` | 1 | **→ domain**, caller in `test/verify/csv.go` |
| `comparator.go` | `CompareCSVExact`, `CompareCSVFuzzy` | **0** | **DELETE — dead code** |

### Dead on arrival (delete, don't migrate)

- `CompareCSVExact`, `CompareCSVFuzzy` — 0 call sites (the parent plan assumed comparator.go
  migrates wholesale; **more than half of it is dead**)
- `Context.WorkbookDir` — 0 references anywhere in the repo

### Domain `Context` fields → the `Env`-vs-wrapper question

| Field | Reads | Writes |
|---|---|---|
| `DataDir` | 8, **all in `actions/commands.go`**, all the same `if != "" { append("--data-dir", …) }` idiom | ~30 `ctx.DataDir = dataDir` lines in Givens |
| `WorkbookPath` | 9, **all in `actions/commands.go`**, same idiom | 2 in tests + 1 in `CopyWorkbookToWorkDir` |
| `WorkbookDir` | 0 | 0 — **dead** |

**The shape matters:** these are not scenario state, they are *CLI-flag defaults*. Every read
is in one file. That is what makes Option A below cheap.

### `test/verify/` — 892 lines, 33 exported funcs, 81 call sites

- **43 sites** call functions that are **name-twins of module functions**:
  `CommandSucceeded` (12), `OutputJSONHasKey` (9), `OutputContains` (8), `OutputFileExists` (7),
  `OutputNotContains` (4), `CommandFailed` (2), `OutputIsValidJSON` (1)
- **38 sites** call domain-only functions (Feedback*, ExpenseLog*, Classifications*, Workbook*,
  HTML*, accuracy, rollover, OutputJSONHas{Type,Category,Action}, OutputFileHas{Rows,Columns})

**All 8 twin bodies read and verified byte-identical** to the module's (career-search seeded the
module from this repo's de-domained copy) — with one exception: module `OutputJSONHasValue` uses
`assert.EqualValues` where expenses uses `actual != expected` (parent plan §3.3). That is
strictly *more permissive*, so it can only turn red→green, never green→red. Safe.

**Specifically checked (advisor, session 59):** `OutputContains`/`OutputNotContains` both scan
`ctx.Stdout + ctx.Stderr` in *both* copies. This was the one pass→fail vector in the whole
migration — had the local `OutputNotContains` scanned stdout only, the module's wider scan would
have flipped ~4 passing sites to failing (2 in `json_output_test.go`). It does not. Vector closed;
no re-check needed.

### Ratio vs the precedent — **this is why expenses can't just copy career-search**

| Repo | Generic sites | Domain sites | Split |
|---|---|---|---|
| career-search | 146 | 44 | 77 / 23 → module obviously wins the `verify` name |
| **expense-reporter** | **43** | **38** | **53 / 47 → no clear winner** |

## 3. Decisions — LOCKED

1. **Everything stays under `expense-reporter/test/`.** "Domain layer" = *this repo's* test
   packages vs *the shared module*. Nothing moves outside `test/`.
2. **`ollama.go` → `test/extern/ollama.go`** (package `extern`), `RequireOllama` unchanged.
   Rationale: lean-core §3.1 forbids it in the module; `extern` names the concept (an external
   service gate) and matches the roadmap's future `extern/llm` module package, so a later
   promotion is a move, not a redesign.
3. **`comparator.go` does NOT become `test/verify/csv_compare.go`.** Delete the two dead
   functions; fold `ReadCSVFile` + `ConfidenceInRange` into `test/verify/` as **unexported**
   helpers (`readCSVFile`, `confidenceInRange`) beside their only callers. A named file for two
   private helpers is ceremony. *(Departs from parent plan §5.1 — justified by the 0-call-site
   finding.)*
4. **`RequireWorkbook` + `CopyWorkbookToWorkDir` + `SetupBinaryConfig` → `test/domain/`**
   (package `domain`). The parent plan never assigned these a home; they are the same class as
   ollama/comparator — expense-shaped helpers the module rejects.
5. **Keep the `acceptance` build tag and `run-acceptance.sh` unchanged** (consumer policy).
6. **Adopt `harness.FindModuleRoot` + `harness.BuildBinary`** in `setup_test.go` — expenses
   hand-rolls both today (module has no `buildbin.go` equivalent locally). Net line deletion.
7. **Pin the module at an exact version** (v0.1.1); expect v0.x breaking changes.

## 4. Decisions — RESOLVED (advisor, session 59)

### D1 — the `verify` namespace collision → **Option A: module owns `verify`; domain → `test/expect/`**

Both packages are named `verify` and 8 function names exist in both. The module's own
`docs/ADOPTION.md` §2 is self-contradictory here — it draws `test/verify/ # domain assertions
(yours)` while also saying "the generic `verify` comes from the library". **Fix that doc bug in
the module this session, documenting Option A** (career-search's pattern) as *the* adoption path.

| Option | Mechanics | Churn | Verdict |
|---|---|---|---|
| **A — module owns `verify`** | delete the 8 twins; domain verifiers → `test/expect/` (package `expect`). Two *distinct* names, **no aliasing** | 38 sites `verify.X`→`expect.X` | **ADOPTED** |
| **B — local `verify` keeps the name, module aliased** | `import base ".../verify"` | 43 sites | Rejected — this (not A) is the "two same-named packages + alias in every file" career-search rejected |
| **C — facade** | local `verify` re-exports the 8 twins as thin wrappers | 0 sites | Rejected — see below |

**Why A beats C** (this reverses the plan's first-draft recommendation):

1. **C's zero-churn benefit is mostly illusory in this session.** The locked domain moves already
   force ~41 prefix renames (`RequireOllama` 19 → `extern.`, `SetupBinaryConfig` 13 + 
   `CopyWorkbookToWorkDir` 6 + `RequireWorkbook` 3 → `domain.`). A's 38 `verify.X`→`expect.X`
   edits are *the same kind of edit in the same files while they're already open* — not a new
   category of cost. C buys zero churn on one dimension of a session that pays it on every other.
2. **My "53/47 evaporates A's benefit" reasoning was wrong.** The ratio drives A's *benefit*
   (how many generic sites ride free), but A's *cost* is the domain-rename count: **~38 here vs
   ~44 in career-search — nearly identical**. If ~44 was acceptable there, ~38 is acceptable here.
3. **A shared library should teach one adoption pattern.** The module already has a consumer on
   A. C would leave two structurally-divergent consumers and a permanent indirection layer to
   describe in the very doc we're fixing.

### D2 — `DataDir`/`WorkbookPath` carrier → **Option A: `ctx.Env`**

| Option | Mechanics | Churn | Verdict |
|---|---|---|---|
| **A — `ctx.Env`** | Givens set `ctx.Env["DATA_DIR"]`; `commands.go` reads it into the `--data-dir` flag | ~40 mechanical | **ADOPTED** |
| **B — action parameters** | closures capture: `actions.RunClassify(dataDir, args...)` | ~40 + every action signature | Rejected — honest but costlier |
| **C — domain wrapper struct** | wrapper holding `*harness.Context` + fields | — | **STRUCTURALLY BLOCKED — do not re-litigate.** `harness.Run` hands `*harness.Context` to Given/When/Then; a wrapper cannot be the callback param without re-wrapping `Run` |

The module inits `Env` to a non-nil map in `Run`, so no nil-panic. Known wart, accepted: `Env`'s
documented purpose is *"extra environment for the command"*, so if `runCommand` forwards it to the
subprocess env, `DATA_DIR` leaks into the binary's environment. Harmless — the binary reads the
flag, not the env.

**Optional refinement (advisor, take it or leave it):** `DataDir` is near-constant — the same path
for every test, with `embedding_fallback_test.go` the *only* override (to a temp dir).
`WorkbookPath` is the only genuinely per-scenario value. Setting a default `DATA_DIR` once in the
exec wrapper and letting `ctx.Env` override it collapses the ~30 `ctx.DataDir = dataDir` Given
lines to a single override — a net simplification over today. Pursue only if fewer lines is worth
more than one uniform mechanism; Env-everywhere is fine as-is. **Decide in Phase 2, not before.**

## 5. Phases

Each phase ends compiling. Commit per phase ([[feedback_tdd_commit_cadence]]).

### Phase 0 — baseline — **DONE (session 59). Roster: `.claude/scratch/phase0-roster.txt`**

**Result: 44 PASS / 5 FAIL / 0 SKIP, 813s.** All 5 failures are **pre-existing, deterministic,
and diagnosed** — none are the migration's. Phase 4 must reproduce exactly these 5; a 6th is mine.

- **Class A (3)** — `TestAuto_KnownExpenseIsClassifiedWithConfidence`,
  `TestAuto_AmbiguousExpenseKeptForManualReview`, `TestAuto_FeedbackLoggedOnInsert`.
  Dark tests exposed because the workbook file *exists*, so `run-acceptance.sh` sets
  `EXPENSE_WORKBOOK_PATH` and `RequireWorkbook` stops skipping them. They then fail
  `taxonomy path not configured` (T-17 class — their Givens never `SetupBinaryConfig` a
  taxonomy). Residue of tasks.md **T-12**; the exact trap in `test/.memories/KNOWLEDGE.md`.
- **Class B (2)** — `TestAuto_HighConfidenceAppendsToLog`,
  `TestAuto_HighConfidenceInstallmentsExpandToNEntries`. **T-32's item swap missed this file:**
  session 57 swapped `Uber Centro`→`Posto Ipiranga` in auto-append *fixtures*, but
  `auto_log_append_test.go` passes the item as a literal CLI arg in `When:`. `Uber Centro` has
  keyword spec 0.8 + AMBIGUOUS → correctly rejected by the agreement gate → no append →
  `classifications.jsonl` never created. Model-independent, so deterministic red.

> **⚠ Class A's error string is `taxonomy path not configured` across several `auto` tests —
> which is precisely the signature §6 warns a `SetupBinaryConfig` temp-dir break would produce.
> They are NOT that. Do not "fix" them in Phase 4. Without this baseline, that hunt was the
> default outcome of this session.**

**Scope call:** both classes are pre-existing bugs with open-task homes (T-12 / T-32 residue).
They stay red through Session B — fixing them here would conflate two changes in one PR and
destroy the like-for-like Phase 4 comparison. File them; do not fix them.

### Phase 0 — procedure (for reference / re-runs)
1. `git checkout -b feat/t2-session-b-harness-migration`
2. `./run-acceptance.sh` → **record the pass/fail/skip roster**. The suite needs live Ollama and
   q3 ≈12 s/classify, so budget ~20–30 min and use `-timeout 30m`.
3. Any test already red/skipped **stays that way** — it is not this migration's job. Without this
   roster you cannot tell a migration break from a pre-existing one. This repo's known trap class
   is exactly this ([[feedback_rename_json_tag_acceptance]]).
4. **Record the roster in two buckets**, because Phase 4 compares them differently:
   - **deterministic tests** (generate*, apply, correct, review, json_output, the pre-flight
     fail-fast cases) → must match pass/fail **exactly**
   - **soft-assertion / Ollama-gated tests** (accuracy, fewshot, batch-auto classification) →
     record the *accuracy number*, not just pass/fail; these legitimately wobble run-to-run

### Phase 1 — add the dep, carve out the domain files
1. `go get github.com/leandror172/acceptance-harness@v0.1.1` (go.mod is already go 1.25.5 ≥ the
   module's 1.25 — no floor conflict, unlike career-search).
2. Create `test/extern/ollama.go` (package `extern`) — `RequireOllama` verbatim, keep the tag.
3. Create `test/domain/workbook.go` + `test/domain/config.go` (package `domain`) —
   `RequireWorkbook`, `CopyWorkbookToWorkDir`, `SetupBinaryConfig` verbatim. **`CopyWorkbookToWorkDir`
   sets `ctx.WorkbookPath`** — it must be rewritten under D2's answer (Phase 2), so port it
   as-is here and fix it there.
4. Fold `ReadCSVFile`→`readCSVFile` and `ConfidenceInRange`→`confidenceInRange` into
   `test/verify/csv.go`; **delete `CompareCSVExact` + `CompareCSVFuzzy`**.
5. `test/harness/` now holds only module-bound code. Do not delete it yet.

### Phase 2 — swap the engine
1. Delete `test/harness/`.
2. Rewrite `setup_test.go` onto `harness.FindModuleRoot` + `harness.BuildBinary`
   (ADOPTION.md §4). Keep `dataDir`/`testWorkbook` globals and `fixturesDir()`.
3. **`harness.RegisterFlags()` in `TestMain` is REQUIRED** — not the judgment call this plan
   first assumed. Verified session 59: `run-acceptance.sh` forwards `-keep-on-failure` /
   `-keep-artifacts` to `go test`, and they bind to the package-level `flag.Bool` vars in the
   current `scenario.go`. The module replaced those with env vars, so without `RegisterFlags()`
   the script dies on an unknown flag. Same reason career-search added it.
4. Rewrite imports `expense-reporter/test/harness` → `github.com/leandror172/acceptance-harness/harness`
   across `test/*.go`, `test/actions/`, `test/verify/`.
5. Apply D2/A: drop `DataDir`/`WorkbookPath`/`WorkbookDir` from every site; carry via `ctx.Env`.
6. Apply D1/A: delete the 8 twins from local `verify`; move the 25 domain verifiers to
   `test/expect/` (package `expect`); rename the 38 call sites `verify.X` → `expect.X`.
7. `go vet -tags=acceptance ./...` + `gofmt -l test/` clean.

### Phase 3 — `FixtureConfig` domain fields
Expenses' `Model`/`Threshold`/`AssertionType`/`AccuracyFloor`/`TopN` are gone from the module's
`FixtureConfig` (which keeps `Command`/`ExtraArgs`/`Raw`). Add a domain loader wrapping the lib's
(ADOPTION.md §6 shows the exact pattern):

```go
// test/domain/fixture.go
type ExpenseFixtureConfig struct {
    harness.FixtureConfig
    Model, AssertionType string
    Threshold, AccuracyFloor float64
    TopN int
}
func LoadExpenseFixtureConfig(dir string) (ExpenseFixtureConfig, error) // decode from base.Raw
```

**Preserve the defaults** the old `LoadFixtureConfig` applied: `Threshold=0.85`, `TopN=3`,
`AssertionType="hard"`. Dropping them silently changes fixture behavior.
**Note:** `Threshold` is inert post-T-32 (`--threshold` deprecated; the agreement gate ignores
it) — decode it anyway to keep fixture configs parsing, but do not add new uses.
Only 3 `LoadFixtureConfig` call sites, all in one file.

### Phase 4 — gates
1. **`./run-acceptance.sh` with `-timeout 30m`** → compare to the Phase-0 roster **per bucket**:
   - **deterministic tests: pass/fail identical.** Any change here is the migration. No excuses.
   - **soft-assertion tests: still above floor.** A soft test that dipped but holds its floor is
     Ollama nondeterminism, **not** a migration break — do not chase it. A soft test that went
     *below* floor is worth one re-run before you believe it.
   - Pre-existing skips must still skip **for the same reason** (check the skip message, not just
     the count — `RequireOllama` and `RequireWorkbook` skip for very different reasons).
2. `go build ./...` and `go test ./...` (untagged) still green.
3. `go vet -tags=replay ./internal/classifier/` — **grep-confirmed the 3 replay files import
   nothing from `test/`**, so this is expected to be a no-op; run it anyway (the go.mod change is
   the only plausible vector).
4. `grep -rn "expense-reporter/test/harness" .` → no hits.

### Phase 5 — docs (do not defer to handoff)
- `test/README.md` — layout shows the module split; retention knobs (env + optional flags)
- `test/PATTERNS.md` — points at the module's `docs/PATTERNS.md` for the generic half
- `test/.memories/{QUICK,KNOWLEDGE}.md` — module import, the `verify`/`expect` split (D1),
  the `Env` carrier (D2). The Given/Then convention text must now say **`expect.*`** inside
  `then*` helper bodies, not `verify.*`
- `test/harness-external-usage.md` — header note: superseded/executed
- `.claude/index.md` — `test/harness` row in `[ref:go-structure]`; new `test/extern`,
  `test/domain`, `test/expect` rows
- **Module repo:** fix the `docs/ADOPTION.md` §2 collision (§4/D1) — document **Option A**
  (module owns `verify`, domain assertions get their own package name) as *the* adoption
  pattern, matching both consumers. This is the doc that told us `test/verify/` was "yours"
  while also handing us a `verify` package
- `.claude/tasks.md` T2 — at handoff only ([[feedback_tasks_md_handoff_only]])

## 6. Risks / traps

- **The build tag hides everything.** `go test ./...` stays green through a totally broken
  acceptance suite. Only `-tags=acceptance` tells the truth. This repo has been burned by exactly
  this twice (T-17, T-18). [[feedback_rename_json_tag_acceptance]]
- **The Phase-0 roster is the whole safety net.** The suite is Ollama-gated and partly
  soft-assertion — "it went red" can mean model drift, not migration breakage. Without the
  before-roster you cannot tell.
- **`SetupBinaryConfig` writes next to the binary.** `BuildBinary` puts the binary in a *different*
  temp dir (`acceptance-bin-*`) than expenses' current `expense-reporter-acceptance-*`. The path
  is still `filepath.Dir(ctx.BinaryPath)/config/config.json`, so it should hold — but this is the
  most likely silent breakage in Phase 2, and it fails as `taxonomy path not configured` across
  ~13 tests, which looks exactly like the T-17 regression. **Check this first if Phase 4 goes red.**
- **Log-string behavior loss:** the module's `Run` logs `→ When: executing command`; expenses'
  logs `… (may take a while — waiting for Ollama)`. That was the de-domaining (§3.1 grep gate).
  Cosmetic; do not "fix" it by patching the module.
- **`OutputJSONHasValue` semantics shift** (`!=` → `EqualValues`) — red→green only, but if a test
  *starts* passing that previously failed, that is this, not luck.
- **Don't chase v1.0.0 in this session.** Tag it only after Phase 4 is green (parent plan §8).

## 7. Definition of done

- [ ] `test/harness/` gone; no import of it anywhere
- [ ] Module pinned at an exact version in go.mod
- [ ] Domain assertions live in `test/expect/`; `verify` resolves to the module (D1/A)
- [ ] Deterministic roster identical to Phase 0; soft tests above floor
- [ ] `go test ./...` + `go vet -tags=acceptance ./...` + `gofmt -l test/` clean
- [ ] Two dead comparator funcs + dead `WorkbookDir` field deleted
- [ ] Docs (§5 Phase 5) updated, including the module's ADOPTION.md fix
- [ ] PR opened against master ([[feedback_pr_base_no_rebase_fuss]])
- [ ] Module v1.0.0 tag considered (separate call, after merge)
