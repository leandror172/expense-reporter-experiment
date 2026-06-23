# Session Log — Expense Reporter

**Current Session:** 2026-06-23 — Session 37: WS-A multi-year log (DONE) + income decisions + WS-C plan
**Current Layer:** Retire-insertion pivot (logs = single source of truth; generation-only)
Most recent entry first. Run `.claude/tools/rotate-session-log.sh` when this grows beyond ~3 sessions.

---
## 2026-06-23 - Session 37: WS-A multi-year log (DONE) + income decisions + WS-C plan

### Context

Continued the retire-insertion pivot from session 36. Goal this session: implement WS-A (multi-year log), settle the deferred income decisions, and plan WS-C — pacing against the session usage limit rather than context.

### What Was Done

- **WS-A / T-11 — DONE** (branch `chore/income-extraction-tooling`, commits `0c011e1`, `95dbabb`): `parseDate` now accepts `DD/MM` and `DD/MM/YYYY` → `(day, month, year, err)` with `year==0` sentinel; `LoadTaxonomy`/`scanEntries` take `targetYear` and filter (`keep iff entryYear==0 || targetYear==0 || entryYear==targetYear`); `generate.go` passes `opts.Year`, `auto.go` skeleton passes 0.
- Acceptance-first: wrote `TestGenerateWorkbook_MultiYearLogFiltersToYear` (RED-first, reuses `generate-basic` `expected-dump-data` as oracle; fixture `entries-multiyear.jsonl` = 2026 entries + 2025 noise) + unit `TestParseDate_MultiYear`. Both green; full unit suite (19 pkgs) + generate acceptance green.
- Throwaway merge script `.claude/scratch/merge_year_logs.py` → gitignored `expenses_log-allyears.jsonl` (2073 records, `DD/MM/YYYY`). Byte-identical gate PASSED all 4 years (per-year `--year N` dump == merged `--year N` dump, excl. manifest source).
- Plan doc + repo-root/taxonomy QUICK.md synced to reality (commit `docs(memory)` + `docs(plan)`).
- Phase 2 (WS-A.1/.2) ran via a Sonnet subagent; I independently re-verified its tree (build/vet/tests) before trusting it.

### Decisions Made

- **Income decisions LOCKED:** (1) 3-level symmetric income model (`Receitas → block → subline`); (2) deduction sign kept signed (negative deductions, net = sum); (3) 2022 income unrecoverable (empty shell).
- **WS-C decisions LOCKED:** income reaches the generator via a SEPARATE `--income-entries` flag consuming the extractor's `income_log.jsonl` as-is (not the unified `--entries`).
- **Currency formatting is a NO-OP** — corrected the stale WS-0 plan finding: the generator already writes a numeric value (`data_sheet.go:108`) with `R$ #,##0.00` cell format (`styles.go:53`); the "bare string" was a dump artifact of the REAL workbook. Dropped from WS-C scope.
- WS-C deferred from this session for usage budget (it is bigger than WS-A — model→loader→router→generator + a new frozen income oracle).

### Next

- **WS-C (income/revenue route) — START HERE.** Subagent-driven. Full task breakdown in `.claude/plans/retire-insertion-keep-generation.md` "WS-C task breakdown": model 3-level, loader, separate income scan/router, `revenue_sheet.go` 3-level render, merge revenue taxonomy proposal, `--income-entries` CLI, new frozen income oracle.
- Then WS-B (commands → log-append), WS-D (retire bare-name fallback), WS-E (delete dead code).
- Decide whether to promote merged `expenses_log-allyears.jsonl` to canonical + retire the per-year split (deferred; user's call).

### Gotchas

- WS-A year-filter `continue` sits AFTER `routeEntry`, so an out-of-year type-less entry still bumps the stderr fallback count before being skipped (cosmetic, but relevant to T-09's "fallback count ~0" gate).
- The Sonnet subagent recorded 0/0/0 for its local-model calls but those were cold-start TIMEOUTS, not rejections — per conventions it should have `warm_model`+retried rather than escalating to hand-writing. Future subagents doing codegen should warm the model themselves first.
- `generate_code` `output_file` with a relative path resolves against the LLM repo's REPO_ROOT, not this repo — it wrote the merge script to `/mnt/i/workspaces/llm/...`; use absolute paths or relocate.
