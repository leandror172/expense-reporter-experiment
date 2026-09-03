# T-75 step 1 — is the 2025 Telegram export a usable oracle?

**Session 75, 2026-09-02. Read-only measurement; nothing was written to either log.**
Reproduce every number in the step-1 sections with
`python3 .claude/scratch/t75_oracle_probe.py --source naive`; the probe's default since
step 4 (see the last section) reads the real converter instead. Both sets of numbers are
pinned by `.claude/scratch/t75_probe_characterize.py`.

**Verdict: the oracle is USABLE, with an asymmetric acceptance criterion.** And the
residue turned out to sit on the LOG's side, not the converter's — in every explained
case the Telegram message is the more complete record.

---

## Why this ran before any converter code

T-75 needs validation, and the obvious source is the export already on disk. But an
oracle is only evidence if it can fail, and this one had a visible problem before a
line was written: **352 messages against 499 log rows** for the same window. Exact
reconciliation was impossible by construction — the log side came from the
hand-maintained workbook via 5.R4 (alias table + dedup + merge), and holds recurring
bills nobody types into the chat group.

So the question was reframed, and that reframing is the reusable part:

> Not *"do the two sides reconcile"* but *"of the rows a naive conversion produces,
> how many find a `(date, value)` partner — and does every miss have a NAMED cause?"*

Order matters for honesty. Building the converter first and then testing the oracle
would be circular: every ambiguous residue row becomes an invitation to tune the
converter until the number looks good. Same reason s67 slice 4 wrote characterization
tests for `canonicalDate` and confirmed them green against UNMODIFIED code.

<!-- ref:t75-oracle -->
## The measurement

```
SIDE A  Telegram export ("Gastos", 393 messages, 2025-05 .. 2025-12)
        352 conforming messages -> 383 rows after installment expansion
SIDE B  live expense log, 2025-05-01 .. 2025-12-31 -> 499 rows

MATCH on the (date, value) multiset
        matched            346
        A unmatched         37     A coverage 90.3%
        B unmatched        153     B coverage 69.3%

PERTURBATION
        shift ONE value by 1 centavo   346 -> 345   OK
        shift ONE date by 1 day        346 -> 345   OK
```

**A coverage (90.3%) is the number that validates a capture tool.** B coverage is
expected to be low and does not impair the oracle: the workbook legitimately holds
rows that were never typed into the group.

**The match is a multiset on `(date, value)`, never a set and never keyed on item.**
Item strings on the log side are twice-transformed (5.R4 aliases, installment
suffixes) and diverge for reasons a converter cannot control. Multiset because
duplicate purchases are REAL — s74 found 8 genuine same-day duplicates that collide
by construction — and a set silently eats them. That is the s72 failure exactly: an
item-keyed join collapsed `San michel` x3 and reported 26 corrections against a
known 25.

**The perturbation check is what makes the 90.3% mean anything.** Corrupt one
converted row and the residue must move by exactly one. It does, in both directions.
Without it, "90.3% matched" is reassuring rather than informative — the s74 rule
applied verbatim (the workbook byte-comparison was first proven to move for a value
change and not for an id change).

## The residue — 37 of 37 explained, 0 unexplained

| Rows | Cause |
|-----:|-------|
| 24 | installment tails absent from the log (see below) |
| 9 | expanded outside the comparison window — a late-April purchase, and series running into 2026. Benign by construction |
| 3 | messages never entered in the workbook at all (`Associação Regera`, `Cartão <person C>`, `Compra martminas` — confirmed absent by item lookup, 0 rows each) |
| 1 | a genuine same-day duplicate (`Pão de queijo` R$14,00 x2) that the multiset correctly refused to collapse |
<!-- /ref:t75-oracle -->

## The finding: every installment series in the log is incomplete

All **14** installment series typed into the group during the window are recorded in
the log with **only their first installment**:

