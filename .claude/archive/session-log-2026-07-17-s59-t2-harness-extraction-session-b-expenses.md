## 2026-07-17 - Session 59: T2 harness extraction Session B — expenses migrated onto the acceptance-harness module (PR #48); consumer report + doc fixes (module PR #1)

### Context

Continued T2 (harness extraction). Session A (s58) published `acceptance-harness` v0.1.1 and repointed career-search; this session executed **Session B** — migrating expense-reporter's own acceptance suite onto the module so `test/harness/` no longer exists locally.

### What Was Done

- Wrote the executable Session B plan (`.claude/plans/harness-extraction-session-b.md`) — the parent plan's §5 was six bullets. Measured every touch point first; advisor-reviewed D1/D2 before implementation.
- **Phase 0** — captured a baseline acceptance roster (`.claude/scratch/phase0-roster.txt`): 44 pass / 5 fail / 0 skip, so migration breakage is distinguishable from pre-existing red under model nondeterminism.
- **Phase 1** — carved expense-shaped helpers out of `test/harness/`: `RequireOllama`→`test/extern/`; workbook gate+copy, `SetupBinaryConfig`, fixture-config decode, Env accessors →`test/domain/`; folded the 2 live comparator funcs into `test/expect/` and deleted 2 dead ones (`CompareCSVExact/Fuzzy`, 0 sites).
- **Phase 2+3** — deleted `test/harness/`; swapped the engine to `acceptance-harness` v0.1.1 (`setup_test.go` onto `FindModuleRoot`/`BuildBinary`/`RegisterFlags`). D1: local `verify`→`test/expect/`, 8 module-twin duplicates deleted (verified byte-identical first). D2: `Context.{DataDir,WorkbookPath}`→`ctx.Env`. `FixtureConfig` domain fields decode from the module's `Raw`.
- **Phase 4 gate** — full acceptance roster **byte-identical pre/post** (same 44/5, same failure reasons 14×/7×, soft-accuracy above floor). Opened expenses **PR #48**.
- **Phase 5** — repointed all acceptance docs/memories at the module; fixed the module's `ADOPTION.md` verify-collision + the `TestMain` temp-dir leak (in ADOPTION + README); added `docs/consumer/expense-reporter.md` (evidence-grounded consumer report). Module changes → **PR #1** (branch `docs/session-b-followups`).

### Decisions Made

- **D1/A — module owns `verify.*`; domain verifiers → `test/expect/`** (over a zero-churn facade). Advisor tipped it: the domain carve-out already forces ~41 prefix renames, so A's 38 `verify.`→`expect.` edits are the same kind of edit, and A keeps ONE adoption pattern across both consumers (career-search chose `tracker`). Scenarios now import both `verify.*` (generic) and `expect.*` (domain).
- **D2/A — `ctx.Env` carrier** for `DataDir`/`WorkbookPath`. They were flag-defaults, not scenario state (every read was in `actions/`); `runCommand` forwards `ctx.Env` to the subprocess, honoring Env's contract. A domain wrapper struct is structurally blocked — `harness.Run` hands `*harness.Context` to the callbacks.
- **Left the 5 pre-existing reds untouched** — fixing them would conflate changes and destroy the like-for-like gate. Filed (T-34 + T-12 residue).
- **`extern/llm` promotion** (RequireOllama + accuracy-floor + drift-tracking) spec'd in the consumer report from expenses' real API, but **gated on LTG** as a second LLM consumer — spec, not build-now.

### Next

- Merge expenses **PR #48** + module **PR #1**; then tag `acceptance-harness` **v1.0.0** (gated on #48 — expenses was the last boundary test).
- Product step: **WS-D (T-09)** — HELD, design gate-to-review not silent insert; or **T-20** dedup (independent, deterministic).
- Fix the 2 red tests: **T-34** (`Uber Centro` CLI-arg swap) and **T-12** (3 dark workbook-gated tests need taxonomy config in their Given).

### Gotchas

- `RegisterFlags()` is **mandatory, not optional**: `run-acceptance.sh` forwards `-keep-artifacts`/`-keep-on-failure` to `go test`, and the module turned those into env vars — without it the script dies on an unknown flag.
- The 3 dark T-12 tests fail with `taxonomy path not configured` — the **same signature** a broken `SetupBinaryConfig` temp-dir would produce post-migration. The Phase-0 baseline is the only thing that proves they're pre-existing, not this session's break.
- Three `test/` files carried CRLF (blobs predate `.gitattributes` `eol=lf`); the stray CR silently defeated an anchored regex mid-refactor. `gofmt -l test/` reported 7 files on master → now clean.
- Consumer-report finding: the module's `CopyTreeToWorkDir`, `DiscoverFixtures`, nested-JSON walker, and **all** of `verify/file.go` are unused by expenses (the only LLM consumer). `FileUnchanged` (byte + same-name) structurally can't express a non-deterministic, differently-seeded "nothing was written" check — a v1.0.0 shape signal.
