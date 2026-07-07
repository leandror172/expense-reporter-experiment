# 649-Replay — Model Half (T-23 gate) — Findings

*Session 52 (2026-07-06). In-process `classifier.Classify` replay of the 649 human-verified
real labels, exact `auto` config (FeedbackPath + T-22 descriptions), no-think,
feedback-LOO-by-item, training kept + stratified clean/leaky. Harness:
`replay_model_test.go` (`//go:build replay`). Analysis: `analyze_model.py`. Raw: `model.jsonl`.*

Run: `go test -tags=replay -run TestReplayModel649 ./internal/classifier/ -timeout 60m`
(resumable, ~15 min). Then `python3 analyze_model.py`.

## Faithfulness
- Config replicates `auto`/`batch-auto` (the real gate path): FeedbackPath set, type descriptions applied, my-classifier-q3, TopN=3, **no-think** (production default is think-on → ~+3pp level shift; relationship unchanged).
- **Feedback LOO by item** (all same-item copies removed — 649 rows = only 260 unique items, heavy recurrence). Training KEPT (a recurring item legitimately has prior instances in production); `in_training` overlap (287/649 rows) reported as the **leaky** stratum, NOT excluded.
- Per-row denominator (stream-weighted toward recurring items).

## Overall accuracy (full-path, per-row)
| stratum | full-path | type | category | subcat |
|---|---|---|---|---|
| all (n=649) | 52.5% | 77.5% | 74.3% | 53.5% |
| **clean** (not in training, n=362) | **44.2%** | 70.4% | 67.7% | 45.3% |
| leaky (in training, n=287) | 63.1% | 86.4% | 82.6% | 63.8% |

Clean 44.2% ≈ T-14's synthetic "clean 49.1%" (no-think, real 2025 data) → consistent. The 19pp
clean/leaky gap = the recurrence/training-presence effect, quantified on real data.

## GATE FINDING (the T-23 payload)

**Confidence is dead — confirmed on real data.** 52.5% → 56.3% across the whole range (conf≥0.95
still 56.3%; conf≥0.85 admits 97.4% of rows at 53.3%). Today's `IsAutoInsertable` gate
(confidence≥0.85) auto-inserts ~everything at a coin flip.

**Specificity (`top_score`, external validator E1) works — monotone, and crushes confidence:**

| top_score ≥ | ALL: coverage / precision | CLEAN: coverage / precision |
|---|---|---|
| 0.0 | 100% / 52.5% | 100% / 44.2% |
| 0.7 | 54.7% / 75.2% | 39.8% / 78.5% |
| 0.85 | 48.7% / 77.2% | 35.9% / 80.8% |
| **1.0** | **34.2% / 86.9%** | **30.4% / 82.7%** |

The signal moves precision 52→87 (all) / 44→83 (clean); confidence moves it 52→56. **Specificity is a
real model-level gate signal; confidence is not.**

**A pure specificity gate does not reach the ~95% unattended bar.** Best band (top_score≥1.0) = 86.9%
all / 82.7% clean full-path → ~1 in 7 wrong.

## RECONCILIATION (advisor stress-test, session 52) — 3 corrections to the above

**(A) Representativeness — the 649 is a confidence-selected review SUBSET, so ABSOLUTE precision levels
are UNMEASURABLE.** `expenses_log.jsonl` has 725 unique expenses; only 347 were ever reviewed (in
`classifications.jsonl`) — **378 (52%) bypassed review entirely** (auto-inserted/added, NO ground-truth
label). Historical selection was on CONFIDENCE (low→review, high→auto), and confidence is ⊥correctness
(proven above), so the subset is biased on an axis orthogonal to difficulty → the correctness-axis
representativeness is INDETERMINATE, and the 378 bypassed items cannot be scored. **Therefore: do NOT
claim any absolute precision (86.9%, the 95%-bar comparison) as production-representative.** The
RELATIVE findings (confidence dead; specificity monotone) are same-sample and SURVIVE.

**(B) Keyword BEATS the model at the top band (apples-to-apples subcat).** The earlier "86.9 vs 90.5"
was full-path-vs-subcat — wrong comparison. Joined by id on the 222 top-band rows, subcat-vs-subcat:
keyword_top1 **91.0%** vs model **87.8%**. A dumb lexical lookup out-predicts the 8B model exactly where
specificity is maximal. → a keyword-first (model-fallback) hybrid is worth considering for the gate.

