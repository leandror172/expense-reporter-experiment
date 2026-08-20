# batch-auto-failed-rows

Every input line is unparseable **on purpose**. Two consequences, both deliberate:

1. **No row reaches the classifier**, so this scenario is deterministic and runs in the
   `-short` group. The precedent is `batch-auto-resume-all-seeded`: a step that precedes
   classification can be tested without Ollama. Here the step is the parse itself.
2. **The file cannot prove that a *parseable* row stays OUT of `failed.csv`** — that needs
   a row that classifies. That claim is pinned instead by the pure-function unit test
   `TestRejectedRows_KeepsOnlyRowsThatFailedToParse`. Do not "improve" this fixture by
   adding a good row: it would drag Ollama in and move the scenario out of the fast group.

The first four lines are the real rejects from the first monthly close on 2026 data
(`expenses-2026-01.csv`, 4 of 69 rows). Each is rejected for a reason that has nothing to
do with the installment notation: a semicolon inside the item, transposed date and value,
a typo'd day, and a missing date.

> **T-64 LANDED (s73), and this fixture was NOT affected — which is worth stating, because
> the s71 note here predicted the opposite.** That note warned these rows would "have to
> move out or be re-spelled" once the multiplier notation shipped, or the fixture would
> assert that a supported notation is invalid. It does not, and the reason is worth
> knowing before anyone edits this file:
>
> - `Anita;Elô ADM;09/01;405,25 - 1/4` has **four fields** (the item contains a `;`) and
>   uses `- 1/4`, which remains a typo. Rejected on both counts, unchanged.
> - `Anita compra chocolate Ruby 299,00 e cacau 49,90;646,25 4x` uses the now-supported
>   `4x`, but has **only two fields** — no date at all. It is rejected on field count
>   **before the value is ever parsed**, which is exactly what the error quoted further
>   down this file shows: `got: 646,25 4x   # expected 3 fields (item`.
>
> So nothing here ever pinned "`x4` is invalid", and there was no guard to unwind.
> Verified by running the scenario after T-64, not by reading. **The corollary is that
> this fixture cannot tell you whether the multiplier parses** — that is pinned by the
> unit table in `pkg/utils/currency_test.go` and by the classified.csv → ReadQueue seam
> tests. Do not add a well-formed `x4` row here expecting it to be rejected: it would
> parse, reach the classifier, and drag Ollama into the fast group.

**The notation itself, for reference.** `405,25/4` states the TOTAL and divides;
`405,25 x4` states the PER-INSTALLMENT amount and multiplies. Same digits, 4× apart
(101,31/month vs 405,25/month). `- 1/4` is not either of them and remains a typo.

## Known limitation: a row with fewer than 3 fields absorbs its reason once

`stripTrailingComment` only treats `#` as a comment when the three data fields are already
complete before it (see `parse3FieldLine`). A row with fewer semicolons than that — here
`Anita compra chocolate Ruby 299,00 e cacau 49,90;646,25 4x` — therefore cannot have its
previous reason stripped, and re-running an **unrepaired** file absorbs that reason into
the line **once**.

Measured across three passes: 106 → 204 → 204 chars. It converges rather than growing,
because the absorbed comment itself contains `;`, which completes the field count and lets
the strip engage from the next pass on. The three well-formed-shape rows never grew at all.

This is deliberate. The alternative — stripping at any whitespace-preceded `#` regardless
of field count — would let a genuine 4-field line (`item;date;value;subcategory`) be
silently truncated to three, which is the exact silent-acceptance failure T-54 and S2 were.
A noisy line on a row too broken to parse is the cheaper cost.

## The fifth row is not from the close

`  Anita cafe ;99/99;1,00` was added deliberately. `batch.CSVReader` right-trims only and
**preserves leading whitespace**, so that row reaches `failed.csv` verbatim. It guards the
read-back in `expect.FailedRowsCarryTheirReason`: if that helper left-trims, `TrimPrefix`
returns the whole string SILENTLY and the assertion blames a missing reason rather than the
prefix mismatch that actually occurred. Mutation-verified.

## A reason can name a stale comment as if it were data

For a row with fewer than 3 fields, `parse3FieldLine` cannot strip a previous run's comment,
so on a re-run the parser reports the failure against text that still contains it — e.g.
`got: 646,25 4x   # expected 3 fields (item`. The message looks like the parser is confused;
it is not. The row is simply too broken for the format to tell data from annotation, and it
resolves as soon as the line has three fields.