```
Ração Orion Petz          3x  PARTIAL (1 of 3)     Café lor x15               3x  PARTIAL (1 of 3)
Shopee <phone>     6x  PARTIAL (1 of 6)     Nail E ash catcher         3x  PARTIAL (1 of 3)
Troca tela g60            2x  PARTIAL (1 of 2)     growplant                  4x  PARTIAL (1 of 4)
Pet ração orion           3x  PARTIAL (1 of 3)     biocultivo                 4x  PARTIAL (1 of 4)
Petlove ração <person E>       3x  PARTIAL (1 of 3)     Nefrologista Orion <person B>  2x  PARTIAL (1 of 2)
Conserto steam deck       3x  PARTIAL (1 of 3)     perlita shopee 25L x2      3x  PARTIAL (1 of 3)
Ingressos Steven Wilson   3x  PARTIAL (1 of 3)     Compras amazon             3x  PARTIAL (0 of 3)
```

This is the defect T-21 fixed in the tool (s69), sitting at scale in the historical
data — the human doing by hand what the review route was silently doing until T-21.

**Magnitude.** R$ 7.309,43 of installment rows are absent from the log, against
R$ 119.050,71 for the 499 rows in the window (R$ 6.589,86 in-window, R$ 719,57 landing
in 2026). Two independent methods — multiset residue and item-based series matching —
agree **to the centavo, with zero disagreeing series.**

**But that is NOT the budget's net error, because 3 of the 14 series carry a
compensating lump.** Verified directly, and all three show the same signature — same
day, same item, one per-installment row AND one full-total row:

```
Ração Orion Petz            31/05/2025    161.67   +  31/05/2025    485.00
Shopee <phone>       24/06/2025    648.33   +  24/06/2025   3890.00
Nefrologista Orion <person B>   11/12/2025    150.00   +  11/12/2025    300.00
```

So those three are **over**-counted by one installment each (R$ 960,00 total), and
**11 of 14 are genuinely short by R$ 3.594,44**. Net: the 2025 window understates by
roughly **R$ 2.634,44**.

⚠️ **The lump check took three attempts and the first two were wrong** — recorded
because the failure mode generalises. (1) An exact-cents lookup for `per × N` missed
Xiaomi entirely: 3890 ÷ 6 = 648,3333, so `per × N` recovers R$ 3.889,98 and misses a
real R$ 3.890,00 lump by two centavos. Installment arithmetic does not round-trip;
compare with a tolerance of ±N centavos. (2) Adding tolerance but matching on VALUE
alone produced 4 hits, two of them coincidental collisions — `Troca tela g60`'s
"lump" was a row named *Consulta <person F>*, and a R$300,00 one was *Diarista <person D>*. In a
corpus this size, value-only matching manufactures false positives. Only requiring
item agreement AND printing the actual rows settled it.

**USER RULING (session 75): if an expense was reported as an installment, it should
have been an installment.** So these are omissions, not a bookkeeping convention, and
the 2025 budget understates by roughly the figure above. Conventions in the workbook
are genuinely mixed: `Shopee <phone>` carries BOTH a R$648,33 row and a
R$3.890,00 lump on 24/06/2025 (648,33 x 6 = 3.890), so that one series is arguably
over-counted while the other thirteen are short. **Filed as its own item — do not fold
a data backfill into T-75.**

## How phase two must use this oracle — asymmetrically

The log is not ground truth. It is a second, independently lossy record. So it cannot
adjudicate a mismatch, and the acceptance criterion is NOT a match rate:

- a converted row that finds a partner is **confirmed**;
- a converted row that does not must be **explainable by a named cause**;
- **the count of unexplained rows must be zero.**

Reading "match rate == 100%" as the target would be actively harmful here: it would
push the converter toward reproducing the log's own omissions.

## Method notes worth keeping

- **A confounded cut looked fine and told me nothing.** The first attempt to classify
  the installment residue asked "does this value appear elsewhere in the log?" Within
  a series every installment shares a value, so the answer is trivially yes. It
  produced a clean-looking 20/9/6/2 breakdown that was pure noise. Matching on ITEM
  settled it. **A breakdown that partitions cleanly is not evidence that the
  partition means anything.**
- **A R$719,57 discrepancy between two magnitude figures was chased, not rounded
  away** (the s72 rule: treat a small unexplained gap as a defect). It was purely the
  comparison window; reconciling it also produced the independent cross-check above.
- **The probe ports `pkg/utils/currency.go` rather than shelling out to the binary.**
  Any divergence from the Go parser shows up as extra residue — the safe direction,
  since it inflates the miss count rather than hiding one.

