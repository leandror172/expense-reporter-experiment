## 2026-07-13 - Session 57: "T-32 agreement gate — replace the dead confidence auto-insert gate (PR #46); verifier false-pass fix; runtime-ref cleanup"

### Context

Resumed with PR #45 (5.R2) merged to master. User picked T-32 (agreement gate — the WS-D unblock) from the three ready tracks.

### What Was Done

- Shipped T-32: `IsAutoInsertable` now gates on keyword AGREEMENT (`matched ∧ ¬ambiguous ∧ top_score>=1.0 ∧ predicted==keyword_top1 ∧ ¬excluded`), confidence dropped. Signal surfaced as a pure `MatchStrength(item, keywords)` in `examples.go` (not a `Classify` return-shape change). Deterministic tiebreak; `--threshold` deprecated. Commit 9cf65d1, PR #46.
- Replay-validated the shipped gate over the frozen 649 `model.jsonl` (`gate_validation_test.go`, no re-inference): shipped `top_score` == harness on all 649 rows, reproducing FINDINGS 34.2/86.9 band + 94.9% agreement subcat / 30.5% coverage.
- Fixed a latent verifier false-pass (`FeedbackMatchesExpected` early-returned on the empty log batch-auto's `O_CREATE` preflight always creates) — the agreement gate exposed it on the installment tests.
- Reconciled the acceptance suite: swapped gate-failing `Uber Centro` (spec 0.8, ambiguous) → `Posto Ipiranga` (spec 1.0 → Combustível) in the auto-append fixtures. 620 unit + full acceptance suite green.
- Corrected every runtime ref describing the confidence gate (index.md `[ref:confidence-thresholds]` + its duplicate in `classification_algorithm.md`; 5 memory files). Wrote `.claude/t32-agreement-gate-report.md`.

### Decisions Made

- Gate = AGREEMENT (not pure-specificity, not agreement-AND-confidence). Confidence is inert; ANDing it is strictly dominated (zero precision, random coverage loss, misleading knob).
- Signal as a pure function, not a `Classify` return-shape change (advisor): model/pool-independent, and it avoids the embedding-fallback contamination case + 11-callsite churn.
- Precision caveat carried into the commit/PR/docs: the ~95% is same-sample RELATIVE (shipped==measured); absolute production precision is unmeasured → WS-D stays HELD.
- Collision caveat (5 cross-type leaves can be subcat-right/type-wrong) recorded in `decision.go`, not fixed.

### Next

- WS-D (T-09): retire the bare-name fallback — now protected by the gate but HELD; design gate-to-review, not silent insert (absolute precision unmeasured).
- Or: Harness extraction Session A (T2), or T-20 log dedup — both independent.
- Merge PR #46.

### Gotchas

- `Uber Centro` FAILS the agreement gate (keyword `uber` spec 0.8, ambiguous across Viagens/Uber-Taxi). Use `Posto Ipiranga` / `Netflix` / `Diarista Letícia` for any test that must auto-append.
- `FeedbackMatchesExpected` used to false-pass on an empty log (batch-auto's preflight `O_CREATE`s one) — fixed; watch for the same early-return-on-nil pattern elsewhere.
- The interactive `auto` append-on-pass path has no running test (`TestAuto_FeedbackLoggedOnInsert` is workbook-gated → skips here).
