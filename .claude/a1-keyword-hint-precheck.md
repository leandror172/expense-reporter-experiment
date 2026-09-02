# A1 pre-check — should the review UI show the keyword's answer? (session 72, 2026-08-18)

**Verdict: GO.** The predicate fires on 14 of 65 rows (21.5%), is right 71.4% of the time,
and recovers 10 of the 25 corrections the first close needed. Only 2 fires are pure noise.

Probe: `expense-reporter/internal/classifier/replay_keyword_hint_test.go`
(`//go:build replay`). Data: `.claude/scratch/close-20260818-120248/reviewed.json`
(gitignored — real expense data); per-row output `reviewed-keyword-hint.jsonl` alongside it.

```
go test -tags=replay -count=1 -run TestReplayKeywordHint ./internal/classifier/ -v
```

## Why this probe existed

`.claude/t42-close-findings.md` [ref:close-accuracy-2026-01] reported that the keyword layer
already held the reviewer's answer in ~60% of corrections, 9 of them at specificity 1.00 — but
it computed that with a Python **re-implementation** of the tokenizer and flagged its own counts
as ±1–2, saying "a production replay would settle it." This is that replay: it drives the real
`classifier.MatchStrength` over the real export, so the numbers are the ones production emits.

## The predicate

The hint fires when the keyword signal **matched**, is **unambiguous**, scores at or above the
threshold, and its top subcategory **differs from the model's**. Agreement has nothing to show.

**Direction matters.** The predicate compares the keyword to the **model**, never to the
reviewer. A predicate consulting the reviewer's answer would need the very thing it is trying
to predict and could not run at review time — that mistake was made once here and rejected
(a local-model draft scored 0 for exactly it).

## Results (65 reviewed rows: 40 confirmed, 25 corrected)

| Measure | strict ≥ 1.00 | loose ≥ 0.70 |
|---|---|---|
| fired | 14 / 65 (21.5%) | 14 / 65 (21.5%) |
| keyword right | 10 (71.4% of fired) | 10 (71.4%) |
| fired on CORRECTED | 12 — recovered **10** | 12 — recovered 10 |
| fired on CONFIRMED (noise) | 2 (14.3% of fired) | 2 |
| **recall of corrections** | **10 / 25 (40.0%)** | 10 / 25 (40.0%) |

Against the s71 report: it estimated 9 corrections recoverable at specificity 1.00; production
says **10**. The ±1–2 caveat was honest. The report's headline 60% (15/25) is the *looser*
"keyword had the answer somewhere" measure, which includes ambiguous and sub-maximal matches
this predicate deliberately excludes.

## Finding: specificity is effectively binary, so the 0.70 threshold is inert

Both thresholds produce identical numbers — and that is a property of the data, not a broken
parameterization. Score distribution over the 51 matched rows: **31 rows score exactly 1.00**,
and **every one of the 14 rows scoring in [0.70, 1.00) is ambiguous**, so the `!Ambiguous`
clause already excludes them at either setting.

**Mutation-verified, because identical output is exactly how a dead parameter looks.** Dropping
the `!Ambiguous` clause makes the thresholds diverge (14 vs **17** fired), proving the threshold
is wired and used. Those 3 extra rows add **zero** recall — strictness costs nothing here.

*Design implication:* ship the strict predicate only. A configurable threshold would be a knob
with no reachable second setting.

## The 4 non-recoveries, and why they are cheap

| | model | keyword | reviewer chose |
|---|---|---|---|
| noise (confirmed) | Casa | Água | Casa |
| noise (confirmed) | Exames | Plano de saúde | Exames |
| wrong (corrected) | Farmácia | Dentista | Exames |
| wrong (corrected) | Dentista | Orion | <person E> |

None are absurd — each is a plausible alternative reading, so the cost of a bad hint is a
glance, not a misfile. The last is the sharpest: the keyword picked the *wrong pet*.

**This is why the hint must be advisory, not a default.** At 71.4% precision, auto-applying it
would introduce errors on ~3 of 14 rows; showing it alongside the model's answer cannot.

The 10 recoveries are dominated by the pet cluster the report already named — 7 `<person E>`/`Orion`
rows the model routed to `Dentista`, a human dental leaf.

## Caveats

- n = 65, one month, one reviewer. Directional, not a benchmark.
- Measured on rows that reached review. The 17 auto-inserted rows cannot fire the predicate by
  construction (the gate requires model == keyword), so the population is complete for this
  question but says nothing about auto-inserted accuracy — that is B1's job.
- `feature_dictionary_enhanced.json` is still the never-rebuilt 694-era artifact. 5.R6 would
  change these numbers upward: `exame`, `exames`, `veterinário` and `nefro` are absent from it.