## What was NOT established

- Whether the same installment gap extends to 2022–2024. Untested, though 5.R4
  extracted those years by the same route, so it is likely.
- Whether the tail installments were actually paid. That is the user's knowledge and
  it gates any backfill.
- Anything about **classification**. This validates capture fidelity only —
  category/subcategory on the log side came from the workbook, and the converter does
  not assign them (`batch-auto` does, downstream).
- Whether the 2026 messages parse the same way. **The export on disk is 2025-05..12;
  the ~7 unprocessed months are 2026 and need a fresh hand-export.** The typing
  vocabulary is evolving — `89,90*3` appears in 2025-11, and T-64 only taught the
  parser `x4` in s73 — so the 2026 export must be re-bucketed before the converter is
  trusted on it.

---

## Step 4 (session 75, 2026-09-03) — the probe now reads the REAL converter

Characterize-first, literally: `t75_probe_characterize.py` pinned the eleven numbers above
black-box (the probe run as a subprocess, its report parsed) and went green on the
unmodified probe — and red when one pinned number was flipped — before a line of the probe
changed. Then the swap: `--source converter` (the default) runs `expense-reporter
telegram-import` into a temp dir and reads its month files, expanding each line's
installments with the Python port; `--source naive` is the step-1 conversion, kept so the
numbers above stay reproducible. The exit code is now the gate.

```
SIDE A  converter output   358 lines (352 as typed + 6 repaired) -> 389 rows
                           14 rejected (telegram-rejects.csv)
MATCH   matched 346 | A unmatched 43 | B unmatched 153
RESIDUE 24 installment tails absent | 9 outside the window | 7 never entered (item
        unknown to the log) | 2 item recurs, this date/value absent | 1 count mismatch
        0 SUSPECT
GATE    PASS — unexplained == 0; perturbations 346 -> 345 in both directions
```

**Finding: not one of the six repaired messages is in the log.** Matched stayed at 346
while side A grew by exactly the six lines D4 recovered, and the item lookup says why:

| date | value | item | log |
|------|------:|------|-----|
| 15/05/2025 | 216,00 | `Gás 2 bujões` | item unknown |
| 19/05/2025 | 50,00 | `Almoço <redacted>` | item recurs (`Almoço pizza na roça` 03/05, `Delivery almoço na roça` 03/06), this instance absent |
| 30/05/2025 | 78,00 | `Almoço <redacted> 3 pessoas` | item unknown |
| 25/07/2025 | 90,00 | `Tela celular` | item unknown |
| 21/08/2025 | 1.200,00 | `Dentista <person A> canal` | item unknown |
| 28/08/2025 | 194,96 | `Mensalidade agosto faculdade Ana` | item unknown |

**R$ 1.828,96 typed into the group in 2025 that the workbook never received — and the
common factor is the typo.** Whatever entered the other 352 (a hand, an earlier import)
skipped exactly the messages that did not parse. That is the converter's case in one
table: the repairs are not cosmetic, they are the lines that were being lost. Not a T-75
concern to backfill; it is the second omission list beside T-78's installment tails.

**"Never entered" is now verified, not assumed.** The step-1 residue classification had a
catch-all: every leftover row that was not an installment tail, not outside the window and
not a same-day duplicate was labelled "never entered", so `unexplained == 0` could not
fail. `leftover_cause` now consults the log for the same ITEM (every token of the export
item inside one log item — "all but one token" was tried first and called `Cartão <person C>`
present because some log item contains `anita`):

- **unknown** — no log item carries the export item: never entered;
- **omission** — the item recurs in the log but nothing sits near this date/value: the
  human skipped this one, a named cause the lossy log allows;
- **SUSPECT** — the item is in the log with the same value within 7 days, or on the same
  date with another value: a converter error or a human edit. Fails the gate.

Under that split the step-1 "3 never entered" reads 2 unknown (`Compra martminas`,
`Associação Regera`) + 1 omission (`Cartão <person C>` 18/08 R$ 1.034,39 — the item recurs as
`<person C> boleto cartão`, so that month's bill was skipped, not an unknown item). Every other
number above is unchanged, and SUSPECT is 0 in both modes — which is the sentence step 4
existed to be able to write.
