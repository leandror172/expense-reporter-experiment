# T2 Closeout Report — acceptance-harness v1.0.0

**Date:** 2026-07-17 (session 60)
**Status:** COMPLETE. T2 (harness extraction) is fully closed.

## What happened

1. **Expenses PR #48** (Session B migration) — merged by the user before this session.
2. **Module PR #1** (consumer report + verify-collision/TestMain doc fixes) — merged.
3. **career-search PR #7** (module repoint) — merged.
4. **`acceptance-harness` v1.0.0 tagged** at `11beb2a` (the PR #1 squash commit) and pushed.
   Annotated tag; message records the two-consumer boundary test as the API-freeze rationale.
5. **Expenses bumped** `go.mod` v0.1.1 → v1.0.0 (commit `chore(t2)` on master).
   Docs-only delta over v0.1.1 (identical `/go.mod` hash in go.sum — no behavioral surface
   moved). Verified: `go build`, `go vet` (incl. `-tags=acceptance ./test/...`), all unit
   tests green.

## The v1.0.0 shape decision

The Session B consumer report flagged module surface unused by expenses
(`CopyTreeToWorkDir`, `DiscoverFixtures`, nested-JSON walker, all of `verify/file.go`) as a
possible pre-1.0 pruning signal. **Checked before tagging: career-search uses all of it**
(`CopyTreeToWorkDir` in `dashboard/test/helpers_test.go`, `verify.FileUnchanged`/
`FileContains`/`WorkFileExists` across its suite, `verify.JSONFieldEquals` nested paths).
Not dead surface — surface justified by the non-LLM consumer. The one open gap
(`FileUnchanged` can't express a non-deterministic "nothing was written under any name"
check) is **additive** → a future minor bump, not a 1.0 blocker.

## Standing consequence

Engine changes are now: PR upstream → version bump in each consumer. Semver contract active
as of v1.0.0 — removals/signature changes cost a major bump.
