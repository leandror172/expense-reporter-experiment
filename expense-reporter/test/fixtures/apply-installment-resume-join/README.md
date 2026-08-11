# apply-installment-resume-join

Guards the **T-21 resume repair** and, with it, the one arithmetic path that had never
round-tripped this seam: a division that does not terminate.

## What it proves that `apply-reviewed-installments` cannot

`apply-reviewed-installments` asserts apply writes three rows. It says nothing about
*which ids* those rows carry — and the ids are what the rest of the system joins on.

The obvious way to check them would be to compare apply's output against
`appender.PredictEntryIDs`. That test would be worthless: `ExpandAndAppend` and
`PredictEntryIDs` both call `expandEntries`, so they agree **by construction**. It is the
same circularity that got a slice-3 test rejected in session 66 — asserting the parser
against a second call to itself (`internal/parse/.memories/QUICK.md`).

So this scenario asserts a *behavior* instead, across two genuinely independent paths:

| | derives the per-installment value from |
|---|---|
| `apply` | `reviewed.json`'s already-typed JSON float (`33.333333333333336`) |
| `batch-auto --resume` | the raw `100,00/3` token, through `internal/parse` |

Given the row was applied, `batch-auto --resume` over the same expense must report it as
already logged. That can only happen if both paths produce the same three suffixed ids —
which requires the count, the per-installment value **and** the monthly date walk to agree
end to end. `GenerateID` hashes `fmt.Sprintf("%.2f", value)`, so the two floats need only
agree to the cent; this fixture is what proves they actually do.

## Why it is Ollama-free (when green)

A full `--resume` match is consumed **before** the model call (`classifyResumeDecision`
runs ahead of classification), so a green run never reaches Ollama.

A RED run does: with the count dropped, the row is not recognised, and batch-auto proceeds
to classify it. The failure is therefore Ollama-shaped rather than a clean assertion
mismatch — expected, and the same property the other `batch-auto-resume-*` fixtures have.

## Do not "simplify"

- **Keep `100,00/3`.** A clean-dividing value would make this a duplicate of the existing
  `batch-auto-resume-all-seeded` coverage and drop the float claim entirely.
- **Keep the explicit years.** The bare-date + `--year` wiring claim belongs to
  `batch-auto-resume-year-flag`; mixing it in here would give a failure two possible
  causes.
- **Do not compare the log's `value` field** in this scenario. `33.333333333333336` is
  exactly the drifting float that `test/.memories/KNOWLEDGE.md` warns against pinning.
  The value claim lives in `apply-reviewed-installments`, which uses `90,00/3` → `30`.
