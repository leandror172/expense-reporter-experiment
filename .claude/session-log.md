# Session Log — Expense Reporter

**Current Session:** 2026-06-17 — Session 32: Taxonomy type persistence — Plans A & B authored (no impl)
**Current Layer:** Internal Taxonomy refactor — T-02 landed; T-05 (persist type) + T-04 (full-path routing) planned, not implemented
Most recent entry first. Run `.claude/tools/rotate-session-log.sh` when this grows beyond ~3 sessions.

---
## 2026-06-17 - Session 32: Taxonomy type persistence — Plans A & B authored (no impl)

### Context

Planning-only session on `refactor/internal-taxonomy`. Goal: map trade-offs for routing logged entries by full path and for making the expense "type" (Fixas/Variáveis/Extras/Adicionais) a first-class persisted field. No implementation — explicit standing constraint "do not start any implementation until I say so".

### What Was Done

- Authored `.claude/plans/persist-expense-type.md` (Plan A) — persist expense type end-to-end + rename ExpenseSheet→ExpenseType + migrate JSON keys (`sheet`→`type`, `sheets`→`types`) + partial backfill. Three phases: F (add `type,omitempty` to feedback.Entry + ExpenseEntry, set post-construction on apply path only; `ReviewedLocation.Sheet`→`Type` with legacy-`sheet` UnmarshalJSON fallback), R (domain rename, keeping "sheet" for Excel worksheet addressing), B-fill (re-export reviewed.json + `backfill-type.py` matching by id, partial by design).
- Authored `.claude/plans/full-path-entry-routing.md` (Plan B / T-04) — two-tier routing: typed entries → full-path key (`expensePath`), type-less entries → retained bare-name map + ambiguous-skip. `buildSubcategoryMap` returns `(byPath, byName, ambiguous, error)`.
- Authored `.claude/plans/proposed-memories.md` — 7 new per-folder QUICK/KNOWLEDGE memories to author + an "UPDATES TO EXISTING MEMORIES" section grouped by triggering plan (so the next session re-derives nothing). User will create the actual files.
- Edited `.claude/index.md` — added Quick Pointers rows for Plan A and Plan B.

### Decisions Made

- **Option 1 chosen (full path as model label / struct identity)** over option 2 (bare name + sheet only when ambiguous) — cleaner; the classifier already outputs the 3 attributes in a struct, and re-training data exists (607/694 training examples carry the type in `source`; reviewed entries carry it too).
- **Domain rename "Expense Sheet" → "Expense Type"** with JSON migration (not just dual-read) — "sheet" stays reserved for Excel worksheet addressing only.
- **ADVISOR-CAUGHT BLOCKER:** Plan B's original deletion of the bare-name map + ambiguous set would have dropped ALL type-less entries — the auto-inserted majority (auto.go:172, batch_auto.go:296 write type-less) and all ~355 existing log lines. Fix: two-tier routing keeps the guard as a permanent fallback, not removed. The ambiguity guard is NOT going away.
- **Sequencing:** merge `refactor/internal-taxonomy` → Plan A (T-05) Phase F + recover → Plan B (T-04) → classifier full-path label (later, out of scope of both plans, tracked 5.R4/RUI-4).

### Next

- Await user go-ahead to implement (standing block in effect). First impl step: Plan A (T-05) Phase F after the branch merges.
- User will hand-create the per-folder memory files from `proposed-memories.md`.

### Gotchas

- The expense type is captured in review + apply (used for workbook insertion) but **dropped at every log-write layer** and absent from the classified 7-field CSV — that is the retraining data loss Plan A repairs. Backfill from reviewed.json is partial by design (only reviewed entries carry the type).
- Memory `KNOWLEDGE.md:188–196` currently says full-path routing is "DEFERRED / guard pending removal" — both INVERTED by the advisor fix (guard is a permanent fallback). proposed-memories.md flags this stale block.
