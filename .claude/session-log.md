# Session Log — Expense Reporter

**Current Session:** 2026-08-17 — Session 70: T-59 + T-03 — the close-cycle check, and the year-less log that would have relabelled 2025 as 2026
**Current Layer:** Close cycle — pre-work COMPLETE; next is T-42, the first real monthly close on 2026 data
Most recent entry first. Run `.claude/tools/rotate-session-log.sh` when this grows beyond ~3 sessions.

---
## 2026-08-17 - Session 70: T-59 + T-03 — the close-cycle check, and the year-less log that would have relabelled 2025 as 2026

### Context

Opened on "PRs merged, local master updated — let's discuss next steps", with the handoff's declared next being T-59 (script the s68 scout), then re-run the scout, then the real T-42 close. Both of the two remaining pre-work items turned out to be materially different from their backlog entries, and the second one was invisible until the question "what is the expected output of running for real?" forced the chain to be traced end to end against the ACTUAL log contents rather than against the fixtures.

### What Was Done

- **`.claude/tools/close-cycle.sh` (T-59, PR #64)** — commits the close as a repeatable check, but deliberately NOT as the sandboxed scout the task described. Two phases (`prepare` / `finish`) around the human browser review, plus `restore`.
- **Two assertions, each derived independently of the command it checks:** `rows_in == classified rows`, and `appended rows == Σ installments over entries apply treats as NEW` — the T-21 property on real data, computed from `reviewed.json` rather than restated from apply's own summary.
- **Every guard mutation-verified:** log grew by 3 instead of 5 (the exact pre-T-21 behaviour) → red; grew by 5 instead of 3 (duplicate append) → red; `installments` deleted from the export (the T-61 mutation) → hard error; stale `reviewed.json` → red; tampered snapshot → restore refused.
- **T-03 / the canonical-log promotion** — `expenses_log-allyears.jsonl` promoted to `expenses_log.jsonl`: 2073 rows, every date `DD/MM/YYYY`. Built and byte-identical-gated in session 37, then left as "the user's call" ever since. Data change, not in git (logs are gitignored); both files backed up.
- **Nine scripts were stored `100644`** while being invoked as `./script.sh` — index-only fix. They ran here only because drvfs forces 777, so on any other clone every one was permission-denied.
- **Three stale claims corrected** in `expense-reporter/.memories/KNOWLEDGE.md` (apply passing `count=1`; `date_year` "still dead for apply"; commands "not-yet-migrated"), with the live T-55 caveat added where the ladder is described.
- **Gitignore hardened** — backups named `.pre-promote-<stamp>` matched neither the log rules nor the `*.bak-*` convention and sat untracked as real financial data.
- Local model attempted twice per policy (default persona, then `my-go-g3-12b`); both returned **verdict 0** — each produced jq that does not parse. Escalated per the tier rule.

### Decisions Made

- **The sandbox was dropped, and that is the substantive design call.** s68 used a scratch install root because the chain had never run and its failure modes were unknown. They are known now, and both logs are append-only JSONL — so a snapshot is a COMPLETE undo, equivalent insurance. The sandbox version also cost the human 45 hand-reviewed rows twice: review once, discard, review again for the real close. That trade was in the task text and was wrong.
- **The two phases are permanent.** `exportReviewed()` downloads via a Blob; there is no headless path. Synthesising `reviewed.json` to make this one command would make the script a hand-authored stand-in for a producer it does not implement — the T-54 shape, and the trap T-61 names for this exact file. Recorded in the script header, not just here, because "make it one command" is the obvious future refactor.
- **TWO snapshots, and they must not be collapsed.** `logs-before/` is the restore point; `logs-after-batch/` is the reconciliation baseline, because apply's dedup set is the classifications log AFTER batch-auto wrote its auto rows. Using one for both mixes two points in time — it cancels out only while the reviewer confirms every auto-inserted row, and fires spuriously the moment one is SKIPPED. Found by advisor review, then mutation-verified both ways.
- **The merged log's ids are stale ON PURPOSE and must not be "fixed".** The merge rewrote `date` and not `id`, so ids no longer equal `GenerateID` of their own stored date. That reads like a breach of "identity is DERIVED, never trusted", but `classifications.jsonl` still stores `DD/MM` and these ids are the only thing still joining the two logs — measured 347 joined ids before the promotion and 347 after. Recomputing would silently drop that to zero. `date` serves the generator's year filter; `id` serves the cross-log join; different bytes, deliberately.
- **T-03 itself stays OPEN.** Its text is "generate year N+1 skeleton from taxonomy alone; decide apply/add fate" — the promotion was a separate unnumbered deferred item, not T-03.
- **Per-year logs kept as backup**, not deleted, so the merge remains re-derivable.

### Next

- **T-42 — the first real monthly close.** `close-cycle.sh prepare ~/workspaces/expenses/expenses-2026-01.csv 2026`, then ~3 min of classification and 65 rows reviewed in the browser, then `finish`. Wants a sitting, not the tail of a session.
- Expected shape, from the s68 measurements: 69 rows → 20 auto / 45 review / 4 parse errors (known typos in the file); apply appends **62** rows where the pre-T-21 code wrote 45; the 2026 workbook contains ~84 rows and nothing else.
- The first real `finish` is also the first time `exportReviewed()` runs for real on the installments field (T-61 is still unguarded), so read apply's summary rather than assuming.
- PR #64 awaits review.

### Gotchas

- **`wc -l` counts NEWLINES, not lines.** It reported 1 for a two-line file whose last line lacked a trailing newline — which an `O_APPEND` JSONL log legitimately can be. The row-delta assertion would have been silently off by one. `awk 'END{print NR}'` is exact.
- **A broken check is indistinguishable from a working one when failure calls `die()`.** The membership jq indexed the seen-ids ARRAY with `.id`, so all three test cases errored — and every one looked exactly like a caught regression. Only running a case that MUST pass revealed it.
- **Sourcing a script runs its `main`.** Every isolation test of the assertions failed identically until a `BASH_SOURCE` guard was added; that guard is what makes them testable at all.
- **`resume.sh` silently truncated `active-decisions`** — it printed a bare `### Parse boundary` heading with no body while `ref-lookup.sh` showed four further sections. Authoring a replace-mode handoff block from that copy would have DELETED them. Re-fetch replace-mode interiors; do not trust the resume rendering. (Filed as T-62.)
- **A backup named outside the `*.bak-*` convention is not gitignored.** `.pre-promote-<stamp>` matched no rule and left real financial data untracked.
- **The merged log was 30 sessions stale and carried a bug s68 had already fixed** — one row still spelled `IRFF` where the live log said `IRRF`. Promoting it unchecked would have restored an unroutable leaf that `generate-workbook` warn-skips AT EXIT 0, i.e. the row vanishes silently. Always diff the merged file's overlapping subset against the live log, and validate every typed path against `config/taxonomy.json`, before promoting anything.
