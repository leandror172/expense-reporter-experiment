# apply-join-id fixture (T-35)

One NEW confirmed row, deliberately dated `15/04` — the short `DD/MM` form.

That short form is the whole point: `apply` normalizes the date to `DD/MM/YYYY` (via
`--year`) for `expenses_log.jsonl` but wrote the RAW string to `classifications.jsonl`,
so the shared `GenerateID` hash — the only join key between the two logs — diverged.
A fixture dated `15/04/2026` cannot catch this: raw and normalized are then identical.

No `seed-classifications.jsonl`: the row must be NEW so it takes the append path
(`appendNewRows`) and lands exactly once in each log, making the join unambiguous.
Single, non-installment expense — installments rewrite the item to `X (i/N)` and shift
dates, so their ids legitimately differ across the two files.

**Slice 4 update (T-41):** the `id` in `reviewed.json` is now advisory —
`entriesWithCanonicalDates` recomputes it from the canonical date, because a function
that rewrites the date must rewrite what was hashed from it. The value here was
refreshed to the canonical id anyway: a fixture carrying an id that does not describe
its own row is the trap this slice exists to remove, even where nothing reads it.

The date stays BARE. That is unchanged and still load-bearing — a full date makes raw
and normalized identical and silently disables the test rather than breaking it.
