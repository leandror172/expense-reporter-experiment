# Session Log — Expense Reporter

**Current Session:** 2026-07-30 — Session 67: T-41 slices 3 & 4 — batch-auto and apply on the parse boundary (T-41 COMPLETE), plus T-53
**Current Layer:** T-41 parse boundary COMPLETE — next T-21 (reviewed installments), then T-42 first monthly close
Most recent entry first. Run `.claude/tools/rotate-session-log.sh` when this grows beyond ~3 sessions.

---
## 2026-07-30 - Session 67: T-41 slices 3 & 4 — batch-auto and apply on the parse boundary (T-41 COMPLETE), plus T-53

### Context

Opened on merged PRs and a clean master, asking to discuss next steps. Chose T-53 as a prelude (it guards the evidence for everything after it), then implemented T-41 slice 3 against the already-locked plan §11, and after an advisor pass continued into slice 4 — completing T-41.

### What Was Done

- `test(acceptance)`: disable the test cache in `run-acceptance.sh` (T-53) — the suite builds the CLI in `TestMain`, so Go's cache key never covered it and the script could report a pass that ran none of the change.
- `refactor(cmd)`: repoint batch-auto at the parse boundary (T-41 slice 3) — `inputRow` deleted, the four re-parses collapsed to one, `resumeParseErr` deleted as unreachable, `--year` added, D2/T-49 counted stale-year warning, D3/T-40 join-id pinned at construction, D4 raw value token.
- `docs`: sync memories and index for T-41 slice 3.
- `test(cmd)`: make the slice-3 contracts enforceable, not just documented — advisor found D4 and the `--year` wiring were correct by INSPECTION only, the exact state D4 warned against. Three guards added, each mutation-verified.
- `refactor(cmd)`: repoint apply at the parse boundary (T-41 slice 4) — `canonicalDate` retired into new `parse.Date`, `entry.ID` recomputed from the canonical date, fixtures swept, new `apply-stale-id` fixture.
- `docs`: sync memories and index for T-41 slice 4.
- PRs #59 (slice 3) and #60 (slice 4, stacked, contains #59) opened against master; both awaiting review.
- README corrected: `--year` for batch-auto, the CSV column contract, and a `date_year` section that had been stale since slice 2 (still claimed the setting was unread by `auto`).

### Decisions Made

- **`classified.csv`/`review.csv` now carry the CANONICAL date.** It falls out of `classifiedRow` carrying a `ParsedExpense`, and it closes a real divergence: `review.ReadQueue` hashes that column into the id it puts in `reviewed.json` while both JSONL logs hash the canonical form.
- **`apply` inherits the FULL boundary including T-47's future-year refusal** (user call). A future-dated reviewed row downgrades to `failed` with a non-zero exit rather than being written — loud rather than silent, which is T-47's purpose. Per-row failure honesty preserved.
- **`parse.Date` added as a date-only entry point, with `Fields` delegating to it.** `apply` does not arrive holding three strings — `reviewed.json`'s item and value are already typed by the JSON decoder — so `Fields` would have forced it to re-serialize a float purely to have the boundary parse it back. Delegation keeps one ladder and one pair of year validations behind both doors.
- **Identity is derived, never trusted.** `entriesWithCanonicalDates` recomputes `entry.ID` alongside the date it rewrites. Third appearance of the T-35/T-18 shape.
- The two help-text corrections (rollover.csv, the `--resume` year note) were kept in slice 3 rather than split out, because the `--resume` note becomes actively wrong as a direct result of the change.

### Next

- **T-21 — reviewed-installment under-recording.** Now genuinely unblocked: `Installments` and `RawValue` are boundary fields and slice 3 made the count reachable at read time. Remaining gap is `review.ReadQueue` (queue.go:63 discards the count) → the HTML export JS → the `reviewed.json` schema → `apply.ReviewedEntry`. UX locked: 1 row + ×N badge.
- **T-54 first if the dogfood is near** — `review` cannot currently read `batch-auto`'s output at all (auto_inserted true/false vs 1/0), which breaks the middle of the T-42 chain. It touches the same `ReadQueue` code as T-21, so doing them together touches that reader once.
- Then T-42 — first real monthly close on 2026 data.
- PRs #59 and #60 await review; #60 contains #59.

### Gotchas

- **A fixture sweep can create a blind spot.** Making `apply-basic` self-consistent meant its lookup succeeds whether or not the id is recomputed — so after fixing the bug the end-to-end test could no longer detect it. Found only by mutating the fix and watching acceptance stay green. `apply-stale-id` exists because of this and must NOT be "corrected".
- **Four instances this session of one shape: a check that reports success while testing nothing** — the cached acceptance pass (T-53), a generated test that asserted the parser against a second call to itself, the silent id miss in `apply`, and the fixture blind spot above. The habit that catches them is mutation: break the thing the guard guards and confirm it goes red.
- **`my-go-qcoder` is 20.7 GB on a 12 GB GPU** — it runs CPU-offloaded, so a long generation blew a 300s ceiling while a shorter one finished in ~2 min. Prefer tightly-scoped asks, and `warm_model` the classifier back before any `-full` run.
- **The background-task notification is not the file.** One result arrived HTML-escaped (`&amp;`) AND shorter than what landed on disk — the real file had three more tests. Always read the file.
- Backgrounded MCP calls still inject no verdict template; grep `calls.jsonl` for the call_id and append by hand.
