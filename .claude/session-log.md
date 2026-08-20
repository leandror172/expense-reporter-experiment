# Session Log — Expense Reporter

**Current Session:** 2026-08-20 — Session 73: a second installment notation that means the opposite of the first (T-64), and the thousands value that was killing review (T-72)
**Current Layer:** Close-cycle follow-up — T-64 shipped (PR #67, open) along with T-72 found while building it. T-65 supersede-by-append is next, DESIGN ONLY by the user's call; then 5.R6, which now has a measured payoff and a ready-made before/after harness
Most recent entry first. Run `.claude/tools/rotate-session-log.sh` when this grows beyond ~3 sessions.

---
## 2026-08-20 - Session 73: a second installment notation that means the opposite of the first (T-64), and the thousands value that was killing review (T-72)

### Context

Opened on a clean master with PR #66 (A1) merged, and began as a discussion of what to do next rather than a coding task. Four candidates were laid out with a recommendation; the user picked T-64 over the previously-planned T-65 design doc. Orientation before any code found more than the task file predicted, which changed the shape of the work twice — once from the user's own reframe of the notation, and once from a defect discovered in the same seam.

### What Was Done

- **T-64 shipped** — `405,25 x4` (multiply, per-installment) alongside `405,25/4` (divide, total). Both token orders, case-insensitive, plus the retail `4x405,25`. PR #67 opened against master.
- **T-72 found and fixed** — `review.ReadQueue` hard-errored on any BR-thousands value, killing the whole review step over one row. Found by feeding the real writer's bytes to the real reader, not by reading code.
- Added `parse.Value`, the value-only door onto the T-41 boundary and mirror of `parse.Date`; `Fields` delegates to it so normalization has exactly one call site, and `review.ReadQueue` routes through it instead of calling `utils` directly.
- Verified: `go test ./...` 0, vet + gofmt clean, acceptance `-short` green, **`-full` 72 PASS / 0 FAIL / 0 SKIP**; currency table at **33 subtests** with the pre-existing 13 unmodified.
- Corrected two false claims rather than leaving them standing: the `batch-auto-failed-rows` README's prediction that its `x`-rows would need to move out (they are rejected on field count, before the value is parsed), and the T-64 commit message's "two live callers" (there are three).
- Filed T-73 (`classify` bypasses the boundary; also names the third caller) and T-74 (dot-only `1.234` resolves 1000x wrong — pre-existing).

### Decisions Made

- **The mode switch is the PRESENCE of `x`/`X`, not a positional grammar** — the user's reframe, and better than the whitespace rule that was about to ship. It splits "is this a multiplier?" from "where does the value end?", and the second question is then answered by trying three readings and requiring EXACTLY ONE to hold. Two holding is an ambiguity and an ERROR, never a guess.
- **Ambiguity is detected, not resolved by fiat.** A grammar answers "what does this mean?"; try-all-readings answers "is there more than one thing this could mean?" — the question the hazard actually turns on, since the two readings are 4x apart. `646,254x` falls out as an error without a special rule.
- **Rejecting is the cheap side here**, which is what justifies the strictness: T-63 returns the row in `failed.csv` with its reason inline, so a rejection costs one edit while a guess writes a silently 4x-wrong budget row indistinguishable from a legitimate one.
- **Normalization stays in the boundary (design Q3).** An advisor pass recommended moving `normalizeThousands` into `pkg/utils` and withdrew it on seeing Q3 and the "never call utils date/currency parsers" rule — the conflict was surfaced rather than silently resolved.
- **Reading helpers report shape-match separately from count-plausibility**, so `405,25 x0` can say "must be positive" instead of degrading to a generic format error, and an implausible count can disambiguate (`4x100` resolves rather than becoming ambiguous).
- **No new acceptance scenario.** An `apply`-level fixture would prove nothing — `apply` reads `reviewed.json`, where value is already a float and installments already an int, so it never parses a value string.
- Committed as four commits (T-72 fix, T-64 feature, docs, correction) rather than one, so the correction is visible as a correction.

### Next

- **T-65 supersede-by-append — DESIGN DOCUMENT ONLY, no code** (still the user's explicit call). Two constraints drive it and both invert the obvious: key off the STORED id (the one deliberate exception to "identity is derived", protecting 347 cross-log joins), and give the supersede record a date that survives `scanEntries`' year filter (a year-0 tombstone leaks into EVERY year — the s70 107K-workbook mechanism).
- Then **5.R6** (regenerate `feature_dictionary_enhanced.json`), which gained a measured payoff in s72: the A1 hint and the auto-insert gate BOTH require a keyword match, and 13 of 81 corpus subcategories have no keyword entry at all. It also arrives with its own before/after harness — the A1 replay probe and the gate-precision join both run over data already on disk. Note that rebuilding the dictionary changes the gate's inputs, so re-running the join is part of the task, not cleanup.
- PR #67 awaits review.
- Line 19 of `expenses-2026-01.csv` is now unblocked but needs a HAND repair to `Anita Elô ADM;09/01;405,25 x4` — the stray `;` inside the item is a separate defect from the notation. Line 45 is NOT unblocked: no date at all, and 299,00 + 49,90 = 348,90 != 646,25.
- Still open and untouched: T-58, T-69, T-56, T-57, T-70, T-71.

### Gotchas

- **"Retired" is a plan, not a property.** Three documents describe the plain-`batch` insert path as dead and all three are accurate about intent; none makes it unreachable. It is still `rootCmd.AddCommand(batchCmd)` at `batch.go:44`, so `batch` silently gained the multiplier on a path writing DIRECTLY to the workbook. `grep -rn AddCommand` settles in a second what prose can only assert.
- **A mutation can prove two things, and the second may matter more.** Deleting the normalization inside `parse.Value` turned exactly the thousands tests red while both multiplier tests stayed green — proving `ReadQueue` routes through the boundary, AND that the multiplier test is blind to that routing. The case that looks redundant is the only one carrying the guarantee, which is exactly what a tidy-up deletes.
- **A `grep -c` that returns 2 is not a failed check.** The D8 "one call site" verification counts the function DEFINITION line too; the honest check excludes it. Separately, a `go test ... | grep -E "^(FAIL|ok)" | grep -v "^ok"` pipeline reports exit 1 when the suite PASSES, because the last grep matched nothing — the exit code being read was the grep's, not the test's.
- **`resume.sh` output is not a safe authoring base for a replace-mode handoff block** (T-62): it can silently truncate a ref block, so re-fetching via `ref-lookup.sh` is what protects against writing back a truncated interior. Checked this session; not truncated, but the check is the point.
- **A backgrounded `generate_code` call ignores `output_file` silently** (T-71, confirmed again): both qcoder calls exceeded the foreground window. Not passing `output_file` at all sidesteps it — take the content from the notification and write it yourself.
- **The local model's first attempt failed on the one requirement the task turned on**, and it was structural, not a slip: a single "possible" boolean cannot distinguish "not a value/count pair" from "a pair with an out-of-range count", which made the specific error unreachable. Stubs-then-Ollama fixed exactly that. Verdicts 0 then 1.
- **A mutation that fails to COMPILE proves nothing** — reconfirmed as a live constraint when choosing which mutation to run; deleting the normalization call keeps the program runnable, which is why it discriminated.
