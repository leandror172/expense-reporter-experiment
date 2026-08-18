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
(`expenses-2026-01.csv`, 4 of 69 rows). They are **input errors, not missing parser
features** — `99,90/3` is the only accepted installment form, so `- 1/4` and `4x` are
typos.

> **SUPERSEDED IN PART (s71, T-64).** The multiplier form is a real notation the user had
> planned but never written down: line 19's true shape is `Anita Elô ADM;09/01;405,25 x4`.
> It is NOT the same as `total/N` — `x4` means the number is PER-INSTALLMENT and must be
> MULTIPLIED, where `405,25/4` divides. Same digits, 4× apart (101,31/month vs 405,25/month). `- 1/4` remains a typo.
> The rows stay in this fixture as rejects until T-64 lands; when it does, they must move
> out or be re-spelled, or this fixture will assert that a supported notation is invalid.

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