**(C) A model⊕keyword AGREEMENT gate reaches ~95% subcat at spec=1.0.** Where model prediction ==
keyword top1 (201 of 222 top-band rows, 90.5% agree), **subcat accuracy = 95.0%** — clears the bar. At
top_score≥0.85 agreement only reaches 86.4% → need BOTH high specificity AND agreement. CAVEATS: (i)
few-shot injects the keyword's examples, so model/keyword are NOT independent (E3 non-independence — the
95% is partly a nudge, likely inflated); (ii) subcat-level, not full-path; (iii) still on the
unrepresentative subset. Promising but NOT a settled auto-insert license.

## Recurrence framing (driver_freq vs full-path)
| driver_freq | rows | full-path |
|---|---|---|
| 0 (no keyword match) | 160 | **18.8%** |
| 2–4 | 152 | 67.1% |
| 5–9 | 155 | 68.4% |
| 10–24 | 138 | 58.7% |
| 25+ | 44 | 50.0% |

Inverted-U: no-match collapses (18.8%), mid-freq is the sweet spot, high-freq generic tokens decline.
Reconfirms **specificity > frequency** as the axis, and the **no-keyword-match 24.7% pool is the accuracy
sinkhole** — the 5.R2 embeddings target.

## Decisions — SHIP (bias-robust, relative findings)
1. **Confidence is dead as a gate** — 52→56% flat, same-sample robust. Definitive on real data.
2. **Specificity (`top_score`) is a real, monotone discriminator that should REPLACE confidence as the
   gate signal** — 52→87% (all) / 44→83% (clean). Same-sample robust (relative to confidence).
3. **The strongest gate candidate is an AGREEMENT gate** (spec=1.0 ∧ model⊕keyword concur → 95.0% subcat,
   201 rows) — carry into gate design, caveated (non-independence, subcat-only, subset).

## Decisions — HOLD (depend on absolute precision level; sample not representative)
4. **"No unattended auto-insert is justified"** — HELD. The 86.9% ceiling is on a confidence-selected
   subset; true full-stream precision is unmeasurable here. Neither "silent insert viable" nor "not
   viable" is provable from this data. Aligns with review-first vision either way, but not a data verdict.
5. **"5.R2 embeddings is THE accuracy lever"** — SOFTENED to: **5.R1 TF-IDF is ruled out** (zero-lexical-
   overlap misses; solid), but 5.R2 is an UNPROVEN hypothesis — "160/160 misses have a same-subcat
   pool-mate" ≠ those pool-mates are embedding-retrievable (esp. grab-bag subcats). **Precondition test
   before roadmap re-order:** run an embedding model over the 160 miss items + pool-mates, measure NN
   subcat-retrieval accuracy. Weak NN → novel items are just hard → permanent review, not 5.R2.
6. **WS-D silent-insert scoping** — HELD pending #4.

## think-on top-band re-run — DONE (settles #4): SHAPE HOLDS, gate can run no-think
Re-ran the top band (top_score≥0.85, 316 rows) think-on; joined to no-think by id (174 unique items).
`compare_think.py`. Result (`model-think.jsonl`):

| band | n | no-think | think-on | Δ |
|---|---|---|---|---|
| [0.85, 1.0) | 47 | 55.3% | 53.2% | −2.1 |
| [1.0] | 127 | 89.0% | 86.6% | −2.4 |

**Lift spread = 0.2pp → UNIFORM: the specificity discriminator's SHAPE is robust to think mode
(does NOT flatten).** And the level delta is NEGATIVE (−2.3pp) — the assumed "+3pp think-on" does NOT
apply on the high-specificity band; thinking adds noise on easy recurring items. **→ the gate can run
no-think (10× faster) with no loss of discrimination.** Feeds T-24 (`--think` default → no-think for the
recurring/high-spec path). Full-path curve conclusions stand as measured (no upward think correction).

## Remaining cheap follow-up
- **NN-retrieval precondition** (settles #5, the 5.R2 go/no-go): embed the 160 no-keyword-match miss items
  + their pool-mates, measure nearest-neighbor subcat hit-rate. Weak → novel items are permanent-review,
  not 5.R2. **This is the settled NEXT step** before any 5.R2 build.
- **E2 value-range NEGATIVE veto** (5.R3): orthogonal safety flag on top of the agreement gate.
