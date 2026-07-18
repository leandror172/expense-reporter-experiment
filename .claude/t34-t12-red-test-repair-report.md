# T-34 + T-12 Red-Test Repair Report

**Date:** 2026-07-17 (session 60)
**Status:** COMPLETE. Full acceptance roster 49 pass / 0 fail / 0 skip (1311s) — first
fully-green suite since the T-32 gate landed. Committed to master (`fix(test)`, after the
T2 go.mod bump commit).

## Baseline

Session 59's Phase-0 roster: 44 pass / 5 fail, all 5 pre-existing and deterministic
(`.claude/scratch/phase0-roster.txt` has the per-test diagnosis).

## T-34 — gate-stale CLI args (2 tests, `test/auto_log_append_test.go`)

- **Was:** both tests passed `Uber Centro` as a literal `When:` arg. Written pre-T-32 when
  confidence gated. The agreement gate rejects it (keyword `uber` spec 0.8, ambiguous
  Viagens/Uber-Taxi) → routes to review → nothing appends → red. The gate was right; the
  expectation was stale. File comments still claimed "Uber Centro → 100% stable" and one
  falsely referenced "Netflix".
- **Now:** both use canonical gate-passing `Posto Ipiranga` (spec 1.0 → Combustível,
  unambiguous, model agrees); item expectations updated; comments rewritten to explain the
  T-32 constraint.

## T-12 — dark workbook-gated tests (3 tests: `auto_test.go` ×2, `feedback_test.go` ×1)

- **Was:** gated on `RequireWorkbook` (so they skipped silently for months), copied the
  workbook into the workdir, and configured no taxonomy. When the workbook file reappeared
  they ran for the first time post-T-13 and failed `taxonomy path not configured`.
- **Now:** workbook gate + copy dropped entirely (WS-B: `auto` is log-append only, no
  workbook involvement); Givens wire `withFeedbackAndTaxonomyConfig`; new
  `fixtures/auto-basic/fixture-taxonomy.json` (Variáveis/Transporte leaves + an
  Extras/Diversos/Diversos leaf so the T-19 sentinel decline has a landing spot);
  redundant `expenseClassifierAvailable` helper deleted; `TestAuto_FeedbackLoggedOnInsert`
  passes a full `15/04/2026` date and the expected files match it (T-11 normalization means
  `ExpandAndAppend` always logs DD/MM/YYYY — a bare `15/04` expectation could never match);
  `expected-expenses_log.jsonl` now also asserts the full typed path
  `Variáveis/Transporte/Combustível` (post-T-13 strengthening at zero flake cost — if the
  gate passed at all, the subcategory IS Combustível).

## Latent findings (NOT fixed — backlog candidates)

1. **Join-ID divergence on short dates:** `auto` passes the RAW date string to
   `logConfirmedFeedback` but the NORMALIZED one to `ExpandAndAppend`. Input `15/04` →
   `classifications.jsonl` logs `15/04`, `expenses_log.jsonl` logs `15/04/2026` → the
   sha256 `GenerateID` join key differs across the two logs for the same expense. Same
   family as T-18's root cause. Candidate fix: normalize once in `runAuto` before both
   consumers.
2. **Stale naming:** `TestAuto_HighConfidence*` names describe the pre-T-32 confidence
   gate; `fixtures/auto-basic/input.csv` references a `TestAuto_Basic` that doesn't exist.
   Cosmetic, untouched.

## Suite-time consequence

Full suite grew 813s → 1311s: the five repaired tests now do real classification work
instead of failing fast. Relevant to the `-timeout` guidance in T-17's caveat (current
`run-acceptance.sh` passes 1800s — still headroom, but T-08's timeout-flake class gets
closer as tests are added).
