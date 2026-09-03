## 2026-09-01 - Session 74: the year-less-id exception, retired — both logs migrated to self-consistent ids

### Context

Resumed on a clean master with PR #67 merged, intending to discuss next steps. The session turned on two questions the user asked: whether the monthly-close backlog was real, and whether T-65 depended on fixing the stale ids first. Both re-scoped the work — the second into a migration that was believed settled against four sessions earlier.

### What Was Done

- Migrated both live JSONL logs so every `id` equals `GenerateID` of its own row and every date is `DD/MM/YYYY` (2159/2159 + 717/717; join 403 → 403, all 717 pairings identical), retiring the year-less-id exception. Tool `.claude/scratch/migrate_ids.py`, idempotent, refuses to write on any unresolved or ambiguous row.
- Retired the now-false stale-id rule everywhere it was asserted — found by grepping for the claim rather than updating remembered files, which surfaced four sites, two of which I would not have thought of.
- Added a `close-cycle.sh restore` guard refusing a pre-migration snapshot, detected by CONTENT rather than the run dir's date; mutation-verified in four directions including the must-pass case.
- Imported January's last parseable reject (`<person C> Elô ADM` → 4 × R$405,25 = R$1.621,00, `Variáveis/<person C>/Empréstimo`): first end-to-end run of T-64 on real data and first exercise of T-63's repair loop. Log 2159 → 2163.
- Established the capture chain as the real bottleneck: expenses are typed into a Telegram group and hand-exported as `result.json`, and no committed tool converts that to CSV.

### Decisions Made

- **Did the migration two-sided rather than declining it again.** s70's "recomputing drops the joins to zero" is true of a one-sided recompute and false of a two-sided one; that option was never costed. The rule generalises: a measurement pins down the operation you measured.
- **Omitted `reading-guide` and `active-decisions` from this handoff payload** — both were edited and committed earlier in the session, and re-authoring 14 long table rows from memory risks silent byte drift inside a replace-mode region.
- **Did not migrate the per-year legacy logs or any test fixture.** The fixtures encode three separate deliberate id conventions (canonical-id/bare-date, deliberately pre-canonical in `apply-stale-id`, and literal placeholders in `type-routing-cycle`); a "make everything consistent" pass would break at least two.
- **Wrote the migration script by hand after two local-model rejections** (verdicts 0 and 0), which is the documented escalation path. The first dropped every field except id/item/date/value; the second wrote a `_verify_integrity` that compared the post-migration list against itself.

### Next

- **T-65 supersede-by-append — DESIGN DOC ONLY, no code** (still the user's explicit call). **Its s72 constraint list is obsolete in both directions — re-read the reading-guide row before reusing it.** "Key off the STORED id" is MOOT; replacing it are three constraints found this session: an id can name TWO rows (8 collisions survive, genuine same-day duplicates); `scanEntries` cannot honour a supersede record at all in its current shape (single streaming pass attaching on sight, while append-only guarantees the supersede line arrives after its target); and `LoadExpenseIDCounts` counts every line by id into a CONSUMABLE budget, so a record carrying its target's id makes `--resume` over-skip and silently swallows a genuine duplicate purchase.
- **T-75 — the `result.json` → CSV converter.** The capture bottleneck, ~7 months of 2026 behind it, and the natural home for the input validation the four January rejects argue for (all were typing errors, none a classification problem).
- **5.R6** — gained another concrete data point: `<person C> Elô ADM` returned an EMPTY `keyword_hint`, so neither the gate nor the A1 hint could help.
- Line 45 of `expenses-2026-01.csv` still needs the user: no date, and 299,00 + 49,90 = 348,90 does not reconcile with the stated 646,25.
- Still open and untouched: T-56, T-57, T-58, T-69, T-70, T-71, T-73, T-74.

### Gotchas

- **A check inherits the assumptions of the code it checks.** The join-pairing check passed at 717/717 while the lookup feeding it silently coin-flipped a year for 38 ambiguous ids. Agreement between two runs of the same mistake is not evidence — the defect was found by questioning the lookup, not by any check.
- **Prove a comparison can FAIL before trusting it.** Before using "the workbooks are byte-identical" as the migration's end-to-end proof, both directions were established: changing a value moves the hash, changing an id does not. Otherwise "identical" is only reassuring.
- **A 2026-only workbook check would have been nearly blind** — 2073 of the 2079 changed ids are in 2022–2025, so the baseline had to cover all five years.
- **Round-trip fidelity is not free.** The two logs mix Go-compact and Python-spaced JSON *within each file*, so a parse-and-re-serialise migration would have cosmetically churned hundreds of untouched lines and destroyed the byte comparison. Surgical in-line field rewriting avoided it.
- **The scratchpad does not survive a session restart.** A determinism check comparing a fresh run against the previous day's dry-run output failed on a missing file — loudly, which was correct; the fix was to rebuild the baseline from the backup rather than trust remembered hashes.
- **`ls -l` on drvfs and remembered hashes are both weak evidence**; the backup was verified by checksum, and the guard by actually restoring a pre-migration snapshot and observing 0/2073 self-consistency before putting the migrated files back.
