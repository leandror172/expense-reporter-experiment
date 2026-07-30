## 2026-07-29 - Session 66: T-41 slice 2 — auto repointed at the parse boundary (PR #58), field sentinels, and the slice-3 design record

### Context

Opened as a "discuss next steps" session with PRs #56/#57 merged and master clean. A trace of who actually parses dates showed the entire T-42 close path (`batch-auto → review → apply → generate-workbook`) still on `utils.ParseDateFlexible` while everything session 65 hardened protected only `add`/`correct` — so the parse-boundary slices were confirmed as genuine pre-work rather than incidental cleanup, and slice 2 was implemented.

### What Was Done

- `feat(parse)`: `ErrInvalidDate`/`ErrInvalidValue` sentinels naming the rejected field, wrapped with TWO `%w` verbs so identifying the field never costs the reason. Chosen over a `FieldError` struct because nothing reads a field name programmatically.
- `refactor(cmd)`: `auto` repointed at the boundary — one `parse.Fields` call replacing a currency parse, a date parse and a manual re-format; `config.Load()` hoisted above the parse (aligning with `add`/`correct`); `parseOptions` extracted as the single owner of the year ladder's inputs and adopted by all three commands; `--year` flag + stale-year warning; `appendExpense` collapsed to take a `ParsedExpense`.
- `docs`: memories/index synced; two KNOWLEDGE statements that had become INSTRUCTIONS were corrected (Config Design's "still dead for auto", and Log-Append's "use explicit-year inputs BECAUSE bare dates take time.Now().Year()").
- `docs(plan)`: slice-3 design record as plan §11 (D1–D4), decided before implementing.
- Established a real characterization baseline first (66 pass / 0 skip / 110.6s) and matched it after (66 / 0 / 105.6s) with `-count=1`; 8 new deterministic unit tests; `gofmt`/`vet` clean including `-tags=replay` and `-tags=acceptance`.
- Verified verdict capture for both local-model runs; one was missing and was appended by hand with a dedupe guard.
- Corrected the `feedback_verdict_transcript_gap` memory twice — first to stop it over-generalizing, then again when direct evidence falsified the correction.

### Decisions Made

- **Slices 2→3→4 before the T-42 dogfood** (user call). The old parser is accidentally CORRECT for a July-2026 close of 2026 data (`time.Now().Year()` == the close year), so the hole only bites on backfill or a December-in-January close — but the close path is exactly the unguarded part.
- **`parseOptions` takes no seam.** `add`/`correct`/`auto` build byte-identical `Options`; `ref:patterns-code-extract-keep-divergence` says byte-identical copies need plain extraction, and rule 3 forbids the speculative parameter. `Options.Now` deliberately left zero.
- **No Ollama-gated test for `auto`'s year ladder.** Every path through `auto` calls `classifier.Classify` (`--json` included), so there is no deterministic CLI route; the ladder is already pinned inside `parse`, so `parseOptions`/`describeParseFailure` are pinned at the helper level per the `apply_dates_test.go` precedent.
- **Slice 3 D1–D4 locked** (plan §11): split stays in `cmd`; one counted warning in `printBatchSummary`; T-40 as a construction unit test at `Installments == 1`; and `classifiedRow`'s `value` column must map to `pe.RawValue`.
- **T-49 and T-40 left UNCHECKED despite their decisions being made.** An unchecked task with a locked decision costs a redundant glance at §11; a checked task whose implementation never happened costs the requirement.
- Deferred by user call: `auto --json` + T-47 refusal shape, and `auto`'s argument shape vs the vision — both to T-42 (now T-51/T-52).

### Next

- Merge PR #58, then implement T-41 slice 3 (`batch-auto`) against plan §11 — the decisions are made, so this is implementation.
- Start from the root cause: `parse3FieldLine` DISCARDS the installment count (`_`), which is why `appendOneRow` re-parses to recover it. Retaining it in the row struct removes the reason the downstream re-parses exist.
- Treat D4 as the hazard: verify the `classified.csv` `value` column against the fixtures, not by inspection.
- Then slice 4 (`apply`, retires `canonicalDate`) → T-21 → T-42.

### Gotchas

- **`./run-acceptance.sh` can report a cached pass that never ran your change** (T-53). `TestMain` builds the CLI at runtime, so edits under `cmd/` do not invalidate the test package's cache entry; the script has no `-count=1` and forwards only `-keep-on-failure`/`-keep-artifacts`. The first "green" this session was a stale cache hit.
- **A foreign resident Ollama model turned a ~110s `-full` run into a 3602s timeout.** `my-python-q25c14` held 9.2 GB of the 12 GB GPU; the 3600s ceiling did not protect anything, it just burned an hour before failing on a cause unrelated to the tests. Check `curl -s localhost:11434/api/ps` before trusting any `-full` timing.
- **`run_result` fixes verdict IDENTITY, not CAPTURE.** Two `run_result` calls, both with a correct `run_id` in the template; one verdict landed in `calls.jsonl`, one did not. The grep is the check, not a backstop.
- **`generate_code`'s router silently overrides CLAUDE.md's model tier** — it picks `my-go-q3` for Go unless `model=` is passed explicitly, where the tier list names `my-go-qcoder`.
- **Naming the right field while replacing the reason is its own bug class.** `describeParseFailure` reported a well-formed `15/04/2027` as a format error when the objection was the year. Caught by running the binary, not by any test — every unit test still passed.
- A stale memory can be factually correct and still produce a wrong conclusion, if it describes a symptom class broadly enough to swallow a mechanism it never examined.
