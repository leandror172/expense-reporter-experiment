## 2026-07-31 - Session 68: T-54, the T-42 scout on real 2026 data, and the close-cycle repairs (S1/S2/S5)

### Context

Opened with PRs #59/#60 merged and T-41 complete, intending to discuss next steps. The discussion settled a sequence — fix the cheap blocker, scout the chain on real data to replace a speculative backlog with a measured one, then fix what the scout found in one informed pass.

### What Was Done

- **T-54 (PR #61)** — `review` could not read real `batch-auto` output: `ReadQueue` accepted only `1`/`0` for `auto_inserted` while the only producer emits `true`/`false`. That spelling had **no producer anywhere in the tree**; it was invented by fixtures. Fixed the reader (strict, not `ParseBool`), re-cut both review fixtures (spelling only), fixed the second producer `.claude/tools/reconstruct-csvs.py` (it emitted 7 columns, failing on field count before the spelling mattered), and corrected the `test/.memories/KNOWLEDGE.md` note that called the fixture's hand-conversion a legitimate "fold point".
- **Seam test** — feeds the REAL writer's bytes to the REAL reader; pins both that it parses and that `ReadQueue`'s id equals the expense log's `GenerateID` (the T-41 slice-3 join, previously untested). Mutation-verified separately.
- **T-42 SCOUT** — ran the real 69-row 2026 CSV through the whole chain in a scratch install root; real logs verified byte-identical after. Report: `.claude/t42-scout-report.md`.
- **S2 (PR #62)** — `review` sourced its picker taxonomy from the WORKBOOK while everything else routed with `config/taxonomy.json`; measured **7 divergent leaves**. Repointed at `config/taxonomy.json`, dropped `--workbook`, deleted both test-only synthetic workbooks (hand-kept duplicates of the fixtures' own taxonomy).
- **S1** — an unparseable row killed the entire review step (4 of 69 did). `ReadQueue` now returns `(entries, unreviewable, error)` and `cmd/review.go` names each skipped row on stderr.
- **Closed the seam-test gap the scout found in my own T-54 guard** — it fed only parsed rows, so it passed while the seam stayed broken for unparsed ones.
- **S5** — `generate-workbook --taxonomy` now falls back to config.
- **Backfilled 3 drift-corrupted real records** (`IRFF`→`IRRF`, backed up first; exactly 3 records touched, line counts unchanged).
- Docs synced: README (intro, `review`, `generate-workbook`, test counts 303/69/35), `internal/review` + repo memories, `.claude/index.md`.

### Decisions Made

- **Scout before fixing.** The backlog was speculation until the chain ran; one throwaway run converted it into a measured list and reordered the work (S2 ahead of T-21).
- **`taxonomy.json` is the corrected vocabulary** (user call) — S2 removes `IRFF`, `Apoia-se 4i20`, `alguma coisa sindicato` and adds `Produtos de casa`.
- **Malformed rows: skip + report (option b).** Carrying them into the review UI for repair (option c) deferred by the user → T-56.
- **`- 1/4` and `4x` are input ERRORS, not notations to support** (user correction) — `99,90/3` is the only accepted form, which makes T-21 plumbing rather than parser work.
- **`--entries` deliberately NOT defaulted from config** — omitting it is how you ask for an empty year skeleton; defaulting it would be a silent capability loss.
- **Reader kept strict** (`true`/`false` only, no `ParseBool`) so a spelling nothing emits cannot creep back in unexercised.

### Next

- **T-21 — thread the installment count.** Now pure plumbing: `ReadQueue` → `exportReviewed()` in `review.html` (~line 1504) → `reviewed.json` schema → `apply.ReviewedEntry` → `appender.ExpandAndAppend`. UX locked: 1 row + ×N badge. Measured payoff: 10 of 45 rows, 17 missing log entries. Do the deferred fixture DATE canonicalization in the same pass.
- Then re-run the scout clean (task 14) — it is now the repeatable pre-close check (T-59 proposes committing it).
- Then the real T-42 monthly close.
- PRs #61 and #62 await review; **#62 contains #61**.

### Gotchas

- **The scout found a defect in the guard I had shipped hours earlier.** The T-54 seam test feeds only successfully-parsed rows, so it passed while the seam was still broken for unparsed ones. A guard built from the happy path certifies the happy path — the same "reports success while testing nothing" shape session 67 catalogued, reproduced by me within the same session.
- **Two sources of truth for one vocabulary is a silent data-loss mechanism, not redundancy.** The workbook/`taxonomy.json` drift had already inverted the feedback loop: the model predicted `IRRF` correctly and the human "corrected" it to `IRFF` because the picker offered that spelling — and corrected entries are the highest-priority few-shot source.
- **`--data-dir` does NOT isolate the logs.** It is the training-data dir (and owns the 5.R2 embedding cache). Isolation requires a scratch install root, because `config.json` and relative log paths both resolve against `filepath.Dir(os.Executable())`. Always hash the real logs before/after.
- **Do not `git checkout` a file to revert a mutation when it holds uncommitted work** — it reverts to the last commit, not to the pre-mutation state. Cost me the S1 wiring; restore from a backup copy instead.
- **When a mutation does NOT go red, record why.** Deleting `dateCell`'s zero-time guard left the seam test green — correctly, since the skip is an OR and the value cell stays empty. That regression is owned by a different test. "The mutation passed" usually means a missing test; this time it did not.
- Backgrounded MCP calls still inject no verdict template — grep `calls.jsonl` for the call_id and append by hand.
