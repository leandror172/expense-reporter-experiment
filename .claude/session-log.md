# Session Log — Expense Reporter

**Current Session:** 2026-07-13 — Session 58: T2 harness extraction Session A — acceptance-harness module published (v0.1.1); career-search repointed (TC-EXT closed, PR #7)
**Current Layer:** Layer 5 — Expense Classifier (T2 harness extraction Session A DONE — module published, career-search repointed; Session B = expenses migration next)
Most recent entry first. Run `.claude/tools/rotate-session-log.sh` when this grows beyond ~3 sessions.

---
## 2026-07-13 - Session 58: T2 harness extraction Session A — acceptance-harness module published (v0.1.1); career-search repointed (TC-EXT closed, PR #7)

### Context

PR #46 (T-32) merged before session start. The session executed the T2 harness-extraction Session A plan end-to-end; all code landed in the acceptance-harness and career-search repos, so this repo has zero session commits (handoff-harvest empty by design).

### What Was Done

- Created + published `github.com/leandror172/acceptance-harness` (public, MIT): engine (Context/Scenario/Run, fixture plumbing incl. recursive tree copy, BuildBinary/FindModuleRoot) + generic verify split cli/json/file, seeded from career-search's de-domained copy (diff vs expenses master first: nothing to backport). Build tags stripped; retention flags → `HARNESS_KEEP_ARTIFACTS`/`HARNESS_KEEP_ON_FAILURE` env vars + opt-in sync.Once `RegisterFlags()`. Tags v0.1.0, then v0.1.1 (go directive 1.24→1.25).
- Ported the methodology docs de-domained (`docs/PATTERNS.md`, `docs/ADOPTION.md`, `docs/LTG-PATH.md`) + README; added `.memories/{QUICK,KNOWLEDGE}.md` and `CLAUDE-DRAFT.md` (tentative CLAUDE.md under an alternate name — adoption undecided).
- 30 lib unit tests, vet/gofmt clean, written via my-go-qcoder (7 calls: 4 accepted/improved, 1 rejected+rewritten, 1 cold-start retry). The tests exposed and fixed a latent `Run()` bug: `t.Name()` fed to `os.MkdirTemp` crashes under `t.Run` subtests ("pattern contains path separator") — present in BOTH consumers' copies.
- Repointed career-search (TC-EXT closed; PR #7 open): local `test/harness/` + generic `verify/{cli,json}.go` deleted (−689 lines); domain verifiers moved to `test/tracker/` so the module keeps the `verify.*` namespace (146 generic call sites untouched; 44 domain renames in 3 files); `harness.RegisterFlags()` in TestMain preserves the `--keep-*` flags; go 1.24.2→1.25.5; full suite 65 tests green ~3s; `test/.memories` rewritten; migration report `.claude/plans/tracker-cli-report-tc-ext.md` (indexed).

### Decisions Made

- Seed from career-search's copy, NOT the expenses original (plan §3 sequencing) — the second consumer's copy encodes two consumers' needs; the pre-seed diff proved the origin hadn't grown anything since.
- Module go directive tracks the LOWEST consumer: shipped at 1.24 (career-search was 1.24.2), bumped to 1.25 in v0.1.1 only after career-search moved to 1.25.5.
- T2 stays UNCHECKED — its own text requires "both repos depend on it"; the expenses migration (Session B, plan §5) is the remaining half.
- No v1.0.0 until the expenses migration shakes the API; consumers pin exact v0.x versions, breaking changes preferred over compat shims.

### Next

- Harness extraction Session B (plan §5): migrate expenses `test/` to the module — add dep, delete `test/harness/` EXCEPT `ollama.go`/`comparator.go` (→ domain layer, e.g. `test/extern/` + `test/verify/csv_compare.go`), Context workbook-field fallout via `Env`/wrapper, `FixtureConfig` domain fields via `Raw` decode. Gate: full `-tags=acceptance -timeout 30m` green AND `-tags=replay` still compiles.
- Merge career-search PR #7 (user); decide CLAUDE-DRAFT.md adoption in acceptance-harness.
- Alternatives: WS-D gate-to-review design (T-09), T-20 log dedup.

### Gotchas

- A lib's go directive must stay ≤ every consumer's — a higher directive forces the consumer's own go line up on `go get`.
- Pushing to GitHub from this environment needs `git -c credential.helper='!gh auth git-credential' push` (plain push has no credential terminal); `gh repo create --public` is permission-classifier-gated — the user runs it via `!`.
- career-search PR #7 is based on a local master 9 commits ahead of origin — those commits appear in the PR until master is pushed (user manages).
