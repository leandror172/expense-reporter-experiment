# T-32 — Agreement Gate: Work Report & Findings

**Session 57 · branch `feat/t32-agreement-gate` · 2026-07-13**

Replaces the dead confidence auto-insert gate with a keyword **agreement** gate. This is the
WS-D unblock (T-09): the settled next product step per the session-56 handoff, unblocked once
5.R2 (PR #45) merged.

---

## 1. Objective

`IsAutoInsertable` decided the AUTO-vs-REVIEW fork for `auto` / `batch-auto` (post-pivot: append
to `expenses_log.jsonl` vs route to `review.csv`). It gated on `confidence ≥ 0.85 ∧ ¬excluded`.
The session-52 649-replay proved confidence is **dead** as a gate — the whole 0.85–0.95 band is a
coin flip (52.5%→56.3% flat), admitting ~97% of rows. T-32 wires the real signal, already computed
in `SelectExamples` and then discarded: keyword **specificity** + **model⊕keyword agreement**.

## 2. Design decisions (with rationale)

- **Gate formula = Agreement** (user-selected over pure-specificity and agreement-AND-confidence):
  `matched ∧ ¬ambiguous ∧ top_score ≥ 1.0 ∧ predicted_subcat == keyword_top1 ∧ ¬excluded`.
  - Pure specificity alone tops out at ~87% precision; agreement reaches ~95% (subcat) by also
    requiring the model to concur with the keyword's dominant subcategory.
  - "Agreement AND confidence" was rejected as **strictly dominated**: confidence is uncorrelated
    with correctness, so ANDing it adds zero precision, costs coverage at random, and leaves a
    "knob wired to nothing." Confidence dropped from the gate entirely.
- **Signal as a PURE function, not a `Classify` return-shape change** (per advisor). The plan said
  thread the signal out of `Classify`; the advisor showed that's fragile — `selectExamples` enters
  the embedding fallback on `len(examples)==0`, which is true both for "no keyword" *and* "high-spec
  keyword whose subcat had no pool examples." Tagging the fallback path a "miss" would wrongly route
  a high-specificity item to review. Fix: `MatchStrength(item, keywords) MatchSignal` is a pure
  function of item + keyword index — model- and pool-independent — computed at the gate sites. This
  also deletes the 11-call-site `Classify` signature churn.
- **Deterministic tiebreak + ambiguity → review** (per advisor). `sortSubcategoriesByScore` seeded
  from a Go map with a strict `>` comparator, so `sorted[0]` flipped run-to-run on ties — unacceptable
  for a gate. Added a name-secondary sort; a top-score tie across distinct subcategories sets
  `Ambiguous`, which fails the gate (a genuinely ambiguous keyword shouldn't auto-insert).
- **`--threshold` deprecated, not removed** (user-selected). `batch-auto`'s flag is kept accepted
  (cobra `MarkDeprecated`) so scripts/fixtures don't error, but it no longer gates.
- **Collision caveat recorded, not fixed** (user-selected). Agreement is verified at subcategory
  level; the appended row carries the model's full path, so the 5 cross-type collision leaves
  (Estacionamento/Dentista/Orion/Lilly/Ambos) can be subcat-right yet type-wrong and auto-append.
  Documented in `decision.go`; revisit if it bites.

## 3. Implementation

| File | Change |
|------|--------|
| `internal/classifier/examples.go` | `MatchSignal` type; `MatchStrength()`; deterministic tiebreak in `sortSubcategoriesByScore` |
| `internal/classifier/decision.go` | New `IsAutoInsertable(result, MatchSignal, excluded)`; `IsExcluded()` helper; dropped `DefaultHighConfidenceThreshold`; collision caveat |
| `cmd/.../auto.go` | Load keyword index + compute signal once; gate both JSON and interactive paths; JSON "excluded" branch now tests real exclusion; `gateReviewReason()` names the failure cause; removed `highConfidenceThreshold` const |
| `cmd/.../batch_auto.go` | `classifyLines` loads the index once, per-row signal; `--threshold` deprecated |
| `README.md` | Documents the agreement gate + deprecated flag |

Local model (`my-go-qcoder`) generated the net-new Go (`MatchStrength`, `IsAutoInsertable`,
`classifyLines`); verdicts 1/2/1. Unit-test tables authored directly (oracle integrity).

## 4. Findings

### 4.1 Replay validation — shipped predicate == measured predicate
A `//go:build replay` test (`gate_validation_test.go`) recomputes the shipped gate over the frozen
649-replay predictions (`model.jsonl`), no re-inference:

| Metric | Shipped | FINDINGS-model.md |
|--------|---------|-------------------|
| `top_score` shipped-vs-stored mismatches | **0 / 649** | — |
| Pure `top_score ≥ 1.0` band coverage / full-path prec | 34.2% / 86.9% | 34.2% / 86.9% ✓ |
| Agreement gate coverage / subcat prec / full-path prec | 30.5% / **94.9%** / 93.9% | ~31% / ~95% ✓ |

0/649 `top_score` mismatches proves `MatchStrength` reproduces the measured signal byte-for-byte —
the ~95% isn't from a parallel analysis path, it's what `decision.go` actually does. Full-path
precision on the gated set (93.9%) exceeds the pure band (86.9%): the agreement check filters exactly
the disagreement rows that dragged full-path down.

### 4.2 Precision caveat (carried into commit + docs)
The ~95% is a **same-sample RELATIVE** validation (shipped == measured). The 649 is a
confidence-selected review subset (FINDINGS caveat A), so **absolute production precision is
unmeasured**. The gate silently auto-appends the ~30% that pass at a precision that's
relatively-validated but absolutely-unknown — which is exactly why **unattended silent auto-insert
(WS-D) stays held**. T-32 unblocks WS-D scoping; it does not license it.

### 4.3 Coverage drop (intended)
Confidence ≥ 0.85 admitted ~97% of rows; the agreement gate admits ~31%. Far more rows now route to
review — the intended precision-over-coverage trade, not a regression.

### 4.4 Latent test bug the gate exposed (fixed)
`batch-auto`'s preflight (`verifyAppendable`) creates the log via `O_CREATE`, so a fully-gated-out
run leaves an **empty** log. `FeedbackMatchesExpected` early-returned when the actual JSONL was
empty/nil — silently PASSING against a non-empty expected fixture. The installment/rollover tests
(single "Uber Centro" row → gated out → empty log) were **false-passing**. Fixed by removing the
early-return so an empty actual fails the count assertion. This is a real defect in the verifier,
independent of the gate; the gate just made it observable.

### 4.5 Fixture coupling — "Uber Centro" no longer auto-appends
`uber` has specificity **0.8** and maps to two subcats (Viagens, Uber/Taxi) → `MatchStrength`
reports `Ambiguous` → gate fails. Fixtures that drove the auto-append path with "Uber Centro" were
swapped to **"Posto Ipiranga"** (`posto`/`ipiranga` spec 1.0 → Combustível, unambiguous; model
agrees; stays type Variáveis). Affected: batch-auto-{feedback,installments,rollover,typed},
auto-basic + `feedback_test.go` RunAuto. `add` tests keep "Uber Centro" (no gate). This is the tests
correctly tracking a behavior change, not a defect.

## 5. Tests

- **Unit (620 green):** rewrote `TestIsAutoInsertable` (agreement table), new `TestIsExcluded`, new
  `match_strength_test.go` (7 cases incl. two ambiguity paths + nil index).
- **Replay (new, tag-gated):** `gate_validation_test.go` — the §4.1 validation.
- **Verifier:** `test/verify/feedback.go` hardened (§4.4).
- **Acceptance (full suite green):** 4 affected batch tests fixed and passing *with* the hardened
  verifier (proving real appends); 26 deterministic tests green; 7 Ollama tests green
  (incl. `TestAutoJSON`, few-shot batch-auto, routing step 1). `TestAuto_FeedbackLoggedOnInsert`
  skips (workbook-gated — the interactive `auto` append-on-pass branch is therefore not *runtime*
  verified; low risk, signal-compute covered via JSON, gate via unit + batch-auto).

## 6. Documentation

Fixed every runtime-read ref that described the gate as confidence-based (an agent would otherwise
read them as truth): `.claude/index.md` `[ref:confidence-thresholds]` (+ its pointer) and the
duplicate in `data/classification/classification_algorithm.md` §5.2; `.memories/KNOWLEDGE.md`
(Decide step + gate design); `expense-reporter/.memories/KNOWLEDGE.md` (auto/batch-auto lines);
`internal/classifier/.memories/{QUICK,KNOWLEDGE}.md` (gate deep-ref, marked T-32-shipped with the
pure-function implementation note); `test/.memories/KNOWLEDGE.md` (threshold-0.0 strategy → gate-
passing items; canonical test item Uber Centro → Posto Ipiranga).

## 7. Follow-ups (not in this PR)

- **WS-D (T-09)** remains held — T-32 wires a protective gate but absolute precision is unmeasured;
  design gate-to-review, not silent insert.
- **T-33** ref-integrity: the `confidence-thresholds` / `classification-overview` /
  `training-data-schema` duplicate-block errors are pre-existing (a canonical home per key is still
  owed). This PR corrected the *content* of the confidence-thresholds duplicate; it did not dedupe.
- Interactive `auto` append-on-pass path has no *running* test (workbook-gated skip).
