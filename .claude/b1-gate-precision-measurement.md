# Absolute agreement-gate precision — MEASURED (session 72, 2026-08-18)

**94.1%.** The number T-32 declared unmeasurable and WS-D has been blocked on since
session 52 was already sitting in the session-71 close data. No new run was needed.

| population | corrected | n | error | precision |
|---|---|---|---|---|
| **gate-PASSING** (auto-inserted) | 1 | 17 | 5.9% | **94.1%** |
| **gate-REFUSED** (sent to review) | 24 | 48 | 50.0% | 50.0% |

The gate separates a 5.9%-error population from a 50%-error one — an **8.5× discrimination**
on a full, unselected month. Source: `.claude/scratch/close-20260818-120248/`.

## Why this was measurable now and not in the 649-replay

T-32's "absolute precision is unmeasured" rested on the 649 being a **confidence-selected
review subset**: 378 of 725 rows bypassed review *unlabeled*, so no ground truth existed for
the rows the gate accepted.

**That condition does not hold for a close.** `close-cycle.sh:220` runs
`review "$run_dir/classified.csv"` — the **full** classified file, not `review.csv`. So every
gate-passing row appears in the review queue (flagged as already inserted) and carries a
recorded reviewer action. 65 of 69 rows were labeled; the 4 missing are the unparsed rows.

## Correction: the rows were NOT unseen

`.claude/t42-close-findings.md` states the gate auto-appended "17 of 69 rows (25%) in this
close **without any human seeing them**." That is **false**, and the report contradicts it two
paragraphs earlier: `4x álcool 70` was auto-inserted *and corrected during review*. A row
cannot be both unseen and corrected by the reviewer.

The defect is not that nobody looked. It is that **someone looked, fixed it, and the fix could
not reach the expense log** — which is T-65, and is strictly worse than an unseen row, because
the human already spent the effort.

This matters because "nobody sees 25% of rows" was the load-bearing real-data argument for
WS-D / gate-to-review. It does not survive contact with the data.

## Method

Positional join of `classified.csv` (69 rows) against `reviewed.json` (65 entries), which
preserves `ReadQueue`'s order over that same file. **Not keyed by item** — the month contains
duplicate item names (`San michel` ×3, `Diarista <person D>` ×2, `Maeda` ×2), and an item-keyed
join silently collapsed them, producing 26 corrections against a known total of 25. The
mismatch is what exposed it.

Verified before use: all 65 pairs agree on **both** item and date, and the corrected counts
sum to the independently known 25.

## Caveats — read before quoting 94.1%

- **n = 17 gate-passing rows**, one month, one reviewer. A single additional error would drop
  it to 88.2%. Directional, not a benchmark.
- **Confirms on auto-inserted rows are weak-positive.** `review.go:93` excludes them from the
  `needsReview` count and the page shows them as already handled, so a reviewer may have
  skimmed. The 1 correction is a hard negative; the 16 confirms are softer. Read 94.1% as an
  **upper** bound.
- The 50% figure for gate-refused rows is the complement of a *selected* population and is not
  a model-accuracy number — the gate refuses precisely where it lacks confirmation.
- The headline "38% correction rate" in the T-42 report conflates both populations. Decomposed,
  it is 5.9% / 50.0%, which is a far more useful picture and strongly validates the gate.

## Consequences

- **T-32 is confirmed on real, unselected data**, not just relatively.
- **B1 (gate-to-review) loses most of its rationale.** Its two justifications were eliminating
  unrepairable rows and obtaining this measurement. The measurement is done, for free; the
  rows are seen; and the gate is 94% precise. Routing 17 rows/month back to a human buys little.
- **T-65 rises to the top.** The failure is persistence, not review coverage — and it is not
  confined to auto-inserted rows: any reviewer who changes their mind later hits the same wall.
- **A1 is unaffected** and stays GO — it targets the gate-REFUSED population, where the error
  rate is 50%.
