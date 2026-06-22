# Session Log — Expense Reporter

**Current Session:** 2026-06-22 — Session 36: WS-0 diff + type backfill + WS-0b income extraction; retire-insertion plan
**Current Layer:** Layer 5 — retire-insertion architecture (logs = single source of truth)
Most recent entry first. Run `.claude/tools/rotate-session-log.sh` when this grows beyond ~3 sessions.

---
## 2026-06-22 - Session 36: WS-0 diff + type backfill + WS-0b income extraction; retire-insertion plan

### Context

Resumed on clean master with PRs #32/#33 already merged. User steered toward a major architectural pivot: retire in-place workbook insertion, keep only generation (logs become the single source of truth). Session was planning-heavy plus a corpus/data cleanup pass.

### What Was Done

- Committed 5.R4 extraction scripts as tooling history; externalized the taxonomy alias map to a gitignored `data/classification/extraction-aliases.json` (loaded via relative path).
- Authored + committed the **retire-insertion-keep-generation plan** (`.claude/plans/retire-insertion-keep-generation.md`): logs = source of truth, generate = only writer; workstreams WS-0…WS-E. Advisor-reviewed.
- Generated workbooks for all years (2022–2025) from the logs into `expense-reporter/generated-workbooks/` (gitignored *.xlsx).
- Backfilled `type` onto 21 type-less log entries (18 auto via taxonomy/typed-twins + 3 user-decided: both ração=Variáveis, Olavo gás→Habitação/Variáveis); fixed taxonomy `Fixas/Pet`→`Pets`. All 4 workbooks now regenerate with **zero skips/fallbacks**.
- **WS-0 diff** (sonnet subagent, verified against project decisions): Referência omission + col-C month start are BY DESIGN, not gaps; **income is the sole real gap** (data absent + revenue taxonomy too coarse). Premise validated for expenses.
- **WS-0b income extraction** (sonnet subagent): recovered 179 historical income records (2023:78, 2024:101; 2022 has no Receitas sheet; 2025 empty) → `~/workspaces/expenses/old/extracted/income_log.jsonl` (outside repo). Produced `extract_income.py` + gitignored `taxonomy-revenue-proposal.json` (5-block payslip taxonomy).
- gitignore hardening: global `*.bak-*` (was leaking taxonomy/log backups), `.~lock.*#`, and the revenue proposal.

### Decisions Made

- Currency formatting needs NO change — generator already writes numeric values with `R$ #,##0.00` cell format (`styles.go:51`); the WS-0 "bare string" finding was a dump-serialization artifact (it's the real workbook that stores currency as strings).
- Income (WS-0b) extracted via subagent from `~/workspaces/expenses/old/`; taxonomy proposal is NOT auto-merged — left for user review.
- Commits this session went direct to master (chore pattern) except income tooling, which is on branch `chore/income-extraction-tooling` (committed, NOT pushed, no PR yet).

### Next

- **Answer 3 deferred income questions** (start here): (1) merge `taxonomy-revenue-proposal.json` into `config/taxonomy.json` revenue side; (2) sign convention for deductions (INSS/IRRF stored negative in source); (3) accept that 2022 income is unrecoverable.
- Then **WS-A / T-11** (multi-year log): `parseDate` accept `DD/MM/YYYY`, per-entry year, merge 4 per-year logs into one, generate filters by year. Prereq for the bigger pivot.
- Then WS-C (income routing in generator — doesn't exist yet; the 179 income entries are forward-compatible until it lands), then WS-B/WS-D/WS-E.
- Push `chore/income-extraction-tooling` / open PR when ready.

### Gotchas

- **Memory audit (read THIS session — repo-root `.memories/`):** `QUICK.md` status block is stale — still says "Session 33", lists PR #32 as the open next step; should be updated to reflect PRs #32/#33 MERGED, 5.R4 extraction + income extraction done, type backfill (21→0 type-less), and the retire-insertion plan. `KNOWLEDGE.md` "Architecture — Pipeline Layers" still lists step 5 *Insert* as core — flag as PENDING-OBSOLETE once the retire-insertion pivot lands (insertion path to be deleted). Both otherwise accurate.
- **Memory candidates possibly outdated (NOT read this session — verify before trusting):** `internal/taxonomy/.memories/QUICK.md` and `internal/feedback/.memories/QUICK.md` likely still frame the bare-name fallback / type-emission as live gaps — this session showed the fallback only carried 21 legacy entries, now backfilled to 0 typed-coverage. `project_workbook_extraction_5r4` memory should get an income-extraction addendum (Receitas was excluded from 5.R4, now covered by WS-0b). No memory yet captures the "logs = single source of truth" architecture — add a project memory pointing at the plan doc.
- The `*.bak` gitignore rule does NOT match timestamped `.bak-YYYYMMDD-…`; fixed with global `*.bak-*` this session (recurring trap).
- One log entry had a malformed date `60/01` (user fixed it manually mid-session); all dates now valid.
