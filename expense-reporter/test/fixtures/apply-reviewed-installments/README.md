# apply-reviewed-installments

Guards **T-21**: a reviewed installment expense must be recorded as N monthly rows in
`expenses_log.jsonl`, not one.

Before T-21 `apply.go` passed a literal `1` as `installmentCount` to
`appender.ExpandAndAppend`, because the count was discarded two hops upstream at
`review.ReadQueue` (`perInstallment, _, err := utils.ParseCurrencyWithInstallments`) and
never existed in the `reviewed.json` contract. Every sibling caller (`add`, `auto`,
`batch-auto`) passed the real count; `apply` was the only one that had nothing to pass.

## Why these values

**`value: 30` with `installments: 3`** — a total of `90,00/3`. Deliberately a value that
divides **exactly** in float64. `expect.ExpenseLogMatches` compares the `value` field, and
`99,90/3` is `33.300000000000004` while `100,00/3` is `33.333333333333336`; either would
put a drifting float in `expected-expenses_log.jsonl`. This is the float-drift trap
recorded in `test/.memories/KNOWLEDGE.md` "Traps That Masked Regressions". The
non-terminating case is covered by `apply-installment-resume-join`, which asserts ids
rather than values and so is immune to it.

**A full `DD/MM/YYYY` date** — this is a non-dry-run fixture that pins dated log lines, so
a bare `DD/MM` would take its year from the ladder's clock rung and the expectations would
expire at the next year boundary. The opposite rule governs join-id fixtures
(`apply-join-id`, `apply-stale-id`), which MUST stay bare; do not unify them.

**No `seed-classifications.jsonl`** — the row must be NEW so it takes the append path.
A seeded row is "found" and `apply` appends nothing at all, which would make the scenario
green for the wrong reason.

## The expectation

Three rows, one per installment: `expandEntries` suffixes **every** installment including
the first (`Posto Ipiranga (1/3)`, never the bare item) and advances the date a month at a
time. Note the id consequence — a series shares no id with the single unsuffixed row the
old code wrote, which is why the pre-T-21 output was invisible to `batch-auto --resume`.

Mutation-verified: restore the literal `1` at `apply.go`'s `ExpandAndAppend` call and this
fixture's scenario goes red (1 line where 3 are expected).
